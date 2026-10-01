package com.swingtrade.api.controller;

import com.swingtrade.api.dto.ApiResponse;
import com.swingtrade.broker.service.DiscordNotificationService;
import com.swingtrade.data.service.AppSettingsService;
import com.swingtrade.data.service.MarketDataClientProvider;
import com.swingtrade.llm.client.LlmClient;
import com.swingtrade.llm.client.PiAgentLlmClient;
import com.swingtrade.llm.config.LlmProperties;
import com.swingtrade.llm.service.LlamaCppServerManager;
import com.swingtrade.llm.service.LlmBackendSelector;
import com.swingtrade.llm.service.MlxServerManager;
import com.swingtrade.llm.service.LlmClientProvider;
import com.swingtrade.llm.service.LlmServerManager;
import com.swingtrade.llm.service.PiLlamaServerManager;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.lang.Nullable;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.util.UriComponentsBuilder;
import jakarta.annotation.PostConstruct;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.math.BigDecimal;
import java.time.Duration;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.regex.Pattern;

import tools.jackson.databind.ObjectMapper;

@RestController
@RequestMapping("/api")
public class SettingsController {

    private static final Logger logger = LoggerFactory.getLogger(SettingsController.class);
    private static final int TEST_INFERENCE_MAX_TOKENS = 512;
    // Reasoning-capable OpenAI-compatible models may spend 20-40 tokens before
    // emitting a one-word answer. Keep the connectivity probe above that budget.
    private static final int OPENAI_TEST_MAX_TOKENS = 128;
    private static final Duration OLLAMA_TEST_TIMEOUT = Duration.ofSeconds(120);
    private static final Pattern OK_PATTERN = Pattern.compile("(?i)\\bok\\b");
    private static final Set<String> SECRET_SETTING_KEYS = Set.of(
        "openai.api_key",
        "ollama.api_key",
        "gpuhub.api_key"
    );

    private final MarketDataClientProvider marketDataClientProvider;
    private final AppSettingsService appSettingsService;
    private final LlmProperties llmProperties;
    private final DiscordNotificationService discordNotificationService;
    private final LlmBackendSelector selector;
    private final LlmClientProvider llmClientProvider;
    private final PiAgentLlmClient piAgentClient;
    private final LlamaCppServerManager localServerManager;
    private final PiLlamaServerManager piServerManager;
    private final MlxServerManager mlxServerManager;

    @Autowired
    public SettingsController(MarketDataClientProvider marketDataClientProvider,
                              AppSettingsService appSettingsService,
                              LlmProperties llmProperties,
                              DiscordNotificationService discordNotificationService,
                              LlmBackendSelector selector,
                              LlmClientProvider llmClientProvider,
                              @Nullable PiAgentLlmClient piAgentClient,
                              LlamaCppServerManager localServerManager,
                              PiLlamaServerManager piServerManager,
                              MlxServerManager mlxServerManager) {
        this.marketDataClientProvider = marketDataClientProvider;
        this.appSettingsService = appSettingsService;
        this.llmProperties = llmProperties;
        this.discordNotificationService = discordNotificationService;
        this.selector = selector;
        this.llmClientProvider = llmClientProvider;
        this.piAgentClient = piAgentClient;
        this.localServerManager = localServerManager;
        this.piServerManager = piServerManager;
        this.mlxServerManager = mlxServerManager;
    }

    @PostConstruct
    void loadPiAgentSettings() {
        configurePiAgentFromSettings();
    }

    @GetMapping("/settings")
    public ResponseEntity<ApiResponse<Map<String, String>>> getSettings() {
        Map<String, String> settings = Map.of(
            "selectedBroker", marketDataClientProvider.getActiveBroker()
        );
        return ResponseEntity.ok(ApiResponse.ok(settings));
    }

    @PostMapping("/settings/broker")
    public ResponseEntity<ApiResponse<Map<String, String>>> setBroker(
            @RequestBody Map<String, String> body
    ) {
        String broker = body.get("broker");
        if (broker == null || broker.isBlank()) {
            return ResponseEntity.badRequest().body(ApiResponse.error("broker is required"));
        }
        marketDataClientProvider.setActiveBroker(broker);
        Map<String, String> settings = Map.of("selectedBroker", broker);
        return ResponseEntity.ok(ApiResponse.ok(settings));
    }

