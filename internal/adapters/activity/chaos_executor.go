package activity

import (
	"context"
	"log/slog"

	"github.com/super-client/internal/core/domain"
	"github.com/super-client/internal/core/ports"
)

// ChaosExecutor wraps an ActivityExecutor and injects chaos behavior.
// It applies random failures, crippling, and drift before/after execution.
type ChaosExecutor struct {
	inner  ports.ActivityExecutor
	chaos  domain.ChaosSource
	logger *slog.Logger
}

// NewChaosExecutor wraps inner with chaos injection using the given source.
func NewChaosExecutor(inner ports.ActivityExecutor, chaos domain.ChaosSource, logger *slog.Logger) *ChaosExecutor {
	return &ChaosExecutor{inner: inner, chaos: chaos, logger: logger}
}

func (e *ChaosExecutor) Execute(
	ctx context.Context,
	clientID string,
	activity domain.Activity,
	state domain.ClientState,
) (*domain.ClientState, *domain.ActivityResult, error) {
	// If client is crippled and this activity type is blocked, return failure.
	if domain.IsCrippledForActivity(state, activity.Type) {
		if state.CrippleMode == domain.CrippleModeSilent {
			// Silent mode: pretend success but don't change state
			e.logger.Debug("chaos: silent cripple, faking success", "client_id", clientID, "activity", activity.Type)
			noOpState := state.Clone()
			return &noOpState, &domain.ActivityResult{Status: domain.ResultSuccess}, nil
		}
		e.logger.Debug("chaos: crippled client failing activity", "client_id", clientID, "activity", activity.Type, "mode", state.CrippleMode)
		noOpState := state.Clone()
		return &noOpState, &domain.ActivityResult{
			Status:   domain.ResultFail,
			ErrorMsg: "client is crippled: " + string(state.CrippleMode),
		}, nil
	}

	// Random activity failure (10% base rate).
	if domain.ShouldActivityFail(e.chaos) {
		e.logger.Debug("chaos: random activity failure", "client_id", clientID, "activity", activity.Type)
		failedState := state.Clone()

		// Check if this failure should cripple the client.
		if domain.ShouldCrippleClient(e.chaos, true) {
			mode := domain.SelectCrippleMode(e.chaos)
			failedState.IsCrippled = true
			failedState.CrippleMode = mode
			e.logger.Info("chaos: client crippled", "client_id", clientID, "mode", mode)
		}

		return &failedState, &domain.ActivityResult{
			Status:   domain.ResultFail,
			ErrorMsg: "chaos: random failure injected",
		}, nil
	}

	// Execute the activity normally.
	newState, result, err := e.inner.Execute(ctx, clientID, activity, state)
	if err != nil {
		return nil, nil, err
	}

	// Apply spontaneous drift after a successful execution.
	if domain.ShouldDriftState(e.chaos) {
		drifted := domain.ApplyDrift(e.chaos, *newState)
		e.logger.Debug("chaos: spontaneous state drift applied", "client_id", clientID)
		newState = &drifted
	}

	return newState, result, nil
}
