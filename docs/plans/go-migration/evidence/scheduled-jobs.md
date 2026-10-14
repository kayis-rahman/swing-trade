# Scheduled job characterization (Phase 0, steps 12a and part of 12)

Status: static characterization complete for all 15 `@Scheduled` methods. The runtime half of step 12a
(`app.scheduling.enabled=false` observed to stop every job) still needs a throwaway stack — see §5.

Derived by: `grep -rn "@Scheduled" backend --include=*.java` (15 hits, all in `src/main`), then a full read of
each enclosing class plus the services, stores and repositories it calls, and a grep of `backend/*/src/test`
for each class name. No product code was changed.

## 1. All 15 jobs

| ID | Class (FQCN) | Method | Module | Trigger | Side effects | Guards | Java tests | Go target |
|---|---|---|---|---|---|---|---|---|
| F-SCHED-01 | `com.swingtrade.llm.service.SynthesisEvaluationJob` | `evaluatePendingSynthesis` | llm | cron `0 15 2 * * *` (hard-coded), `Asia/Kolkata` — [SynthesisEvaluationJob.java:36](../../../../backend/llm/src/main/java/com/swingtrade/llm/service/SynthesisEvaluationJob.java#L36) | writes `synthesis_evaluations` only; reads candles; no HTTP, no notify | `app.features.scheduler.enabled` (default true); unmeasured-row selection + `findBySymbolAndAnalysisDate` dedupe → re-run safe | `SynthesisEvaluationJobTest` (3 methods) | `llm/scheduler/synthesis-evaluation-job.go` |
| F-SCHED-02 | `com.swingtrade.broker.service.PaperTradingMonitorService` | `monitorPositions` | broker | cron `${paper.trading.monitor-cron:0 45 16 * * MON-FRI}`, `Asia/Kolkata` — [PaperTradingMonitorService.java:58](../../../../backend/broker/src/main/java/com/swingtrade/broker/service/PaperTradingMonitorService.java#L58) | writes `positions`, `paper_trading_orders`, `paper_trading_portfolio`, `paper_trading_portfolio_snapshots`; reads latest candle + price band; applies trailing/partial-exit policy | `properties.isRiskManagementEnabled()` + LONG/SL/target completeness + price-band block. No holiday guard, no "ran today" check | `PaperTradingMonitorServiceTest` (17 methods) | `broker/scheduler/paper-trading-monitor-service.go` |
| F-SCHED-03 | `com.swingtrade.broker.scheduler.PendingOrderExecutionScheduler` | `executePendingOrders` | broker | cron `${paper.trading.pending-order-cron:0 35 16 * * MON-FRI}`, `Asia/Kolkata` — [PendingOrderExecutionScheduler.java:49](../../../../backend/broker/src/main/java/com/swingtrade/broker/scheduler/PendingOrderExecutionScheduler.java#L49) | fills eligible pending `paper_trading_orders` (MARKET always, LIMIT vs bar open), writes position rows | eligibility re-check per run; no flag/holiday guard; integration tests disable with `paper.trading.pending-order-cron=-` | `PendingOrderExecutionSchedulerTest` (4 methods) | `broker/scheduler/pending-order-execution-scheduler.go` |
| F-SCHED-04 | `com.swingtrade.data.service.IntradayIngestionScheduler` | `ingestSessionBars` | data | cron `0 45 15 * * MON-FRI` (hard-coded), `Asia/Kolkata` — [IntradayIngestionScheduler.java:42](../../../../backend/data/src/main/java/com/swingtrade/data/service/IntradayIngestionScheduler.java#L42) | provider HTTP per watchlist symbol (15-minute bars), upserts `ohlcv_candles` | `intraday.ingestion.enabled`; `MarketCalendar.isNseTradingSession` holiday/weekend gate; per-symbol resume → idempotent | none for the scheduler (only `IntradayIngestionServiceTest`, service level) | `data/scheduler/intraday-ingestion-scheduler.go` |
| F-SCHED-05 | `com.swingtrade.broker.scheduler.PortfolioSnapshotScheduler` | `takeSnapshot` | broker | cron `${paper.trading.snapshot-cron:0 45 15 * * MON-FRI}`, `Asia/Kolkata` — [PortfolioSnapshotScheduler.java:24](../../../../backend/broker/src/main/java/com/swingtrade/broker/scheduler/PortfolioSnapshotScheduler.java#L24) | appends one `paper_trading_portfolio_snapshots` row per run | none — no flag, no holiday guard, no "ran today" check (append-only, so re-runs are safe but duplicate) | `PortfolioSnapshotSchedulerTest` (17 methods) | `broker/scheduler/portfolio-snapshot-scheduler.go` |
| F-SCHED-06 | `com.swingtrade.data.service.FyersSymbolMasterService` | `scheduledRefresh` | data | cron `0 45 8 * * MON-FRI` (hard-coded), `Asia/Kolkata` — [FyersSymbolMasterService.java:157](../../../../backend/data/src/main/java/com/swingtrade/data/service/FyersSymbolMasterService.java#L157) | unauthenticated GET of `https://public.fyers.in/sym_details/NSE_CM.csv`, then `deleteAllInBatch()` + `saveAll` on `fyers_symbol_master` | active-broker must be `fyers`; `@Transactional`; destructive wipe-and-replace (not idempotent in shape) | `FyersSymbolMasterServiceTest` (4 methods) | `data/scheduler/fyers-symbol-master-service.go` |
| F-SCHED-07 | `com.swingtrade.data.service.EodIngestionScheduler` | `ingestLatestForAll` | data | cron `0 30 16 * * MON-FRI` (hard-coded), `Asia/Kolkata` — [EodIngestionScheduler.java:91](../../../../backend/data/src/main/java/com/swingtrade/data/service/EodIngestionScheduler.java#L91) | per active watchlist symbol + `NIFTY50`, provider HTTP, writes `ohlcv_candles` and `price_band`; startup benchmark backfill | `data.eod.ingestion.enabled`; weekend + `isMarketClosed(yesterday)` gate; per-symbol try/catch. No lock, no "ran today" | `EodIngestionSchedulerTest` (6 methods) | `data/scheduler/eod-ingestion-scheduler.go` |
| F-SCHED-08 | `com.swingtrade.api.scheduler.CandidateScanHandoffDispatcher` | `dispatchPendingHandoffs` | api | `fixedDelayString = ${candidate-scan.handoff-dispatch-ms:60000}` (no zone) — [CandidateScanHandoffDispatcher.java:33](../../../../backend/api/src/main/java/com/swingtrade/api/scheduler/CandidateScanHandoffDispatcher.java#L33) | turns `COMPLETED`/`PENDING` candidate scans into `job_runs` + async pipeline + run-summary notification; writes `candidate_scan_runs.orchestration_*` | status state machine + active-run check + `ConcurrentRunException`; concurrency is only in-JVM `synchronized (dispatchLock)` — **no advisory lock** | `CandidateScanHandoffDispatcherTest` (5 methods) | `api/scheduler/candidate-scan-handoff-dispatcher.go` |
| F-SCHED-09 | `com.swingtrade.api.scheduler.SentimentEvaluationJob` | `evaluatePendingSentiments` | api | cron `0 0 2 * * *` (hard-coded), `Asia/Kolkata` — [SentimentEvaluationJob.java:65](../../../../backend/api/src/main/java/com/swingtrade/api/scheduler/SentimentEvaluationJob.java#L65) | writes `sentiment_accuracy`; reads sentiments, candles, benchmark; no HTTP, no notify | `app.features.scheduler.enabled`; `existsBySymbolAndAnalysisDate` per-record dedupe | **none** — no test references the scheduler or `triggerEvaluation` | `api/scheduler/sentiment-evaluation-job.go` |
| F-SCHED-10 | `com.swingtrade.api.scheduler.WeeklySectorDigestScheduler` | `sendWeeklySectorDigest` | api | cron `0 0 18 * * SUN` (hard-coded), `Asia/Kolkata` — [WeeklySectorDigestScheduler.java:33](../../../../backend/api/src/main/java/com/swingtrade/api/scheduler/WeeklySectorDigestScheduler.java#L33) | Discord webhook POST with a sector digest generated from stored sentiments; no DB writes | `app.features.scheduler.enabled` only; no "already sent" check → re-fire re-posts | `WeeklyDigestE2ETest` is `@Disabled` (needs Docker) and tests `SentimentService`, not the scheduler | `api/scheduler/weekly-sector-digest-scheduler.go` |
| F-SCHED-11 | `com.swingtrade.api.scheduler.CandidateScanScheduler` | `runScheduledScan` | api | cron `0 0 17 * * MON-FRI` (hard-coded), `Asia/Kolkata` — [CandidateScanScheduler.java:44](../../../../backend/api/src/main/java/com/swingtrade/api/scheduler/CandidateScanScheduler.java#L44) | creates a `candidate_scan_runs` row, backfills market data per symbol, writes `candidate_scan_results`, persists a `PENDING` handoff for F-SCHED-08 | `app.features.scheduler.enabled` + `candidate-scan.scheduler.enabled` + `existsByStatus("RUNNING")` guard. **Not idempotent**: every fire is a new run row | `CandidateScanSchedulerTest` (4 methods) | `api/scheduler/candidate-scan-scheduler.go` |
| F-SCHED-12 | `com.swingtrade.api.service.MonthlyReportService` | `generateMonthlyReport` | api | cron `0 0 3 1-7 * SUN` (hard-coded), `Asia/Kolkata` — [MonthlyReportService.java:47](../../../../backend/api/src/main/java/com/swingtrade/api/service/MonthlyReportService.java#L47) | two full-table `findAll()` scans and one `log.info`; **no Discord, no PDF, no DB write** | `app.features.scheduler.enabled` only; no "already ran this month" check | **none** | `api/scheduler/monthly-report-service.go` |
| F-SCHED-13 | `com.swingtrade.api.scheduler.JobRunScheduler` | `runScheduledPipeline` | api | cron `0 0 18 * * MON-FRI` (hard-coded), `Asia/Kolkata` — [JobRunScheduler.java:42](../../../../backend/api/src/main/java/com/swingtrade/api/scheduler/JobRunScheduler.java#L42) | heaviest write path: `job_runs` + `job_run_stages` (7 stages × symbols), signals, backtests, paper trades, provider HTTP, LLM calls, run-summary notification | `app.features.scheduler.enabled`; candidate-scan cross-guard `hasActiveRun() \|\| hasPendingHandoff()`; `findActiveRun()` fast path; `runStartLock` + `ConcurrentRunException` | `JobRunSchedulerTest` (3 methods; the handoff-guard branch is untested because the test constructor passes `null` for `candidateScanService`) | `api/scheduler/job-run-scheduler.go` |
| F-SCHED-14 | `com.swingtrade.api.scheduler.WeeklyStrategyDigestScheduler` | `sendWeeklyStrategyDigest` | api | cron `${strategy.report.weekly-digest.cron:0 30 18 * * SUN}`, `Asia/Kolkata` — [WeeklyStrategyDigestScheduler.java:40](../../../../backend/api/src/main/java/com/swingtrade/api/scheduler/WeeklyStrategyDigestScheduler.java#L40) | Discord embed with the strategy attribution report (read-only over signals, selections, closed trades) | `strategy.report.weekly-digest.enabled` — **default `false`**, never set in any properties file → never fires | `WeeklyStrategyDigestSchedulerTest` (3 methods) | `api/scheduler/weekly-strategy-digest-scheduler.go` |
| F-SCHED-15 | `com.swingtrade.api.service.JobOrchestratorService` | `reapOrphanedRuns` | api | `initialDelayString = ${job.orchestrator.reaper.initial-delay-ms:15000}`, `fixedDelayString = ${job.orchestrator.reaper.interval-ms:300000}` — no cron/zone — [JobOrchestratorService.java:1157](../../../../backend/api/src/main/java/com/swingtrade/api/service/JobOrchestratorService.java#L1157) | force-fails stale `job_runs` → `FAILED` and their RUNNING `job_run_stages` → `ERROR`; records a `JobOrchestratorMetrics` run-reaped counter. Sibling `reapAllRunningRunsOnStartup` runs on `ApplicationReadyEvent` | `job.orchestrator.reaper.enabled` (default true) + a staleness threshold `STAGE_TIMEOUT_SUM_SECONDS × ceilDiv(symbols, maxConcurrent) × 6`; idempotent by construction (only RUNNING rows past threshold) | `JobOrchestratorServiceTest#ReapOrphanedRuns` (4) and `#ReapAllRunningRunsOnStartup` (4) | `api/scheduler/job-orchestrator-reaper.go` |

## 2. The scheduling on/off switch (step 12a)

There is exactly one `@EnableScheduling` in `backend/`: `com.swingtrade.data.config.SchedulingConfig`
([SchedulingConfig.java:8-9](../../../../backend/data/src/main/java/com/swingtrade/data/config/SchedulingConfig.java#L8)),
annotated `@ConditionalOnProperty(prefix = "app", name = "scheduling.enabled", havingValue = "true", matchIfMissing = true)`.
`SwingTradeApiApplication` component-scans `com.swingtrade.{api,broker,data,strategy,llm,core}`, so that one class gates all 15
jobs: `app.scheduling.enabled=false` removes the `ScheduledAnnotationBeanPostProcessor` for the whole context.

Findings that matter for the cutover (README decision: `app.scheduling.enabled=false` on Java, advisory locks in Go):

- `app.scheduling.enabled` appears in **no** properties file, so scheduling is always on unless explicitly disabled;
  `app.features.scheduler.enabled=true` is set in `application.properties:308`, `application-stage.properties:144`
  and `application-local.properties:202`.
- The two switches are independent and only 6 of the 15 jobs read the second one. Setting
  `app.features.scheduler.enabled=false` alone leaves F-SCHED-02, 03, 04, 05, 06, 07, 08, 14 firing
  (data/broker jobs and the two property-flagged jobs). Only `app.scheduling.enabled=false` stops everything.
- Per-job flags that exist alongside the global switch: `data.eod.ingestion.enabled` (F-SCHED-07),
  `intraday.ingestion.enabled` (F-SCHED-04), `strategy.report.weekly-digest.enabled=false` (F-SCHED-14, off),
  `job.orchestrator.reaper.enabled=true` (F-SCHED-15).
- No test anywhere asserts either switch (`grep` for `SchedulingConfig` under `src/test` returns nothing), so the
  "scheduling-off test" of step 12a has no existing Java coverage to port — it must be written for the Go side and,
  before cutover, for the Java side.

## 3. Overlap, duplication and ordering hazards

1. **Every Monday EOD run is skipped.** F-SCHED-07 ingests *yesterday*, and on Monday yesterday is Sunday, which
   `NseHolidayService.isMarketClosed` returns `true` for
   ([NseHolidayService.java:36-42](../../../../backend/data/src/main/java/com/swingtrade/data/service/NseHolidayService.java#L36)).
   Friday's post-close candle is therefore never ingested by the scheduler. A Go port that "fixes" this by checking
   today instead changes behavior; keep the skip and let the captain decide.
2. **Snapshot double-write.** F-SCHED-05 appends a snapshot at 15:45 and F-SCHED-02 appends another at 16:45 into the
   same `paper_trading_portfolio_snapshots` table.
3. **Chained jobs without a completion signal.** F-SCHED-04 and F-SCHED-05 both fire at 15:45; F-SCHED-03 (16:35)
   depends on the bar F-SCHED-07 (16:30) just fetched, and silently skips a pending order when the bar is missing
   ([PendingOrderExecutionScheduler.java:52](../../../../backend/broker/src/main/java/com/swingtrade/broker/scheduler/PendingOrderExecutionScheduler.java#L52)).
4. **17:00 → 18:00 candidate-scan handoff.** F-SCHED-11 persists a `PENDING` handoff that F-SCHED-08 (60 s fixedDelay)
   turns into an orchestrator run — the same entry point F-SCHED-13 uses at 18:00. Only the cross-guard
   `hasActiveRun() || hasPendingHandoff()` keeps them apart; F-SCHED-08's lock is `synchronized` in-JVM only, so
   multi-instance deployment can double-dispatch. This is the concrete instance of risk R4 in
   [risks.md](../risks.md) and the reason the cutover needs Go advisory locks.
5. **Sunday 18:00 / 18:30 digest pair.** F-SCHED-10 and F-SCHED-14 land 30 minutes apart; safe today only because
   F-SCHED-14 defaults to off.
6. **Two schedulers fire every day of the week.** F-SCHED-01 (`0 15 2 * * *`) and F-SCHED-09 (`0 0 2 * * *`) have no
   day-of-week restriction, unlike the `MON-FRI` crons.
7. **`0 0 3 1-7 * SUN` restricts both day-of-month and day-of-week.** Spring's cron ORs those two fields, so F-SCHED-12
   can fire twice in a month. It has no side effects beyond a log line, but a Go port using a strict "first Sunday"
   rule is not parity.
8. **Unguarded manual entry points.** `SentimentEvaluationJob.triggerEvaluation` (reached from
   `POST /api/sentiment/evaluate/trigger`) does not check `app.features.scheduler.enabled`
   ([SentimentEvaluationJob.java:77-80](../../../../backend/api/src/main/java/com/swingtrade/api/scheduler/SentimentEvaluationJob.java#L77)),
   and `FyersSymbolMasterService.refreshIfEmpty` is an `ApplicationReadyEvent` listener that bypasses the broker guard
   for the `AdminController`-constructed instance ([AdminController.java:47](../../../../backend/api/src/main/java/com/swingtrade/api/controller/AdminController.java#L47)).
9. **Legacy 2-arg constructors pass `null`** for optional collaborators (`SentimentEvaluationJob.java:60-63`,
   `JobRunScheduler.java:38`, `CandidateScanScheduler.java:38-42`) and silently degrade behavior. The Go ports need
   the same variants or they will not boot.
10. **No distributed lock anywhere.** No job uses a DB advisory lock; F-SCHED-08 is the only one with a lock, and it is
    JVM-local.

## 4. Test coverage gaps for the port (feeds the traceability matrix)

Schedulers with no test that touches them at all: F-SCHED-09, F-SCHED-12. Scheduler-without-tests where only the
underlying service is tested: F-SCHED-04, F-SCHED-10. Disabled scheduler tests that carry real knowledge:
`DailySchedulerIntegrationTest` ([api/src/test/.../DailySchedulerIntegrationTest.java:70](../../../../backend/api/src/test/java/com/swingtrade/api/scheduler/DailySchedulerIntegrationTest.java#L70))
and `WeeklyDigestE2ETest` ([:44](../../../../backend/api/src/test/java/com/swingtrade/api/scheduler/WeeklyDigestE2ETest.java#L44)).

## 5. Open item (step 12a runtime half)

To close step 12a: boot the isolated local stack twice with identical fixtures, once with
`app.scheduling.enabled=true` and once with `app.scheduling.enabled=false`, and prove that no job ran in the second
boot (log lines and the absence of new `job_runs` / `candidate_scan_runs` / `paper_trading_portfolio_snapshots` /
`synthesis_evaluations` / `sentiment_accuracy` / `fyers_symbol_master` mutations over a window longer than the
60 s handoff delay). Add the same assertion as a Go test at Phase 1 step 7. This is blocked on the throwaway stack
recorded as unavailable in this environment; it is the only part of step 12a not done.
