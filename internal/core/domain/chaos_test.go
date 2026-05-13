package domain_test

import (
	"testing"

	"github.com/super-duper-bassoon/internal/core/domain"
)

// deterministicChaos is a controllable ChaosSource for testing.
type deterministicChaos struct {
	float64Val float64
	intnVal    int
}

func (d deterministicChaos) Float64() float64 { return d.float64Val }
func (d deterministicChaos) Intn(_ int) int   { return d.intnVal }

func TestShouldActivityFail_Below10Pct(t *testing.T) {
	src := deterministicChaos{float64Val: 0.05}
	if !domain.ShouldActivityFail(src) {
		t.Error("expected failure at 5% (below 10% threshold)")
	}
}

func TestShouldActivityFail_Above10Pct(t *testing.T) {
	src := deterministicChaos{float64Val: 0.15}
	if domain.ShouldActivityFail(src) {
		t.Error("expected no failure at 15% (above 10% threshold)")
	}
}

func TestShouldActivityFail_AtBoundary(t *testing.T) {
	// Exactly at 0.10 should NOT fail (< not <=)
	src := deterministicChaos{float64Val: 0.10}
	if domain.ShouldActivityFail(src) {
		t.Error("expected no failure at exactly 0.10")
	}
}

func TestShouldCrippleClient_NoFailure(t *testing.T) {
	src := deterministicChaos{float64Val: 0.01} // would cripple if failure happened
	if domain.ShouldCrippleClient(src, false) {
		t.Error("should not cripple when didFail=false")
	}
}

func TestShouldCrippleClient_WithFailureBelow3Pct(t *testing.T) {
	src := deterministicChaos{float64Val: 0.02}
	if !domain.ShouldCrippleClient(src, true) {
		t.Error("expected cripple at 2% (below 3% threshold)")
	}
}

func TestShouldCrippleClient_WithFailureAbove3Pct(t *testing.T) {
	src := deterministicChaos{float64Val: 0.05}
	if domain.ShouldCrippleClient(src, true) {
		t.Error("expected no cripple at 5% (above 3% threshold)")
	}
}

func TestSelectCrippleMode_AllModes(t *testing.T) {
	expectedModes := []domain.CrippleMode{
		domain.CrippleModeFailPackageOps,
		domain.CrippleModeFailConfig,
		domain.CrippleModeSilent,
	}
	for i, expected := range expectedModes {
		src := deterministicChaos{intnVal: i}
		got := domain.SelectCrippleMode(src)
		if got != expected {
			t.Errorf("Intn=%d: expected %s, got %s", i, expected, got)
		}
	}
}

func TestShouldDriftState_Below7_5Pct(t *testing.T) {
	src := deterministicChaos{float64Val: 0.05}
	if !domain.ShouldDriftState(src) {
		t.Error("expected drift at 5% (below 7.5% threshold)")
	}
}

func TestShouldDriftState_Above7_5Pct(t *testing.T) {
	src := deterministicChaos{float64Val: 0.10}
	if domain.ShouldDriftState(src) {
		t.Error("expected no drift at 10% (above 7.5% threshold)")
	}
}

func TestApplyDrift_ConfigVersionUp(t *testing.T) {
	src := deterministicChaos{intnVal: 0} // choice 0 = config version up
	state := domain.ClientState{
		Packages:      make(map[string]string),
		ConfigVersion: 5,
	}
	newState := domain.ApplyDrift(src, state)
	if newState.ConfigVersion != 6 {
		t.Errorf("expected ConfigVersion=6, got %d", newState.ConfigVersion)
	}
	// Original should not be mutated
	if state.ConfigVersion != 5 {
		t.Error("ApplyDrift mutated original state")
	}
}

func TestApplyDrift_AddPackage(t *testing.T) {
	src := deterministicChaos{intnVal: 1} // choice 1 = add package
	state := domain.ClientState{
		Packages:      make(map[string]string),
		ConfigVersion: 1,
	}
	newState := domain.ApplyDrift(src, state)
	if len(newState.Packages) != 1 {
		t.Errorf("expected 1 package after drift, got %d", len(newState.Packages))
	}
}

func TestApplyDrift_RemovePackage(t *testing.T) {
	src := deterministicChaos{intnVal: 2} // choice 2 = remove package
	state := domain.ClientState{
		Packages:      map[string]string{"vim": "9.0"},
		ConfigVersion: 1,
	}
	newState := domain.ApplyDrift(src, state)
	if len(newState.Packages) != 0 {
		t.Errorf("expected 0 packages after drift removal, got %d", len(newState.Packages))
	}
}

func TestApplyDrift_RemovePackage_EmptyMap(t *testing.T) {
	src := deterministicChaos{intnVal: 2} // choice 2 = remove package
	state := domain.ClientState{
		Packages:      make(map[string]string),
		ConfigVersion: 1,
	}
	// Should not panic when no packages to remove
	newState := domain.ApplyDrift(src, state)
	if len(newState.Packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(newState.Packages))
	}
}

func TestIsCrippledForActivity_NotCrippled(t *testing.T) {
	state := domain.ClientState{IsCrippled: false}
	if domain.IsCrippledForActivity(state, domain.ActivityInstallPackage) {
		t.Error("should not be crippled when IsCrippled=false")
	}
}

func TestIsCrippledForActivity_FailPackageOps(t *testing.T) {
	state := domain.ClientState{IsCrippled: true, CrippleMode: domain.CrippleModeFailPackageOps}
	packageActivities := []domain.ActivityType{
		domain.ActivityInstallPackage,
		domain.ActivityUpgradePackage,
		domain.ActivityRemovePackage,
	}
	for _, at := range packageActivities {
		if !domain.IsCrippledForActivity(state, at) {
			t.Errorf("expected %s to be crippled in FailPackageOps mode", at)
		}
	}
	// Non-package activities should not be blocked
	nonPackage := []domain.ActivityType{domain.ActivityReboot, domain.ActivityApplyConfig}
	for _, at := range nonPackage {
		if domain.IsCrippledForActivity(state, at) {
			t.Errorf("expected %s NOT to be crippled in FailPackageOps mode", at)
		}
	}
}

func TestIsCrippledForActivity_FailConfig(t *testing.T) {
	state := domain.ClientState{IsCrippled: true, CrippleMode: domain.CrippleModeFailConfig}
	if !domain.IsCrippledForActivity(state, domain.ActivityApplyConfig) {
		t.Error("expected ApplyConfig to be crippled in FailConfig mode")
	}
	if !domain.IsCrippledForActivity(state, domain.ActivityValidateConfig) {
		t.Error("expected ValidateConfig to be crippled in FailConfig mode")
	}
	if domain.IsCrippledForActivity(state, domain.ActivityInstallPackage) {
		t.Error("expected InstallPackage NOT to be crippled in FailConfig mode")
	}
}

func TestIsCrippledForActivity_Silent(t *testing.T) {
	state := domain.ClientState{IsCrippled: true, CrippleMode: domain.CrippleModeSilent}
	allActivities := []domain.ActivityType{
		domain.ActivityReboot,
		domain.ActivityInstallPackage,
		domain.ActivityUpgradePackage,
		domain.ActivityRemovePackage,
		domain.ActivityApplyConfig,
		domain.ActivityValidateConfig,
		domain.ActivityRunScript,
	}
	for _, at := range allActivities {
		if !domain.IsCrippledForActivity(state, at) {
			t.Errorf("expected %s to be crippled in Silent mode", at)
		}
	}
}
