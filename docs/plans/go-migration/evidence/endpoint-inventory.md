# Endpoint inventory (Phase 0, steps 2-3)

Status: complete for steps 2 (list every live endpoint) and 3 (diff against the dashboard).
Step 2 remainder (springdoc OpenAPI export) is pending a running stack — see §9.

Derived by: full read of all 28 controller files in `backend/api/src/main/java/com/swingtrade/api/controller/`,
cross-checked against a mechanical annotation count, plus a full read of `dashboard/src/api/*.ts` (24 modules).

## 1. Summary

| Measure | Count |
|---|---|
| Live REST endpoints | 156 |
| Disabled (commented-out) endpoints | 13 (1 Admin, 8 GpuHub, 4 Upstox) |
| Endpoints consumed by the dashboard (fetch) | 107 |
| Dashboard-reached via browser redirect (OAuth callback) | 1 (`GET /api/fyers/callback`) |
| Live but not consumed by the dashboard | 48 |
| SSE endpoints | 3 (plan assumed five — see §4) |
| `/api` class-prefix collisions | 4 controllers (`Ingestion`, `Watchlist`, `Settings`, `SentimentApi`), plus `AnalysisController` and `AnalysisOrchestrationController` = 6 controllers on `/api` |

## 2. Method (reproducible)

```bash
# mechanical count of live method-level mappings
for f in backend/api/src/main/java/com/swingtrade/api/controller/*.java; do
  grep -c "^\s*@\(Get\|Post\|Put\|Patch\|Delete\)Mapping" "$f"
done
# GpuHubController's 8 mappings sit inside a /* */ block comment and must be excluded;
# UpstoxAuthController is entirely commented out.
```
Every controller file was then read in full to resolve class prefixes, request/response types, and the
commented-out mappings. Counts: 156 live, 13 disabled (8 GpuHub + 4 Upstox + 1 Admin).

## 3. Cross-cutting request behavior

- **Base path**: everything is under `/api`. The dashboard hard-codes `API_BASE_URL = '/api'`
  (`dashboard/src/api/config.ts`); dev proxied by `vite.config.ts` to `localhost:8080`.
- **Response envelopes are not uniform** (migration decision point):
  - `ApiResponse<T>` wrapper: most controllers (Ingestion, Watchlist, Settings, Sentiment, StrategyConfig, Fyers partly).
  - Raw body / bare lists: SignalController, PositionController (`PaginatedResponse`), CandidateScan history,
    PaperPortfolio, StrategyType (ApiResponse), SignalSelection, SymbolStrategy, StrategyReport, FyersAuth (`Map`).
  - `ResponseEntity<?>` with hand-built 400/404/409 bodies: JobRun, CandidateScan, Analysis, SignalController gate-effectiveness.
- **Authentication: none by default.** No Spring Security on trading endpoints (documented accepted risk).
  Optional shared-key gate exists: `ApiSecurityInterceptor` with `app.security.api-key.enabled=false` (default off),
  `app.security.api-key`, and a public rate limit `app.security.public-rate-limit-per-minute` (default 100/min).
  Dashboard sends no auth headers. Migration scope is parity only — keep the same off-by-default behavior.
- **Rate limiting**: Resilience4j `@RateLimiter` on `sentimentAnalysis` (Sentiment analyse/evaluate/pdf-extract),
  `candidateScan` (candidate scan start, intraday paper scan), `llmAnalysis` (analysis analyze, run-full).
- **Ad-hoc token gate**: `POST /api/ingestion/reconcile` requires `apply=true` + `ingestion.reconcile.apply.enabled`
  flag + single symbol, and checks optional header `X-Reconcile-Token` (mismatch = 403).
- **Trace/correlation**: no correlation response header is emitted — `TraceIdFilter` only populates the MDC with
  `traceId`, `symbol` and `module` (see [security-and-error-handling.md §6](security-and-error-handling.md));
  the dashboard only reads such headers opportunistically for error display, with a body-field fallback. No
  correlation header is sent by the client.

## 4. SSE endpoints (3 found; plan's "five SSE event streams" is inaccurate)

