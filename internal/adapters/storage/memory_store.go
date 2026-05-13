package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/super-duper-bassoon/internal/core/domain"
)

// MemoryStore is an in-memory implementation of ports.StateStore.
type MemoryStore struct {
	mu     sync.RWMutex
	states map[string]*domain.ClientState
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{states: make(map[string]*domain.ClientState)}
}

func (s *MemoryStore) GetState(_ context.Context, clientID string) (*domain.ClientState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.states[clientID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrInnerClientNotFound, clientID)
	}
	clone := st.Clone()
	return &clone, nil
}

func (s *MemoryStore) UpdateState(_ context.Context, clientID string, state *domain.ClientState) error {
	if state == nil {
		return fmt.Errorf("nil state for client %s", clientID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	clone := state.Clone()
	s.states[clientID] = &clone
	return nil
}

func (s *MemoryStore) GetAllStates(_ context.Context) (map[string]*domain.ClientState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*domain.ClientState, len(s.states))
	for k, v := range s.states {
		clone := v.Clone()
		result[k] = &clone
	}
	return result, nil
}

func (s *MemoryStore) ListClientIDs(_ context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.states))
	for k := range s.states {
		ids = append(ids, k)
	}
	return ids, nil
}