    @GetMapping("/settings/llm")
    public ResponseEntity<ApiResponse<Map<String, String>>> getLlmSettings() {
        Map<String, String> settings = new LinkedHashMap<>();
        settings.put("llm.base_url", appSettingsService.get(
            "llm.base_url", llmProperties.getBaseUrl().toString()));
        settings.put("llm.backend", appSettingsService.get(
            "llm.backend", llmProperties.getBackend()));
        settings.put("openai.base_url", appSettingsService.get(
            "openai.base_url", llmProperties.getProviders().getOpenai().getBaseUrl().toString()));
        settings.put("openai.model", appSettingsService.get(
            "openai.model", llmProperties.getProviders().getOpenai().getModel()));
        settings.put("ollama.base_url", appSettingsService.get(
            "ollama.base_url", llmProperties.getProviders().getOllama().getBaseUrl().toString()));
        settings.put("ollama.model", appSettingsService.get(
            "ollama.model", llmProperties.getProviders().getOllama().getModel()));
        settings.put("llamacpp.model", appSettingsService.get(
            "llamacpp.model", llmProperties.getLlamaCpp().getModel()));
        settings.put("laya.base_url", appSettingsService.get(
            "laya.base_url", llmProperties.getProviders().getLaya().getBaseUrl().toString()));
        settings.put("laya.model", appSettingsService.get(
            "laya.model", llmProperties.getProviders().getLaya().getModel()));
        settings.put("mlx.server.url", appSettingsService.get(
            "mlx.server.url", llmProperties.getProviders().getMlx().getBaseUrl().toString()));
        settings.put("mlx.model", appSettingsService.get(
            "mlx.model", llmProperties.getProviders().getMlx().getModel()));
        settings.put("pi-agent.provider", appSettingsService.get(
            "pi-agent.provider", "openai-codex"));
        settings.put("pi-agent.model", appSettingsService.get(
            "pi-agent.model", "gpt-5.6-luna"));
        settings.put("llm.pdf.base_url", appSettingsService.get(
            "llm.pdf.base_url",
            llmProperties.getPdf().getBaseUrl() != null ? llmProperties.getPdf().getBaseUrl().toString() : ""));
        settings.put("llm.pdf.model", appSettingsService.get(
            "llm.pdf.model", llmProperties.getPdf().getModel()));
        addSecretConfiguredFlag(settings, "openai.api_key");
        addSecretConfiguredFlag(settings, "ollama.api_key");
        addSecretConfiguredFlag(settings, "gpuhub.api_key");
        return ResponseEntity.ok(ApiResponse.ok(settings));
    }

    @PutMapping("/settings/llm")
    public ResponseEntity<ApiResponse<Map<String, String>>> setLlmSettings(
            @RequestBody Map<String, String> body) {
        String oldBackend = appSettingsService.get("llm.backend", llmProperties.getBackend());
        String oldLlamaCppModel = appSettingsService.get(
            "llamacpp.model", llmProperties.getLlamaCpp().getModel());
        String oldMlxModel = appSettingsService.get(
            "mlx.model", llmProperties.getProviders().getMlx().getModel());
        body.forEach((key, value) -> appSettingsService.set(key, value));
        configurePiAgentFromSettings();

        // Handle server restart when backend or model changes
        String newBackend = body.getOrDefault("llm.backend", oldBackend);
        boolean backendChanged = !oldBackend.equals(newBackend);
        boolean modelChanged = body.containsKey("llamacpp.model")
                && !body.get("llamacpp.model").equals(oldLlamaCppModel);
        boolean mlxModelChanged = body.containsKey("mlx.model")
                && !body.get("mlx.model").equals(oldMlxModel);

        if (backendChanged || modelChanged || mlxModelChanged) {
            try {
                var backend = selector.resolve();
                LlmServerManager manager = switch (backend) {
                    case LOCAL -> localServerManager;
                    case PI_SSH -> piServerManager;
                    case OPENAI, OLLAMA, LAYA, PI_AGENT -> null; // no server to manage
                    case MLX -> mlxServerManager;
                };
                if (manager != null && manager.isRunning()) {
                    manager.restart();
                }
            } catch (Exception e) {
                logger.warn("LLM server restart failed: {}", e.getMessage());
            }
        }
        return ResponseEntity.ok(ApiResponse.ok(safeSettingsResponse(body)));
    }