| Endpoint | Events | Timeout | Client |
|---|---|---|---|
| `POST /api/signals/generate-all/stream` | `progress`, `signal`, `error` | 300s | dashboard fetch-based SSE (`apiSseEvents`), terminal `COMPLETE` |
| `GET /api/candidate-scans/{runId}/stream` | `log` (payload `ScanLogEvent`) | none (`new SseEmitter(0L)`) | dashboard native `EventSource` (only use in the codebase) |
| `POST /api/analysis/run-full` | `ping` (keepalive every 30s), `started`, `progress`, `complete`, `error` | `analysis.stream.timeout-ms`, default 7,200,000ms | dashboard fetch-based SSE |

## 5. Live endpoint inventory (156)

Convention: `Request` = body DTO (B) or notable params; `Response` = body type inside the envelope/raw.

### HealthController — `/api/health` (6)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/health | health | — | HealthStatus | bare mapping = class prefix |
| GET | /api/health/details | detailedHealth | `verbose:boolean=false` | HealthStatus | |
| GET | /api/health/database | databaseHealth | — | HealthStatus | |
| GET | /api/health/market-data | marketDataHealth | — | HealthStatus | hardcoded status, no live probe |
| GET | /api/health/llm | llmHealth | — | HealthStatus | hardcoded provider detail |
| GET | /api/health/full | fullHealth | — | HealthStatus | aggregates components |

### IngestionController — `/api` (13)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| POST | /api/ingestion/backfill | backfillStock | `symbol`, `years:int=5` | ApiResponse\<Map> | synchronous |
| POST | /api/ingestion/backfill-all | backfillAll | `years:int=5` | ApiResponse\<Map> | async, returns pullId |
| POST | /api/ingestion/range | ingestRange | `from:LocalDate`, `to:LocalDate` | ApiResponse\<Map> | |
| POST | /api/data/pull/incremental | incrementalPull | `symbol`, `from`, `to` | ApiResponse\<Map> | 400 on bad symbol |
| POST | /api/ingestion/date | ingestDate | `symbol`, `date` | ApiResponse\<Map> | |
| POST | /api/ingestion/latest | ingestLatest | `symbol`, `exchange=NSE` | ApiResponse\<Map> | |
| GET | /api/ingestion/status | getIngestionStatus | — | ApiResponse\<List\<Map>> | candle counts per watchlist symbol |
| GET | /api/ingestion/reconcile | reconcile | `from`, `to`, `symbol?` | ApiResponse\<Map> | read-only |
| POST | /api/ingestion/reconcile | applyReconcile | `from`, `to`, `symbol?`, `apply:boolean`; header `X-Reconcile-Token` | ApiResponse\<Map> | write path, flag + token gated, 403 on token mismatch |
| GET | /api/holidays/today | checkToday | — | ApiResponse\<Map> | Asia/Kolkata |
| GET | /api/holidays | listHolidays | `from?`, `to?` | ApiResponse\<Map> | |
| POST | /api/holidays | addHoliday | `date`, `occasion`, `type=FULL` | ApiResponse\<Map> | |
| DELETE | /api/holidays/{date} | removeHoliday | `date:LocalDate` (path) | ApiResponse\<Map> | 400 when none found |

### WatchlistController — `/api` (10)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/watchlist | getWatchlist | — | ApiResponse\<Object> | active only |
| GET | /api/watchlist/all | getAllWatchlist | — | ApiResponse\<Object> | includes inactive |
| POST | /api/watchlist | addToWatchlist | `symbol`, `name?=""`, `exchange?=NSE` | ApiResponse\<Object> | 400 duplicate/oversized |
| DELETE | /api/watchlist/{symbol} | removeFromWatchlist | `symbol` (path) | ApiResponse\<String> | |
| PATCH | /api/watchlist/{symbol}/toggle | toggleWatchlist | `symbol` (path), `activate:boolean=false` | ApiResponse\<Object> | 404 unknown |
| GET | /api/data/status | getIngestionStatus | — | ApiResponse\<Object> | same data as `/api/ingestion/status` (duplicate) |
| POST | /api/data/pull | triggerDataPull | `yearsBack:int=1` | ApiResponse\<Object> | 400 if pull in progress |
| GET | /api/data/pull/progress | getPullProgress | `pullId?` | ApiResponse\<Object> | `no_active_pull` when idle |
| POST | /api/data/pull/cancel | cancelPull | — | ApiResponse\<String> | |
| POST | /api/data/backfill | backfillSymbol | `symbol`, `yearsBack:int=2` | ApiResponse\<Map> | fire-and-forget; always 200 |

