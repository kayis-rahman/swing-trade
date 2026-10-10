package com.swingtrade.llm.service;

import org.springframework.stereotype.Component;

/**
 * Provider that returns the correct LlmServerManager based on the selected backend.
 * Returns null for OPENAI, OLLAMA, or LAYA (no server to manage).
 */
@Component
public class LlmServerManagerProvider {

    private final LlmBackendSelector selector;
    private final LlamaCppServerManager localServerManager;
    private final PiLlamaServerManager piServerManager;

    public LlmServerManagerProvider(LlmBackendSelector selector,
                                    LlamaCppServerManager localServerManager,
                                    PiLlamaServerManager piServerManager) {
        this.selector = selector;
        this.localServerManager = localServerManager;
        this.piServerManager = piServerManager;
    }

    /**
     * Returns the LlmServerManager for the currently selected backend.
     * Returns null for OPENAI, OLLAMA, or LAYA (no server to manage).
     */
    public LlmServerManager getManager() {
        return getManager(selector.resolve());
    }

    public LlmServerManager getManager(LlmBackendSelector.Backend backend) {
        return switch (backend) {
            case LOCAL -> localServerManager;
            case PI_SSH -> piServerManager;
            case OPENAI, OLLAMA, LAYA, PI_AGENT -> null;
        };
    }
}