    private void configurePiAgentFromSettings() {
        if (piAgentClient != null) {
            piAgentClient.configure(
                appSettingsService.get("pi-agent.provider", "openai-codex"),
                appSettingsService.get("pi-agent.model", "gpt-5.6-luna"));
        }
    }

    @GetMapping("/settings/gpuhub")
    public ResponseEntity<ApiResponse<Map<String, String>>> getGpuHubSettings() {
        Map<String, String> settings = new LinkedHashMap<>();
        addSecretConfiguredFlag(settings, "gpuhub.api_key");
        return ResponseEntity.ok(ApiResponse.ok(settings));
    }

    @PutMapping("/settings/gpuhub")
    public ResponseEntity<ApiResponse<Map<String, String>>> setGpuHubSettings(
            @RequestBody Map<String, String> body) {
        body.forEach((key, value) -> appSettingsService.set(key, value));
        return ResponseEntity.ok(ApiResponse.ok(safeSettingsResponse(body)));
    }

    private Map<String, String> safeSettingsResponse(Map<String, String> settings) {
        Map<String, String> safeSettings = new LinkedHashMap<>();
        settings.forEach((key, value) -> {
            if (!SECRET_SETTING_KEYS.contains(key)) {
                safeSettings.put(key, value);
            }
        });
        SECRET_SETTING_KEYS.stream()
            .filter(settings::containsKey)
            .forEach(key -> addSecretConfiguredFlag(safeSettings, key));
        return safeSettings;
    }

    private void addSecretConfiguredFlag(Map<String, String> settings, String secretKey) {
        boolean configured = appSettingsService.get(secretKey)
            .filter(value -> !value.isBlank())
            .isPresent();
        settings.put("%s.configured".formatted(secretKey), Boolean.toString(configured));
    }

    @GetMapping("/settings/discord")
    public ResponseEntity<ApiResponse<Map<String, String>>> getDiscordSettings() {
        Map<String, String> settings = Map.of(
            "discord.webhook.url", appSettingsService.get("discord.webhook.url", ""),
            "discord.webhook.enabled", appSettingsService.get("discord.webhook.enabled", "false")
        );
        return ResponseEntity.ok(ApiResponse.ok(settings));
    }

    @PutMapping("/settings/discord")
    public ResponseEntity<ApiResponse<Map<String, String>>> setDiscordSettings(
            @RequestBody Map<String, String> body) {
        body.forEach((key, value) -> appSettingsService.set(key, value));
        return ResponseEntity.ok(ApiResponse.ok(body));
    }

    @PostMapping("/settings/test/discord")
    public ResponseEntity<ApiResponse<Map<String, Boolean>>> testDiscord() {
        boolean success = discordNotificationService.sendMessage("SwingTrade: webhook test successful");
        return ResponseEntity.ok(ApiResponse.ok(Map.of("success", success)));
    }

    @PostMapping("/settings/test/pi")
    public ResponseEntity<ApiResponse<Map<String, Object>>> testPiConnection() {
        Map<String, Object> result = new LinkedHashMap<>();
        try {
            logger.info("Testing Pi SSH connection and lazy start...");
            piServerManager.ensureRunning();
            boolean serverRunning = piServerManager.isRunning();
            if (!serverRunning) {
                result.put("success", false);
                result.put("message", "SSH connected but llama-server failed to start");
                return ResponseEntity.ok(ApiResponse.ok(result));
            }

            // Use the same OkHttp transport as the active Spring AI LLM client.
            // JDK HttpClient requests to llama.cpp can be rejected with an empty 400.
            boolean inferenceOk = piServerManager.testInferenceConnection();
            result.put("success", inferenceOk);
            result.put("message", inferenceOk
                ? "Pi SSH connection successful, llama-server started and responded to inference"
                : "Pi connected and server started, but inference failed");

            if (!inferenceOk) {
                piServerManager.stop();
            }
            return ResponseEntity.ok(ApiResponse.ok(result));
        } catch (Exception e) {
            logger.warn("Pi SSH test failed: {}", e.getMessage());
            result.put("success", false);
            result.put("message", "SSH connection failed");
            return ResponseEntity.ok(ApiResponse.ok(result));
        }
    }

