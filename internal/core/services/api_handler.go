package services

import (
	"context"
	"runtime"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
)

// APIHandlerService is a thin coordinator that the HTTP layer calls. It does not
// build HTTP responses directly; it returns domain values or APIError.
type APIHandlerService struct {
	workflows ports.WorkflowRepository
	orch      *WorkflowOrchestrationService
	clients   ports.ClientRepository
	runs      ports.RunRepository
	results   ports.ResultRepository
	bans      ports.BanRepository
	banEnf    *BanEnforcementService
	circuits  *CircuitBreakerService
	health    *HealthMonitoringService
	startedAt time.Time
}

func NewAPIHandlerService(workflows ports.WorkflowRepository, orch *WorkflowOrchestrationService, clients ports.ClientRepository, runs ports.RunRepository, results ports.ResultRepository, bans ports.BanRepository, banEnf *BanEnforcementService, circuits *CircuitBreakerService, health *HealthMonitoringService) *APIHandlerService {
	return &APIHandlerService{
		workflows: workflows,
		orch:      orch,
		clients:   clients,
		runs:      runs,
		results:   results,
		bans:      bans,
		banEnf:    banEnf,
		circuits:  circuits,
		health:    health,
		startedAt: time.Now().UTC(),
	}
}

func (a *APIHandlerService) CreateWorkflow(ctx context.Context, req *domain.CreateWorkflowRequest) (*domain.Workflow, error) {
	return a.orch.CreateWorkflow(ctx, req)
}

func (a *APIHandlerService) EditWorkflow(ctx context.Context, id string, req *domain.EditWorkflowRequest) (*domain.Workflow, error) {
	return a.orch.EditWorkflow(ctx, id, req)
}

func (a *APIHandlerService) DeleteWorkflow(ctx context.Context, id string) error {
	return a.orch.DeleteWorkflow(ctx, id)
}

func (a *APIHandlerService) ActivateWorkflow(ctx context.Context, id string) (*domain.Workflow, error) {
	if err := a.orch.ActivateWorkflow(ctx, id); err != nil {
		return nil, err
	}
	return a.workflows.GetWorkflow(ctx, id)
}

func (a *APIHandlerService) DeactivateWorkflow(ctx context.Context, id string) (*domain.Workflow, error) {
	if err := a.orch.DeactivateWorkflow(ctx, id); err != nil {
		return nil, err
	}
	return a.workflows.GetWorkflow(ctx, id)
}

func (a *APIHandlerService) GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error) {
	return a.workflows.GetWorkflow(ctx, id)
}

func (a *APIHandlerService) ListWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	return a.workflows.ListAllWorkflows(ctx)
}

func (a *APIHandlerService) TriggerWorkflow(ctx context.Context, id, reason string) (*domain.Run, error) {
	return a.orch.TriggerWorkflow(ctx, id, reason)
}

func (a *APIHandlerService) GetClient(ctx context.Context, id string) (*domain.ClientMetadata, error) {
	return a.clients.GetClientByID(ctx, id)
}

func (a *APIHandlerService) ListClients(ctx context.Context) ([]*domain.ClientMetadata, error) {
	return a.clients.ListClients(ctx)
}

func (a *APIHandlerService) GetRun(ctx context.Context, id string) (*domain.Run, error) {
	return a.runs.GetRun(ctx, id)
}

func (a *APIHandlerService) ListRuns(ctx context.Context, workflowID string, limit int) ([]*domain.Run, error) {
	return a.runs.ListRuns(ctx, workflowID, limit)
}

func (a *APIHandlerService) GetRunResults(ctx context.Context, runID string) ([]*domain.Result, error) {
	return a.results.GetRunResults(ctx, runID)
}

func (a *APIHandlerService) GetHealth(ctx context.Context, workflowType string) (*domain.WorkflowTypeHealth, error) {
	return a.health.GetCurrentHealth(ctx, workflowType)
}

// ListAllHealth is implemented in the HTTP layer using HealthRepo directly.

func (a *APIHandlerService) GetBans(ctx context.Context, clientID string) ([]*domain.BanRecord, error) {
	return a.bans.GetBans(ctx, clientID)
}

func (a *APIHandlerService) ListAllBans(ctx context.Context) ([]*domain.BanRecord, error) {
	return a.bans.ListAllBans(ctx)
}

func (a *APIHandlerService) UnbanClient(ctx context.Context, clientID string, req *domain.UnbanRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}
	return a.banEnf.UnbanClient(ctx, clientID, req.WorkflowType, req.AdminID, req.Reason)
}

func (a *APIHandlerService) GetCircuitState(ctx context.Context, workflowID string) (*domain.WorkflowCircuitBreaker, error) {
	return a.circuits.GetCircuitState(ctx, workflowID)
}

func (a *APIHandlerService) ListCircuitStates(ctx context.Context) ([]*domain.WorkflowCircuitBreaker, error) {
	return a.circuits.ListCircuitStates(ctx)
}

func (a *APIHandlerService) SystemStatus(ctx context.Context, natsConnected bool, dbHealthy bool) *domain.SystemStatus {
	uptime := time.Since(a.startedAt)
	dbStatus := "healthy"
	if !dbHealthy {
		dbStatus = "unhealthy"
	}
	natsStatus := "connected"
	if !natsConnected {
		natsStatus = "disconnected"
	}
	return &domain.SystemStatus{
		Uptime:     uptime,
		UptimeSec:  int64(uptime.Seconds()),
		DBStatus:   dbStatus,
		NATSStatus: natsStatus,
		StartedAt:  a.startedAt,
		Goroutines: runtime.NumGoroutine(),
	}
}
