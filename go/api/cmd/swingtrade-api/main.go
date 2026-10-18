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

	"github.com/kayis-rahman/swing-trade/go/api"
	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/core/config"
	corelog "github.com/kayis-rahman/swing-trade/go/core/log"
	"github.com/kayis-rahman/swing-trade/go/data/db"
)

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

	if cfg.DatabaseURL != "" {
		pool, err := db.NewLocalOrReadOnlyPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("connect to database: %w", err)
		}
		defer pool.Close()
	}

	handler, err := api.NewHandlerWithLogger(cfg, logger, coreclock.System{})
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
