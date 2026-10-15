# Table ownership sheet (Phase 0, step 7)

Step 7 of [phase-0-characterization.md](../phase-0-characterization.md) asks for a table ownership sheet —
writers, readers, retention, unique keys — and an explicit confirmation that ingestion is idempotent. The schema
half (the `pg_dump --schema-only` snapshot) is in [schema-snapshot.sql](schema-snapshot.sql); this document is the
code half. Everything here is static analysis of `backend/` at HEAD (`15766cca`) cross-referenced against that
snapshot. No product code was changed, and no database was needed: the ownership map is derivable from which
classes inject which repository and which methods they call.

| Fact | Value | Note |
|---|---|---|
| tables | **42** (41 application + `flyway_schema_history`) | matches `CREATE TABLE` count in [schema-snapshot.sql](schema-snapshot.sql) |
| JPA entity classes | **39** (35 `data/entity`, 3 `broker/entity`, 1 `llm`) | corrects the ledger's area row, which counted only `data/entity` |
| repository interfaces | **38** (34 `data`, 3 `broker`, 1 `llm`) | corrects the ledger's "39" |
| store facades | **15** (`data/store/*Impl.java`) | most tables are reachable only through one of these |
| tables with **no** production writer | **6** (`corporate_actions`, `news_items`, `nse_calendar_coverage`, `sentiment_evaluation_audits`, `trade_labels`, `universe_snapshots`) | §6 |
| foreign keys | **8** | §5 |
| retention/purge jobs | **0** — no `retention`, `purge`, `cleanup` or `prune` property exists in any properties file | §7 |
| Flyway rows | 53, `max(version)=77`, sequence `V1→V25` with `V71` missing | [golden-corpus.md](golden-corpus.md) §9 |

## 1. Method

1. `find backend -name "*Repository.java"` → 38 interfaces; each `JpaRepository<Entity, Id>` maps to one entity, and
   each entity's `@Table(name=…)` maps to one table. The mapping is 1:1 except for
   `candidate_scan_strategy_outcomes`, which has no entity of its own (it is a `@CollectionTable` child of
   `candidate_scan_results`), and `kill_switch`, which has no entity or repository at all.
2. For every class under `*/src/main`, collect the fields and constructor parameters whose type is one of those 38
   interfaces, then record every `field.method(` call site and classify it as a write (`save`, `saveAll`,
   `saveAndFlush`, `saveOrUpdate`, `delete*`, `insertIfAbsent`, `updateValue`, `incrementCompletedCount`,
   `clearCandidateScanRunId`) or a read (`find*`, `count*`, `exists*`, `search`). Production usage only — the test
   tree is excluded.
3. Cross-reference the resulting writers against the write endpoints of the 28 controllers and the side effects of
   the 15 `@Scheduled` jobs ([scheduled-jobs.md](scheduled-jobs.md) §1).
4. Take primary keys, unique constraints, indexes and foreign keys from the committed snapshot, not from the
   entities, so the parity surface is the schema Java actually produced.

"Writer"/"reader" below name the production class that calls the repository. Where a table is reached through a
`domain/store` facade, the facade is the access point and its callers are named in §3.

## 2. Ownership sheet

