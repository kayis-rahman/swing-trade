package com.swingtrade.llm.service;

import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.llm.config.LlmProperties;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.io.BufferedReader;
import java.io.File;
import java.io.InputStreamReader;
import java.nio.file.Path;
import java.time.Duration;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Manages the mlx_lm.server process lifecycle from Java.
 *
 * <p>Responsibilities:
 * <ul>
 *   <li>Start mlx_lm.server as a child process on first LLM request (lazy start)</li>
 *   <li>Wait for model load via health endpoint polling</li>
 *   <li>Auto-stop after idle timeout to conserve resources</li>
 *   <li>Restart when model path changes</li>
 * </ul>
 */
@Service
public class MlxServerManager implements LlmServerManager {

    private static final Logger logger = LoggerFactory.getLogger(MlxServerManager.class);

    private static final String HEALTH_URL = "%s/health";
    private static final int STARTUP_TIMEOUT_SECONDS = 120;
    private static final int IDLE_CHECK_INTERVAL_SEC = 5;
    private static final Path DEFAULT_PID_FILE = Path.of(System.getProperty("user.home"), ".swingtrade", "mlx.pid");

    private final AppSettingsStore appSettingsStore;

    private final String defaultServerUrl;
    private final int idleTimeoutSec;
    private final Path pidFile;

    private volatile Process serverProcess;
    private final AtomicBoolean starting = new AtomicBoolean(false);
    private final AtomicReference<ScheduledFuture<?>> idleMonitor = new AtomicReference<>();
    private final ScheduledExecutorService scheduler = Executors.newSingleThreadScheduledExecutor(r -> {
        Thread t = new Thread(r, "mlx-idle-monitor");
        t.setDaemon(true);
        return t;
    });

    public MlxServerManager(AppSettingsStore appSettingsStore,
                            LlmProperties llmProperties,
                            @Value("${mlx.idle-timeout:300}") int idleTimeoutSec) {
        this(appSettingsStore, llmProperties, idleTimeoutSec, DEFAULT_PID_FILE);
    }

    MlxServerManager(AppSettingsStore appSettingsStore,
                     LlmProperties llmProperties,
                     int idleTimeoutSec,
                     Path pidFile) {
        this.appSettingsStore = appSettingsStore;
        this.defaultServerUrl = llmProperties.getProviders().getMlx().getBaseUrl().toString();
        this.idleTimeoutSec = idleTimeoutSec;
        this.pidFile = pidFile;
    }

    /**
     * Ensures the mlx_lm.server is running. Blocks until the server is ready.
     * If the server is already running, returns immediately.
     *
     * @throws IllegalStateException if the server fails to start
     */
    @Override
    public void ensureRunning() {
        if (isRunning()) {
            logger.debug("mlx_lm.server already running at {}", configuredEndpoint());
            return;
        }

        if (!isLocalEndpoint()) {
            throw new IllegalStateException(
                    "MLX server URL is remote; start mlx_lm.server on that host and use Refresh to check it");
        }

        if (!starting.compareAndSet(false, true)) {
            // Another thread is already starting — wait for it
            logger.debug("Another thread is starting mlx_lm.server, waiting...");
            while (!isRunning()) {
                sleepQuietly(500);
            }
            starting.set(false);
            return;
        }

        try {
            logger.info("Starting mlx_lm.server at {} (model: {})", configuredEndpoint(), getModelName());
            startServer();
            logger.info("mlx_lm.server started successfully at {}", configuredEndpoint());
        } catch (Exception e) {
            logger.error("Failed to start mlx_lm.server: {}", e.getMessage(), e);
            throw new IllegalStateException("mlx_lm.server failed to start: " + e.getMessage(), e);
        } finally {
            starting.set(false);
        }
    }

    /**
     * Stops the mlx_lm.server process.
     */
    @Override
    public void stop() {
        if (!isLocalEndpoint()) {
            logger.info("Remote mlx_lm.server is manually managed; skipping local stop");
            return;
        }

        Process process = serverProcess;
        if (process != null) {
            stopProcess(process);
            serverProcess = null;
        }

        stopByPid();
        cancelIdleMonitor();
        logger.info("mlx_lm.server stopped");
    }

    /**
     * Checks whether the configured MLX endpoint is healthy.
     */
    @Override
    public boolean isRunning() {
        return healthCheck();
    }

