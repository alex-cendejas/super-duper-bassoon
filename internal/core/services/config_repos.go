package services

import (
	"context"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
)

// DefaultConfigRepository pulls per-workflow_type thresholds from the workflow repository
// using the first matching workflow's settings, with fallback to defaults.
type DefaultConfigRepository struct {
	workflows        ports.WorkflowRepository
	defaultThreshold float64
	defaultWindow    int
}

func NewDefaultConfigRepository(workflows ports.WorkflowRepository, defaultThreshold float64, defaultWindow int) *DefaultConfigRepository {
	if defaultThreshold <= 0 {
		defaultThreshold = 80
	}
	if defaultWindow <= 0 {
		defaultWindow = 10
	}
	return &DefaultConfigRepository{workflows: workflows, defaultThreshold: defaultThreshold, defaultWindow: defaultWindow}
}

func (c *DefaultConfigRepository) GetHealthThreshold(ctx context.Context, workflowType string) (*domain.HealthThreshold, error) {
	t := &domain.HealthThreshold{SuccessThreshold: c.defaultThreshold, WindowSize: c.defaultWindow}
	wfs, err := c.workflows.ListWorkflowsByType(ctx, workflowType)
	if err == nil && len(wfs) > 0 {
		wf := wfs[0]
		if wf.SuccessThreshold > 0 {
			t.SuccessThreshold = wf.SuccessThreshold
		}
	}
	return t, nil
}

// DefaultPolicyRepo provides circuit breaker policies from workflows.
type DefaultPolicyRepo struct {
	workflows        ports.WorkflowRepository
	defaultThreshold float64
	defaultWindow    int
	defaultCooldown  time.Duration
}

func NewDefaultPolicyRepo(workflows ports.WorkflowRepository, defaultThreshold float64, defaultWindow int, defaultCooldownMS int64) *DefaultPolicyRepo {
	if defaultThreshold <= 0 {
		defaultThreshold = 80
	}
	if defaultWindow <= 0 {
		defaultWindow = 10
	}
	cooldown := time.Duration(defaultCooldownMS) * time.Millisecond
	if cooldown <= 0 {
		cooldown = 5 * time.Minute
	}
	return &DefaultPolicyRepo{workflows: workflows, defaultThreshold: defaultThreshold, defaultWindow: defaultWindow, defaultCooldown: cooldown}
}

func (p *DefaultPolicyRepo) GetPolicy(ctx context.Context, workflowID string) (*domain.CircuitBreakerPolicy, error) {
	wf, err := p.workflows.GetWorkflow(ctx, workflowID)
	if err != nil {
		return p.GetDefaultPolicy(ctx)
	}
	threshold := p.defaultThreshold
	if wf.SuccessThreshold > 0 {
		threshold = wf.SuccessThreshold
	}
	return &domain.CircuitBreakerPolicy{
		SuccessThreshold: threshold,
		EvaluationWindow: p.defaultWindow,
		CooldownPeriod:   p.defaultCooldown,
	}, nil
}

func (p *DefaultPolicyRepo) GetDefaultPolicy(ctx context.Context) (*domain.CircuitBreakerPolicy, error) {
	return &domain.CircuitBreakerPolicy{
		SuccessThreshold: p.defaultThreshold,
		EvaluationWindow: p.defaultWindow,
		CooldownPeriod:   p.defaultCooldown,
	}, nil
}
