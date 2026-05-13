package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/super-duper-bassoon/server/internal/adapters/alert"
	"github.com/super-duper-bassoon/server/internal/adapters/enforcement"
	httpserver "github.com/super-duper-bassoon/server/internal/adapters/http"
	"github.com/super-duper-bassoon/server/internal/adapters/messaging"
	"github.com/super-duper-bassoon/server/internal/adapters/repository"
	"github.com/super-duper-bassoon/server/internal/adapters/trigger"
	"github.com/super-duper-bassoon/server/internal/core/domain"
	"github.com/super-duper-bassoon/server/internal/core/services"
)

type harness struct {
	t          *testing.T
	registry   *repository.Registry
	dispatcher *messaging.ChannelDispatcher
	eventBus   *messaging.InMemoryEventBus
	alerts     *alert.StdoutAlertPublisher
	banEnf     *services.BanEnforcementService
	healthSvc  *services.HealthMonitoringService
	circuitSvc *services.CircuitBreakerService
	loopSvc    *services.LoopDetectionService
	orch       *services.WorkflowOrchestrationService
	api        *services.APIHandlerService
	resultDisp *services.ResultMessageDispatcher
	triggerSvc *services.TriggerCoordinationService
	router     http.Handler
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	db, err := repository.OpenSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	registry := repository.NewRegistry(db)
	dispatcher := messaging.NewChannelDispatcher()
	bus := messaging.NewInMemoryEventBus(nil)
	alerter := alert.NewStdoutAlertPublisher(nil)
	blocker := enforcement.NewInMemoryDispatchBlocker()
	banEnf := services.NewBanEnforcementService(registry.Ban(), alerter, blocker, nil)
	filter := services.NewDispatchFilterService(banEnf, nil)
	dispCoord := services.NewDispatchCoordinationService(registry.Run(), dispatcher, registry.Client(), filter, nil)
	group := services.NewDynamicGroupingService(registry.Client())
	orch := services.NewWorkflowOrchestrationService(registry.Workflow(), registry.Run(), registry.Client(), group, dispCoord, bus, nil)

	cfg := services.NewDefaultConfigRepository(registry.Workflow(), 80, 5)
	healthSvc := services.NewHealthMonitoringService(registry.Run(), registry.Result(), registry.Ban(), registry.Health(), registry.Workflow(), bus, cfg, 5, nil)

	stateMgr := services.NewWorkflowStateManager(registry.Workflow())
	policy := services.NewDefaultPolicyRepo(registry.Workflow(), 80, 5, 60000)
	pol := &domain.CircuitBreakerPolicy{SuccessThreshold: 80, EvaluationWindow: 5, CooldownPeriod: time.Minute}
	circuit := services.NewCircuitBreakerService(registry.Health(), registry.Circuit(), policy, registry.Workflow(), stateMgr, alerter, bus, pol, nil)
	_ = bus.Subscribe("health.updated", circuit.OnHealthUpdatedEvent)

	loop := services.NewLoopDetectionService(registry.Run(), registry.Workflow(), registry.Ban(), banEnf, alerter, 5000, nil)

	rd := services.NewResultMessageDispatcher(nil)
	rd.RegisterHandler(loop)
	rd.RegisterHandler(healthSvc)

	api := services.NewAPIHandlerService(registry.Workflow(), orch, registry.Client(), registry.Run(), registry.Result(), registry.Ban(), banEnf, circuit, healthSvc)

	router := httpserver.NewRouter(httpserver.Deps{
		API:           api,
		HealthRepo:    registry.Health(),
		NATSConnected: func() bool { return true },
		DBHealthy:     func() bool { return true },
		Logger:        nil,
	})

	triggerSvc := services.NewTriggerCoordinationService(registry.Workflow(), orch, trigger.NewCronEvaluator(), bus, 60000, nil)

	return &harness{
		t: t, registry: registry, dispatcher: dispatcher, eventBus: bus,
		alerts: alerter, banEnf: banEnf, healthSvc: healthSvc, circuitSvc: circuit,
		loopSvc: loop, orch: orch, api: api, resultDisp: rd, triggerSvc: triggerSvc,
		router: router,
	}
}

