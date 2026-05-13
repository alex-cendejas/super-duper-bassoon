package enforcement

import (
	"context"
	"testing"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

func TestInMemoryDispatchBlocker(t *testing.T) {
	b := NewInMemoryDispatchBlocker()
	ctx := context.Background()
	// nil and inactive bans are no-ops
	b.Add(nil)
	b.Add(&domain.BanRecord{Active: false})
	if b.ShouldBlockDispatch(ctx, "c1", "t1") {
		t.Error("nothing added")
	}
	b.Add(&domain.BanRecord{ClientID: "c1", WorkflowType: "t1", Active: true})
	if !b.ShouldBlockDispatch(ctx, "c1", "t1") {
		t.Error("blocked")
	}
	if b.ShouldBlockDispatch(ctx, "c1", "t2") {
		t.Error("not blocked for other type")
	}
	// Global ban
	b.Add(&domain.BanRecord{ClientID: "c2", WorkflowType: "", Active: true})
	if !b.ShouldBlockDispatch(ctx, "c2", "anything") {
		t.Error("global ban")
	}
	// Remove
	b.Remove("c1", "t1")
	if b.ShouldBlockDispatch(ctx, "c1", "t1") {
		t.Error("should be unblocked")
	}
	// Remove from non-existing bucket
	b.Remove("c5", "nope")
}
