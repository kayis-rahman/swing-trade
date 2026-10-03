plugins {
    java
    id("io.spring.dependency-management")
}

repositories {
    mavenLocal()
    mavenCentral()
}

dependencies {
    implementation(project(":core"))

    implementation("org.springframework.boot:spring-boot-starter")
    implementation("org.springframework.boot:spring-boot-starter-webflux")
    implementation("org.springframework.boot:spring-boot-starter-data-jpa")
    implementation("org.springframework.boot:spring-boot-starter-validation")

    implementation("com.fasterxml.jackson.core:jackson-annotations")
    // Required for Hibernate's @JdbcTypeCode(SqlTypes.JSON) mapping (strategy_config.params/
    // overlays, signals.rule_outcomes/gate_outcomes - V46/V47) to find a JSON format mapper.
    // Matches the Jackson 3 ("tools.jackson") namespace already used by core/strategy.
    implementation("tools.jackson.core:jackson-databind")

    implementation("org.postgresql:postgresql")
    implementation("org.flywaydb:flyway-core:12.4.0")
    implementation("org.flywaydb:flyway-database-postgresql:12.4.0")

    // Resilience4j — circuit breaker, retry, bulkhead, time limiter.
    // 2.4.0+ is required for the resilience4j-spring-boot4 autoconfiguration module.
    implementation("io.github.resilience4j:resilience4j-spring-boot4:2.4.0")
    implementation("io.github.resilience4j:resilience4j-circuitbreaker:2.4.0")
    implementation("io.github.resilience4j:resilience4j-retry:2.4.0")
    implementation("io.github.resilience4j:resilience4j-bulkhead:2.4.0")
    implementation("io.github.resilience4j:resilience4j-timelimiter:2.4.0")
    implementation("io.github.resilience4j:resilience4j-micrometer:2.4.0")
    implementation("io.github.resilience4j:resilience4j-reactor:2.4.0")

    implementation("com.fyers:sdk:1.9.0")
    implementation("org.json:json:20231013")
    implementation("org.apache.commons:commons-csv:1.11.0")
    implementation("org.jsoup:jsoup:1.18.3")

    testImplementation("com.h2database:h2")
    testImplementation("com.squareup.okhttp3:mockwebserver:4.12.0")
    testImplementation("com.squareup.okhttp3:okhttp:4.12.0")
    testImplementation("org.springframework.boot:spring-boot-starter-test")
    testImplementation("org.junit.jupiter:junit-jupiter")
    testImplementation("org.junit.platform:junit-platform-launcher")
    testImplementation("org.mockito:mockito-core")
    testImplementation("org.assertj:assertj-core")
    testImplementation("org.wiremock:wiremock:3.8.0")
}

tasks {
    val checkMigrationVersions = register("checkMigrationVersions") {
        val migrationFiles = fileTree("src/main/resources/db/migration") {
            include("V*__*.sql")
        }
        inputs.files(migrationFiles)
        doLast {
            val migrationsByVersion = migrationFiles.files
                .mapNotNull { file ->
                    Regex("^V(\\d+)__.*\\.sql$")
                        .matchEntire(file.name)
                        ?.let { it.groupValues[1] to file }
                }
                .groupBy({ it.first }, { it.second })
            val duplicates = migrationsByVersion.filterValues { it.size > 1 }
            check(duplicates.isEmpty()) {
                duplicates.entries.joinToString(
                    prefix = "Duplicate Flyway migration versions:\n",
                    separator = "\n"
                ) { (version, files) -> "  V$version: ${files.joinToString { it.name }}" }
            }
        }
    }

    check {
        dependsOn(checkMigrationVersions)
    }

    compileJava {
        options.compilerArgs.add("-proc:none")
    }

    test {
        useJUnitPlatform()
        jvmArgs(
            "-Xms256m",
            "-Xmx512m",
            "--add-opens", "java.base/java.lang=ALL-UNNAMED",
            "--add-opens", "java.base/java.util=ALL-UNNAMED",
            "--add-opens", "java.base/java.math=ALL-UNNAMED",
            "--add-opens", "java.base/java.lang.reflect=ALL-UNNAMED"
        )
    }

    compileJava {
        options.compilerArgs.add("-parameters")
        options.encoding = "UTF-8"
    }
}
