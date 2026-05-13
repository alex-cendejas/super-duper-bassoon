package ports

import (
	"context"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
)

// WorkflowRepository persists workflow definitions.
type WorkflowRepository interface {
	SaveWorkflow(ctx context.Context, w *domain.Workflow) error
	GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error)
	ListAllWorkflows(ctx context.Context) ([]*domain.Workflow, error)
	ListActiveWorkflows(ctx context.Context) ([]*domain.Workflow, error)
	ListWorkflowsByType(ctx context.Context, workflowType string) ([]*domain.Workflow, error)
	UpdateWorkflowState(ctx context.Context, id string, active bool, reason string) error
	DeleteWorkflow(ctx context.Context, id string) error
}

// ClientRepository accesses client metadata.
type ClientRepository interface {
	SaveClient(ctx context.Context, c *domain.ClientMetadata) error
	GetClientByID(ctx context.Context, id string) (*domain.ClientMetadata, error)
	ListClients(ctx context.Context) ([]*domain.ClientMetadata, error)
	GetClientsByIDs(ctx context.Context, ids []string) ([]*domain.ClientMetadata, error)
}

// RunRepository persists run state.
type RunRepository interface {
	CreateRun(ctx context.Context, r *domain.Run) error
	GetRun(ctx context.Context, runID string) (*domain.Run, error)
	UpdateRun(ctx context.Context, r *domain.Run) error
	ListRuns(ctx context.Context, workflowID string, limit int) ([]*domain.Run, error)
	ListAllRuns(ctx context.Context, limit, offset int) ([]*domain.Run, error)
	ListRunsByWorkflowType(ctx context.Context, workflowType string, limit int) ([]*domain.Run, error)
	GetPreviousRun(ctx context.Context, clientID, workflowType, currentRunID string, before time.Time) (*domain.Run, error)
}

// ResultRepository persists results.
type ResultRepository interface {
	SaveResult(ctx context.Context, r *domain.Result) error
	GetResult(ctx context.Context, runID, clientID string) (*domain.Result, error)
	GetRunResults(ctx context.Context, runID string) ([]*domain.Result, error)
	ListClientResults(ctx context.Context, clientID, workflowType string, limit int) ([]*domain.Result, error)
}

// BanRepository persists bans.
type BanRepository interface {
	SaveBan(ctx context.Context, b *domain.BanRecord) error
	GetBans(ctx context.Context, clientID string) ([]*domain.BanRecord, error)
	GetActiveBans(ctx context.Context, clientID string) ([]*domain.BanRecord, error)
	GetActiveBansByWorkflowType(ctx context.Context, workflowType string) ([]*domain.BanRecord, error)
	UnbanClient(ctx context.Context, clientID, workflowType string) error
	ListAllBans(ctx context.Context) ([]*domain.BanRecord, error)
}

// HealthRepository persists health calculations.
type HealthRepository interface {
	SaveRunHealth(ctx context.Context, h *domain.RunHealth) error
	GetRunHealth(ctx context.Context, runID string) (*domain.RunHealth, error)
	ListRunHealths(ctx context.Context, workflowType string, limit int) ([]*domain.RunHealth, error)
	SaveWorkflowTypeHealth(ctx context.Context, h *domain.WorkflowTypeHealth) error
	GetWorkflowTypeHealth(ctx context.Context, workflowType string) (*domain.WorkflowTypeHealth, error)
	ListAllWorkflowTypeHealths(ctx context.Context) ([]*domain.WorkflowTypeHealth, error)
}

// CircuitBreakerStateRepository persists circuit states.
type CircuitBreakerStateRepository interface {
	SaveCircuitState(ctx context.Context, s *domain.WorkflowCircuitBreaker) error
	GetCircuitState(ctx context.Context, workflowID string) (*domain.WorkflowCircuitBreaker, error)
	ListCircuitStates(ctx context.Context) ([]*domain.WorkflowCircuitBreaker, error)
}

// MessageDispatcher publishes/sends dispatch messages and subscribes to results.
type MessageDispatcher interface {
	SendDispatch(ctx context.Context, d *domain.Dispatch) error
	SendBatchDispatches(ctx context.Context, list []*domain.Dispatch) error
}

// EventBus is in-process pub/sub.
type EventBus interface {
	Publish(ctx context.Context, event domain.Event) error
	Subscribe(eventType string, handler EventHandler) error
}

type EventHandler func(ctx context.Context, event domain.Event) error

// AlertPublisher sends alerts.
type AlertPublisher interface {
	PublishAlert(ctx context.Context, alert *domain.Alert) error
}

// DispatchBlocker is the in-memory ban cache.
type DispatchBlocker interface {
	ShouldBlockDispatch(ctx context.Context, clientID, workflowType string) bool
	Add(ban *domain.BanRecord)
	Remove(clientID, workflowType string)
}

// ResultHandler is called by ResultMessageDispatcher.
type ResultHandler interface {
	HandleResult(ctx context.Context, r *domain.Result) error
	Priority() int
	Name() string
}

// AggregationRepo for circuit breaker policy
type PolicyRepository interface {
	GetPolicy(ctx context.Context, workflowID string) (*domain.CircuitBreakerPolicy, error)
	GetDefaultPolicy(ctx context.Context) (*domain.CircuitBreakerPolicy, error)
}

type ConfigRepository interface {
	GetHealthThreshold(ctx context.Context, workflowType string) (*domain.HealthThreshold, error)
}

type WorkflowStateManager interface {
	DeactivateWorkflow(ctx context.Context, workflowID, reason string) error
	ActivateWorkflow(ctx context.Context, workflowID, reason string) error
	IsWorkflowActive(ctx context.Context, workflowID string) bool
}
