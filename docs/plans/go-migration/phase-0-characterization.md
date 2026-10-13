# Phase 0: Characterization and baseline (AUTHORIZED)

Goal: produce the evidence needed for the Gate G0 go/no-go. Read-only against the Java system. No product code or dependency changes.

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
