package services

import (
	"context"
	"fmt"

	"github.com/super-client/internal/core/domain"
	"github.com/super-client/internal/core/ports"
)

// StateOrchestrator coordinates state mutations, chaos side effects, and
// cripple recovery detection.
type StateOrchestrator struct {
	store ports.StateStore
	clock ports.Clock
	chaos domain.ChaosSource
}

// NewStateOrchestrator creates a StateOrchestrator.
func NewStateOrchestrator(store ports.StateStore, clock ports.Clock, chaos domain.ChaosSource) *StateOrchestrator {
	return &StateOrchestrator{store: store, clock: clock, chaos: chaos}
}

// ApplyActivityResult persists the updated state resulting from an activity.
func (so *StateOrchestrator) ApplyActivityResult(
	ctx context.Context,
	clientID string,
	newState *domain.ClientState,
) error {
	if err := so.store.UpdateState(ctx, clientID, newState); err != nil {
		return fmt.Errorf("update state: %w", err)
	}
	return nil
}

// ApplyDriftIfNeeded rolls the dice and applies spontaneous state drift.
// Returns true if drift was applied.
func (so *StateOrchestrator) ApplyDriftIfNeeded(ctx context.Context, clientID string) (bool, error) {
	if !domain.ShouldDriftState(so.chaos) {
		return false, nil
	}
	state, err := so.store.GetState(ctx, clientID)
	if err != nil {
		return false, fmt.Errorf("get state: %w", err)
	}
	drifted := domain.ApplyDrift(so.chaos, *state)
	if err := so.store.UpdateState(ctx, clientID, &drifted); err != nil {
		return false, fmt.Errorf("update drift state: %w", err)
	}
	return true, nil
}

// CheckRecoveryFromCripple returns true if a successful reboot activity
// has cleared the crippled state. It compares the state before/after.
func (so *StateOrchestrator) CheckRecoveryFromCripple(
	prevState *domain.ClientState,
	newState *domain.ClientState,
	activity domain.Activity,
) bool {
	if activity.Type != domain.ActivityReboot {
		return false
	}
	return prevState.IsCrippled && !newState.IsCrippled
}