### JobRunController — `/api/job/runs` (5)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| POST | /api/job/runs/start | startRun | B `RunRequest` optional; `triggerType=MANUAL`, `symbols`, `variantIds`, `stages`, `skipLlm`, `dryRun` | JobRunResponse / ErrorResponse | 409 when run active, 400 bad scope |
| GET | /api/job/runs/{runId}/progress | getProgress | `runId:UUID` (path) | Map | 404 unknown |
| GET | /api/job/runs/{runId}/summary | getSummary | `runId:UUID` (path) | Map | degraded/skipped breakdown |
| GET | /api/job/runs | listRuns | — | List\<JobRunResponse> | bare mapping |
| POST | /api/job/runs/{runId}/cancel | cancelRun | `runId:UUID` (path) | Map | 200 even when unknown |

### AdminController — `/api/admin` (8 live)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| POST | /api/admin/symbols/refresh | refreshSymbolMaster | — | ApiResponse\<Map> | Fyers symbol master NSE_CM.csv |
| GET | /api/admin/symbols/status | getSymbolMasterStatus | — | ApiResponse\<Map> | row count |
| POST | /api/admin/kill-switch | killSwitch | B `KillSwitchRequest` @Valid | ApiResponse\<Map> | destructive |
| GET | /api/admin/kill-switch/status | getKillSwitchStatus | — | ApiResponse\<KillSwitchState> | |
| POST | /api/admin/kill-switch/toggle | toggleKillSwitch | `enable:boolean` | ApiResponse\<Map> | |
| GET | /api/admin/health | adminHealth | — | ApiResponse\<Map> | |
| POST | /api/admin/data/validate | validateData | B `DataQualityValidationRequest` | ApiResponse\<DataQualityReport> | 400 unordered range |
| GET | /api/admin/data/status | getDataStatus | `symbol`, `days:int` | ApiResponse\<Map> | |

### SettingsController — `/api` (21)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/settings | getSettings | — | ApiResponse\<Map\<String,String>> | selectedBroker |
| POST | /api/settings/broker | setBroker | B Map (`broker`) | ApiResponse\<Map> | 400 blank |
| GET | /api/settings/llm | getLlmSettings | — | ApiResponse\<Map> | secrets as `<key>.configured` |
| PUT | /api/settings/llm | setLlmSettings | B Map | ApiResponse\<Map> | restarts llama-server on change |
| GET | /api/settings/gpuhub | getGpuHubSettings | — | ApiResponse\<Map> | secret flag only |
| PUT | /api/settings/gpuhub | setGpuHubSettings | B Map | ApiResponse\<Map> | masks secrets |
| GET | /api/settings/discord | getDiscordSettings | — | ApiResponse\<Map> | |
| PUT | /api/settings/discord | setDiscordSettings | B Map | ApiResponse\<Map> | no masking here |
| POST | /api/settings/test/discord | testDiscord | — | ApiResponse\<Map\<String,Boolean>> | live webhook |
| POST | /api/settings/test/pi | testPiConnection | — | ApiResponse\<Map> | SSH + llama-server + inference |
| POST | /api/settings/test/openai | testOpenAiConnection | — | ApiResponse\<Map> | 128-token probe |
| POST | /api/settings/test/ollama | testOllamaConnection | — | ApiResponse\<Map> | 120s, origin-allowlisted URL |
| POST | /api/settings/test/pi-agent | testPiAgentConnection | — | ApiResponse\<Map> | 190s |
| POST | /api/settings/pi/start | startPiServer | — | ApiResponse\<Map> | |
| POST | /api/settings/pi/stop | stopPiServer | — | ApiResponse\<Map> | |
| GET | /api/settings/pi/status | getPiServerStatus | — | ApiResponse\<Map> | lifecycle diagnostics |
| GET | /api/settings/trading | getTradingSettings | — | ApiResponse\<Map> | |
| PUT | /api/settings/trading | setTradingSettings | B Map | ApiResponse\<Map> | 400 initial_capital ≤ 0 |
| GET | /api/settings/scanning | getScanningSettings | — | ApiResponse\<Map> | |
| PUT | /api/settings/scanning | setScanningSettings | B Map | ApiResponse\<Map> | workers 1-12 |
| POST | /api/settings/save | saveAllSettings | B Map (nested broker/llm/gpuhub/discord/trading/scanning) | ApiResponse\<Map> | unified save |

