package com.swingtrade.llm.service;

import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.llm.config.LlmProperties;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import java.net.InetSocketAddress;
import java.net.URI;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class MlxServerManagerTest {

    @Test
    void doesNotTreatAnUnownedHealthyLocalListenerAsMlx() throws Exception {
        HttpServer server = healthServer(200);
        server.start();
        try {
            MlxServerManager manager = managerFor(server);

            assertThat(manager.isRunning()).isFalse();
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

    @Test
    void launchCommandUsesTheConfiguredEndpointPort() {
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("mlx.server.url"))
                .thenReturn(Optional.of("http://127.0.0.1:8090/v1"));
        LlmProperties properties = new LlmProperties();
        properties.getProviders().getMlx().setBaseUrl(URI.create("http://127.0.0.1:8081/v1"));
        properties.getProviders().getMlx().setModel("configured-mlx-model");
        MlxServerManager manager = new MlxServerManager(settings, properties, 300);

        assertThat(manager.getModelName()).isEqualTo("configured-mlx-model");

        List<String> command = manager.buildStartCommand("mlx-model");
        int portOption = command.indexOf("--port");

        assertThat(command.get(portOption + 1)).isEqualTo("8090");
    }

    @Test
    void stopLeavesUnrelatedProcessFromPidFileRunning(@TempDir Path tempDir) throws Exception {
        Process unrelatedProcess = new ProcessBuilder("sleep", "30").start();
        Path pidFile = tempDir.resolve("mlx.pid");
        Files.writeString(pidFile, Long.toString(unrelatedProcess.pid()));
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("mlx.server.url")).thenReturn(Optional.empty());
        LlmProperties properties = new LlmProperties();
        properties.getProviders().getMlx().setBaseUrl(URI.create("http://127.0.0.1:8081/v1"));
        MlxServerManager manager = new MlxServerManager(settings, properties, 300, pidFile);

        try {
            manager.stop();

            assertThat(unrelatedProcess.isAlive()).isTrue();
        } finally {
            unrelatedProcess.destroyForcibly();
            unrelatedProcess.waitFor();
        }
    }

    @Test
    void waitingCallerFailsWhenConcurrentStartupFails(@TempDir Path tempDir) throws Exception {
        HttpServer server = healthServer(200);
        int port = server.getAddress().getPort();
        server.stop(0);
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("mlx.server.url")).thenReturn(Optional.empty());
        LlmProperties properties = new LlmProperties();
        properties.getProviders().getMlx().setBaseUrl(URI.create("http://127.0.0.1:" + port + "/v1"));
        CountDownLatch availabilityCheckStarted = new CountDownLatch(1);
        CountDownLatch allowStartupFailure = new CountDownLatch(1);
        MlxServerManager manager = new MlxServerManager(settings, properties, 300, tempDir.resolve("mlx.pid")) {
            @Override
            boolean isMlxAvailable() {
                availabilityCheckStarted.countDown();
                try {
                    allowStartupFailure.await(3, TimeUnit.SECONDS);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                }
                return false;
            }
        };
        ExecutorService executor = Executors.newFixedThreadPool(2);

        try {
            Future<?> starter = executor.submit(manager::ensureRunning);
            assertThat(availabilityCheckStarted.await(3, TimeUnit.SECONDS)).isTrue();
            Future<?> waiter = executor.submit(manager::ensureRunning);
            allowStartupFailure.countDown();

            assertThatThrownBy(() -> starter.get(5, TimeUnit.SECONDS))
                    .hasRootCauseInstanceOf(IllegalStateException.class);
            assertThatThrownBy(() -> waiter.get(5, TimeUnit.SECONDS))
                    .hasRootCauseInstanceOf(IllegalStateException.class);
        } finally {
            allowStartupFailure.countDown();
            executor.shutdownNow();
        }
    }

    @Test
    void requestActivityRefreshesIdleClockAndTracksConcurrentRequests() {
        MlxServerManager manager = managerForEndpoint("http://127.0.0.1:8081/v1");
        manager.setIdleCheckTime(System.currentTimeMillis() - 10_000);

        manager.beginRequest();
        manager.beginRequest();
        assertThat(manager.hasInFlightRequests()).isTrue();
        assertThat(manager.getIdleSeconds()).isLessThan(manager.getIdleTimeoutSec());

        manager.endRequest();
        assertThat(manager.hasInFlightRequests()).isTrue();
        manager.endRequest();
        assertThat(manager.hasInFlightRequests()).isFalse();
        assertThat(manager.getIdleSeconds()).isLessThan(manager.getIdleTimeoutSec());
    }

    @Test
    void defersRestartUntilInFlightRequestsFinish() throws Exception {
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("mlx.server.url")).thenReturn(Optional.empty());
        LlmProperties properties = new LlmProperties();
        properties.getProviders().getMlx().setBaseUrl(URI.create("http://127.0.0.1:8081/v1"));
        CountDownLatch restartStarted = new CountDownLatch(1);
        AtomicInteger restartCount = new AtomicInteger();
        MlxServerManager manager = new MlxServerManager(settings, properties, 300) {
            @Override
            void restartServerProcess() {
                restartCount.incrementAndGet();
                restartStarted.countDown();
            }
        };

        manager.beginRequest();
        manager.restart();
        assertThat(restartCount.get()).isZero();
        assertThat(manager.hasInFlightRequests()).isTrue();

        manager.endRequest();

        assertThat(restartStarted.await(2, TimeUnit.SECONDS)).isTrue();
        assertThat(restartCount.get()).isEqualTo(1);
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
        return managerForEndpoint(endpoint);
    }

    private MlxServerManager managerForEndpoint(String endpoint) {
        LlmProperties properties = new LlmProperties();
        properties.getProviders().getMlx().setBaseUrl(URI.create(endpoint));
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("mlx.server.url")).thenReturn(Optional.empty());
        return new MlxServerManager(settings, properties, 300,
                Path.of("build", "test-fixtures", "mlx-test.pid"));
    }
}
