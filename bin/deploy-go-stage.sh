#!/usr/bin/env bash
# shellcheck disable=SC2317,SC2329
# Remote SSH commands intentionally interpolate fixed local deployment paths.
# shellcheck disable=SC2029
set -euo pipefail
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAGE_HOST="dietpi@piworm.local"
STAGE_PATH="/home/dietpi/swing-trade-go-stage"
JAVA_STAGE_PATH="/home/dietpi/swing-trade"
COMPOSE_PROJECT="swing-trade-go-stage"
COMPOSE_FILE="docker-compose.go-stage.yml"
JAVA_CONTAINER="swing-trade-stage-api"

STAGE_AUTHORITY_VERIFIER="$PROJECT_ROOT/bin/verify-go-stage-authority.sh"
if [[ ! -x "$STAGE_AUTHORITY_VERIFIER" ]]; then
  echo "Refusing Go stage replacement: no authoritative verifier is configured for the Phase 1 gate and stage backup/restore." >&2
  echo "Caller-supplied records cannot authorize stage access; obtain approval to integrate an authoritative verifier." >&2
  exit 2
fi
if ! "$STAGE_AUTHORITY_VERIFIER"; then
  echo "Refusing Go stage replacement: authoritative stage access verification failed." >&2
  exit 2
fi

BUILD_COMMIT="$(cd "$PROJECT_ROOT" && git rev-parse HEAD)"
if [[ -n "$(cd "$PROJECT_ROOT" && git status --porcelain)" ]]; then
  echo "Refusing Go stage replacement from a dirty worktree." >&2
  exit 2
fi

BUILD_ID="${BUILD_COMMIT:0:12}"
ROLLBACK_CONTAINER="swing-trade-stage-api-rollback-${BUILD_ID}"
BUILD_DIR="$(mktemp -d "${TMPDIR:-/tmp}/go-stage.XXXXXX")"
ROLLBACK_REQUIRED=0
DEPLOY_SUCCEEDED=0

cleanup() {
  local status=$?
  trap - EXIT
  if [[ "$ROLLBACK_REQUIRED" == "1" && "$DEPLOY_SUCCEEDED" != "1" ]]; then
    echo "Go stage health verification failed; restoring the previous stage API container." >&2
    ssh "$STAGE_HOST" "
      if docker inspect '$ROLLBACK_CONTAINER' >/dev/null 2>&1; then
        docker rm -f '$JAVA_CONTAINER' >/dev/null 2>&1 || true
        docker rename '$ROLLBACK_CONTAINER' '$JAVA_CONTAINER'
      fi
      docker start '$JAVA_CONTAINER' >/dev/null
    " || echo "Stage rollback did not complete; inspect the stage API container before retrying." >&2
  fi
  rm -rf "$BUILD_DIR"
  exit "$status"
}
trap cleanup EXIT

REMOTE_ARCH="$(ssh "$STAGE_HOST" uname -m)"
case "$REMOTE_ARCH" in
  aarch64|arm64)
    GOARCH=arm64
    ;;
  x86_64|amd64)
    GOARCH=amd64
    ;;
  armv7l)
    GOARCH=arm
    export GOARM=7
    ;;
  *)
    echo "Unsupported stage host architecture: $REMOTE_ARCH" >&2
    exit 2
    ;;
esac

echo "Building Go API for Linux/$GOARCH at $BUILD_ID."
(
  cd "$PROJECT_ROOT"
  GOOS=linux GOARCH="$GOARCH" CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" \
      -o "$BUILD_DIR/swingtrade-api" ./go/api/cmd/swingtrade-api
)

echo "Preparing the Go image on the stage host; the running API is unchanged so far."
ssh "$STAGE_HOST" "mkdir -p '$STAGE_PATH'"
scp "$BUILD_DIR/swingtrade-api" \
  "$PROJECT_ROOT/infra/Dockerfile.go-stage" \
  "$PROJECT_ROOT/infra/docker-compose.go-stage.yml" \
  "$STAGE_HOST:$STAGE_PATH/"

ssh "$STAGE_HOST" "
  test -s '$JAVA_STAGE_PATH/.env.stage' &&
  docker network inspect 'swing-trade_swingtrade-network' >/dev/null &&
  docker inspect --format '{{.State.Running}}' '$JAVA_CONTAINER' | grep -qx true &&
  ! docker inspect '$ROLLBACK_CONTAINER' >/dev/null 2>&1 &&
  cd '$STAGE_PATH' &&
  docker compose --env-file '$JAVA_STAGE_PATH/.env.stage' \
    --project-name '$COMPOSE_PROJECT' -f '$COMPOSE_FILE' config --quiet &&
  docker compose --env-file '$JAVA_STAGE_PATH/.env.stage' \
    --project-name '$COMPOSE_PROJECT' -f '$COMPOSE_FILE' \
    build --build-arg BUILD_COMMIT='$BUILD_COMMIT' api
"