### FyersAuthController — `/api/fyers` (5)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/fyers/login | getLoginUrl | — | Map\<String,String> | 400 when unconfigured |
| GET | /api/fyers/callback | handleCallback | `auth_code` param; HttpServletResponse | void | raw HTML page postMessages opener; browser redirect, not fetched |
| POST | /api/fyers/auth | authenticate | B Map (`authCode`) | Map\<String,String> | token exchange |
| GET | /api/fyers/status | getStatus | — | Map\<String,Object> | `connected` + `clientId` |
| POST | /api/fyers/logout | logout | — | Map\<String,Object> | clears tokens |

### SentimentApiController — `/api` (17)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/sentiment/{symbol}/latest | getLatestSentiment | `symbol` (path) | ApiResponse\<SentimentResult> | fabricates NEUTRAL when none |
| GET | /api/sentiment/{symbol}/history | getSentimentHistory | `symbol` (path), `page=0`, `size=20` | ApiResponse\<List> | in-memory pagination |
| GET | /api/sentiment/accuracy | getAccuracyStats | — | ApiResponse\<Map> | |
| GET | /api/sentiment/accuracy/summary | getAccuracySummary | — | ApiResponse\<Map> | |
| GET | /api/sentiment/accuracy/by-window | getAccuracyByWindow | — | ApiResponse\<List> | |
| GET | /api/sentiment/accuracy/by-regime | getAccuracyByRegime | — | ApiResponse\<List> | |
| GET | /api/sentiment/accuracy/by-symbol | getAccuracyBySymbol | — | ApiResponse\<List> | |
| GET | /api/sentiment/accuracy/calibration | getCalibration | — | ApiResponse\<List> | |
| GET | /api/sentiment/accuracy/rolling-ic | getRollingIC | `windowDays:int=30` | ApiResponse\<List> | |
| GET | /api/sentiment/accuracy/signal-volume | getSignalVolume | — | ApiResponse\<SignalVolumeStats> | |
| GET | /api/sentiment/accuracy/ece | getECE | — | ApiResponse\<ECEStats> | |
| POST | /api/sentiment/evaluate/trigger | triggerEvaluation | — | ApiResponse\<Map> | @RateLimiter sentimentAnalysis |
| GET | /api/sentiment/evaluate/status | getJobStatus | — | ApiResponse\<Map> | "never run" default |
| GET | /api/news/{symbol}/latest | getLatestNews | `symbol` (path) | ApiResponse\<List\<NewsArticle>> | |
| GET | /api/pdf/{symbol}/latest | getLatestEarnings | `symbol` (path) | ApiResponse\<EarningsData> | ok(null) when none |
| POST | /api/sentiment/{symbol}/analyse | triggerAnalysis | `symbol` (path) | ApiResponse\<SentimentResult> | live LLM call; rate limited |
| POST | /api/pdf/{symbol}/extract | triggerPdfExtraction | `symbol` (path), `url` (query) | ApiResponse\<EarningsData> | caller-supplied URL (SSRF surface) |

