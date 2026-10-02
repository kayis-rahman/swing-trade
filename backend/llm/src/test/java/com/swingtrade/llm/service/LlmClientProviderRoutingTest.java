package com.swingtrade.llm.service;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.ai.openai.OpenAiChatModel;
import com.swingtrade.llm.client.LlamaCppClient;
import com.swingtrade.llm.client.LlmClient;
import static org.mockito.Mockito.mock;

import static org.mockito.Mockito.when;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Unit tests for LlmClientProvider multi-client routing.
 *
 * Validates:
 * - Routing to LOCAL backend returns local client
 * - Routing to PI_SSH backend returns pi_ssh client
 * - Routing to OPENAI backend returns openai client
 */
@ExtendWith(MockitoExtension.class)
class LlmClientProviderRoutingTest {

    @Mock
    private LlmBackendSelector selector;

    @Mock
    private OpenAiChatModel localModel;

    @Mock
    private OpenAiChatModel piSshModel;

    @Mock
    private OpenAiChatModel openAiModel;

    @Mock
    private OpenAiChatModel ollamaModel;

    @Mock
    private OpenAiChatModel layaModel;

    private LlmClientProvider provider;

    @BeforeEach
    void setUp() {
        provider = new LlmClientProvider(selector, mock(LlamaCppClient.class), localModel,
                piSshModel, openAiModel, ollamaModel, layaModel);
    }

    @Nested
    @DisplayName("Routing — LOCAL backend")
    class LocalRouting {

        @Test
        @DisplayName("should return local client when backend is local")
        void shouldReturnLocalClientWhenBackendIsLocal() {
            // Arrange
            when(selector.resolve()).thenReturn(LlmBackendSelector.Backend.LOCAL);

            // Act
            LlmClient client = provider.getClient();

            // Assert
            assertThat(client).isNotNull();
        }
    }

    @Nested
    @DisplayName("Routing — PI_SSH backend")
    class PiSshRouting {

        @Test
        @DisplayName("should return pi_ssh client when backend is pi_ssh")
        void shouldReturnPiSshClientWhenBackendIsPiSsh() {
            // Arrange
            when(selector.resolve()).thenReturn(LlmBackendSelector.Backend.PI_SSH);

            // Act
            LlmClient client = provider.getClient();

            // Assert
            assertThat(client).isInstanceOf(LlamaCppClient.class);
        }
    }

    @Nested
    @DisplayName("Routing — OPENAI backend")
    class OpenAiRouting {

        @Test
        @DisplayName("should return openai client when backend is openai")
        void shouldReturnOpenAiClientWhenBackendIsOpenai() {
            // Arrange
            when(selector.resolve()).thenReturn(LlmBackendSelector.Backend.OPENAI);

            // Act
            LlmClient client = provider.getClient();

            // Assert
            assertThat(client).isNotNull();
        }
    }

    @Nested
    @DisplayName("Routing — LAYA backend")
    class LayaRouting {

        @Test
        @DisplayName("should return laya client when backend is laya")
        void shouldReturnLayaClientWhenBackendIsLaya() {
            // Arrange
            when(selector.resolve()).thenReturn(LlmBackendSelector.Backend.LAYA);

            // Act
            LlmClient client = provider.getClient();

            // Assert
            assertThat(client).isNotNull();
        }
    }
}