| Table | Entity · repository (module) | Production writers | Production readers |
|---|---|---|---|
| `app_settings` | AppSettingEntity · AppSettingRepository (data) | AppSettingsService `set`/`save`/`updateValue` | AppSettingsService `findByKey`/`findAll` — the only access point for every module |
| `backtest_result` | BacktestResultEntity · BacktestResultRepository (data) | BacktestResultStoreImpl `saveOrUpdate` ← BacktestStage (pipeline) | BacktestResultStoreImpl `findBySymbolAndRunDate` ← JobOrchestratorService:783 |
| `candidate_history_eligibility` | CandidateHistoryEligibilityEntity · repository (data) | CandidateScanService `save` | CandidateScanService `findById` |
| `candidate_scan_results` | CandidateScanResultEntity (+ child table) · repository (data) | CandidateScanService (9 `save` sites) | CandidateScanHandoffDispatcher `findByRunIdOrderBySymbolAsc`, CandidateScanService `search` |
| `candidate_scan_runs` | CandidateScanRunEntity · repository (data) | CandidateScanScheduler (F-SCHED-11), CandidateScanService, CandidateScanHandoffDispatcher (`orchestration_*`) | both of the above + JobRunController reads |
| `candidate_scan_strategy_outcomes` | **no entity** — `@CollectionTable` of CandidateScanResultEntity | only through the parent `save` (JPA cascade) | read back with the parent row; no standalone query exists |
| `corporate_actions` | CorporateActionEntity · repository (data) | **none** — `CorporateActionStoreImpl.save` has no production caller | BacktestEngine:266 (split/dividend adjustment) |
| `daily_loss_circuit_breaker_state` | DailyLossCircuitBreakerStateEntity · repository (data) | DailyLossCircuitBreaker `save` | DailyLossCircuitBreaker `findFirstByOrderByUpdatedAtDesc` |
| `flyway_schema_history` | — | Flyway only | Flyway validate/baseline |
| `fyers_symbol_master` | FyersSymbolEntity · FyersSymbolRepository (data) | FyersSymbolMasterService (`deleteAllInBatch` + `saveAll`) | FyersSymbolMasterService (search/count), CandidateScanService (universe filter) |
| `gate_effectiveness_audit` | GateEffectivenessAuditEntity · repository (data) | GateEffectivenessAuditService `save` (pipeline) | GateEffectivenessAuditService (gate-effectiveness reads) |
| `intraday_paper_sessions` | IntradayPaperSessionEntity · repository (data) | IntradayPaperController `save` (POST `/sessions`) | IntradayPaperController `findByRunId` (GET `/{runId}`) |
| `job_run_stages` | JobRunStageEntity · repository (data) | JobOrchestratorService `save`/`saveAll`, OrphanedRunReaper | JobOrchestratorService |
| `job_runs` | JobRunEntity · JobRunRepository (data) | JobOrchestratorService, CandidateScanHandoffDispatcher, OrphanedRunReaper; JobRunController `POST /start`, `/{runId}/cancel` | JobOrchestratorService, dispatcher, JobRunController reads |
| `kill_switch` | **no entity** — raw `JdbcTemplate` in broker/KillSwitchService | KillSwitchService `persistToDatabase` ← enable/disable ← AdminController `POST /kill-switch`, `/kill-switch/toggle` | KillSwitchService `loadFromDatabase`, `GET /api/admin/kill-switch/status`, RiskControlsService |
| `llm_analysis_audit` | LlmAnalysisAuditEntity · repository (data) | llm/SentimentService `save` (one row per LLM call) | **none** — write-only audit |
| `llm_analysis_result` | LlmAnalysisResultEntity · repository (data) | LlmAnalysisResultStoreImpl ← JobOrchestratorService | LlmAnalysisResultStoreImpl `findBySymbolAndAnalysisDate` ← JobOrchestratorService |
| `news_articles` | NewsArticleEntity · NewsArticleRepository (data) | NewsArticleStoreImpl `save` ← llm/NewsIngestionService `saveAll` | NewsArticleStoreImpl `findRecentBySymbol` / `findPersistedForDecisionWindow` / `countBySymbol` |
| `news_items` | NewsItemEntity · NewsItemRepository (data) | **none** | **none** — dead table and dead repository |
| `nse_calendar_coverage` | NseCalendarCoverageEntity · repository (data) | **none in Java** — seeded by `V30` and `V72` SQL | MarketCalendar |
| `nse_holidays` | NseHolidayEntity · NseHolidayRepository (data) | NseHolidayService `save` ← IngestionController `POST /holidays` | NseHolidayService (`isMarketClosed`), `GET /api/holidays*`, IntradayIngestionService |
| `ohlcv_candles` | OhlcvCandleEntity · OhlcvCandleRepository (data) | CandleStoreImpl `save`, DataIngestionService (`insertIfAbsent`, `deleteBySymbolAndDate`), IntradayIngestionService (`insertIfAbsent`) | CandleStoreImpl consumers: strategy (BacktestEngine), paper engine, intraday, exports, WatchlistService |
| `paper_trading_orders` | PaperTradingOrderEntity · repository (broker) | PaperTradingStateService `save` | PaperTradingStateService (portfolio reads) |
| `paper_trading_portfolio` | PaperTradingPortfolioEntity · repository (broker) | PaperTradingStateService, VariantPaperTradingService | PaperTradingStateService, PortfolioQueryServiceImpl (`GET /api/paper-portfolios`), VariantPaperTradingService |
| `paper_trading_portfolio_snapshots` | PaperTradingSnapshotEntity · repository (broker) | PaperTradingStateService `save` (append-only) — F-SCHED-02 and F-SCHED-05 both write | PaperTradingStateService (snapshot reads) |
| `pdf_extractions` | PdfExtractionEntity · PdfExtractionRepository (data) | llm/PdfExtractionService ← `POST /api/pdf/{symbol}/extract` | PdfExtractionService, SentimentApiController |
| `positions` | PositionEntity · PositionRepository (data) | PositionService (`closePosition`), PaperTradingStateService, VariantPaperTradingService, PositionStoreImpl | PositionService, PerformanceService, GateEffectivenessAuditService, MonthlyReportService, SignalArbiter, PortfolioQueryServiceImpl, VariantPaperTradingService, PaperTradingStateService |
| `price_band` | PriceBandEntity · PriceBandRepository (data) | DataIngestionService:274 (same ingestion path as candles) | BacktestEngine, PaperTradingMonitorService, PendingOrderExecutionScheduler, LiveEligibilityService |
| `reconciliation_audits` | ReconciliationAuditEntity · repository (data) | DataIngestionService:596 (`POST /api/ingestion/reconcile` with apply) | **none** — write-only audit |
| `sentiment_accuracy` | SentimentAccuracyEntity · SentimentAccuracyRepository (data) | SentimentAccuracyStoreImpl `save` ← api/SentimentEvaluationJob (F-SCHED-09) and `POST /api/sentiment/evaluate/trigger` | SentimentAccuracyService (`GET /api/sentiment/accuracy*`) |
| `sentiment_classification_log` | SentimentClassificationLogEntity · repository (data) | llm/SentimentService `save` (every classification) | llm/SentimentService `findByAnalysisDateBetweenAndShadowModeTrue` |
| `sentiment_evaluation_audits` | — | **none** | **none** — V35/V41 migrations only, no Java reference |
| `sentiment_results` | SentimentResultEntity · SentimentResultRepository (data) | SentimentStoreImpl `save`/`saveOrUpdate` ← llm/SentimentService:334 | SentimentStoreImpl consumers (`GET /api/sentiment/{symbol}/*`) |
| `signal_selection` | SignalSelectionEntity · SignalSelectionRepository (data) | api/SignalArbiter (one row per symbol per day; deletes superseded rows) | SignalArbiter (`GET /api/signal-selections*`) |
| `signals` | SignalEntity · SignalRepository (data) | SignalStoreImpl `save` ← SignalPersistenceService/SignalPipeline; SignalController generate/scan/delete endpoints | SignalStoreImpl consumers (`GET /api/signals*`), GateEffectivenessAuditService, MonthlyReportService |
| `stocks` | StockEntity · StockRepository (data) | WatchlistService:110 (watchlist add), SignalController.ensureStocksExist:458 (placeholder rows), PositionService:410 | StockStoreImpl consumers (WatchlistService, strategy universe) |
| `strategy_config` | StrategyConfigEntity · StrategyConfigRepository (data) | StrategyConfigService (`clearCurrent` JPQL update + `save`) ← StrategyConfigController PUT/DELETE | ActiveVariantService, JobOrchestratorService, PromotionEligibilityService, SignalArbiter, StrategyConfigService, store |
| `synthesis_evaluations` | SynthesisEvaluationEntity · SynthesisEvaluationRepository (llm) | SynthesisEvaluationService `save` (F-SCHED-01 and the pipeline) | SynthesisEvaluationService (`GET /api/synthesis/evaluations*`) |
| `trade_labels` | TradeLabel · TradeLabelRepository (data) | TradeLabelService — but no production caller reaches `labelTrade`/`bulkLabelTrades` | TradeLabelService — same, no production caller |
| `trades` | TradeEntity · TradeRepository (data) | PositionService:378/485 (close/open), PaperTradingStateService:269 (paper fill) | TradeStoreImpl consumers (PerformanceService, MonthlyReportService, TradeLabelService) |
| `universe_snapshots` | UniverseSnapshotEntity · repository (data) | **none** — `UniverseSnapshotStoreImpl.save` has no production caller | BacktestEngine:269 (point-in-time universe membership) |
| `watchlist` | WatchlistEntity · WatchlistRepository (data) | WatchlistService (add/remove/toggle) | WatchlistService, IntradayPaperScanService, EodIngestionScheduler, IntradayIngestionService |