### SignalController — `/api/signals` (17)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/signals/gate-effectiveness | getGateEffectiveness | `from`, `to`, `symbol?`, `strategy?`, `regime?`, `gate=SENTIMENT` | EffectivenessReport | 400 from>to; second overload when gate≠SENTIMENT |
| GET | /api/signals/latest | getLatestSignals | — | List\<SignalResponse> | |
| GET | /api/signals/symbol/{symbol} | getSignalsBySymbol | `symbol` (path) | List\<SignalResponse> | |
| GET | /api/signals/date-range | getSignalsByDateRange | `startDate`, `endDate` (required) | List\<SignalResponse> | |
| GET | /api/signals/type/{type} | getSignalsByType | `type:SignalType` (path, enum) | List\<SignalResponse> | 400 on bad enum |
| GET | /api/signals/high-confidence | getHighConfidenceSignals | `minConfidence:double=0.7` | List\<SignalResponse> | 400 outside 0-1 |
| POST | /api/signals/generate | generateSignal | B `SymbolRequest` @Valid | SignalResponse | 400 invalid |
| POST | /api/signals/price-action/{symbol}/generate | generatePriceActionSignal | `symbol` (path) | SignalResponse | 404 insufficient history |
| POST | /api/signals/generate-all | generateAllSignals | — | GenerateAllResponse | @Transactional; deletes latest PRICE_ACTION first; skip reasons |
| POST | /api/signals/generate-all/stream | generateAllSignalsStream | — | SseEmitter | SSE: progress, signal, error; 300s |
| DELETE | /api/signals | clearAllSignals | — | Map | @Transactional |
| DELETE | /api/signals/{symbol} | clearSignalForSymbol | `symbol` (path) | Map | reserved segments latest/date-range/scan guarded |
| POST | /api/signals/scan | triggerScan | B `ScanRequest` (optional) | ScanResponse | NO_SIGNALS when 0 |
| GET | /api/signals/scan/history | getScanHistory | — | List\<ScanResponse> | |
| GET | /api/signals/analysis/{symbol} | getTechnicalAnalysis | `symbol` (path) | TechnicalAnalysisResponse | 404 null |
| GET | /api/signals/sentiment/{symbol} | getSentimentAnalysis | `symbol` (path) | SentimentAnalysisResponse | 404 none stored |
| GET | /api/signals/combined/{symbol} | getCombinedSignal | `symbol` (path) | CombinedSignalResponse | live news+sentiment, failures swallowed |

### BacktestController — `/api/backtest` (8)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| POST | /api/backtest/portfolio | runPortfolioBacktest | B `PortfolioBacktestRequest` | PortfolioBacktestResponse | 400/404 semantics |
| GET | /api/backtest/strategies | listStrategies | — | List\<String> | registry names |
| POST | /api/backtest/run-tune | runTunedBacktest | B `BacktestConfig`; `symbol`, `exchange=NSE`, `strategyName?` | BacktestResult | |
| POST | /api/backtest/run | runBacktest | `symbol`, `exchange=NSE`, `strategyName?` | BacktestResult | defaults() config; exchange unused for filtering |
| POST | /api/backtest/run-all-tune | runTunedBacktestAll | B `BacktestConfig`; `exchange=NSE`, `strategyName?` | BacktestReportSummary | |
| POST | /api/backtest/run-all | runBacktestAll | `exchange=NSE`, `strategyName?` | BacktestReportSummary | writes JSON+CSV to reportsDir |
| GET | /api/backtest/reports | listReports | — | List\<String> | `^backtest_\d{8}_\d{6}\.json$` |
| GET | /api/backtest/reports/{filename} | getReport | `filename` (path) | JsonNode | traversal guard, 400/404/500 |

### CandidateScanController — `/api/candidate-scans` (11)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/candidate-scans/settings | getSettings | — | Map\<String,String> | |
| PUT | /api/candidate-scans/settings | updateSettings | B Map\<String,String> | Map\<String,String> | 400 on bad value |
| POST | /api/candidate-scans | start | — | CandidateScanRunEntity | 409 when running |
| GET | /api/candidate-scans | history | — | List\<CandidateScanRunEntity> | raw list |
| GET | /api/candidate-scans/{runId} | run | `runId:UUID` (path) | CandidateScanRunEntity | 404 |
| GET | /api/candidate-scans/{runId}/results | results | `runId` (path), `offset=0`, `limit=100`, `symbol=""`, `signalType=""` | ResultPage | 404 missing |
| GET | /api/candidate-scans/{runId}/stream | stream | `runId` (path) | SseEmitter | SSE `log` events, infinite timeout |
| POST | /api/candidate-scans/{runId}/analysis | startAnalysis | B `AnalysisScopeRequest`; `runId` (path) | CandidateScanRunEntity | 400/409 |
| POST | /api/candidate-scans/{runId}/cancel | cancel | `runId` (path) | CandidateScanRunEntity | 404 |
| POST | /api/candidate-scans/{runId}/pause | pause | `runId` (path) | CandidateScanRunEntity | 400 not running |
| POST | /api/candidate-scans/{runId}/resume | resume | `runId` (path) | CandidateScanRunEntity | 400 not paused |

