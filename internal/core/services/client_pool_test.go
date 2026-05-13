package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/super-duper-bassoon/internal/adapters/clock"
	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/services"
	"log/slog"
	"os"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func buildPool(t *testing.T, prefix string) (
	*services.ClientPoolManager,
	*MockStateStore,
	*MockMessageBroker,
	*MockActivityExecutor,
) {
	t.Helper()
	store := NewMockStateStore()
	broker := NewMockMessageBroker()
	executor := &MockActivityExecutor{}
	clk := clock.NewMockClock(time.Now())
	chaos := deterministicChaos{float64Val: 0.5, intnVal: 0} // no failures

	dispatcher := services.NewDispatchHandler(store, executor)
	orchestrator := services.NewStateOrchestrator(store, clk, chaos)
	collector := services.NewResultCollector(broker)
	pool := services.NewClientPoolManager(store, broker, dispatcher, collector, orchestrator, prefix, testLogger())
	return pool, store, broker, executor
}

func TestClientPool_Initialize(t *testing.T) {
	pool, store, _, _ := buildPool(t, "test")
	ctx := context.Background()

	if err := pool.Initialize(ctx, 3); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	ids := pool.ClientIDs()
	if len(ids) != 3 {
		t.Errorf("expected 3 client IDs, got %d", len(ids))
	}

	for _, id := range ids {
		state, err := store.GetState(ctx, id)
		if err != nil {
			t.Errorf("client %s not found in store: %v", id, err)
		}
		if state.PowerState != domain.PowerStateOn {
			t.Errorf("client %s expected PowerState=on, got %s", id, state.PowerState)
		}
	}
}

func TestClientPool_Initialize_ZeroSize(t *testing.T) {
	pool, _, _, _ := buildPool(t, "test")
	if err := pool.Initialize(context.Background(), 0); err == nil {
		t.Error("expected error for pool size 0")
	}
}

func TestClientPool_Initialize_NegativeSize(t *testing.T) {
	pool, _, _, _ := buildPool(t, "test")
	if err := pool.Initialize(context.Background(), -1); err == nil {
		t.Error("expected error for negative pool size")
	}
}

func TestClientPool_ClientIDsPrefix(t *testing.T) {
	pool, _, _, _ := buildPool(t, "myprefix")
	pool.Initialize(context.Background(), 2)
	ids := pool.ClientIDs()
	for _, id := range ids {
		if len(id) < len("myprefix") || id[:len("myprefix")] != "myprefix" {
			t.Errorf("expected client ID to start with 'myprefix', got %s", id)
		}
	}
}

func TestClientPool_HandleDispatch_Success(t *testing.T) {
	pool, store, broker, _ := buildPool(t, "client")
	ctx := context.Background()
	pool.Initialize(ctx, 1)

	clientID := pool.ClientIDs()[0]
	// Ensure the client has no packages so install succeeds
	store.Seed(clientID, domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
	})

	dispatch := domain.DispatchMessage{
		RunID:    "run-001",
		WfID:     "wf-001",
		ClientID: clientID,
		Activity: domain.Activity{
			Type:   domain.ActivityInstallPackage,
			Params: map[string]interface{}{"package": "curl"},
		},
	}

	if err := pool.HandleDispatch(ctx, dispatch); err != nil {
		t.Fatalf("HandleDispatch failed: %v", err)
	}

	// HandleDispatch collects results; flush is separate (Run flushes after each dispatch).
	// Use Shutdown to trigger the final flush.
	if err := pool.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	published := broker.Published()
	if len(published) != 1 {
		t.Fatalf("expected 1 published result, got %d", len(published))
	}
	if published[0].Status != domain.ResultSuccess {
		t.Errorf("expected success result, got %s", published[0].Status)
	}
}

func TestClientPool_HandleDispatch_UnknownClient(t *testing.T) {
	pool, _, _, _ := buildPool(t, "client")
	ctx := context.Background()
	pool.Initialize(ctx, 1)

	dispatch := domain.DispatchMessage{
		RunID:    "run-001",
		WfID:     "wf-001",
		ClientID: "unknown-client",
		Activity: domain.Activity{
			Type:   domain.ActivityReboot,
		},
	}

	// Should error but not panic
	err := pool.HandleDispatch(ctx, dispatch)
	if err == nil {
		t.Error("expected error for unknown client")
	}
}

func TestClientPool_HandleDispatch_InvalidDispatch(t *testing.T) {
	pool, _, _, _ := buildPool(t, "client")
	ctx := context.Background()
	pool.Initialize(ctx, 1)

	clientID := pool.ClientIDs()[0]
	dispatch := domain.DispatchMessage{
		// Missing RunID — result cannot be collected (needs run_id), so 0 published
		WfID:     "wf-001",
		ClientID: clientID,
		Activity: domain.Activity{Type: domain.ActivityReboot},
	}

	err := pool.HandleDispatch(ctx, dispatch)
	if err == nil {
		t.Error("expected error for invalid dispatch")
	}
}

func TestClientPool_GetClientState(t *testing.T) {
	pool, _, _, _ := buildPool(t, "client")
	ctx := context.Background()
	pool.Initialize(ctx, 2)

	clientID := pool.ClientIDs()[0]
	state, err := pool.GetClientState(ctx, clientID)
	if err != nil {
		t.Fatalf("GetClientState failed: %v", err)
	}
	if state == nil {
		t.Error("expected non-nil state")
	}
}

func TestClientPool_Initialize_RegistersClients(t *testing.T) {
	pool, _, broker, _ := buildPool(t, "reg")
	ctx := context.Background()

	if err := pool.Initialize(ctx, 3); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	registered := broker.Registered
	if len(registered) != 3 {
		t.Errorf("expected 3 registered clients, got %d", len(registered))
	}

	ids := pool.ClientIDs()
	regByID := make(map[string]domain.ClientMetadata, len(registered))
	for _, r := range registered {
		regByID[r.ClientID] = r
	}
	for _, id := range ids {
		r, ok := regByID[id]
		if !ok {
			t.Errorf("client %s was not registered", id)
			continue
		}
		if !r.Active {
			t.Errorf("client %s should be active", id)
		}
		if r.Labels == nil {
			t.Errorf("client %s Labels should not be nil", id)
		}
		if r.InnerState == nil {
			t.Errorf("client %s InnerState should not be nil", id)
		}
	}
}

func TestClientPool_Run_ProcessesDispatch(t *testing.T) {
	pool, store, broker, _ := buildPool(t, "client")
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	pool.Initialize(context.Background(), 1)
	clientID := pool.ClientIDs()[0]
	store.Seed(clientID, domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
	})

	// Send a dispatch before Run starts (buffered channel)
	broker.SendDispatch(domain.DispatchMessage{
		RunID:    "run-001",
		WfID:     "wf-001",
		ClientID: clientID,
		Activity: domain.Activity{
			Type:   domain.ActivityReboot,
		},
	})

	// Run blocks until ctx cancels
	pool.Run(ctx)

	published := broker.Published()
	if len(published) != 1 {
		t.Errorf("expected 1 published result after run, got %d", len(published))
	}
}
