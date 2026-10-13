# Phase 4: Strategy and backtest engine

Goal: bit-exact results on decimal paths for indicators, strategies and backtests.

## Scope
~20 TA4j indicators in use, all strategies (pullback, RSI2, squeeze, volatility squeeze, 52-week high breakout, relative strength momentum, price action, intraday ORB/VWAP/RSI2), gates (data quality, regime, sentiment), param schema and validation, backtest, portfolio backtest, intraday backtest, walk-forward, DSR, cost models, promotion eligibility, exit policies.

## Steps
1. Ledger rows per indicator, strategy, engine and policy.
2. Port indicators first: red tests from Java golden vectors (including unstable-period, NaN and rounding behavior), then implement.
3. Strategies and gates: red from Java strategy tests plus golden signals over the full watchlist history.
4. Engines: red from Java engine tests; compare full trade lists, equity curves and metrics.
5. Property and fuzz tests for numeric edge cases.
6. Parallel run with Java on the full universe; diff outputs.
7. Performance comparison on the Pi.

## Gate G4 exit criteria
Zero diff on decimal paths; trade lists identical; documented tolerances only on float paths that are float in Java too; performance recorded.
