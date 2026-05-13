package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
)

func TestInMemoryEventBus_Subscribe(t *testing.T) {
	bus := NewInMemoryEventBus(nil)
	var mu sync.Mutex
	got := []string{}
	_ = bus.Subscribe("health.updated", func(ctx context.Context, e domain.Event) error {
		mu.Lock()
		got = append(got, e.EventType())
		mu.Unlock()
		return nil
	})
	_ = bus.Subscribe("health.updated", func(ctx context.Context, e domain.Event) error {
		return errors.New("boom") // should be logged but not stop other handlers
	})
	_ = bus.Publish(context.Background(), &domain.HealthUpdatedEvent{WorkflowType: "x", Timestamp: time.Now()})
	_ = bus.Publish(context.Background(), &domain.CircuitBreakerStateChangedEvent{Timestamp: time.Now()})

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Errorf("expected single health event, got: %v", got)
	}
}

func TestChannelDispatcher(t *testing.T) {
	d := NewChannelDispatcher()
	ctx := context.Background()
	disp := &domain.Dispatch{RunID: "r", WorkflowID: "w", ClientID: "c", Activity: domain.ActivityReboot}
	if err := d.SendDispatch(ctx, disp); err != nil {
		t.Fatal(err)
	}
	if err := d.SendBatchDispatches(ctx, []*domain.Dispatch{disp, disp}); err != nil {
		t.Fatal(err)
	}
	if len(d.Snapshot()) != 3 {
		t.Errorf("expected 3, got %d", len(d.Snapshot()))
	}
	d.OnSend = func(d *domain.Dispatch) error { return errors.New("nope") }
	if err := d.SendDispatch(ctx, disp); err == nil {
		t.Error("expected error")
	}
}
