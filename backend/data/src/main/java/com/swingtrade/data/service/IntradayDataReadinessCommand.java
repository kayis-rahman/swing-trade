package com.swingtrade.data.service;

import tools.jackson.databind.ObjectMapper;
import com.swingtrade.data.client.YahooFinanceClient;

import java.io.IOException;
import java.math.BigDecimal;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Clock;
import java.time.LocalDate;
import java.time.LocalTime;
import java.util.ArrayList;
import java.util.List;

/** Standalone Yahoo-only I0 readiness sampler; never initializes the app or a broker. */
public final class IntradayDataReadinessCommand {
    private static final List<String> WATCHLIST = List.of("RELIANCE", "TCS", "INFY", "HDFCBANK", "ICICIBANK");
    private static final List<Interval> INTERVALS = List.of(Interval.DAILY, Interval.FIFTEEN_MINUTES);

    private IntradayDataReadinessCommand() {}

    public static void main(String[] args) throws Exception {
        Options options = Options.parse(args);
        List<LocalDate> sessions = IntradayDataReadinessReport.weekdaySessions(options.from(), options.to());
        if (sessions.isEmpty()) throw new IllegalArgumentException("The requested date range contains no weekdays");
        SamplingResult sample = options.fixture()
            ? new SamplingResult(fixtureLedger(sessions), null) : fetchYahoo(sessions);
        writeOutputs(options.outputDirectory(), sessions, sample, options.fixture());
        System.out.println("I0 report: " + options.outputDirectory().resolve("report.md"));
        System.out.println("I0 ledger: " + options.outputDirectory().resolve("ledger.csv"));
    }

    private static SamplingResult fetchYahoo(List<LocalDate> sessions) {
        YahooFinanceClient yahoo = new YahooFinanceClient(new ObjectMapper(), Clock.systemUTC());
        LocalDate from = sessions.getFirst();
        LocalDate to = sessions.getLast();
        List<IntradayDataReadinessReport.LedgerRow> rows = new ArrayList<>();
        long previousStartMillis = -1;
        long minimumStartGapMillis = Long.MAX_VALUE;
        for (String symbol : WATCHLIST) {
            for (Interval interval : INTERVALS) {
                long startedAtMillis = System.currentTimeMillis();
                if (previousStartMillis >= 0) minimumStartGapMillis = Math.min(minimumStartGapMillis, startedAtMillis - previousStartMillis);
                previousStartMillis = startedAtMillis;
                long started = System.nanoTime();
                String status;
                List<CandleData> candles = new ArrayList<>();
                try {
                    for (CandleData candle : yahoo.fetchCandles(symbol, from, to, interval)) candles.add(candle);
                    status = candles.isEmpty() ? "NO_BARS" : "AVAILABLE";
                } catch (RuntimeException error) {
                    status = "ERROR:" + error.getClass().getSimpleName();
                }
                long elapsedMillis = (System.nanoTime() - started) / 1_000_000;
                rows.addAll(IntradayDataReadinessReport.analyze(symbol, interval, sessions,
                    candles, elapsedMillis, status));
            }
        }
        return new SamplingResult(rows, minimumStartGapMillis == Long.MAX_VALUE ? null : minimumStartGapMillis);
    }

    private static List<IntradayDataReadinessReport.LedgerRow> fixtureLedger(List<LocalDate> sessions) {
        List<IntradayDataReadinessReport.LedgerRow> rows = new ArrayList<>();
        for (String symbol : WATCHLIST) {
            for (Interval interval : INTERVALS) {
                List<CandleData> candles = new ArrayList<>();
                for (LocalDate session : sessions) {
                    int count = interval == Interval.DAILY ? 1 : NseSessionCalendar.NORMAL_SESSION_BAR_COUNT;
                    if (symbol.equals("ICICIBANK") && interval == Interval.FIFTEEN_MINUTES) count--;
                    LocalTime time = interval == Interval.DAILY ? LocalTime.MIDNIGHT : NseSessionCalendar.NORMAL_SESSION_START;
                    for (int i = 0; i < count; i++) {
                        LocalTime barTime = interval == Interval.DAILY ? time : time.plusMinutes((long) i * NseSessionCalendar.BAR_MINUTES);
                        candles.add(CandleData.of(symbol, session, barTime, BigDecimal.TEN, BigDecimal.TEN,
                            BigDecimal.TEN, BigDecimal.TEN, 1_000L, BigDecimal.TEN, interval));
                    }
                }
                rows.addAll(IntradayDataReadinessReport.analyze(symbol, interval, sessions,
                    candles, 0, "AVAILABLE"));
            }
        }
        return rows;
    }

