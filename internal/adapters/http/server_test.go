package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/super-duper-bassoon/internal/adapters/alert"
	"github.com/super-duper-bassoon/internal/adapters/enforcement"
	"github.com/super-duper-bassoon/internal/adapters/messaging"
	"github.com/super-duper-bassoon/internal/adapters/repository"
	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/services"
)

func setupRouter(t *testing.T) (http.Handler, *repository.Registry, *services.WorkflowOrchestrationService, *services.BanEnforcementService) {
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
	coord := services.NewDispatchCoordinationService(registry.Run(), dispatcher, registry.Client(), filter, nil)
	group := services.NewDynamicGroupingService(registry.Client())
	orch := services.NewWorkflowOrchestrationService(registry.Workflow(), registry.Run(), registry.Client(), group, coord, bus, nil)

	cfg := services.NewDefaultConfigRepository(registry.Workflow(), 80, 5)
	healthSvc := services.NewHealthMonitoringService(registry.Run(), registry.Result(), registry.Ban(), registry.Health(), registry.Workflow(), bus, cfg, 5, nil)
	stateMgr := services.NewWorkflowStateManager(registry.Workflow())
	policy := services.NewDefaultPolicyRepo(registry.Workflow(), 80, 5, 60000)
	pol := &domain.CircuitBreakerPolicy{SuccessThreshold: 80, EvaluationWindow: 5, CooldownPeriod: time.Minute}
	circuit := services.NewCircuitBreakerService(registry.Health(), registry.Circuit(), policy, registry.Workflow(), stateMgr, alerter, bus, pol, nil)
	api := services.NewAPIHandlerService(registry.Workflow(), orch, registry.Client(), registry.Run(), registry.Result(), registry.Ban(), banEnf, circuit, healthSvc)

	router := NewRouter(Deps{
		API:           api,
		HealthRepo:    registry.Health(),
		NATSConnected: func() bool { return true },
		DBHealthy:     func() bool { return true },
		Logger:        nil,
	})
	return router, registry, orch, banEnf
}

func get(t *testing.T, srv *httptest.Server, path string) (*http.Response, error) {
	t.Helper()
	return srv.Client().Get(srv.URL + path)
}

func TestServer_HealthRoutes(t *testing.T) {
	router, _, _, _ := setupRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()

	cases := []string{"/health/liveness", "/health/readiness", "/status", "/health", "/bans", "/clients", "/circuits"}
	for _, p := range cases {
		resp, err := get(t, srv, p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: %d", p, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestServer_ReadinessFails(t *testing.T) {
	db, err := repository.OpenSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	registry := repository.NewRegistry(db)
	api := services.NewAPIHandlerService(registry.Workflow(), nil, registry.Client(), registry.Run(), registry.Result(), registry.Ban(), nil, nil, nil)
	router := NewRouter(Deps{
		API:           api,
		HealthRepo:    registry.Health(),
		NATSConnected: func() bool { return false },
		DBHealthy:     func() bool { return false },
		Logger:        nil,
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, _ := get(t, srv, "/health/readiness")
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestServer_NotFound(t *testing.T) {
	router, _, _, _ := setupRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()
	resp, _ := get(t, srv, "/workflows/missing")
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp, _ = get(t, srv, "/runs/missing")
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp, _ = get(t, srv, "/clients/missing")
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestServer_CircuitBreaker_NotFound(t *testing.T) {
	router, _, _, _ := setupRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()
	resp, _ := get(t, srv, "/circuits/missing")
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestServer_BadJSON(t *testing.T) {
	router, _, _, _ := setupRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, _ := srv.Client().Post(srv.URL+"/workflows", "application/json", strings.NewReader("not json"))
	if resp.StatusCode != 400 {
		t.Errorf("create bad: %d", resp.StatusCode)
	}
	resp.Body.Close()

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/workflows/x", strings.NewReader("nope"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = srv.Client().Do(req)
	if resp.StatusCode != 400 {
		t.Errorf("edit bad: %d", resp.StatusCode)
	}
	resp.Body.Close()

	req, _ = http.NewRequest(http.MethodPut, srv.URL+"/bans/x/unban", strings.NewReader("nope"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = srv.Client().Do(req)
	if resp.StatusCode != 400 {
		t.Errorf("unban bad: %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestServer_TriggerInactive(t *testing.T) {
	router, _, orch, _ := setupRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()

	wf, _ := orch.CreateWorkflow(context.Background(), &domain.CreateWorkflowRequest{Name: "n", WorkflowType: "t", Activity: domain.ActivityReboot})
	_ = orch.DeactivateWorkflow(context.Background(), wf.ID)
	resp, _ := srv.Client().Post(srv.URL+"/workflows/"+wf.ID+"/trigger", "application/json", strings.NewReader("{}"))
	if resp.StatusCode != 409 {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestServer_DeleteMissing(t *testing.T) {
	router, _, _, _ := setupRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/workflows/missing", nil)
	resp, _ := srv.Client().Do(req)
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestServer_TriggerFilterError(t *testing.T) {
	router, _, orch, _ := setupRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()
	// Create with an invalid filter
	wf, err := orch.CreateWorkflow(context.Background(), &domain.CreateWorkflowRequest{Name: "n", WorkflowType: "t", Activity: domain.ActivityReboot, TargetFilter: "@@@"})
	if err != nil {
		t.Fatal(err)
	}
	resp, _ := srv.Client().Post(srv.URL+"/workflows/"+wf.ID+"/trigger", "application/json", strings.NewReader("{}"))
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for bad filter, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestToAPIError_Default(t *testing.T) {
	if e := toAPIError(nil); e.Code != "INTERNAL" {
		t.Error("nil err => INTERNAL")
	}
	if e := toAPIError(domain.NewAPIError("X", "msg")); e.Code != "X" {
		t.Error("wraps APIError")
	}
	if e := toAPIError(errors.New("other")); e.Code != "INTERNAL" {
		t.Error("other => INTERNAL")
	}
	if e := toAPIError(domain.ErrInvalidFilter); e.Code != "VALIDATION_ERROR" {
		t.Error("invalid filter => VALIDATION_ERROR")
	}
}
