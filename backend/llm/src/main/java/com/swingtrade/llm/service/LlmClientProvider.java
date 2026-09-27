package com.swingtrade.llm.service;

import com.swingtrade.llm.client.LlmClient;
import com.swingtrade.llm.client.LlamaCppClient;
import com.swingtrade.llm.client.SpringAiLlmClient;
import com.swingtrade.llm.client.PiAgentLlmClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.ai.chat.client.ChatClient;
import org.springframework.ai.openai.OpenAiChatModel;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.stereotype.Component;

/**
 * Provider that routes LLM requests to the correct backend.
 * Creates a SpringAiLlmClient backed by the OpenAiChatModel
 * selected by LlmBackendSelector at runtime.
 */
@Component
public class LlmClientProvider {

    private static final Logger logger = LoggerFactory.getLogger(LlmClientProvider.class);

    private final LlmBackendSelector selector;
    private final LlamaCppClient llamaCppClient;
    private final OpenAiChatModel localModel;
    private final OpenAiChatModel piSshModel;
    private final OpenAiChatModel openAiModel;
    private final OpenAiChatModel ollamaModel;
    private final OpenAiChatModel layaModel;
    private final PiAgentLlmClient piAgentClient;
    private final String ollamaReasoningEffort;

    public LlmClientProvider(LlmBackendSelector selector,
                             LlamaCppClient llamaCppClient,
                             OpenAiChatModel localModel,
                             OpenAiChatModel piSshModel,
                             OpenAiChatModel openAiModel,
                             OpenAiChatModel ollamaModel,
                             OpenAiChatModel layaModel) {
        this(selector, llamaCppClient, localModel, piSshModel, openAiModel, ollamaModel,
                layaModel, new PiAgentLlmClient("pi", "openai-codex", "gpt-5.6-luna", java.time.Duration.ofSeconds(180), false), null);
    }

    /**
     * @param ollamaReasoningEffort {@code reasoning_effort} sent only to the Ollama backend.
     *        Measured against qwen3.5:4b: with default thinking the answer can land in
     *        {@code reasoning} with empty {@code content}; {@code none} disables thinking
     *        (the {@code think:false} flag is ignored on the OpenAI-compatible endpoint).
     */
    @org.springframework.beans.factory.annotation.Autowired
    public LlmClientProvider(LlmBackendSelector selector,
                             LlamaCppClient llamaCppClient,
                             @Qualifier("localChatModel") OpenAiChatModel localModel,
                             @Qualifier("piSshChatModel") OpenAiChatModel piSshModel,
                             @Qualifier("openAiChatModel") OpenAiChatModel openAiModel,
                             @Qualifier("ollamaChatModel") OpenAiChatModel ollamaModel,
                             @Qualifier("layaChatModel") OpenAiChatModel layaModel,
                             PiAgentLlmClient piAgentClient,
                             @org.springframework.beans.factory.annotation.Value("${llm.providers.ollama.reasoning-effort:none}") String ollamaReasoningEffort) {
        this.ollamaReasoningEffort = ollamaReasoningEffort;
        this.selector = selector;
        this.llamaCppClient = llamaCppClient;
        this.localModel = localModel;
        this.piSshModel = piSshModel;
        this.openAiModel = openAiModel;
        this.ollamaModel = ollamaModel;
        this.layaModel = layaModel;
        this.piAgentClient = piAgentClient;
    }

    /**
     * Returns an LlmClient backed by the OpenAiChatModel
     * selected by the backend selector.
     */
    public LlmClient getClient() {
        if (selector.resolve() == LlmBackendSelector.Backend.PI_SSH) {
            logger.info("Using native llama.cpp HTTP client for PI_SSH backend");
            return llamaCppClient;
        }
        if (selector.resolve() == LlmBackendSelector.Backend.PI_AGENT) {
            logger.info("Using Pi CLI agent provider");
            return piAgentClient;
        }
        OpenAiChatModel model = switch (selector.resolve()) {
            case LOCAL -> localModel;
            case PI_SSH -> piSshModel;
            case OPENAI -> openAiModel;
            case OLLAMA -> ollamaModel;
            case LAYA -> layaModel;
            case PI_AGENT -> throw new IllegalStateException("Pi agent should be selected before model routing");
        };
        if (model.getOptions() != null) {
            logger.info("Selected LLM backend {} with model {} at {}", selector.resolve(),
                    model.getOptions().getModel(), model.getOptions().getBaseUrl());
        }
        ChatClient chatClient = ChatClient.create(model);
        String effort = selector.resolve() == LlmBackendSelector.Backend.OLLAMA
                ? ollamaReasoningEffort : null;
        return new SpringAiLlmClient(chatClient, false, effort);
    }

    public LlmBackendSelector.Backend getBackend() {
        return selector.resolve();
    }
}
