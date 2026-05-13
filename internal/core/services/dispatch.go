package services

import (
	"context"
	"fmt"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
)

// DispatchHandler validates and routes incoming dispatch messages to the
// activity executor.
type DispatchHandler struct {
	store    ports.StateStore
	executor ports.ActivityExecutor
}

// NewDispatchHandler creates a DispatchHandler with the given dependencies.
func NewDispatchHandler(store ports.StateStore, executor ports.ActivityExecutor) *DispatchHandler {
	return &DispatchHandler{store: store, executor: executor}
}

// ValidateDispatch checks that a DispatchMessage is well-formed.
func ValidateDispatch(d domain.DispatchMessage) error {
	if d.RunID == "" {
		return fmt.Errorf("missing run_id")
	}
	if d.WfID == "" {
		return fmt.Errorf("missing wf_id")
	}
	if d.ClientID == "" {
		return fmt.Errorf("missing client_id")
	}
	if !d.Activity.IsValid() {
		return fmt.Errorf("%w: %s", domain.ErrInvalidActivity, d.Activity.Type)
	}
	return nil
}

// Handle processes a dispatch: validates, fetches state, executes activity,
// and returns the updated state and result. It does NOT persist state changes;
// the caller is responsible for that.
func (h *DispatchHandler) Handle(
	ctx context.Context,
	dispatch domain.DispatchMessage,
) (*domain.ClientState, *domain.ActivityResult, error) {
	if err := ValidateDispatch(dispatch); err != nil {
		return nil, nil, fmt.Errorf("invalid dispatch: %w", err)
	}

	state, err := h.store.GetState(ctx, dispatch.ClientID)
	if err != nil {
		return nil, nil, fmt.Errorf("get state for %s: %w", dispatch.ClientID, err)
	}

	newState, result, err := h.executor.Execute(ctx, dispatch.ClientID, dispatch.Activity, *state)
	if err != nil {
		return nil, nil, fmt.Errorf("execute activity: %w", err)
	}

	return newState, result, nil
}
