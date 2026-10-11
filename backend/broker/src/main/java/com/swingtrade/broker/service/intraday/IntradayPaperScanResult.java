package com.swingtrade.broker.service.intraday;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.time.ZonedDateTime;
import java.util.List;

/** Ephemeral, watchlist-bounded summary of a historical as-of paper scan. */
public record IntradayPaperScanResult(
    String mode,
    LocalDate sessionDate,
    LocalTime asOfBarTime,
    String universe,
    int watchlistSize,
    int scannedSymbols,
    int candidateCount,
    int liveBrokerCalls,
    List<String> excludedSymbols,
    List<Candidate> results
) {
    public IntradayPaperScanResult {
        excludedSymbols = List.copyOf(excludedSymbols);
        results = List.copyOf(results);
    }

    public record Candidate(
        String symbol,
        String strategy,
        String status,
        int expectedBars,
        int receivedBars,
        int completedTrades,
        BigDecimal netPnl,
        BigDecimal openEntryPrice,
        ZonedDateTime openEntryTime
    ) {}
}
