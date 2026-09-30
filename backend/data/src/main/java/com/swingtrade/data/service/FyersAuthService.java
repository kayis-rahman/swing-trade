package com.swingtrade.data.service;

import tools.jackson.core.type.TypeReference;
import tools.jackson.databind.JsonNode;
import tools.jackson.databind.ObjectMapper;
import com.swingtrade.data.config.FyersConfig;
import com.tts.in.utilities.Utility;
import jakarta.annotation.PostConstruct;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.reactive.function.client.WebClient;
import org.springframework.web.reactive.function.client.WebClientResponseException;
import reactor.core.publisher.Mono;
import reactor.netty.http.client.HttpClient;

import io.github.resilience4j.bulkhead.Bulkhead;
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.reactor.bulkhead.operator.BulkheadOperator;
import io.github.resilience4j.reactor.circuitbreaker.operator.CircuitBreakerOperator;

import java.io.File;
import java.time.Duration;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Service for handling Fyers authentication, token generation, and refresh logic.
 * Not a Spring bean — Fyser SDK has broken internal dependencies.
 */
public class FyersAuthService {
    private static final Logger logger = LoggerFactory.getLogger(FyersAuthService.class);
    private static final String BASE_URL = "https://api-t1.fyers.in";
    private static final String VALIDATE_AUTHCODE_PATH = "/api/v3/validate-authcode";
    private static final String VALIDATE_REFRESH_PATH = "/api/v3/validate-refresh-token";
    private static final String LOGOUT_PATH = "/api/v3/logout";

    private final FyersConfig fyersConfig;
    private final WebClient webClient;
    private final ObjectMapper objectMapper;

    // In-memory token storage (for the current session)
    private final AtomicReference<String> accessTokenRef = new AtomicReference<>();
    private final AtomicReference<String> refreshTokenRef = new AtomicReference<>();
    private volatile boolean missingAuthLogged = false;

// Resilience4j fields
    private CircuitBreaker fyersAuthCircuitBreaker;
    private Bulkhead fyersAuthBulkhead;

    static final Duration READ_TIMEOUT = Duration.ofSeconds(15);

    @Autowired
    public FyersAuthService(FyersConfig fyersConfig, WebClient.Builder webClientBuilder) {
        this(fyersConfig, webClientBuilder, BASE_URL);
    }

    // Package-private constructor for testing with a custom base URL (MockWebServer)
    FyersAuthService(FyersConfig fyersConfig, WebClient.Builder webClientBuilder, String baseUrl) {
        this.fyersConfig = fyersConfig;
        this.webClient = webClientBuilder
                .baseUrl(baseUrl)
                .defaultHeader("Content-Type", "application/json")
                .clientConnector(new org.springframework.http.client.reactive.ReactorClientHttpConnector(
                    HttpClient.create().responseTimeout(READ_TIMEOUT)))
                .build();
        this.objectMapper = new ObjectMapper();
    }

    // Setter for Resilience4j injection (called by FyersResilienceConfig)
    public void setResilience4j(CircuitBreaker circuitBreaker, Bulkhead bulkhead) {
        this.fyersAuthCircuitBreaker = circuitBreaker;
        this.fyersAuthBulkhead = bulkhead;
    }

    @PostConstruct
    public void init() {
        // Priority 1: env var / config property
        String envToken = fyersConfig.getAccessToken();
        if (envToken != null && !envToken.isEmpty()) {
            accessTokenRef.set(envToken);
            String envRefresh = fyersConfig.getRefreshToken();
            if (envRefresh != null && !envRefresh.isEmpty()) refreshTokenRef.set(envRefresh);
            logger.info("Loaded Fyers tokens from environment/config");
            return;
        }
        // Priority 2: persisted token file
        loadTokensFromFile();
    }

    /**
     * Computes appIdHash = SHA-256("{appId}:{appSecret}") as hex string.
     * Uses SDK's Utility.GenerateAppHashID for correctness.
     */
    private String computeAppIdHash() {
        return Utility.GenerateAppHashID(fyersConfig.getClientId(), fyersConfig.getSecretKey());
    }