## 3. Write surface: HTTP endpoints, scheduled jobs, delete paths

Only the tables that have a write surface are listed; every other table is written exclusively from inside the
pipeline (`JobOrchestratorService` and its stages) or not at all.

| Table | HTTP endpoints that write it | Scheduled writers | Delete paths |
|---|---|---|---|
| `app_settings` | `PUT /api/settings/{llm,gpuhub,discord,trading,scanning}`, `POST /api/settings/save`, `POST /api/settings/broker`, `PUT /api/candidate-scans/settings` (10 scan keys) | none | none — keys are never removed; `set` is a no-op when an environment variable overrides the key (AppSettingsService:64-68) |
| `backtest_result` | none — `BacktestController` injects no store, so `POST /api/backtest/run*` never persists | pipeline `BacktestStage` | none; the unique key makes re-runs overwrite |
| `candidate_scan_runs`, `candidate_scan_results`, `candidate_history_eligibility` | `POST /api/candidate-scans/{runId}/{analysis,cancel,pause,resume}`, `PUT /api/candidate-scans/settings` | F-SCHED-11 (17:00), F-SCHED-08 (60 s dispatch) | none — no delete endpoint, rows accumulate |
| `fyers_symbol_master` | `POST /api/admin/symbols/refresh` | F-SCHED-06 (08:45) and `refreshIfEmpty` on `ApplicationReadyEvent` | `deleteAllInBatch()` immediately before `saveAll` — wipe-and-replace |
| `intraday_paper_sessions` | `POST /api/intraday/paper/sessions` | none | none |
| `job_runs`, `job_run_stages` | `POST /api/job/runs/start`, `POST /api/job/runs/{runId}/cancel` | F-SCHED-08, F-SCHED-13, F-SCHED-15 (reaper writes status transitions only) | none — rows are never deleted |
| `kill_switch` | `POST /api/admin/kill-switch`, `POST /api/admin/kill-switch/toggle` | none | none |
| `llm_analysis_result`, `llm_analysis_audit`, `synthesis_evaluations`, `signal_selection`, `gate_effectiveness_audit` | `POST /api/analysis/analyze`, `POST /api/analysis/run-full` (orchestrated) | F-SCHED-13 pipeline; F-SCHED-01 for synthesis | superseded `signal_selection` rows are deleted by SignalArbiter |
| `nse_holidays` | `POST /api/holidays`, `DELETE /api/holidays/{date}` | none | the `DELETE /api/holidays/{date}` endpoint |
| `ohlcv_candles`, `price_band` | `POST /api/ingestion/{backfill,backfill-all,range,date,latest,reconcile}`, `POST /api/data/pull/incremental`, `POST /api/data/pull`, `POST /api/data/backfill` | F-SCHED-07 (16:30 EOD), F-SCHED-04 (15:45 intraday) | `reconcile` with apply deletes invalid stored bars per symbol+date (DataIngestionService:558) |
| `paper_trading_orders`, `paper_trading_portfolio`, `paper_trading_portfolio_snapshots`, `positions`, `trades` | `POST /api/positions/{symbol}/close` (positions + trades) | F-SCHED-02 (16:45), F-SCHED-03 (16:35), F-SCHED-05 (15:45) | none |
| `pdf_extractions` | `POST /api/pdf/{symbol}/extract` | none | none |
| `reconciliation_audits` | `POST /api/ingestion/reconcile` (apply only) | none | none |
| `sentiment_accuracy`, `sentiment_results`, `sentiment_classification_log`, `news_articles` | `POST /api/sentiment/evaluate/trigger`, `POST /api/sentiment/{symbol}/analyse` | F-SCHED-09 (02:00) | none |
| `signals` | `POST /api/signals/generate`, `/generate-all` (+`/stream`), `/scan`, `/price-action/{symbol}/generate`; `DELETE /api/signals/{symbol}`; `DELETE /api/signals` | F-SCHED-13 pipeline | `DELETE /api/signals/{symbol}` (every date for the symbol), `DELETE /api/signals` (all rows), and delete-then-insert on the DEFAULT and PRICE_ACTION paths (SignalPipeline:118, SignalController:235/315) |
| `stocks` | implicit only — watchlist add, `ensureStocksExist`, PositionService | none | none — but `ON DELETE CASCADE` from `stocks` removes that symbol's signals, positions and trades |
| `strategy_config` | `PUT /api/strategy-configs/{variantId}`, `PUT /api/strategy-configs/{variantId}/mode`, `DELETE /api/strategy-configs/{variantId}` | none | the `DELETE /{variantId}` endpoint |
| `watchlist` | `POST /api/watchlist`, `DELETE /api/watchlist/{symbol}`, `PATCH /api/watchlist/{symbol}/toggle` | none | the `DELETE /api/watchlist/{symbol}` endpoint |

