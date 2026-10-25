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

## Porting notes (2026-10-25, worker v2)

- Java's indicator classes (`TechnicalIndicators`, `AdvancedTechnicalIndicators`,
  `TechnicalIndicatorsDouble`) are thin wrappers over `org.ta4j:ta4j-core:0.16`
  and every public method returns `Double`, so indicator values are float
  paths under G4: reproduce the TA4J algorithm and assert against the Java
  test expectations with tight tolerances. Decimal bit-exactness applies at
  the strategy, gate, and engine layers that consume these values.
- `TechnicalIndicatorsTest` (86 methods, nested per indicator) is mostly
  behavioral: nulls for short inputs, `IllegalArgumentException` for bad
  periods, boundary sizes, with only five exact-value assertions (SMA 102.0,
  EMA 100.0 twice, VolumeMA 1000000.0 twice). Port every behavioral branch
  first; the deeper numeric regression arrives with step 3 golden signals and
  step 6's parallel run. Do not add new Java probe classes to harvest
  vectors: the pipeline removed `FinancialGoldenProbe` and no equivalent is
  authorized.
- For algorithms whose seeding or smoothing is unclear (EMA's first value,
  Wilder-style RSI, TA4J's ATR), the exact implementation is in the cached
  jar `~/.gradle/caches/modules-2/files-2.1/org.ta4j/ta4j-core/0.16/` (no
  sources jar is cached); extract the class file and read `javap -c` output
  rather than guessing from generic descriptions.
- The `BigDecimal` and `Double` Java variants feed the same TA4J math through
  `doubleValue()`, so one Go implementation per indicator covers both where
  the Java classes only differ in input type.
