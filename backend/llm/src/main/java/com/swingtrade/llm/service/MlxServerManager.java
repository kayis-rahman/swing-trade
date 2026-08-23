package com.swingtrade.llm.service;

import com.swingtrade.domain.store.AppSettingsStore;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.io.BufferedReader;
import java.io.File;
import java.io.InputStreamReader;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;
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
    private static final String PID_FILE_PATH = System.getProperty("user.home") + "/.swingtrade/mlx.pid";

    private final AppSettingsStore appSettingsStore;

    private final String port;
    private final int idleTimeoutSec;

    private volatile Process serverProcess;
    private final AtomicBoolean starting = new AtomicBoolean(false);
    private final AtomicReference<ScheduledFuture<?>> idleMonitor = new AtomicReference<>();
    private final ScheduledExecutorService scheduler = Executors.newSingleThreadScheduledExecutor(r -> {
        Thread t = new Thread(r, "mlx-idle-monitor");
        t.setDaemon(true);
        return t;
    });

    public MlxServerManager(AppSettingsStore appSettingsStore,
                            @Value("${mlx.server.url:http://192.168.1.50:8081}") String serverUrl,
                            @Value("${mlx.port:8081}") String port,
                            @Value("${mlx.idle-timeout:300}") int idleTimeoutSec) {
        this.appSettingsStore = appSettingsStore;
        this.port = port;
        this.idleTimeoutSec = idleTimeoutSec;
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
            logger.debug("mlx_lm.server already running on port {}", port);
            return;
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
            logger.info("Starting mlx_lm.server on port {} (model: {})", port, getModelName());
            startServer();
            logger.info("mlx_lm.server started successfully on port {}", port);
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
        // Stop by PID file first
        stopByPid();

        // Also stop by port
        stopByPort();

        serverProcess = null;
        cancelIdleMonitor();
        logger.info("mlx_lm.server stopped");
    }

    /**
     * Checks if the server process is running.
     * Uses three checks: PID file, port socket, health endpoint.
     */
    @Override
    public boolean isRunning() {
        // Check 1: PID file + process alive
        if (serverProcess != null && serverProcess.isAlive()) {
            return true;
        }

        // Check 2: Port socket (server may have been started externally)
        if (isPortInUse(Integer.parseInt(port))) {
            return true;
        }

        // Check 3: Health endpoint (most reliable)
        return healthCheck();
    }

    /**
     * Restarts the server with the current model from settings.
     * Stops the current process then starts a new one.
     */
    @Override
    public void restart() {
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
        List<String> command = new ArrayList<>();
        command.add("python");
        command.add("-m");
        command.add("mlx_lm.server");
        command.add("--model");
        command.add(modelName);
        command.add("--port");
        command.add(port);
        command.add("--host");
        command.add("0.0.0.0");

        ProcessBuilder pb = new ProcessBuilder(command);
        pb.redirectErrorStream(true);
        pb.redirectOutput(new File("/dev/null"));

        serverProcess = pb.start();
        logger.info("mlx_lm.server process started (pid={})", serverProcess.pid());

        // Write PID file
        writePidFile(serverProcess.pid());

        // Wait for health endpoint
        long waited = 0;
        while (waited < STARTUP_TIMEOUT_SECONDS) {
            if (isPortInUse(Integer.parseInt(port))) {
                // Give it a moment more for the HTTP layer to be ready
                sleepQuietly(2000);
                if (healthCheck()) {
                    logger.info("mlx_lm.server is healthy after {}s", waited);
                    break;
                }
            }
            sleepQuietly(1000);
            waited++;

            // Check if process died
            if (!serverProcess.isAlive()) {
                int exitCode = serverProcess.exitValue();
                stopByPid();
                throw new IllegalStateException("mlx_lm.server exited with code " + exitCode);
            }
        }

        if (!healthCheck()) {
            stop();
            throw new IllegalStateException("mlx_lm.server failed to become healthy within " + STARTUP_TIMEOUT_SECONDS + "s");
        }

        // Start idle monitor
        startIdleMonitor();
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
            String url = appSettingsStore.get("mlx.server.url")
                    .orElse("http://192.168.1.50:8081");
            java.net.http.HttpClient client = java.net.http.HttpClient.newHttpClient();
            java.net.http.HttpRequest request = java.net.http.HttpRequest.newBuilder()
                    .uri(java.net.URI.create(url + "/health"))
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

    private boolean isPortInUse(int port) {
        try (Socket socket = new Socket()) {
            socket.connect(new InetSocketAddress("127.0.0.1", port), 100);
            return true;
        } catch (Exception e) {
            return false;
        }
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
            File dir = new File(new File(PID_FILE_PATH).getParent());
            if (!dir.exists()) {
                dir.mkdirs();
            }
            java.nio.file.Files.writeString(
                    java.nio.file.Paths.get(PID_FILE_PATH),
                    String.valueOf(pid));
        } catch (Exception e) {
            logger.debug("Failed to write PID file: {}", e.getMessage());
        }
    }

    private void stopByPid() {
        File pidFile = new File(PID_FILE_PATH);
        if (pidFile.exists()) {
            try {
                String pidStr = java.nio.file.Files.readString(pidFile.toPath()).trim();
                long pid = Long.parseLong(pidStr);
                // Kill the process group
                ProcessBuilder pb = new ProcessBuilder("kill", "-TERM", String.valueOf(pid));
                pb.redirectErrorStream(true);
                Process p = pb.start();
                p.waitFor(5, TimeUnit.SECONDS);
                // Also kill child processes
                pb = new ProcessBuilder("sh", "-c", String.format(
                        "pkill -TERM -P %d 2>/dev/null; kill -TERM %d 2>/dev/null", pid, pid));
                pb.redirectErrorStream(true);
                p = pb.start();
                p.waitFor(5, TimeUnit.SECONDS);
                pidFile.delete();
                logger.info("Stopped mlx_lm.server by PID {}", pid);
            } catch (Exception e) {
                logger.debug("PID stop failed: {}", e.getMessage());
            }
        }
    }

    private void stopByPort() {
        try {
            ProcessBuilder pb = new ProcessBuilder("sh", "-c",
                    String.format("lsof -ti:%s | xargs kill -TERM 2>/dev/null", port));
            pb.redirectErrorStream(true);
            Process p = pb.start();
            p.waitFor(5, TimeUnit.SECONDS);
        } catch (Exception e) {
            logger.debug("Port stop failed: {}", e.getMessage());
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