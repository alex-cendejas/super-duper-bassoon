package services

import (
	"context"
	"testing"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
)

func TestClientRegistrationService_HandleResult_SavesClient(t *testing.T) {
	clients := newMemClientRepo()
	svc := NewClientRegistrationService(clients, nil)
	ctx := context.Background()

	r := &domain.Result{
		RunID:      "run-1",
		WorkflowID: "wf-1",
		ClientID:   "client-abc",
		Status:     domain.ResultSuccess,
		ReceivedAt: time.Now().UTC(),
		InnerState: map[string]interface{}{"packages": map[string]interface{}{}, "config_version": 1},
	}

	if err := svc.HandleResult(ctx, r); err != nil {
		t.Fatalf("HandleResult returned error: %v", err)
	}

	saved, err := clients.GetClientByID(ctx, "client-abc")
	if err != nil {
		t.Fatalf("client not saved: %v", err)
	}
	if saved.ClientID != "client-abc" {
		t.Errorf("expected client_id=client-abc, got %s", saved.ClientID)
	}
	if !saved.Active {
		t.Error("expected client to be active")
	}
	if saved.Labels == nil {
		t.Error("expected Labels to be non-nil")
	}
}

func TestClientRegistrationService_HandleResult_NilResult(t *testing.T) {
	clients := newMemClientRepo()
	svc := NewClientRegistrationService(clients, nil)

	// Should not panic or error
	if err := svc.HandleResult(context.Background(), nil); err != nil {
		t.Errorf("expected no error for nil result, got %v", err)
	}
}

func TestClientRegistrationService_HandleResult_EmptyClientID(t *testing.T) {
	clients := newMemClientRepo()
	svc := NewClientRegistrationService(clients, nil)

	r := &domain.Result{
		RunID:      "run-1",
		WorkflowID: "wf-1",
		ClientID:   "", // empty
		Status:     domain.ResultSuccess,
	}

	if err := svc.HandleResult(context.Background(), r); err != nil {
		t.Errorf("expected no error for empty ClientID, got %v", err)
	}

	// No client should have been saved
	all, _ := clients.ListClients(context.Background())
	if len(all) != 0 {
		t.Errorf("expected no clients saved, got %d", len(all))
	}
}

func TestClientRegistrationService_HandleResult_InnerStatePassed(t *testing.T) {
	clients := newMemClientRepo()
	svc := NewClientRegistrationService(clients, nil)
	ctx := context.Background()

	inner := map[string]interface{}{
		"packages":       map[string]interface{}{"curl": "7.0"},
		"config_version": float64(3),
		"power_state":    "on",
	}

	r := &domain.Result{
		RunID:      "run-2",
		WorkflowID: "wf-2",
		ClientID:   "client-xyz",
		Status:     domain.ResultSuccess,
		InnerState: inner,
		ReceivedAt: time.Now().UTC(),
	}

	if err := svc.HandleResult(ctx, r); err != nil {
		t.Fatalf("HandleResult failed: %v", err)
	}

	saved, err := clients.GetClientByID(ctx, "client-xyz")
	if err != nil {
		t.Fatalf("client not found: %v", err)
	}
	if saved.InnerState == nil {
		t.Fatal("expected InnerState to be non-nil")
	}
	if v, ok := saved.InnerState["config_version"]; !ok || v != float64(3) {
		t.Errorf("expected config_version=3, got %v", v)
	}
}

func TestClientRegistrationService_HandleResult_NilInnerState(t *testing.T) {
	clients := newMemClientRepo()
	svc := NewClientRegistrationService(clients, nil)
	ctx := context.Background()

	r := &domain.Result{
		RunID:      "run-3",
		WorkflowID: "wf-3",
		ClientID:   "client-noinnerstate",
		Status:     domain.ResultFail,
		InnerState: nil,
		ReceivedAt: time.Now().UTC(),
	}

	if err := svc.HandleResult(ctx, r); err != nil {
		t.Fatalf("HandleResult failed: %v", err)
	}

	saved, err := clients.GetClientByID(ctx, "client-noinnerstate")
	if err != nil {
		t.Fatalf("client not found: %v", err)
	}
	if saved.InnerState == nil {
		t.Error("expected InnerState to be empty map, not nil")
	}
}

func TestClientRegistrationService_Metadata(t *testing.T) {
	svc := NewClientRegistrationService(newMemClientRepo(), nil)
	if svc.Name() != "client_registration" {
		t.Errorf("expected name=client_registration, got %s", svc.Name())
	}
	if svc.Priority() != 0 {
		t.Errorf("expected priority=0, got %d", svc.Priority())
	}
}