    /**
     * Restarts the server with the current model from settings.
     * Stops the current process then starts a new one.
     */
    @Override
    public void restart() {
        if (!isLocalEndpoint()) {
            logger.info("Remote mlx_lm.server is manually managed; skipping local restart");
            return;
        }
        logger.info("Restarting mlx_lm.server with updated model");
        stop();
        try {
            startServer();
            logger.info("mlx_lm.server restarted successfully");
        } catch (Exception e) {
            logger.error("Failed to restart mlx_lm.server: {}", e.getMessage(), e);
            throw new IllegalStateException("mlx_lm.server restart failed: " + e.getMessage(), e);
        }
    }

    private void startServer() throws Exception {
        String modelName = getModelName();

        // Check if mlx_lm module is available (Python import check)
        if (!isMlxAvailable()) {
            throw new IllegalStateException("mlx_lm module not found. Install with: pip install mlx-lm");
        }

        // Build command
        List<String> command = buildStartCommand(modelName);

        ProcessBuilder pb = new ProcessBuilder(command);
        pb.redirectErrorStream(true);
        pb.redirectOutput(new File("/dev/null"));

        serverProcess = pb.start();
        logger.info("mlx_lm.server process started (pid={})", serverProcess.pid());

        // Write PID file
        writePidFile(serverProcess.pid());

        long startupStartedAt = System.nanoTime();
        long startupDeadline = startupStartedAt + TimeUnit.SECONDS.toNanos(STARTUP_TIMEOUT_SECONDS);
        boolean healthy = false;
        while (System.nanoTime() < startupDeadline) {
            if (!serverProcess.isAlive()) {
                int exitCode = serverProcess.exitValue();
                stopByPid();
                throw new IllegalStateException("mlx_lm.server exited with code " + exitCode);
            }
            if (healthCheck()) {
                healthy = true;
                long elapsedSeconds = TimeUnit.NANOSECONDS.toSeconds(System.nanoTime() - startupStartedAt);
                logger.info("mlx_lm.server is healthy after {}s", elapsedSeconds);
                break;
            }
            long remainingMillis = TimeUnit.NANOSECONDS.toMillis(startupDeadline - System.nanoTime());
            if (remainingMillis > 0) {
                sleepQuietly(Math.min(1000, remainingMillis));
            }
        }

        if (!healthy) {
            stop();
            throw new IllegalStateException("mlx_lm.server failed to become healthy within " + STARTUP_TIMEOUT_SECONDS + "s");
        }

        // Start idle monitor
        startIdleMonitor();
    }

    List<String> buildStartCommand(String modelName) {
        return List.of("python", "-m", "mlx_lm.server", "--model", modelName,
                "--port", Integer.toString(configuredPort()), "--host", "127.0.0.1");
    }

    private void startIdleMonitor() {
        cancelIdleMonitor();
        ScheduledFuture<?> future = scheduler.scheduleWithFixedDelay(() -> {
            if (!isRunning()) return;
            try {
                long idleSeconds = getIdleSeconds();
                if (idleSeconds >= idleTimeoutSec) {
                    logger.info("mlx_lm.server idle for {}s >= {}s, auto-stopping", idleSeconds, idleTimeoutSec);
                    stop();
                }
            } catch (Exception e) {
                logger.debug("Idle monitor check failed: {}", e.getMessage());
            }
        }, IDLE_CHECK_INTERVAL_SEC, IDLE_CHECK_INTERVAL_SEC, TimeUnit.SECONDS);
        idleMonitor.set(future);
    }

    private void cancelIdleMonitor() {
        ScheduledFuture<?> f = idleMonitor.getAndSet(null);
        if (f != null) f.cancel(false);
    }

    private long idleCheckTime = 0;

    long getIdleSeconds() {
        return idleCheckTime > 0 ? (System.currentTimeMillis() - idleCheckTime) / 1000 : 0;
    }

    void setIdleCheckTime(long time) {
        idleCheckTime = time;
    }

    int getIdleTimeoutSec() {
        return idleTimeoutSec;
    }

