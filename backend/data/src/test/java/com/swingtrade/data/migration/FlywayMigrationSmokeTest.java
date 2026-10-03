package com.swingtrade.data.migration;

import org.flywaydb.core.Flyway;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assumptions.assumeTrue;

class FlywayMigrationSmokeTest {

    @Test
    void migrationsApplyToFreshPostgresWhenConfigured() {
        String url = System.getenv("MIGRATION_TEST_JDBC_URL");
        assumeTrue(url != null && !url.isBlank(), "Set MIGRATION_TEST_JDBC_URL to run against disposable PostgreSQL");

        Flyway.configure()
            .dataSource(url, System.getenv().getOrDefault("MIGRATION_TEST_DB_USER", ""),
                System.getenv().getOrDefault("MIGRATION_TEST_DB_PASSWORD", ""))
            .load()
            .migrate();
    }
}
