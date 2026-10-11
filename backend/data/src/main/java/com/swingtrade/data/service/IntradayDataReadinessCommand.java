package com.swingtrade.data.service;

import tools.jackson.databind.ObjectMapper;
import com.swingtrade.data.client.YahooFinanceClient;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Clock;
import java.time.LocalDate;
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
        SamplingResult sample = fetchYahoo(sessions);
        writeOutputs(options.outputDirectory(), sessions, sample);
        System.out.println("I0 report: " + options.outputDirectory().resolve("report.md"));
        System.out.println("I0 ledger: " + options.outputDirectory().resolve("ledger.csv"));
    }

    private static SamplingResult fetchYahoo(List<LocalDate> sessions) {
        YahooFinanceClient yahoo = new YahooFinanceClient(new ObjectMapper(), Clock.systemUTC());
        LocalDate from = sessions.getFirst();
        LocalDate to = sessions.getLast();
        List<IntradayDataReadinessReport.LedgerRow> rows = new ArrayList<>();
        for (String symbol : WATCHLIST) {
            for (Interval interval : INTERVALS) {
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
        return new SamplingResult(rows, yahoo.requestMetrics());
    }

    private static void writeOutputs(Path directory, List<LocalDate> sessions,
                                     SamplingResult sample) throws IOException {
        List<IntradayDataReadinessReport.LedgerRow> rows = sample.rows();
        Files.createDirectories(directory);
        StringBuilder csv = new StringBuilder("symbol,provider,interval,session_date,requested_bars,returned_bars,valid_session_bars,missing_bars,invalid_timestamp_bars,missing_rate,provider_status,elapsed_ms\n");
        for (IntradayDataReadinessReport.LedgerRow row : rows) {
            csv.append(row.symbol()).append(",yahoo,")
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
        report.append("- Source: Yahoo Finance public chart API only\n")
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
            report.append("| Yahoo | ").append(interval.getCode()).append(" | ")
                .append(available).append(" | ").append(WATCHLIST.size()).append(" |\n");
        }
        YahooFinanceClient.RequestMetricsSnapshot metrics = sample.requestMetrics();
        report.append("\nReturned-bar counts are the candles accepted by `YahooFinanceClient` after its null/zero-OHLCV and out-of-range filtering; this command does not capture raw provider payloads.\n\n## Rate limits\n\nYahoo requests use the existing `YahooFinanceClient`: at least 1 second between attempts and up to 3 retries with exponential backoff for retryable failures. The command makes one request per symbol/interval and does not deliberately probe or induce a 429.\n\n")
            .append("- HTTP attempts: ").append(metrics.attempts()).append("\n")
            .append("- Retries: ").append(metrics.retries()).append("\n")
            .append("- Minimum spacing between HTTP attempts: ").append(metrics.minimumAttemptSpacingMillis()).append(" ms (0 when fewer than two attempts)\n")
            .append("- HTTP 429 responses: ").append(metrics.non2xxStatuses().getOrDefault(429, 0L)).append("\n")
            .append("- Other non-2xx responses: ").append(metrics.non2xxStatuses().entrySet().stream()
                .filter(entry -> entry.getKey() != 429)
                .mapToLong(java.util.Map.Entry::getValue).sum()).append("\n")
            .append("- Non-2xx status counts: ").append(metrics.non2xxStatuses().isEmpty() ? "0" : metrics.non2xxStatuses()).append("\n");
        Files.writeString(directory.resolve("report.md"), report);
    }

    private record SamplingResult(List<IntradayDataReadinessReport.LedgerRow> rows,
                                  YahooFinanceClient.RequestMetricsSnapshot requestMetrics) {}

    private record Options(LocalDate from, LocalDate to, Path outputDirectory) {
        static Options parse(String[] args) {
            LocalDate from = LocalDate.of(2026, 10, 5);
            LocalDate to = LocalDate.of(2026, 10, 7);
            Path output = Path.of(System.getProperty("java.io.tmpdir"), "swing-trade-intraday-readiness");
            for (int i = 0; i < args.length; i++) {
                switch (args[i]) {
                    case "--from" -> from = LocalDate.parse(args[++i]);
                    case "--to" -> to = LocalDate.parse(args[++i]);
                    case "--output-dir" -> output = Path.of(args[++i]);
                    default -> throw new IllegalArgumentException("Unknown argument: " + args[i]);
                }
            }
            if (to.isBefore(from)) throw new IllegalArgumentException("--to must be on or after --from");
            return new Options(from, to, output);
        }
    }
}
