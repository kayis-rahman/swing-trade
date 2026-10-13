# Phase 7: Paper trading, risk controls, kill switch

Goal: the highest-safety component, last, proven in shadow with Java as the only writer.

## Scope
Paper engine, orders, positions, trades and labels, portfolio snapshots, monitor, risk controls, kill switch, daily-loss circuit breaker, intraday paper sessions and scans, cost models. Kite: interface plus refusing stub; live code behind the `live` build tag, excluded from every released binary (verified in CI by build and symbol check).

## Steps
1. Ledger rows per component. Broker has the largest Java test suite (13.6k lines); map every test.
2. Red tests translated from Java, including kill-switch gating, loss and trade-count limits, one position at a time, 15:15 forced square-off, partial fills, optimistic-lock retries, zero live-broker call assertions.
3. Implement; green.
4. Shadow: Go computes decisions on the same inputs as Java without writing; compare orders, fills, costs, positions, P&L exactly.
5. Replay the intraday acceptance fixture (25 bars, 2 orders, 1 trade, costs 47.57, net -27.6802601).
6. Drills: kill switch mid-session, loss breaker trip, restart mid-session, concurrent updates.
7. Verify the released binary has no live order path and refuses LIVE mode at startup.

## Gate G7 exit criteria
Zero divergence across the shadow window including drills; kill switch honored within 1 s; live-path absence verified.
