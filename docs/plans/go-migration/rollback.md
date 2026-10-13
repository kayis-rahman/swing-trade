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
1. Set job owners to java, route nginx back to Java, confirm Java health.
2. Reconcile data written by Go during the window (additive-only schema rule makes this safe); if irreconcilable, restore the verified backup.
3. Post-incident note appended to this plan; captain decides retry.
