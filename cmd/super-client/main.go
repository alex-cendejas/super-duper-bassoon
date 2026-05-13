package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/super-duper-bassoon/internal/app"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)

	application, err := app.New(app.Config{
		NATSUrl:      cfg.NATSUrl,
		PoolSize:     cfg.PoolSize,
		ClientPrefix: cfg.ClientPrefix,
		LogLevel:     cfg.LogLevel,
	}, logger)
	if err != nil {
		logger.Error("failed to create application", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	logger.Info("starting super-client", "pool_size", cfg.PoolSize, "prefix", cfg.ClientPrefix)
	if err := application.Start(ctx, cfg.PoolSize); err != nil {
		logger.Error("application error", "err", err)
		os.Exit(1)
	}

	shutdownCtx := context.Background()
	if err := application.Shutdown(shutdownCtx); err != nil {
		logger.Warn("shutdown error", "err", err)
	}
	logger.Info("super-client stopped")
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