    boolean healthCheck() {
        try {
            java.net.URI baseUri = configuredEndpoint();
            java.net.http.HttpClient client = java.net.http.HttpClient.newHttpClient();
            java.net.http.HttpRequest request = java.net.http.HttpRequest.newBuilder()
                    .uri(baseUri.resolve("/health"))
                    .timeout(Duration.ofSeconds(3))
                    .GET()
                    .build();
            java.net.http.HttpResponse<String> response = client.send(request,
                    java.net.http.HttpResponse.BodyHandlers.ofString());
            return response.statusCode() == 200;
        } catch (Exception e) {
            return false;
        }
    }

    private String getModelName() {
        return appSettingsStore.get("mlx.model")
                .orElse("Qwen/Qwen2.5-3B-Instruct");
    }

    private boolean isLocalEndpoint() {
        try {
            java.net.URI uri = configuredEndpoint();
            String host = uri.getHost();
            return host != null && (host.equalsIgnoreCase("localhost")
                    || host.equals("127.0.0.1") || host.equals("::1"));
        } catch (IllegalArgumentException e) {
            return false;
        }
    }

    private java.net.URI configuredEndpoint() {
        return java.net.URI.create(appSettingsStore.get("mlx.server.url").orElse(defaultServerUrl));
    }

    private int configuredPort() {
        java.net.URI endpoint = configuredEndpoint();
        if (endpoint.getPort() >= 0) {
            return endpoint.getPort();
        }
        if ("http".equalsIgnoreCase(endpoint.getScheme())) {
            return 80;
        }
        if ("https".equalsIgnoreCase(endpoint.getScheme())) {
            return 443;
        }
        throw new IllegalStateException("MLX server URL must specify a port or use HTTP(S)");
    }

    private boolean isMlxAvailable() {
        try {
            ProcessBuilder pb = new ProcessBuilder("python", "-c",
                    "import mlx_lm; print('ok')");
            pb.redirectErrorStream(true);
            Process p = pb.start();
            boolean finished = p.waitFor(10, TimeUnit.SECONDS);
            if (!finished) {
                p.destroyForcibly();
                return false;
            }
            try (BufferedReader reader = new BufferedReader(new InputStreamReader(p.getInputStream()))) {
                String output = reader.readLine();
                return "ok".equals(output);
            }
        } catch (Exception e) {
            logger.debug("mlx_lm availability check failed: {}", e.getMessage());
            return false;
        }
    }

    private void writePidFile(long pid) {
        try {
            File dir = pidFile.toFile().getParentFile();
            if (!dir.exists()) {
                dir.mkdirs();
            }
            java.nio.file.Files.writeString(pidFile, String.valueOf(pid));
        } catch (Exception e) {
            logger.debug("Failed to write PID file: {}", e.getMessage());
        }
    }

    private void stopByPid() {
        if (java.nio.file.Files.exists(pidFile)) {
            try {
                String pidStr = java.nio.file.Files.readString(pidFile).trim();
                long pid = Long.parseLong(pidStr);
                ProcessHandle handle = ProcessHandle.of(pid).orElse(null);
                if (handle != null && handle.isAlive() && isMlxProcess(handle)) {
                    stopProcess(handle);
                    logger.info("Stopped mlx_lm.server by PID {}", pid);
                }
                java.nio.file.Files.deleteIfExists(pidFile);
            } catch (Exception e) {
                logger.debug("PID stop failed: {}", e.getMessage());
            }
        }
    }

    private boolean isMlxProcess(ProcessHandle handle) {
        return handle.info().commandLine().map(command -> command.contains("mlx_lm.server")).orElse(false);
    }

    private void stopProcess(Process process) {
        process.destroy();
        try {
            if (!process.waitFor(5, TimeUnit.SECONDS)) {
                process.destroyForcibly();
                process.waitFor(5, TimeUnit.SECONDS);
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            process.destroyForcibly();
        }
    }

    private void stopProcess(ProcessHandle process) {
        process.destroy();
        try {
            process.onExit().get(5, TimeUnit.SECONDS);
        } catch (TimeoutException e) {
            process.destroyForcibly();
            try {
                process.onExit().get(5, TimeUnit.SECONDS);
            } catch (Exception forcedStopFailure) {
                logger.debug("Forced MLX process stop failed: {}", forcedStopFailure.getMessage());
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            process.destroyForcibly();
        } catch (Exception e) {
            logger.debug("MLX process stop failed: {}", e.getMessage());
        }
    }

    private void sleepQuietly(long ms) {
        try {
            Thread.sleep(ms);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }
}
