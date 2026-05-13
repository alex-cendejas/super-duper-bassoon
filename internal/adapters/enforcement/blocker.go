package enforcement

import (
	"context"
	"sync"

	"github.com/super-duper-bassoon/internal/core/domain"
)

// InMemoryDispatchBlocker is an in-memory cache for active bans.
type InMemoryDispatchBlocker struct {
	mu    sync.RWMutex
	bans  map[string]map[string]struct{} // workflowType -> set of clientIDs (workflowType == "" means global)
}

func NewInMemoryDispatchBlocker() *InMemoryDispatchBlocker {
	return &InMemoryDispatchBlocker{bans: map[string]map[string]struct{}{}}
}

func (b *InMemoryDispatchBlocker) ShouldBlockDispatch(ctx context.Context, clientID, workflowType string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if m, ok := b.bans[workflowType]; ok {
		if _, present := m[clientID]; present {
			return true
		}
	}
	if m, ok := b.bans[""]; ok {
		if _, present := m[clientID]; present {
			return true
		}
	}
	return false
}

func (b *InMemoryDispatchBlocker) Add(ban *domain.BanRecord) {
	if ban == nil || !ban.IsActive() {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.bans[ban.WorkflowType]; !ok {
		b.bans[ban.WorkflowType] = map[string]struct{}{}
	}
	b.bans[ban.WorkflowType][ban.ClientID] = struct{}{}
}

func (b *InMemoryDispatchBlocker) Remove(clientID, workflowType string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if m, ok := b.bans[workflowType]; ok {
		delete(m, clientID)
	}
}
