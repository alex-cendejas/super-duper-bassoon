package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/super-duper-bassoon/server/internal/adapters/messaging"
	"github.com/super-duper-bassoon/server/internal/core/domain"
)

func TestInMemoryEventBus_PubSub(t *testing.T) {
	bus := messaging.NewInMemoryEventBus(nil)
	var mu sync.Mutex
	count := 0
	_ = bus.Subscribe("health.updated", func(ctx context.Context, e domain.Event) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	})
	_ = bus.Publish(context.Background(), &domain.HealthUpdatedEvent{WorkflowType: "x", Timestamp: time.Now()})
	_ = bus.Publish(context.Background(), &domain.CircuitBreakerStateChangedEvent{Timestamp: time.Now()})
	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 call, got %d", count)
	}
}
