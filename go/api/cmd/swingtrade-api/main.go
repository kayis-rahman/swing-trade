package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kayis-rahman/swing-trade/go/api"
	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/core/config"
	corelog "github.com/kayis-rahman/swing-trade/go/core/log"
	"github.com/kayis-rahman/swing-trade/go/data"
	"github.com/kayis-rahman/swing-trade/go/data/db"
	"github.com/kayis-rahman/swing-trade/go/data/scheduler"
)

// yahooBaseURL matches Java's yahoo.api.base-url default.
const yahooBaseURL = "https://query1.finance.yahoo.com"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	exitCode := runMain(ctx)
	stop()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func runMain(ctx context.Context) int {
	if err := run(ctx); err != nil {
		slog.Error("API stopped", "module", "api", "error", err)
		return 1
	}
	return 0
}

func run(ctx context.Context) error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	logger := corelog.New(os.Stdout, slog.LevelInfo, "api")

	var watchlistReader data.WatchlistReadStore
	var holidayReader data.HolidayReadStore
	var settingsReader data.SettingsReadStore
	var ingestionStatus data.IngestionStatusLookup
	if cfg.DatabaseURL != "" {
		pool, err := db.NewLocalOrReadOnlyPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("connect to database: %w", err)
		}
		defer pool.Close()
		watchlistReader = data.NewWatchlistReader(pool)
		holidayReader = data.NewHolidayReader(pool)
		settingsReader = data.NewSettingsReader(pool)
		status, err := data.NewIngestionStatusService(watchlistReader, data.NewCandleReader(pool), time.Now)
		if err != nil {
			return fmt.Errorf("create ingestion status service: %w", err)
		}
		ingestionStatus = status
		if err := wireIngestionJobs(ctx, logger, cfg, pool, settingsReader, holidayReader, watchlistReader); err != nil {
			return err
		}
	}

	handler, err := api.NewHandlerWithStores(cfg, logger, coreclock.System{}, watchlistReader, holidayReader, settingsReader, ingestionStatus)
	if err != nil {
		return fmt.Errorf("initialize HTTP platform: %w", err)
	}
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.HTTPAddr, err)
	}
	logger.Info("API listening", "addr", cfg.HTTPAddr)
	return serveHTTP(ctx, server, listener)
}

// wireIngestionJobs starts every explicitly enabled Go data job against the
// live pool; disabled gates construct nothing.
func wireIngestionJobs(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
	pool *pgxpool.Pool,
	settings data.SettingsReadStore,
	holidays data.HolidayReadStore,
	watchlist data.WatchlistReadStore,
) error {
	if cfg.EODEnabled || cfg.BenchmarkBackfillOnStartup || cfg.IntradayIngestionEnabled {
		writer, err := data.NewCandleWriter(pool)
		if err != nil {
			return fmt.Errorf("create candle writer: %w", err)
		}
		candles := data.NewCandleReader(pool)
		if cfg.EODEnabled || cfg.BenchmarkBackfillOnStartup {
			locker := func(ctx context.Context, fn func(context.Context) error) (bool, error) {
				return db.WithAdvisoryLock(ctx, pool, scheduler.JobLockEODIngestion, fn)
			}
			if err := startEODJobs(ctx, logger, cfg, settings, holidays, watchlist, candles, writer, locker); err != nil {
				return fmt.Errorf("initialize EOD ingestion job: %w", err)
			}
		}
		if cfg.IntradayIngestionEnabled {
			locker := func(ctx context.Context, fn func(context.Context) error) (bool, error) {
				return db.WithAdvisoryLock(ctx, pool, scheduler.JobLockIntradayIngestion, fn)
			}
			if err := startIntradayIngestionJob(ctx, logger, cfg, watchlist, holidays, candles, writer, locker); err != nil {
				return fmt.Errorf("initialize intraday ingestion job: %w", err)
			}
		}
	}
	if cfg.FyersSymbolMasterRefreshEnabled {
		symbolMaster, err := data.NewFyersSymbolMaster(
			data.NewFyersSymbolMasterClient(nil), data.NewFyersSymbolStore(pool), time.Now, logger)
		if err != nil {
			return fmt.Errorf("create Fyers symbol master service: %w", err)
		}
		locker := func(ctx context.Context, fn func(context.Context) error) (bool, error) {
			return db.WithAdvisoryLock(ctx, pool, scheduler.JobLockFyersSymbolMaster, fn)
		}
		if err := startSymbolMasterJob(ctx, logger, cfg, settings, symbolMaster, locker); err != nil {
			return fmt.Errorf("initialize Fyers symbol master job: %w", err)
		}
	}
	return nil
}

