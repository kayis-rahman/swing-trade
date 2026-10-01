package com.swingtrade.api.config;

import com.swingtrade.api.controller.SettingsController;
import com.swingtrade.broker.service.DiscordNotificationService;
import com.swingtrade.data.service.AppSettingsService;
import com.swingtrade.data.service.MarketDataClientProvider;
import com.swingtrade.llm.config.LlmProperties;
import com.swingtrade.llm.service.LlamaCppServerManager;
import com.swingtrade.llm.service.LlmBackendSelector;
import com.swingtrade.llm.service.LlmClientProvider;
import com.swingtrade.llm.service.MlxServerManager;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Import;

@SpringBootApplication
@Import(SettingsController.class)
@EnableConfigurationProperties(LlmProperties.class)
public class SettingsTestConfiguration {

    @Bean
    public MarketDataClientProvider marketDataClientProvider() {
        return org.mockito.Mockito.mock(MarketDataClientProvider.class);
    }

    @Bean
    public AppSettingsService appSettingsService() {
        return org.mockito.Mockito.mock(AppSettingsService.class);
    }

    @Bean
    public DiscordNotificationService discordNotificationService() {
        return org.mockito.Mockito.mock(DiscordNotificationService.class);
    }

    @Bean
    public LlmBackendSelector llmBackendSelector() {
        return org.mockito.Mockito.mock(LlmBackendSelector.class);
    }

    @Bean
    public LlmClientProvider llmClientProvider() {
        return org.mockito.Mockito.mock(LlmClientProvider.class);
    }

    @Bean
    public LlamaCppServerManager localServerManager() {
        return org.mockito.Mockito.mock(LlamaCppServerManager.class);
    }

    @Bean
    public com.swingtrade.llm.service.PiLlamaServerManager piServerManager() {
        return org.mockito.Mockito.mock(com.swingtrade.llm.service.PiLlamaServerManager.class);
    }

    @Bean
    public MlxServerManager mlxServerManager() {
        return org.mockito.Mockito.mock(MlxServerManager.class);
    }
}