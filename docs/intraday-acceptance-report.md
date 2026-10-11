# Intraday I3–I5 acceptance report

## Acceptance scope

This report covers the paper-only intraday session runner, the operator cockpit, and a deterministic local acceptance scenario. The run consumes complete, stored NSE 15-minute candles, returns linked simulated order/position/trade records, applies the configured daily loss and trade-count limits, enforces one concurrent position, honors the kill switch, and leaves the UI on the complete session result. No live broker client, credentials, provider calls, or orders are involved.

The API contract and UI behavior are documented in [Paper intraday session execution](intraday-paper-execution.md).

## Reproducible data fixture

- Checked-in test fixture: [`tcs-2026-10-09-15m.csv`](../backend/broker/src/test/resources/com/swingtrade/broker/intraday/tcs-2026-10-09-15m.csv)
- Contract-level replay and risk tests: [`IntradayPaperExecutionServiceTest`](../backend/broker/src/test/java/com/swingtrade/broker/service/intraday/IntradayPaperExecutionServiceTest.java)
- API interval/strategy tests: [`IntradayPaperControllerTest`](../backend/api/src/test/java/com/swingtrade/api/controller/IntradayPaperControllerTest.java)
- Dashboard view states: [`IntradayView.test.ts`](../dashboard/src/views/IntradayView.test.ts)
- API request test: [`intraday.test.ts`](../dashboard/src/api/intraday.test.ts)

The fixture contains all 25 expected bar starts (09:15–15:15 IST) for Friday 2026-10-09. Its opening-range breakout produces one target-hit trade when run with `INTRADAY_ORB`, ₹10,000 starting capital, quantity 10, maximum one trade, and ₹1,000 maximum daily realized loss.

## Seeded completed-session result

| Measure | Result |
|---|---:|
| Session / strategy | TCS · 2026-10-09 · `INTRADAY_ORB` |
| Expected / received bars | 25 / 25 |
| Paper trades / filled orders / closed positions | 1 / 2 / 1 |
| Concurrent / open positions at end | 1 / 0 |
| Entry / exit | BUY 10 at ₹102.051 / SELL 10 at ₹104.0399739900 |
| Exit reason | `TARGET_HIT` |
| Round-trip costs | ₹47.57 |
| Net-of-cost P&L | -₹27.6802601000 |
| Live broker calls reported | 0 |

The result is intentionally instructive: the fixture's positive-price target exit is net-negative after the configured brokerage, exchange, tax, and slippage costs. The cockpit shows the closed paper position and order/trade references, while its open-position panel correctly reports zero positions for a completed replay.

A separate complete-session fixture tests forced square-off on the 15:15 bar-open proxy for the 15:20 cutoff. Risk tests verify that a ₹1 daily realized-loss limit stops further entries after a loss, a one-trade limit is obeyed, and an active kill switch emits no paper ledger. A partial session returns `INCOMPLETE_DATA` and does not synthesize fills or forced exits.

## Local demonstration

The local review used the same checked-in CSV to seed a throwaway PostgreSQL database in the `pi-node` Docker context, then ran the API and Vue dev server as ordinary Mac processes. The cockpit's **Load seeded session** action replayed those stored rows; simulated orders were returned in memory and were not persisted. The EOD, intraday ingestion, and startup-backfill schedulers were disabled to prevent provider calls while the review stack remains running.

- Review URL: `http://127.0.0.1:3003/intraday`
- Stack/replay report: `firstmate/data/swing-trade-intraday-i4-i5-cockpit-acceptance-v1/report.md`
- Completed UI screenshot: `firstmate/data/swing-trade-intraday-i4-i5-cockpit-acceptance-v1/screenshots/intraday-cockpit-completed.png`
- Empty-state screenshot: `firstmate/data/swing-trade-intraday-i4-i5-cockpit-acceptance-v1/screenshots/intraday-cockpit-empty.png`
- API response: `firstmate/data/swing-trade-intraday-i4-i5-cockpit-acceptance-v1/session-result.json`
- Database scope: throwaway compose project and volume only; shared/stage databases and Docker's global context were not used.

## Verification

Verification performed on the combined branch:

```bash
./bin/verify-changes

# Complete backend suite, independently confirmed:
cd backend && ./gradlew test --offline --no-daemon

# Complete dashboard unit suite (bounded workers avoid an unrelated parallel-load timeout):
cd dashboard && yarn vitest run --maxWorkers=2
```

The dashboard selected checks also run Prettier, typecheck, lint, and a production build. The final local verification output and browser result are in the external review record. CI is paused and is intentionally not used as evidence.

## Known limitations

- Session results are synchronous API responses, not a durable session-history store. Refreshing the page requires replaying the bars again.
- The page reports the completed historical replay state, not a live exchange feed or real-time market status.
- The full-session simulator force-flattens every position; a completed session normally has zero open positions. The cockpit exposes the open-position state and its empty case, plus the closed position ledger.
- Current risk is single-symbol and caps concurrency at one position; portfolio-wide, multi-symbol exposure and capital allocation are out of scope.
- Daily-loss control is based on realized session P&L. Partial/incomplete sessions are refused; missing expected bars do not create order or position records.
- API authentication remains deferred for the private MVP and is required before external exposure or live broker integration.
- No live trading, real credentials, live orders, provider calls, or staging deployment were performed.
