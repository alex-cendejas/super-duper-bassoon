package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/super-duper-bassoon/internal/adapters/clock"
	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/services"
)

func TestStateOrchestrator_ApplyActivityResult(t *testing.T) {
	store := NewMockStateStore()
	clk := clock.NewMockClock(time.Now())
	chaos := deterministicChaos{float64Val: 0.5, intnVal: 0} // no failures, no drift
	orch := services.NewStateOrchestrator(store, clk, chaos)

	clientID := "client-1"
	newState := &domain.ClientState{
		Packages:      map[string]string{"curl": "7.88"},
		ConfigVersion: 2,
		PowerState:    domain.PowerStateOn,
	}

	if err := orch.ApplyActivityResult(context.Background(), clientID, newState); err != nil {
		t.Fatalf("ApplyActivityResult failed: %v", err)
	}

	stored, err := store.GetState(context.Background(), clientID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if stored.ConfigVersion != 2 {
		t.Errorf("expected ConfigVersion=2, got %d", stored.ConfigVersion)
	}
}

func TestStateOrchestrator_ApplyActivityResult_StoreError(t *testing.T) {
	store := NewMockStateStore()
	store.UpdateErr = errForcedFailure
	clk := clock.NewMockClock(time.Now())
	chaos := deterministicChaos{float64Val: 0.5}
	orch := services.NewStateOrchestrator(store, clk, chaos)

	err := orch.ApplyActivityResult(context.Background(), "client-1", &domain.ClientState{Packages: map[string]string{}})
	if err == nil {
		t.Error("expected error when store.UpdateState fails")
	}
}

func TestStateOrchestrator_ApplyDriftIfNeeded_DriftApplied(t *testing.T) {
	store := NewMockStateStore()
	store.Seed("client-1", domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
	})
	clk := clock.NewMockClock(time.Now())
	// float64Val=0.05 → ShouldDriftState returns true; intnVal=0 → config version up
	chaos := deterministicChaos{float64Val: 0.05, intnVal: 0}
	orch := services.NewStateOrchestrator(store, clk, chaos)

	drifted, err := orch.ApplyDriftIfNeeded(context.Background(), "client-1")
	if err != nil {
		t.Fatalf("ApplyDriftIfNeeded failed: %v", err)
	}
	if !drifted {
		t.Error("expected drift to be applied")
	}

	state, _ := store.GetState(context.Background(), "client-1")
	if state.ConfigVersion != 2 {
		t.Errorf("expected ConfigVersion=2 after drift, got %d", state.ConfigVersion)
	}
}

func TestStateOrchestrator_ApplyDriftIfNeeded_NoDrift(t *testing.T) {
	store := NewMockStateStore()
	store.Seed("client-1", domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 5,
		PowerState:    domain.PowerStateOn,
	})
	clk := clock.NewMockClock(time.Now())
	// float64Val=0.20 → ShouldDriftState returns false
	chaos := deterministicChaos{float64Val: 0.20}
	orch := services.NewStateOrchestrator(store, clk, chaos)

	drifted, err := orch.ApplyDriftIfNeeded(context.Background(), "client-1")
	if err != nil {
		t.Fatalf("ApplyDriftIfNeeded failed: %v", err)
	}
	if drifted {
		t.Error("expected no drift to be applied")
	}

	state, _ := store.GetState(context.Background(), "client-1")
	if state.ConfigVersion != 5 {
		t.Errorf("expected ConfigVersion=5 (unchanged), got %d", state.ConfigVersion)
	}
}

func TestStateOrchestrator_ApplyDriftIfNeeded_ClientNotFound(t *testing.T) {
	store := NewMockStateStore()
	clk := clock.NewMockClock(time.Now())
	chaos := deterministicChaos{float64Val: 0.05} // triggers drift
	orch := services.NewStateOrchestrator(store, clk, chaos)

	_, err := orch.ApplyDriftIfNeeded(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent client")
	}
}

func TestStateOrchestrator_CheckRecoveryFromCripple_RebootClearsIt(t *testing.T) {
	store := NewMockStateStore()
	clk := clock.NewMockClock(time.Now())
	chaos := deterministicChaos{float64Val: 0.5}
	orch := services.NewStateOrchestrator(store, clk, chaos)

	prev := &domain.ClientState{IsCrippled: true, CrippleMode: domain.CrippleModeFailPackageOps}
	next := &domain.ClientState{IsCrippled: false, CrippleMode: domain.CrippleModeNone}
	activity := domain.Activity{Type: domain.ActivityReboot}

	if !orch.CheckRecoveryFromCripple(prev, next, activity) {
		t.Error("expected recovery detected after successful reboot")
	}
}

func TestStateOrchestrator_CheckRecoveryFromCripple_NotReboot(t *testing.T) {
	store := NewMockStateStore()
	clk := clock.NewMockClock(time.Now())
	chaos := deterministicChaos{float64Val: 0.5}
	orch := services.NewStateOrchestrator(store, clk, chaos)

	prev := &domain.ClientState{IsCrippled: true}
	next := &domain.ClientState{IsCrippled: false}
	activity := domain.Activity{Type: domain.ActivityInstallPackage}

	if orch.CheckRecoveryFromCripple(prev, next, activity) {
		t.Error("recovery should only be detected for reboot activity")
	}
}

func TestStateOrchestrator_CheckRecoveryFromCripple_WasNotCrippled(t *testing.T) {
	store := NewMockStateStore()
	clk := clock.NewMockClock(time.Now())
	chaos := deterministicChaos{float64Val: 0.5}
	orch := services.NewStateOrchestrator(store, clk, chaos)

	prev := &domain.ClientState{IsCrippled: false}
	next := &domain.ClientState{IsCrippled: false}
	activity := domain.Activity{Type: domain.ActivityReboot}

	if orch.CheckRecoveryFromCripple(prev, next, activity) {
		t.Error("recovery should not be detected if client was not crippled before")
	}
}
