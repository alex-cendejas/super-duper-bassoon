package services

import (
	"context"
	"testing"
	"time"

	"github.com/super-duper-bassoon/internal/adapters/trigger"
	"github.com/super-duper-bassoon/internal/core/domain"
)

func newOrch(t *testing.T) (*WorkflowOrchestrationService, *memoryWorkflowRepo, *memoryRunRepo, *memoryClientRepo, *stubEventBus) {
	t.Helper()
	wfs := newMemWorkflowRepo()
	runs := newMemRunRepo()
	clients := newMemClientRepo()
	dispatcher := newStubDispatcher()
	bans := newMemBanRepo()
	enf := NewBanEnforcementService(bans, newStubAlerter(), newStubBlocker(), nil)
	filter := NewDispatchFilterService(enf, nil)
	coord := NewDispatchCoordinationService(runs, dispatcher, clients, filter, nil)
	group := NewDynamicGroupingService(clients)
	bus := newStubEventBus()
	orch := NewWorkflowOrchestrationService(wfs, runs, clients, group, coord, bus, nil)
	return orch, wfs, runs, clients, bus
}

func TestTriggerCoordination_Scheduled(t *testing.T) {
	orch, wfs, runs, clients, bus := newOrch(t)
	ctx := context.Background()
	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "a", OS: "linux", Active: true})
	wf, _ := orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "n", WorkflowType: "t", Activity: domain.ActivityReboot})
	// Add scheduled trigger
	wf.Trigger = domain.TriggerSpec{Kind: domain.TriggerScheduled, Cron: "* * * * *"}
	_ = wfs.SaveWorkflow(ctx, wf)

	cron := trigger.NewCronEvaluator()
	// Force a "fire" by manually setting and re-evaluating
	tc := NewTriggerCoordinationService(wfs, orch, cron, bus, 60000, nil)
	// First evaluate seeds, doesn't fire
	tc.evaluate(ctx, time.Now())
	// Force scheduled fire by evaluating well in the future
	tc.evaluate(ctx, time.Now().Add(2*time.Minute))
	got, _ := runs.ListRuns(ctx, wf.ID, 10)
	if len(got) != 1 {
		t.Errorf("expected 1 run from scheduled trigger, got %d", len(got))
	}
}

func TestTriggerCoordination_EventDriven(t *testing.T) {
	orch, wfs, runs, clients, bus := newOrch(t)
	ctx := context.Background()
	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "a", OS: "linux", Active: true})

	wfA, _ := orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "A", WorkflowType: "a", Activity: domain.ActivityReboot})
	wfB, _ := orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "B", WorkflowType: "b", Activity: domain.ActivityReboot})
	wfB.Trigger = domain.TriggerSpec{Kind: domain.TriggerEvent, OnComplete: wfA.ID}
	_ = wfs.SaveWorkflow(ctx, wfB)

	tc := NewTriggerCoordinationService(wfs, orch, nil, bus, 60000, nil)
	tc.Start(ctx)
	defer tc.Stop()

	// Trigger wfA which publishes workflow.completed
	_, err := orch.TriggerWorkflow(ctx, wfA.ID, "kick")
	if err != nil {
		t.Fatal(err)
	}
	// Give the handler a brief moment
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		list, _ := runs.ListRuns(ctx, wfB.ID, 10)
		if len(list) > 0 {
			return
		}
	}
	t.Error("wfB never triggered")
}

func TestTriggerCoordination_StartStop(t *testing.T) {
	orch, wfs, _, _, bus := newOrch(t)
	tc := NewTriggerCoordinationService(wfs, orch, nil, bus, 10, nil) // short tick
	ctx := context.Background()
	tc.Start(ctx)
	// Let the ticker fire
	time.Sleep(50 * time.Millisecond)
	tc.Stop()
}
