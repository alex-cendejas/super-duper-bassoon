package services

import (
	"context"

	"github.com/super-duper-bassoon/server/internal/core/ports"
)

type WorkflowStateManager struct {
	workflows ports.WorkflowRepository
}

func NewWorkflowStateManager(workflows ports.WorkflowRepository) *WorkflowStateManager {
	return &WorkflowStateManager{workflows: workflows}
}

func (m *WorkflowStateManager) DeactivateWorkflow(ctx context.Context, workflowID, reason string) error {
	return m.workflows.UpdateWorkflowState(ctx, workflowID, false, reason)
}

func (m *WorkflowStateManager) ActivateWorkflow(ctx context.Context, workflowID, reason string) error {
	return m.workflows.UpdateWorkflowState(ctx, workflowID, true, reason)
}

func (m *WorkflowStateManager) IsWorkflowActive(ctx context.Context, workflowID string) bool {
	wf, err := m.workflows.GetWorkflow(ctx, workflowID)
	if err != nil {
		return false
	}
	return wf.Active
}
