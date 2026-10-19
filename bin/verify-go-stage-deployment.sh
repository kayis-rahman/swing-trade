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

python3 - "$TEMP_DIR" "$(cd "$PROJECT_ROOT" && git rev-parse HEAD)" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
commit = sys.argv[2]
gate = {
    "kind": "no-mistakes-phase1",
    "scope": "go-stage-replacement",
    "commit": commit,
    "status": "passed",
    "run_reference": "run-123",
}
backup = {
    "kind": "stage-backup-restore",
    "scope": "stage-database-backup-restore",
    "status": "verified",
    "backup_reference": "backup-123",
    "backup_sha256": "a" * 64,
    "restore_reference": "restore-123",
    "restore_target": "isolated",
    "restore_verified": True,
}
(root / "gate.json").write_text(json.dumps(gate), encoding="utf-8")
(root / "backup.json").write_text(json.dumps(backup), encoding="utf-8")
(root / "stale-gate.json").write_text(
    json.dumps({**gate, "commit": "0" * 40}), encoding="utf-8"
)
(root / "wrong-scope-gate.json").write_text(
    json.dumps({**gate, "scope": "other"}), encoding="utf-8"
)
(root / "bad-backup.json").write_text(
    json.dumps({**backup, "restore_target": "stage"}), encoding="utf-8"
)
(root / "wrong-scope-backup.json").write_text(
    json.dumps({**backup, "scope": "other"}), encoding="utf-8"
)
(root / "malformed.json").write_text("{", encoding="utf-8")
PY

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
    echo "Stage deployment contacted SSH before satisfying its safety preconditions." >&2
    exit 1
  fi
}

assert_refused_without_ssh \
  'invalid prerequisite evidence' \
  STAGE_GO_PHASE1_GATE_PASSED=YES \
  STAGE_GO_PHASE1_GATE_REFERENCE=fabricated \
  STAGE_GO_BACKUP_RESTORE_VERIFIED=YES \
  STAGE_GO_BACKUP_RESTORE_REFERENCE=fabricated
assert_refused_without_ssh \
  'invalid prerequisite evidence' \
  STAGE_GO_PHASE1_GATE_RECORD="$TEMP_DIR/stale-gate.json" \
  STAGE_GO_BACKUP_RESTORE_RECORD="$TEMP_DIR/backup.json"
assert_refused_without_ssh \
  'invalid prerequisite evidence' \
  STAGE_GO_PHASE1_GATE_RECORD="$TEMP_DIR/wrong-scope-gate.json" \
  STAGE_GO_BACKUP_RESTORE_RECORD="$TEMP_DIR/backup.json"
assert_refused_without_ssh \
  'invalid prerequisite evidence' \
  STAGE_GO_PHASE1_GATE_RECORD="$TEMP_DIR/gate.json" \
  STAGE_GO_BACKUP_RESTORE_RECORD="$TEMP_DIR/wrong-scope-backup.json"
assert_refused_without_ssh \
  'invalid prerequisite evidence' \
  STAGE_GO_PHASE1_GATE_RECORD="$TEMP_DIR/gate.json" \
  STAGE_GO_BACKUP_RESTORE_RECORD="$TEMP_DIR/malformed.json"
assert_refused_without_ssh \
  'invalid prerequisite evidence' \
  STAGE_GO_PHASE1_GATE_RECORD="$TEMP_DIR/gate.json" \
  STAGE_GO_BACKUP_RESTORE_RECORD="$TEMP_DIR/bad-backup.json"

bash -n "$PROJECT_ROOT/bin/deploy-go-stage.sh" "$PROJECT_ROOT/dev-stack.sh"
if command -v shellcheck >/dev/null 2>&1; then
  shellcheck "$PROJECT_ROOT/bin/deploy-go-stage.sh" "$PROJECT_ROOT/dev-stack.sh"
fi

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  API_KEY=validation-placeholder docker compose \
    -f "$PROJECT_ROOT/infra/docker-compose.go-stage.yml" config --quiet
elif command -v ruby >/dev/null 2>&1; then
  ruby - "$PROJECT_ROOT/infra/docker-compose.go-stage.yml" <<'RUBY'
require 'yaml'

compose = YAML.load_file(ARGV.fetch(0))
api = compose.fetch('services').fetch('api')
raise 'stage API container identity changed' unless api.fetch('container_name') == 'swing-trade-stage-api'
raise 'stage API port mapping changed' unless api.fetch('ports') == ['8081:8080']
raise 'stage API must enforce admin key auth' unless api.fetch('environment').fetch('API_KEY_AUTH_ENABLED') == 'true'
raise 'stage API must not receive a database URL' if api.fetch('environment').key?('DATABASE_URL')
raise 'stage API must retain the memory cap' unless api.fetch('mem_limit') == '1g'
raise 'stage network must remain external' unless compose.fetch('networks').fetch('stage').fetch('external') == true
RUBY
else
  echo 'Install Docker Compose or Ruby to validate the Go stage Compose file.' >&2
  exit 1
fi

echo 'Go stage deployment safety checks passed; no stage host was contacted.'
