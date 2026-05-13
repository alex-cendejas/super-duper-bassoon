package clock

import (
	"sync"
	"time"
)

// MockClock is a controllable clock for deterministic tests.
type MockClock struct {
	mu      sync.Mutex
	current time.Time
	slept   []time.Duration
}

// NewMockClock creates a MockClock set to the given start time.
func NewMockClock(start time.Time) *MockClock {
	return &MockClock{current: start}
}

func (c *MockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

// Sleep records the duration but does not block.
func (c *MockClock) Sleep(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = c.current.Add(d)
	c.slept = append(c.slept, d)
}

// Advance moves the clock forward by d.
func (c *MockClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = c.current.Add(d)
}

// SleptDurations returns all durations passed to Sleep.
func (c *MockClock) SleptDurations() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]time.Duration, len(c.slept))
	copy(result, c.slept)
	return result
}
