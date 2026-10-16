# Prometheus metrics inventory (Phase 0, step 8)

Status: static inventory complete — every application-registered meter, its exported Prometheus series, every
consumer, and every dead or orphan series. The runtime half (scrape `/actuator/prometheus` from the stage API
container on pi-node and diff against §4/§5) is still pending Pi access — see §7.

Derived by: `grep -rn "Counter.builder\|Timer.builder\|Gauge.builder\|meterRegistry\." backend --include=*.java`
restricted to `src/main`, then a full read of the 11 files that register meters
(`backend/core/src/main/java/com/swingtrade/core/metrics/*.java`, `backend/api/src/main/java/com/swingtrade/api/metrics/*.java`,
`backend/api/src/main/java/com/swingtrade/api/config/ApiSecurityInterceptor.java`), a grep of every
`record*`/`setActive`/`registerGauge` call site to find the writers, a grep of `backend/*/src/test` for each
registering class to find the tests, and a full read of `infra/monitoring/**` plus the monitoring-sections of
`docs/plans/monitoring-stack.md`, `docs/infra/ci-cd-deployment.md` and `dashboard/tests/e2e/views/stage-sanity.spec.ts`
to find the consumers. No product code was changed.

## 1. Summary

| Measure | Count |
|---|---|
| Java classes registering meters | 11 (7 `core/metrics`, 3 `api/metrics`, 1 `api/config`) |
| Application meters (ledger rows) | 35 (20 counters, 8 gauges, 7 timers) → 42+ exported series: a meter with several label values exports one series per value |
| Meters with a Grafana panel or alert consumer | 17 (15 through Grafana panels, 2 through alert rules only) |
| Meters with no consumer at all | 18 |
| Meters registered but never incremented | 4 (`trades.duration`, `trades.pnl`, `signals.filtered`, `data.ingestion.duration`) — 7 exported series that can only ever be 0 |
| Prometheus queries that can never match a series | 4 (all in Grafana panel 9) |
| Grafana panels | 15 (1 dead panel, 14 with data) |
| Prometheus alert rules | 2 (`StrategySkipped`, `StageDegraded`) |
| Scrape jobs targeting this app | 1 (`swing-trade-stage`) |
| Platform (auto-configured) meter groups | expected from classpath; see §5 |

The plan's `F-OBS` ledger area expands to **45 rows**: 35 application meters (`F-OBS-01…35`) plus 10 platform
groups (`F-OBS-P01…P10`) that the Go port must reproduce or explicitly not reproduce.

## 2. Export and scrape path

- **Export endpoint**: `/actuator/prometheus`, enabled by `spring-boot-starter-actuator` +
  `micrometer-registry-prometheus` (`backend/api/build.gradle.kts:28,52`, also pulled into `core` at
  `backend/core/build.gradle.kts:9`).
- **Exposure** (`backend/api/src/main/resources/application.properties:262-284`):
  `management.endpoints.web.exposure.include=health,metrics,prometheus,info`, base path `/actuator`,
  `management.metrics.export.prometheus.enabled=true`, `step=60s`,
  `management.metrics.distribution.percentiles-histogram.http.server.requests=true`,
  `management.metrics.distribution.percentiles.http.server.requests=0.5,0.95,0.99`,
  `management.metrics.enable.all=true`,
  `management.metrics.tags.application=${spring.application.name}` (i.e. `swing-trade-api`).
  `application-stage.properties:117-124` adds `flyway` to the exposure list and changes nothing about metrics;
  `application-local.properties:152` exposes `env,configprops,threaddump` for local only.
