package com.swingtrade.data.service;

import com.swingtrade.core.metrics.DataIngestionMetrics;
import jakarta.annotation.PostConstruct;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.util.Map;
import java.util.HashMap;
import java.util.IdentityHashMap;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Resolves the active MarketDataClient at runtime based on user-selected broker.
 * Default is Yahoo Finance (no auth required). Users can switch to fyers/upstox
 * via the /api/settings endpoint.
 */
@Service
public class MarketDataClientProvider {

    private static final Logger logger = LoggerFactory.getLogger(MarketDataClientProvider.class);
    private static final String FYERS = "fyers";
    private static final String SELECTED_BROKER_SETTING = "selectedBroker";

    private final Map<String, MarketDataClient> clients;
    private final AppSettingsService appSettingsService;
    private final boolean fyersEnabled;
    private final AtomicReference<String> activeBroker;

    public MarketDataClientProvider(Map<String, MarketDataClient> clients, AppSettingsService appSettingsService) {
        this(clients, appSettingsService, true);
    }

    MarketDataClientProvider(Map<String, MarketDataClient> clients, AppSettingsService appSettingsService,
                             boolean fyersEnabled) {
        this.clients = normalizeClientNames(clients);
        this.appSettingsService = appSettingsService;
        this.fyersEnabled = fyersEnabled;
        String configuredBroker = appSettingsService.get(SELECTED_BROKER_SETTING, "yahoo");
        this.activeBroker = new AtomicReference<>(isAvailable(configuredBroker) ? configuredBroker : "yahoo");
        logger.info("Market data provider initialized without request throttling (compatibility constructor)");
    }

    @Autowired
    MarketDataClientProvider(Map<String, MarketDataClient> clients, AppSettingsService appSettingsService,
                             DataIngestionMetrics metrics,
                             @Value("${market-data.rate-limit-ms:250}") long rateLimitMillis,
                             @Value("${data.rate-limit.upstox.requests-per-minute:100}") long upstoxRpm,
                             @Value("${data.rate-limit.fyers.requests-per-minute:60}") long fyersRpm,
                             @Value("${data.rate-limit.default.requests-per-minute:30}") long defaultRpm,
                             @Value("${market-data.fyers.enabled:true}") boolean fyersEnabled) {
        this.clients = rateLimitClients(normalizeClientNames(clients), metrics, rateLimitMillis,
            upstoxRpm, fyersRpm, defaultRpm);
        this.appSettingsService = appSettingsService;
        this.fyersEnabled = fyersEnabled;
        String configuredBroker = appSettingsService.get(SELECTED_BROKER_SETTING, "yahoo");
        this.activeBroker = new AtomicReference<>(isAvailable(configuredBroker) ? configuredBroker : "yahoo");
        logger.info("MarketDataClientProvider initialized with: {}", this.clients.keySet());
        logger.info("Active market data provider: {}", activeBroker.get());
    }

    /** Compatibility constructor for lightweight tests. */
    public MarketDataClientProvider(Map<String, MarketDataClient> clients) {
        this.clients = normalizeClientNames(clients);
        this.appSettingsService = null;
        this.fyersEnabled = true;
        this.activeBroker = new AtomicReference<>(this.clients.containsKey("yahoo") ? "yahoo" : FYERS);
    }

    private static Map<String, MarketDataClient> normalizeClientNames(Map<String, MarketDataClient> clients) {
        Map<String, MarketDataClient> normalized = new HashMap<>(clients);
        MarketDataClient fyersClient = normalized.get("fyersServiceClient");
        if (fyersClient != null) {
            normalized.putIfAbsent(FYERS, fyersClient);
        }
        return normalized;
    }

    private static Map<String, MarketDataClient> rateLimitClients(Map<String, MarketDataClient> clients,
                                                                   DataIngestionMetrics metrics,
                                                                   long yahooIntervalMillis,
                                                                   long upstoxRpm, long fyersRpm,
                                                                   long defaultRpm) {
        Map<String, MarketDataClient> limited = new HashMap<>();
        Map<MarketDataClient, MarketDataClient> wrappedByDelegate = new IdentityHashMap<>();
        clients.forEach((name, client) -> {
            if (client instanceof RateLimitedMarketDataClient) {
                limited.put(name, client);
                return;
            }
            MarketDataClient wrapped = wrappedByDelegate.computeIfAbsent(client, delegate -> {
                long rpm = switch (name.toLowerCase()) {
                    case "upstox" -> upstoxRpm;
                    case "fyers", "fyersserviceclient" -> fyersRpm;
                    default -> yahooIntervalMillis <= 0 ? defaultRpm
                        : Math.max(1L, 60_000L / yahooIntervalMillis);
                };
                return new RateLimitedMarketDataClient(delegate, rpm, metrics, name);
            });
            limited.put(name, wrapped);
        });
        return limited;
    }

    /**
     * Rehydrates the active broker from the persisted setting at startup. Without
     * this, a restart always resets to "yahoo" in memory while the DB still says
     * (e.g.) "fyers" - GET /settings then reports the wrong broker, and the
     * dashboard's next save writes "yahoo" back over the user's real selection.
     */
    @PostConstruct
    void restorePersistedBroker() {
        if (appSettingsService == null) {
            return;
        }
        String persisted = appSettingsService.get(SELECTED_BROKER_SETTING, null);
        if (persisted != null && !persisted.equals(activeBroker.get())) {
            setActiveBroker(persisted);
        }
    }

    private boolean isAvailable(String broker) {
        return broker != null && clients.containsKey(broker)
            && (fyersEnabled || !broker.toLowerCase().startsWith(FYERS));
    }

    public MarketDataClient getClient() {
        String broker = activeBroker.get();
        MarketDataClient client = clients.get(broker);
        if (client == null) {
            throw new IllegalStateException("""
                Market data provider '%s' is unavailable; authentication/configuration is
                required and no fallback provider is enabled
                """.formatted(broker));
        }
        return client;
    }

    public String getActiveBroker() {
        return activeBroker.get();
    }

    public void setActiveBroker(String broker) {
        if (!isAvailable(broker)) {
            logger.warn("Cannot set broker to '{}', available: {}, Fyers enabled: {}",
                broker, clients.keySet(), fyersEnabled);
            return;
        }
        logger.info("Switched data source to '{}'", broker);
        activeBroker.set(broker);
        if (appSettingsService != null) {
            appSettingsService.set(SELECTED_BROKER_SETTING, broker);
        }
    }
}
