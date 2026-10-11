plugins {
    java
    id("io.spring.dependency-management")
}

dependencies {
    implementation(project(":core"))
    implementation(project(":data"))

    implementation("org.springframework.boot:spring-boot-starter")
    implementation("org.ta4j:ta4j-core:0.16")

    // Jackson 3 — versions managed by jackson-bom
    implementation("tools.jackson.core:jackson-databind")

    testImplementation(project(":data"))
    testImplementation("org.springframework.boot:spring-boot-starter-test")
    testImplementation("org.junit.jupiter:junit-jupiter")
    testImplementation("org.junit.platform:junit-platform-launcher")
    testImplementation("org.mockito:mockito-core")
    testImplementation("org.assertj:assertj-core")
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
