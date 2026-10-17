# api

Go module `github.com/kayis-rahman/swing-trade/go/api`, corresponding to Gradle module `backend/api`; the executable lives in `cmd/swingtrade-api`. Owns HTTP controllers, orchestration, DTOs, config wiring, schedulers, middleware and API metrics. May import all six supporting modules.

Phase 1 exposes infrastructure health and metrics routes only. It is not routed to production traffic.

## Local run and verification

From the repository root:

- Start the local loopback service: `./bin/go-run` (default `http://127.0.0.1:8081`)
- Run unit tests, a real-binary HTTP E2E test, optional authorized development-database integration, race tests, vet, lint, vulnerability scan and architecture checks: `./bin/go-verify`

The binary exposes only `/api/health`, `/api/admin/health`, `/actuator/health`, `/actuator/health/liveness`, `/actuator/health/readiness` and `/actuator/prometheus`. The E2E test starts the actual binary, checks every route, and verifies admin-key and CORS preflight behavior. Configuration and database safety are documented in [`docs/plans/go-migration/phase-1-config.md`](../../docs/plans/go-migration/phase-1-config.md).