## 4. Keys, uniqueness and idempotency

Keys and indexes are quoted from the snapshot, not from the entity annotations, because the migration history
sometimes adds a constraint the entity does not declare.

| Table | Primary key | Unique constraints | Idempotency mechanism (Java) |
|---|---|---|---|
| `app_settings` | `id` | `app_settings_key_key (key)` | `set` = `UPDATE` by key, `INSERT` when absent (AppSettingsService:69-74) |
| `backtest_result` | `id` | `uq_backtest_result_symbol_date (symbol, run_date)` | read-then-write (`saveOrUpdate` reuses the existing id) |
| `candidate_history_eligibility` | **`symbol`** (no surrogate id) | — | one row per symbol |
| `candidate_scan_results` | `id` | — (indexes on `run_id`, `run_id+qualified`) | **no guard**: a re-scan of the same run appends new result rows |
| `candidate_scan_runs` | `id` | `candidate_scan_runs_run_id_key (run_id)` | deliberately not idempotent: every fire is a new run row |
| `candidate_scan_strategy_outcomes` | **none** | — (index on `candidate_scan_result_id`) | child rows; `ON DELETE CASCADE` from the parent |
| `corporate_actions` | `id` | `uq_corporate_action_symbol_date_type (symbol, effective_date, action_type)` | never written |
| `daily_loss_circuit_breaker_state` | `id` | — | append-only; the latest row by `updated_at` wins |
| `flyway_schema_history` | `installed_rank` | — | Flyway |
| `fyers_symbol_master` | `id` | `fyers_symbol_master_fy_token_key (fy_token)`, `ux_fyers_symbol_master_symbol (fyers_symbol)` | wipe-and-replace — **not** idempotent under concurrency |
| `gate_effectiveness_audit` | `id` | **two identical** constraints on `(symbol, signal_date, gate_name, strategy)` | read-then-write in GateEffectivenessAuditService |
| `intraday_paper_sessions` | **`run_id`** (business key, no surrogate id) | — | one row per `runId` |
| `job_run_stages` | `id` | — | 7 stage rows per run; `ON DELETE CASCADE` from `job_runs` |
| `job_runs` | `id` | `job_runs_run_id_key (run_id)` | `runId` is a generated UUID business key |
| `kill_switch` | `id` | — | fixed `id = 1`: `UPDATE … WHERE id = 1` then `INSERT … id = 1` (KillSwitchService:105-117) |
| `llm_analysis_audit` | `id` | — | append-only |
| `llm_analysis_result` | `id` | `uq_llm_analysis_result_symbol_date (symbol, analysis_date)` | read-then-write by date |
| `news_articles` | `id` | `ux_news_articles_symbol_article_key (symbol, article_key)` | dedupe on ingest (article_key) |
| `news_items` | `id` | — (index on `lower(headline)`) | dead |
| `nse_calendar_coverage` | **`calendar_year`** | — | migration-seeded, never written at runtime |
| `nse_holidays` | `id` | `uq_nse_holiday_date (holiday_date)` | upsert by date |
| `ohlcv_candles` | `id` | `uq_ohlcv_candles_symbol_timeframe_date_bar_time (symbol, timeframe, date, bar_time)` | **the proven one**: `insertIfAbsent … ON CONFLICT (symbol, timeframe, date, bar_time) DO NOTHING` (OhlcvCandleRepository:37-47) |
| `paper_trading_orders` | `id` | `paper_trading_orders_order_id_key (order_id)` | business key |
| `paper_trading_portfolio` | `id` | `uq_paper_trading_portfolio_portfolio_id (portfolio_id)` | business key |
| `paper_trading_portfolio_snapshots` | `id` | — | **none** — append-only, so the 15:45 and 16:45 jobs write duplicate snapshots by design |
| `pdf_extractions` | `id` | — (index on `symbol, extraction_date`) | latest-per-symbol read; re-extraction appends |
| `positions` | `id` | — | **none** — duplicate open rows for one symbol are possible |
| `price_band` | `id` | `uq_price_band_symbol_date (symbol, band_date)` | re-ingest overwrites through the store |
| `reconciliation_audits` | `id` | — | append-only audit |
| `sentiment_accuracy` | `id` | — (index on `symbol, analysis_date`) | Java-side guard: `existsBySymbolAndAnalysisDate` before save |
| `sentiment_classification_log` | `id` | — | append-only |
| `sentiment_evaluation_audits` | `id` | — | dead |
| `sentiment_results` | `id` | `idx_sentiment_symbol_date (symbol, date)` — a unique constraint with an index-style name | read-then-write (`saveOrUpdate`) |
| `signal_selection` | `id` | `uq_signal_selection_symbol_date (symbol, selection_date)` | one winner per symbol per day; superseded rows deleted |
| `signals` | `id` | **none** | delete-then-insert in Java — no database-level guard at all |
| `stocks` | `id` | **two** unique constraints on `symbol` (`stocks_symbol_key`, `idx_stocks_symbol`) | read-then-write; placeholder rows are created on demand |
| `strategy_config` | `id` | `uq_strategy_config_variant_version (variant_id, version)`; partial unique indexes `ux_strategy_config_current (variant_id) WHERE is_current` and `ux_strategy_config_champion ((1)) WHERE is_current AND mode = 'CHAMPION'` | version-per-variant rows; the "current" flag is flipped with a JPQL bulk update (StrategyConfigService:148) |
| `synthesis_evaluations` | `id` | `uq_synthesis_evaluation_symbol_date (symbol, analysis_date)` | read-then-write by date |
| `trade_labels` | `id` | — | dormant |
| `trades` | `id` | — | append-only per position |
| `universe_snapshots` | `id` | `uq_universe_snapshot_symbol_date (symbol, snapshot_date)` | never written |
| `watchlist` | `id` | **two** unique constraints on `symbol` (`watchlist_symbol_key`, `idx_watchlist_symbol`) | read-then-write; `V1` seeds 20 symbols |

