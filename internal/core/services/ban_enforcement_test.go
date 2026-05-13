package services

import (
	"context"
	"testing"

	"github.com/super-duper-bassoon/internal/core/domain"
)

func TestBanEnforcement_BanAndUnban(t *testing.T) {
	bans := newMemBanRepo()
	alerter := newStubAlerter()
	blocker := newStubBlocker()
	enf := NewBanEnforcementService(bans, alerter, blocker, nil)
	ctx := context.Background()

	b, err := enf.BanClient(ctx, "c1", "t1", "r1", "evidence", domain.ReasonLoopDetected)
	if err != nil {
		t.Fatal(err)
	}
	if !b.Active {
		t.Error("expected active")
	}
	if !blocker.ShouldBlockDispatch(ctx, "c1", "t1") {
		t.Error("blocker not updated")
	}
	if alerter.Count(domain.AlertClientBanned) != 1 {
		t.Error("expected ban alert")
	}

	isB, err := enf.IsBanned(ctx, "c1", "t1")
	if err != nil || !isB {
		t.Errorf("expected banned: %v, %v", isB, err)
	}
	isB, _ = enf.IsBanned(ctx, "c1", "different")
	if isB {
		t.Error("not banned for other type")
	}

	if err := enf.UnbanClient(ctx, "c1", "t1", "admin", "ok"); err != nil {
		t.Fatal(err)
	}
	if blocker.ShouldBlockDispatch(ctx, "c1", "t1") {
		t.Error("blocker should be cleared")
	}
	if alerter.Count(domain.AlertClientUnbanned) != 1 {
		t.Error("expected unban alert")
	}
	// Unbanning a missing ban yields error
	if err := enf.UnbanClient(ctx, "c1", "t1", "admin", "ok"); err == nil {
		t.Error("expected error")
	}
}

func TestBanEnforcement_WarmCache(t *testing.T) {
	bans := newMemBanRepo()
	ctx := context.Background()
	_ = bans.SaveBan(ctx, &domain.BanRecord{ClientID: "c1", WorkflowType: "t1", Active: true})
	_ = bans.SaveBan(ctx, &domain.BanRecord{ClientID: "c2", WorkflowType: "t1", Active: false}) // inactive

	blocker := newStubBlocker()
	enf := NewBanEnforcementService(bans, newStubAlerter(), blocker, nil)
	if err := enf.WarmCache(ctx); err != nil {
		t.Fatal(err)
	}
	if !blocker.ShouldBlockDispatch(ctx, "c1", "t1") {
		t.Error("c1 should be cached")
	}
	if blocker.ShouldBlockDispatch(ctx, "c2", "t1") {
		t.Error("inactive ban should not be cached")
	}
}
