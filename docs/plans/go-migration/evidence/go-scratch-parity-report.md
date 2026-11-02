# Go scratch Postgres parity report (Phase 6 step 4)

## Environment

- Branch: `fm/swing-trade-go-e2e-scratch-parity-v1`
- Postgres 16 disposable cluster on `127.0.0.1:55433`, database/user `go_port_test`
- Java Flyway migrations V1-V77 applied manually
- Go API launched via `go run ./go/api/cmd/swingtrade-api` (port 8090)

## What was run

- Go API health: HTTP 200.
- Go persisted reads: `/api/watchlist`, `/api/strategy-configs`, `/api/signals/latest`,
  `/api/settings` all returned HTTP 200 with seeded watchlist data after grants.
- Harness script: `docs/plans/go-migration/evidence/scripts/scratch-parity-harness.sh`.

## What was NOT run (pending)

- Java backend run: no JDK 21 available locally. Recorded golden outputs remain the baseline.
- Orchestrator runs and stage details: Go has the in-process `orchestrator` package but
  no HTTP routes/store wiring in `server.go` yet, so runs cannot be driven through the API.
- Persisted signal generation and the three SSE streams (`/api/signals/generate-all/stream`,
  `/api/candidate-scans/{runId}/stream`, `/api/analysis/run-full`): not wired.
- Watchlist and strategy write endpoints: controller packages exist but are not registered
  in `NewHandlerWithStores`, so POST/PATCH/DELETE cannot currently be driven.
- Candidate scans and scheduler slice: depends on in-flight PRs; written against documented
  interface and left pending.

## Pending re-run

Re-run after the pipeline/scans and notifications slices merge, then update this report and
the feature ledger.
