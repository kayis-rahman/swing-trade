package com.swingtrade.api.dto;

import com.swingtrade.strategy.IntradayBacktestConfig;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;

import java.time.LocalDate;
import java.time.LocalTime;

/** Request to replay one stored 15-minute session through the isolated paper simulator. */
public record IntradayPaperSessionRequest(
    @NotBlank String symbol,
    @NotNull LocalDate sessionDate,
    @NotBlank String strategyType,
    IntradayBacktestConfig riskConfig,
    LocalTime asOfBarTime
) {
    public IntradayPaperSessionRequest(String symbol, LocalDate sessionDate, String strategyType,
                                      IntradayBacktestConfig riskConfig) {
        this(symbol, sessionDate, strategyType, riskConfig, null);
    }

    public IntradayBacktestConfig resolvedRiskConfig() {
        return riskConfig == null ? IntradayBacktestConfig.defaults() : riskConfig;
    }
}
