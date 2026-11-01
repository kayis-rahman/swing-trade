#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/verify-go-stage.XXXXXX")"
trap 'rm -rf "$TEMP_DIR"' EXIT

mkdir -p "$TEMP_DIR/bin"
cat >"$TEMP_DIR/bin/ssh" <<'SH'
#!/bin/sh
printf 'unexpected SSH call\n' >>"$GO_STAGE_SSH_MARKER"
exit 97
SH
chmod +x "$TEMP_DIR/bin/ssh"

assert_refused_without_ssh() {
  local expected_message="$1"
  shift
  local status=0

  if env \
    -u STAGE_GO_PHASE1_GATE_PASSED \
    -u STAGE_GO_PHASE1_GATE_REFERENCE \
    -u STAGE_GO_BACKUP_RESTORE_VERIFIED \
    -u STAGE_GO_BACKUP_RESTORE_REFERENCE \
    -u STAGE_GO_PHASE1_GATE_RECORD \
    -u STAGE_GO_BACKUP_RESTORE_RECORD \
    PATH="$TEMP_DIR/bin:$PATH" \
    GO_STAGE_SSH_MARKER="$TEMP_DIR/ssh-called" \
    "$@" \
    "$PROJECT_ROOT/bin/deploy-go-stage.sh" >"$TEMP_DIR/output" 2>&1; then
    echo "Expected stage deployment to refuse before SSH." >&2
    exit 1
  else
    status=$?
  fi

  if [[ "$status" -ne 2 ]] || ! grep -Fq "$expected_message" "$TEMP_DIR/output"; then
    cat "$TEMP_DIR/output" >&2
    echo "Stage deployment refused for an unexpected reason (status $status)." >&2
    exit 1
  fi
  if [[ -e "$TEMP_DIR/ssh-called" ]]; then
    echo "Stage deployment contacted SSH despite missing authoritative verification." >&2
    exit 1
  fi
}

# Simulate the absence of the authoritative verifier without creating an override.
mv "$PROJECT_ROOT/bin/verify-go-stage-authority.sh" "$PROJECT_ROOT/bin/verify-go-stage-authority.sh.disabled"
trap 'mv "$PROJECT_ROOT/bin/verify-go-stage-authority.sh.disabled" "$PROJECT_ROOT/bin/verify-go-stage-authority.sh" 2>/dev/null || true; rm -rf "$TEMP_DIR"' EXIT

assert_refused_without_ssh \
  'no authoritative verifier is configured'

mv "$PROJECT_ROOT/bin/verify-go-stage-authority.sh.disabled" "$PROJECT_ROOT/bin/verify-go-stage-authority.sh"
trap 'rm -rf "$TEMP_DIR"' EXIT

bash -n "$PROJECT_ROOT/bin/deploy-go-stage.sh" "$PROJECT_ROOT/dev-stack.sh"
if command -v shellcheck >/dev/null 2>&1; then
  shellcheck "$PROJECT_ROOT/bin/deploy-go-stage.sh" "$PROJECT_ROOT/dev-stack.sh"
fi

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  API_KEY=validation-placeholder DATABASE_URL=postgres://stage@stage/swingtrade_stage docker compose \
    -f "$PROJECT_ROOT/infra/docker-compose.go-stage.yml" config --quiet
fi
if command -v ruby >/dev/null 2>&1; then
  ruby - "$PROJECT_ROOT/infra/docker-compose.go-stage.yml" <<'RUBY'
require 'yaml'

compose = YAML.load_file(ARGV.fetch(0))
api = compose.fetch('services').fetch('api')
raise 'stage API container identity changed' unless api.fetch('container_name') == 'swing-trade-stage-api'
raise 'stage API port mapping changed' unless api.fetch('ports') == ['8081:8080']
raise 'stage API must enforce admin key auth' unless api.fetch('environment').fetch('API_KEY_AUTH_ENABLED') == 'true'
raise 'stage API must require a database URL' unless api.fetch('environment').fetch('DATABASE_URL').include?('${DATABASE_URL:?')
raise 'stage API must retain the memory cap' unless api.fetch('mem_limit') == '1g'
raise 'stage network must remain external' unless compose.fetch('networks').fetch('stage').fetch('external') == true
raise 'stage network must match the dashboard network' unless compose.fetch('networks').fetch('stage').fetch('name') == 'swing-trade_swingtrade-network'
raise 'stage API must retain its dashboard alias' unless api.fetch('networks').fetch('stage').fetch('aliases').include?('swing-trade-stage-api')
RUBY
else
  echo 'Install Docker Compose or Ruby to validate the Go stage Compose file.' >&2
  exit 1
fi

echo 'Go stage deployment safety checks passed; no stage host was contacted.'
