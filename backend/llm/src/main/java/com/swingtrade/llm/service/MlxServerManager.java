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
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;
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
    private static final String PYTHON_EXECUTABLE = "python3";
    private static final int STARTUP_TIMEOUT_SECONDS = 120;
    private static final int IDLE_CHECK_INTERVAL_SEC = 5;
    private static final Path DEFAULT_PID_FILE = Path.of(System.getProperty("user.home"), ".swingtrade", "mlx.pid");

    private final AppSettingsStore appSettingsStore;

    private final String defaultServerUrl;
    private final String defaultModel;
    private final int idleTimeoutSec;
    private final Path pidFile;

    private volatile Process serverProcess;
    private final AtomicBoolean starting = new AtomicBoolean(false);
    private final AtomicLong idleCheckTime = new AtomicLong(0);
    private final AtomicInteger inFlightRequests = new AtomicInteger(0);
    private final Object activityLock = new Object();
    private boolean restartPending;
    private boolean restartInProgress;
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
        this.defaultModel = llmProperties.getProviders().getMlx().getModel();
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
        recordActivity();
        if (isRunning()) {
            logger.debug("mlx_lm.server already running at {}", configuredEndpoint());
            if (isLocalEndpoint()) {
                startIdleMonitor();
            }
            return;
        }

        if (!isLocalEndpoint()) {
            throw new IllegalStateException(
                    "MLX server URL is remote; start mlx_lm.server on that host and use Refresh to check it");
        }

        if (!starting.compareAndSet(false, true)) {
            logger.debug("Another thread is starting mlx_lm.server, waiting...");
            long deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(STARTUP_TIMEOUT_SECONDS);
            while (starting.get() && System.nanoTime() < deadline) {
                if (isRunning()) {
                    return;
                }
                sleepQuietly(500);
            }
            if (isRunning()) {
                startIdleMonitor();
                return;
            }
            throw new IllegalStateException("Concurrent mlx_lm.server startup failed or timed out");
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
        if (!healthCheck()) {
            return false;
        }
        return !isLocalEndpoint() || hasManagedMlxProcess();
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

        synchronized (activityLock) {
            if (inFlightRequests.get() > 0 || restartInProgress) {
                restartPending = true;
                return;
            }
            restartInProgress = true;
        }
        runRestart(true);
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

        recordActivity();
        // Start idle monitor
        startIdleMonitor();
    }

    List<String> buildStartCommand(String modelName) {
        return List.of(PYTHON_EXECUTABLE, "-m", "mlx_lm.server", "--model", modelName,
                "--port", Integer.toString(configuredPort()), "--host", "127.0.0.1");
    }

    private void startIdleMonitor() {
        cancelIdleMonitor();
        ScheduledFuture<?> future = scheduler.scheduleWithFixedDelay(() -> {
            if (!isRunning()) return;
            try {
                synchronized (activityLock) {
                    if (inFlightRequests.get() > 0) {
                        return;
                    }
                    long idleSeconds = getIdleSeconds();
                    if (idleSeconds >= idleTimeoutSec) {
                        logger.info("mlx_lm.server idle for {}s >= {}s, auto-stopping", idleSeconds, idleTimeoutSec);
                        stop();
                    }
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

    long getIdleSeconds() {
        long lastActivity = idleCheckTime.get();
        return lastActivity > 0 ? (System.currentTimeMillis() - lastActivity) / 1000 : 0;
    }

    void setIdleCheckTime(long time) {
        idleCheckTime.set(time);
    }

    @Override
    public void beginRequest() {
        synchronized (activityLock) {
            while (restartPending || restartInProgress) {
                try {
                    activityLock.wait();
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    throw new IllegalStateException("Interrupted while waiting for MLX model restart", e);
                }
            }
            inFlightRequests.incrementAndGet();
            idleCheckTime.set(System.currentTimeMillis());
        }
    }

    @Override
    public void endRequest() {
        boolean runPendingRestart = false;
        synchronized (activityLock) {
            inFlightRequests.updateAndGet(count -> Math.max(0, count - 1));
            idleCheckTime.set(System.currentTimeMillis());
            if (inFlightRequests.get() == 0 && restartPending && !restartInProgress) {
                restartPending = false;
                restartInProgress = true;
                runPendingRestart = true;
            }
        }
        if (runPendingRestart) {
            scheduler.execute(() -> runRestart(false));
        }
    }

    boolean hasInFlightRequests() {
        return inFlightRequests.get() > 0;
    }

    private void recordActivity() {
        synchronized (activityLock) {
            idleCheckTime.set(System.currentTimeMillis());
        }
    }

    void restartServerProcess() throws Exception {
        stop();
        startServer();
    }

    private void runRestart(boolean propagateFailure) {
        IllegalStateException failure = null;
        try {
            logger.info("Restarting mlx_lm.server with updated model");
            restartServerProcess();
            logger.info("mlx_lm.server restarted successfully");
        } catch (Exception e) {
            logger.error("Failed to restart mlx_lm.server: {}", e.getMessage(), e);
            failure = new IllegalStateException("mlx_lm.server restart failed: " + e.getMessage(), e);
        } finally {
            finishRestart();
        }
        if (failure != null && propagateFailure) {
            throw failure;
        }
    }

    private void finishRestart() {
        boolean runAgain = false;
        synchronized (activityLock) {
            if (restartPending && inFlightRequests.get() == 0) {
                restartPending = false;
                runAgain = true;
            } else {
                restartInProgress = false;
                activityLock.notifyAll();
            }
        }
        if (runAgain) {
            scheduler.execute(() -> runRestart(false));
        }
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

    String getModelName() {
        return appSettingsStore.get("mlx.model")
                .orElse(defaultModel);
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

    private boolean hasManagedMlxProcess() {
        Process process = serverProcess;
        if (process != null && process.isAlive()) {
            return true;
        }
        try {
            String pidText = java.nio.file.Files.readString(pidFile).trim();
            ProcessHandle processHandle = ProcessHandle.of(Long.parseLong(pidText)).orElse(null);
            return processHandle != null && processHandle.isAlive() && isMlxProcess(processHandle);
        } catch (Exception e) {
            return false;
        }
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

    boolean isMlxAvailable() {
        try {
            ProcessBuilder pb = new ProcessBuilder(PYTHON_EXECUTABLE, "-c",
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
