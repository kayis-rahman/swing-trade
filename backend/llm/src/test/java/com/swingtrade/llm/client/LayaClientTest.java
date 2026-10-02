package com.swingtrade.llm.client;

import com.swingtrade.llm.SentimentType;
import com.swingtrade.llm.config.LayaProperties;
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.circuitbreaker.CircuitBreakerConfig;
import okhttp3.mockwebserver.MockResponse;
import okhttp3.mockwebserver.MockWebServer;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.web.reactive.function.client.WebClient;

import java.net.URI;
import java.time.Duration;
import java.util.Optional;
import java.util.concurrent.TimeUnit;

import static org.assertj.core.api.Assertions.assertThat;

class LayaClientTest {

    private MockWebServer server;
    private LayaProperties properties;

    @BeforeEach
    void setUp() throws Exception {
        server = new MockWebServer();
        server.start();

        properties = new LayaProperties();
        properties.setEnabled(true);
        properties.setTimeout(Duration.ofMillis(500));
        properties.getService().setUrl(URI.create(server.url("/").toString().replaceAll("/$", "")));
        properties.getService().setApiKey("test-key");
    }

    @AfterEach
    void tearDown() throws Exception {
        server.shutdown();
    }

    private LayaClient newClient() {
        return newClient(freshCircuitBreaker());
    }

    private LayaClient newClient(CircuitBreaker circuitBreaker) {
        return new LayaClient(WebClient.builder(), properties, circuitBreaker);
    }

    private CircuitBreaker freshCircuitBreaker() {
        return CircuitBreaker.of("laya-test", CircuitBreakerConfig.custom()
                .failureRateThreshold(50)
                .slidingWindowSize(10)
                .waitDurationInOpenState(Duration.ofSeconds(30))
                .permittedNumberOfCallsInHalfOpenState(3)
                .build());
    }

    @Test
    void returnsParsedResultOnSuccess() throws Exception {
        server.enqueue(new MockResponse().setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("{\"sentiment\":\"POSITIVE\",\"confidence\":0.91}"));

        Optional<LayaResult> result = newClient().classify("Company beats estimates");

        assertThat(result).isPresent();
        assertThat(result.get().sentiment()).isEqualTo(SentimentType.POSITIVE);
        assertThat(result.get().confidence()).isEqualTo(0.91);

        var request = server.takeRequest(1, TimeUnit.SECONDS);
        assertThat(request).isNotNull();
        assertThat(request.getPath()).isEqualTo("/classify");
        assertThat(request.getHeader("X-API-Key")).isEqualTo("test-key");
        assertThat(request.getBody().readUtf8()).contains("Company beats estimates");
    }

    @Test
    void returnsEmptyOnTimeout() throws Exception {
        // No enqueued response within the client's timeout -> MockWebServer holds
        // the connection open, so the client's own timeout must fire first.
        server.enqueue(new MockResponse().setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("{\"sentiment\":\"POSITIVE\",\"confidence\":0.9}")
                .setBodyDelay(2, TimeUnit.SECONDS));

        Optional<LayaResult> result = newClient().classify("slow response");

        assertThat(result).isEmpty();
    }

    @Test
    void returnsEmptyOnMalformedJsonResponse() {
        server.enqueue(new MockResponse().setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("not json at all"));

        Optional<LayaResult> result = newClient().classify("text");

        assertThat(result).isEmpty();
    }

    @Test
    void returnsEmptyWhenResponseMissingFields() {
        server.enqueue(new MockResponse().setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("{\"sentiment\":\"POSITIVE\"}"));

        Optional<LayaResult> result = newClient().classify("text");

        assertThat(result).isEmpty();
    }

    @Test
    void returnsEmptyOnUnrecognizedSentimentValue() {
        server.enqueue(new MockResponse().setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("{\"sentiment\":\"MEH\",\"confidence\":0.5}"));

        Optional<LayaResult> result = newClient().classify("text");

        assertThat(result).isEmpty();
    }

    @Test
    void returnsEmptyOnHttpErrorStatus() {
        server.enqueue(new MockResponse().setResponseCode(500)
                .setHeader("Content-Type", "text/plain")
                .setBody("internal error"));

        Optional<LayaResult> result = newClient().classify("text");

        assertThat(result).isEmpty();
    }

    @Test
    void returnsEmptyWhenServiceUrlNotConfigured() {
        LayaProperties unconfigured = new LayaProperties();
        unconfigured.setEnabled(true);
        LayaClient client = new LayaClient(WebClient.builder(), unconfigured, freshCircuitBreaker());

        Optional<LayaResult> result = client.classify("text");

        assertThat(result).isEmpty();
    }

    @Test
    void returnsEmptyWhenCircuitBreakerIsOpen() {
        CircuitBreaker openBreaker = CircuitBreaker.of("laya-open-test", CircuitBreakerConfig.custom()
                .failureRateThreshold(50)
                .slidingWindowSize(2)
                .waitDurationInOpenState(Duration.ofMinutes(1))
                .build());
        openBreaker.transitionToOpenState();

        // No response enqueued: an attempt would hang/fail, proving the circuit
        // breaker short-circuited before any request was made.
        Optional<LayaResult> result = newClient(openBreaker).classify("text");

        assertThat(result).isEmpty();
        assertThat(server.getRequestCount()).isZero();
    }
}
