package com.swingtrade.llm.service;

import com.swingtrade.domain.store.AppSettingsStore;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

/**
 * Resolver for LLM backend selection.
 * Reads "llm.backend" from AppSettingsStore (DB/env), falls back to Spring property default.
 */
@Component
public class LlmBackendSelector {

    public enum Backend {
        LOCAL("local"),
        PI_SSH("pi_ssh"),
        OPENAI("openai"),
        OLLAMA("ollama"),
        LAYA("laya"),
        PI_AGENT("pi_agent"),
        MLX("mlx");

        private final String key;

        Backend(String key) {
            this.key = key;
        }

        public String getKey() {
            return key;
        }
    }

    private final AppSettingsStore appSettingsStore;
    private final String springDefault;

    public LlmBackendSelector(AppSettingsStore appSettingsStore,
                              @Value("${llm.backend:local}") String springDefault) {
        this.appSettingsStore = appSettingsStore;
        this.springDefault = springDefault;
    }

    public Backend resolve() {
        String key = appSettingsStore.get("llm.backend").orElse(springDefault);
        return fromKey(key);
    }

    public static Backend fromKey(String key) {
        for (Backend backend : Backend.values()) {
            if (backend.key.equals(key)) {
                return backend;
            }
        }
        return Backend.LOCAL;
    }
}
