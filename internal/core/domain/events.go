package domain

import "time"

type Event interface {
	EventType() string
	Time() time.Time
}

type HealthUpdatedEvent struct {
	WorkflowID   string
	WorkflowType string
	RunHealth    *RunHealth
	TypeHealth   *WorkflowTypeHealth
	Timestamp    time.Time
}

func (e *HealthUpdatedEvent) EventType() string { return "health.updated" }
func (e *HealthUpdatedEvent) Time() time.Time   { return e.Timestamp }

type CircuitBreakerStateChangedEvent struct {
	WorkflowID   string
	WorkflowType string
	OldState     CircuitState
	NewState     CircuitState
	Reason       string
	Timestamp    time.Time
}

func (e *CircuitBreakerStateChangedEvent) EventType() string { return "circuit.state.changed" }
func (e *CircuitBreakerStateChangedEvent) Time() time.Time   { return e.Timestamp }

type WorkflowCompletionEvent struct {
	WorkflowID   string
	WorkflowType string
	RunID        string
	State        RunState
	Timestamp    time.Time
}

func (e *WorkflowCompletionEvent) EventType() string { return "workflow.completed" }
func (e *WorkflowCompletionEvent) Time() time.Time   { return e.Timestamp }
