package com.swingtrade.api.app;

import com.swingtrade.api.config.TestConfigurationTypeFilter;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.persistence.autoconfigure.EntityScan;
import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.annotation.FilterType;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;
import java.util.Locale;


/**
 * Main application class for Swing Trade API module
 * Enables Spring Boot auto-configuration and component scanning
 * Enables scheduled tasks for weekly sector digest and other periodic jobs
 * <p>
 * Boot's default JPA repository/entity auto-configuration only scans the package of this
 * class (com.swingtrade.api.app) and below, not the extra @ComponentScan base packages -
 * repositories/entities under com.swingtrade.data were never being registered as beans.
 * <p>
 * basePackages includes com.swingtrade.api, which also contains test-only @TestConfiguration
 * classes (e.g. com.swingtrade.api.config.ErrorHandlingTestConfig) once the test source set is
 * on the classpath, as it is for full @SpringBootTest / integrationTest runs. Without this
 * explicit exclude filter, those classes' mock @Bean definitions (Mockito mocks of
 * PositionService, SignalService, etc.) get auto-detected and, because
 * spring.main.allow-bean-definition-overriding=true, silently replace the real service beans
 * in any full-context test - even ones that never reference the test config class.
 */
@ComponentScan(
    basePackages = {"com.swingtrade.api", "com.swingtrade.broker", "com.swingtrade.data", "com.swingtrade.strategy", "com.swingtrade.llm", "com.swingtrade.core"},
    excludeFilters = @ComponentScan.Filter(type = FilterType.CUSTOM, classes = TestConfigurationTypeFilter.class)
)
@EnableJpaRepositories(basePackages = {
        "com.swingtrade.data.repository",
        "com.swingtrade.broker.repository",
        "com.swingtrade.llm"
    })
@EntityScan(basePackages = {
        "com.swingtrade.data.entity",
        "com.swingtrade.broker.entity",
        "com.swingtrade.llm"
    })
@SpringBootApplication
public class SwingTradeApiApplication {

    static {
        Locale.setDefault(Locale.ROOT);
    }

    private SwingTradeApiApplication() {
    }

    public static void main(String[] args) {
        SpringApplication.run(SwingTradeApiApplication.class, args);
    }
}
