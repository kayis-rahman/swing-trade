# Phase 1: Go platform skeleton (no traffic)

Goal: a Go workspace that builds, lints, tests and runs with production-grade scaffolding but serves no production traffic.

## Entry
Captain waived Phase 0 and G0 on 2026-10-17 and authorized Phase 1 locally. Do not create a G0 report or collect more characterization evidence; see [Phase 1 authorization](phase-1-authorization.md).

## Steps
1. Create `go.work` and the seven modules per standards.md; add empty module READMEs mapping each to its Java module.
2. Configure golangci-lint, depguard, gosec, govulncheck, forbidigo and the local module/package architecture gate; add the rule mapping table from Phase 0.
3. Extend `bin/verify-changes` to select Go checks by changed path; keep non-fixing behavior.
4. Typed config from env with the existing property names documented in a table; `.env.example` stays the template.
5. `log/slog` JSON handler with traceId, symbol, module; request trace middleware.
6. Metrics registry with the Micrometer-compatible naming table; `/actuator/prometheus` and `/actuator/health` aliases with liveness and readiness.
7. Middleware: recover, trace id, request metrics, per-IP rate limit with the same headers, admin API key with constant-time compare, CORS list; port `ApiSecurityInterceptorTest` cases red first.
8. pgx pool, transaction helper, optimistic-lock retry helper, clock port.
9. Decimal package with scale and rounding constants; red tests from Java `FinancialScale` and cost-model tests.
10. Local test harness uses the authorized development database only: read-only reads; writes confined to a generated `go_port_test_<random>` schema that is dropped after each test. Migrations, `public` writes, testcontainers, golden comparator and shadow comparator are out of scope for this no-traffic skeleton.
11. Add a Go stage deployment set that packages the no-traffic skeleton; actual stage replacement remains gated by the no-mistakes/evidence gate and stage backup/restore prerequisites. No image push or stage traffic is authorized by this plan.
12. Pi deployment and Pi RSS/startup measurement are deferred; local real-binary startup is covered by HTTP E2E.
13. Stage database backup/restore is deferred by the captain's no-stage-operations boundary.

## Gate G1 exit criteria
Lint, race tests and architecture checks green; metrics scrape-compatible; local real-binary route E2E green. The Pi RSS/startup target and stage restore drill remain explicitly deferred and block full G1 until separately authorized.
