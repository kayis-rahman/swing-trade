package api

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRealBinaryServesEveryInfrastructureRoute(t *testing.T) {
	binary := buildAPIBinary(t)
	client := &http.Client{Timeout: time.Second}
	cmd, output, baseURL := startAPIBinary(t, binary)
	waitForHealth(t, client, baseURL, cmd, output)
	assertHealthRoutes(t, client, baseURL)
	assertWatchlistRoutesRequireDatabase(t, client, baseURL)
	assertAdminRoute(t, client, baseURL)
	assertAdminPreflight(t, client, baseURL)
	assertPrometheusRoute(t, client, baseURL)
}

func buildAPIBinary(t *testing.T) string {
	t.Helper()
	binary := t.TempDir() + "/swingtrade-api"
	// #nosec G204 -- executable and build arguments are fixed; the output is confined to t.TempDir.
	build := exec.Command("go", "build", "-o", binary, "./cmd/swingtrade-api")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build real API binary: %v\n%s", err, output)
	}
	return binary
}

func startAPIBinary(t *testing.T, binary string) (*exec.Cmd, *bytes.Buffer, string) {
	t.Helper()
	port, err := freeLocalPort()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary) // #nosec G204 -- binary was just built from this repository by buildAPIBinary.
	cmd.Env = replaceEnv(os.Environ(), map[string]string{
		"SERVER_PORT":                      strconv.Itoa(port),
		"API_KEY_AUTH_ENABLED":             "true",
		"API_KEY":                          "local-integration-key",
		"API_PUBLIC_RATE_LIMIT_PER_MINUTE": "100",
		"DATABASE_URL":                     "",
		"HTTP_HOST":                        "127.0.0.1",
	})
	output := &bytes.Buffer{}
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start real API binary: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
			_ = cmd.Wait()
		}
	})
	return cmd, output, fmt.Sprintf("http://127.0.0.1:%d", port)
}

func waitForHealth(t *testing.T, client *http.Client, baseURL string, cmd *exec.Cmd, output *bytes.Buffer) {
	t.Helper()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		response, err := client.Get(baseURL + "/actuator/health/liveness")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			_ = cmd.Process.Kill()
			t.Fatalf("real API binary did not become healthy; output:\n%s", output.String())
		}
	}
}

func assertWatchlistRoutesRequireDatabase(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()
	for _, path := range []string{"/api/watchlist", "/api/watchlist/all"} {
		response := makeRequest(t, client, http.MethodGet, baseURL+path, "")
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("GET %s without database = %d, want 503: %s", path, response.StatusCode, body)
		}
	}
}

func assertHealthRoutes(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()
	for _, path := range []string{
		"/api/health",
		"/actuator/health",
		"/actuator/health/liveness",
		"/actuator/health/readiness",
	} {
		response := makeRequest(t, client, http.MethodGet, baseURL+path, "")
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d", path, response.StatusCode)
		}
	}
}

func assertAdminRoute(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()
	url := baseURL + "/api/admin/health"
	unauthorized := makeRequest(t, client, http.MethodGet, url, "")
	if unauthorized.StatusCode != http.StatusUnauthorized || unauthorized.Header.Get("WWW-Authenticate") != "X-API-Key" {
		t.Fatalf("admin route without key = %d %v", unauthorized.StatusCode, unauthorized.Header)
	}
	_ = unauthorized.Body.Close()
	authorized := makeRequest(t, client, http.MethodGet, url, "local-integration-key")
	if authorized.StatusCode != http.StatusOK {
		t.Fatalf("admin route with key = %d", authorized.StatusCode)
	}
	_ = authorized.Body.Close()
}

func assertAdminPreflight(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodOptions, baseURL+"/api/admin/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", "GET")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNoContent || response.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		_ = response.Body.Close()
		t.Fatalf("admin preflight = %d %v", response.StatusCode, response.Header)
	}
	_ = response.Body.Close()
}

func assertPrometheusRoute(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()
	response := makeRequest(t, client, http.MethodGet, baseURL+"/actuator/prometheus", "")
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(response.Header.Get("Content-Type"), "text/plain") {
		t.Fatalf("Prometheus route = %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	for _, series := range []string{"http_server_requests_seconds_count", "api_request_duration_seconds_count", `application="swing-trade-api"`} {
		if !bytes.Contains(body, []byte(series)) {
			t.Errorf("Prometheus response missing %q", series)
		}
	}
}

func freeLocalPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

func replaceEnv(environment []string, values map[string]string) []string {
	out := make([]string, 0, len(environment)+len(values))
	for _, item := range environment {
		key, _, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if _, replaced := values[key]; !replaced {
			out = append(out, item)
		}
	}
	for key, value := range values {
		out = append(out, key+"="+value)
	}
	return out
}

func makeRequest(t *testing.T, client *http.Client, method, url, apiKey string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if apiKey != "" {
		request.Header.Set("X-API-Key", apiKey)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return response
}
