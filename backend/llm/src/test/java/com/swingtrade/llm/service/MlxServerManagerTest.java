package com.swingtrade.llm.service;

import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.llm.config.LlmProperties;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.Test;

import java.net.InetSocketAddress;
import java.net.URI;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class MlxServerManagerTest {

    @Test
    void usesTheConfiguredProviderUrlForStatusWhenNoDatabaseOverrideExists() throws Exception {
        HttpServer server = healthServer(200);
        server.start();
        try {
            MlxServerManager manager = managerFor(server);

            assertThat(manager.isRunning()).isTrue();
        } finally {
            server.stop(0);
        }
    }

    @Test
    void reportsAnUnhealthyListenerAsNotRunning() throws Exception {
        HttpServer server = healthServer(503);
        server.start();
        try {
            MlxServerManager manager = managerFor(server);

            assertThat(manager.isRunning()).isFalse();
        } finally {
            server.stop(0);
        }
    }

    private HttpServer healthServer(int status) throws Exception {
        HttpServer server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/health", exchange -> {
            exchange.sendResponseHeaders(status, -1);
            exchange.close();
        });
        return server;
    }

    private MlxServerManager managerFor(HttpServer server) {
        String endpoint = "http://127.0.0.1:" + server.getAddress().getPort() + "/v1";
        LlmProperties properties = new LlmProperties();
        properties.getProviders().getMlx().setBaseUrl(URI.create(endpoint));
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("mlx.server.url")).thenReturn(Optional.empty());
        return new MlxServerManager(settings, properties,
                String.valueOf(server.getAddress().getPort()), 300);
    }
}
