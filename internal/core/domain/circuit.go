package domain

import "time"

type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

func (s CircuitState) CanDispatch() bool {
	return s == CircuitClosed || s == CircuitHalfOpen
}

func (s CircuitState) Description() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	}
	return "unknown"
}

type CircuitBreakerPolicy struct {
	SuccessThreshold float64
	EvaluationWindow int
	CooldownPeriod   time.Duration
}

func (p *CircuitBreakerPolicy) Validate() error {
	if p.SuccessThreshold < 0 || p.SuccessThreshold > 100 {
		return ErrInvalidPolicy
	}
	if p.EvaluationWindow < 0 {
		return ErrInvalidPolicy
	}
	return nil
}

func (p *CircuitBreakerPolicy) IsHealthyEnough(h *WorkflowTypeHealth) bool {
	if h.RunsConsidered == 0 {
		return true
	}
	return h.SuccessPercentageAvg >= p.SuccessThreshold
}

type WorkflowCircuitBreaker struct {
	WorkflowID      string       `json:"workflow_id"`
	WorkflowType    string       `json:"workflow_type"`
	State           CircuitState `json:"state"`
	OpenedAt        time.Time    `json:"opened_at,omitempty"`
	LastEvaluatedAt time.Time    `json:"last_evaluated_at"`
	OpenedReason    string       `json:"opened_reason,omitempty"`
	EvaluationCount int          `json:"evaluation_count"`
}

func (b *WorkflowCircuitBreaker) CanDispatch() bool { return b.State.CanDispatch() }

func (b *WorkflowCircuitBreaker) IsRecoveryReady(cooldown time.Duration, now time.Time) bool {
	if b.State != CircuitOpen {
		return false
	}
	if b.OpenedAt.IsZero() {
		return true
	}
	return now.Sub(b.OpenedAt) >= cooldown
}

type CircuitBreakerLogic struct{}

func NewCircuitBreakerLogic() *CircuitBreakerLogic { return &CircuitBreakerLogic{} }

func (l *CircuitBreakerLogic) EvaluateHealth(policy *CircuitBreakerPolicy, h *WorkflowTypeHealth) CircuitState {
	if policy == nil || h == nil {
		return CircuitClosed
	}
	if h.RunsConsidered == 0 {
		return CircuitClosed
	}
	if policy.IsHealthyEnough(h) {
		return CircuitClosed
	}
	return CircuitOpen
}

type CircuitBreakerEvent struct {
	WorkflowID   string
	WorkflowType string
	OldState     CircuitState
	NewState     CircuitState
	Reason       string
	Timestamp    time.Time
}