    /**
     * Generates the Fyers authorization URL for the user to log in.
     */
    public String getAuthorizationUrl() {
        return String.format(
            "https://api-t1.fyers.in/api/v3/generate-authcode?client_id=%s&redirect_uri=%s&response_type=code&state=swingtrade",
            fyersConfig.getClientId(),
            fyersConfig.getRedirectUrl()
        );
    }

    /**
     * Gets a valid access token. Attempts to refresh if the current one is missing/expired.
     */
    public String getAccessToken() {
        String token = accessTokenRef.get();
        if (token == null || token.isEmpty()) {
            logger.warn("No access token available. Attempting to refresh...");
            refreshToken();
        }
        return accessTokenRef.get();
    }

    /**
     * Exchanges an authorization code for access and refresh tokens.
     * Call this after the user completes the browser login.
     */
    public void exchangeAuthCodeForTokens(String authCode) {
        try {
            String body = objectMapper.writeValueAsString(Map.of(
                "grant_type", "authorization_code",
                "code", authCode,
                "appIdHash", computeAppIdHash()
            ));

            String response = postTokenRequest(VALIDATE_AUTHCODE_PATH, body);
            logger.info("Fyers validate-authcode raw response: {}", response);
            updateTokensFromResponse(response);
            logger.info("Successfully exchanged auth code for tokens.");
        } catch (Exception e) {
            logger.error("Failed to exchange auth code: {}", e.getMessage(), e);
        }
    }

    /**
     * Refreshes the access token using the stored refresh token.
     */
    public void refreshToken() {
        String refreshToken = refreshTokenRef.get();
        if (refreshToken == null || refreshToken.isEmpty()) {
            if (!missingAuthLogged) {
                logger.warn("No refresh token available. Fyers authentication not configured — data ingestion will be skipped. Configure FYERS_CLIENT_ID, FYERS_SECRET_KEY, and FYERS_PIN to enable.");
                missingAuthLogged = true;
            }
            return;
        }

        String pin = fyersConfig.getPin();
        if (pin == null || pin.isBlank()) {
            logger.error("fyers.pin required for token refresh — set FYERS_PIN. Skipping refresh request.");
            return;
        }

        try {
            String body = objectMapper.writeValueAsString(Map.of(
                "grant_type", "refresh_token",
                "refresh_token", refreshToken,
                "appIdHash", computeAppIdHash(),
                "pin", pin
            ));

            String response = postTokenRequest(VALIDATE_REFRESH_PATH, body);
            updateTokensFromResponse(response);
            logger.info("Successfully refreshed access token.");
        } catch (WebClientResponseException e) {
            if (e.getStatusCode().value() == 401) {
                logger.error("Refresh token expired or invalid. Please re-authenticate via browser.");
                refreshTokenRef.set(null); // Invalidate bad refresh token
            } else {
                logger.error("Failed to refresh token: {} {}", e.getStatusCode(), e.getResponseBodyAsString());
            }
        } catch (Exception e) {
            logger.error("Token refresh failed: {}", e.getMessage(), e);
        }
    }

    private String postTokenRequest(String path, String body) {
        Mono<String> mono = webClient.post()
            .uri(path)
            .header("Content-Type", "application/json")
            .header("Accept", "application/json")
            .bodyValue(body)
            .retrieve()
            .bodyToMono(String.class);
        if (fyersAuthCircuitBreaker != null) {
            mono = mono.transformDeferred(CircuitBreakerOperator.of(fyersAuthCircuitBreaker));
        }
        if (fyersAuthBulkhead != null) {
            mono = mono.transformDeferred(BulkheadOperator.of(fyersAuthBulkhead));
        }
        return mono
            .onErrorResume(io.github.resilience4j.circuitbreaker.CallNotPermittedException.class,
                e -> {
                    logger.warn("Circuit breaker open for fyers auth");
                    return Mono.empty();
                })
            .onErrorResume(io.github.resilience4j.bulkhead.BulkheadFullException.class,
                e -> {
                    logger.warn("Bulkhead full for fyers auth");
                    return Mono.empty();
                })
            .block();
    }

