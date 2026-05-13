package services_test

import (
	"context"
	"testing"

	"github.com/super-client/internal/core/domain"
	"github.com/super-client/internal/core/services"
)

func successResult() domain.ActivityResult {
	return domain.ActivityResult{Status: domain.ResultSuccess}
}

func failResult(msg string) domain.ActivityResult {
	return domain.ActivityResult{Status: domain.ResultFail, ErrorMsg: msg}
}

func sampleState() *domain.ClientState {
	return &domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
	}
}

func TestResultCollector_CollectAndFlush(t *testing.T) {
	broker := NewMockMessageBroker()
	rc := services.NewResultCollector(broker)

	if err := rc.Collect("run-1", "wf-1", "client-1", successResult(), sampleState()); err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if rc.GetPendingCount() != 1 {
		t.Errorf("expected 1 pending, got %d", rc.GetPendingCount())
	}

	if err := rc.FlushResults(context.Background()); err != nil {
		t.Fatalf("FlushResults failed: %v", err)
	}

	if rc.GetPendingCount() != 0 {
		t.Errorf("expected 0 pending after flush, got %d", rc.GetPendingCount())
	}

	published := broker.Published()
	if len(published) != 1 {
		t.Fatalf("expected 1 published result, got %d", len(published))
	}
	if published[0].RunID != "run-1" {
		t.Errorf("expected run_id=run-1, got %s", published[0].RunID)
	}
	if published[0].ClientID != "client-1" {
		t.Errorf("expected client_id=client-1, got %s", published[0].ClientID)
	}
}

func TestResultCollector_EmptyFlush(t *testing.T) {
	broker := NewMockMessageBroker()
	rc := services.NewResultCollector(broker)
	if err := rc.FlushResults(context.Background()); err != nil {
		t.Errorf("empty flush should not error: %v", err)
	}
}

func TestResultCollector_MultiplePendingResults(t *testing.T) {
	broker := NewMockMessageBroker()
	rc := services.NewResultCollector(broker)

	for i := range 5 {
		clientID := "client-" + string(rune('0'+i))
		if err := rc.Collect("run-1", "wf-1", clientID, successResult(), sampleState()); err != nil {
			t.Fatalf("Collect %d failed: %v", i, err)
		}
	}

	if rc.GetPendingCount() != 5 {
		t.Errorf("expected 5 pending, got %d", rc.GetPendingCount())
	}

	if err := rc.FlushResults(context.Background()); err != nil {
		t.Fatalf("FlushResults failed: %v", err)
	}
	if len(broker.Published()) != 5 {
		t.Errorf("expected 5 published, got %d", len(broker.Published()))
	}
}

func TestResultCollector_PublishError_RequeuesOnFailure(t *testing.T) {
	broker := NewMockMessageBroker()
	broker.PublishErr = errForcedFailure
	rc := services.NewResultCollector(broker)

	rc.Collect("run-1", "wf-1", "client-1", successResult(), sampleState())
	err := rc.FlushResults(context.Background())
	if err == nil {
		t.Error("expected error when broker.PublishResult fails")
	}

	// Failed result should be re-queued.
	if rc.GetPendingCount() != 1 {
		t.Errorf("expected 1 re-queued result after publish error, got %d", rc.GetPendingCount())
	}
}

func TestResultCollector_ValidationErrors(t *testing.T) {
	broker := NewMockMessageBroker()
	rc := services.NewResultCollector(broker)

	if err := rc.Collect("", "wf-1", "client-1", successResult(), nil); err == nil {
		t.Error("expected error for empty run_id")
	}
	if err := rc.Collect("run-1", "", "client-1", successResult(), nil); err == nil {
		t.Error("expected error for empty wf_id")
	}
	if err := rc.Collect("run-1", "wf-1", "", successResult(), nil); err == nil {
		t.Error("expected error for empty client_id")
	}
}

func TestResultCollector_StateIncludedInResult(t *testing.T) {
	broker := NewMockMessageBroker()
	rc := services.NewResultCollector(broker)

	state := &domain.ClientState{ConfigVersion: 42, Packages: map[string]string{}, PowerState: domain.PowerStateOn}
	rc.Collect("run-1", "wf-1", "client-1", successResult(), state)
	rc.FlushResults(context.Background())

	published := broker.Published()
	if published[0].InnerState == nil {
		t.Fatal("expected InnerState in published result")
	}
	if published[0].InnerState.ConfigVersion != 42 {
		t.Errorf("expected ConfigVersion=42, got %d", published[0].InnerState.ConfigVersion)
	}
}
