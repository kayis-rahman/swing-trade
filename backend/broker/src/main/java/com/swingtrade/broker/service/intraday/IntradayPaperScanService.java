package com.swingtrade.broker.service.intraday;

import com.swingtrade.data.entity.OhlcvCandleEntity;
import com.swingtrade.data.entity.WatchlistEntity;
import com.swingtrade.data.repository.OhlcvCandleRepository;
import com.swingtrade.data.repository.WatchlistRepository;
import com.swingtrade.data.service.Interval;
import com.swingtrade.strategy.IntradayBacktestConfig;
import com.swingtrade.strategy.IntradayBar;
import com.swingtrade.strategy.SignalStrategy;
import com.swingtrade.strategy.StrategyParamsView;
import com.swingtrade.strategy.StrategyTypeRegistry;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.domain.Pageable;
import org.springframework.stereotype.Service;

import java.time.LocalDate;
import java.time.LocalTime;
import java.time.ZoneId;
import java.util.Comparator;
import java.util.List;
import java.util.Locale;

/** Runs a retrospective as-of scan over stored bars for the bounded active NSE watchlist. */
@Service
public class IntradayPaperScanService {
    private static final ZoneId NSE_ZONE = ZoneId.of("Asia/Kolkata");
    private static final String MODE = "PAPER_SCAN_AS_OF";

    private final WatchlistRepository watchlist;
    private final OhlcvCandleRepository candles;
    private final StrategyTypeRegistry strategies;
    private final IntradayPaperExecutionService execution;
    private final int maxWatchlistSymbols;

    public IntradayPaperScanService(WatchlistRepository watchlist, OhlcvCandleRepository candles,
                                    StrategyTypeRegistry strategies, IntradayPaperExecutionService execution,
                                    @Value("${intraday.scan.max-watchlist-symbols:20}") int maxWatchlistSymbols) {
        if (maxWatchlistSymbols <= 0) throw new IllegalArgumentException("Scan symbol limit must be positive");
        this.watchlist = watchlist;
        this.candles = candles;
        this.strategies = strategies;
        this.execution = execution;
        this.maxWatchlistSymbols = maxWatchlistSymbols;
    }

    public IntradayPaperScanResult scan(LocalDate sessionDate, LocalTime asOfBarTime,
                                        IntradayBacktestConfig riskConfig) {
        if (sessionDate == null || asOfBarTime == null) {
            throw new IllegalArgumentException("Session date and as-of bar time are required");
        }
        List<WatchlistEntity> active = watchlist.findByIsActiveTrueOrderBySymbolAsc();
        if (active.size() > maxWatchlistSymbols) {
            throw new IllegalStateException("Active watchlist has %d symbols; intraday scan limit is %d"
                .formatted(active.size(), maxWatchlistSymbols));
        }

        List<String> symbols = active.stream().filter(IntradayPaperScanService::isNseSymbol)
            .map(row -> row.getSymbol().trim().toUpperCase(Locale.ROOT)).distinct().sorted().toList();
        List<String> excluded = active.stream().filter(row -> !isNseSymbol(row))
            .map(WatchlistEntity::getSymbol).sorted().toList();
        List<SignalStrategy> intradayStrategies = strategies.all().stream()
            .filter(strategy -> strategy.type().startsWith("INTRADAY_"))
            .sorted(Comparator.comparing(SignalStrategy::type)).toList();

        IntradayBacktestConfig resolvedRisk = riskConfig == null ? IntradayBacktestConfig.defaults() : riskConfig;
        List<IntradayPaperScanResult.Candidate> results = symbols.stream().flatMap(symbol -> {
            List<IntradayBar> bars = candles.findBySymbolAndIntervalAndDateRange(symbol,
                    Interval.FIFTEEN_MINUTES.getCode(), sessionDate, sessionDate, Pageable.unpaged())
                .stream().map(IntradayPaperScanService::toBar)
                .filter(bar -> !bar.sessionTime().isAfter(asOfBarTime)).toList();
            return intradayStrategies.stream().map(strategy -> summarize(symbol, sessionDate, asOfBarTime,
                bars, strategy, resolvedRisk));
        }).toList();

        int candidates = (int) results.stream().filter(row -> row.status().equals("CANDIDATE")).count();
        return new IntradayPaperScanResult(MODE, sessionDate, asOfBarTime, "ACTIVE_NSE_WATCHLIST",
            active.size(), symbols.size(), candidates, 0, excluded, results);
    }

    private IntradayPaperScanResult.Candidate summarize(String symbol, LocalDate date, LocalTime asOf,
                                                         List<IntradayBar> bars, SignalStrategy strategy,
                                                         IntradayBacktestConfig riskConfig) {
        IntradayPaperSessionResult replay = execution.runSession(symbol, date, bars, strategy,
            StrategyParamsView.of(java.util.Map.of()), riskConfig, asOf);
        var openPosition = replay.positions().stream().filter(position -> position.status().equals("OPEN"))
            .findFirst().orElse(null);
        String status = switch (replay.status()) {
            case "INCOMPLETE_DATA" -> "INCOMPLETE_DATA";
            case "BLOCKED_KILL_SWITCH" -> "BLOCKED_KILL_SWITCH";
            default -> openPosition == null ? "NO_OPEN_SETUP" : "CANDIDATE";
        };
        return new IntradayPaperScanResult.Candidate(symbol, strategy.type(), status,
            replay.expectedBars(), replay.receivedBars(), replay.trades().size(), replay.netPnl(),
            openPosition == null ? null : openPosition.entryPrice(),
            openPosition == null ? null : openPosition.entryTime());
    }

    private static IntradayBar toBar(OhlcvCandleEntity candle) {
        return new IntradayBar(java.time.ZonedDateTime.of(candle.getDate(), candle.getBarTime(), NSE_ZONE),
            candle.getOpenPrice(), candle.getHighPrice(), candle.getLowPrice(), candle.getClosePrice(),
            candle.getVolume() == null ? 0 : candle.getVolume());
    }

    private static boolean isNseSymbol(WatchlistEntity row) {
        String exchange = row.getExchange();
        return exchange == null || exchange.isBlank() || exchange.equalsIgnoreCase("NSE");
    }
}
