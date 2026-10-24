# api

Go module `github.com/kayis-rahman/swing-trade/go/api`, corresponding to Gradle module `backend/api`; the executable lives in `cmd/swingtrade-api`. Owns HTTP controllers, orchestration, DTOs, config wiring, schedulers, middleware and API metrics. May import all six supporting modules.

Phase 1 established the no-traffic metrics skeleton. Phase 2 adds the Java-compatible `/api/health` read endpoints and read-only `GET /api/watchlist` and `GET /api/watchlist/all`; watchlist reads require an authorized `DATABASE_URL` and return 503 when no store is configured. Health response field names remain compatible with current clients, with Java-era runtime fields mapped to Go runtime values until Phase 8. No Go endpoint is routed to production traffic.

## Local run and verification

From the repository root:

- Start the local loopback service: `./bin/go-run` (default `http://127.0.0.1:8081`)
- Run unit tests, a real-binary HTTP E2E test, optional authorized development-database integration, race tests, vet, lint, vulnerability scan and architecture checks: `./bin/go-verify`

The binary exposes the Phase 2 health read routes, watchlist reads, platform health/metrics, and admin health. The real-binary E2E test checks infrastructure routes and verifies that watchlist reads fail safely when no database is configured; handler and authorized read-only database integration tests cover watchlist response fields, ordering, and empty/null values. Health handler tests cover all six routes, legacy JSON field names, runtime mapping, verbosity, and configured/unconfigured database details. It also verifies admin-key and CORS preflight behavior. Configuration and database safety are documented in [`docs/plans/go-migration/phase-1-config.md`](../../docs/plans/go-migration/phase-1-config.md).