    // Reasoning-capable local models spend part of the token budget on internal reasoning
    // before emitting content. This budget leaves enough headroom for a short health-check reply.
    private boolean testInference(URI baseUrl, String model) {
        return testInference(baseUrl, model, "", Duration.ofSeconds(30));
    }

    private boolean testInference(URI baseUrl, String model, String apiKey) {
        return testInference(baseUrl, model, apiKey, Duration.ofSeconds(30));
    }

    private boolean testInference(URI baseUrl, String model, String apiKey, Duration timeout) {
        try {
            String payload = """
                {"model":"%s","messages":[{"role":"user","content":"Reply with exactly: OK"}],"max_tokens":%d,"temperature":0.2}"""
                .formatted(model, TEST_INFERENCE_MAX_TOKENS);
            // llama.cpp's OpenAI-compatible listener is HTTP/1.1-only; pin the
            // JDK client to avoid protocol negotiation that it rejects with 400.
            HttpClient client = HttpClient.newBuilder()
                .version(HttpClient.Version.HTTP_1_1)
                .build();
            HttpRequest.Builder requestBuilder = HttpRequest.newBuilder()
                .uri(chatCompletionsUri(baseUrl))
                .header("Content-Type", "application/json")
                .timeout(timeout);
            if (apiKey != null && !apiKey.isBlank()) {
                requestBuilder.header("Authorization", "Bearer " + apiKey);
            }
            HttpRequest request = requestBuilder
                .POST(HttpRequest.BodyPublishers.ofString(payload))
                .build();
            HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
            if (response.statusCode() != 200) {
                logger.warn("Inference test at {} using model {} returned HTTP {}: {}",
                    baseUrl, model, response.statusCode(), response.body());
                return false;
            }
            ObjectMapper mapper = new ObjectMapper();
            var node = mapper.readTree(response.body());
            var content = node.path("choices").path(0).path("message").path("content").asText(null);
            return content != null && !content.isBlank() && OK_PATTERN.matcher(content).find();
        } catch (Exception e) {
            logger.warn("Inference test failed for {} using model {}: {}", baseUrl, model, e.getMessage());
            return false;
        }
    }

    URI ollamaInferenceBaseUrl() {
        URI configuredBaseUrl = llmProperties.getProviders().getOllama().getBaseUrl();
        try {
            URI runtimeBaseUrl = URI.create(appSettingsService.get(
                "ollama.base_url", configuredBaseUrl.toString()));
            if (hasSameOrigin(runtimeBaseUrl, configuredBaseUrl)) {
                return runtimeBaseUrl;
            }
            logger.warn("Ignoring Ollama inference URL with unapproved origin");
        } catch (IllegalArgumentException ignored) {
            logger.warn("Ignoring invalid Ollama inference URL");
        }
        return configuredBaseUrl;
    }

    private boolean hasSameOrigin(URI candidate, URI configured) {
        return Objects.equals(candidate.getScheme(), configured.getScheme())
            && Objects.equals(candidate.getHost(), configured.getHost())
            && candidate.getPort() == configured.getPort()
            && candidate.getUserInfo() == null
            && candidate.getQuery() == null
            && candidate.getFragment() == null;
    }

    URI mlxInferenceBaseUrl(URI configuredBaseUrl) {
        try {
            return URI.create(appSettingsService.get(
                "mlx.server.url", configuredBaseUrl.toString()));
        } catch (IllegalArgumentException ignored) {
            logger.warn("Ignoring invalid MLX inference URL");
        }
        return configuredBaseUrl;
    }

    static URI chatCompletionsUri(URI baseUrl) {
        return UriComponentsBuilder.fromUri(baseUrl)
            .pathSegment("chat", "completions")
            .build()
            .toUri();
    }

