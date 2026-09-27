package com.swingtrade.api.controller;

import com.swingtrade.api.dto.FullAnalysisResult;
import com.swingtrade.api.service.AnalysisOrchestratorService;
import io.github.resilience4j.ratelimiter.annotation.RateLimiter;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.io.IOException;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

@RestController
@RequestMapping("/api")
public class AnalysisOrchestrationController {

    private static final Logger logger = LoggerFactory.getLogger(AnalysisOrchestrationController.class);

    private final AnalysisOrchestratorService orchestrator;

    /**
     * SSE stream lifetime. Must exceed the worst case of the whole pipeline, not
     * just one stage: sentiment and synthesis each allow up to their own
     * ANALYSIS_TIMEOUT_SECONDS/TIMEOUT_SECONDS (2880s each) against the CPU-bound
     * local backend, so a slow run legitimately outlives the old hardcoded 10
     * minutes. When it did, the emitter closed underneath the still-running
     * analysis and every later stage failed with "ResponseBodyEmitter has
     * already completed".
     *
     * 120 min = both LLM stages at their worst case back-to-back (~96 min, sized
     * from measured decode throughput INCLUDING mid-request thermal-throttling
     * decay — see LlmConfig.LOCAL_LLAMA_TIMEOUT's javadoc) plus headroom for the
     * non-LLM pipeline stages around them. A real observed run finished in well
     * under a quarter of this (36 min); this ceiling exists for the rare
     * worst-case run, not the typical one.
     */
    private final long streamTimeoutMs;

    public AnalysisOrchestrationController(
            AnalysisOrchestratorService orchestrator,
            @Value("${analysis.stream.timeout-ms:7200000}") long streamTimeoutMs) {
        this.orchestrator = orchestrator;
        this.streamTimeoutMs = streamTimeoutMs;
    }

    @PostMapping("/analysis/run-full")
    @RateLimiter(name = "llmAnalysis")
    public SseEmitter runFullAnalysis(
            @RequestParam String symbol,
            @RequestParam(defaultValue = "5") int backfillYears) {

        SseEmitter emitter = new SseEmitter(streamTimeoutMs);

        // Send keep-alive pings every 30s to prevent connection drop
        ScheduledExecutorService keepAlive = Executors.newSingleThreadScheduledExecutor();
        keepAlive.scheduleAtFixedRate(() -> {
            try {
                emitter.send(SseEmitter.event().name("ping").data("keepalive"));
            } catch (IOException e) {
                keepAlive.shutdown();
            }
        }, 30, 30, TimeUnit.SECONDS);

        try {
            emitter.send(SseEmitter.event()
                .name("started")
                .data(Map.of("symbol", symbol, "backfillYears", backfillYears)));
        } catch (IOException e) {
            keepAlive.shutdown();
            return emitter;
        }

        CompletableFuture.runAsync(() -> {
            try {
                FullAnalysisResult result = orchestrator.runFullAnalysis(symbol, emitter, backfillYears);
                emitter.send(SseEmitter.event()
                    .name("complete")
                    .data(result));
                emitter.complete();
            } catch (Exception e) {
                try {
                    emitter.send(SseEmitter.event()
                        .name("error")
                        .data(Map.of("message", e.getMessage())));
                } catch (IOException ioEx) {
                    logger.error("Failed to send error: {}", ioEx.getMessage());
                }
                emitter.completeWithError(e);
            } finally {
                keepAlive.shutdown();
            }
        });

        emitter.onCompletion(() -> {
            keepAlive.shutdown();
            logger.info("Client disconnected from analysis stream for {}", symbol);
        });
        emitter.onTimeout(() -> {
            keepAlive.shutdown();
            logger.warn("Analysis stream timed out for {}", symbol);
        });
        emitter.onError(e -> {
            keepAlive.shutdown();
            logger.error("Analysis stream error for {}: {}", symbol, e.getMessage());
        });

        return emitter;
    }
}