    private static void writeOutputs(Path directory, List<LocalDate> sessions,
                                     SamplingResult sample, boolean fixture) throws IOException {
        List<IntradayDataReadinessReport.LedgerRow> rows = sample.rows();
        Files.createDirectories(directory);
        StringBuilder csv = new StringBuilder("symbol,provider,interval,session_date,requested_bars,returned_bars,valid_session_bars,missing_bars,invalid_timestamp_bars,missing_rate,provider_status,elapsed_ms\n");
        for (IntradayDataReadinessReport.LedgerRow row : rows) {
            csv.append(row.symbol()).append(',').append(fixture ? "fixture" : "yahoo").append(',')
                .append(row.interval().getCode()).append(',').append(row.sessionDate()).append(',')
                .append(row.requestedBars()).append(',').append(row.returnedBars()).append(',')
                .append(row.validSessionBars()).append(',').append(row.missingBars()).append(',')
                .append(row.invalidTimestampBars()).append(',').append(String.format(java.util.Locale.ROOT, "%.4f", row.missingRate())).append(',')
                .append(row.providerStatus()).append(',').append(row.elapsedMillis()).append('\n');
        }
        Files.writeString(directory.resolve("ledger.csv"), csv);

        long requested = rows.stream().mapToLong(IntradayDataReadinessReport.LedgerRow::requestedBars).sum();
        long returned = rows.stream().mapToLong(IntradayDataReadinessReport.LedgerRow::returnedBars).sum();
        long valid = rows.stream().mapToLong(IntradayDataReadinessReport.LedgerRow::validSessionBars).sum();
        long missing = rows.stream().mapToLong(IntradayDataReadinessReport.LedgerRow::missingBars).sum();
        long invalid = rows.stream().mapToLong(IntradayDataReadinessReport.LedgerRow::invalidTimestampBars).sum();
        StringBuilder report = new StringBuilder("# I0 intraday data readiness\n\n");
        report.append("- Source: ").append(fixture ? "deterministic local fixture (not provider evidence)" : "Yahoo Finance public chart API only").append("\n")
            .append("- Watchlist: `").append(String.join("`, `", WATCHLIST)).append("`\n")
            .append("- Requested window: ").append(sessions.getFirst()).append(" through ").append(sessions.getLast()).append(" (inclusive)\n")
            .append("- Expected sessions: ").append(sessions.size()).append(" weekdays; this standalone sampler does not load the app's NSE holiday database. Choose a window of confirmed sessions.\n")
            .append("- Fyers: not measured; no authentication was attempted because the public-only I0 run does not use credentials.\n\n")
            .append("## Aggregate metrics\n\n| Metric | Value |\n|---|---:|\n")
            .append("| Requested bars | ").append(requested).append(" |\n")
            .append("| Returned bars (accepted by Yahoo client) | ").append(returned).append(" |\n")
            .append("| Session-valid bars | ").append(valid).append(" |\n")
            .append("| Missing expected bars | ").append(missing).append(" |\n")
            .append("| Invalid timestamps | ").append(invalid).append(" |\n")
            .append("| Missing-bar rate (missing / requested) | ").append(requested == 0 ? "n/a" : String.format(java.util.Locale.ROOT, "%.2f%%", 100.0 * missing / requested)).append(" |\n\n")
            .append("## Provider / interval outcome\n\n| Provider | Interval | Symbols with data | Symbols requested |\n|---|---|---:|---:|\n");
        for (Interval interval : INTERVALS) {
            long available = WATCHLIST.stream().filter(symbol -> rows.stream().anyMatch(row -> row.symbol().equals(symbol)
                && row.interval() == interval && row.providerStatus().equals("AVAILABLE"))).count();
            report.append("| ").append(fixture ? "fixture" : "Yahoo").append(" | ").append(interval.getCode()).append(" | ")
                .append(available).append(" | ").append(WATCHLIST.size()).append(" |\n");
        }
        report.append("\nReturned-bar counts are the candles accepted by `YahooFinanceClient` after its null/zero-OHLCV and out-of-range filtering; this command does not capture raw provider payloads.\n\n## Rate limits\n\nYahoo requests use the existing `YahooFinanceClient`: at least 1 second between attempts and up to 3 retries with exponential backoff for retryable failures. The command makes one request per symbol/interval and does not deliberately probe or induce a 429. ");
        if (sample.minimumCallStartGapMillis() == null) {
            report.append("No live request spacing was measured in fixture mode. ");
        } else {
            report.append("Observed minimum command-call start spacing: ").append(sample.minimumCallStartGapMillis()).append(" ms; this is not wire-level rate measurement because the client sleeps inside each call before HTTP attempts. The client enforces request-attempt spacing but does not separately expose attempt timestamps. ");
        }
        report.append("The existing `YahooFinanceClientTest.fetchCandlesRetriesRateLimitThenReturnsData` covers a 429 followed by success; actual 429s are not exposed as a per-request counter by the client.\n");
        Files.writeString(directory.resolve("report.md"), report);
    }

    private record SamplingResult(List<IntradayDataReadinessReport.LedgerRow> rows, Long minimumCallStartGapMillis) {}

    private record Options(LocalDate from, LocalDate to, Path outputDirectory, boolean fixture) {
        static Options parse(String[] args) {
            LocalDate from = LocalDate.of(2026, 10, 5);
            LocalDate to = LocalDate.of(2026, 10, 7);
            Path output = Path.of(System.getProperty("java.io.tmpdir"), "swing-trade-intraday-readiness");
            boolean fixture = false;
            for (int i = 0; i < args.length; i++) {
                switch (args[i]) {
                    case "--from" -> from = LocalDate.parse(args[++i]);
                    case "--to" -> to = LocalDate.parse(args[++i]);
                    case "--output-dir" -> output = Path.of(args[++i]);
                    case "--fixture" -> fixture = true;
                    default -> throw new IllegalArgumentException("Unknown argument: " + args[i]);
                }
            }
            if (to.isBefore(from)) throw new IllegalArgumentException("--to must be on or after --from");
            return new Options(from, to, output, fixture);
        }
    }
}