    @PostMapping("/settings/test/openai")
    public ResponseEntity<ApiResponse<Map<String, Object>>> testOpenAiConnection() {
        Map<String, Object> result = new LinkedHashMap<>();
        try {
            LlmClient client = llmClientProvider.getClient();
            if (client == null) {
                result.put("success", false);
                result.put("message", "No LLM client configured");
                return ResponseEntity.ok(ApiResponse.ok(result));
            }
            // Send a minimal test prompt to verify connectivity
            var messages = List.of(
                Map.of("role", "system", "content", "Respond with a single word."),
                Map.of("role", "user", "content", "Respond with a single word.")
            );
            LlmServerManager manager = switch (selector.resolve()) {
                case LOCAL -> localServerManager;
                case PI_SSH -> piServerManager;
                case MLX -> mlxServerManager;
                case OPENAI, OLLAMA, LAYA, PI_AGENT -> null;
            };
            if (manager != null) {
                manager.ensureRunning();
                manager.beginRequest();
            }
            String response;
            try {
                response = client.generateChatCompletion(messages, OPENAI_TEST_MAX_TOKENS, 0.0)
                    .block(Duration.ofSeconds(30));
            } finally {
                if (manager != null) manager.endRequest();
            }
            boolean ok = response != null && !response.isBlank();
            result.put("success", ok);
            result.put("message", ok ? "OpenAI-compatible LLM responded successfully" : "LLM returned empty response");
            return ResponseEntity.ok(ApiResponse.ok(result));
        } catch (Exception e) {
            logger.warn("OpenAI test failed: {}", e.getMessage());
            result.put("success", false);
            result.put("message", "Connection failed");
            return ResponseEntity.ok(ApiResponse.ok(result));
        }
    }

    @PostMapping("/settings/test/ollama")
    public ResponseEntity<ApiResponse<Map<String, Object>>> testOllamaConnection() {
        Map<String, Object> result = new LinkedHashMap<>();
        try {
            // Test directly against Ollama settings without changing the active backend.
            LlmProperties.Provider ollamaDefaults = llmProperties.getProviders().getOllama();
            URI baseUrl = ollamaInferenceBaseUrl();
            String model = appSettingsService.get("ollama.model", ollamaDefaults.getModel());
            String apiKey = appSettingsService.get("ollama.api_key", "");

            boolean inferenceOk = testInference(baseUrl, model, apiKey, OLLAMA_TEST_TIMEOUT);
            result.put("success", inferenceOk);
            result.put("message", inferenceOk
                ? "Ollama responded successfully"
                : "Ollama connection failed or returned an unexpected response");
            return ResponseEntity.ok(ApiResponse.ok(result));
        } catch (Exception e) {
            logger.warn("Ollama test failed: {}", e.getMessage());
            result.put("success", false);
            result.put("message", "Connection failed");
            return ResponseEntity.ok(ApiResponse.ok(result));
        }
    }

    @PostMapping("/settings/test/pi-agent")
    public ResponseEntity<ApiResponse<Map<String, Object>>> testPiAgentConnection() {
        Map<String, Object> result = new LinkedHashMap<>();
        try {
            if (piAgentClient == null) {
                throw new IllegalStateException("Pi agent client is not configured");
            }
            var messages = List.of(
                Map.of("role", "system", "content", "Respond with a single word."),
                Map.of("role", "user", "content", "Respond with a single word.")
            );
            String response = piAgentClient
                .generateChatCompletion(messages, 16, 0.0)
                .block(Duration.ofSeconds(190));
            boolean ok = response != null && !response.isBlank();
            result.put("success", ok);
            result.put("message", ok ? "Pi agent responded successfully" : "Pi agent returned an empty response");
        } catch (Exception e) {
            logger.warn("Pi agent test failed: {}", e.getMessage());
            result.put("success", false);
            result.put("message", "Pi agent unavailable: " + e.getMessage());
        }
        return ResponseEntity.ok(ApiResponse.ok(result));
    }

    // -----------------------------------------------------------------------
    // Pi SSH LLM Server Lifecycle
    // -----------------------------------------------------------------------

