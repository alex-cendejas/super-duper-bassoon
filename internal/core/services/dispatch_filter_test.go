package services

import (
	"context"
	"testing"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

func TestDispatchFilterService_Filter(t *testing.T) {
	bans := newMemBanRepo()
	blocker := newStubBlocker()
	enf := NewBanEnforcementService(bans, newStubAlerter(), blocker, nil)
	ctx := context.Background()
	_, _ = enf.BanClient(ctx, "c2", "t1", "r1", "ev", domain.ReasonLoopDetected)

	filter := NewDispatchFilterService(enf, nil)
	allowed, filtered, err := filter.FilterDispatchList(ctx, "t1", []string{"c1", "c2", "c3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(allowed) != 2 {
		t.Errorf("allowed: %v", allowed)
	}
	if len(filtered) != 1 || filtered[0] != "c2" {
		t.Errorf("filtered: %v", filtered)
	}

	ds := []*domain.Dispatch{
		{ClientID: "c1", RunID: "r", WorkflowID: "w", Activity: domain.ActivityReboot},
		{ClientID: "c2", RunID: "r", WorkflowID: "w", Activity: domain.ActivityReboot},
	}
	a, f, _ := filter.FilterDispatches(ctx, "t1", ds)
	if len(a) != 1 || len(f) != 1 {
		t.Errorf("dispatch filter: %d/%d", len(a), len(f))
	}
}
