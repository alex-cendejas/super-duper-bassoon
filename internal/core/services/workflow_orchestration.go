package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
)

type WorkflowOrchestrationService struct {
	workflows ports.WorkflowRepository
	runs      ports.RunRepository
	clients   ports.ClientRepository
	grouping  *DynamicGroupingService
	dispatch  *DispatchCoordinationService
	events    ports.EventBus
	logger    *log.Logger
}

func NewWorkflowOrchestrationService(workflows ports.WorkflowRepository, runs ports.RunRepository, clients ports.ClientRepository, grouping *DynamicGroupingService, dispatch *DispatchCoordinationService, events ports.EventBus, logger *log.Logger) *WorkflowOrchestrationService {
	if logger == nil {
		logger = log.Default()
	}
	return &WorkflowOrchestrationService{
		workflows: workflows,
		runs:      runs,
		clients:   clients,
		grouping:  grouping,
		dispatch:  dispatch,
		events:    events,
		logger:    logger,
	}
}

func (s *WorkflowOrchestrationService) CreateWorkflow(ctx context.Context, req *domain.CreateWorkflowRequest) (*domain.Workflow, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	wf := &domain.Workflow{
		ID:               uuid.NewString(),
		Name:             req.Name,
		Description:      req.Description,
		WorkflowType:     req.WorkflowType,
		Activity:         req.Activity,
		Params:           req.Params,
		TargetFilter:     req.TargetFilter,
		Trigger:          req.Trigger,
		SuccessThreshold: req.SuccessThreshold,
		LoopThresholdMS:  req.LoopThresholdMS,
		TimeoutMS:        req.TimeoutMS,
		Active:           true,
	}
	if req.Enabled != nil {
		wf.Active = *req.Enabled
	}
	if wf.SuccessThreshold == 0 {
		wf.SuccessThreshold = 80
	}
	if wf.LoopThresholdMS == 0 {
		wf.LoopThresholdMS = 5000
	}
	if wf.TimeoutMS == 0 {
		wf.TimeoutMS = 30000
	}
	if err := wf.ValidateDefinition(); err != nil {
		return nil, err
	}
	if err := s.workflows.SaveWorkflow(ctx, wf); err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *WorkflowOrchestrationService) EditWorkflow(ctx context.Context, id string, req *domain.EditWorkflowRequest) (*domain.Workflow, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	wf, err := s.workflows.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		wf.Name = *req.Name
	}
	if req.Description != nil {
		wf.Description = *req.Description
	}
	if req.Params != nil {
		wf.Params = req.Params
	}
	if req.TargetFilter != nil {
		wf.TargetFilter = *req.TargetFilter
	}
	if req.SuccessThreshold != nil {
		wf.SuccessThreshold = *req.SuccessThreshold
	}
	if req.LoopThresholdMS != nil {
		wf.LoopThresholdMS = *req.LoopThresholdMS
	}
	if req.TimeoutMS != nil {
		wf.TimeoutMS = *req.TimeoutMS
	}
	if req.Enabled != nil {
		wf.Active = *req.Enabled
	}
	if err := wf.ValidateDefinition(); err != nil {
		return nil, err
	}
	if err := s.workflows.SaveWorkflow(ctx, wf); err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *WorkflowOrchestrationService) DeleteWorkflow(ctx context.Context, id string) error {
	return s.workflows.DeleteWorkflow(ctx, id)
}

func (s *WorkflowOrchestrationService) ActivateWorkflow(ctx context.Context, id string) error {
	return s.workflows.UpdateWorkflowState(ctx, id, true, "manually activated")
}

func (s *WorkflowOrchestrationService) DeactivateWorkflow(ctx context.Context, id string) error {
	return s.workflows.UpdateWorkflowState(ctx, id, false, "manually deactivated")
}

func (s *WorkflowOrchestrationService) GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error) {
	return s.workflows.GetWorkflow(ctx, id)
}

func (s *WorkflowOrchestrationService) ListWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	return s.workflows.ListAllWorkflows(ctx)
}

// TriggerWorkflow runs a workflow: filter clients, create run, dispatch.
func (s *WorkflowOrchestrationService) TriggerWorkflow(ctx context.Context, id, reason string) (*domain.Run, error) {
	wf, err := s.workflows.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}
	if !wf.IsActive() {
		return nil, fmt.Errorf("%w: %s", domain.ErrWorkflowInactive, id)
	}
	var clientIDs []string
	if wf.TargetFilter == "" {
		all, err := s.clients.ListClients(ctx)
		if err != nil {
			return nil, err
		}
		for _, c := range all {
			if c.Active {
				clientIDs = append(clientIDs, c.ClientID)
			}
		}
	} else {
		fr, err := s.grouping.ResolveClients(ctx, wf.TargetFilter)
		if err != nil {
			return nil, err
		}
		clientIDs = fr.MatchingClientIDs
	}
	run := &domain.Run{
		RunID:                uuid.NewString(),
		WorkflowID:           wf.ID,
		WorkflowType:         wf.WorkflowType,
		TriggeredAt:          time.Now().UTC(),
		ParticipatingClients: clientIDs,
		State:                domain.RunPending,
		Reason:               reason,
	}
	if err := s.runs.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	_, _, err = s.dispatch.SendDispatches(ctx, run, wf, clientIDs)
	if err != nil {
		return run, err
	}
	if s.events != nil {
		_ = s.events.Publish(ctx, &domain.WorkflowCompletionEvent{
			WorkflowID:   wf.ID,
			WorkflowType: wf.WorkflowType,
			RunID:        run.RunID,
			State:        run.State,
			Timestamp:    time.Now().UTC(),
		})
	}
	return run, nil
}
