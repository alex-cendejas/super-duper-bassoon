package services_test

import (
	"context"
	"testing"

	"github.com/super-client/internal/core/domain"
	"github.com/super-client/internal/core/services"
)

func seedStore(store *MockStateStore, clientID string) {
	store.Seed(clientID, domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
	})
}

func validDispatch(clientID string) domain.DispatchMessage {
	return domain.DispatchMessage{
		RunID:    "run-001",
		WfID:     "wf-001",
		ClientID: clientID,
		Activity: domain.Activity{
			Type:   domain.ActivityInstallPackage,
			Params: map[string]interface{}{"package": "curl"},
		},
	}
}

func TestValidateDispatch_Valid(t *testing.T) {
	d := validDispatch("client-1")
	if err := services.ValidateDispatch(d); err != nil {
		t.Errorf("expected valid dispatch, got error: %v", err)
	}
}

func TestValidateDispatch_MissingRunID(t *testing.T) {
	d := validDispatch("client-1")
	d.RunID = ""
	if err := services.ValidateDispatch(d); err == nil {
		t.Error("expected error for missing run_id")
	}
}

func TestValidateDispatch_MissingWfID(t *testing.T) {
	d := validDispatch("client-1")
	d.WfID = ""
	if err := services.ValidateDispatch(d); err == nil {
		t.Error("expected error for missing wf_id")
	}
}

func TestValidateDispatch_MissingClientID(t *testing.T) {
	d := validDispatch("client-1")
	d.ClientID = ""
	if err := services.ValidateDispatch(d); err == nil {
		t.Error("expected error for missing client_id")
	}
}

func TestValidateDispatch_InvalidActivity(t *testing.T) {
	d := validDispatch("client-1")
	d.Activity.Type = "not_an_activity"
	if err := services.ValidateDispatch(d); err == nil {
		t.Error("expected error for invalid activity type")
	}
}

func TestDispatchHandler_Handle_Success(t *testing.T) {
	store := NewMockStateStore()
	executor := &MockActivityExecutor{}
	seedStore(store, "client-1")

	handler := services.NewDispatchHandler(store, executor)
	dispatch := validDispatch("client-1")

	newState, result, err := handler.Handle(context.Background(), dispatch)
	if err != nil {
		t.Fatalf("Handle returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if newState == nil {
		t.Fatal("expected non-nil newState")
	}
	if executor.CallCount() != 1 {
		t.Errorf("expected executor called once, got %d", executor.CallCount())
	}
}

func TestDispatchHandler_Handle_ClientNotFound(t *testing.T) {
	store := NewMockStateStore()
	executor := &MockActivityExecutor{}

	handler := services.NewDispatchHandler(store, executor)
	dispatch := validDispatch("nonexistent-client")

	_, _, err := handler.Handle(context.Background(), dispatch)
	if err == nil {
		t.Error("expected error for nonexistent client")
	}
}

func TestDispatchHandler_Handle_InvalidDispatch(t *testing.T) {
	store := NewMockStateStore()
	executor := &MockActivityExecutor{}
	seedStore(store, "client-1")

	handler := services.NewDispatchHandler(store, executor)
	dispatch := validDispatch("client-1")
	dispatch.RunID = "" // invalid

	_, _, err := handler.Handle(context.Background(), dispatch)
	if err == nil {
		t.Error("expected error for invalid dispatch")
	}
	if executor.CallCount() != 0 {
		t.Error("executor should not be called for invalid dispatch")
	}
}

func TestDispatchHandler_Handle_ExecutorError(t *testing.T) {
	store := NewMockStateStore()
	executor := &MockActivityExecutor{Err: errForcedFailure}
	seedStore(store, "client-1")

	handler := services.NewDispatchHandler(store, executor)
	dispatch := validDispatch("client-1")

	_, _, err := handler.Handle(context.Background(), dispatch)
	if err == nil {
		t.Error("expected error when executor fails")
	}
}

func TestDispatchHandler_Handle_StoreError(t *testing.T) {
	store := NewMockStateStore()
	store.GetErr = errForcedFailure
	executor := &MockActivityExecutor{}

	handler := services.NewDispatchHandler(store, executor)
	dispatch := validDispatch("client-1")

	_, _, err := handler.Handle(context.Background(), dispatch)
	if err == nil {
		t.Error("expected error when store.GetState fails")
	}
	if executor.CallCount() != 0 {
		t.Error("executor should not be called when store fails")
	}
}

func TestDispatchHandler_Handle_AllActivityTypes(t *testing.T) {
	activityTypes := []domain.ActivityType{
		domain.ActivityReboot,
		domain.ActivityInstallPackage,
		domain.ActivityUpgradePackage,
		domain.ActivityRemovePackage,
		domain.ActivityApplyConfig,
		domain.ActivityValidateConfig,
		domain.ActivityRunScript,
	}

	for _, at := range activityTypes {
		t.Run(string(at), func(t *testing.T) {
			store := NewMockStateStore()
			executor := &MockActivityExecutor{}
			seedStore(store, "client-1")

			// Seed initial packages for upgrade/remove
			store.Seed("client-1", domain.ClientState{
				Packages:      map[string]string{"vim": "8.0"},
				ConfigVersion: 1,
				PowerState:    domain.PowerStateOn,
			})

			handler := services.NewDispatchHandler(store, executor)

			params := map[string]interface{}{}
			switch at {
			case domain.ActivityInstallPackage:
				params["package"] = "curl"
			case domain.ActivityUpgradePackage:
				params["package"] = "vim"
				params["version"] = "9.0"
			case domain.ActivityRemovePackage:
				params["package"] = "vim"
			case domain.ActivityApplyConfig, domain.ActivityValidateConfig:
				params["config_version"] = 1
			case domain.ActivityRunScript:
				params["script"] = "echo hello"
			}

			dispatch := domain.DispatchMessage{
				RunID:    "run-001",
				WfID:     "wf-001",
				ClientID: "client-1",
				Activity: domain.Activity{Type: at, Params: params},
			}

			_, _, err := handler.Handle(context.Background(), dispatch)
			if err != nil {
				t.Errorf("Handle(%s) returned unexpected error: %v", at, err)
			}
		})
	}
}
