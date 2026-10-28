---
name: go-platform-dev
description: Implement or troubleshoot Go platform work in this repository. Use for edits under go/**, go.work, go/data/sqlc.yaml, go/data/queries/**, Go verification scripts, or Go parity tests.
---

# Go platform development

Work as a senior/staff engineer: preserve module ownership, dependency direction, and Java-compatible contracts when changing Go code.

## Scope

In scope: workspace/module boundaries; `core` primitives (decimal, config, clock, domain, log); `data` (SQLC-backed readers/stores, market clients, gateway jobs); `api` (contracts/envelope, middleware, parity); module dependency direction; feature gates; Go test/lint/verify conventions.

Out of scope (do not generalise): Java Spring backend (`backend-dev`), dashboard (`dashboard-dev`), and local/stage stack operations (`dev-stack`).

## Invariants

- Keep domain logic in `core`, persistence/HTTP client logic in `data`, REST/transport logic in `api`, and strategy math in `strategy`.
- Preserve the `go.work` module set and dependency direction; do not add reverse imports.
- Use `go/core/decimal.Decimal` and defined scale constants for money/percent/indicator math.
- Validate runtime config through `core/config.Load(lookup)` and expose gates as booleans.
- Put public read seams on `data` interfaces (for example `WatchlistReadStore`) and use fakes in handler tests.
- Maintain Java-compatible `ApiResponse` envelopes, status codes, timestamp format, and sanitized 500 responses.
- Use SQLC through `go/data/sqlc.yaml`; generated client changes must keep `go/data/*.go` wrappers compiling.
- Propagate `context.Context`, wrap errors with `%w`, avoid `time.Sleep`, and keep functions under the `funlen` budget (`.golangci.yml`).
- Add golden and parity coverage; keep DB-backed changes on isolated schemas.
- Run `./bin/go-verify` for Go changes; inspect `--dry-run` only when it is unclear what will run.

## Verification

- Run the narrowest relevant Go tests first, then `./bin/go-verify` for cross-module changes.
- Report checks actually run and any that could not be run.

Preserve unrelated worktree changes.