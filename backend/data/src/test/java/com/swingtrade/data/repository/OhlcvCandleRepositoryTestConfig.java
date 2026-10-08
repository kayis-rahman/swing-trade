package com.swingtrade.data.repository;

import org.springframework.boot.SpringBootConfiguration;
import org.springframework.boot.autoconfigure.EnableAutoConfiguration;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;
import org.springframework.boot.persistence.autoconfigure.EntityScan;

/**
 * Minimal sliced configuration for repository contract tests: JPA repositories and
 * entities only, no web server, no component scan of the main sources. The test
 * profile supplies the H2 datasource (create-drop, Flyway disabled).
 */
@SpringBootConfiguration
@EnableAutoConfiguration
@EnableJpaRepositories(basePackages = "com.swingtrade.data.repository")
@EntityScan(basePackages = "com.swingtrade.data.entity")
class OhlcvCandleRepositoryTestConfig {
}
