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
  "riskConfig": {
    "initialCapital": 100000,
    "quantity": 1,
    "maxTradesPerDay": 3,
    "maxDailyLoss": 1000
  }
}
```

`riskConfig` is optional and defaults to the intraday backtest configuration. The independent intraday replay applies the per-session realized-loss and trade-count limits, enforces one open position at a time, and force-flattens at the NSE 15:20 cutoff using the 15:15 bar-open proxy. An active kill switch rejects the session before any simulated order is created.

The response is the auditable paper ledger for that run: simulated entry/exit orders, linked closed position records, the underlying trades, costs, realized P&L, session status, and explicit `liveBrokerCalls: 0`. Order fills are derived from replay bars; the service has no `BrokerClient` or live broker dependency. The result is returned synchronously and is not yet a durable session-history store. No live credentials, authentication, broker requests, or orders are used.

Local verification uses deterministic 15-minute fixtures in broker/API tests; live Yahoo or broker calls are not part of the I3 tests.
