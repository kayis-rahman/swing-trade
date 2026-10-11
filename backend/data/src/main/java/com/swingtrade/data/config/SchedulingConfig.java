package com.swingtrade.data.config;

import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.EnableScheduling;

@Configuration
@EnableScheduling
@ConditionalOnProperty(prefix = "app", name = "scheduling.enabled", havingValue = "true", matchIfMissing = true)
public class SchedulingConfig {
    // This configuration enables Spring's scheduling capabilities
}