### Ingestion idempotency, confirmed

- `ohlcv_candles` is the only table whose idempotency is enforced by the database: both ingestion services
  (`DataIngestionService`, `IntradayIngestionService`) write exclusively through `insertIfAbsent`, whose
  `ON CONFLICT (symbol, timeframe, date, bar_time) DO NOTHING` makes a re-run of the same day a no-op.
- Every other write path is idempotent only in Java, and only in the single-JVM sense: read-then-write
  (`app_settings`, `backtest_result`, `llm_analysis_result`, `sentiment_results`, `synthesis_evaluations`,
  `strategy_config`, `price_band`, `watchlist`, `stocks`) or delete-then-insert (`signals`).
- Reconciliation is the one place where Java *deletes* stored data: `DataIngestionService:558` removes per-bar rows
  that the provider says are invalid, then re-inserts, and only then appends a `reconciliation_audits` row.

## 5. Foreign keys and insert ordering

| Constraint | Columns | Target | On delete |
|---|---|---|---|
| `fk_signals_stock` | `signals.symbol` | `stocks.symbol` | CASCADE |
| `fk_positions_stock` | `positions.symbol` | `stocks.symbol` | CASCADE |
| `fk_trades_stock` | `trades.symbol` | `stocks.symbol` | CASCADE |
| `fk_trades_position` | `trades.position_id` | `positions.id` | SET NULL |
| `fk4eegadsq0r26prbtdlw9oqp6x` | `trade_labels.position_id` | `positions.id` | — (Hibernate-generated constraint name, no `ON DELETE` action) |
| `job_run_stages_run_id_fkey` | `job_run_stages.run_id` | `job_runs.run_id` | CASCADE |
| `fk_candidate_scan_result_run` | `candidate_scan_results.run_id` | `candidate_scan_runs.run_id` | — |
| `candidate_scan_strategy_outcomes_candidate_scan_result_id_fkey` | `candidate_scan_strategy_outcomes.candidate_scan_result_id` | `candidate_scan_results.id` | CASCADE |