    /**
     * Logout — invalidates the current access token.
     */
    public void logout() {
        String token = accessTokenRef.get();
        if (token == null) return;
        try {
            String appId = fyersConfig.getClientId();
            webClient.delete()
                .uri(LOGOUT_PATH)
                .header("Authorization", appId + ":" + token)
                .retrieve()
                .bodyToMono(String.class)
                .block();
            clearTokens();
            logger.info("Fyers logout successful. Tokens cleared.");
        } catch (Exception e) {
            logger.error("Fyers logout failed: {}", e.getMessage());
        }
    }

    private void clearTokens() {
        accessTokenRef.set(null);
        refreshTokenRef.set(null);
        fyersConfig.setAccessToken(null);
        fyersConfig.setRefreshToken(null);
        String path = fyersConfig.getTokenStorePath();
        if (path != null && !path.isEmpty()) {
            File file = new File(path);
            if (file.exists() && !file.delete()) {
                logger.warn("Failed to delete persisted Fyers token file at {}", path);
            }
        }
    }

    private void updateTokensFromResponse(String response) throws Exception {
        JsonNode root = objectMapper.readTree(response);
        if (!"ok".equals(root.get("s").asText())) {
            throw new RuntimeException("Fyers token API error: " + root.get("message"));
        }

        String newAccess = root.get("access_token").asText();
        String newRefresh = root.has("refresh_token") ? root.get("refresh_token").asText() : refreshTokenRef.get();

        accessTokenRef.set(newAccess);
        refreshTokenRef.set(newRefresh);

        // Persist back to config
        fyersConfig.setAccessToken(newAccess);
        fyersConfig.setRefreshToken(newRefresh);
        // Persist to file
        persistTokens();
    }

    public String getClientId() { return fyersConfig.getClientId(); }

    /**
     * Checks the current session state without initiating network I/O.
     * Status endpoints poll this method, so an unauthenticated session must
     * not repeatedly attempt refresh and fill the logs with expected errors.
     */
    public boolean validateToken() {
        String token = accessTokenRef.get();
        return token != null && !token.isEmpty();
    }

    private void persistTokens() {
        String path = fyersConfig.getTokenStorePath();
        if (path == null || path.isEmpty()) return;
        try {
            File file = new File(path);
            file.getParentFile().mkdirs();
            objectMapper.writeValue(file, Map.of(
                "access_token", accessTokenRef.get(),
                "refresh_token", refreshTokenRef.get(),
                "stored_at", java.time.Instant.now().toString()
            ));
            logger.debug("Persisted Fyers tokens to {}", path);
        } catch (Exception e) {
            logger.warn("Failed to persist Fyers tokens to {}: {}", path, e.getMessage());
        }
    }

    private void loadTokensFromFile() {
        String path = fyersConfig.getTokenStorePath();
        if (path == null || path.isEmpty()) return;
        File file = new File(path);
        if (!file.exists()) {
            logger.debug("No Fyers token file found at {}", path);
            return;
        }
        try {
            Map<String, Object> data = objectMapper.readValue(file,
                new TypeReference<Map<String, Object>>() {});
            String access = (String) data.get("access_token");
            String refresh = (String) data.get("refresh_token");
            if (access != null && !access.isEmpty()) {
                accessTokenRef.set(access);
                fyersConfig.setAccessToken(access);
            }
            if (refresh != null && !refresh.isEmpty()) {
                refreshTokenRef.set(refresh);
                fyersConfig.setRefreshToken(refresh);
            }
            if (access != null) logger.info("Loaded Fyers tokens from file {}", path);
        } catch (Exception e) {
            logger.warn("Failed to load Fyers tokens from {}: {}", path, e.getMessage());
        }
    }
}
