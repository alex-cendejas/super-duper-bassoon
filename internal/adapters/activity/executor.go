package activity

import (
	"context"
	"fmt"

	"github.com/super-duper-bassoon/internal/core/domain"
)

// StandardExecutor executes activities deterministically (no chaos).
// Chaos injection is handled by the ChaosExecutor wrapper.
type StandardExecutor struct{}

// NewStandardExecutor creates a StandardExecutor.
func NewStandardExecutor() *StandardExecutor {
	return &StandardExecutor{}
}

func (e *StandardExecutor) Execute(
	_ context.Context,
	_ string,
	activity domain.Activity,
	state domain.ClientState,
) (*domain.ClientState, *domain.ActivityResult, error) {
	if !activity.IsValid() {
		return nil, nil, fmt.Errorf("%w: %s", domain.ErrInvalidActivity, activity.Type)
	}
	newState, result := domain.ExecuteActivity(activity, state)
	return &newState, &result, nil
}