- **Histograms are enabled for exactly one meter name**: `http.server.requests` (Spring MVC's auto timer). None of
  the 35 application meters publish buckets or client-side percentiles, so every application timer exports only
  `_count` and `_sum`.
- **The scrape endpoint is not behind the API gate.** `ApiSecurityWebConfig` registers `ApiSecurityInterceptor` on
  `/api/**` only ([ApiSecurityWebConfig.java:19](../../../../backend/api/src/main/java/com/swingtrade/api/config/ApiSecurityWebConfig.java#L19)),
  so `GET /actuator/prometheus` is unauthenticated (Prometheus sends no `X-API-Key`). The Go port must keep the same
  split — API-key boundary on `/api/**`, open actuator — or the stage scrape breaks.
- **Scrape config** (`infra/monitoring/prometheus.yml`): job `swing-trade-stage`,
  `metrics_path: /actuator/prometheus`, target `swing-trade-stage-api:8080`,
  labels `application: swing-trade-stage`, `module: api`, `scrape_interval: 15s`, `evaluation_interval: 15s`,
  `rule_files: /etc/prometheus/alerts.yml`.
- **Label collision (matters for parity)**: with `honor_labels` at its default `false`, Prometheus drops a
  target label when the scraped sample already exposes that label. Every application series already carries
  `application="swing-trade-api"` from `management.metrics.tags`, so the scrape job's
  `application: swing-trade-stage` never lands; only `module: api` does. Alert rules and dashboards must not
  assume `application="swing-trade-stage"` exists on this app's series.
- **Deployed config has drifted from the repo copy.** The config the runbook writes on the node
  (`docs/plans/monitoring-stack.md:48-85`) adds `caddy` (`:9187`) and `vllm` (external HTTPS) jobs that the repo
  `infra/monitoring/prometheus.yml` does not have, and the runbook snippet **omits `rule_files`**, so
  `alerts.yml` is probably not loaded on pi-node at all (confirm in §7 step 3). Deployed Prometheus joins both
  `pi-stack_monitoring` and `swing-trade-stage_swingtrade-network`
  (`docs/infra/ci-cd-deployment.md:14`), which is what makes `swing-trade-stage-api:8080` resolvable.
- **Second dashboard no longer exists**: `docs/infra/ci-cd-deployment.md:134` still points Grafana at
  `backend/monitoring/grafana/dashboards/jvm-spring-boot.json`, which was deleted in `d464e12c`
  (`git log --diff-filter=D -- backend/monitoring`). `infra/monitoring/provisioning/dashboards/swingtrade.json`
  is the only real consumer left, and `docker-compose.monitoring-stage.yml` (also cited there) does not exist
  either.

## 3. Consumer inventory

| ID | Consumer | What it queries |
|---|---|---|
| G1 | Grafana panel 1 "Active Positions" | `portfolio_active_positions` |
| G2 | Grafana panel 2 "Kill Switch" | `killswitch_active` |
| G3 | Grafana panel 3 "Active Job Runs" | `job_runs_active` |
| G4 | Grafana panel 4 "Total Trades Opened" | `trades_open_total` |
| G5 | Grafana panel 5 "Total Signals Generated" | `signals_generated_total` |
| G6 | Grafana panel 6 "Candles Ingested" | `data_candles_ingested_total` |
| G7 | Grafana panel 7 "Trade & Signal Rate (5m)" | `rate(trades_open_total[5m])`, `rate(trades_closed_total[5m])`, `rate(signals_generated_total[5m])` |
| G8 | Grafana panel 8 "Trade Outcomes by Reason" | `trades_outcome_total` (legend `{{reason}}`) |
| G9 | Grafana panel 9 "95th Percentile Latencies" | `llm_call_duration_bucket`, `sentiment_analysis_duration_bucket`, `job_run_duration_bucket`, `data_ingestion_duration_bucket` — **all four are dead, see §6 finding 1** |
| G10 | Grafana panel 10 "Job Run Completion" | `job_runs_completed_total`, `job_runs_failed_total` |
| G11 | Grafana panel 11 "Trades Per Symbol" | `trades_per_symbol_total` (legend `{{symbol}}`) |
| G12 | Grafana panel 12 "LLM Call Outcomes" | `llm_calls_total` (legend `{{result}}`) |
| G13 | Grafana panel 13 "Data Fetch Failures" | `data_fetch_failures_total` (legend `{{source}}`) |
| G14 | Grafana panel 14 "Sentiment Analysis" | `sentiment_analyses_completed_total`, `sentiment_analyses_failed_total` |
| G15 | Grafana panel 15 "Database Connection Pool" | `hikaricp_connections_active`, `hikaricp_connections_idle`, `hikaricp_connections_pending` |
| A1 | alert `StrategySkipped` (warning) | `increase(strategy_skipped_total[30m]) > 0`, labels `severity=warning`, `application=swing-trade-stage` |
| A2 | alert `StageDegraded` (warning) | `increase(stage_degraded_total[30m]) > 0`, same labels |
| E1 | Playwright `dashboard/tests/e2e/views/stage-sanity.spec.ts:61-67` | `GET ${STAGE_API_URL:-http://localhost:8080}/actuator/prometheus` → 200 and body contains `application_` and `jvm_` |

Dashboard/provider config: `infra/monitoring/provisioning/dashboards/{dashboards.yml,swingtrade.json}`,
`infra/monitoring/provisioning/datasources/prometheus.yml` (datasource `prometheus` at
`http://prometheus:9090`, 15s interval, POST). Timezone `asia/kolkata`.

## 4. Application meter inventory (35 meters)

Path convention: `core/…` = `backend/core/src/main/java/com/swingtrade/core/metrics/`,
`api/…` = `backend/api/src/main/java/com/swingtrade/api/metrics/`,
`api/config/…` = `backend/api/src/main/java/com/swingtrade/api/config/`.

`_total` on counters, `_seconds_count`/`_seconds_sum` on timers and no suffix on gauges are produced by
Micrometer's `PrometheusNamingConvention` (dots to underscores; `_total` appended for counters; base unit
`seconds` inserted for timers) — §8.

### 4.1 TradeMetrics — `core/…/metrics/TradeMetrics.java`

| ID | Series | Type | Unit | Labels | Registered | Written by | Java tests | Consumers |
|---|---|---|---|---|---|---|---|---|
| F-OBS-01 | `trades_open_total` | counter | — | `broker=paper` | [TradeMetrics.java:36](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/TradeMetrics.java#L36) | `TradeMetrics.recordTradeOpen` — [PaperTradingEngine.java:206](../../../../backend/broker/src/main/java/com/swingtrade/broker/engine/PaperTradingEngine.java#L206) | `TradeMetricsTest#recordsOpenCloseOutcomesDurationsPnlAndSymbols` | G4, G7 |
| F-OBS-02 | `trades_closed_total` | counter | — | `broker=paper` | [TradeMetrics.java:41](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/TradeMetrics.java#L41) | `recordTradeClose` — [PaperTradingEngine.java:620](../../../../backend/broker/src/main/java/com/swingtrade/broker/engine/PaperTradingEngine.java#L620), [883](../../../../backend/broker/src/main/java/com/swingtrade/broker/engine/PaperTradingEngine.java#L883) | same | G7 |
| F-OBS-03 | `trades_outcome_total` | counter | — | `reason` ∈ {`stop_loss`,`target_hit`,`time_stop`,`trend_break`,`manual`} | [TradeMetrics.java:46-69](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/TradeMetrics.java#L46) | `recordTradeClose` via a closed `switch` on `reason.toUpperCase()` | same | G8 |
| F-OBS-04 | `trades_duration_seconds_count`, `_sum` | timer | seconds | — | [TradeMetrics.java:71](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/TradeMetrics.java#L71) | **no production caller** (`recordTradeDuration` unused) | same | none — dead |
| F-OBS-05 | `trades_pnl_seconds_count`, `_sum` | timer | seconds | — | [TradeMetrics.java:75](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/TradeMetrics.java#L75) | **no production caller** (`recordTradePnL` unused) | same | none — dead |
| F-OBS-06 | `trades_per_symbol_total` | counter | — | `symbol` (= `signal.symbol()`, watchlist-bounded) | [TradeMetrics.java:109](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/TradeMetrics.java#L109) | `recordSymbolTrade` — [PaperTradingEngine.java:207](../../../../backend/broker/src/main/java/com/swingtrade/broker/engine/PaperTradingEngine.java#L207) | same | G11 |

### 4.2 JobOrchestratorMetrics — `core/…/metrics/JobOrchestratorMetrics.java`

| ID | Series | Type | Unit | Labels | Registered | Written by | Java tests | Consumers |
|---|---|---|---|---|---|---|---|---|
| F-OBS-07 | `job_runs_completed_total` | counter | — | — | [JobOrchestratorMetrics.java:29](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/JobOrchestratorMetrics.java#L29) | `recordRunCompleted` — [JobOrchestratorService.java:1073](../../../../backend/api/src/main/java/com/swingtrade/api/service/JobOrchestratorService.java#L1073) | `JobOrchestratorMetricsTest#shouldCountRunWithWarningsAsCompleted` | G10 |
| F-OBS-08 | `job_runs_failed_total` | counter | — | — | [JobOrchestratorMetrics.java:33](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/JobOrchestratorMetrics.java#L33) | `recordRunFailed` — [JobOrchestratorService.java:1075](../../../../backend/api/src/main/java/com/swingtrade/api/service/JobOrchestratorService.java#L1075), [1104](../../../../backend/api/src/main/java/com/swingtrade/api/service/JobOrchestratorService.java#L1104); `recordRunReaped` — [OrphanedRunReaper.java:42](../../../../backend/api/src/main/java/com/swingtrade/api/service/OrphanedRunReaper.java#L42) | `…#shouldTrackActiveRunsCompletedAndFailedCounters`, `…#shouldRecordReapedRunAsFailedWithoutTouchingActiveGauge` | G10 |
| F-OBS-09 | `job_run_duration_seconds_count`, `_sum` | timer | seconds | — | [JobOrchestratorMetrics.java:37](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/JobOrchestratorMetrics.java#L37) | `recordRunCompleted`/`recordRunFailed`/`recordRunFailed(0L)` | `…#shouldTrackActiveRunsCompletedAndFailedCounters` | G9 (dead query) |
| F-OBS-10 | `job_runs_active` | gauge | — | — | [JobOrchestratorMetrics.java:41](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/JobOrchestratorMetrics.java#L41) | `recordRunStarted` (+1), `recordRunCompleted`/`Failed`/`Cancelled` (−1) | `…#shouldTrackActiveRunsCompletedAndFailedCounters`, `…#shouldDecrementActiveGaugeOnCancel` | G3 |
| F-OBS-11 | `strategy_skipped_total` | counter | — | `variant`, `reason` ∈ {`UNSUPPORTED_TYPE`,`INSUFFICIENT_HISTORY`,`INVALID_PARAMS`,`OTHER`,`ERROR`} | [JobOrchestratorMetrics.java:91](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/JobOrchestratorMetrics.java#L91) | `recordStrategySkipped` — [JobOrchestratorService.java:712](../../../../backend/api/src/main/java/com/swingtrade/api/service/JobOrchestratorService.java#L712), [721](../../../../backend/api/src/main/java/com/swingtrade/api/service/JobOrchestratorService.java#L721), [726](../../../../backend/api/src/main/java/com/swingtrade/api/service/JobOrchestratorService.java#L726); codes from [ConfiguredSignalTally.java:105-111](../../../../backend/api/src/main/java/com/swingtrade/api/service/ConfiguredSignalTally.java#L105) | `JobOrchestratorMetricsTest#shouldCountSkippedStrategiesByVariantAndReason` | A1 |
| F-OBS-12 | `stage_degraded_total` | counter | — | `stage`, `reason` ∈ {`STRATEGY_ERROR`,`STRATEGY_SKIPPED`,`UNKNOWN`} | [JobOrchestratorMetrics.java:99](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/JobOrchestratorMetrics.java#L99) | `recordStageDegraded` — [JobOrchestratorService.java:586](../../../../backend/api/src/main/java/com/swingtrade/api/service/JobOrchestratorService.java#L586); codes from [ConfiguredSignalTally.java:54-58](../../../../backend/api/src/main/java/com/swingtrade/api/service/ConfiguredSignalTally.java#L54) | `JobOrchestratorMetricsTest#shouldCountDegradedStagesByStageAndReason` | A2 |

### 4.3 DataIngestionMetrics — `core/…/metrics/DataIngestionMetrics.java`

| ID | Series | Type | Unit | Labels | Registered | Written by | Java tests | Consumers |
|---|---|---|---|---|---|---|---|---|
| F-OBS-13 | `data_candles_ingested_total` | counter | — | — | [DataIngestionMetrics.java:26](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/DataIngestionMetrics.java#L26) | `recordCandleIngested` — [DataIngestionService.java:260](../../../../backend/data/src/main/java/com/swingtrade/data/service/DataIngestionService.java#L260), [IntradayIngestionService.java:167](../../../../backend/data/src/main/java/com/swingtrade/data/service/IntradayIngestionService.java#L167) | `DataIngestionMetricsTest#recordsCandlesDurationsFailuresAndRateLimitHitsBySource` | G6 |
| F-OBS-14 | `data_ingestion_duration_seconds_count`, `_sum` | timer | seconds | — | [DataIngestionMetrics.java:30](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/DataIngestionMetrics.java#L30) | **no production caller** (`recordIngestion` unused) | same | G9 (dead query) — dead |
| F-OBS-15 | `data_fetch_failures_total` | counter | — | `source` ∈ {`unknown`, active-broker name} | [DataIngestionMetrics.java:47](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/DataIngestionMetrics.java#L47) | `recordFetchFailure` — [DataIngestionService.java:266](../../../../backend/data/src/main/java/com/swingtrade/data/service/DataIngestionService.java#L266), [IntradayIngestionService.java:124](../../../../backend/data/src/main/java/com/swingtrade/data/service/IntradayIngestionService.java#L124) | same | G13 |
| F-OBS-16 | `data_rate_limit_hits_total` | counter | — | `source` (active broker, or `fyers`) | [DataIngestionMetrics.java:55](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/DataIngestionMetrics.java#L55) | `recordRateLimitWait` — [RateLimitedMarketDataClient.java:42](../../../../backend/data/src/main/java/com/swingtrade/data/service/RateLimitedMarketDataClient.java#L42); `recordRateLimitHit` — [FyersServiceClient.java:416](../../../../backend/data/src/main/java/com/swingtrade/data/service/FyersServiceClient.java#L416) | same | none |

### 4.4 LlmMetrics and SentimentMetrics — `core/…/metrics/`

| ID | Series | Type | Unit | Labels | Registered | Written by | Java tests | Consumers |
|---|---|---|---|---|---|---|---|---|
| F-OBS-17 | `llm_call_duration_seconds_count`, `_sum` | timer | seconds | — | [LlmMetrics.java:23](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/LlmMetrics.java#L23) | `recordCall` — [SentimentService.java:468](../../../../backend/llm/src/main/java/com/swingtrade/llm/service/SentimentService.java#L468), [491](../../../../backend/llm/src/main/java/com/swingtrade/llm/service/SentimentService.java#L491) | `LlmMetricsTest#recordsCallOutcomesDurationAndSentimentAnalyses` | G9 (dead query) |
| F-OBS-18 | `llm_calls_total` | counter | — | `result` ∈ {`success`,`failure`} | [LlmMetrics.java:27](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/LlmMetrics.java#L27), [32](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/LlmMetrics.java#L32) | `recordCall(duration, success)` | same | G12 |
| F-OBS-19 | `llm_sentiment_analyzed_total` | counter | — | — | [LlmMetrics.java:37](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/LlmMetrics.java#L37) | `recordSentimentAnalyzed` — [SentimentService.java:469](../../../../backend/llm/src/main/java/com/swingtrade/llm/service/SentimentService.java#L469) | same | none |
| F-OBS-20 | `sentiment_analyses_completed_total` | counter | — | — | [SentimentMetrics.java:22](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/SentimentMetrics.java#L22) | `recordCompleted` — [SentimentService.java:345](../../../../backend/llm/src/main/java/com/swingtrade/llm/service/SentimentService.java#L345) | `SentimentMetricsTest#recordsCompletedFailedAndTimedAnalyses` | G14 |
| F-OBS-21 | `sentiment_analyses_failed_total` | counter | — | — | [SentimentMetrics.java:26](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/SentimentMetrics.java#L26) | `recordFailed` — [SentimentService.java:351](../../../../backend/llm/src/main/java/com/swingtrade/llm/service/SentimentService.java#L351) | same | G14 |
| F-OBS-22 | `sentiment_analysis_duration_seconds_count`, `_sum` | timer | seconds | — | [SentimentMetrics.java:30](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/SentimentMetrics.java#L30) | `recordDuration` — [SentimentService.java:346](../../../../backend/llm/src/main/java/com/swingtrade/llm/service/SentimentService.java#L346), [352](../../../../backend/llm/src/main/java/com/swingtrade/llm/service/SentimentService.java#L352) | same | G9 (dead query) |

### 4.5 SignalMetrics — `core/…/metrics/SignalMetrics.java`

| ID | Series | Type | Unit | Labels | Registered | Written by | Java tests | Consumers |
|---|---|---|---|---|---|---|---|---|
| F-OBS-23 | `signals_generated_total` | counter | — | — | [SignalMetrics.java:24](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/SignalMetrics.java#L24) | `recordSignalGenerated` — [PriceActionSignalEngine.java:264](../../../../backend/strategy/src/main/java/com/swingtrade/strategy/PriceActionSignalEngine.java#L264) | `SignalMetricsTest#recordsGeneratedFilteredAndPerTypeSignals` | G5, G7 |
| F-OBS-24 | `signals_filtered_total` | counter | — | — | [SignalMetrics.java:28](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/SignalMetrics.java#L28) | **no production caller** (`recordSignalFiltered` unused) | same | none — dead |
| F-OBS-25 | `signals_by_type_total` | counter | — | `type` (lowercased signal type) | [SignalMetrics.java:45](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/SignalMetrics.java#L45) | `recordSignalType` — [PriceActionSignalEngine.java:265](../../../../backend/strategy/src/main/java/com/swingtrade/strategy/PriceActionSignalEngine.java#L265) | same | none |

### 4.6 KillSwitchMetrics — `core/…/metrics/KillSwitchMetrics.java`

| ID | Series | Type | Unit | Labels | Registered | Written by | Java tests | Consumers |
|---|---|---|---|---|---|---|---|---|
| F-OBS-26 | `killswitch_active` | gauge | — | 0/1 value from an internal `AtomicBoolean` | [KillSwitchMetrics.java:26](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/KillSwitchMetrics.java#L26) | `setActive` — [KillSwitchService.java:54](../../../../backend/broker/src/main/java/com/swingtrade/broker/risk/KillSwitchService.java#L54) (startup, after `loadFromDatabase`), [139](../../../../backend/broker/src/main/java/com/swingtrade/broker/risk/KillSwitchService.java#L139), [163](../../../../backend/broker/src/main/java/com/swingtrade/broker/risk/KillSwitchService.java#L163) | `KillSwitchMetricsTest#exposesCurrentStateAndCountsEachStateChange` | G2 |
| F-OBS-27 | `killswitch_state_changed_total` | counter | — | `state` ∈ {`enabled`,`disabled`} | [KillSwitchMetrics.java:37](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/KillSwitchMetrics.java#L37) | same as above, including the startup call | same | none |

### 4.7 ApiRequestMetricsFilter — `api/…/metrics/ApiRequestMetricsFilter.java`

| ID | Series | Type | Unit | Labels | Registered | Written by | Java tests | Consumers |
|---|---|---|---|---|---|---|---|---|
| F-OBS-28 | `api_request_duration_seconds_count`, `_sum` | timer | seconds | `method`, `status` | [ApiRequestMetricsFilter.java:31](../../../../backend/api/src/main/java/com/swingtrade/api/metrics/ApiRequestMetricsFilter.java#L31) | `doFilterInternal` closes the `Timer.Sample` in `finally`, for every request that reaches the servlet chain | `ApiRequestMetricsFilterTest#recordsDurationWithMethodAndStatusTagsEvenWhenHandlerFails` | none |

### 4.8 PiThermalMetrics — `api/…/metrics/PiThermalMetrics.java`

| ID | Series | Type | Unit | Labels | Registered | Written by | Java tests | Consumers |
|---|---|---|---|---|---|---|---|---|
| F-OBS-29 | `swingtrade_pi_cpu_temperature_celsius` | gauge | celsius | — | [PiThermalMetrics.java:31](../../../../backend/api/src/main/java/com/swingtrade/api/metrics/PiThermalMetrics.java#L31) | sysfs read of `/sys/class/thermal/thermal_zone0/temp` ÷1000; `NaN` when the file is absent | **none** — no test file exists for this class | **none** (see §6 finding 2) |
| F-OBS-30 | `swingtrade_pi_cpu_frequency_hertz` | gauge | hertz | — | [PiThermalMetrics.java:38](../../../../backend/api/src/main/java/com/swingtrade/api/metrics/PiThermalMetrics.java#L38) | sysfs read of `/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq` ×1000 (kHz); `NaN` when absent | none | none |

Both gauges use `strongReference(true)`; the `NaN` fallback is deliberate so startup succeeds on non-Pi hardware.

### 4.9 PortfolioMetrics — `api/…/metrics/PortfolioMetrics.java`

This is the **api-module** `com.swingtrade.api.metrics.PortfolioMetrics`, not
`com.swingtrade.strategy.PortfolioMetrics` (a record built by `MetricsCalculator`/`BacktestMetrics`, pure math, no
meters). The Go port must keep both, under different files.

| ID | Series | Type | Unit | Labels | Registered | Written by | Java tests | Consumers |
|---|---|---|---|---|---|---|---|---|
| F-OBS-31 | `portfolio_active_positions` | gauge | — | weak ref to the `TradingService` bean | [PortfolioMetrics.java:50](../../../../backend/api/src/main/java/com/swingtrade/api/metrics/PortfolioMetrics.java#L50) | samples `TradingService.getOpenPositions().size()` on scrape | `PortfolioMetricsTest#registersGaugesBackedByLivePaperState` | G1 |
| F-OBS-32 | `portfolio_total_pnl` | gauge | — | same | [PortfolioMetrics.java:56](../../../../backend/api/src/main/java/com/swingtrade/api/metrics/PortfolioMetrics.java#L56) | `TradingService.getTotalPnL().doubleValue()` | same | none |
| F-OBS-33 | `portfolio_daily_return` | gauge | — (percent, unlabeled) | same | [PortfolioMetrics.java:63](../../../../backend/api/src/main/java/com/swingtrade/api/metrics/PortfolioMetrics.java#L63) | `pnl ÷ initial × 100` with `HALF_UP`/scale 8, or `0.0` when initial is 0 | same | none |
| F-OBS-34 | `portfolio_value` | gauge | — | same | [PortfolioMetrics.java:23-25](../../../../backend/api/src/main/java/com/swingtrade/api/metrics/PortfolioMetrics.java#L23) | `TradingService.getTotalValue().doubleValue()` | same | none |

All four are registered only by the `@Autowired` constructor that takes a `TradingService`; the one-argument
"isolated metric tests" constructor registers nothing. Gauges hold weak references to the bean, which is safe only
because Spring keeps the singleton reachable.

### 4.10 ApiSecurityInterceptor — `api/config/ApiSecurityInterceptor.java`

| ID | Series | Type | Unit | Labels | Registered | Written by | Java tests | Consumers |
|---|---|---|---|---|---|---|---|---|
| F-OBS-35 | `api_rate_limit_hits_total` | counter | — | no labels | [ApiSecurityInterceptor.java:59](../../../../backend/api/src/main/java/com/swingtrade/api/config/ApiSecurityInterceptor.java#L59) | incremented once per request that exceeds the fixed 100/min window and is rejected with 429 ([ApiSecurityInterceptor.java:102](../../../../backend/api/src/main/java/com/swingtrade/api/config/ApiSecurityInterceptor.java#L102)) | `ApiSecurityInterceptorTest#limitsPublicRequestsAndExposesHitMetric` | none |

The registry is obtained through `ObjectProvider.getIfAvailable(SimpleMeterRegistry::new)`, so in a context without
a Prometheus registry the counter is silently discarded instead of failing startup.

## 5. Platform (auto-configured) meters

No application code registers these; they come from Spring Boot's actuator autoconfiguration over the
`spring-boot-starter-actuator`/`micrometer-core`/HikariCP/Logback stack on the api module classpath. E1 asserts
`application_` and `jvm_` exist in the live scrape. Everything in this section is "expected from classpath" and
must be confirmed by scraping the real output (§7); each is a single ledger row.

| ID | Series group | Source | Consumer | Notes |
|---|---|---|---|---|
| F-OBS-P01 | `http_server_requests_seconds_count`, `_sum`, `_bucket`, `quantile` | Spring MVC auto timer, `percentiles-histogram.http.server.requests=true` | none in this repo | The only histogram in the system; `p99`/`p50` client percentiles are also published |
| F-OBS-P02 | `jvm_memory_used_bytes`, `jvm_memory_max_bytes`, `jvm_memory_committed_bytes`, `jvm_gc_pause_seconds*`, `jvm_threads_*`, `jvm_classes_loaded*` | `JvmMemoryMetrics`, `JvmGcMetrics`, `JvmThreadMetrics`, `ClassLoaderMetrics` | E1 (asserts `jvm_`), plus `docs/infra/ci-cd-deployment.md:134` still cites the deleted `jvm-spring-boot.json` dashboard | The Pi memory-baseline comparison (Phase 0 step 10) reads these |
| F-OBS-P03 | `application_started_time_seconds`, `application_ready_time_seconds` | Spring Boot startup gauges | E1 (asserts `application_`) | restart-detection signal |
| F-OBS-P04 | `system_cpu_usage`, `system_load_average_1m` | `ProcessorMetrics` | none | |
| F-OBS-P05 | `process_cpu_usage`, `process_uptime_seconds`, `process_files_*` | `ProcessMetrics` | none | |
| F-OBS-P06 | `logback_events_total` | `LogbackMetrics` appender via `logback-spring.xml` | none | |
| F-OBS-P07 | `executor_*` (Tomcat and async pools) | `ExecutorServiceMetrics` | none | |
| F-OBS-P08 | `executor_completed_*`, `executor_active_*` | `TomcatMetrics` | none | |
| F-OBS-P09 | `hikaricp_connections_active`, `_idle`, `_pending`, `_connections`, `_acquire`, `_creation`, `_usage` | Spring Boot `HikariDataSource` binder | G15 (`active`/`idle`/`pending`) | |
| F-OBS-P10 | `resilience4j_*` (if any) | `resilience4j-micrometer` is on the classpath in api/broker/core/data/llm (`*build.gradle.kts:49`) but every `CircuitBreaker`, `Bulkhead`, `TimeLimiter` and `RateLimiter` in this app is created programmatically in [Resilience4jConfig.java](../../../../backend/api/src/main/java/com/swingtrade/api/config/Resilience4jConfig.java) and [LayaConfig.java](../../../../backend/llm/src/main/java/com/swingtrade/llm/config/LayaConfig.java) with no `MeterRegistry` binding, so **no** resilience4j meter is expected | none | Confirm by scraping for the `resilience4j_` prefix |

## 6. Findings that change or shape the Go port

1. **Grafana panel 9 is dead today.** It queries `llm_call_duration_bucket`, `sentiment_analysis_duration_bucket`,
   `job_run_duration_bucket` and `data_ingestion_duration_bucket`. Micrometer exports timers as
   `<name>_seconds_count`/`<name>_seconds_sum` and only emits `_bucket` when a histogram is enabled — which
   `application.properties:274` turns on for `http.server.requests` alone. So these four targets are wrong twice
   over (missing `_seconds` and no buckets configured). Any Go port that "fixes" them by publishing buckets adds a
   cardinality/CPU cost the Java side never had; the parity decision (keep the panel dead, or keep the series names
   and fix the panel) is a captain call, so it is recorded rather than absorbed.
2. **The Pi thermal gauges are registered but never read.** `PiThermalMetrics`' own javadoc says the point is to
   watch LLM heat load in the Grafana that already scrapes the app, yet no panel, alert or test queries
   `swingtrade_pi_cpu_temperature_celsius` or `swingtrade_pi_cpu_frequency_hertz`. On the Pi memory/footprint
   programme (the metric this whole migration is judged on) thermal throttling is currently unobservable. The Go
   port should reproduce the gauges *and* add the missing consumer, and that addition must be raised with the captain
   rather than absorbed into this parity port.
3. **Four meters (seven exported series) can never be non-zero**: `trades_duration_seconds_*`, `trades_pnl_seconds_*`
   (`TradeMetrics`), `signals_filtered_total` (`SignalMetrics`) and `data_ingestion_duration_seconds_*`
   (`DataIngestionMetrics`) have zero production call sites — only tests. They are dead weight; reproducing them is
   optional for parity but the ledger keeps them as explicit rows so nobody "discovers" them mid-port and assumes a
   missing writer.
4. **`trades_outcome_total` has a closed reason set.** `recordTradeClose` increments `trades_closed_total` for
   every close but only bumps the pre-registered `trades.outcome` counter when the reason matches one of the five
   known values, so any other reason value is invisible in the outcome breakdown. A Go port using one
   `prometheus.CounterVec` with a dynamic `reason` label would *add* series the Java app never had.
5. **`job_runs_failed_total` conflates real failures with reaped orphans.** `recordRunReaped` increments the same
   counter as `recordRunFailed`, and `job_run_duration_seconds` is not recorded for a reap at all. Dashboard panel
   10 cannot distinguish "the pipeline failed" from "the watchdog force-failed a stale run".
6. **Reaped and cancelled runs leak visibility differently.** `recordRunReaped` deliberately does not decrement
   `job_runs_active` (a reaped run was never active in this JVM), and `recordRunCancelled` decrements it without
   any counter — the fix history is in [JobOrchestratorMetrics.java:64-83](../../../../backend/core/src/main/java/com/swingtrade/core/metrics/JobOrchestratorMetrics.java#L64). Reproduce this exact asymmetry or the Go gauge drifts after a restart.
7. **`data_rate_limit_hits_total` has two public writers** (`recordRateLimitWait` and `recordRateLimitHit`, one
   delegating to the other) and no consumer; `api_rate_limit_hits_total` has no label at all, so it cannot even
   say which client was throttled. Low value either way.
8. **`api_request_duration_seconds` only covers `/api/**`** because the filter is not registered for `/actuator`, so
   the 15s Prometheus scrapes do not pollute it. `status` includes 429 (rate limit) and 401 (admin key rejection),
   which makes the timer the only visibility into API-boundary rejections.
9. **Two `PortfolioMetrics` classes and one `MetricsCalculator` exist.** Only
   `com.swingtrade.api.metrics.PortfolioMetrics` owns meters; `com.swingtrade.strategy.PortfolioMetrics` is a
   record. Ledger rows F-OBS-31…34 refer to the api one only.
10. **Nothing here needs the Pi.** Unlike the golden corpus, the schema snapshot and the baseline measurements, the
    metric contract is fully pinned by this document; §7 is a confirmation checklist against the live stack, not the
    deliverable itself. The Go port therefore has a fixed, testable target from day one: the 35 meter names, their
    label names and value sets, the `_total`/`_seconds` conventions in §8 and the two alert expressions.

## 7. Runtime confirmation (blocked on the Pi)

1. `curl -s http://<pi-node>:9090/api/v1/targets` → `swing-trade-stage` job UP (reaches `swing-trade-stage-api:8080`).
2. `curl -s http://<pi-node>:9090/api/v1/label/__name__/values` (or
   `curl -s http://localhost:8080/actuator/prometheus` from inside the api container) and diff the series list
   against §4/§5: add any missing app series, delete any listed series that never appear, and confirm the
   `resilience4j_` absence of P10.
3. `curl -s http://<pi-node>:9090/api/v1/rules` → confirm whether the `StrategySkipped`/`StageDegraded` rules are
   loaded at all (the deployed config snippet has no `rule_files`, §2).
4. Open the Grafana "Swing Trade" dashboard and note which panels are non-empty, especially panel 9.
5. Record the 60s export `step` behaviour against the native-image variant when Phase 0 step 11 runs, since
   `management.prometheus.metrics.export.step=60s` interacts with the 15s scrape interval.

## 8. Prometheus naming mapping for the Go port

| Java | Exported series | Go (`client_golang`) requirement |
|---|---|---|
| `Counter.builder("trades.open")` | `trades_open_total` | `prometheus.NewCounter(prometheus.CounterOpts{Name: "trades_open_total", ConstLabels: …})` — the `_total` suffix must be explicit, and re-check the output does not turn into `trades_open_total_total` when OpenMetrics negotiation is enabled |
| `Timer.builder("llm.call.duration")` | `llm_call_duration_seconds_count`, `llm_call_duration_seconds_sum` | `prometheus.NewHistogramVec`-style timer named `llm_call_duration_seconds` (or a hand-rolled counter+summary), no buckets unless the Java histogram behaviour is reproduced |
| `Gauge.builder("killswitch.active")` | `killswitch_active` | `prometheus.NewGauge(prometheus.GaugeOpts{Name: "killswitch_active"})` |
| `Gauge.builder("swingtrade_pi_cpu_temperature_celsius").baseUnit("celsius")` | `swingtrade_pi_cpu_temperature_celsius` (unit in `# HELP`/`# TYPE` only, base unit becomes `_base` in OpenMetrics) | name as-is |
| `management.metrics.tags.application=swing-trade-api` | every series carries `application="swing-trade-api"` | one `ConstLabels: prometheus.Labels{"application": "swing-trade-api"}` on the registry, and do not re-add it per-collector |
| `management.prometheus.metrics.export.step=60s` | unchanged (scrape-side concern in Java) | not applicable; Prometheus pulls the Go text format directly |

`log/slog` JSON and `client_golang` metric naming parity both belong to Phase 1 per
[standards.md](../standards.md); the ledger rows above are the acceptance surface for that work.
