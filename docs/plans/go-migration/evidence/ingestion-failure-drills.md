# Phase 3 failure drills and row-comparison status

Status: the deterministic half of Phase 3 step 7 (failure drills) is covered
by committed tests listed below; the live drills and the step 5 row-for-row
Java-versus-Go session-window comparison remain outstanding and are not
claimed here. No Java code was changed and no scheduler runs against anything
but tests: every Go job gate defaults false until G8.

## Drill coverage (deterministic)

| Drill (step 7) | Go coverage | Where |
|---|---|---|
| Provider timeout / transient server error | Yahoo retries transient 5xx once with backoff and treats 404 as empty; intraday and EOD runs isolate a failing symbol and continue with the rest; the symbol master reports a failed refresh without aborting its schedule loop | `TestYahooClientRetriesTransientServerErrors`, `TestYahooClientHonorsContextCancellation`, `TestIngestWatchlistIntervalIsolatesPerSymbolFailures`, `TestSymbolFailureDoesNotStopRemainingSymbols`, `TestRefreshFailuresAreReportedNotFatal` |
| Rate limiting | The shared request pacer enforces the provider quota, waits are context-cancellable, and a non-positive quota is unlimited; the Fyers client refreshes an expired token once and does not retry network errors | `TestRequestPacerAllowsFirstRequestAndPacesLaterRequests`, `TestRequestPacerCancellationInterruptsWait`, `TestRequestPacerNonpositiveQuotaIsUnlimited`, `TestFyersHistoryClientBuildsRequestAndUsesRefreshAfterUnauthorized`, `TestFyersHistoryRequestDoesNotRetryNetworkErrors` |
| Malformed payload | Malformed Yahoo JSON and invalid bars are rejected before storage; malformed Fyers history rows are skipped; a truncated symbol-master CSV keeps the rows parsed so far; a failed or empty master download never wipes stored symbols; ingestion classifies invalid rows separately from saved rows | `TestParseYahooChartReportsMalformedJSON`, `TestParseYahooChartFallsBackToCloseAndRejectsInvalidBars`, `TestParseFyersHistoryRejectsMalformedPayloadAndSkipsIncompleteRows`, `TestParseFyersSymbolMasterCSVDropsNonEquityAndEmptyIdentity`, `TestParseFyersSymbolMasterCSVStopsOnMalformedQuoting`, `TestRefreshNeverWipesOnDownloadProblems`, `TestCandleIngesterFiltersNonSessionsAndPersistsValidRows` |
| Partial day | Bars outside `[09:15, 15:30)` or on non-session dates are rejected; the reconciliation preview classifies a source that cannot fill every expected session as `SOURCE_INCOMPLETE`; the EOD and intraday windows clamp to IST today | `TestIngestWatchlistIntervalDropsBarsOutsideSession`, `TestIsNormalSessionBarTimeMatchesJavaWindow`, `TestReconciliationPreviewClassifiesInvalidAndSourceIncompleteData`, `TestIngestWatchlistIntervalClampsEndToTodayIST` |
| No duplicate job runs | All three Go jobs take a per-job PostgreSQL advisory lock and skip when another worker holds it; the lock serializes concurrent workers and always releases; the real EOD integration run exercises the lock against the authorized database | `TestWithAdvisoryLockSerializesWorkersAndReleases`, `TestSkipsWorkWhenAnotherWorkerHoldsJobLock`, `TestSymbolMasterLockBoundaries`, `TestIntradaySchedulerLockBoundaries`, `TestEODSchedulerStoresCandlesThroughRealQueriesAndJobLock` |
| Restart-safe re-ingest | Candle inserts are `ON CONFLICT ... DO NOTHING` on the widened identity; intraday resumes at the latest stored bar and a repeat run stores nothing; daily ingestion short-circuits on an existing session date (integration-tested with the real query) | `TestIngestWatchlistIntervalIsIdempotentAcrossRuns`, `TestIntradayIngestionStoresAndResumesThroughRealSQL`, `TestEODSchedulerStoresCandlesThroughRealQueriesAndJobLock`, `TestCandleWriterUsesIdempotentV75IdentityInIsolatedSchema` |
| Calendar correctness feeding every drill | Holiday-first session decisions, inclusive expected-session ranges, and the weekend-as-closed Monday EOD quirk are pinned to Java behavior | `TestIsNSETradingSessionMatchesHolidayAndWeekendRules`, `TestExpectedNSETradingSessionsUsesInclusiveRangeAndHolidayOverrides`, `TestSkipsMondayWhenYesterdayWasSundayLikeJava` |
| Deterministic dataset comparison primitive | The order-insensitive candle-set comparator that the row comparison will use is committed and tested (identity includes timeframe and bar time; field and missing-row differences are reported deterministically) | `TestCompareCandleSetsIgnoresRowOrder`, `TestCompareCandleSetsReportsMissingDuplicateAndFieldDifferencesDeterministically`, `TestCompareCandleSetsIdentityIncludesTimeframeAndBarTime` |

## Outstanding for gate G3

1. **Row-for-row Java-versus-Go session-window comparison (step 5).** Run
   Go ingestion into a scratch database over an agreed session window and
   compare rows exactly (OHLCV, adjustments, rejects, audit rows) against the
   Java run over the same window, using the committed comparator. This needs a
   controlled run environment with both stacks ingesting the same fixtures; it
   has not been performed and no equality claim is made.
2. **Live failure drills.** Exercise the same failure modes against running
   processes (real provider timeout, real rate-limit response, truncated
   payload, mid-session restart) and capture the logs. The deterministic tests
   above are not a substitute.
3. The scheduling-off runtime observation recorded in
   [scheduled-jobs.md](scheduled-jobs.md) §5 remains the open half of that
   Phase 0 item and shares the same throwaway-stack requirement.

Until those runs happen, Phase 3 status is parity-pending everywhere the
[feature ledger](../feature-ledger.md) says so.