Ordering rules the Go port inherits:

1. A `stocks` row must exist before any `signals`, `positions` or `trades` row for that symbol. Java enforces this
   lazily: `SignalController.ensureStocksExist` creates a placeholder stock (`isin` null, `sector = OTHERS`) when a
   signal is generated for an unknown symbol, and `PositionService` writes one when a position is opened. The Go
   port needs the same lazy insert or the same insert ordering, and must reproduce the placeholder shape.
2. `job_runs` before `job_run_stages` (cascade); `candidate_scan_runs` before `candidate_scan_results` — note this
   FK points at the `run_id` **business key**, not the `id` primary key.
3. `positions` before `trades`; closing a position sets no FK, but deleting a position nulls `trades.position_id`.
4. Nothing deletes a `stocks` row today. If the Go port ever does, it silently destroys that symbol's signals,
   positions and trades.

### 5.1 State that lives outside the database

Two write surfaces are not in the schema snapshot at all, and a port that mirrors only Postgres loses them:

| Surface | Writer | Reader | Notes |
|---|---|---|---|
| `backtest.reports.dir` (default `reports`), files `backtest_\d{8}_\d{6}.json` | `BacktestController` run endpoints | `GET /api/backtest/reports`, `GET /api/backtest/reports/{filename}` | filename is pattern-validated and resolved with a `startsWith(dir)` guard, so the traversal is blocked twice; a missing directory returns `[]` (BacktestController:198, 225) |
| per-run trade-list CSV | `BacktestEngine:975` | nothing over HTTP | another file-system side effect of the same run endpoints |
| `ledger.csv` and `report.md` in a readiness directory | `IntradayDataReadinessCommand` (CLI runner, not HTTP) | operator | data-module command; not part of the API surface |

