# Rollback

Gated big-bang: before G8 Go serves no production traffic, so rollback means stopping the shadow or scratch work; production is untouched. After the G8 switch, Java stays warm for the window.

## Triggers (approved as proposed)
- Any diff in orders, positions or P&L between Go and Java (shadow) or any wrong paper ledger entry after cutover.
- Any live-broker call from either runtime.
- 5xx rate above 1% for 5 minutes on any route.
- p95 latency above 1.5x the Phase 0 baseline for 15 minutes.
- Missed or duplicated scheduled job.
- Ingestion row mismatch.
- Kill switch not honored within 1 second in a drill.
- Memory above 85% of the container cap.
- Unreviewed schema drift.

## Procedure after cutover
1. Stop Go jobs, re-enable Java scheduling (`app.scheduling.enabled=true`), route nginx back to Java, confirm Java health.
2. Reconcile data written by Go during the window (see the schema rules below); if irreconcilable, restore the verified backup.
3. Post-incident note appended to this plan; captain decides retry.

## Schema rules after the goose baseline
- The goose baseline V76 is a no-op that records the schema as applied, verified equal to the Phase 0 schema snapshot.
- `flyway_schema_history` is never dropped or edited, so Java's Flyway validation still passes on rollback.
- Until Java is retired and the rollback window is closed, only additive changes are allowed: new tables, nullable or defaulted columns, new indexes. No drops, renames, type changes, or NOT NULL columns without a default.
- Rollback is forward-fix with new additive migrations. There are no production down-migrations.
- Before G8, rehearse booting Java with `ddl-auto=validate` against a goose-migrated database.
