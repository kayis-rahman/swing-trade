package com.swingtrade.llm.config;

import com.swingtrade.llm.client.LayaClient;
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.circuitbreaker.CircuitBreakerConfig;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.reactive.function.client.WebClient;

import java.time.Duration;

/**
 * Configuration for the Laya local sentiment pre-filter client. Follows the
 * same per-client circuit breaker pattern as {@code Resilience4jConfig}
 * (api module) and {@code FyersResilienceConfig} (data module): a dedicated,
 * qualified {@link CircuitBreaker} bean wired into a single client bean.
 */
@Configuration
@EnableConfigurationProperties(LayaProperties.class)
public class LayaConfig {

    @Bean
    @Qualifier("laya")
    public CircuitBreaker layaCircuitBreaker() {
        return CircuitBreaker.of("laya", CircuitBreakerConfig.custom()
                .failureRateThreshold(50)
                .slidingWindowSize(10)
                .waitDurationInOpenState(Duration.ofSeconds(30))
                .permittedNumberOfCallsInHalfOpenState(3)
                .build());
    }

    @Bean
    public LayaClient layaClient(WebClient.Builder webClientBuilder, LayaProperties layaProperties,
                                 @Qualifier("laya") CircuitBreaker layaCircuitBreaker) {
        return new LayaClient(webClientBuilder, layaProperties, layaCircuitBreaker);
    }
}