## 6. Tables with no production writer

| Table | Why it matters |
|---|---|
| `corporate_actions` | `CorporateActionStoreImpl.save` has no caller, so BacktestEngine's split/dividend adjustment reads a table nothing writes (empty unless it was populated by hand). **Do not** start writing it in Go: adjusted backtests would no longer match Java's unadjusted ones. |
| `universe_snapshots` | `UniverseSnapshotStoreImpl.save` has no caller; BacktestEngine reads point-in-time membership, so every vintage resolves to "no snapshot" and falls through to the current universe. Same warning as above. |
| `trade_labels` | `TradeLabelService.labelTrade`/`bulkLabelTrades` have no production caller, so the attribution exit-reason feature is dormant; the repository and its reads stay unreachable. |
| `news_items` | Entity and repository exist; neither is used. Superseded by `news_articles`. |
| `sentiment_evaluation_audits` | Created by `V35`/`V41`; no Java class references it at all. |
| `nse_calendar_coverage` | Read by `MarketCalendar`; seeded only by `V30`/`V72`. An empty or superseded seed changes which years the market calendar treats as verified. |

Two more tables are written but never read, so they are pure audit sinks: `llm_analysis_audit` (one row per LLM
call) and `reconciliation_audits` (one row per applied reconcile). Neither has an endpoint that reads it back; the Go
port may keep them as audit tables but must not expose them as new read surface without a decision.

## 7. Retention

There is **no retention behaviour to port**. No property containing `retention`, `purge`, `cleanup` or `prune`
exists in any `application*.properties`, no `@Scheduled` job deletes history, and every delete in the codebase is the
endpoint- or reconcile-driven path listed in §3. `SignalRepository.deleteByDate(cutoff)` exists on the store interface
(`core/domain/store/SignalStore.java:136`) but has no production caller — it is a retention hook that was never
wired. Adding retention in Go would be a behaviour change, not parity, and would change the shadow comparison in
Phase 2.

Two unbounded-growth tables to record for the captain: `job_runs` + `job_run_stages` (never deleted, one run per
pipeline execution, 7 stage rows per symbol per run) and `paper_trading_portfolio_snapshots` (two rows per trading
day per portfolio, with no unique key).

