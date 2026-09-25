package com.swingtrade.llm.client;

import com.swingtrade.llm.SentimentType;
import com.swingtrade.llm.config.LayaProperties;
import io.github.resilience4j.circuitbreaker.CallNotPermittedException;
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.reactor.circuitbreaker.operator.CircuitBreakerOperator;
import io.netty.channel.ChannelOption;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.client.reactive.ReactorClientHttpConnector;
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;
import reactor.netty.http.client.HttpClient;

import java.util.Map;
import java.util.Optional;
import java.util.concurrent.TimeoutException;

/**
 * Client for the Laya local sentiment pre-filter service (a FastAPI service
 * run on a Mac M1, see {@code laya-service/}). Meant as a fast, short-timeout
 * classifier that can run ahead of the heavier Qwen/LLM sentiment call in
 * {@code SentimentService}.
 *
 * <p>Never throws out to the caller: any failure (unreachable host, timeout,
 * open circuit breaker, malformed response) is logged and results in an
 * empty {@link Optional}, so {@code SentimentService} can fall back to Qwen
 * without try/catch sprawl at the call site.
 */
public class LayaClient {

    private static final Logger logger = LoggerFactory.getLogger(LayaClient.class);
    private static final String CLASSIFY_PATH = "/classify";

    private final WebClient webClient;
    private final LayaProperties properties;
    private final CircuitBreaker circuitBreaker;

    public LayaClient(WebClient.Builder webClientBuilder, LayaProperties properties,
                      CircuitBreaker circuitBreaker) {
        this.properties = properties;
        this.circuitBreaker = circuitBreaker;
        var timeout = properties.getTimeout() != null ? properties.getTimeout()
                : java.time.Duration.ofSeconds(5);
        this.webClient = webClientBuilder
                .clientConnector(new ReactorClientHttpConnector(
                        HttpClient.create()
                                .option(ChannelOption.CONNECT_TIMEOUT_MILLIS, (int) timeout.toMillis())
                                .responseTimeout(timeout)))
                .build();
    }

    /**
     * Classifies {@code text} via the configured Laya service.
     *
     * @param text the combined news content to classify — must be exactly what
     *             would be sent to Qwen, so the two are comparable
     * @return the parsed result, or {@link Optional#empty()} if Laya is not
     *         configured, unreachable, slow, or returns a response this client
     *         can't parse
     */
    public Optional<LayaResult> classify(String text) {
        if (properties.getService() == null || properties.getService().getUrl() == null) {
            logger.debug("Laya service URL not configured; skipping Laya classification");
            return Optional.empty();
        }
        var timeout = properties.getTimeout() != null ? properties.getTimeout()
                : java.time.Duration.ofSeconds(5);
        try {
            Mono<LayaResult> mono = webClient.post()
                    .uri(properties.getService().getUrl().toString() + CLASSIFY_PATH)
                    .header("X-API-Key", properties.getService().getApiKey())
                    .bodyValue(Map.of("text", text))
                    .retrieve()
                    .bodyToMono(Map.class)
                    .map(this::parseResponse)
                    .timeout(timeout);

            if (circuitBreaker != null) {
                mono = mono.transformDeferred(CircuitBreakerOperator.of(circuitBreaker));
            }

            return mono
                    .onErrorResume(CallNotPermittedException.class, e -> {
                        logger.warn("Laya circuit breaker open; skipping Laya classification");
                        return Mono.empty();
                    })
                    .onErrorResume(TimeoutException.class, e -> {
                        logger.warn("Laya classification timed out after {}", timeout);
                        return Mono.empty();
                    })
                    .onErrorResume(Exception.class, e -> {
                        logger.warn("Laya classification failed: {}", e.toString());
                        return Mono.empty();
                    })
                    .blockOptional();
        } catch (Exception e) {
            // Defensive: nothing above should throw synchronously, but Laya being
            // unreachable must never propagate out of this client.
            logger.warn("Unexpected error calling Laya: {}", e.toString());
            return Optional.empty();
        }
    }

    @SuppressWarnings("unchecked")
    private LayaResult parseResponse(Map<?, ?> body) {
        Object sentimentValue = body.get("sentiment");
        Object confidenceValue = body.get("confidence");
        if (sentimentValue == null || confidenceValue == null) {
            throw new IllegalArgumentException("Laya response missing sentiment or confidence: " + body);
        }
        SentimentType sentiment = parseSentiment(sentimentValue.toString());
        double confidence = ((Number) confidenceValue).doubleValue();
        return new LayaResult(sentiment, confidence);
    }

    private SentimentType parseSentiment(String raw) {
        try {
            return SentimentType.valueOf(raw.trim().toUpperCase());
        } catch (IllegalArgumentException e) {
            throw new IllegalArgumentException("Unrecognized Laya sentiment value: " + raw, e);
        }
    }
}
