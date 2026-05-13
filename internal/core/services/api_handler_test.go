package services

import (
	"context"
	"testing"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
)

func newAPIHarness(t *testing.T) (*APIHandlerService, *memoryWorkflowRepo, *memoryClientRepo, *memoryBanRepo) {
	t.Helper()
	wfs := newMemWorkflowRepo()
	runs := newMemRunRepo()
	clients := newMemClientRepo()
	dispatcher := newStubDispatcher()
	bans := newMemBanRepo()
	alerter := newStubAlerter()
	blocker := newStubBlocker()
	enf := NewBanEnforcementService(bans, alerter, blocker, nil)
	filter := NewDispatchFilterService(enf, nil)
	coord := NewDispatchCoordinationService(runs, dispatcher, clients, filter, nil)
	group := NewDynamicGroupingService(clients)
	bus := newStubEventBus()
	orch := NewWorkflowOrchestrationService(wfs, runs, clients, group, coord, bus, nil)

	results := newMemResultRepo()
	health := newMemHealthRepo()
	circuits := newMemCircuitRepo()
	cfg := NewDefaultConfigRepository(wfs, 80, 5)
	healthSvc := NewHealthMonitoringService(runs, results, bans, health, wfs, bus, cfg, 5, nil)
	stateMgr := NewWorkflowStateManager(wfs)
	policy := NewDefaultPolicyRepo(wfs, 80, 5, 60000)
	pol := &domain.CircuitBreakerPolicy{SuccessThreshold: 80, EvaluationWindow: 5, CooldownPeriod: time.Minute}
	circuit := NewCircuitBreakerService(health, circuits, policy, wfs, stateMgr, alerter, bus, pol, nil)
	api := NewAPIHandlerService(wfs, orch, clients, runs, results, bans, enf, circuit, healthSvc)
	return api, wfs, clients, bans
}

func TestAPIHandler_AllMethods(t *testing.T) {
	api, _, clients, _ := newAPIHarness(t)
	ctx := context.Background()
	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "a", OS: "linux", Active: true})

	// CreateWorkflow
	wf, err := api.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{Name: "n", WorkflowType: "t", Activity: domain.ActivityReboot})
	if err != nil {
		t.Fatal(err)
	}
	got, err := api.GetWorkflow(ctx, wf.ID)
	if err != nil || got.ID != wf.ID {
		t.Errorf("get: %v %v", got, err)
	}
	list, _ := api.ListWorkflows(ctx)
	if len(list) != 1 {
		t.Error("list")
	}

	// EditWorkflow
	name := "renamed"
	updated, err := api.EditWorkflow(ctx, wf.ID, &domain.EditWorkflowRequest{Name: &name})
	if err != nil || updated.Name != "renamed" {
		t.Errorf("edit: %+v %v", updated, err)
	}

	// Trigger
	run, err := api.TriggerWorkflow(ctx, wf.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := api.GetRun(ctx, run.RunID); err != nil || got == nil {
		t.Errorf("get run: %v %v", got, err)
	}
	if list, _ := api.ListRuns(ctx, wf.ID, 10); len(list) != 1 {
		t.Error("list runs")
	}
	if _, err := api.GetRunResults(ctx, run.RunID); err != nil {
		t.Error("get run results")
	}

	// Clients
	if got, err := api.GetClient(ctx, "a"); err != nil || got == nil {
		t.Error("client")
	}
	if list, _ := api.ListClients(ctx); len(list) != 1 {
		t.Error("list clients")
	}

	// Bans
	if list, _ := api.ListAllBans(ctx); len(list) != 0 {
		t.Errorf("expected no bans: %v", list)
	}
	if list, _ := api.GetBans(ctx, "a"); len(list) != 0 {
		t.Errorf("client bans: %v", list)
	}

	// Unban requires admin/reason
	if err := api.UnbanClient(ctx, "a", &domain.UnbanRequest{}); err == nil {
		t.Error("expected validation error")
	}

	// Deactivate/Activate
	if _, err := api.DeactivateWorkflow(ctx, wf.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ActivateWorkflow(ctx, wf.ID); err != nil {
		t.Fatal(err)
	}

	// Circuit states
	if _, err := api.ListCircuitStates(ctx); err != nil {
		t.Error("list circuits")
	}
	if _, err := api.GetCircuitState(ctx, wf.ID); err == nil {
		// state not yet saved
	}

	// Health
	if _, err := api.GetHealth(ctx, "t"); err == nil {
		// type health not yet saved -- ok either way
	}

	// SystemStatus
	st := api.SystemStatus(ctx, true, true)
	if st.DBStatus != "healthy" || st.NATSStatus != "connected" {
		t.Errorf("status: %+v", st)
	}
	st = api.SystemStatus(ctx, false, false)
	if st.DBStatus != "unhealthy" || st.NATSStatus != "disconnected" {
		t.Errorf("bad status: %+v", st)
	}

	// Delete
	if err := api.DeleteWorkflow(ctx, wf.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := api.GetWorkflow(ctx, wf.ID); err == nil {
		t.Error("expected not found")
	}
}

func TestAPIHandler_UnbanIntegrated(t *testing.T) {
	api, _, _, bans := newAPIHarness(t)
	ctx := context.Background()
	_ = bans.SaveBan(ctx, &domain.BanRecord{ClientID: "c", WorkflowType: "t", Reason: domain.ReasonLoopDetected, Active: true})
	if err := api.UnbanClient(ctx, "c", &domain.UnbanRequest{WorkflowType: "t", AdminID: "admin", Reason: "manual"}); err != nil {
		t.Fatal(err)
	}
	active, _ := bans.GetActiveBans(ctx, "c")
	if len(active) != 0 {
		t.Errorf("expected unbanned")
	}
}