## 8. Findings for the Go port

1. **Only `ohlcv_candles` has database-enforced idempotency.** Everything else is read-then-write or
   delete-then-insert inside one JVM. The plan's cutover model (advisory locks in Go) is what makes the rest safe;
   the Go port should reproduce these semantics with `INSERT … ON CONFLICT` plus an advisory lock rather than
   porting the read-then-write races.
2. **`signals` and `positions` have no unique constraint at all.** Duplicate rows for the same symbol/date/strategy
   are reachable in Java today. A Go port that adds a unique index (or an upsert that silently dedupes) changes
   observable counts on `GET /api/signals*` and `GET /api/positions*`.
3. **`kill_switch` is the only actively-written application table without an entity.** It is raw `JdbcTemplate` SQL, fixed `id = 1`, and the
   service tolerates a `null` JdbcTemplate (in-memory only, silently). Any Go equivalent must keep the same
   single-row shape and the same degradation, or the kill switch survives restarts in one stack and not the other.
4. **`stocks` is an FK parent with `ON DELETE CASCADE`** for `signals`, `positions` and `trades`, and Java hides that
   dependency behind lazy placeholder inserts. Ordering matters more than the schema in the port.
5. **Duplicate unique constraints exist on `stocks.symbol`, `watchlist.symbol` and `gate_effectiveness_audit`**, and
   `ohlcv_candles` duplicates two identical `symbol, date` indexes. The goose `V76` no-op baseline must reproduce the
   constraints (and ideally their names) so a rollback diff against the Flyway schema stays empty.
6. **`strategy_config` encodes business rules as partial unique indexes** (`ux_strategy_config_current`,
   `ux_strategy_config_champion`): at most one current row per variant and at most one champion overall. A Go schema
   without them loses the invariant that `ActiveVariantService` and the champion mode rely on.
7. **`paper_trading_portfolio_snapshots` is append-only with two writers** (15:45 and 16:45), so duplicate daily
   snapshots are correct output, not a bug to fix.
8. **HTTP backtests are not persisted.** `BacktestController` injects no store or repository; only the pipeline's
   `BacktestStage` writes `backtest_result`. A Go port that persists on `POST /api/backtest/run` changes the data.
9. **`AppSettingsService.set` refuses to write a key that an environment variable overrides**
   (AppSettingsService:64-68), so a deployed container's DB settings can be silently read-only. The Go settings
   service must keep the env-over-db precedence.
10. **`fyers_symbol_master` refresh is a non-atomic wipe-and-replace** (`deleteAllInBatch` then `saveAll`), so a
    concurrent refresh or a mid-refresh crash leaves the table empty; `refreshIfEmpty` on startup is the recovery.
11. **A tracked backup file sits inside a source package**: `backend/data/src/main/java/com/swingtrade/data/repository/OhlcvCandleRepository.java.bak2`
    (142 lines, an older copy of the repository, tracked since `196905a6`). A mechanical "one Go file per Java
    repository" port must skip it. Worth deleting in the Java tree before G8.
12. **The plan's area-table counts were wrong and are now corrected**: 42 tables, 39 entity classes, 38 repository
    interfaces and 53 migrations reaching V77 — not the "35 entities / 37 repositories / 76 migrations" the area
    table claimed. Any future citation of those numbers must come from this sheet.
13. **Nothing reads `llm_analysis_audit` or `reconciliation_audits` back.** They are the only durable record of LLM
    calls and reconciles, which means the Go port's observability story depends on tables the Java dashboards never
    query.

## 9. What this document cannot answer (needs the running stack)

- Row counts and actual emptiness of the six never-written tables in §6 — the static analysis proves no writer
  exists, not that the tables are empty in the dev/stage databases.
- Growth figures for `job_runs`/`job_run_stages`/`paper_trading_portfolio_snapshots` (step 10's Pi measurement).
- The runtime half of step 12a still needs a scheduling-on/off pair of boots over the throwaway stack; this sheet
  gives that test its expected mutation set (`job_runs`, `candidate_scan_runs`, `paper_trading_portfolio_snapshots`,
  `synthesis_evaluations`, `sentiment_accuracy`, `fyers_symbol_master`).
