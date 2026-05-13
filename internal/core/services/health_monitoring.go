package services

import (
	"context"
	"log"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
)

type HealthMonitoringService struct {
	runs       ports.RunRepository
	results    ports.ResultRepository
	bans       ports.BanRepository
	health     ports.HealthRepository
	workflows  ports.WorkflowRepository
	events     ports.EventBus
	config     ports.ConfigRepository
	aggregator *domain.HealthAggregator
	logger     *log.Logger
	windowSize int
}

func NewHealthMonitoringService(runs ports.RunRepository, results ports.ResultRepository, bans ports.BanRepository, health ports.HealthRepository, workflows ports.WorkflowRepository, events ports.EventBus, config ports.ConfigRepository, windowSize int, logger *log.Logger) *HealthMonitoringService {
	if logger == nil {
		logger = log.Default()
	}
	if windowSize <= 0 {
		windowSize = 10
	}
	return &HealthMonitoringService{
		runs:       runs,
		results:    results,
		bans:       bans,
		health:     health,
		workflows:  workflows,
		events:     events,
		config:     config,
		aggregator: domain.NewHealthAggregator(),
		windowSize: windowSize,
		logger:     logger,
	}
}

func (s *HealthMonitoringService) Name() string  { return "health_monitoring" }
func (s *HealthMonitoringService) Priority() int { return 2 }

// HandleResult is called by ResultMessageDispatcher. It persists the result and recalculates run health.
func (s *HealthMonitoringService) HandleResult(ctx context.Context, r *domain.Result) error {
	if r == nil || !r.IsValid() {
		return nil
	}
	// Save result (idempotent)
	wf, _ := s.workflows.GetWorkflow(ctx, r.WorkflowID)
	wfType := ""
	if wf != nil {
		wfType = wf.WorkflowType
	}
	if saver, ok := s.results.(interface {
		SaveResultWithType(ctx context.Context, r *domain.Result, workflowType string) error
	}); ok {
		_ = saver.SaveResultWithType(ctx, r, wfType)
	} else {
		_ = s.results.SaveResult(ctx, r)
	}
	_, err := s.CalculateRunHealth(ctx, r.RunID)
	return err
}

func (s *HealthMonitoringService) CalculateRunHealth(ctx context.Context, runID string) (*domain.RunHealth, error) {
	run, err := s.runs.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	results, err := s.results.GetRunResults(ctx, runID)
	if err != nil {
		return nil, err
	}
	bannedClients, _ := s.bans.GetActiveBansByWorkflowType(ctx, run.WorkflowType)
	banSet := map[string]struct{}{}
	for _, b := range bannedClients {
		banSet[b.ClientID] = struct{}{}
	}
	bannedCount := 0
	for _, cid := range run.ParticipatingClients {
		if _, ok := banSet[cid]; ok {
			bannedCount++
		}
	}
	h := s.aggregator.CalculateRunHealth(runID, run.WorkflowID, run.WorkflowType, len(run.ParticipatingClients), results, bannedCount)
	if err := s.health.SaveRunHealth(ctx, h); err != nil {
		return nil, err
	}
	// Re-aggregate type health
	typeHealth, _ := s.AggregateWorkflowTypeHealth(ctx, run.WorkflowType)
	if s.events != nil {
		_ = s.events.Publish(ctx, &domain.HealthUpdatedEvent{
			WorkflowID:   run.WorkflowID,
			WorkflowType: run.WorkflowType,
			RunHealth:    h,
			TypeHealth:   typeHealth,
			Timestamp:    time.Now().UTC(),
		})
	}
	// If all results in
	if h.IsComplete() && run.State != domain.RunCompleted {
		run.State = domain.RunCompleted
		if updErr := s.runs.UpdateRun(ctx, run); updErr != nil {
			s.logger.Printf("failed to mark run completed: %v", updErr)
		}
	}
	return h, nil
}

func (s *HealthMonitoringService) AggregateWorkflowTypeHealth(ctx context.Context, workflowType string) (*domain.WorkflowTypeHealth, error) {
	window := s.windowSize
	if s.config != nil {
		if t, err := s.config.GetHealthThreshold(ctx, workflowType); err == nil && t.WindowSize > 0 {
			window = t.WindowSize
		}
	}
	runs, err := s.health.ListRunHealths(ctx, workflowType, window)
	if err != nil {
		return nil, err
	}
	out := s.aggregator.AggregateWorkflowHealth(workflowType, runs, window)
	if err := s.health.SaveWorkflowTypeHealth(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *HealthMonitoringService) GetCurrentHealth(ctx context.Context, workflowType string) (*domain.WorkflowTypeHealth, error) {
	return s.health.GetWorkflowTypeHealth(ctx, workflowType)
}
