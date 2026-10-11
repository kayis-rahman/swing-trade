package com.swingtrade.api.controller;

import com.swingtrade.api.dto.IntradayPaperSessionRequest;
import com.swingtrade.broker.service.intraday.IntradayPaperExecutionService;
import com.swingtrade.broker.service.intraday.IntradayPaperSessionResult;
import com.swingtrade.data.entity.OhlcvCandleEntity;
import com.swingtrade.data.repository.OhlcvCandleRepository;
import com.swingtrade.data.service.Interval;
import com.swingtrade.strategy.IntradayBar;
import com.swingtrade.strategy.StrategyParamsView;
import com.swingtrade.strategy.StrategyTypeRegistry;
import jakarta.validation.Valid;
import org.springframework.data.domain.Pageable;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.List;
import java.util.Map;

/** Paper-only intraday session runner; it never obtains or invokes a broker client. */
@RestController
@RequestMapping("/api/intraday/paper")
public class IntradayPaperController {
    private static final ZoneId NSE_ZONE = ZoneId.of("Asia/Kolkata");

    private final OhlcvCandleRepository candles;
    private final StrategyTypeRegistry strategies;
    private final IntradayPaperExecutionService execution;

    public IntradayPaperController(OhlcvCandleRepository candles, StrategyTypeRegistry strategies,
                                   IntradayPaperExecutionService execution) {
        this.candles = candles;
        this.strategies = strategies;
        this.execution = execution;
    }

    /** Replays stored 15-minute bars for a single date and returns paper audit records. */
    @PostMapping("/sessions")
    public ResponseEntity<IntradayPaperSessionResult> runSession(
            @Valid @RequestBody IntradayPaperSessionRequest request) {
        var strategy = strategies.findByType(request.strategyType())
            .filter(candidate -> candidate.type().startsWith("INTRADAY_"));
        if (strategy.isEmpty()) return ResponseEntity.badRequest().build();

        List<OhlcvCandleEntity> stored = candles.findBySymbolAndIntervalAndDateRange(
            request.symbol().trim().toUpperCase(), Interval.FIFTEEN_MINUTES.getCode(),
            request.sessionDate(), request.sessionDate(), Pageable.unpaged());
        try {
            List<IntradayBar> bars = stored.stream().map(candle -> new IntradayBar(
                ZonedDateTime.of(candle.getDate(), candle.getBarTime(), NSE_ZONE), candle.getOpenPrice(),
                candle.getHighPrice(), candle.getLowPrice(), candle.getClosePrice(),
                candle.getVolume() == null ? 0 : candle.getVolume())).toList();
            IntradayPaperSessionResult result = execution.runSession(
                request.symbol().trim().toUpperCase(), request.sessionDate(), bars,
                strategy.get(), StrategyParamsView.of(Map.of()), request.resolvedRiskConfig(), request.asOfBarTime());
            return ResponseEntity.ok(result);
        } catch (IllegalArgumentException e) {
            return ResponseEntity.badRequest().build();
        }
    }
}
