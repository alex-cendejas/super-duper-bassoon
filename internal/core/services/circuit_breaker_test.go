package services

import (
	"context"
	"testing"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
)

func setupCircuitSvc(t *testing.T) (*CircuitBreakerService, *memoryWorkflowRepo, *memoryHealthRepo, *memoryCircuitRepo, *stubAlerter, *stubEventBus) {
	t.Helper()
	wfs := newMemWorkflowRepo()
	healthRepo := newMemHealthRepo()
	circuits := newMemCircuitRepo()
	policy := NewDefaultPolicyRepo(wfs, 80, 5, 60000)
	stateMgr := NewWorkflowStateManager(wfs)
	alerter := newStubAlerter()
	bus := newStubEventBus()
	pol := &domain.CircuitBreakerPolicy{SuccessThreshold: 80, EvaluationWindow: 5, CooldownPeriod: time.Minute}
	svc := NewCircuitBreakerService(healthRepo, circuits, policy, wfs, stateMgr, alerter, bus, pol, nil)
	return svc, wfs, healthRepo, circuits, alerter, bus
}

func TestCircuitBreaker_OpensOnUnhealthy(t *testing.T) {
	svc, wfs, healthRepo, circuits, alerter, bus := setupCircuitSvc(t)
	ctx := context.Background()
	wf := &domain.Workflow{ID: "w1", WorkflowType: "t1", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80}
	_ = wfs.SaveWorkflow(ctx, wf)
	_ = healthRepo.SaveWorkflowTypeHealth(ctx, &domain.WorkflowTypeHealth{WorkflowType: "t1", RunsConsidered: 5, SuccessPercentageAvg: 30})

	if err := svc.EvaluateWorkflowType(ctx, "t1", nil); err != nil {
		t.Fatal(err)
	}
	state, err := circuits.GetCircuitState(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	if state.State != domain.CircuitOpen {
		t.Errorf("expected open, got %v", state.State)
	}
	// Workflow deactivated
	got, _ := wfs.GetWorkflow(ctx, "w1")
	if got.Active {
		t.Error("expected deactivated")
	}
	if alerter.Count(domain.AlertCircuitOpened) != 1 {
		t.Error("expected open alert")
	}
	if len(bus.Events) == 0 {
		t.Error("expected event")
	}

	// Re-evaluate with healthy data => closes
	_ = healthRepo.SaveWorkflowTypeHealth(ctx, &domain.WorkflowTypeHealth{WorkflowType: "t1", RunsConsidered: 5, SuccessPercentageAvg: 95})
	if err := svc.EvaluateWorkflowType(ctx, "t1", nil); err != nil {
		t.Fatal(err)
	}
	state, _ = circuits.GetCircuitState(ctx, "w1")
	if state.State != domain.CircuitClosed {
		t.Errorf("expected closed, got %v", state.State)
	}
	if alerter.Count(domain.AlertCircuitClosed) != 1 {
		t.Error("expected closed alert")
	}
	got, _ = wfs.GetWorkflow(ctx, "w1")
	if !got.Active {
		t.Error("expected reactivated")
	}
}

func TestCircuitBreaker_OnHealthUpdatedEvent(t *testing.T) {
	svc, wfs, _, circuits, _, _ := setupCircuitSvc(t)
	ctx := context.Background()
	wf := &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80}
	_ = wfs.SaveWorkflow(ctx, wf)
	evt := &domain.HealthUpdatedEvent{
		WorkflowID:   "w",
		WorkflowType: "t",
		TypeHealth:   &domain.WorkflowTypeHealth{WorkflowType: "t", RunsConsidered: 3, SuccessPercentageAvg: 10},
	}
	if err := svc.OnHealthUpdatedEvent(ctx, evt); err != nil {
		t.Fatal(err)
	}
	state, _ := circuits.GetCircuitState(ctx, "w")
	if state.State != domain.CircuitOpen {
		t.Errorf("expected open, got %v", state.State)
	}
}

func TestCircuitBreaker_EvaluateAllWorkflows(t *testing.T) {
	svc, wfs, healthRepo, _, _, _ := setupCircuitSvc(t)
	ctx := context.Background()
	_ = wfs.SaveWorkflow(ctx, &domain.Workflow{ID: "w1", WorkflowType: "t1", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80})
	_ = wfs.SaveWorkflow(ctx, &domain.Workflow{ID: "w2", WorkflowType: "t2", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80})
	_ = healthRepo.SaveWorkflowTypeHealth(ctx, &domain.WorkflowTypeHealth{WorkflowType: "t1", RunsConsidered: 5, SuccessPercentageAvg: 90})
	_ = healthRepo.SaveWorkflowTypeHealth(ctx, &domain.WorkflowTypeHealth{WorkflowType: "t2", RunsConsidered: 5, SuccessPercentageAvg: 20})
	if err := svc.EvaluateAllWorkflows(ctx); err != nil {
		t.Fatal(err)
	}
	list, _ := svc.ListCircuitStates(ctx)
	if len(list) != 2 {
		t.Errorf("expected 2 circuit states: %d", len(list))
	}
}

func TestCircuitBreaker_GetState_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := setupCircuitSvc(t)
	ctx := context.Background()
	if _, err := svc.GetCircuitState(ctx, "missing"); err == nil {
		t.Error("expected error")
	}
}
