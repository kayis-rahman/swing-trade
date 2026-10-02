package com.swingtrade.llm.config;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.springframework.ai.openai.OpenAiChatModel;
import org.springframework.boot.autoconfigure.EnableAutoConfiguration;
import org.springframework.boot.data.jpa.autoconfigure.DataJpaRepositoriesAutoConfiguration;
import org.springframework.boot.hibernate.autoconfigure.HibernateJpaAutoConfiguration;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.ApplicationContext;
import org.springframework.test.context.TestPropertySource;

import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Integration tests for multi-client LLM configuration.
 *
 * Validates:
 * - Four OpenAiChatModel beans are created (local, pi_ssh, openai, ollama)
 * - Each reads its own base URL from settings with fallback defaults
 */
@SpringBootTest(classes = LlmConfig.class)
@EnableAutoConfiguration(
    exclude = {
        HibernateJpaAutoConfiguration.class,
        DataJpaRepositoriesAutoConfiguration.class
    },
    excludeName = "org.springframework.boot.flyway.autoconfigure.FlywayAutoConfiguration"
)
@TestPropertySource(properties = {
    "llm.backend=local",
    "spring.ai.openai.base-url=",
    "spring.ai.openai.api-key=test-key",
    "spring.ai.openai.chat.options.model=qwen3-4b",
    "llm.providers.laya.base-url=https://laya-default.test/v1",
    "llm.providers.laya.model=laya-model"
})
class LlmConfigMultiClientTest {

    @Autowired
    private ApplicationContext context;

    @MockitoBean
    private com.swingtrade.domain.store.AppSettingsStore appSettingsStore;

    @Test
    @DisplayName("should create localChatModel bean")
    void shouldCreateLocalChatModelBean() {
        OpenAiChatModel local = context.getBean("localChatModel", OpenAiChatModel.class);
        assertThat(local).isNotNull();
    }

    @Test
    @DisplayName("should create piSshChatModel bean")
    void shouldCreatePiSshChatModelBean() {
        OpenAiChatModel pi = context.getBean("piSshChatModel", OpenAiChatModel.class);
        assertThat(pi).isNotNull();
    }

    @Test
    @DisplayName("should create openAiChatModel bean")
    void shouldCreateOpenAiChatModelBean() {
        OpenAiChatModel openai = context.getBean("openAiChatModel", OpenAiChatModel.class);
        assertThat(openai).isNotNull();
    }

    @Test
    @DisplayName("should create ollamaChatModel bean")
    void shouldCreateOllamaChatModelBean() {
        OpenAiChatModel ollama = context.getBean("ollamaChatModel", OpenAiChatModel.class);
        assertThat(ollama).isNotNull();
    }

    @Test
    @DisplayName("should create layaChatModel bean")
    void shouldCreateLayaChatModelBean() {
        OpenAiChatModel laya = context.getBean("layaChatModel", OpenAiChatModel.class);
        assertThat(laya).isNotNull();
    }

    @Test
    @DisplayName("layaChatModel is wired with the configured endpoint and model")
    void layaChatModelIsWiredWithConfiguredEndpointAndModel() {
        OpenAiChatModel laya = context.getBean("layaChatModel", OpenAiChatModel.class);
        assertThat(laya.getOptions().getBaseUrl()).isEqualTo("https://laya-default.test/v1");
        assertThat(laya.getOptions().getModel()).isEqualTo("laya-model");
    }

    @Test
    @DisplayName("should have exactly five ChatModel beans")
    void shouldHaveExactlyFiveChatModelBeans() {
        Map<String, OpenAiChatModel> beans = context.getBeansOfType(OpenAiChatModel.class);
        assertThat(beans).hasSize(5);
    }
}