JAVA_IMAGE="$(ssh "$STAGE_HOST" "docker inspect --format '{{.Config.Image}}' '$JAVA_CONTAINER'")"
JAVA_ID="$(ssh "$STAGE_HOST" "docker inspect --format '{{.Id}}' '$JAVA_CONTAINER'")"
printf 'Existing API container image: %s\n' "$JAVA_IMAGE"
printf 'Existing API container ID: %s\n' "$JAVA_ID"
echo "The existing API will be preserved as $ROLLBACK_CONTAINER for rollback."

ROLLBACK_REQUIRED=1
if ! ssh "$STAGE_HOST" "docker stop '$JAVA_CONTAINER' >/dev/null && docker rename '$JAVA_CONTAINER' '$ROLLBACK_CONTAINER'"; then
  echo "Could not safely stop and preserve the existing API container; deployment aborted." >&2
  exit 1
fi

ssh "$STAGE_HOST" "
  cd '$STAGE_PATH' &&
  docker compose --env-file '$JAVA_STAGE_PATH/.env.stage' \
    --project-name '$COMPOSE_PROJECT' -f '$COMPOSE_FILE' up -d api
"

# Prometheus runs in pi-stack's monitoring network and must be attached to the
# Go stage network so it can resolve swing-trade-stage-api. Connect it before
# verification; ignore the "already connected" case.
ssh "$STAGE_HOST" "
  net='swing-trade-stage_swingtrade-network'
  docker network inspect \"\$net\" >/dev/null 2>&1 || exit 0
  if ! docker network inspect \"\$net\" --format '{{range .Containers}}{{.Name}} {{end}}' | grep -qw pi-prometheus; then
    docker network connect \"\$net\" pi-prometheus
  fi
" || true

# Reload Prometheus (bound to the Tailscale address) so scrape targets refresh.
ssh "$STAGE_HOST" "curl -fsS -X POST http://100.104.74.58:9090/-/reload >/dev/null 2>&1 || true"

HEALTHY=0
for _ in $(seq 1 45); do
  if curl -fsS http://piworm.local:8081/actuator/health >/dev/null; then
    HEALTHY=1
    break
  fi
  sleep 2
done

if [[ "$HEALTHY" != "1" ]]; then
  echo "Go API did not become healthy within 90 seconds." >&2
  exit 1
fi

for path in /api/health /actuator/health/liveness /actuator/health/readiness /actuator/prometheus; do
  if ! curl -fsS "http://piworm.local:8081$path" >/dev/null; then
    echo "Go API stage smoke check failed for $path." >&2
    exit 1
  fi
  printf 'Go API stage smoke check passed: %s\n' "$path"
done

for path in /api/health /actuator/health /actuator/health/liveness /actuator/health/readiness; do
  if ! curl -fsS "http://piworm.local:8082$path" >/dev/null; then
    echo "Dashboard proxy health check failed for $path." >&2
    exit 1
  fi
  printf 'Dashboard proxy health check passed: %s\n' "$path"
done

if ! curl -fsS http://piworm.local:8082/api/strategy-configs | python3 -c 'import json,sys; d=json.load(sys.stdin); assert d.get("success") is True and isinstance(d.get("data"),list) and len(d["data"]) > 0'; then
  echo "Dashboard strategy configuration read check failed." >&2
  exit 1
fi
if ! curl -fsS http://piworm.local:8082/api/signals/latest | python3 -c 'import json,sys; d=json.load(sys.stdin); assert isinstance(d,list) and len(d) > 0'; then
  echo "Dashboard latest signals read check failed." >&2
  exit 1
fi
echo "Dashboard strategy and signal read checks passed."

ADMIN_STATUS="$(curl -sS -o /dev/null -w '%{http_code}' http://piworm.local:8081/api/admin/health)"
if [[ "$ADMIN_STATUS" != "401" ]]; then
  echo "Go API admin-key boundary check failed (expected HTTP 401 without a key)." >&2
  exit 1
fi
echo "Go API stage smoke check passed: admin API key is required."

PROMETHEUS_UP=0
for _ in $(seq 1 15); do
  PROMETHEUS_TARGETS="$(curl -fsS http://100.104.74.58:9090/api/v1/targets 2>/dev/null || true)"
  if printf '%s' "$PROMETHEUS_TARGETS" | python3 -c 'import json,sys; data=json.load(sys.stdin); targets=data.get("data",{}).get("activeTargets",[]); sys.exit(not any(t.get("labels",{}).get("job")=="swing-trade-stage" and t.get("health")=="up" for t in targets))' 2>/dev/null; then
    PROMETHEUS_UP=1
    break
  fi
  sleep 2
done
if [[ "$PROMETHEUS_UP" != "1" ]]; then
  echo "Prometheus did not report the swing-trade-stage target as up." >&2
  exit 1
fi
echo "Prometheus scrape target is up."

DEPLOY_SUCCEEDED=1
ROLLBACK_REQUIRED=0
echo "Go API is healthy on stage. The previous API remains stopped in $ROLLBACK_CONTAINER for rollback."
