#!/usr/bin/env bash
# Scratch Postgres end-to-end parity probe for the Go API.
# Prerequisites: local Postgres 16 on 127.0.0.1:55433 with database/user go_port_test,
# Go API started with SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE=1 and DATABASE_URL set to
# the guaranteed-failing placeholder (not-a-postgres-url), via
# `SERVER_PORT=8090 go run ./go/api/cmd/swingtrade-api`. See scratch-go-parity-runbook.md.
set -euo pipefail
BASE="${1:-http://127.0.0.1:8090}"
out="${2:-/tmp/scratch-parity}"
mkdir -p "$out"
: > "$out/summary.txt"
for path in /api/health /api/watchlist /api/strategy-configs /api/signals/latest /api/settings; do
  code=$(curl -s -o "$out/$(echo "$path" | tr '/' '_').json" -w "%{http_code}" "$BASE$path")
  printf '%s %s\n' "$code" "$path" >>"$out/summary.txt"
done
cat "$out/summary.txt"
