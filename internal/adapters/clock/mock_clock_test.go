package clock_test

import (
	"testing"
	"time"

	"github.com/super-duper-bassoon/internal/adapters/clock"
)

func TestMockClock_Now(t *testing.T) {
	start := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	c := clock.NewMockClock(start)
	if got := c.Now(); !got.Equal(start) {
		t.Errorf("expected %v, got %v", start, got)
	}
}

func TestMockClock_Sleep_AdvancesTime(t *testing.T) {
	start := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	c := clock.NewMockClock(start)

	c.Sleep(5 * time.Second)
	expected := start.Add(5 * time.Second)
	if got := c.Now(); !got.Equal(expected) {
		t.Errorf("expected %v after sleep, got %v", expected, got)
	}
}

func TestMockClock_Sleep_NonBlocking(t *testing.T) {
	c := clock.NewMockClock(time.Now())
	done := make(chan struct{})
	go func() {
		c.Sleep(1 * time.Hour) // should not block
		close(done)
	}()
	select {
	case <-done:
		// good
	case <-time.After(1 * time.Second):
		t.Error("MockClock.Sleep blocked for too long")
	}
}

func TestMockClock_Advance(t *testing.T) {
	start := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	c := clock.NewMockClock(start)
	c.Advance(10 * time.Minute)
	expected := start.Add(10 * time.Minute)
	if got := c.Now(); !got.Equal(expected) {
		t.Errorf("expected %v after advance, got %v", expected, got)
	}
}

func TestMockClock_SleptDurations(t *testing.T) {
	c := clock.NewMockClock(time.Now())
	c.Sleep(1 * time.Second)
	c.Sleep(2 * time.Second)
	c.Sleep(3 * time.Second)

	slept := c.SleptDurations()
	if len(slept) != 3 {
		t.Fatalf("expected 3 slept durations, got %d", len(slept))
	}
	if slept[0] != 1*time.Second {
		t.Errorf("expected 1s, got %v", slept[0])
	}
	if slept[1] != 2*time.Second {
		t.Errorf("expected 2s, got %v", slept[1])
	}
	if slept[2] != 3*time.Second {
		t.Errorf("expected 3s, got %v", slept[2])
	}
}

func TestSystemClock_Now(t *testing.T) {
	sc := clock.SystemClock{}
	before := time.Now()
	got := sc.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Errorf("SystemClock.Now() returned %v, expected between %v and %v", got, before, after)
	}
}
