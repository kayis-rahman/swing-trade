package com.swingtrade.llm.config;

import com.swingtrade.domain.store.StockStore;
import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.domain.store.SentimentStore;
import com.swingtrade.llm.client.SpringAiLlmClient;
import com.swingtrade.llm.service.NewsFilterService;
import com.swingtrade.llm.service.NewsIngestionService;
import com.swingtrade.llm.service.SentimentService;
import com.swingtrade.llm.service.SentimentAnalyzer;
import com.swingtrade.llm.service.LlmBackendSelector;
import com.swingtrade.llm.service.LlmClientProvider;
import com.swingtrade.llm.service.LlmServerManagerProvider;
import org.springframework.ai.openai.OpenAiChatModel;
import tools.jackson.databind.ObjectMapper;
import org.springframework.boot.autoconfigure.EnableAutoConfiguration;
import org.springframework.boot.data.jpa.autoconfigure.DataJpaRepositoriesAutoConfiguration;
import org.springframework.boot.hibernate.autoconfigure.HibernateJpaAutoConfiguration;
import org.springframework.boot.test.context.TestConfiguration;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.annotation.Primary;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;
import org.springframework.orm.jpa.LocalContainerEntityManagerFactoryBean;
import org.springframework.orm.jpa.vendor.HibernateJpaVendorAdapter;

import javax.sql.DataSource;
import java.util.Properties;

/**
 * Test configuration for LLM module E2E tests.
 * Provides real beans for integration testing with local llama.cpp.
 */
@TestConfiguration
@EnableAutoConfiguration(
    exclude = {
        DataJpaRepositoriesAutoConfiguration.class,
        HibernateJpaAutoConfiguration.class
    },
    excludeName = "org.springframework.boot.flyway.autoconfigure.FlywayAutoConfiguration"
)
@ComponentScan(basePackages = {"com.swingtrade.llm", "com.swingtrade.data", "com.swingtrade.core.metrics"})
@EnableJpaRepositories(basePackages = "com.swingtrade.data.repository")
public class TestLlmConfig {

    @MockitoBean
    private AppSettingsStore appSettingsStore;

    @MockitoBean
    private SentimentStore sentimentStore;

    // LlmServerManagerProvider handles server lifecycle now

    @Primary
    @Bean
    public SentimentPromptLoader sentimentPromptLoader() {
        return new SentimentPromptLoader(
                new org.springframework.core.io.ClassPathResource("prompts/sentiment-system.md"),
                new org.springframework.core.io.ClassPathResource("prompts/sentiment-user.md"));
    }

    @Primary
    @Bean
    public SynthesisPromptLoader synthesisPromptLoader() {
        return new SynthesisPromptLoader(
                new org.springframework.core.io.ClassPathResource("prompts/synthesis-system.md"),
                new org.springframework.core.io.ClassPathResource("prompts/synthesis-user.md"));
    }

    @Primary
    @Bean
    public PdfExtractionPromptLoader pdfExtractionPromptLoader() {
        return new PdfExtractionPromptLoader(
                new org.springframework.core.io.ClassPathResource("prompts/pdf-extraction.md"));
    }

    @Primary
    @Bean
    public SentimentAnalyzer sentimentAnalyzer() {
        return new SentimentAnalyzer(new tools.jackson.databind.ObjectMapper());
    }

    @Primary
    @Bean
    public NewsFilterService newsFilterService() {
        return new NewsFilterService();
    }

    @Primary
    @Bean
    public ObjectMapper objectMapper() {
        return new ObjectMapper();
    }