func (h *harness) makeClients(ctx context.Context, ids ...string) {
	for _, id := range ids {
		_ = h.registry.Client().SaveClient(ctx, &domain.ClientMetadata{ClientID: id, OS: "linux", Active: true})
	}
}

func (h *harness) makeWorkflow(ctx context.Context, name, wfType string, threshold float64) *domain.Workflow {
	wf, err := h.orch.CreateWorkflow(ctx, &domain.CreateWorkflowRequest{
		Name: name, WorkflowType: wfType, Activity: domain.ActivityReboot,
		SuccessThreshold: threshold, LoopThresholdMS: 5000, TimeoutMS: 30000,
	})
	if err != nil {
		h.t.Fatal(err)
	}
	return wf
}

func TestIntegration_TriggerToHealth(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.makeClients(ctx, "a", "b", "c")
	wf := h.makeWorkflow(ctx, "deploy", "deploy", 80)

	run, err := h.orch.TriggerWorkflow(ctx, wf.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(h.dispatcher.Snapshot()) != 3 {
		t.Errorf("expected 3 dispatches, got %d", len(h.dispatcher.Snapshot()))
	}

	// Simulate results: 2 success, 1 fail
	for _, cid := range []string{"a", "b"} {
		_ = h.resultDisp.Dispatch(ctx, &domain.Result{RunID: run.RunID, WorkflowID: wf.ID, ClientID: cid, Status: domain.StatusSuccess, ReceivedAt: time.Now()})
	}
	_ = h.resultDisp.Dispatch(ctx, &domain.Result{RunID: run.RunID, WorkflowID: wf.ID, ClientID: "c", Status: domain.StatusFail, ReceivedAt: time.Now()})

	got, err := h.healthSvc.GetCurrentHealth(ctx, "deploy")
	if err != nil {
		t.Fatal(err)
	}
	if got.SuccessPercentageAvg < 60 || got.SuccessPercentageAvg > 70 {
		t.Errorf("expected ~66.67%%, got %v", got.SuccessPercentageAvg)
	}
}

func TestIntegration_LoopDetectionLeadsToBan(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.makeClients(ctx, "looper")
	// Threshold 1% so circuit breaker doesn't auto-deactivate the workflow
	wf := h.makeWorkflow(ctx, "deploy", "deploy", 1)

	run1, _ := h.orch.TriggerWorkflow(ctx, wf.ID, "first")
	_ = h.resultDisp.Dispatch(ctx, &domain.Result{RunID: run1.RunID, WorkflowID: wf.ID, ClientID: "looper", Status: domain.StatusSuccess})

	// Second run within loop_threshold (5s)
	run2, _ := h.orch.TriggerWorkflow(ctx, wf.ID, "loop")
	_ = h.resultDisp.Dispatch(ctx, &domain.Result{RunID: run2.RunID, WorkflowID: wf.ID, ClientID: "looper", Status: domain.StatusFail})

	bans, _ := h.registry.Ban().GetActiveBans(ctx, "looper")
	if len(bans) != 1 {
		t.Errorf("expected 1 ban, got %d", len(bans))
	}

	// Third trigger should not include the banned client
	run3, err := h.orch.TriggerWorkflow(ctx, wf.ID, "after-ban")
	if err != nil {
		t.Fatalf("third trigger failed: %v", err)
	}
	if len(run3.ParticipatingClients) != 0 {
		t.Errorf("expected 0 clients (looper banned), got %v", run3.ParticipatingClients)
	}
}

func TestIntegration_CircuitBreakerOpenOnUnhealthy(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.makeClients(ctx, "a", "b")
	wf := h.makeWorkflow(ctx, "deploy", "deploy", 80)

	// Several runs all fail. Stop as soon as workflow becomes inactive,
	// because the circuit breaker deactivates after the first all-fail run.
	for i := 0; i < 3; i++ {
		run, err := h.orch.TriggerWorkflow(ctx, wf.ID, "run")
		if err != nil {
			break
		}
		for _, cid := range run.ParticipatingClients {
			_ = h.resultDisp.Dispatch(ctx, &domain.Result{RunID: run.RunID, WorkflowID: wf.ID, ClientID: cid, Status: domain.StatusFail})
		}
	}
	cs, err := h.circuitSvc.GetCircuitState(ctx, wf.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cs.State != domain.CircuitOpen {
		t.Errorf("expected open, got %v", cs.State)
	}
	got, _ := h.registry.Workflow().GetWorkflow(ctx, wf.ID)
	if got.Active {
		t.Error("expected workflow deactivated")
	}
}

func TestIntegration_UnbanRestoresDispatch(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.makeClients(ctx, "x")
	wf := h.makeWorkflow(ctx, "deploy", "deploy", 80)
	_, _ = h.banEnf.BanClient(ctx, "x", "deploy", "rX", "ev", domain.ReasonLoopDetected)

	run, _ := h.orch.TriggerWorkflow(ctx, wf.ID, "blocked")
	if len(run.ParticipatingClients) != 0 {
		t.Error("expected empty due to ban")
	}
	if err := h.banEnf.UnbanClient(ctx, "x", "deploy", "admin", "ok"); err != nil {
		t.Fatal(err)
	}
	run, _ = h.orch.TriggerWorkflow(ctx, wf.ID, "unblocked")
	if len(run.ParticipatingClients) != 1 {
		t.Errorf("expected 1 client, got %v", run.ParticipatingClients)
	}
}

func TestIntegration_HTTPApi(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.makeClients(ctx, "a")

	srv := httptest.NewServer(h.router)
	t.Cleanup(srv.Close)
	c := srv.Client()

	// Create workflow
	body := `{"name":"deploy","workflow_type":"deploy","activity":"reboot","success_threshold":80,"loop_threshold_ms":5000,"timeout_ms":30000}`
	resp, err := c.Post(srv.URL+"/workflows", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var wf domain.Workflow
	_ = json.NewDecoder(resp.Body).Decode(&wf)
	resp.Body.Close()
	if wf.ID == "" {
		t.Error("missing id")
	}

	// List workflows
	resp, err = c.Get(srv.URL + "/workflows")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Get workflow
	resp, err = c.Get(srv.URL + "/workflows/" + wf.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("get workflow: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Trigger workflow
	resp, err = c.Post(srv.URL+"/workflows/"+wf.ID+"/trigger", "application/json", strings.NewReader(`{"reason":"manual"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("trigger: %d", resp.StatusCode)
	}
	var run domain.Run
	_ = json.NewDecoder(resp.Body).Decode(&run)
	resp.Body.Close()
	if run.RunID == "" {
		t.Error("missing run id")
	}

	// Get run
	resp, err = c.Get(srv.URL + "/runs/" + run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("get run: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Deactivate
	resp, err = c.Post(srv.URL+"/workflows/"+wf.ID+"/deactivate", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	// Activate
	resp, _ = c.Post(srv.URL+"/workflows/"+wf.ID+"/activate", "application/json", strings.NewReader("{}"))
	resp.Body.Close()

	// Edit
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/workflows/"+wf.ID, strings.NewReader(`{"description":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = c.Do(req)
	if resp.StatusCode != 200 {
		t.Errorf("edit: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Liveness/readiness
	resp, _ = c.Get(srv.URL + "/health/liveness")
	if resp.StatusCode != 200 {
		t.Error("liveness")
	}
	resp.Body.Close()
	resp, _ = c.Get(srv.URL + "/health/readiness")
	if resp.StatusCode != 200 {
		t.Error("readiness")
	}
	resp.Body.Close()

	// Status
	resp, _ = c.Get(srv.URL + "/status")
	if resp.StatusCode != 200 {
		t.Error("status")
	}
	resp.Body.Close()

	// List clients
	resp, _ = c.Get(srv.URL + "/clients")
	if resp.StatusCode != 200 {
		t.Error("list clients")
	}
	resp.Body.Close()
	resp, _ = c.Get(srv.URL + "/clients/a")
	if resp.StatusCode != 200 {
		t.Error("get client")
	}
	resp.Body.Close()

	// Bans empty
	resp, _ = c.Get(srv.URL + "/bans")
	if resp.StatusCode != 200 {
		t.Error("list bans")
	}
	resp.Body.Close()

	// Manually ban a client to test list/unban endpoints
	_, _ = h.banEnf.BanClient(ctx, "z", "deploy", "rX", "ev", domain.ReasonLoopDetected)
	resp, _ = c.Get(srv.URL + "/bans/z")
	if resp.StatusCode != 200 {
		t.Error("get client bans")
	}
	resp.Body.Close()
	req, _ = http.NewRequest(http.MethodPut, srv.URL+"/bans/z/unban", strings.NewReader(`{"workflow_type":"deploy","admin_id":"admin","reason":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = c.Do(req)
	if resp.StatusCode != 200 {
		t.Errorf("unban: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Workflow not found
	resp, _ = c.Get(srv.URL + "/workflows/missing")
	if resp.StatusCode != 404 {
		t.Errorf("missing: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Bad JSON
	resp, _ = c.Post(srv.URL+"/workflows", "application/json", strings.NewReader("not json"))
	if resp.StatusCode != 400 {
		t.Errorf("bad json: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Validation error - empty body
	resp, _ = c.Post(srv.URL+"/workflows", "application/json", strings.NewReader("{}"))
	if resp.StatusCode != 400 {
		t.Errorf("validation: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Edit with no changes
	req, _ = http.NewRequest(http.MethodPut, srv.URL+"/workflows/"+wf.ID, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = c.Do(req)
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Delete
	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/workflows/"+wf.ID, nil)
	resp, _ = c.Do(req)
	if resp.StatusCode != 204 {
		t.Errorf("delete: %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestIntegration_TriggerService_FiresOnComplete(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.makeClients(ctx, "a")

	wfA := h.makeWorkflow(ctx, "wfA", "tA", 80)
	wfB := h.makeWorkflow(ctx, "wfB", "tB", 80)
	wfB.Trigger = domain.TriggerSpec{Kind: domain.TriggerEvent, OnComplete: wfA.ID}
	_ = h.registry.Workflow().SaveWorkflow(ctx, wfB)

	// Start service so it subscribes to workflow.completed
	h.triggerSvc.Start(ctx)
	t.Cleanup(h.triggerSvc.Stop)

	// Trigger wfA which publishes workflow.completed
	_, err := h.orch.TriggerWorkflow(ctx, wfA.ID, "first")
	if err != nil {
		t.Fatal(err)
	}
	// Wait briefly so wfB is fired
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runs, _ := h.registry.Run().ListRuns(ctx, wfB.ID, 10)
		if len(runs) > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Error("wfB never triggered by event")
}

func TestIntegration_ResultDispatcher_FromBytes(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	h.makeClients(ctx, "a")
	wf := h.makeWorkflow(ctx, "deploy", "deploy", 80)
	run, _ := h.orch.TriggerWorkflow(ctx, wf.ID, "x")

	ch := make(chan []byte, 1)
	go h.resultDisp.Start(ctx, ch)

	payload := []byte(`{"run_id":"` + run.RunID + `","wf_id":"` + wf.ID + `","client_id":"a","status":"success"}`)
	ch <- payload

	// Allow time
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		res, err := h.registry.Result().GetResult(ctx, run.RunID, "a")
		if err == nil && res != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("result not processed")
}
