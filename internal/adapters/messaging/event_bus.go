package messaging

import (
	"context"
	"log"
	"sync"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
)

type InMemoryEventBus struct {
	mu       sync.RWMutex
	subs     map[string][]ports.EventHandler
	logger   *log.Logger
}

func NewInMemoryEventBus(logger *log.Logger) *InMemoryEventBus {
	if logger == nil {
		logger = log.Default()
	}
	return &InMemoryEventBus{
		subs:   make(map[string][]ports.EventHandler),
		logger: logger,
	}
}

func (b *InMemoryEventBus) Subscribe(eventType string, handler ports.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[eventType] = append(b.subs[eventType], handler)
	return nil
}

func (b *InMemoryEventBus) Publish(ctx context.Context, event domain.Event) error {
	b.mu.RLock()
	handlers := append([]ports.EventHandler(nil), b.subs[event.EventType()]...)
	b.mu.RUnlock()
	for _, h := range handlers {
		// run synchronously so causal ordering is preserved
		if err := h(ctx, event); err != nil {
			b.logger.Printf("event handler error type=%s: %v", event.EventType(), err)
		}
	}
	return nil
}
