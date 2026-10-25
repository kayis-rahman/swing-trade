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

- Domain placement for the gate slice is already answered by the standards,
  not an open decision: `com.swingtrade.domain` types (`Signal.SignalType`,
  `SentimentResult`) belong in `go/core/domain/` (the core README reserves
  `domain` packages for them; strategy, llm, and data may all import core).
  The GateEvaluator slice therefore needs `core/domain` signal and sentiment
  types plus the strategy-local `StrategyDecision`, `RuleOutcome`,
  `GateOutcome`, `RegimeGateConfig`, `SentimentGateConfig`, and the
  `IndexSeries` interface (Java keeps it fakeable for tests; mirror that).
- Go naming: Java's `StrategyDecision` becomes `strategy.Decision` (package
  stutter rule); its `NewDecision` constructor keeps the Java validation
  messages verbatim with nolint annotations where staticcheck conflicts.
- Parameter validation resolved maps use Go-native value types (INT -> int,
  DECIMAL -> `coredecimal.Decimal` at financial scale, BOOL -> bool, ENUM ->
  string). Range messages render at financial scale where Java prints plain
  BigDecimal text; the Java tests pin only substring contracts (name plus
  bound keyword), so asserted parity holds.

- MarketContext recipe (the next big slice): it runs TA4J on **DecimalNum**
  (BigDecimal-backed Num), so G4's decimal bit-exactness lives here. Recover
  DecimalNum's add/subtract/multiply/divide scale rules from the cached ta4j
  jar bytecode before implementing EMA/RSI/ATR/SMA/highest/lowest/Bollinger/
  Keltner math. The slice also needs: `IndicatorKey` (plus the period and
  scaled cache keys), the core `OhlcvCandle` domain type with
  `adjustedForAnalysis()`, the intraday series build (timestamps + 15
  minutes), the `View` no-look-ahead guard with Java's exact
  `IllegalStateException` text, bbWidth's divide scale10 HALF_UP, the lazy
  `(indicator, period)` cache with `cachedIndicatorCount`, and
  `MarketContextTest`'s pinned behaviors. `UniformExitEvaluator` plus its
 141-line test follows once `MarketContext.View`, `StrategyParamsView`, and
  `OpenPosition` all exist.
