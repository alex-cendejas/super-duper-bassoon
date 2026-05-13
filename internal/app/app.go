package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/super-client/internal/adapters/activity"
	"github.com/super-client/internal/adapters/clock"
	"github.com/super-client/internal/adapters/messaging"
	"github.com/super-client/internal/adapters/storage"
	"github.com/super-client/internal/core/domain"
	"github.com/super-client/internal/core/services"
)

// Config holds all runtime configuration for the super-client application.
type Config struct {
	NATSUrl      string
	PoolSize     int
	ClientPrefix string
	LogLevel     string
}

// App composes all layers and manages application lifecycle.
type App struct {
	pool   *services.ClientPoolManager
	broker *messaging.NATSBroker
	logger *slog.Logger
}

// New builds and wires the full application from configuration.
func New(cfg Config, logger *slog.Logger) (*App, error) {
	broker, err := messaging.NewNATSBroker(cfg.NATSUrl, logger)
	if err != nil {
		return nil, fmt.Errorf("create nats broker: %w", err)
	}

	store := storage.NewMemoryStore()
	standardExec := activity.NewStandardExecutor()
	chaosExec := activity.NewChaosExecutor(standardExec, domain.DefaultChaos, logger)
	clk := clock.SystemClock{}

	dispatcher := services.NewDispatchHandler(store, chaosExec)
	orchestrator := services.NewStateOrchestrator(store, clk, domain.DefaultChaos)
	collector := services.NewResultCollector(broker)

	pool := services.NewClientPoolManager(
		store, broker, dispatcher, collector, orchestrator,
		cfg.ClientPrefix, logger,
	)

	return &App{pool: pool, broker: broker, logger: logger}, nil
}

// Start initializes the client pool and begins the dispatch loop.
// Blocks until ctx is cancelled.
func (a *App) Start(ctx context.Context, poolSize int) error {
	if err := a.pool.Initialize(ctx, poolSize); err != nil {
		return fmt.Errorf("initialize pool: %w", err)
	}
	return a.pool.Run(ctx)
}

// Shutdown performs graceful teardown.
func (a *App) Shutdown(ctx context.Context) error {
	return a.pool.Shutdown(ctx)
}
