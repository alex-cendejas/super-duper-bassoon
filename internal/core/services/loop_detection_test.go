package services

import (
	"context"
	"testing"
	"time"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

func TestLoopDetectionService_DetectAndBan(t *testing.T) {
	runs := newMemRunRepo()
	wfs := newMemWorkflowRepo()
	bans := newMemBanRepo()
	blocker := newStubBlocker()
	alerter := newStubAlerter()
	enf := NewBanEnforcementService(bans, alerter, blocker, nil)
	svc := NewLoopDetectionService(runs, wfs, bans, enf, alerter, 5000, nil)

	ctx := context.Background()
	wf := &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80, LoopThresholdMS: 5000}
	_ = wfs.SaveWorkflow(ctx, wf)

	prev := &domain.Run{RunID: "r1", WorkflowID: "w", WorkflowType: "t", TriggeredAt: time.Now().Add(-time.Second), ParticipatingClients: []string{"c1"}, State: domain.RunInProgress}
	_ = runs.CreateRun(ctx, prev)
	cur := &domain.Run{RunID: "r2", WorkflowID: "w", WorkflowType: "t", TriggeredAt: time.Now(), ParticipatingClients: []string{"c1"}, State: domain.RunInProgress}
	_ = runs.CreateRun(ctx, cur)

	res := &domain.Result{RunID: "r2", WorkflowID: "w", ClientID: "c1", Status: domain.StatusFail}
	if err := svc.HandleResult(ctx, res); err != nil {
		t.Fatal(err)
	}
	bansFor, _ := bans.GetActiveBans(ctx, "c1")
	if len(bansFor) != 1 {
		t.Errorf("expected 1 ban: %d", len(bansFor))
	}
	if alerter.Count(domain.AlertLoopDetected) != 1 {
		t.Error("expected loop alert")
	}
	if alerter.Count(domain.AlertClientBanned) != 1 {
		t.Error("expected ban alert")
	}
	if svc.Priority() != 1 {
		t.Error("priority")
	}
	if svc.Name() != "loop_detection" {
		t.Error("name")
	}
}

func TestLoopDetectionService_NoPrev_NoBan(t *testing.T) {
	runs := newMemRunRepo()
	wfs := newMemWorkflowRepo()
	bans := newMemBanRepo()
	blocker := newStubBlocker()
	alerter := newStubAlerter()
	enf := NewBanEnforcementService(bans, alerter, blocker, nil)
	svc := NewLoopDetectionService(runs, wfs, bans, enf, alerter, 5000, nil)

	ctx := context.Background()
	wf := &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80, LoopThresholdMS: 5000}
	_ = wfs.SaveWorkflow(ctx, wf)
	cur := &domain.Run{RunID: "r1", WorkflowID: "w", WorkflowType: "t", TriggeredAt: time.Now(), ParticipatingClients: []string{"c1"}, State: domain.RunInProgress}
	_ = runs.CreateRun(ctx, cur)

	res := &domain.Result{RunID: "r1", WorkflowID: "w", ClientID: "c1", Status: domain.StatusSuccess}
	_ = svc.HandleResult(ctx, res)
	all, _ := bans.ListAllBans(ctx)
	if len(all) != 0 {
		t.Errorf("expected no bans")
	}
}

func TestLoopDetectionService_OutOfWindow(t *testing.T) {
	runs := newMemRunRepo()
	wfs := newMemWorkflowRepo()
	bans := newMemBanRepo()
	enf := NewBanEnforcementService(bans, newStubAlerter(), newStubBlocker(), nil)
	svc := NewLoopDetectionService(runs, wfs, bans, enf, newStubAlerter(), 1, nil) // 1ms threshold

	ctx := context.Background()
	wf := &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, Active: true, Name: "n", SuccessThreshold: 80, LoopThresholdMS: 1}
	_ = wfs.SaveWorkflow(ctx, wf)
	prev := &domain.Run{RunID: "r1", WorkflowID: "w", WorkflowType: "t", TriggeredAt: time.Now().Add(-time.Hour), ParticipatingClients: []string{"c1"}, State: domain.RunInProgress}
	_ = runs.CreateRun(ctx, prev)
	cur := &domain.Run{RunID: "r2", WorkflowID: "w", WorkflowType: "t", TriggeredAt: time.Now(), ParticipatingClients: []string{"c1"}, State: domain.RunInProgress}
	_ = runs.CreateRun(ctx, cur)
	_ = svc.HandleResult(ctx, &domain.Result{RunID: "r2", WorkflowID: "w", ClientID: "c1", Status: domain.StatusSuccess})
	all, _ := bans.ListAllBans(ctx)
	if len(all) != 0 {
		t.Error("expected no bans (out of window)")
	}
}

func TestLoopDetectionService_InvalidAndMissingWorkflow(t *testing.T) {
	runs := newMemRunRepo()
	wfs := newMemWorkflowRepo()
	bans := newMemBanRepo()
	enf := NewBanEnforcementService(bans, newStubAlerter(), newStubBlocker(), nil)
	svc := NewLoopDetectionService(runs, wfs, bans, enf, newStubAlerter(), 5000, nil)
	ctx := context.Background()
	// Invalid result -> nil
	if err := svc.HandleResult(ctx, &domain.Result{}); err != nil {
		t.Error(err)
	}
	// nil result
	if err := svc.HandleResult(ctx, nil); err != nil {
		t.Error(err)
	}
	// Workflow missing -> no error, no ban
	res := &domain.Result{RunID: "r", WorkflowID: "missing", ClientID: "c", Status: domain.StatusSuccess}
	_ = svc.HandleResult(ctx, res)
}
