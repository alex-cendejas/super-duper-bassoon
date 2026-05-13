package services

import (
	"context"
	"testing"
	"time"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

func setupHealthSvc(t *testing.T) (*HealthMonitoringService, *memoryRunRepo, *memoryResultRepo, *memoryBanRepo, *memoryHealthRepo, *memoryWorkflowRepo, *stubEventBus) {
	t.Helper()
	runs := newMemRunRepo()
	results := newMemResultRepo()
	bans := newMemBanRepo()
	health := newMemHealthRepo()
	wfs := newMemWorkflowRepo()
	bus := newStubEventBus()
	cfg := NewDefaultConfigRepository(wfs, 80, 5)
	svc := NewHealthMonitoringService(runs, results, bans, health, wfs, bus, cfg, 5, nil)
	return svc, runs, results, bans, health, wfs, bus
}

func TestHealthMonitoring_CalculateRunHealth(t *testing.T) {
	svc, runs, results, _, health, wfs, bus := setupHealthSvc(t)
	ctx := context.Background()
	wf := &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80}
	_ = wfs.SaveWorkflow(ctx, wf)
	run := &domain.Run{RunID: "r1", WorkflowID: "w", WorkflowType: "t", TriggeredAt: time.Now(), ParticipatingClients: []string{"a", "b"}, State: domain.RunInProgress}
	_ = runs.CreateRun(ctx, run)

	_ = results.SaveResult(ctx, &domain.Result{RunID: "r1", ClientID: "a", WorkflowID: "w", Status: domain.StatusSuccess})
	_ = results.SaveResult(ctx, &domain.Result{RunID: "r1", ClientID: "b", WorkflowID: "w", Status: domain.StatusFail})

	h, err := svc.CalculateRunHealth(ctx, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if h.SuccessCount != 1 || h.FailCount != 1 {
		t.Errorf("counts: %+v", h)
	}
	if h.PendingCount != 0 {
		t.Errorf("pending: %d", h.PendingCount)
	}
	stored, _ := health.GetRunHealth(ctx, "r1")
	if stored == nil {
		t.Error("not persisted")
	}
	// Event was published
	if len(bus.Events) == 0 {
		t.Error("expected event")
	}
	updated, _ := runs.GetRun(ctx, "r1")
	if updated.State != domain.RunCompleted {
		t.Errorf("expected completed, got %v", updated.State)
	}
}

func TestHealthMonitoring_HandleResult(t *testing.T) {
	svc, runs, _, _, _, wfs, _ := setupHealthSvc(t)
	ctx := context.Background()
	wf := &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80}
	_ = wfs.SaveWorkflow(ctx, wf)
	run := &domain.Run{RunID: "r1", WorkflowID: "w", WorkflowType: "t", TriggeredAt: time.Now(), ParticipatingClients: []string{"a"}, State: domain.RunInProgress}
	_ = runs.CreateRun(ctx, run)

	// Invalid result is ignored
	if err := svc.HandleResult(ctx, &domain.Result{}); err != nil {
		t.Error("expected nil")
	}

	r := &domain.Result{RunID: "r1", WorkflowID: "w", ClientID: "a", Status: domain.StatusSuccess}
	if err := svc.HandleResult(ctx, r); err != nil {
		t.Fatal(err)
	}
	if svc.Name() != "health_monitoring" {
		t.Error("name")
	}
	if svc.Priority() != 2 {
		t.Error("priority")
	}
}

func TestHealthMonitoring_BannedExcluded(t *testing.T) {
	svc, runs, results, bans, _, wfs, _ := setupHealthSvc(t)
	ctx := context.Background()
	wf := &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80}
	_ = wfs.SaveWorkflow(ctx, wf)
	run := &domain.Run{RunID: "r1", WorkflowID: "w", WorkflowType: "t", TriggeredAt: time.Now(), ParticipatingClients: []string{"a", "b", "c"}, State: domain.RunInProgress}
	_ = runs.CreateRun(ctx, run)

	_ = bans.SaveBan(ctx, &domain.BanRecord{ClientID: "c", WorkflowType: "t", Active: true})
	_ = results.SaveResult(ctx, &domain.Result{RunID: "r1", ClientID: "a", WorkflowID: "w", Status: domain.StatusSuccess})
	_ = results.SaveResult(ctx, &domain.Result{RunID: "r1", ClientID: "b", WorkflowID: "w", Status: domain.StatusSuccess})

	h, err := svc.CalculateRunHealth(ctx, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if h.BannedClientCount != 1 {
		t.Errorf("banned: %d", h.BannedClientCount)
	}
	if h.SuccessPercentage() != 100 {
		t.Errorf("expected 100%% (excluding banned): %v", h.SuccessPercentage())
	}
}

func TestHealthMonitoring_AggregateAndGet(t *testing.T) {
	svc, _, _, _, health, wfs, _ := setupHealthSvc(t)
	ctx := context.Background()
	_ = wfs.SaveWorkflow(ctx, &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80})
	_ = health.SaveRunHealth(ctx, &domain.RunHealth{RunID: "r1", WorkflowID: "w", WorkflowType: "t", TotalClients: 2, SuccessCount: 2, CalculatedAt: time.Now().Add(-2 * time.Second)})
	_ = health.SaveRunHealth(ctx, &domain.RunHealth{RunID: "r2", WorkflowID: "w", WorkflowType: "t", TotalClients: 2, FailCount: 2, CalculatedAt: time.Now()})

	out, err := svc.AggregateWorkflowTypeHealth(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}
	if out.RunsConsidered != 2 {
		t.Errorf("runs: %d", out.RunsConsidered)
	}
	if out.SuccessPercentageAvg != 50 {
		t.Errorf("avg: %v", out.SuccessPercentageAvg)
	}
	cur, err := svc.GetCurrentHealth(ctx, "t")
	if err != nil {
		t.Fatal(err)
	}
	if cur.WorkflowType != "t" {
		t.Error("workflow type")
	}
}
