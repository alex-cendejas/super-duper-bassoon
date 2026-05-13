package domain_test

import (
	"testing"

	"github.com/super-duper-bassoon/internal/core/domain"
)

func baseState() domain.ClientState {
	return domain.ClientState{
		Packages:      map[string]string{"vim": "8.0"},
		ConfigVersion: 1,
		PowerState:    domain.PowerStateOn,
	}
}

func TestActivityIsValid(t *testing.T) {
	valid := []domain.ActivityType{
		domain.ActivityReboot,
		domain.ActivityInstallPackage,
		domain.ActivityUpgradePackage,
		domain.ActivityRemovePackage,
		domain.ActivityApplyConfig,
		domain.ActivityValidateConfig,
		domain.ActivityRunScript,
	}
	for _, at := range valid {
		a := domain.Activity{Type: at}
		if !a.IsValid() {
			t.Errorf("expected %s to be valid", at)
		}
	}

	invalid := domain.Activity{Type: "unknown"}
	if invalid.IsValid() {
		t.Error("expected 'unknown' activity type to be invalid")
	}
}

func TestExecuteReboot(t *testing.T) {
	state := baseState()
	state.IsCrippled = true
	state.CrippleMode = domain.CrippleModeFailPackageOps
	state.CrippleRecoveryAttempts = 2

	activity := domain.Activity{Type: domain.ActivityReboot}
	newState, result := domain.ExecuteActivity(activity, state)

	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success, got %s: %s", result.Status, result.ErrorMsg)
	}
	if newState.IsCrippled {
		t.Error("expected IsCrippled=false after reboot")
	}
	if newState.CrippleMode != domain.CrippleModeNone {
		t.Errorf("expected CrippleMode=none after reboot, got %s", newState.CrippleMode)
	}
	if newState.PowerState != domain.PowerStateOn {
		t.Errorf("expected PowerState=on after reboot, got %s", newState.PowerState)
	}
	if newState.CrippleRecoveryAttempts != 0 {
		t.Errorf("expected CrippleRecoveryAttempts=0 after reboot, got %d", newState.CrippleRecoveryAttempts)
	}
}

func TestExecuteInstallPackage_Success(t *testing.T) {
	state := baseState()
	activity := domain.Activity{
		Type:   domain.ActivityInstallPackage,
		Params: map[string]interface{}{"package": "curl", "version": "7.88"},
	}
	newState, result := domain.ExecuteActivity(activity, state)

	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success, got %s: %s", result.Status, result.ErrorMsg)
	}
	if newState.Packages["curl"] != "7.88" {
		t.Errorf("expected curl=7.88 in packages, got %s", newState.Packages["curl"])
	}
}

func TestExecuteInstallPackage_AlreadyInstalled(t *testing.T) {
	state := baseState()
	// vim is already in baseState
	activity := domain.Activity{
		Type:   domain.ActivityInstallPackage,
		Params: map[string]interface{}{"package": "vim"},
	}
	_, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultFail {
		t.Errorf("expected fail for already-installed package, got %s", result.Status)
	}
}

func TestExecuteInstallPackage_MissingName(t *testing.T) {
	state := baseState()
	activity := domain.Activity{
		Type:   domain.ActivityInstallPackage,
		Params: map[string]interface{}{},
	}
	_, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultError {
		t.Errorf("expected error for missing package name, got %s", result.Status)
	}
}

func TestExecuteInstallPackage_DefaultVersion(t *testing.T) {
	state := baseState()
	activity := domain.Activity{
		Type:   domain.ActivityInstallPackage,
		Params: map[string]interface{}{"package": "htop"},
	}
	newState, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultSuccess {
		t.Fatalf("expected success, got %s", result.Status)
	}
	if newState.Packages["htop"] != "latest" {
		t.Errorf("expected version=latest, got %s", newState.Packages["htop"])
	}
}

func TestExecuteUpgradePackage_Success(t *testing.T) {
	state := baseState()
	activity := domain.Activity{
		Type:   domain.ActivityUpgradePackage,
		Params: map[string]interface{}{"package": "vim", "version": "9.0"},
	}
	newState, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
	if newState.Packages["vim"] != "9.0" {
		t.Errorf("expected vim=9.0, got %s", newState.Packages["vim"])
	}
}