    @Primary
    @Bean
    public SpringAiLlmClient springAiLlmClient() {
        // Mock ChatClient for tests — returns empty content
        var mockChatClient = org.mockito.Mockito.mock(org.springframework.ai.chat.client.ChatClient.class);
        var mockRequestSpec = org.mockito.Mockito.mock(org.springframework.ai.chat.client.ChatClient.ChatClientRequestSpec.class);
        var mockCallSpec = org.mockito.Mockito.mock(org.springframework.ai.chat.client.ChatClient.CallResponseSpec.class);
        var mockChatResponse = org.mockito.Mockito.mock(org.springframework.ai.chat.model.ChatResponse.class);
        var mockGeneration = org.mockito.Mockito.mock(org.springframework.ai.chat.model.Generation.class);
        var mockAssistantMessage = org.mockito.Mockito.mock(org.springframework.ai.chat.messages.AssistantMessage.class);

        org.mockito.Mockito.when(mockChatClient.prompt())
                .thenReturn(mockRequestSpec);
        org.mockito.Mockito.when(mockRequestSpec.system(org.mockito.ArgumentMatchers.anyString()))
                .thenReturn(mockRequestSpec);
        org.mockito.Mockito.when(mockRequestSpec.user(org.mockito.ArgumentMatchers.anyString()))
                .thenReturn(mockRequestSpec);
        org.mockito.Mockito.when(mockRequestSpec.call())
                .thenReturn(mockCallSpec);
        org.mockito.Mockito.when(mockCallSpec.chatResponse())
                .thenReturn(mockChatResponse);
        org.mockito.Mockito.when(mockChatResponse.getResult())
                .thenReturn(mockGeneration);
        org.mockito.Mockito.when(mockGeneration.getOutput())
                .thenReturn(mockAssistantMessage);
        org.mockito.Mockito.when(mockAssistantMessage.getText())
                .thenReturn("");

        return new SpringAiLlmClient(mockChatClient, false);
    }

    @Primary
    @Bean
    public LlmClientProvider llmClientProvider(
            LlmBackendSelector selector,
            com.swingtrade.llm.client.LlamaCppClient llamaCppClient,
            OpenAiChatModel localChatModel,
            OpenAiChatModel piSshChatModel,
            OpenAiChatModel openAiChatModel,
            OpenAiChatModel ollamaChatModel,
            OpenAiChatModel layaChatModel) {
        return new LlmClientProvider(selector, llamaCppClient,
                localChatModel, piSshChatModel, openAiChatModel, ollamaChatModel, layaChatModel);
    }

    // ===== H2 Database Configuration for Testing =====

    @Primary
    @Bean
    public DataSource dataSource() {
        org.springframework.boot.jdbc.DataSourceBuilder<?> dataSourceBuilder =
                org.springframework.boot.jdbc.DataSourceBuilder.create();
        dataSourceBuilder.driverClassName("org.h2.Driver");
        dataSourceBuilder.url("jdbc:h2:mem:testdb;DB_CLOSE_DELAY=-1;DB_CLOSE_ON_EXIT=FALSE");
        dataSourceBuilder.username("sa");
        dataSourceBuilder.password("");
        return dataSourceBuilder.build();
    }

    @Primary
    @Bean
    public LocalContainerEntityManagerFactoryBean entityManagerFactory(
            DataSource dataSource) {
        LocalContainerEntityManagerFactoryBean em = new LocalContainerEntityManagerFactoryBean();
        em.setDataSource(dataSource);
        em.setPackagesToScan("com.swingtrade.data.entity");

        HibernateJpaVendorAdapter vendorAdapter = new HibernateJpaVendorAdapter();
        em.setJpaVendorAdapter(vendorAdapter);

        Properties properties = new Properties();
        properties.setProperty("hibernate.hbm2ddl.auto", "create-drop");
        properties.setProperty("hibernate.show_sql", "false");
        properties.setProperty("hibernate.dialect", "org.hibernate.dialect.H2Dialect");
        em.setJpaProperties(properties);

        return em;
    }

    @Primary
    @Bean
    public SentimentService sentimentAnalysisService(
            LlmClientProvider clientProvider,
            LlmServerManagerProvider serverManagerProvider,
            SentimentPromptLoader promptLoader,
            SentimentAnalyzer sentimentAnalyzer,
            NewsIngestionService newsIngestionService,
            SentimentStore sentimentStore,
            StockStore stockStore,
            AppSettingsStore appSettingsStore,
            com.swingtrade.core.metrics.LlmMetrics llmMetrics,
            com.swingtrade.core.metrics.SentimentMetrics sentimentMetrics) {
        return new SentimentService(
                clientProvider,
                serverManagerProvider,
                promptLoader,
                sentimentAnalyzer,
                newsIngestionService,
                sentimentStore,
                stockStore,
                appSettingsStore,
                llmMetrics,
                sentimentMetrics,
                0.75
        );
    }
}
