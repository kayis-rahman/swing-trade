# Intraday paper watchlist scan

The intraday cockpit can scan stored 15-minute bars for the active NSE watchlist at a required as-of bar time. This is a retrospective paper-analysis tool: it does not poll a live market, call a data provider, send broker orders, persist scan history, or rank symbols by profitability.

## Current behavior

- Universe: active watchlist rows whose exchange is NSE or unset. Other explicit exchanges are returned in `excludedSymbols`; the entire active watchlist is counted against `intraday.scan.max-watchlist-symbols` (default 20) so oversized scans fail closed instead of silently truncating.
- Strategies: every registered `INTRADAY_*` strategy, sorted by type, is evaluated for each included symbol. Current strategies are ORB, VWAP pullback, and RSI(2).
- As-of semantics: `asOfBarTime` is required and must be one of the expected NSE 15-minute bar starts. The scanner passes only bars through that time to the replay; the engine uses completed bars before it and only the selected bar's open. A candidate means a paper position is open at that selected time. A setup that already exited is not shown as an open candidate.
- Completeness: each symbol/strategy reports expected and received bars through the selected as-of time. A scan can run on a partial current session when the selected bar and all earlier expected bars are present. Missing bars through that cutoff and kill-switch blocks remain visible; neither creates a candidate.
- Results are synchronous and in-memory. No paper orders, positions, trades, or scan rows are persisted by this endpoint.

This first scanner uses the bounded active watchlist as a starter universe. It is not an intraday universe scanner with liquidity, freshness, or historical-coverage qualification, and it is not connected to the separate daily swing Candidate Explorer. Provider measurement, including Fyers, is outside this workflow.

## API

```http
POST /api/intraday/paper/scans
Content-Type: application/json
```

```json
{
  "sessionDate": "2026-10-09",
  "asOfBarTime": "10:00",
  "riskConfig": {
    "initialCapital": 10000,
    "quantity": 10,
    "maxTradesPerDay": 1,
    "maxDailyLoss": 1000
  }
}
```

`riskConfig` is optional and defaults to `IntradayBacktestConfig.defaults()`. The response reports `mode: PAPER_SCAN_AS_OF`, universe and coverage counts, `candidateCount`, `liveBrokerCalls: 0`, excluded symbols, and one summary row per symbol/strategy. It contains no generated broker order IDs.

The dashboard exposes the action in **Paper trading → Intraday paper**. Select a session date and as-of bar, then choose **Scan watchlist at selected time**. “Use in replay form” copies a candidate's symbol and strategy, plus the scan session date and normalized `HH:mm` cutoff, into the single-session form; the operator must still run the replay separately.

## Limits and tests

- `intraday.scan.max-watchlist-symbols` (environment override `INTRADAY_SCAN_MAX_WATCHLIST_SYMBOLS`) defaults to 20.
- Backend coverage: `IntradayPaperScanServiceTest`, `IntradayPaperScanControllerTest`.
- Dashboard coverage: `dashboard/src/api/intraday.test.ts`, `dashboard/src/views/IntradayView.test.ts`, and the Playwright acceptance flow in `dashboard/tests/e2e/views/intraday-view.spec.ts`.
- Run backend checks from `backend/`: `gradle :broker:test :api:test --offline --no-daemon`.
- Run dashboard checks from `dashboard/`: `yarn vitest run --maxWorkers=2`.
