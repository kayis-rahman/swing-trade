package com.swingtrade.api.controller;

import com.swingtrade.data.config.FyersConfig;
import com.swingtrade.data.service.FyersAuthService;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.reactive.function.client.WebClient;

import java.io.IOException;
import java.util.Map;
import java.util.Objects;

/**
 * Controller for Fyers authentication flow.
 * Instantiates FyersAuthService manually to avoid SDK class loading at startup.
 */
@RestController
@RequestMapping("/api/fyers")
public class FyersAuthController {

    private static final Logger logger = LoggerFactory.getLogger(FyersAuthController.class);

    private final FyersAuthService authService;
    private final FyersConfig fyersConfig;

    public FyersAuthController(FyersConfig fyersConfig, WebClient.Builder webClientBuilder) {
        this.fyersConfig = fyersConfig;
        this.authService = new FyersAuthService(fyersConfig, webClientBuilder);
    }

    /**
     * GET /api/fyers/login
     * Returns the Fyers authorization URL for the user to start the login process.
     */
    @GetMapping("/login")
    public ResponseEntity<Map<String, String>> getLoginUrl() {
        if (fyersConfig.getClientId() == null || fyersConfig.getClientId().isBlank()) {
            return ResponseEntity.badRequest().body(Map.of(
                "error", "FYERS_CLIENT_ID not configured",
                "code", "FYERS_NOT_CONFIGURED"
            ));
        }
        if (fyersConfig.getSecretKey() == null || fyersConfig.getSecretKey().isBlank()) {
            return ResponseEntity.badRequest().body(Map.of(
                "error", "FYERS_SECRET_KEY not configured",
                "code", "FYERS_NOT_CONFIGURED"
            ));
        }
        return ResponseEntity.ok(Map.of(
            "url", authService.getAuthorizationUrl(),
            "message", "Please visit the URL to authenticate with Fyers"
        ));
    }

    /**
     * GET /api/fyers/callback
     * Fyers OAuth2 callback. Exchanges auth code for tokens, sends postMessage to popup, then closes.
     * Frontend polls /api/fyers/status to detect successful auth — no token in URL.
     */
    @GetMapping("/callback")
    public void handleCallback(HttpServletRequest request, HttpServletResponse response) throws IOException {
        String params = request.getQueryString();
        logger.info("Fyers callback received, query={}", params);

        // Fyers v3 sends "auth_code" as param name
        String authCode = request.getParameter("auth_code");

        logger.info("Fyers callback, extracted authcode={}", authCode);
        if (authCode == null || authCode.isBlank()) {
            logger.warn("Fyers callback without authcode — sending error");
            sendCallbackHtml(response, false);
            return;
        }
        authService.exchangeAuthCodeForTokens(authCode);
        boolean success = authService.validateToken();
        logger.info("Fyers auth result: {}", success ? "success" : "failed");

        sendCallbackHtml(response, success);
    }

    private void sendCallbackHtml(HttpServletResponse response, boolean success) throws IOException {
        String status = success ? "success" : "error";
        String msg = success ? "Fyers connected!" : "Authentication failed.";
        String origin = fyersConfig.getFrontendUrl();
        response.setContentType("text/html;charset=UTF-8");
        response.getWriter().write(String.format(
            "<!DOCTYPE html><html><body>"
            + "<p>%s</p>"
            + "<script>"
            + "if(window.opener){"
            + "  window.opener.postMessage({type:'fyers_auth_%s'},'%s');"
            + "  setTimeout(()=>window.close(),500);"
            + "}else{"
            + "  window.location='%s/settings?auth=%s';"
            + "}"
            + "</script></body></html>",
            msg, status, origin, origin, status
        ));
    }

    /**
     * POST /api/fyers/auth
     * Manual token exchange for API clients.
     * Body: { "authCode": "..." }
     */
    @PostMapping("/auth")
    public ResponseEntity<Map<String, String>> authenticate(@RequestBody Map<String, String> payload) {
        String authCode = payload.get("authCode");
        if (authCode == null) {
            return ResponseEntity.badRequest().body(Map.of("error", "Missing authCode"));
        }

        authService.exchangeAuthCodeForTokens(authCode);
        boolean success = authService.validateToken();

        return ResponseEntity.ok(Map.of(
            "status", success ? "success" : "failed",
            "message", success ? "Tokens generated and stored" : "Token exchange failed"
        ));
    }

    /**
     * GET /api/fyers/status
     * Returns current token status.
     */
    @GetMapping("/status")
    public ResponseEntity<Map<String, Object>> getStatus() {
        // Map.of(...) throws NPE on any null value/key. clientId can be null when
        // FYERS_CLIENT_ID is not configured (e.g. not yet authenticated on stage),
        // so build the response with a null-safe fallback instead of crashing.
        return ResponseEntity.ok(Map.of(
            "connected", authService.validateToken(),
            "clientId", Objects.requireNonNullElse(authService.getClientId(), "not configured")
        ));
    }

    /**
     * POST /api/fyers/logout
     * Invalidates and clears the stored access/refresh tokens.
     */
    @PostMapping("/logout")
    public ResponseEntity<Map<String, Object>> logout() {
        authService.logout();
        return ResponseEntity.ok(Map.of("connected", authService.validateToken()));
    }
}
