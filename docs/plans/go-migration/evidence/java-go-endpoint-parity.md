# Java/Go side-by-side endpoint parity (2026-11-14)

The Java API and the Go API ran at the same time against one disposable PostgreSQL 16 cluster, and the same requests
went to both. This replaces the "Java backend not run" gap in [go-scratch-parity-report.md](go-scratch-parity-report.md):
JDK 21 is available here as `/opt/homebrew/opt/openjdk@21`.

## Environment

- Branch `fm/swing-trade-land-go-port-main-v1` (Go port branch landed on `main`).
- PostgreSQL 16 on `127.0.0.1:55433`, database and role `go_port_test`. Java's Flyway applied V1-V77 at startup;
  [java-go-parity/seed.sql](java-go-parity/seed.sql) then added signals, October 2026 candles and watchlist sync times.
- Java: `backend/api/build/libs/api.jar` on port 8080 with the [golden-corpus.md](golden-corpus.md) flags
  (`--spring.profiles.active=local --app.scheduling.enabled=false --llm.enabled=false --llm.backend=local
  --llm.base-url=http://127.0.0.1:1/v1`) and `-Duser.timezone=Asia/Kolkata`.
- Go: `go/api/cmd/swingtrade-api` on port 8090 with `SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE=1`, `DATABASE_URL` and
  `SWINGTRADE_LOCAL_TEST_DATABASE_URL` pointing at the same cluster, `TZ=Asia/Kolkata`, and
  `LLM_BASE_URL=http://127.0.0.1:1/v1` (the Go spelling of Java's `--llm.base-url`).
- Comparator: [scripts/java-go-parity.py](scripts/java-go-parity.py) with the case files in
  [java-go-parity/](java-go-parity). It compares status codes and JSON bodies, ignoring per-process values
  (`timestamp`, uptime, trace ids, and the values of health `systemInfo`, which reports the serving runtime under
  Java's keys by design; its key set is still compared). Lists that Java leaves unordered are compared as multisets:
  `/api/watchlist/all` (`findAll()` without `ORDER BY`) and reconcile `storedDates` (a `HashSet`).

## Results

Final run order: writes, then reconcile, then reads.

| Case file | Match | Differ |
| --- | --- | --- |
| [cases-write.json](java-go-parity/cases-write.json) | 15 | 0 |
| [cases-reconcile.json](java-go-parity/cases-reconcile.json) | 10 | 1 |
| [cases-read.json](java-go-parity/cases-read.json) | 17 | 4 |

Matching rows (now `closed` in the [feature ledger](../feature-ledger.md)): F-API-HEALTH-01, -03, -06;
F-API-INGEST-07, -10, -11; F-API-WATCH-01, -02; F-API-SETTINGS-01, -02, -03, -05, -07, -08, -17, -18, -19, -20;
F-API-SIGNALS-02; F-API-STRATTYPE-01. The write cases cover valid writes, every Java validation branch
(blank or missing broker, zero and non-numeric initial capital, scan workers out of range or missing) and the read
after each write.

Remaining differences, all still `parity-pending`:

| Row | Difference |
| --- | --- |
| F-API-HEALTH-02, -04, -05 | Go has no LLM, strategy or market-data services wired, so those components say `(not configured)` and lack Java's `serviceName`, `status` and `provider` details. |
| F-API-ADMIN-02 (and -01, -07) | `controller.RegisterAdminRoutes` exists but `server.go` never registers it: Go answers 404. |
| F-API-INGEST-08 | Go's reconciler always reads Yahoo; Java uses the selected broker's client. With `fyers` selected, Java returned `SOURCE_INCOMPLETE` for INFY 2026-10-01..05 while Go returned `COMPLETED`. Also `?symbol=` (blank): Java reconciles the literal blank symbol, Go treats it as absent and aggregates the watchlist. Every other reconcile case, including all validation branches, matches. |

## Bugs fixed by this run

Each was a live Java/Go difference before the fix and matches after it; each fix has a Go regression test.

1. **LocalDateTime encoding.** Go printed six fixed fraction digits (`2026-11-13T18:00:00.000000`); Java's
   `ISO_LOCAL_DATE_TIME` trims trailing zeros and drops a zero fraction (`2026-11-13T18:00:00`,
   `2026-11-13T18:00:00.5`). Affected every envelope `timestamp` and watchlist, ingestion-status and
   strategy-config times. Test: `TestJavaLocalDateTimeTrimsTrailingFractionZeros`.
2. **Reconcile validation envelope.** Go returned `{"status","error","message","timestamp","path"}` with one generic
   message; Java's `GlobalExceptionHandler` returns `{"code","fieldErrors","message","path","status","timestamp"}`
   with `Missing required parameter: from`, `Invalid value for 'to': expected LocalDate`, and the path without its
   query string. Test: `TestReconcileEndpointValidatesParameters`.
3. **Reconcile against Yahoo matched no dates.** Yahoo bars are dated at IST midnight; Go keyed them by their UTC
   date (the previous day), so every expected session looked missing from the source and every report was
   `SOURCE_INCOMPLETE`. Test: `TestReconciliationPreviewMatchesYahooBarsOnExchangeDate`.
4. **Reconcile aggregate status.** Java writes `IngestionController.OperationStatus` through its lower-case
   `toString()` and maps report statuses it does not know to `unknown`; Go echoed `COMPLETED` or the raw report
   status. Test: `TestJavaOperationStatusLowercasesKnownNames`.

## Indicator vectors

[scripts/IndicatorVectors.java](scripts/IndicatorVectors.java) runs the real Java `TechnicalIndicators` class over a
fixed 120-bar OHLCV series (windows of 30, 60 and 120 bars, plus a two-bar Stochastic for the partial D window) and
writes [go/strategy/testdata/java_indicator_vectors.json](../../../../go/strategy/testdata/java_indicator_vectors.json).
`TestIndicatorsMatchJavaVectorsBitForBit` replays the same inputs through the Go port and requires identical doubles.

Before the fix all ten indicators differed, mostly in the last digits (Java SMA `1118.746`, Go `1118.7459999999999`;
MACD differed from the eleventh significant digit), because TA4J 0.16 computes in `DecimalNum` (precision 32, HALF_UP)
while Go used float64. Go also fused multiply-adds in VWAP and Bollinger, and divided a partial Stochastic D window by
three where TA4J averages the values available. The indicators now run on the existing `decimalNum` port and all 43
vectors match. This closes F-STRAT-IND-01 to -10.

## Shared-database finding for Phase 8

With both services on one database, a settings write through Go is persisted correctly (verified in `app_settings`),
but Java keeps serving the old value for any key it has already read: `AppSettingsService` caches values in memory and
never invalidates them on external writes. Go reads the database on every request, so Java's writes are visible to
Go immediately. Any dual-run or shadow window that lets Go accept settings writes therefore needs Java restarted (or
settings writes held on one side) to avoid Java acting on stale settings.

## Playwright against Go

The unchanged suite ran through the Vite dashboard on `127.0.0.1:3003`, proxied to Go on `127.0.0.1:8081`, with
[scripts/run-playwright-go.sh](scripts/run-playwright-go.sh) plus `JAVA_SERVICE_URL=http://127.0.0.1:8080` so
`go-read-parity.spec.ts` also runs.

| Run | Passed | Failed | Skipped |
| --- | --- | --- | --- |
| Default public rate window (100 requests per minute) | 117 | 20 | 6 |
| `API_PUBLIC_RATE_LIMIT_PER_MINUTE=100000` on Go | 123 | 14 | 5 |

**Correction to [playwright-go-report.md](playwright-go-report.md).** Its eight "Go parity bug" rows in
`stage-sanity.spec.ts` (health, Prometheus, positions, signals, watchlist) were HTTP 429 responses: the page tests
before them use up the fixed 100-request window that Java enforces the same way, so the API assertions never reached
a handler. With the window lifted, the health, signals and watchlist checks pass.

`go-read-parity.spec.ts` compared `timestamp` and the runtime values of health `systemInfo`, so it could not pass
against any two live processes. It now compares only the `systemInfo` key set and skips `timestamp`, and it passes:
`/api/health`, `/api/holidays`, `/api/holidays/today`, `/api/watchlist` and `/api/watchlist/all` match Java.

The 14 remaining failures are routes Go does not serve yet. Java answers 200 for each and Go answers 404:

| Spec | Missing Go surface |
| --- | --- |
| `api-openapi-audit.spec.ts` | `/api-docs` (springdoc OpenAPI) |
| `stage-sanity.spec.ts`: positions stats, paginated positions, performance, sidebar navigation | `/api/positions*` (Phase 7) |
| `stage-sanity.spec.ts`: Prometheus | `jvm_*` and `application_*` series (F-OBS) |
| `multi-strategy.live.spec.ts` (7 tests), `orchestrator-debug.spec.ts` | strategy report, job runs and orchestrator, symbol strategy matrix, backtest reports, shadow books |
