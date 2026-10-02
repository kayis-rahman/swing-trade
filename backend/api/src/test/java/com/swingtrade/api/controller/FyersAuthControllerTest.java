package com.swingtrade.api.controller;

import com.swingtrade.data.config.FyersConfig;
import org.junit.jupiter.api.Test;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.reactive.function.client.WebClient;

import static org.hamcrest.Matchers.containsString;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

/**
 * Guards the contract the dashboard relies on to distinguish
 * "Fyers is not configured on this environment" from a genuine
 * authentication failure (staging has no Fyers credentials).
 */
class FyersAuthControllerTest {

    private static MockMvc mvcWith(FyersConfig config) {
        WebClient.Builder builder = WebClient.builder();
        return MockMvcBuilders.standaloneSetup(new FyersAuthController(config, builder)).build();
    }

    private static FyersConfig configWith(String clientId, String secretKey) {
        FyersConfig config = new FyersConfig();
        config.setClientId(clientId);
        config.setSecretKey(secretKey);
        return config;
    }

    @Test
    void loginReturnsNotConfiguredCodeWhenClientIdMissing() throws Exception {
        mvcWith(configWith(null, null))
            .perform(get("/api/fyers/login"))
            .andExpect(status().isBadRequest())
            .andExpect(jsonPath("$.code").value("FYERS_NOT_CONFIGURED"))
            .andExpect(jsonPath("$.error").value("FYERS_CLIENT_ID not configured"));
    }

    @Test
    void loginReturnsNotConfiguredCodeWhenSecretKeyMissing() throws Exception {
        mvcWith(configWith("client-id", "   "))
            .perform(get("/api/fyers/login"))
            .andExpect(status().isBadRequest())
            .andExpect(jsonPath("$.code").value("FYERS_NOT_CONFIGURED"))
            .andExpect(jsonPath("$.error").value("FYERS_SECRET_KEY not configured"));
    }

    @Test
    void loginReturnsUrlWhenConfigured() throws Exception {
        mvcWith(configWith("client-id", "secret"))
            .perform(get("/api/fyers/login"))
            .andExpect(status().isOk())
            .andExpect(jsonPath("$.url").value(containsString("client_id=client-id")));
    }
}