### PositionController — `/api/positions` (12)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/positions | getPositions | `page=0`, `size=20` | PaginatedResponse\<PositionResponse> | cosmetic pagination (full fetch) |
| GET | /api/positions/{symbol} | getPosition | `symbol` (path) | PositionResponse | 404 |
| GET | /api/positions/closed | getClosedPositions | `page=0`, `size=20` | PaginatedResponse\<PositionResponse> | cosmetic pagination |
| GET | /api/positions/status/{status} | getPositionsByStatus | `status` (path, free-form) | List\<PositionResponse> | no enum validation |
| GET | /api/positions/symbol/{symbol} | getPositionsBySymbol | `symbol` (path) | PaginatedResponse | cosmetic pagination |
| GET | /api/positions/sector/{sector} | getPositionsBySector | `sector` (path) | List\<PositionResponse> | |
| GET | /api/positions/stats | getPositionStats | — | PositionStats | raw, no wrapper |
| GET | /api/positions/sector-allocation | getSectorAllocation | — | SectorAllocation | raw |
| POST | /api/positions/{symbol}/close | closePosition | `symbol` (path); B `ClosePositionRequest` optional | PositionResponse | 404; live write |
| POST | /api/positions | createPosition | B `TradeRequest` @Valid | PositionResponse | 200 on create (not 201) |
| GET | /api/positions/performance | getPortfolioPerformance | — | PerformanceResponse | |
| GET | /api/positions/risk-summary | getRiskSummary | — | RiskSummary | raw |

### AnalysisController — `/api` (1)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| POST | /api/analysis/analyze | analyze | `symbol` (query, required) | CompositeAnalysis / Map error | @RateLimiter llmAnalysis; 400/500 errors |

### AnalysisOrchestrationController — `/api` (1)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| POST | /api/analysis/run-full | runFullAnalysis | `symbol`, `backfillYears:int=5` | SseEmitter | 9-stage pipeline; SSE ping/started/progress/complete/error; @RateLimiter llmAnalysis |

### StrategyConfigController — `/api/strategy-configs` (8)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/strategy-configs | list | `variantId?`, `mode?` (enum) | ApiResponse\<List> | |
| POST | /api/strategy-configs | create | B `StrategyConfigRequest` @Valid | ApiResponse\<StrategyConfigResponse> | 201 |
| GET | /api/strategy-configs/{variantId} | current | `variantId` (path) | ApiResponse\<StrategyConfigResponse> | active version |
| GET | /api/strategy-configs/{variantId}/versions | versions | `variantId` (path) | ApiResponse\<List> | |
| GET | /api/strategy-configs/{variantId}/versions/{version} | version | `variantId`, `version:int` (path) | ApiResponse\<StrategyConfigResponse> | |
| PUT | /api/strategy-configs/{variantId} | update | `variantId` (path); B `StrategyConfigRequest` @Valid | ApiResponse\<StrategyConfigResponse> | IllegalArgumentException (not 400) on id mismatch; create-version semantics |
| PUT | /api/strategy-configs/{variantId}/mode | mode | `variantId` (path); B `StrategyModeRequest` @Valid | ApiResponse\<StrategyConfigResponse> | champion/challenger |
| DELETE | /api/strategy-configs/{variantId} | delete | `variantId` (path) | ApiResponse\<StrategyConfigResponse> | returns deleted rep, not 204 |

### StrategyReportController — `/api/strategy-report` (2)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/strategy-report/arbitration-comparison | arbitrationComparison | `from?`, `to?` (ISO date, default 90d) | Comparison | raw, no ApiResponse |
| GET | /api/strategy-report | report | `from?`, `to?` (ISO date, default 30d) | Report | raw, no ApiResponse |

### StrategyTypeController — `/api/strategy-types` (1)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/strategy-types | list | — | ApiResponse\<List\<StrategyTypeResponse>> | warmupBars nullable |

### SymbolStrategyController — `/api/symbols` (1)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/symbols/{symbol}/strategy-matrix | matrix | `symbol` (path) | StrategyMatrixResponse | raw; synthetic TOURNAMENT/SELECTED row; history ≤ 20 |

### SignalSelectionController — `/api/signal-selections` (1)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/signal-selections | list | `from?`, `to?` (ISO date, default 30d) | List\<SelectionResponse> | inclusive; candidates untyped |

### PromotionEligibilityController — `/api/strategy-configs` (1)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/strategy-configs/{variantId}/promotion-eligibility | check | `variantId` (path) | ApiResponse\<PromotionEligibilityResponse> | dataLimitations field documents degraded inputs |

