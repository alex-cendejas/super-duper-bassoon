package ports

import (
	"context"

	"github.com/super-duper-bassoon/internal/core/domain"
)

// StateStore abstracts state persistence for inner clients.
type StateStore interface {
	// GetState retrieves the current state of a client by ID.
	GetState(ctx context.Context, clientID string) (*domain.ClientState, error)

	// UpdateState atomically stores the new state for a client.
	UpdateState(ctx context.Context, clientID string, state *domain.ClientState) error

	// GetAllStates returns the state of every known client.
	GetAllStates(ctx context.Context) (map[string]*domain.ClientState, error)

	// ListClientIDs returns the IDs of all registered clients.
	ListClientIDs(ctx context.Context) ([]string, error)
}
