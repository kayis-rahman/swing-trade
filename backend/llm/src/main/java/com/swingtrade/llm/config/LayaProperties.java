package com.swingtrade.llm.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

import java.net.URI;
import java.time.Duration;

/**
 * Type-safe application defaults for the Laya local sentiment pre-filter.
 * Laya is a fast local classifier (FastAPI service on a Mac M1, see
 * {@code laya-service/}) that can run ahead of the heavier Qwen/LLM sentiment
 * call in {@code SentimentService}. Ships disabled ({@link #enabled} defaults
 * to {@code false}) since it is new and unvalidated; {@link #shadowMode}
 * defaults to {@code true} so that, once enabled, it only observes and logs
 * alongside Qwen rather than replacing it until its accuracy is confirmed.
 */
@ConfigurationProperties(prefix = "laya")
public class LayaProperties {

    private boolean enabled = false;
    private boolean shadowMode = true;
    private double confidenceThreshold = 0.75;
    /**
     * Short outer deadline for the Laya call. Laya is meant to be a fast
     * pre-filter, not the multi-minute budget the existing LLM stage uses
     * ({@code LlmProperties.stageTimeout}).
     */
    private Duration timeout = Duration.ofSeconds(5);
    private final Service service = new Service();

    public boolean isEnabled() {
        return enabled;
    }

    public void setEnabled(boolean enabled) {
        this.enabled = enabled;
    }

    public boolean isShadowMode() {
        return shadowMode;
    }

    public void setShadowMode(boolean shadowMode) {
        this.shadowMode = shadowMode;
    }

    public double getConfidenceThreshold() {
        return confidenceThreshold;
    }

    public void setConfidenceThreshold(double confidenceThreshold) {
        this.confidenceThreshold = confidenceThreshold;
    }

    public Duration getTimeout() {
        return timeout;
    }

    public void setTimeout(Duration timeout) {
        this.timeout = timeout;
    }

    public Service getService() {
        return service;
    }

    public static class Service {

        private URI url;
        private String apiKey;

        public URI getUrl() {
            return url;
        }

        public void setUrl(URI url) {
            this.url = url;
        }

        public String getApiKey() {
            return apiKey;
        }

        public void setApiKey(String apiKey) {
            this.apiKey = apiKey;
        }
    }
}
