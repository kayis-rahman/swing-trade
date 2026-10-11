package com.swingtrade.api.dto;

import com.swingtrade.strategy.IntradayBacktestConfig;
import jakarta.validation.constraints.NotNull;

import java.time.LocalDate;
import java.time.LocalTime;

/** Request to scan the active NSE watchlist at a historical 15-minute bar boundary. */
public record IntradayPaperScanRequest(
    @NotNull LocalDate sessionDate,
    @NotNull LocalTime asOfBarTime,
    IntradayBacktestConfig riskConfig
) {
    public IntradayBacktestConfig resolvedRiskConfig() {
        return riskConfig == null ? IntradayBacktestConfig.defaults() : riskConfig;
    }
}