### PaperPortfolioController — `/api/paper-portfolios` (2)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/paper-portfolios | list | — | List\<PaperPortfolioSummary> | raw list |
| GET | /api/paper-portfolios/{portfolioId}/positions | positions | `portfolioId` (path, free-form) | List\<ShadowPositionView> | raw list |

### IntradayPaperController — `/api/intraday/paper` (2)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| POST | /api/intraday/paper/sessions | runSession | B `IntradayPaperSessionRequest` @Valid | IntradayPaperSessionResult | paper-only; INTRADAY_* types only; replay single date; persists on COMPLETED |
| GET | /api/intraday/paper/sessions/{runId} | getSession | `runId` (path) | IntradayPaperSessionResult | 404; JSON from resultJson |

### IntradayPaperScanController — `/api/intraday/paper/scans` (1)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| POST | /api/intraday/paper/scans | scan | B `IntradayPaperScanRequest` @Valid | IntradayPaperScanResult | read-only replay; 422 on IllegalStateException (unusual status) |

### SynthesisEvaluationController — `/api/synthesis/evaluations` (1)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/synthesis/evaluations/summary | summary | — | ApiResponse\<SynthesisEvaluationSummary> | |

### DataExportController — `/api/data` (1)

| Method | Path | Handler | Request | Response | Notes |
|---|---|---|---|---|---|
| GET | /api/data/export | export | `symbol` (repeatable + comma-split), `fromDate`, `toDate`, `format=CSV` | StreamingResponseBody | streaming CSV/JSON; 204 empty; 1000-row paging |

## 6. Disabled endpoints (13, do not port as live)

| Controller | Method | Path | Handler | Reason |
|---|---|---|---|---|
| GpuHubController (whole controller inside `/* */`) | POST | /api/gpuhub/deployments | createDeployment | No auth guard; caller-controlled `cmd` = unauthenticated RCE on remote GPU container |
| GpuHubController | GET | /api/gpuhub/deployments | listDeployments | same block; page 1 size 20 hardcoded |
| GpuHubController | GET | /api/gpuhub/deployments/{uuid} | getDeploymentStatus | same block |
| GpuHubController | POST | /api/gpuhub/deployments/{uuid}/stop | stopDeployment | destructive on billed infra |
| GpuHubController | DELETE | /api/gpuhub/deployments/{uuid} | deleteDeployment | destructive on billed infra |
| GpuHubController | POST | /api/gpuhub/deployments/{uuid}/containers | listContainers | POST verb on a read |
| GpuHubController | POST | /api/gpuhub/containers/{containerUuid}/stop | stopContainer | same block |
| GpuHubController | POST | /api/gpuhub/images | listPrivateImages | POST for a listing |
| UpstoxAuthController (whole controller commented) | GET | /api/auth/upstox/login-url | getLoginUrl | Upstox integration unconfigured; kept for restore |
| UpstoxAuthController | GET | /api/auth/upstox/callback | handleCallback | same |
| UpstoxAuthController | GET | /api/auth/upstox/validate | validateToken | same |
| UpstoxAuthController | POST | /api/auth/upstox/token | setToken | same |
| AdminController | POST | /api/admin/data/pull-historical | pullHistoricalData | commented out; needs Upstox re-enablement |

## 7. Dashboard consumption diff

The dashboard consumes 108 of the 156 live endpoints: 107 via `fetch` (through `dashboard/src/api/shared.ts`)
plus `GET /api/fyers/callback` via browser redirect from the Fyers OAuth popup.

### 7.1 Live but unconsumed by the dashboard (48)

