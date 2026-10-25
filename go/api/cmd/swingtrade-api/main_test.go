package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kayis-rahman/swing-trade/go/core/config"
	"github.com/kayis-rahman/swing-trade/go/data"
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

func TestStartEODJobsRefusesFyersSelection(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))
	err := startEODJobs(context.Background(), logger, config.Config{EODEnabled: true, BenchmarkBackfillYears: 10},
		stubSettings{broker: "fyers"}, stubHolidayStore{}, stubWatchlist{}, stubCandles{}, writerStub(t), stubLock)
	if err != nil {
		t.Fatalf("startEODJobs() error = %v", err)
	}
	if !strings.Contains(buffer.String(), "no Go token source") {
		t.Fatalf("missing fyers refusal log: %s", buffer.String())
	}
}

func TestStartEODJobsFallsBackToYahooForUnavailableBroker(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))
	err := startEODJobs(context.Background(), logger, config.Config{BenchmarkBackfillYears: 10},
		stubSettings{broker: "upstox"}, stubHolidayStore{}, stubWatchlist{}, stubCandles{}, writerStub(t), stubLock)
	if err != nil {
		t.Fatalf("startEODJobs() error = %v", err)
	}
	if !strings.Contains(buffer.String(), "using Yahoo like Java's fallback") {
		t.Fatalf("missing provider fallback log: %s", buffer.String())
	}
}

func TestStartEODJobsAcceptsYahooWithoutStartingJobs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := startEODJobs(context.Background(), logger, config.Config{BenchmarkBackfillYears: 10},
		stubSettings{broker: "yahoo"}, stubHolidayStore{}, stubWatchlist{}, stubCandles{}, writerStub(t), stubLock)
	if err != nil {
		t.Fatalf("startEODJobs() error = %v", err)
	}
}

func TestStartEODJobsPropagatesSettingsErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := startEODJobs(context.Background(), logger, config.Config{BenchmarkBackfillYears: 10},
		stubSettings{err: errors.New("settings down")}, stubHolidayStore{}, stubWatchlist{}, stubCandles{}, writerStub(t), stubLock)
	if err == nil || !strings.Contains(err.Error(), "selected market data broker") {
		t.Fatalf("settings error = %v", err)
	}
}

func TestStartEODJobsRequiresValidIngesterDependencies(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := startEODJobs(context.Background(), logger, config.Config{BenchmarkBackfillYears: 10},
		stubSettings{broker: "yahoo"}, nil, stubWatchlist{}, stubCandles{}, writerStub(t), stubLock)
	if err == nil || !strings.Contains(err.Error(), "holiday store") {
		t.Fatalf("nil-holiday error = %v", err)
	}
	err = startEODJobs(context.Background(), logger, config.Config{BenchmarkBackfillYears: 10},
		stubSettings{broker: "yahoo"}, stubHolidayStore{}, stubWatchlist{}, stubCandles{}, nil, stubLock)
	if err == nil || !strings.Contains(err.Error(), "candle writer") {
		t.Fatalf("nil-writer error = %v", err)
	}
}

func TestStartEODJobsRejectsInvalidBenchmarkWindow(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := startEODJobs(context.Background(), logger, config.Config{BenchmarkBackfillYears: 0},
		stubSettings{broker: "yahoo"}, stubHolidayStore{}, stubWatchlist{}, stubCandles{}, writerStub(t), stubLock)
	if err == nil || !strings.Contains(err.Error(), "backfill years") {
		t.Fatalf("invalid benchmark window error = %v", err)
	}
}

func TestStartEODJobsStartsEnabledJobGates(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	err := startEODJobs(ctx, logger, config.Config{EODEnabled: true, BenchmarkBackfillOnStartup: true, BenchmarkBackfillYears: 10},
		stubSettings{broker: "yahoo"}, stubHolidayStore{}, stubWatchlist{}, stubCandles{}, writerStub(t), stubLock)
	if err != nil {
		t.Fatalf("startEODJobs() error = %v", err)
	}
}