    @PostMapping("/settings/pi/start")
    public ResponseEntity<ApiResponse<Map<String, Object>>> startPiServer() {
        Map<String, Object> result = new LinkedHashMap<>();
        try {
            logger.info("Starting Pi llama-server via SSH...");
            piServerManager.ensureRunning();
            boolean running = piServerManager.isRunning();
            result.put("success", running);
            result.put("running", running);
            result.put("message", running
                ? "Pi llama-server started and healthy"
                : "Pi llama-server failed to start");
            return ResponseEntity.ok(ApiResponse.ok(result));
        } catch (Exception e) {
            logger.warn("Pi start failed: {}", e.getMessage());
            result.put("success", false);
            result.put("running", false);
            result.put("message", "Failed to start");
            return ResponseEntity.ok(ApiResponse.ok(result));
        }
    }

    @PostMapping("/settings/pi/stop")
    public ResponseEntity<ApiResponse<Map<String, Object>>> stopPiServer() {
        Map<String, Object> result = new LinkedHashMap<>();
        try {
            logger.info("Stopping Pi llama-server via SSH...");
            piServerManager.stop();
            boolean running = piServerManager.isRunning();
            boolean stopped = !running;
            try {
                Map<String, Object> lifecycle = piServerManager.lifecycleStatus();
                Object lifecycleState = lifecycle == null ? null : lifecycle.get("state");
                stopped = lifecycleState == null
                    ? true
                    : (!running && "STOPPED".equals(lifecycleState));
            } catch (Exception diagnosticFailure) {
                logger.debug("Pi lifecycle status unavailable after stop: {}", diagnosticFailure.getMessage());
            }
            result.put("success", stopped);
            result.put("running", stopped ? false : running);
            result.put("message", stopped ? "Pi llama-server stopped" : "Failed to confirm Pi llama-server stopped");
            return ResponseEntity.ok(ApiResponse.ok(result));
        } catch (Exception e) {
            logger.warn("Pi stop failed: {}", e.getMessage());
            result.put("success", false);
            result.put("running", true);
            result.put("message", "Failed to stop");
            return ResponseEntity.ok(ApiResponse.ok(result));
        }
    }

    @GetMapping("/settings/pi/status")
    public ResponseEntity<ApiResponse<Map<String, Object>>> getPiServerStatus() {
        Map<String, Object> result = new LinkedHashMap<>();
        // Preserve the existing response while exposing lazy-start/idle-stop diagnostics.
        Map<String, Object> lifecycle = piServerManager.lifecycleStatus();
        if (lifecycle != null) result.putAll(lifecycle);
        boolean running = piServerManager.isRunning();
        result.put("running", running);
        result.put("success", true);
        result.put("message", running
            ? "Pi llama-server is running"
            : "Pi llama-server is not running");
        return ResponseEntity.ok(ApiResponse.ok(result));
    }

    // -----------------------------------------------------------------------
    // MLX Server Lifecycle
    // -----------------------------------------------------------------------

    @PostMapping("/settings/mlx/start")
    public ResponseEntity<ApiResponse<Map<String, Object>>> startMlxServer() {
        Map<String, Object> result = new LinkedHashMap<>();
        try {
            logger.info("Starting MLX server...");
            mlxServerManager.ensureRunning();
            boolean running = mlxServerManager.isRunning();
            result.put("success", running);
            result.put("running", running);
            result.put("message", running
                ? "MLX server started and healthy"
                : "MLX server failed to start");
            return ResponseEntity.ok(ApiResponse.ok(result));
        } catch (Exception e) {
            logger.warn("MLX start failed: {}", e.getMessage());
            result.put("success", false);
            result.put("running", false);
            result.put("message", "Failed to start: " + e.getMessage());
            return ResponseEntity.ok(ApiResponse.ok(result));
        }
    }

    @PostMapping("/settings/mlx/stop")
    public ResponseEntity<ApiResponse<Map<String, Object>>> stopMlxServer() {
        Map<String, Object> result = new LinkedHashMap<>();
        try {
            logger.info("Stopping MLX server...");
            mlxServerManager.stop();
            boolean running = mlxServerManager.isRunning();
            result.put("success", !running);
            result.put("running", running);
            result.put("message", running
                ? "MLX server remains running"
                : "MLX server stopped");
            return ResponseEntity.ok(ApiResponse.ok(result));
        } catch (Exception e) {
            logger.warn("MLX stop failed: {}", e.getMessage());
            result.put("success", false);
            result.put("running", mlxServerManager.isRunning());
            result.put("message", "Failed to stop: " + e.getMessage());
            return ResponseEntity.ok(ApiResponse.ok(result));
        }
    }

