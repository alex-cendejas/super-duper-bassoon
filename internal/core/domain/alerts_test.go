package domain

import "testing"

func TestNewAlert(t *testing.T) {
	a := NewAlert(AlertLoopDetected, SeverityCritical, "msg")
	if a.Kind != AlertLoopDetected {
		t.Error("kind")
	}
	if a.Severity != SeverityCritical {
		t.Error("severity")
	}
	if a.Message != "msg" {
		t.Error("msg")
	}
	if a.Timestamp.IsZero() {
		t.Error("ts")
	}
	if a.Details == nil {
		t.Error("details map")
	}
}

func TestEvents(t *testing.T) {
	h := &HealthUpdatedEvent{}
	if h.EventType() != "health.updated" {
		t.Error("health.updated")
	}
	c := &CircuitBreakerStateChangedEvent{}
	if c.EventType() != "circuit.state.changed" {
		t.Error("circuit.state.changed")
	}
	w := &WorkflowCompletionEvent{}
	if w.EventType() != "workflow.completed" {
		t.Error("workflow.completed")
	}
	_ = h.Time()
	_ = c.Time()
	_ = w.Time()
}
