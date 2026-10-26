package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestMainHandlesInterruptGracefully(t *testing.T) {
	setLocalEnvironment(t, freePort(t))
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	signalErr := make(chan error, 1)
	go func() {
		timer := time.NewTimer(50 * time.Millisecond)
		defer timer.Stop()
		<-timer.C
		signalErr <- process.Signal(os.Interrupt)
	}()
	main()
	if err := <-signalErr; err != nil {
		t.Fatalf("send interrupt signal: %v", err)
	}
}

func TestRunMainReturnsZeroAfterGracefulShutdown(t *testing.T) {
	port := freePort(t)
	setLocalEnvironment(t, port)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := runMain(ctx); got != 0 {
		t.Fatalf("runMain() = %d, want 0", got)
	}
}

func TestRunMainReturnsNonZeroOnConfigurationFailure(t *testing.T) {
	t.Setenv("SERVER_PORT", "invalid")
	if got := runMain(context.Background()); got != 1 {
		t.Fatalf("runMain() = %d, want 1", got)
	}
}

func TestRunGracefullyShutsDownWhenContextIsCancelled(t *testing.T) {
	port := freePort(t)
	setLocalEnvironment(t, port)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx); err != nil {
		t.Fatalf("run() error = %v", err)
	}
}

func TestRunRejectsInvalidDatabaseURLWithoutConnecting(t *testing.T) {
	setLocalEnvironment(t, freePort(t))
	t.Setenv("DATABASE_URL", "not-a-postgres-url")
	if err := run(context.Background()); err == nil {
		t.Fatal("run() succeeded with an invalid database URL")
	}
}

func TestRunReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	setLocalEnvironment(t, port)
	defer func() { _ = listener.Close() }()
	if err := run(context.Background()); err == nil {
		t.Fatal("run() succeeded while its address was already bound")
	}
}

func TestServeHTTPReturnsServerClosedWithoutError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	server := &http.Server{ReadHeaderTimeout: time.Second}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if err := serveHTTP(context.Background(), server, listener); err != nil {
		t.Fatalf("serveHTTP() error = %v, want nil for closed server", err)
	}
}

func TestServeHTTPReturnsUnexpectedListenerError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if err := serveHTTP(context.Background(), &http.Server{ReadHeaderTimeout: time.Second}, listener); err == nil {
		t.Fatal("serveHTTP() succeeded with a closed listener")
	}
}

func setLocalEnvironment(t *testing.T, port int) {
	t.Helper()
	t.Setenv("SERVER_PORT", strconv.Itoa(port))
	t.Setenv("HTTP_HOST", "127.0.0.1")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("API_KEY_AUTH_ENABLED", "false")
	t.Setenv("API_KEY", "")
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}