    @GetMapping("/settings/mlx/status")
    public ResponseEntity<ApiResponse<Map<String, Object>>> getMlxServerStatus() {
        Map<String, Object> result = new LinkedHashMap<>();
        boolean running = mlxServerManager.isRunning();
        result.put("running", running);
        result.put("success", true);
        result.put("message", running
            ? "MLX server is running"
            : "MLX server is not running");
        return ResponseEntity.ok(ApiResponse.ok(result));
    }

    @PostMapping("/settings/test/mlx")
    public ResponseEntity<ApiResponse<Map<String, Object>>> testMlxConnection() {
        Map<String, Object> result = new LinkedHashMap<>();
        try {
            logger.info("Testing MLX connection and lazy start...");
            mlxServerManager.ensureRunning();
            boolean serverRunning = mlxServerManager.isRunning();
            if (!serverRunning) {
                result.put("success", false);
                result.put("message", "MLX server failed to start");
                return ResponseEntity.ok(ApiResponse.ok(result));
            }

            // Test actual LLM inference with a temporary client — does NOT affect the active backend.
            LlmProperties.Provider mlxDefaults = llmProperties.getProviders().getMlx();
            URI baseUrl = mlxInferenceBaseUrl(mlxDefaults.getBaseUrl());
            String model = appSettingsService.get("mlx.model", mlxDefaults.getModel());
            boolean inferenceOk = testInference(baseUrl, model, "", OLLAMA_TEST_TIMEOUT);
            result.put("success", inferenceOk);
            result.put("message", inferenceOk
                ? "MLX connection successful, server started and responded to inference"
                : "MLX connected and server started, but inference failed");

            if (!inferenceOk) {
                mlxServerManager.stop();
            }
            return ResponseEntity.ok(ApiResponse.ok(result));
        } catch (Exception e) {
            logger.warn("MLX test failed: {}", e.getMessage());
            result.put("success", false);
            result.put("message", "Connection failed: " + e.getMessage());
            return ResponseEntity.ok(ApiResponse.ok(result));
        }
    }

    // -----------------------------------------------------------------------
    // Trading Configuration
    // -----------------------------------------------------------------------

    @GetMapping("/settings/trading")
    public ResponseEntity<ApiResponse<Map<String, String>>> getTradingSettings() {
        Map<String, String> settings = new LinkedHashMap<>();
        settings.put("trading.mode", appSettingsService.get("trading.mode", "paper"));
        settings.put("trading.max_position_size", appSettingsService.get("trading.max_position_size", "10"));
        settings.put("trading.stop_loss", appSettingsService.get("trading.stop_loss", "5"));
        settings.put("trading.take_profit", appSettingsService.get("trading.take_profit", "15"));
        settings.put("trading.allocation_per_position", appSettingsService.get("trading.allocation_per_position", "100000"));
        settings.put("trading.initial_capital", appSettingsService.get(
            "trading.initial_capital", "500000"));
        return ResponseEntity.ok(ApiResponse.ok(settings));
    }

    @PutMapping("/settings/trading")
    public ResponseEntity<ApiResponse<Map<String, String>>> setTradingSettings(
            @RequestBody Map<String, String> body) {
        try {
            validateInitialCapital(body.get("trading.initial_capital"));
        } catch (IllegalArgumentException e) {
            return ResponseEntity.badRequest().body(ApiResponse.error(e.getMessage()));
        }
        body.forEach((key, value) -> appSettingsService.set(key, value));
        return ResponseEntity.ok(ApiResponse.ok(body));
    }

    private void validateInitialCapital(String raw) {
        if (raw == null) return;
        try {
            if (new BigDecimal(raw).signum() <= 0) throw new NumberFormatException();
        } catch (NumberFormatException e) {
            throw new IllegalArgumentException("Initial capital must be greater than zero");
        }
    }

