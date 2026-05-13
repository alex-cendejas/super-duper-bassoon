package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
)

// ClientPoolManager manages the lifecycle and dispatch routing for all inner
// clients in this super-client instance.
type ClientPoolManager struct {
	store     ports.StateStore
	broker    ports.MessageBroker
	dispatcher *DispatchHandler
	collector  *ResultCollector
	orchestrator *StateOrchestrator
	logger    *slog.Logger
	prefix    string
	clientIDs []string
}

// NewClientPoolManager creates a ClientPoolManager.
func NewClientPoolManager(
	store ports.StateStore,
	broker ports.MessageBroker,
	dispatcher *DispatchHandler,
	collector *ResultCollector,
	orchestrator *StateOrchestrator,
	prefix string,
	logger *slog.Logger,
) *ClientPoolManager {
	return &ClientPoolManager{
		store:        store,
		broker:       broker,
		dispatcher:   dispatcher,
		collector:    collector,
		orchestrator: orchestrator,
		logger:       logger,
		prefix:       prefix,
	}
}

// Initialize creates N inner clients with default state in the store.
func (m *ClientPoolManager) Initialize(ctx context.Context, poolSize int) error {
	if poolSize <= 0 {
		return fmt.Errorf("pool size must be positive, got %d", poolSize)
	}
	m.clientIDs = make([]string, 0, poolSize)
	for i := range poolSize {
		clientID := fmt.Sprintf("%s-%04d", m.prefix, i+1)
		client := domain.NewInnerClient(clientID)
		if err := m.store.UpdateState(ctx, clientID, &client.State); err != nil {
			return fmt.Errorf("initialize client %s: %w", clientID, err)
		}
		m.clientIDs = append(m.clientIDs, clientID)
	}
	m.logger.Info("client pool initialized", "pool_size", poolSize, "prefix", m.prefix)
	return nil
}

// ClientIDs returns the IDs of all managed clients.
func (m *ClientPoolManager) ClientIDs() []string {
	result := make([]string, len(m.clientIDs))
	copy(result, m.clientIDs)
	return result
}

// HandleDispatch processes a dispatch message for a single client:
// validates, executes the activity, persists the updated state, and queues
// the result for publication.
func (m *ClientPoolManager) HandleDispatch(ctx context.Context, dispatch domain.DispatchMessage) error {
	m.logger.Debug("handling dispatch",
		"client_id", dispatch.ClientID,
		"run_id", dispatch.RunID,
		"activity", dispatch.Activity.Type,
	)

	// Snapshot state before execution (for recovery detection).
	prevState, err := m.store.GetState(ctx, dispatch.ClientID)
	if err != nil {
		return fmt.Errorf("get state before dispatch: %w", err)
	}

	newState, result, err := m.dispatcher.Handle(ctx, dispatch)
	if err != nil {
		// Record an error result so the server knows what happened.
		errResult := domain.ActivityResult{
			Status:   domain.ResultError,
			ErrorMsg: err.Error(),
		}
		if collectErr := m.collector.Collect(dispatch.RunID, dispatch.WfID, dispatch.ClientID, errResult, prevState); collectErr != nil {
			m.logger.Warn("failed to collect error result", "err", collectErr)
		}
		return fmt.Errorf("dispatch handle: %w", err)
	}

	// Persist the new state.
	if err := m.orchestrator.ApplyActivityResult(ctx, dispatch.ClientID, newState); err != nil {
		return fmt.Errorf("apply activity result: %w", err)
	}

	if m.orchestrator.CheckRecoveryFromCripple(prevState, newState, dispatch.Activity) {
		m.logger.Info("client recovered from cripple via reboot", "client_id", dispatch.ClientID)
	}

	// Queue result for publishing.
	if err := m.collector.Collect(dispatch.RunID, dispatch.WfID, dispatch.ClientID, *result, newState); err != nil {
		return fmt.Errorf("collect result: %w", err)
	}

	return nil
}

// Run starts the main dispatch loop: subscribes to broker, processes messages,
// and periodically flushes results. Blocks until ctx is cancelled.
func (m *ClientPoolManager) Run(ctx context.Context) error {
	dispatches, err := m.broker.SubscribeDispatch(ctx, m.clientIDs)
	if err != nil {
		return fmt.Errorf("subscribe dispatch: %w", err)
	}

	m.logger.Info("super-client running", "clients", len(m.clientIDs))
	for {
		select {
		case <-ctx.Done():
			m.logger.Info("context cancelled, flushing results")
			// Best-effort final flush.
			_ = m.collector.FlushResults(context.Background())
			return nil
		case msg, ok := <-dispatches:
			if !ok {
				return nil
			}
			if err := m.HandleDispatch(ctx, msg); err != nil {
				m.logger.Warn("dispatch error", "client_id", msg.ClientID, "err", err)
			}
			// Flush after each result for the PoC (no batching delay needed).
			if err := m.collector.FlushResults(ctx); err != nil {
				m.logger.Warn("flush results error", "err", err)
			}
		}
	}
}

// GetClientState fetches the current state of a managed client.
func (m *ClientPoolManager) GetClientState(ctx context.Context, clientID string) (*domain.ClientState, error) {
	return m.store.GetState(ctx, clientID)
}

// Shutdown performs graceful teardown.
func (m *ClientPoolManager) Shutdown(ctx context.Context) error {
	if err := m.collector.FlushResults(ctx); err != nil {
		m.logger.Warn("flush on shutdown failed", "err", err)
	}
	return m.broker.Close(ctx)
}
