package com.swingtrade.data.service;

import com.swingtrade.core.metrics.DataIngestionMetrics;
import io.micrometer.core.instrument.simple.SimpleMeterRegistry;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class MarketDataClientProviderTest {

    private MarketDataClient yahoo;
    private MarketDataClient fyers;
    private AppSettingsService appSettingsService;
    private MarketDataClientProvider provider;

    @BeforeEach
    void setUp() {
        yahoo = mock(MarketDataClient.class);
        fyers = mock(MarketDataClient.class);
        appSettingsService = mock(AppSettingsService.class);
        when(appSettingsService.get(eq("selectedBroker"), any())).thenReturn(null);
        provider = new MarketDataClientProvider(Map.of("yahoo", yahoo, "fyers", fyers), appSettingsService);
    }

    @Test
    void defaultsToYahoo() {
        assertThat(provider.getActiveBroker()).isEqualTo("yahoo");
        assertThat(provider.getClient()).isSameAs(yahoo);
    }

    @Test
    void switchesActiveBroker() {
        provider.setActiveBroker("fyers");

        assertThat(provider.getActiveBroker()).isEqualTo("fyers");
        assertThat(provider.getClient()).isSameAs(fyers);
    }

    @Test
    void unknownBrokerIsRejectedAndKeepsPreviousActive() {
        provider.setActiveBroker("fyers");
        provider.setActiveBroker("upstox"); // not registered in this map

        assertThat(provider.getActiveBroker()).isEqualTo("fyers");
        assertThat(provider.getClient()).isSameAs(fyers);
    }

    @Test
    void stagePaperModeIgnoresPersistedFyersSelectionWithoutChangingSimulationBroker() {
        when(appSettingsService.get(eq("selectedBroker"), any())).thenReturn("fyers");
        MarketDataClientProvider stage = new MarketDataClientProvider(
            Map.of("yahoo", yahoo, "fyers", fyers), appSettingsService, false);

        stage.restorePersistedBroker();
        stage.setActiveBroker("fyers");

        assertThat(stage.getActiveBroker()).isEqualTo("yahoo");
        assertThat(stage.getClient()).isSameAs(yahoo);
    }

    @Test
    void restoresPersistedBrokerOnStartup() {
        when(appSettingsService.get(eq("selectedBroker"), any())).thenReturn("fyers");
        MarketDataClientProvider restarted =
            new MarketDataClientProvider(Map.of("yahoo", yahoo, "fyers", fyers), appSettingsService);

        restarted.restorePersistedBroker();

        assertThat(restarted.getActiveBroker()).isEqualTo("fyers");
        assertThat(restarted.getClient()).isSameAs(fyers);
    }

    @Test
    void restorePersistedBrokerIsNoOpWhenNothingPersisted() {
        provider.restorePersistedBroker();

        assertThat(provider.getActiveBroker()).isEqualTo("yahoo");
    }

    @Test
    void compatibilityConstructorDoesNotWrapClients() {
        assertThat(provider.getClient()).isSameAs(yahoo);
    }

    @Test
    void productionConstructorSharesOneLimiterAcrossProviderAliases() {
        DataIngestionMetrics metrics = new DataIngestionMetrics(new SimpleMeterRegistry());
        MarketDataClient productionProvider = new MarketDataClientProvider(
            Map.of("yahoo", yahoo, "fyers", fyers, "fyersServiceClient", fyers),
            appSettingsService, metrics, 0, 100, 60, 30, true).getClient();

        MarketDataClientProvider configured = new MarketDataClientProvider(
            Map.of("yahoo", yahoo, "fyers", fyers, "fyersServiceClient", fyers),
            appSettingsService, metrics, 0, 100, 60, 30, true);
        configured.setActiveBroker("fyers");
        MarketDataClient fyersAliasLimiter = configured.getClient();
        configured.setActiveBroker("fyersServiceClient");

        assertThat(productionProvider).isNotSameAs(yahoo);
        assertThat(configured.getClient()).isSameAs(fyersAliasLimiter);
    }
}
