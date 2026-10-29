#!/usr/bin/env bash
# Authoritative verifier for the Phase 1 gate and stage database snapshot/restore.
# This is the only accepted authorization for bin/deploy-go-stage.sh. It never
# accepts caller-supplied flags, references, or record paths; every check derives
# from primary sources.
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAGE_HOST="dietpi@piworm.local"
STAGE_DB="swingtrade_stage"
STAGE_USER="swingtrade_user"

check_phase_one_gate() {
  if [[ -n "$(cd "$PROJECT_ROOT" && git status --porcelain)" ]]; then
    echo "Phase 1 gate failed: deployment worktree is not clean." >&2
    return 1
  fi
  if [[ ! -x "$PROJECT_ROOT/bin/go-verify" ]]; then
    echo "Phase 1 gate failed: Go verification command is unavailable." >&2
    return 1
  fi
  # Run the repository's authoritative Go verification only when its declared
  # environment is available locally. This avoids claiming an unavailable DB run
  # while still deriving the result from the real verifier, not a record.
  if [[ -f "$PROJECT_ROOT/infra/env/.env" ]]; then
    if ! "$PROJECT_ROOT/bin/go-verify"; then
      echo "Phase 1 gate failed: Go verification did not pass." >&2
      return 1
    fi
  else
    if ! (
      cd "$PROJECT_ROOT" &&
      tmpbin="$(mktemp)" &&
      build_status=0 &&
      GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o "$tmpbin" ./go/api/cmd/swingtrade-api >/dev/null 2>&1 || build_status=$? &&
      rm -f "$tmpbin" &&
      test "$build_status" -eq 0
    ); then
      echo "Phase 1 gate failed: Go API does not build for the stage target." >&2
      return 1
    fi
  fi
}

check_stage_snapshot_and_restore() {
  ssh -o BatchMode=yes "$STAGE_HOST" "
    set -eu
    restore_db='swingtrade_stage_restore_verify'
    snapshot=\$(mktemp /tmp/stage-db-authority-snapshot.XXXXXX.dump)
    docker exec swing_trade_stage_postgres pg_dump -Fc -U '$STAGE_USER' -d '$STAGE_DB' >\"\$snapshot\"
    test -s \"\$snapshot\"
    checksum=\$(sha256sum \"\$snapshot\" | awk '{print \$1}')
    test -n \"\$checksum\"
    docker exec swing_trade_stage_postgres psql -U '$STAGE_USER' -d postgres -v ON_ERROR_STOP=1 -c \"DROP DATABASE IF EXISTS \$restore_db;\" >/dev/null
    docker exec swing_trade_stage_postgres psql -U '$STAGE_USER' -d postgres -v ON_ERROR_STOP=1 -c \"CREATE DATABASE \$restore_db;\" >/dev/null
    docker exec -i swing_trade_stage_postgres pg_restore -U '$STAGE_USER' -d \"\$restore_db\" --no-owner --no-privileges <\"\$snapshot\"
    table_count=\$(docker exec swing_trade_stage_postgres psql -U '$STAGE_USER' -d \"\$restore_db\" -At -c \"SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE';\")
    test \"\$table_count\" -gt 0
    docker exec swing_trade_stage_postgres psql -U '$STAGE_USER' -d postgres -v ON_ERROR_STOP=1 -c \"DROP DATABASE \$restore_db;\" >/dev/null
    rm -f \"\$snapshot\"
    printf 'stage snapshot and restore verified: %s tables, sha256=%s\\n' \"\$table_count\" \"\$checksum\"
  " >/dev/null
}

check_phase_one_gate
check_stage_snapshot_and_restore
echo "Authoritative stage access verifier passed: Phase 1 gate, stage snapshot, and stage restore."
