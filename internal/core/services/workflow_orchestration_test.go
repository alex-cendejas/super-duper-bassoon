package services

import (
	"context"
	"testing"
	"time"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

func setupOrch(t *testing.T) (*WorkflowOrchestrationService, *memoryWorkflowRepo, *memoryRunRepo, *memoryClientRepo, *stubDispatcher, *BanEnforcementService) {
	t.Helper()
	wfs := newMemWorkflowRepo()
	runs := newMemRunRepo()
	clients := newMemClientRepo()
	dispatcher := newStubDispatcher()
	bans := newMemBanRepo()
	blocker := newStubBlocker()
	enf := NewBanEnforcementService(bans, newStubAlerter(), blocker, nil)
	filter := NewDispatchFilterService(enf, nil)
	coord := NewDispatchCoordinationService(runs, dispatcher, clients, filter, nil)
	group := NewDynamicGroupingService(clients)
	orch := NewWorkflowOrchestrationService(wfs, runs, clients, group, coord, newStubEventBus(), nil)
	return orch, wfs, runs, clients, dispatcher, enf
}

func TestOrch_CreateWorkflow(t *testing.T) {
	orch, _, _, _, _, _ := setupOrch(t)
	ctx := context.Background()
	wf, err := orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "wf1", WorkflowType: "t", Activity: domain.ActivityReboot})
	if err != nil {
		t.Fatal(err)
	}
	if wf.ID == "" || !wf.Active {
		t.Errorf("bad: %+v", wf)
	}
	if wf.SuccessThreshold != 80 {
		t.Error("default success threshold")
	}

	// Invalid
	if _, err := orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{}); err == nil {
		t.Error("expected validation")
	}
}

func TestOrch_EditWorkflow(t *testing.T) {
	orch, wfs, _, _, _, _ := setupOrch(t)
	ctx := context.Background()
	wf, _ := orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "n1", WorkflowType: "t", Activity: domain.ActivityReboot})

	name := "new"
	enabled := false
	updated, err := orch.EditWorkflow(ctx, wf.ID, &domain.EditWorkflowRequest{Name: &name, Enabled: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "new" || updated.Active != false {
		t.Errorf("%+v", updated)
	}
	got, _ := wfs.GetWorkflow(ctx, wf.ID)
	if got.Name != "new" {
		t.Error("not persisted")
	}

	// Missing fields
	if _, err := orch.EditWorkflow(ctx, wf.ID, &domain.EditWorkflowRequest{}); err == nil {
		t.Error("expected validation")
	}
	// Missing workflow
	if _, err := orch.EditWorkflow(ctx, "missing", &domain.EditWorkflowRequest{Name: &name}); err == nil {
		t.Error("expected not found")
	}
}

func TestOrch_ActivateDeactivateDelete(t *testing.T) {
	orch, _, _, _, _, _ := setupOrch(t)
	ctx := context.Background()
	wf, _ := orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "n", WorkflowType: "t", Activity: domain.ActivityReboot})
	if err := orch.DeactivateWorkflow(ctx, wf.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := orch.GetWorkflow(ctx, wf.ID)
	if got.Active {
		t.Error("should be deactivated")
	}
	if err := orch.ActivateWorkflow(ctx, wf.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = orch.GetWorkflow(ctx, wf.ID)
	if !got.Active {
		t.Error("should be active")
	}
	if err := orch.DeleteWorkflow(ctx, wf.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := orch.GetWorkflow(ctx, wf.ID); err == nil {
		t.Error("expected not found")
	}
}

func TestOrch_TriggerWorkflow(t *testing.T) {
	orch, _, runs, clients, dispatcher, _ := setupOrch(t)
	ctx := context.Background()
	wf, _ := orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "n", WorkflowType: "t", Activity: domain.ActivityReboot, TargetFilter: "os == 'linux'"})

	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "a", OS: "linux", Active: true})
	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "b", OS: "darwin", Active: true})

	run, err := orch.TriggerWorkflow(ctx, wf.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(run.ParticipatingClients) != 1 || run.ParticipatingClients[0] != "a" {
		t.Errorf("clients: %v", run.ParticipatingClients)
	}
	if len(dispatcher.Sent) != 1 {
		t.Errorf("dispatches: %d", len(dispatcher.Sent))
	}
	got, _ := runs.GetRun(ctx, run.RunID)
	if got.State != domain.RunInProgress {
		t.Errorf("state: %v", got.State)
	}
	if got.DispatchedAt.IsZero() {
		t.Error("dispatched at not set")
	}
}

func TestOrch_TriggerWorkflow_InactiveBlocked(t *testing.T) {
	orch, _, _, clients, _, _ := setupOrch(t)
	ctx := context.Background()
	wf, _ := orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "n", WorkflowType: "t", Activity: domain.ActivityReboot})
	_ = orch.DeactivateWorkflow(ctx, wf.ID)
	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "a", OS: "linux", Active: true})
	if _, err := orch.TriggerWorkflow(ctx, wf.ID, "x"); err == nil {
		t.Error("expected inactive error")
	}
}

func TestOrch_TriggerWorkflow_BannedFiltered(t *testing.T) {
	orch, _, _, clients, dispatcher, enf := setupOrch(t)
	ctx := context.Background()
	wf, _ := orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "n", WorkflowType: "t", Activity: domain.ActivityReboot})
	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "a", OS: "linux", Active: true})
	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "b", OS: "linux", Active: true})
	_, _ = enf.BanClient(ctx, "b", "t", "r0", "ev", domain.ReasonLoopDetected)

	run, err := orch.TriggerWorkflow(ctx, wf.ID, "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(run.ParticipatingClients) != 1 || run.ParticipatingClients[0] != "a" {
		t.Errorf("expected only a, got: %v", run.ParticipatingClients)
	}
	if len(dispatcher.Sent) != 1 {
		t.Errorf("expected 1 dispatch, got %d", len(dispatcher.Sent))
	}
}

func TestOrch_ListAndGet(t *testing.T) {
	orch, _, _, _, _, _ := setupOrch(t)
	ctx := context.Background()
	_, _ = orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "n1", WorkflowType: "t", Activity: domain.ActivityReboot})
	_, _ = orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "n2", WorkflowType: "t", Activity: domain.ActivityReboot})
	list, err := orch.ListWorkflows(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("%d", len(list))
	}
}

func TestDispatchCoordination_Generate(t *testing.T) {
	runs := newMemRunRepo()
	clients := newMemClientRepo()
	dispatcher := newStubDispatcher()
	bans := newMemBanRepo()
	enf := NewBanEnforcementService(bans, newStubAlerter(), newStubBlocker(), nil)
	filter := NewDispatchFilterService(enf, nil)
	coord := NewDispatchCoordinationService(runs, dispatcher, clients, filter, nil)

	wf := &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, Params: map[string]interface{}{"k": "v"}}
	run := &domain.Run{RunID: "r1", WorkflowID: "w", WorkflowType: "t", TriggeredAt: time.Now(), State: domain.RunPending}
	ds := coord.GenerateDispatches(run, wf, []string{"c1", "c2"})
	if len(ds) != 2 {
		t.Error("count")
	}
	for _, d := range ds {
		if !d.IsValid() {
			t.Error("invalid dispatch")
		}
	}
}
