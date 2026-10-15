# Phase 0: Characterization and baseline (AUTHORIZED)

Goal: produce the evidence needed for the Gate G0 go/no-go. Read-only against the Java system. No product code or dependency changes.

## Progress log (added by implementation runs; not part of the plan spec)

| Date | Steps | Outcome | Evidence |
|---|---|---|---|
| 2026-10-14 | 2, 3 | Endpoint inventory (156 live, 13 disabled, 3 SSE) and dashboard-consumption diff | [evidence/endpoint-inventory.md](evidence/endpoint-inventory.md) |
| 2026-10-14 | 12 (partial), 12a (static) | Feature ledger expanded to 169 endpoint rows + 15 job rows; all 15 `@Scheduled` jobs characterized with triggers, side effects, idempotency guards and test coverage | [feature-ledger.md](feature-ledger.md), [evidence/scheduled-jobs.md](evidence/scheduled-jobs.md) |
| 2026-10-14 | 8 | Full Prometheus inventory — 35 application meters from 11 Java classes + 10 platform groups with registration sites, writers, label value sets, every consumer (15 Grafana panels, 2 alert rules, 1 Playwright spec) and 10 findings; ledger `F-OBS` expanded to 45 rows | [evidence/metrics-inventory.md](evidence/metrics-inventory.md), [feature-ledger.md](feature-ledger.md) |

| 2026-10-14 | 12 (cross-cutting) | Six cross-cutting security/error concerns expanded (admin API-key boundary, public fixed-window rate limit, Resilience4j endpoint rate limits, CORS, trace/log correlation, global error envelope) with dev/stage property defaults, every emitted header, both 429 shapes and 12 findings; ledger `F-API-SEC` expanded to 6 rows and `F-API-SECURITY` dropped from §7; corrected the earlier claim that the backend emits correlation response headers | [evidence/security-and-error-handling.md](evidence/security-and-error-handling.md), [feature-ledger.md](feature-ledger.md) |

Still outstanding for G0 (in dependency order): golden corpus and OpenAPI export (steps 2 remainder, 4), time/decimal
formatting rules (5), numeric golden vectors (6), schema snapshot and table ownership sheet (7), runtime confirmation
of the metrics inventory against the live `/actuator/prometheus` (8, checklist in §7 of the inventory), log-field
inventory (9), Pi and native-image measurements (10, 11), the runtime half of the scheduling-off test (12a), the
remaining ledger rows for entities, indicators/strategies, clients/risk controls and scheduling settings (12), the
test traceability matrix (13), the Checkstyle/PMD rule mapping (14) and the G0 report (16). Steps 1 and 4-14 need a
throwaway stack or Pi hardware that this environment does not provide.

## Entry
Captain authorization (given 2026-10-13). A seeded local or throwaway stack (never shared or stage infrastructure without explicit confirmation).

## Steps
1. Stand up an isolated local stack with a throwaway database seeded from checked-in fixtures.
2. Export OpenAPI from springdoc; list every live endpoint (resolve the commented-out mappings) with method, path, request and response types.
3. Diff the endpoint list against `dashboard/src/api/*.ts` to find the truly consumed surface.
4. Record golden request and response pairs per endpoint: success, validation error, not found, conflict, rate-limit, unauthorized admin; include headers and the five SSE event streams.
5. Record time and decimal formatting rules actually emitted (LocalDateTime without offset, BigDecimal as number, enum casing, omitted vs null fields).
6. Generate golden numeric vectors from Java for every indicator, strategy, gate, backtest, portfolio backtest, walk-forward, DSR and cost model.
7. Snapshot the schema (`pg_dump --schema-only`) and write a table ownership sheet: writers, readers, retention, unique keys (confirm idempotency of ingestion).
8. Inventory Prometheus series (names, labels, types) and every dependency on them (Grafana JSON, alerts).
9. Inventory log fields and consumers of `logs/*.log`.
10. Measure the Pi baseline for the current JVM: RSS, heap, startup time, p50/p95/p99, CPU, full-universe backtest wall time. Confirm the Pi CPU architecture.
11. Build and measure the native-image variant (`infra/Dockerfile.native`) against the same metrics.
12. Build the feature ledger (see feature-ledger.md): expand every area row to one row per endpoint, job, entity, metric and strategy.
12a. Characterize every scheduled job (trigger, side effects, idempotency) and test that `app.scheduling.enabled=false` stops all Java scheduling.
13. Build the test traceability matrix: all Java test methods listed with module, class, target feature.
14. Map every Checkstyle and PMD rule to a Go equivalent or a recorded reason.
15. File a separate ticket for rotating the committed Finnhub key and cleaning secrets (independent of this program).
16. Write the G0 report.

## Deliverables
Golden corpus (outside product tree), OpenAPI file, schema snapshot, ownership sheet, metrics inventory, baseline and native-image numbers, feature ledger, traceability matrix, rule mapping, G0 report.

## Gate G0 exit criteria
- Baseline and native-image numbers are in the report with method and caveats.
- Ledger is fully expanded: one row per endpoint, job, entity, metric and strategy, with no area-level row left. G0 cannot pass with unexpanded rows.
- Every scheduled job is characterized and the scheduling-off test passes.
- Golden corpus reproducible by a script.
- Captain decides: proceed to Phase 1, or stay on Java with the native-image path. If Go does not clearly beat the native-image baseline on the Q1 metric, the default recommendation is to stay on Java.
