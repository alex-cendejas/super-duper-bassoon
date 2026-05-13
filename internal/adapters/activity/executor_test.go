package activity_test

import (
	"context"
	"testing"

	"github.com/super-duper-bassoon/internal/adapters/activity"
	"github.com/super-duper-bassoon/internal/core/domain"
)

func baseState() domain.ClientState {
	return domain.ClientState{
		Packages:      map[string]string{"vim": "8.0"},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
	}
}

func TestStandardExecutor_Execute_ValidActivity(t *testing.T) {
	exec := activity.NewStandardExecutor()
	state := baseState()
	act := domain.Activity{
		Type:   domain.ActivityInstallPackage,
		Params: map[string]interface{}{"package": "curl"},
	}

	newState, result, err := exec.Execute(context.Background(), "client-1", act, state)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if newState == nil {
		t.Fatal("expected non-nil newState")
	}
	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
	if newState.Packages["curl"] != "latest" {
		t.Errorf("expected curl=latest, got %s", newState.Packages["curl"])
	}
}

func TestStandardExecutor_Execute_InvalidActivity(t *testing.T) {
	exec := activity.NewStandardExecutor()
	state := baseState()
	act := domain.Activity{Type: "invalid_type"}

	_, _, err := exec.Execute(context.Background(), "client-1", act, state)
	if err == nil {
		t.Error("expected error for invalid activity type")
	}
}

func TestStandardExecutor_Execute_Reboot(t *testing.T) {
	exec := activity.NewStandardExecutor()
	state := baseState()
	state.IsCrippled = true
	state.CrippleMode = domain.CrippleModeFailConfig

	act := domain.Activity{Type: domain.ActivityReboot}
	newState, result, err := exec.Execute(context.Background(), "client-1", act, state)
	if err != nil {
		t.Fatalf("Execute reboot failed: %v", err)
	}
	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success on reboot, got %s", result.Status)
	}
	if newState.IsCrippled {
		t.Error("expected IsCrippled=false after reboot")
	}
}

func TestStandardExecutor_Execute_DoesNotMutateInput(t *testing.T) {
	exec := activity.NewStandardExecutor()
	state := baseState()
	origVersion := state.ConfigVersion

	act := domain.Activity{
		Type:   domain.ActivityApplyConfig,
		Params: map[string]interface{}{"config_version": 50},
	}
	exec.Execute(context.Background(), "client-1", act, state)

	if state.ConfigVersion != origVersion {
		t.Error("Execute mutated input state")
	}
}

func TestChaosExecutor_CrippledSilentMode_FakesSuccess(t *testing.T) {
	inner := activity.NewStandardExecutor()
	// chaos that never fails randomly (float64=0.5), and drift also off
	chaos := deterministicChaos{float64Val: 0.5, intnVal: 0}
	logger := testLogger()
	exec := activity.NewChaosExecutor(inner, chaos, logger)

	state := domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
		IsCrippled:    true,
		CrippleMode:   domain.CrippleModeSilent,
	}

	act := domain.Activity{Type: domain.ActivityInstallPackage, Params: map[string]interface{}{"package": "curl"}}
	newState, result, err := exec.Execute(context.Background(), "client-1", act, state)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != domain.ResultSuccess {
		t.Errorf("silent cripple should fake success, got %s", result.Status)
	}
	// State should NOT change (it's fake success)
	if _, ok := newState.Packages["curl"]; ok {
		t.Error("silent cripple should not change state")
	}
}

func TestChaosExecutor_CrippledFailPackageOps_ReturnsFail(t *testing.T) {
	inner := activity.NewStandardExecutor()
	chaos := deterministicChaos{float64Val: 0.5, intnVal: 0}
	logger := testLogger()
	exec := activity.NewChaosExecutor(inner, chaos, logger)

	state := domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
		IsCrippled:    true,
		CrippleMode:   domain.CrippleModeFailPackageOps,
	}

	act := domain.Activity{Type: domain.ActivityInstallPackage, Params: map[string]interface{}{"package": "curl"}}
	_, result, err := exec.Execute(context.Background(), "client-1", act, state)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != domain.ResultFail {
		t.Errorf("crippled FailPackageOps should return fail, got %s", result.Status)
	}
}

func TestChaosExecutor_RandomFailure_ReturnsFailWithoutCrippling(t *testing.T) {
	inner := activity.NewStandardExecutor()
	// float64Val=0.05 → ShouldActivityFail=true; next float64Val same → ShouldCrippleClient with 0.05 > 0.03 → no cripple
	chaos := &sequentialChaos{values: []float64{0.05, 0.05}, intnVal: 0}
	logger := testLogger()
	exec := activity.NewChaosExecutor(inner, chaos, logger)

	state := domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
	}

	act := domain.Activity{Type: domain.ActivityReboot}
	newState, result, err := exec.Execute(context.Background(), "client-1", act, state)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != domain.ResultFail {
		t.Errorf("expected fail from random chaos, got %s", result.Status)
	}
	if newState.IsCrippled {
		t.Error("should not be crippled at 5% (above 3% threshold)")
	}
}

func TestChaosExecutor_RandomFailureWithCripple(t *testing.T) {
	inner := activity.NewStandardExecutor()
	// float64Val=0.05 → fails; then 0.01 → cripples; intnVal=0 → FailPackageOps
	chaos := &sequentialChaos{values: []float64{0.05, 0.01}, intnVal: 0}
	logger := testLogger()
	exec := activity.NewChaosExecutor(inner, chaos, logger)

	state := domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
	}

	act := domain.Activity{Type: domain.ActivityReboot}
	newState, result, err := exec.Execute(context.Background(), "client-1", act, state)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != domain.ResultFail {
		t.Errorf("expected fail, got %s", result.Status)
	}
	if !newState.IsCrippled {
		t.Error("expected client to be crippled after 1% cripple roll")
	}
	if newState.CrippleMode != domain.CrippleModeFailPackageOps {
		t.Errorf("expected CrippleModeFailPackageOps (intnVal=0), got %s", newState.CrippleMode)
	}
}

func TestChaosExecutor_NormalExecution(t *testing.T) {
	inner := activity.NewStandardExecutor()
	// No failures, no drift
	chaos := deterministicChaos{float64Val: 0.5, intnVal: 0}
	logger := testLogger()
	exec := activity.NewChaosExecutor(inner, chaos, logger)

	state := domain.ClientState{
		Packages:      map[string]string{},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
	}

	act := domain.Activity{Type: domain.ActivityReboot}
	_, result, err := exec.Execute(context.Background(), "client-1", act, state)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success with no chaos, got %s", result.Status)
	}
}
