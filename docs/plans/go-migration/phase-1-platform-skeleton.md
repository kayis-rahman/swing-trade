# Phase 1: Go platform skeleton (no traffic)

Goal: a Go workspace that builds, lints, tests and runs with production-grade scaffolding but serves no production traffic.

## Entry
G0 approved with a go decision.

## Steps
1. Create `go.work` and the seven modules per standards.md; add empty module READMEs mapping each to its Java module.
2. Configure golangci-lint, depguard, go-arch-lint, gosec, govulncheck, forbidigo; add the rule mapping table from Phase 0.
3. Extend `bin/verify-changes` to select Go checks by changed path; keep non-fixing behavior.
4. Typed config from env with the existing property names documented in a table; `.env.example` stays the template.
5. `log/slog` JSON handler with traceId, symbol, module; request trace middleware.
6. Metrics registry with the Micrometer-compatible naming table; `/actuator/prometheus` and `/actuator/health` aliases with liveness and readiness.
7. Middleware: recover, trace id, request metrics, per-IP rate limit with the same headers, admin API key with constant-time compare, CORS list; port `ApiSecurityInterceptorTest` cases red first.
8. pgx pool, transaction helper, optimistic-lock retry helper, clock port.
9. Decimal package with scale and rounding constants; red tests from Java `FinancialScale` and cost-model tests.
10. Test harness: testcontainers Postgres applying the real migrations; golden-file comparator; shadow comparator tool.
11. Container image and a stage target alongside the Java one (separate container, separate port, not routed).
12. Run the skeleton on the Pi; record RSS and startup.
13. Backup and restore drill for the stage database.

## Gate G1 exit criteria
Lint, race tests and arch checks green; metrics scrape-compatible; skeleton RSS and startup within the Q1 target; restore drill verified.
