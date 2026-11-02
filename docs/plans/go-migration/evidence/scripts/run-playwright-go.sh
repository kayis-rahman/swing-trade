#!/usr/bin/env bash
# Run the existing Playwright suite unchanged against the Go server on a scratch stack.
# Prerequisites:
#   1. Disposable PostgreSQL 16 already running on 127.0.0.1:55433 as go_port_test/go_port_test
#      with the phase schema loaded (see docs/plans/go-migration/evidence/schema-snapshot.sql).
#   2. Go API already built/running on 127.0.0.1:8081 with
#      SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE=1.
#   3. Dashboard Vite dev server on 127.0.0.1:3003 (yarn dev --host 127.0.0.1).
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
cd "${REPO_ROOT}/dashboard"

export STAGE_DASHBOARD_URL="${STAGE_DASHBOARD_URL:-http://127.0.0.1:3003}"
export STAGE_API_URL="${STAGE_API_URL:-http://127.0.0.1:8081}"
export GO_SERVICE_URL="${GO_SERVICE_URL:-http://127.0.0.1:8081}"

npx playwright test --reporter=list,json 2>&1 | tee "${TMPDIR:-/tmp}/playwright-go-run.log"