// startIntradayIngestionJob wires the scheduled intraday session-bar job
// behind its Go-only gate; a disabled flag never constructs anything.
func startIntradayIngestionJob(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
	watchlist data.WatchlistReadStore,
	holidays data.HolidayReadStore,
	latest data.IntervalLatestStore,
	writer *data.CandleWriter,
	locker scheduler.Locker,
) error {
	if !cfg.IntradayIngestionEnabled {
		return nil
	}
	client, err := data.NewYahooClient(yahooBaseURL, nil)
	if err != nil {
		return fmt.Errorf("create Yahoo market data client: %w", err)
	}
	ingestion, err := data.NewIntradayIngestion(watchlist, latest, holidays, client, writer, time.Now, logger)
	if err != nil {
		return err
	}
	scheduled, err := scheduler.NewIntradayIngestionScheduler(
		scheduler.IntradayIngestionConfig{
			Enabled: cfg.IntradayIngestionEnabled, LookbackDays: cfg.IntradayIngestionLookbackDays,
		},
		logger, time.Now, holidays, ingestion, locker,
	)
	if err != nil {
		return err
	}
	logger.Info("intraday ingestion job started", "next_run", scheduler.NextIntradayIngestionRun(time.Now()))
	go scheduled.StartLoop(ctx)
	return nil
}

// eodCandles is the daily-candle read surface the EOD ingestion job needs.
type eodCandles interface {
	data.CandleReadStore
	CandleExistsOnDate(ctx context.Context, symbol string, date time.Time) (bool, error)
}

// startEODJobs wires the EOD ingestion job over the selected market data broker.
// Java's provider map has no Fyers token source in Go yet, so a Fyers selection
// refuses to start rather than silently ingesting through a different provider.
func startEODJobs(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
	settings data.SettingsReadStore,
	holidays data.HolidayReadStore,
	watchlist data.WatchlistReadStore,
	candles eodCandles,
	writer *data.CandleWriter,
	locker scheduler.Locker,
) error {
	broker, err := settings.GetSelectedBroker(ctx)
	if err != nil {
		return fmt.Errorf("read selected market data broker: %w", err)
	}
	switch broker {
	case "fyers":
		logger.Warn("EOD ingestion job not started: the selected broker has no Go token source yet", "broker", broker)
		return nil
	case "yahoo":
	default:
		logger.Info("selected market data broker is unavailable in Go; using Yahoo like Java's fallback", "broker", broker)
	}
	client, err := data.NewYahooClient(yahooBaseURL, nil)
	if err != nil {
		return fmt.Errorf("create Yahoo market data client: %w", err)
	}
	ingester, err := data.NewCandleIngester(client, writer, holidays)
	if err != nil {
		return fmt.Errorf("create candle ingester: %w", err)
	}
	job, err := scheduler.NewEodIngestionScheduler(
		scheduler.EodConfig{
			Enabled:                    cfg.EODEnabled,
			BenchmarkBackfillOnStartup: cfg.BenchmarkBackfillOnStartup,
			BenchmarkBackfillYears:     cfg.BenchmarkBackfillYears,
		},
		logger, time.Now, watchlist, holidays, candles, candles, ingester, locker,
	)
	if err != nil {
		return err
	}
	if cfg.BenchmarkBackfillOnStartup {
		go job.EnsureNiftyBenchmarkHistory(ctx)
	}
	if cfg.EODEnabled {
		logger.Info("EOD ingestion job loop started", "next_run", scheduler.NextEodRun(time.Now()))
		go job.StartLoop(ctx)
	}
	return nil
}

// startSymbolMasterJob wires the symbol master refresh job behind its Go-only
// gate; a disabled flag never constructs or starts anything.
func startSymbolMasterJob(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
	settings data.SettingsReadStore,
	job scheduler.SymbolMasterJob,
	locker scheduler.Locker,
) error {
	if !cfg.FyersSymbolMasterRefreshEnabled {
		return nil
	}
	master, err := scheduler.NewFyersSymbolMasterScheduler(
		scheduler.FyersSymbolMasterConfig{Enabled: cfg.FyersSymbolMasterRefreshEnabled},
		logger,
		time.Now,
		func(ctx context.Context) (string, error) { return settings.GetSelectedBroker(ctx) },
		job,
		locker,
	)
	if err != nil {
		return err
	}
	logger.Info("Fyers symbol master refresh job started", "next_run", scheduler.NextFyersSymbolMasterRun(time.Now()))
	go master.RefreshIfEmpty(ctx)
	go master.StartLoop(ctx)
	return nil
}

func serveHTTP(ctx context.Context, server *http.Server, listener net.Listener) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	select {
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	}
}