func TestRunWiresEODJobsWithAuthorizedDatabase(t *testing.T) {
	databaseURL := os.Getenv("SWINGTRADE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("authorized development database URL is unavailable")
	}
	port := freePort(t)
	setLocalEnvironment(t, port)
	t.Setenv("DATABASE_URL", databaseURL)
	t.Setenv("EOD_INGESTION_ENABLED", "true")
	t.Setenv("FYERS_SYMBOL_MASTER_REFRESH_ENABLED", "true")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx) }()

	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(10 * time.Second)
	listening := false
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			listening = true
			break
		}
	}
	if !listening {
		cancel()
		if drainErr := <-done; drainErr != nil {
			t.Logf("run() error while abandoning the listen wait: %v", drainErr)
		}
		t.Fatalf("service listening on %s was never reached", address)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run() error = %v", err)
	}
}

type stubSettings struct {
	broker string
	err    error
}

func (s stubSettings) GetSetting(context.Context, string) (string, bool, error) {
	return "", false, nil
}

func (s stubSettings) GetSelectedBroker(context.Context) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.broker, nil
}

type stubHolidayStore struct{}

func (stubHolidayStore) GetHolidayByDate(context.Context, time.Time) (*data.Holiday, error) {
	return nil, nil
}

func (stubHolidayStore) ListHolidaysByDateRange(context.Context, time.Time, time.Time) ([]data.Holiday, error) {
	return nil, nil
}

func (stubHolidayStore) ListUpcomingHolidays(context.Context, time.Time) ([]data.Holiday, error) {
	return nil, nil
}

type stubWatchlist struct{}

func (stubWatchlist) ListActiveWatchlist(context.Context) ([]data.WatchlistEntry, error) {
	return nil, nil
}

func (stubWatchlist) ListAllWatchlist(context.Context) ([]data.WatchlistEntry, error) {
	return nil, nil
}

// stubCandles fails window reads so the startup backfill never reaches the network.
type stubCandles struct{}

func (stubCandles) GetDailyDateWindow(context.Context, string) (data.CandleDateWindow, error) {
	return data.CandleDateWindow{}, errors.New("stub window failure")
}

func (stubCandles) CountDailyCandles(context.Context, string) (int64, error) { return 0, nil }

func (stubCandles) CandleExistsOnDate(context.Context, string, time.Time) (bool, error) {
	return false, nil
}

type execStub struct{}

func (execStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func writerStub(t *testing.T) *data.CandleWriter {
	t.Helper()
	writer, err := data.NewCandleWriter(execStub{})
	if err != nil {
		t.Fatal(err)
	}
	return writer
}

func stubLock(context.Context, func(context.Context) error) (bool, error) { return true, nil }

func TestStartSymbolMasterJobDoesNothingWhenDisabled(t *testing.T) {
	settings := &countingSettings{broker: "fyers"}
	err := startSymbolMasterJob(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)),
		config.Config{}, settings, stubSymbolMasterJob{}, stubLock)
	if err != nil {
		t.Fatalf("startSymbolMasterJob() error = %v", err)
	}
	if settings.calls() != 0 {
		t.Fatalf("disabled job read settings %d times", settings.calls())
	}
}

func TestStartSymbolMasterJobStartsEnabledGates(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	err := startSymbolMasterJob(ctx, logger,
		config.Config{FyersSymbolMasterRefreshEnabled: true},
		stubSettings{broker: "yahoo"}, stubSymbolMasterJob{}, stubLock)
	if err != nil {
		t.Fatalf("startSymbolMasterJob() error = %v", err)
	}
	if !strings.Contains(buffer.String(), "Fyers symbol master refresh job started") {
		t.Fatalf("missing job start log: %s", buffer.String())
	}
}

func TestStartSymbolMasterJobRejectsMissingJob(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	err := startSymbolMasterJob(context.Background(), logger,
		config.Config{FyersSymbolMasterRefreshEnabled: true},
		stubSettings{broker: "fyers"}, nil, stubLock)
	if err == nil {
		t.Fatal("nil symbol master accepted")
	}
}

type countingSettings struct {
	broker string
	read   int
}

func (s *countingSettings) GetSetting(context.Context, string) (string, bool, error) {
	return "", false, nil
}

func (s *countingSettings) GetSelectedBroker(context.Context) (string, error) {
	s.read++
	return s.broker, nil
}

func (s *countingSettings) calls() int { return s.read }

type stubSymbolMasterJob struct{}

func (stubSymbolMasterJob) Refresh(context.Context) (int, error) { return 0, nil }

func (stubSymbolMasterJob) CountSymbols(context.Context) (int64, error) { return 0, nil }