    @GetMapping("/settings/scanning")
    public ResponseEntity<ApiResponse<Map<String, String>>> getScanningSettings() {
        return ResponseEntity.ok(ApiResponse.ok(Map.of(
            "candidate-scan.max-concurrent",
            appSettingsService.get("candidate-scan.max-concurrent", "3"))));
    }

    @PutMapping("/settings/scanning")
    public ResponseEntity<ApiResponse<Map<String, String>>> setScanningSettings(
            @RequestBody Map<String, String> body) {
        String raw = body.get("candidate-scan.max-concurrent");
        try {
            int workers = Integer.parseInt(raw);
            if (workers < 1 || workers > 12) throw new NumberFormatException();
        } catch (NumberFormatException | NullPointerException e) {
            return ResponseEntity.badRequest().body(ApiResponse.error(
                "Candidate scan workers must be between 1 and 12"));
        }
        appSettingsService.set("candidate-scan.max-concurrent", raw);
        return ResponseEntity.ok(ApiResponse.ok(Map.of("candidate-scan.max-concurrent", raw)));
    }

    // -----------------------------------------------------------------------
    // Unified Save — persist all settings in one call
    // -----------------------------------------------------------------------

    @PostMapping("/settings/save")
    public ResponseEntity<ApiResponse<Map<String, String>>> saveAllSettings(
            @RequestBody Map<String, Object> body) {
        // Accept nested objects: { broker, llm, discord, trading, scanning, gpuhub }
        if (body.containsKey("broker")) {
            String broker = String.valueOf(body.get("broker"));
            if (broker != null && !broker.isBlank() && !"null".equals(broker)) {
                marketDataClientProvider.setActiveBroker(broker);
                appSettingsService.set("selectedBroker", broker);
            }
        }
        if (body.containsKey("llm") && body.get("llm") instanceof Map<?, ?>) {
            ((Map<?, ?>) body.get("llm")).forEach((key, value) -> appSettingsService.set(String.valueOf(key), String.valueOf(value)));
        }
        if (body.containsKey("openai") && body.get("openai") instanceof Map<?, ?>) {
            ((Map<?, ?>) body.get("openai")).forEach((key, value) -> appSettingsService.set(String.valueOf(key), String.valueOf(value)));
        }
        if (body.containsKey("discord") && body.get("discord") instanceof Map<?, ?>) {
            ((Map<?, ?>) body.get("discord")).forEach((key, value) -> appSettingsService.set(String.valueOf(key), String.valueOf(value)));
        }
        if (body.containsKey("trading") && body.get("trading") instanceof Map<?, ?>) {
            Map<?, ?> trading = (Map<?, ?>) body.get("trading");
            Object initialCapital = trading.get("trading.initial_capital");
            try {
                validateInitialCapital(initialCapital == null ? null : String.valueOf(initialCapital));
            } catch (IllegalArgumentException e) {
                return ResponseEntity.badRequest().body(ApiResponse.error(e.getMessage()));
            }
            trading.forEach((key, value) -> appSettingsService.set(String.valueOf(key), String.valueOf(value)));
        }
        if (body.containsKey("scanning") && body.get("scanning") instanceof Map<?, ?>) {
            ((Map<?, ?>) body.get("scanning")).forEach((key, value) -> {
                if ("candidate-scan.max-concurrent".equals(String.valueOf(key))) {
                    try {
                        int workers = Integer.parseInt(String.valueOf(value));
                        if (workers >= 1 && workers <= 12) {
                            appSettingsService.set(String.valueOf(key), String.valueOf(value));
                        }
                    } catch (NumberFormatException ignored) {
                        logger.warn("Ignoring invalid candidate scan worker count: {}", value);
                    }
                }
            });
        }

        // Return consolidated settings
        Map<String, String> result = new LinkedHashMap<>();
        result.put("selectedBroker", marketDataClientProvider.getActiveBroker());
        result.putAll(getLlmSettings().getBody().data());
        result.putAll(getGpuHubSettings().getBody().data());
        result.putAll(getDiscordSettings().getBody().data());
        result.putAll(getTradingSettings().getBody().data());
        result.putAll(getScanningSettings().getBody().data());
        return ResponseEntity.ok(ApiResponse.ok(result));
    }
}