| Controller | Endpoints (count) |
|---|---|
| HealthController | /api/health/details, /database, /market-data, /llm (4) |
| IngestionController | /ingestion/backfill-all, /data/pull/incremental, /ingestion/latest, /ingestion/status, GET+POST /ingestion/reconcile, POST /holidays, DELETE /holidays/{date} (8) |
| WatchlistController | /api/watchlist/all, /api/data/backfill (2) |
| AdminController | all 8 live (admin surface, no UI client) (8) |
| SignalController | /signals/date-range, /signals/generate, /signals/scan/history, /signals/analysis/{symbol}, /signals/sentiment/{symbol}, /signals/combined/{symbol} (6) |
| BacktestController | /backtest/portfolio, /backtest/run-tune, /backtest/run-all-tune, /backtest/strategies (4) |
| PositionController | /positions/{symbol}, /positions/status/{status}, /positions/symbol/{symbol}, /positions/sector/{sector}, /positions/sector-allocation (5) |
| StrategyConfigController | POST /strategy-configs, GET /{variantId}, GET /{variantId}/versions, GET /{variantId}/versions/{version}, DELETE /{variantId} (5) |
| SentimentApiController | POST /sentiment/evaluate/trigger, GET /sentiment/evaluate/status, GET /pdf/{symbol}/latest, POST /pdf/{symbol}/extract (4) |
| SynthesisEvaluationController | GET /synthesis/evaluations/summary (1) |
| DataExportController | GET /api/data/export (1) |

Total: 4+8+2+8+6+4+5+5+4+1+1 = 48.

Playwright e2e specs exercise a few of these directly (`/api/sentiment/evaluate/trigger`,
`/api/data/pull/incremental`, `/api/ingestion/reconcile`), so the runtime diff is not identical to the test diff.

### 7.2 Dashboard findings that affect the Go port

- `dashboard/src/api/config.ts` hard-codes `API_BASE_URL = '/api'`; `VITE_API_BASE_URL` / `VITE_API_TIMEOUT`
  in `.env.example` are dead config (never read). Same-origin assumption must hold in the Go deployment too.
- `vite.config.ts` SSE de-buffering (strips Content-Length, forces keep-alive) is applied to any URL containing
  `/stream` or any `text/event-stream` response — the Go SSE responses must be flush-friendly.
- The dashboard reads `x-correlation-id` / `x-request-id` / `request-id` response headers for error display and
  sends none. **Correction (recorded in [security-and-error-handling.md §6](security-and-error-handling.md)): the Java
  backend never emits those headers** — `traceId` exists only as an MDC log field — so the Go port does not need one
  for parity; adding one is optional and safe because the dashboard falls back to a body field.
- `/positions` pagination is cosmetic in Java (`total = positions.size()`); Go can paginate properly as long as
  the wire shape `PaginatedResponse<PositionResponse>` is unchanged.
- Envelope handling differs per call site (`responseContract: 'direct' | 'envelope'`), so the Go port must keep
  both shapes exactly; normalization is a breaking change requiring a dashboard change.
- The unused `/fyers` dev-proxy entry and the duplicate `/api/data/status` vs `/api/ingestion/status` routes
  are consolidation candidates, tracked as findings rather than fixed silently (gate protocol).

## 8. Migration-relevant findings (brought back for the gate)

1. **Plan assumption corrected**: three SSE endpoints exist, not five. Golden corpus for step 4 covers 3 streams.
2. **13 endpoints are disabled in Java** (GpuHub RCE class): the Go port must keep them off behind a build/feature
   flag, matching Java; the GpuHub auth-guard ticket is a precondition for ever enabling them.
3. **Reserved-segment guard**: `DELETE /api/signals/{symbol}` must not swallow `latest`, `date-range`, `scan`;
   Java uses an explicit reserved list. Router port must replicate.
4. **Error-status inconsistencies to preserve**: 422 (intraday scan), 409 (job run / candidate scan / candidate analysis),
   403 (reconcile token), 200-on-DELETE-with-body, 201 only on strategy-config create, `IllegalArgumentException`
   surfacing as 500 on strategy-config update mismatch (bug preserved by goldens).
5. **Rate-limit and API-key gates are opt-in** and off by default locally, but the **stage profile turns the API key
   on** and the stage compose file refuses to start without it; Go parity therefore means "same off-by-default
   developer ergonomics, same enforced-on stage admin boundary" — fully characterized in
   [security-and-error-handling.md](security-and-error-handling.md).

## 9. Open items feeding the next Phase 0 steps

- Step 2 remainder: export OpenAPI from springdoc (`springdoc-openapi-starter-webmvc-ui:3.1.1` is already a
  dependency) once the isolated local stack is up; diff against this table.
- Step 4 (golden corpus) now has a concrete target list: 156 live endpoints + 3 SSE streams + 13 disabled.
- Step 12 (ledger expansion) can be driven directly from §5 tables: one row per endpoint.