func TestExecuteUpgradePackage_NotInstalled(t *testing.T) {
	state := baseState()
	activity := domain.Activity{
		Type:   domain.ActivityUpgradePackage,
		Params: map[string]interface{}{"package": "nonexistent"},
	}
	_, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultFail {
		t.Errorf("expected fail for uninstalled package upgrade, got %s", result.Status)
	}
}

func TestExecuteRemovePackage_Success(t *testing.T) {
	state := baseState()
	activity := domain.Activity{
		Type:   domain.ActivityRemovePackage,
		Params: map[string]interface{}{"package": "vim"},
	}
	newState, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
	if _, ok := newState.Packages["vim"]; ok {
		t.Error("expected vim to be removed from packages")
	}
}

func TestExecuteRemovePackage_NotInstalled(t *testing.T) {
	state := baseState()
	activity := domain.Activity{
		Type:   domain.ActivityRemovePackage,
		Params: map[string]interface{}{"package": "nothere"},
	}
	_, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultFail {
		t.Errorf("expected fail for removing uninstalled package, got %s", result.Status)
	}
}

func TestExecuteApplyConfig(t *testing.T) {
	state := baseState()
	activity := domain.Activity{
		Type:   domain.ActivityApplyConfig,
		Params: map[string]interface{}{"config_version": 5},
	}
	newState, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
	if newState.ConfigVersion != 5 {
		t.Errorf("expected ConfigVersion=5, got %d", newState.ConfigVersion)
	}
}

func TestExecuteValidateConfig_Match(t *testing.T) {
	state := baseState() // ConfigVersion=1
	activity := domain.Activity{
		Type:   domain.ActivityValidateConfig,
		Params: map[string]interface{}{"config_version": 1},
	}
	_, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success for matching config version, got %s", result.Status)
	}
}

func TestExecuteValidateConfig_Mismatch(t *testing.T) {
	state := baseState() // ConfigVersion=1
	activity := domain.Activity{
		Type:   domain.ActivityValidateConfig,
		Params: map[string]interface{}{"config_version": 99},
	}
	_, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultFail {
		t.Errorf("expected fail for mismatched config version, got %s", result.Status)
	}
}

func TestExecuteRunScript(t *testing.T) {
	state := baseState()
	activity := domain.Activity{
		Type:   domain.ActivityRunScript,
		Params: map[string]interface{}{"script": "echo hello"},
	}
	_, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success for run_script, got %s", result.Status)
	}
	if result.Payload == nil {
		t.Error("expected payload for run_script")
	}
}

func TestExecuteUnknownActivity(t *testing.T) {
	state := baseState()
	activity := domain.Activity{Type: "totally_unknown"}
	_, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultError {
		t.Errorf("expected error for unknown activity type, got %s", result.Status)
	}
}

func TestExecuteActivity_DoesNotMutateOriginalState(t *testing.T) {
	state := baseState()
	originalPkgCount := len(state.Packages)

	activity := domain.Activity{
		Type:   domain.ActivityInstallPackage,
		Params: map[string]interface{}{"package": "new-pkg"},
	}
	domain.ExecuteActivity(activity, state)

	if len(state.Packages) != originalPkgCount {
		t.Error("ExecuteActivity mutated the original state's Packages map")
	}
}

func TestExecuteApplyConfig_Float64Param(t *testing.T) {
	// JSON unmarshaling produces float64 for numbers, so we must handle that.
	state := baseState()
	activity := domain.Activity{
		Type:   domain.ActivityApplyConfig,
		Params: map[string]interface{}{"config_version": float64(7)},
	}
	newState, result := domain.ExecuteActivity(activity, state)
	if result.Status != domain.ResultSuccess {
		t.Errorf("expected success, got %s", result.Status)
	}
	if newState.ConfigVersion != 7 {
		t.Errorf("expected ConfigVersion=7, got %d", newState.ConfigVersion)
	}
}
