package com.swingtrade.api.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.UrlBasedCorsConfigurationSource;
import org.springframework.web.filter.CorsFilter;

import java.util.List;

/**
 * CORS configuration for frontend integration.
 * Allows the Vue dashboard to make API requests to the backend.
 */
@Configuration
public class CorsConfig {

    @Bean
    public CorsFilter corsFilter() {
        CorsConfiguration config = new CorsConfiguration();
        
        // Allowed origins - adjust for production
        config.setAllowedOrigins(List.of(
            "http://localhost:3000",
            "http://localhost:3003",
            "http://localhost:3004",
            "http://localhost:3005",
            "http://127.0.0.1:3003",
            "http://127.0.0.1:3005",
            "http://localhost:8080",
            "http://piworm.local:8081",
            "http://piworm.local:8082",
            "http://127.0.0.1:8081",
            "http://127.0.0.1:8082"
        ));
        
        config.setAllowedMethods(List.of("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"));
        config.setAllowedHeaders(List.of(
            "Authorization",
            "X-API-Key",
            "Content-Type",
            "Accept",
            "X-Requested-With"
        ));
        config.setExposedHeaders(List.of("Authorization"));
        config.setAllowCredentials(true);
        config.setMaxAge(3600L); // Cache preflight for 1 hour

        UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
        source.registerCorsConfiguration("/api/**", config);
        
        return new CorsFilter(source);
    }
}
