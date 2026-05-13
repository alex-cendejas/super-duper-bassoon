package domain_test

import (
	"testing"

	"github.com/super-duper-bassoon/internal/core/domain"
)

func TestNewInnerClient(t *testing.T) {
	c := domain.NewInnerClient("client-001")
	if c.ClientID != "client-001" {
		t.Errorf("expected ClientID=client-001, got %s", c.ClientID)
	}
	if c.State.PowerState != domain.PowerStateOn {
		t.Errorf("expected PowerState=on, got %s", c.State.PowerState)
	}
	if c.State.ConfigVersion != 1 {
		t.Errorf("expected ConfigVersion=1, got %d", c.State.ConfigVersion)
	}
	if c.State.IsCrippled {
		t.Error("expected IsCrippled=false on new client")
	}
	if c.State.Packages == nil {
		t.Error("expected Packages to be initialized (not nil)")
	}
}

func TestClientStateClone(t *testing.T) {
	original := domain.ClientState{
		Packages:      map[string]string{"vim": "9.0", "curl": "7.88"},
		ConfigVersion: 3,
		PowerState:    domain.PowerStateOn,
		IsCrippled:    true,
		CrippleMode:   domain.CrippleModeFailPackageOps,
	}

	clone := original.Clone()

	// Verify equality
	if clone.ConfigVersion != original.ConfigVersion {
		t.Errorf("ConfigVersion mismatch: %d != %d", clone.ConfigVersion, original.ConfigVersion)
	}
	if clone.PowerState != original.PowerState {
		t.Errorf("PowerState mismatch")
	}
	if clone.IsCrippled != original.IsCrippled {
		t.Errorf("IsCrippled mismatch")
	}
	if clone.CrippleMode != original.CrippleMode {
		t.Errorf("CrippleMode mismatch")
	}
	if len(clone.Packages) != len(original.Packages) {
		t.Errorf("Packages length mismatch")
	}

	// Verify deep copy: mutating clone should not affect original.
	clone.Packages["new-pkg"] = "1.0"
	if _, ok := original.Packages["new-pkg"]; ok {
		t.Error("clone mutation leaked into original Packages map")
	}

	clone.ConfigVersion = 99
	if original.ConfigVersion == 99 {
		t.Error("clone ConfigVersion mutation leaked into original")
	}
}

func TestPowerStateConstants(t *testing.T) {
	states := []domain.PowerState{domain.PowerStateOn, domain.PowerStateOff, domain.PowerStateRestarting}
	if len(states) != 3 {
		t.Error("expected 3 power states")
	}
}

func TestCrippleModeConstants(t *testing.T) {
	modes := []domain.CrippleMode{
		domain.CrippleModeNone,
		domain.CrippleModeFailPackageOps,
		domain.CrippleModeFailConfig,
		domain.CrippleModeSilent,
	}
	if len(modes) != 4 {
		t.Error("expected 4 cripple modes")
	}
}
