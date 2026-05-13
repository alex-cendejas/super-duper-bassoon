package domain

import (
	"testing"
	"time"
)

func TestCircuitState_CanDispatch(t *testing.T) {
	if !CircuitClosed.CanDispatch() {
		t.Error("closed should allow")
	}
	if !CircuitHalfOpen.CanDispatch() {
		t.Error("half open allows")
	}
	if CircuitOpen.CanDispatch() {
		t.Error("open blocks")
	}
	for _, s := range []CircuitState{CircuitClosed, CircuitOpen, CircuitHalfOpen} {
		if s.Description() == "unknown" {
			t.Errorf("unexpected desc for %s", s)
		}
	}
	if CircuitState("X").Description() != "unknown" {
		t.Error("unknown failed")
	}
}

func TestCircuitBreakerPolicy_Validate(t *testing.T) {
	if err := (&CircuitBreakerPolicy{SuccessThreshold: 80, EvaluationWindow: 10}).Validate(); err != nil {
		t.Error("good policy")
	}
	if err := (&CircuitBreakerPolicy{SuccessThreshold: -1}).Validate(); err == nil {
		t.Error("neg")
	}
	if err := (&CircuitBreakerPolicy{SuccessThreshold: 101}).Validate(); err == nil {
		t.Error(">100")
	}
	if err := (&CircuitBreakerPolicy{SuccessThreshold: 50, EvaluationWindow: -1}).Validate(); err == nil {
		t.Error("neg window")
	}
}

func TestCircuitBreakerLogic_EvaluateHealth(t *testing.T) {
	logic := NewCircuitBreakerLogic()
	policy := &CircuitBreakerPolicy{SuccessThreshold: 80, EvaluationWindow: 10}

	if logic.EvaluateHealth(nil, &WorkflowTypeHealth{}) != CircuitClosed {
		t.Error("nil policy => closed")
	}
	if logic.EvaluateHealth(policy, nil) != CircuitClosed {
		t.Error("nil health => closed")
	}
	if logic.EvaluateHealth(policy, &WorkflowTypeHealth{RunsConsidered: 0}) != CircuitClosed {
		t.Error("no runs => closed")
	}
	if logic.EvaluateHealth(policy, &WorkflowTypeHealth{RunsConsidered: 5, SuccessPercentageAvg: 90}) != CircuitClosed {
		t.Error("healthy => closed")
	}
	if logic.EvaluateHealth(policy, &WorkflowTypeHealth{RunsConsidered: 5, SuccessPercentageAvg: 50}) != CircuitOpen {
		t.Error("unhealthy => open")
	}
}

func TestWorkflowCircuitBreaker_IsRecoveryReady(t *testing.T) {
	b := &WorkflowCircuitBreaker{State: CircuitClosed}
	if b.IsRecoveryReady(time.Second, time.Now()) {
		t.Error("not open => not ready")
	}
	now := time.Now()
	b.State = CircuitOpen
	b.OpenedAt = now.Add(-10 * time.Second)
	if !b.IsRecoveryReady(5*time.Second, now) {
		t.Error("should be ready")
	}
	if b.IsRecoveryReady(20*time.Second, now) {
		t.Error("not yet ready")
	}
	// Zero OpenedAt => assume ready
	b.OpenedAt = time.Time{}
	if !b.IsRecoveryReady(time.Second, time.Now()) {
		t.Error("zero opened_at should be ready")
	}
	if !b.CanDispatch() {
		// open state should not dispatch
		if b.State == CircuitOpen && b.CanDispatch() {
			t.Error("open should not dispatch")
		}
	}
}

func TestPolicy_IsHealthyEnough(t *testing.T) {
	p := &CircuitBreakerPolicy{SuccessThreshold: 80}
	if !p.IsHealthyEnough(&WorkflowTypeHealth{RunsConsidered: 0}) {
		t.Error("0 runs => healthy")
	}
	if !p.IsHealthyEnough(&WorkflowTypeHealth{RunsConsidered: 5, SuccessPercentageAvg: 90}) {
		t.Error("90 => healthy")
	}
	if p.IsHealthyEnough(&WorkflowTypeHealth{RunsConsidered: 5, SuccessPercentageAvg: 50}) {
		t.Error("50 => unhealthy")
	}
}
