package domain

import (
	"testing"
	"time"
)

func TestActivityType_IsValid(t *testing.T) {
	tests := []struct {
		a    ActivityType
		want bool
	}{
		{ActivityReboot, true},
		{ActivityInstallPackage, true},
		{ActivityUpgradePackage, true},
		{ActivityRemovePackage, true},
		{ActivityApplyConfig, true},
		{ActivityValidateConfig, true},
		{ActivityRunScript, true},
		{ActivityType("bogus"), false},
		{ActivityType(""), false},
	}
	for _, tc := range tests {
		if got := tc.a.IsValid(); got != tc.want {
			t.Errorf("%q.IsValid()=%v want %v", tc.a, got, tc.want)
		}
	}
}

func TestWorkflow_ValidateDefinition(t *testing.T) {
	good := &Workflow{Name: "n", WorkflowType: "t", Activity: ActivityReboot, SuccessThreshold: 80, LoopThresholdMS: 100, TimeoutMS: 1000}
	if err := good.ValidateDefinition(); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
	cases := map[string]*Workflow{
		"missing name":      {WorkflowType: "t", Activity: ActivityReboot},
		"missing type":      {Name: "x", Activity: ActivityReboot},
		"bad activity":      {Name: "x", WorkflowType: "t", Activity: ActivityType("X")},
		"bad threshold":     {Name: "x", WorkflowType: "t", Activity: ActivityReboot, SuccessThreshold: 200},
		"neg loop":          {Name: "x", WorkflowType: "t", Activity: ActivityReboot, SuccessThreshold: 50, LoopThresholdMS: -1},
		"neg timeout":       {Name: "x", WorkflowType: "t", Activity: ActivityReboot, SuccessThreshold: 50, TimeoutMS: -1},
	}
	for name, wf := range cases {
		if err := wf.ValidateDefinition(); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestWorkflow_Timeouts(t *testing.T) {
	wf := &Workflow{}
	if wf.GetActivityTimeout() != 30*time.Second {
		t.Errorf("default activity timeout wrong")
	}
	if wf.LoopThreshold() != 5*time.Second {
		t.Errorf("default loop threshold wrong")
	}
	wf.TimeoutMS = 100
	wf.LoopThresholdMS = 200
	if wf.GetActivityTimeout() != 100*time.Millisecond {
		t.Error("activity timeout mismatch")
	}
	if wf.LoopThreshold() != 200*time.Millisecond {
		t.Error("loop threshold mismatch")
	}
	if !wf.IsActive() && wf.Active {
		t.Error("IsActive mismatch")
	}
	wf.Active = true
	if !wf.IsActive() {
		t.Error("expected active")
	}
}
