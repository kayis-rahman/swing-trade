# Backtesting

`BacktestEngine` (backend `strategy` module, `com.swingtrade.strategy`) replays the Phase 2
price-action entry rules bar-by-bar against historical OHLCV candles to measure how the
strategy would have performed. It reuses the exact rule thresholds from
`PriceActionSignalEngine` so a backtest can never drift from the rules the live signal
generator applies.

## Entry rules (same as the live signal engine)

All 4 must hold on a given day for a trade to be entered at the **next** day's open:

1. `close > EMA20 > EMA50`
2. `RSI(14)` between 50 and 65
3. `volume > 1.5 × VolumeMA(20)`
4. `close` within 3% of the trailing 252-day high

## Exit rules (checked in this priority order, each day a position is open)

1. **STOP_LOSS** — day's low touches `entry - atrMultiplierStop × ATR(14)`
2. **TARGET_HIT** — day's high touches `entry + rewardRiskRatio × (entry - stopLoss)`
3. **SIGNAL_EXIT** — opt-in via `BacktestConfig.signalExitEnabled` (default `false`, so this is
   skipped entirely unless explicitly enabled). When enabled, fires on **any 1 of 3** conditions
   — the same exit confluence the live `PriceActionSignalEngine` uses to produce a SELL signal
   (see its class-level javadoc), so the two rule sets are never described inconsistently:
   - `close < EMA20`
   - `EMA20 < EMA50`
   - `RSI(14) < 50`
4. **TREND_BREAK** — `close < EMA20` for 2 consecutive days
5. **TIME_STOP** — held for `maxHoldingDays` bars

Position size is `floor(capital × riskPerTradePct / (entry - stopLoss))`.

## Triggering a backtest

```bash
# Single symbol, default config (see BacktestConfig.defaults())
curl -X POST "http://localhost:8080/api/backtest/run?symbol=RELIANCE&exchange=NSE"

# Every symbol in the active watchlist, saves a JSON+CSV report under ./reports
curl -X POST "http://localhost:8080/api/backtest/run-all?exchange=NSE"

# List saved reports (most recent first)
curl "http://localhost:8080/api/backtest/reports"

# Fetch a specific saved report
curl "http://localhost:8080/api/backtest/reports/backtest_20260705_020000.json"
```

A weekly job (`BacktestScheduler`) also runs `/run-all` automatically every Sunday at 02:00 IST
across the active watchlist.

`exchange` is accepted for API symmetry but does not currently filter candles — OHLCV data
isn't partitioned by exchange in the schema today.

## Paper-only intraday backtest

Intraday replay is a separate 15-minute-bar path; it does not call `BacktestEngine` or
`ZerodhaDeliveryCostModel`. The strategy module provides `IntradayOpeningRangeBreakoutStrategy`,
`IntradayVwapPullbackStrategy`, and `IntradayRsi2Strategy`, all implementing `SignalStrategy`, plus
`IntradayBacktestEngine` and the independent `IntradayCostModel`. A paper-only session execution API
now replays stored bars and returns a simulated audit ledger; see
[Paper intraday session execution](intraday-paper-execution.md) for its request and response contract.
It does not use broker authentication or place live orders.

To run the deterministic one-symbol sample locally:

```bash
cd backend
gradle :strategy:test --tests 'com.swingtrade.strategy.IntradayBacktestEngineTest'
```

For a different single symbol/dataset, load its timestamped 15-minute bars as `IntradayBar` values
and call `new IntradayBacktestEngine(nseSessionCalendar, new IntradayCostModel(),
IntradayBacktestConfig.defaults()).run(symbol, bars, strategy, params)`; the sample test demonstrates
the exact call and fixture format. Only normal NSE-session bars are replayed. The 15:15 bar is used
as the available-bar proxy for the 15:20 square-off (its open is the forced exit reference); no
closing-auction data is admitted. Daily loss and trade-count limits apply per session. The cost model
uses flat ₹20 brokerage per order, sell-side intraday STT, buy-side stamp duty, NSE transaction and
SEBI charges, GST, and configurable slippage; no delivery DP charge is applied.

## Interpreting `BacktestResult`

| Field | Meaning |
|---|---|
| `totalTrades` / `winningTrades` / `losingTrades` | Trade counts. A trade counts as a win if `pnl > 0`. |
| `winRate` | `winningTrades / totalTrades × 100`. |
| `avgGainPct` / `avgLossPct` | Average `pnlPct` across winners / average absolute `pnlPct` across losers. |
| `maxDrawdownPct` | Largest peak-to-trough drop in the equity curve. |
| `sharpeRatio` | `(mean daily return / stdev daily return) × √252`, computed from an equity curve that only changes on trade-exit days (no intraday mark-to-market of open positions). |
| `totalReturn` | `(finalCapital - initialCapital) / initialCapital × 100`. |
| `expectancy` | `winRate/100 × avgGainPct − (1 − winRate/100) × avgLossPct`, in percent-per-trade terms. |
| `trades` | Full list of `BacktestTrade` records (entry/exit dates, prices, stop/target, exit reason, pnl). |

`BacktestReportSummary` (returned by `/run-all` and saved as the JSON report) additionally
aggregates: top 10 symbols by win rate, top 10 by total return, and portfolio-wide
(trade-weighted) win rate and (mean) Sharpe ratio.

## Known limitations

- `BacktestConfig.maxConcurrentPositions` is reserved for a future multi-symbol, shared-capital
  portfolio simulation. Today each symbol is backtested independently with its own capital pool
  (`runBacktestAll` loops `runBacktest` per symbol), so at most one position is ever open per
  symbol — there's no cross-symbol capital contention to cap yet.
- The Sharpe ratio and drawdown use an equity curve that only updates at trade exits (no
  intraday mark-to-market of unrealized P&L on open positions), a common simplification for
  daily-bar swing backtests.
