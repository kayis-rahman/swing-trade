package com.swingtrade.llm.config;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.boot.test.context.runner.ApplicationContextRunner;
import org.springframework.context.annotation.Configuration;

import java.net.URI;

import static org.assertj.core.api.Assertions.assertThat;

class LlmPropertiesTest {

    private final ApplicationContextRunner contextRunner = new ApplicationContextRunner()
        .withUserConfiguration(PropertiesConfiguration.class)
        .withPropertyValues(
            "llm.base-url=http://active.test/v1",
            "llm.backend=pi_ssh",
            "llm.providers.local.base-url=http://local.test/v1",
            "llm.providers.local.model=local-model",
            "llm.providers.pi-ssh.base-url=http://pi.test/v1",
            "llm.providers.pi-ssh.model=pi-model",
            "llm.providers.openai.base-url=https://openai.test/v1",
            "llm.providers.openai.model=openai-model",
            "llm.providers.ollama.base-url=http://ollama.test/v1",
            "llm.providers.ollama.model=ollama-model",
            "llm.providers.laya.base-url=http://laya.test/v1",
            "llm.providers.laya.model=laya-model",
            "llm.llama-cpp.model=/models/local.gguf",
            "llm.pdf.base-url=http://pdf.test/v1",
            "llm.pdf.model=pdf-model"
        );

    @Test
    void stageTimeoutDefaultsToLegacyDeadlineAndIsOverridable() {
        contextRunner.run(context -> assertThat(context.getBean(LlmProperties.class).getStageTimeout())
                .isEqualTo(java.time.Duration.ofSeconds(2880)));
        contextRunner.withPropertyValues("llm.stage-timeout=90s").run(context ->
                assertThat(context.getBean(LlmProperties.class).getStageTimeout())
                        .isEqualTo(java.time.Duration.ofSeconds(90)));
    }

    @Nested
    @DisplayName("Property binding")
    class PropertyBinding {

        @Test
        void shouldBindAllProviderDefaults() {
            contextRunner.run(context -> {
                LlmProperties properties = context.getBean(LlmProperties.class);

                assertThat(properties.getBaseUrl()).isEqualTo(URI.create("http://active.test/v1"));
                assertThat(properties.getBackend()).isEqualTo("pi_ssh");
                assertThat(properties.getProviders().getLocal().getBaseUrl())
                    .isEqualTo(URI.create("http://local.test/v1"));
                assertThat(properties.getProviders().getLocal().getModel()).isEqualTo("local-model");
                assertThat(properties.getProviders().getPiSsh().getBaseUrl())
                    .isEqualTo(URI.create("http://pi.test/v1"));
                assertThat(properties.getProviders().getPiSsh().getModel()).isEqualTo("pi-model");
                assertThat(properties.getProviders().getOpenai().getBaseUrl())
                    .isEqualTo(URI.create("https://openai.test/v1"));
                assertThat(properties.getProviders().getOpenai().getModel()).isEqualTo("openai-model");
                assertThat(properties.getProviders().getOllama().getBaseUrl())
                    .isEqualTo(URI.create("http://ollama.test/v1"));
                assertThat(properties.getProviders().getOllama().getModel()).isEqualTo("ollama-model");
                assertThat(properties.getProviders().getLaya().getBaseUrl())
                    .isEqualTo(URI.create("http://laya.test/v1"));
                assertThat(properties.getProviders().getLaya().getModel()).isEqualTo("laya-model");
                assertThat(properties.getLlamaCpp().getModel()).isEqualTo("/models/local.gguf");
                assertThat(properties.getPdf().getBaseUrl()).isEqualTo(URI.create("http://pdf.test/v1"));
                assertThat(properties.getPdf().getModel()).isEqualTo("pdf-model");
            });
        }
    }

    @Configuration(proxyBeanMethods = false)
    @EnableConfigurationProperties(LlmProperties.class)
    static class PropertiesConfiguration {
    }
}
