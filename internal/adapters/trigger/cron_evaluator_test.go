package trigger

import (
	"testing"
	"time"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

func TestCronEvaluator_ShouldFire(t *testing.T) {
	e := NewCronEvaluator()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// First call seeds next fire time
	got, err := e.ShouldFire("w1", domain.TriggerSpec{Kind: domain.TriggerScheduled, Cron: "* * * * *"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("first call shouldn't fire")
	}
	// Advance time past next minute
	next := now.Add(2 * time.Minute)
	got, _ = e.ShouldFire("w1", domain.TriggerSpec{Kind: domain.TriggerScheduled, Cron: "* * * * *"}, next)
	if !got {
		t.Error("should fire after interval")
	}

	// Non-scheduled trigger
	got, _ = e.ShouldFire("w2", domain.TriggerSpec{Kind: domain.TriggerManual}, now)
	if got {
		t.Error("manual should not fire")
	}
	// Empty cron
	got, _ = e.ShouldFire("w3", domain.TriggerSpec{Kind: domain.TriggerScheduled, Cron: ""}, now)
	if got {
		t.Error("empty cron")
	}
	// Bad cron
	if _, err := e.ShouldFire("w4", domain.TriggerSpec{Kind: domain.TriggerScheduled, Cron: "not a cron"}, now); err == nil {
		t.Error("expected parse error")
	}
	e.Reset("w1")
}
