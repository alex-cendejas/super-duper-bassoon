package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
)

type CircuitBreakerService struct {
	health      ports.HealthRepository
	circuits    ports.CircuitBreakerStateRepository
	policy      ports.PolicyRepository
	workflows   ports.WorkflowRepository
	stateMgr    ports.WorkflowStateManager
	alerts      ports.AlertPublisher
	events      ports.EventBus
	logic       *domain.CircuitBreakerLogic
	defaultPol  *domain.CircuitBreakerPolicy
	logger      *log.Logger
}

func NewCircuitBreakerService(health ports.HealthRepository, circuits ports.CircuitBreakerStateRepository, policy ports.PolicyRepository, workflows ports.WorkflowRepository, stateMgr ports.WorkflowStateManager, alerts ports.AlertPublisher, events ports.EventBus, defaultPolicy *domain.CircuitBreakerPolicy, logger *log.Logger) *CircuitBreakerService {
	if logger == nil {
		logger = log.Default()
	}
	if defaultPolicy == nil {
		defaultPolicy = &domain.CircuitBreakerPolicy{SuccessThreshold: 80, EvaluationWindow: 10, CooldownPeriod: 5 * time.Minute}
	}
	return &CircuitBreakerService{
		health:     health,
		circuits:   circuits,
		policy:     policy,
		workflows:  workflows,
		stateMgr:   stateMgr,
		alerts:     alerts,
		events:     events,
		logic:      domain.NewCircuitBreakerLogic(),
		defaultPol: defaultPolicy,
		logger:     logger,
	}
}

func (s *CircuitBreakerService) OnHealthUpdatedEvent(ctx context.Context, evt domain.Event) error {
	e, ok := evt.(*domain.HealthUpdatedEvent)
	if !ok {
		return nil
	}
	return s.EvaluateWorkflowType(ctx, e.WorkflowType, e.TypeHealth)
}

func (s *CircuitBreakerService) EvaluateWorkflowType(ctx context.Context, workflowType string, h *domain.WorkflowTypeHealth) error {
	if h == nil {
		var err error
		h, err = s.health.GetWorkflowTypeHealth(ctx, workflowType)
		if err != nil {
			return nil
		}
	}
	workflows, err := s.workflows.ListWorkflowsByType(ctx, workflowType)
	if err != nil {
		return err
	}
	for _, wf := range workflows {
		policy := s.defaultPol
		if s.policy != nil {
			if p, err := s.policy.GetPolicy(ctx, wf.ID); err == nil && p != nil {
				policy = p
			}
		}
		newState := s.logic.EvaluateHealth(policy, h)
		current, err := s.circuits.GetCircuitState(ctx, wf.ID)
		if err != nil {
			current = &domain.WorkflowCircuitBreaker{
				WorkflowID:   wf.ID,
				WorkflowType: wf.WorkflowType,
				State:        domain.CircuitClosed,
			}
		}
		current.LastEvaluatedAt = time.Now().UTC()
		current.EvaluationCount++
		oldState := current.State
		if oldState != newState {
			current.State = newState
			reason := fmt.Sprintf("success=%.2f%% threshold=%.2f%% runs=%d", h.SuccessPercentageAvg, policy.SuccessThreshold, h.RunsConsidered)
			if newState == domain.CircuitOpen {
				current.OpenedAt = time.Now().UTC()
				current.OpenedReason = reason
				if s.stateMgr != nil {
					_ = s.stateMgr.DeactivateWorkflow(ctx, wf.ID, reason)
				}
				if s.alerts != nil {
					a := domain.NewAlert(domain.AlertCircuitOpened, domain.SeverityCritical, fmt.Sprintf("circuit opened for %s: %s", wf.ID, reason))
					a.Details["workflow_id"] = wf.ID
					a.Details["workflow_type"] = wf.WorkflowType
					a.Details["reason"] = reason
					_ = s.alerts.PublishAlert(ctx, a)
				}
			} else if newState == domain.CircuitClosed {
				current.OpenedAt = time.Time{}
				current.OpenedReason = ""
				if s.stateMgr != nil {
					_ = s.stateMgr.ActivateWorkflow(ctx, wf.ID, "circuit recovered")
				}
				if s.alerts != nil {
					a := domain.NewAlert(domain.AlertCircuitClosed, domain.SeverityInfo, fmt.Sprintf("circuit closed for %s", wf.ID))
					a.Details["workflow_id"] = wf.ID
					a.Details["workflow_type"] = wf.WorkflowType
					_ = s.alerts.PublishAlert(ctx, a)
				}
			}
			if s.events != nil {
				_ = s.events.Publish(ctx, &domain.CircuitBreakerStateChangedEvent{
					WorkflowID:   wf.ID,
					WorkflowType: wf.WorkflowType,
					OldState:     oldState,
					NewState:     newState,
					Reason:       reason,
					Timestamp:    time.Now().UTC(),
				})
			}
		}
		if err := s.circuits.SaveCircuitState(ctx, current); err != nil {
			return err
		}
	}
	return nil
}

func (s *CircuitBreakerService) EvaluateAllWorkflows(ctx context.Context) error {
	workflows, err := s.workflows.ListAllWorkflows(ctx)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, w := range workflows {
		if _, ok := seen[w.WorkflowType]; ok {
			continue
		}
		seen[w.WorkflowType] = struct{}{}
		_ = s.EvaluateWorkflowType(ctx, w.WorkflowType, nil)
	}
	return nil
}

func (s *CircuitBreakerService) GetCircuitState(ctx context.Context, workflowID string) (*domain.WorkflowCircuitBreaker, error) {
	return s.circuits.GetCircuitState(ctx, workflowID)
}

func (s *CircuitBreakerService) ListCircuitStates(ctx context.Context) ([]*domain.WorkflowCircuitBreaker, error) {
	return s.circuits.ListCircuitStates(ctx)
}
