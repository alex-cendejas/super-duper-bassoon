package ports

import (
	"context"

	"github.com/super-client/internal/core/domain"
)

// ActivityExecutor abstracts the execution of activities on client state.
type ActivityExecutor interface {
	// Execute runs an activity against the given client state and returns the
	// updated state and result.
	Execute(ctx context.Context, clientID string, activity domain.Activity, state domain.ClientState) (*domain.ClientState, *domain.ActivityResult, error)
}
