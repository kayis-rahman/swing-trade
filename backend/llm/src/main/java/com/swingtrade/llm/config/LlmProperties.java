package com.swingtrade.llm.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

import java.net.URI;

/**
 * Type-safe application defaults for LLM providers and settings.
 * Runtime values persisted in {@code AppSettingsStore} take precedence over these defaults.
 * API keys are deliberately excluded so this bean is safe to expose through config metadata.
 */
@ConfigurationProperties(prefix = "llm")
public class LlmProperties {

    private URI baseUrl;
    private String backend;
    private final Providers providers = new Providers();
    private final LlamaCpp llamaCpp = new LlamaCpp();
    private final Pdf pdf = new Pdf();
    /**
     * Outer deadline for one LLM stage call (sentiment, synthesis). Default matches the
     * historical 2880s ceiling sized for CPU-bound llama.cpp; lower it for fast backends
     * so a hung local model cannot hold a run for hours.
     */
    private java.time.Duration stageTimeout = java.time.Duration.ofSeconds(2880);

    public URI getBaseUrl() {
        return baseUrl;
    }

    public void setBaseUrl(URI baseUrl) {
        this.baseUrl = baseUrl;
    }

    public String getBackend() {
        return backend;
    }

    public void setBackend(String backend) {
        this.backend = backend;
    }

    public Providers getProviders() {
        return providers;
    }

    public LlamaCpp getLlamaCpp() {
        return llamaCpp;
    }

    public java.time.Duration getStageTimeout() {
        return stageTimeout;
    }

    public void setStageTimeout(java.time.Duration stageTimeout) {
        this.stageTimeout = stageTimeout;
    }

    public Pdf getPdf() {
        return pdf;
    }

    public static class Providers {

        private final Provider local = new Provider();
        private final Provider piSsh = new Provider();
        private final Provider openai = new Provider();
        private final Provider ollama = new Provider();
        private final Provider laya = new Provider();

        public Provider getLocal() {
            return local;
        }

        public Provider getPiSsh() {
            return piSsh;
        }

        public Provider getOpenai() {
            return openai;
        }

        public Provider getOllama() {
            return ollama;
        }

        public Provider getLaya() {
            return laya;
        }
    }

    public static class Provider {

        private URI baseUrl;
        private String model;

        public URI getBaseUrl() {
            return baseUrl;
        }

        public void setBaseUrl(URI baseUrl) {
            this.baseUrl = baseUrl;
        }

        public String getModel() {
            return model;
        }

        public void setModel(String model) {
            this.model = model;
        }
    }

    public static class LlamaCpp {

        private String model;

        public String getModel() {
            return model;
        }

        public void setModel(String model) {
            this.model = model;
        }
    }

    public static class Pdf {

        private URI baseUrl;
        private String model;

        public URI getBaseUrl() {
            return baseUrl;
        }

        public void setBaseUrl(URI baseUrl) {
            this.baseUrl = baseUrl;
        }

        public String getModel() {
            return model;
        }

        public void setModel(String model) {
            this.model = model;
        }
    }
}
