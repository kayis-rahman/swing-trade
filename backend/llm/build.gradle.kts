plugins {
    java
    id("io.spring.dependency-management")
}

dependencies {
    implementation(project(":core"))
    implementation(project(":data"))

    implementation("org.springframework.boot:spring-boot-starter")
    implementation("org.springframework.boot:spring-boot-starter-data-jpa")

    // Spring AI 2.0.1 — uses OkHttp (no Jetty dependency). Spring AI 1.1.0 (main's prior
    // pin, with Jetty 11 forced to match its JettyClientHttpRequestFactory) does not support
    // Spring Boot 4, so this upgrade keeps the Jetty-free 2.0.1 client instead.
    implementation("org.springframework.ai:spring-ai-starter-model-openai:2.0.1")

    implementation("com.fasterxml.jackson.core:jackson-annotations")

    // Resilience4j — circuit breaker, retry, bulkhead, time limiter.
    // 2.4.0+ is required for the resilience4j-spring-boot4 autoconfiguration module.
    implementation("io.github.resilience4j:resilience4j-spring-boot4:2.4.0")
    implementation("io.github.resilience4j:resilience4j-circuitbreaker:2.4.0")
    implementation("io.github.resilience4j:resilience4j-retry:2.4.0")
    implementation("io.github.resilience4j:resilience4j-bulkhead:2.4.0")
    implementation("io.github.resilience4j:resilience4j-timelimiter:2.4.0")
    implementation("io.github.resilience4j:resilience4j-micrometer:2.4.0")
    // Reactor operators (circuit breaker / retry) used by the reactive Laya client.
    implementation("io.github.resilience4j:resilience4j-reactor:2.4.0")
    implementation("commons-codec:commons-codec:1.16.0")
    implementation("org.apache.commons:commons-lang3:3.14.0")
    implementation("org.jsoup:jsoup:1.18.3")
    implementation("com.jcraft:jsch:0.1.55")

    testImplementation("org.testcontainers:testcontainers:1.21.3")
    testImplementation("org.testcontainers:postgresql:1.21.3")
    testImplementation("org.testcontainers:junit-jupiter:1.21.3")
    testImplementation("com.h2database:h2")
    testImplementation("org.springframework.boot:spring-boot-starter-test")
    testImplementation("org.junit.platform:junit-platform-launcher")
    testImplementation("org.mockito:mockito-core:5.12.0")
    testImplementation("org.assertj:assertj-core:3.26.3")
    testImplementation("com.squareup.okhttp3:mockwebserver:4.12.0")
    testImplementation("org.wiremock:wiremock:3.8.0")
}

tasks {
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
