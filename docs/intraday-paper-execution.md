# Paper intraday session execution (I3)

The paper-only session endpoint replays one stored NSE 15-minute session through the I1/I2 intraday strategy and replay path. It reads only `15m` candles for the requested symbol/date and accepts only registered `INTRADAY_*` strategies.

```http
POST /api/intraday/paper/sessions
Content-Type: application/json
```

```json
{
  "symbol": "TCS",
  "sessionDate": "2026-10-09",
  "strategyType": "INTRADAY_ORB",
  "asOfBarTime": "10:00",
  "riskConfig": {
    "initialCapital": 100000,
    "quantity": 1,
    "maxTradesPerDay": 3,
    "maxDailyLoss": 1000
  }
}
```

`riskConfig` is optional and defaults to the intraday backtest configuration. The independent intraday replay applies the per-session realized-loss and trade-count limits, enforces one open position at a time, and honors the kill switch before creating simulated orders. By default, a full-session replay force-flattens at the NSE 15:20 cutoff using the 15:15 bar-open proxy. An optional `asOfBarTime` must be one of the expected NSE bar starts; that replay uses completed bars before the selected time plus that bar's open, and can return an open position without applying the full-session square-off.

A full-session run requires every expected 15-minute bar start for the requested NSE session. An as-of run requires complete coverage only through the selected bar, so it can replay a partial current session without reading future bars. Missing required starts return `INCOMPLETE_DATA` with zero orders, positions, or trades; the engine does not simulate fills from a gapped prefix.

The response is the auditable paper ledger for that run: simulated entry/exit orders, linked open or closed position records, trade records carrying their position and entry/exit order IDs, costs, realized and unrealized P&L, session status, and explicit `liveBrokerCalls: 0`. Order fills are derived from replay bars; the service has no `BrokerClient` or live broker dependency. The result is returned synchronously and is not yet a durable session-history store. No live credentials, authentication, broker requests, or orders are used.

The Vue operator surface is available at `/intraday`. It displays session completeness/state, strategy outcomes, open and closed paper positions, cost totals, net-of-cost P&L, square-off state, and linked audit IDs. The “Load seeded session” action replays the TCS sample session and is useful for a local paper-mode demonstration. A separate retrospective active-watchlist scanner evaluates all registered intraday strategies at a required as-of bar time; see [Intraday paper watchlist scan](intraday-paper-scan.md).

Local verification uses deterministic 15-minute fixtures in broker/API tests; live Yahoo or broker calls are not part of the I3 tests.
