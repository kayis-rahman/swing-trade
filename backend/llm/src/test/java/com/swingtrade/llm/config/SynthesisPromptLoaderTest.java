package com.swingtrade.llm.config;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.extension.ExtendWith;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.context.junit.jupiter.SpringExtension;
import org.springframework.core.io.ClassPathResource;

import static org.assertj.core.api.Assertions.assertThat;

@ExtendWith(SpringExtension.class)
@ContextConfiguration(classes = SynthesisPromptLoaderTest.TestConfig.class)
@TestPropertySource(properties = {
    "spring.main.web-application-type=none"
})
class SynthesisPromptLoaderTest {

    @Autowired
    private SynthesisPromptLoader loader;

    @Test
    @DisplayName("Synthesis system prompt loads from classpath")
    void systemPromptLoads() {
        String prompt = loader.getSystemPrompt();
        assertThat(prompt).isNotBlank();
        assertThat(prompt).contains("senior equity analyst");
        assertThat(prompt).contains("Indian equity markets");
        assertThat(prompt).contains("9 analysis stages");
    }

    @Test
    @DisplayName("Synthesis system prompt limits narrative length to preserve the complete JSON object")
    void systemPromptRequiresCompactCompleteJson() {
        String prompt = loader.getSystemPrompt();
        assertThat(prompt).contains("exactly one compact, syntactically valid JSON object");
        assertThat(prompt).contains("Complete every field and close the object");
        assertThat(prompt).contains("one concise sentence");
    }

    @Test
    @DisplayName("Synthesis user prompt template loads with format specifiers")
    void userPromptHasPlaceholders() {
        String prompt = loader.getUserPromptTemplate();
        assertThat(prompt).contains("%s");
        assertThat(prompt).contains("%d");
        assertThat(prompt).contains("%.0f");
        assertThat(prompt).contains("%.1f");
    }

    @Test
    @DisplayName("Synthesis system prompt matches old hardcoded content")
    void systemPromptMatchesHardcoded() {
        String prompt = loader.getSystemPrompt();
        assertThat(prompt).contains("senior equity analyst specializing in Indian equity markets");
        assertThat(prompt).contains("9 analysis stages");
        assertThat(prompt).contains("\"recommendation\": \"BUY or SELL or HOLD\"");
        assertThat(prompt).contains("\"bullishFactors\"");
        assertThat(prompt).contains("\"bearishFactors\"");
    }

    static class TestConfig {
        @org.springframework.context.annotation.Bean
        public SynthesisPromptLoader synthesisPromptLoader() {
            return new SynthesisPromptLoader(
                new ClassPathResource("prompts/synthesis-system.md"),
                new ClassPathResource("prompts/synthesis-user.md"));
        }
    }
}