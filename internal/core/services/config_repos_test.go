package services

import (
	"context"
	"testing"

	"github.com/super-duper-bassoon/internal/core/domain"
)

func TestDefaultConfigRepo(t *testing.T) {
	wfs := newMemWorkflowRepo()
	ctx := context.Background()
	r := NewDefaultConfigRepository(wfs, 80, 10)
	// No workflow exists, returns defaults
	t1, _ := r.GetHealthThreshold(ctx, "missing")
	if t1.SuccessThreshold != 80 || t1.WindowSize != 10 {
		t.Errorf("defaults: %+v", t1)
	}
	_ = wfs.SaveWorkflow(ctx, &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, SuccessThreshold: 50})
	t2, _ := r.GetHealthThreshold(ctx, "t")
	if t2.SuccessThreshold != 50 {
		t.Errorf("override: %+v", t2)
	}
}

func TestDefaultPolicyRepo(t *testing.T) {
	wfs := newMemWorkflowRepo()
	ctx := context.Background()
	r := NewDefaultPolicyRepo(wfs, 80, 10, 60000)
	p, _ := r.GetDefaultPolicy(ctx)
	if p.SuccessThreshold != 80 {
		t.Error("default")
	}
	_ = wfs.SaveWorkflow(ctx, &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, SuccessThreshold: 40})
	p2, _ := r.GetPolicy(ctx, "w")
	if p2.SuccessThreshold != 40 {
		t.Errorf("override: %+v", p2)
	}
	// Unknown workflow falls back to default
	p3, _ := r.GetPolicy(ctx, "missing")
	if p3.SuccessThreshold != 80 {
		t.Errorf("fallback: %+v", p3)
	}
}

func TestWorkflowStateManager(t *testing.T) {
	wfs := newMemWorkflowRepo()
	ctx := context.Background()
	m := NewWorkflowStateManager(wfs)
	_ = wfs.SaveWorkflow(ctx, &domain.Workflow{ID: "w", WorkflowType: "t", Activity: domain.ActivityReboot, Active: true})
	if !m.IsWorkflowActive(ctx, "w") {
		t.Error("active")
	}
	if err := m.DeactivateWorkflow(ctx, "w", "reason"); err != nil {
		t.Fatal(err)
	}
	if m.IsWorkflowActive(ctx, "w") {
		t.Error("expected deactivated")
	}
	if m.IsWorkflowActive(ctx, "missing") {
		t.Error("missing should not be active")
	}
	if err := m.ActivateWorkflow(ctx, "w", "manual"); err != nil {
		t.Fatal(err)
	}
	if !m.IsWorkflowActive(ctx, "w") {
		t.Error("expected active")
	}
}
