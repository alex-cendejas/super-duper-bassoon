package domain

import "testing"

func TestRunHealth_Percentages(t *testing.T) {
	h := &RunHealth{TotalClients: 10, BannedClientCount: 2, SuccessCount: 4, FailCount: 2, ErrorCount: 1, PendingCount: 1}
	// effective = 8
	if got := h.SuccessPercentage(); got != 50 {
		t.Errorf("success pct: %v", got)
	}
	if got := h.FailPercentage(); got != 25 {
		t.Errorf("fail pct: %v", got)
	}
	if got := h.ErrorPercentage(); got != 12.5 {
		t.Errorf("error pct: %v", got)
	}
	if got := h.PendingPercentage(); got != 12.5 {
		t.Errorf("pending pct: %v", got)
	}
	if h.IsComplete() {
		t.Error("should not be complete")
	}
	if h.CompletedCount() != 7 {
		t.Errorf("completed: %d", h.CompletedCount())
	}

	zero := &RunHealth{}
	if got := zero.SuccessPercentage(); got != 0 {
		t.Error("zero clients pct should be 0")
	}
	if got := zero.FailPercentage(); got != 0 {
		t.Error("zero pct")
	}
	if got := zero.ErrorPercentage(); got != 0 {
		t.Error("zero pct")
	}
	if got := zero.PendingPercentage(); got != 0 {
		t.Error("zero pct")
	}
	if !zero.IsComplete() {
		t.Error("zero pending should be complete")
	}
}

func TestHealthAggregator_CalculateRunHealth(t *testing.T) {
	a := NewHealthAggregator()
	results := []*Result{
		{ClientID: "c1", Status: StatusSuccess},
		{ClientID: "c2", Status: StatusFail},
		{ClientID: "c3", Status: StatusError},
		{ClientID: "c1", Status: StatusFail}, // duplicate, ignored
		nil,                                  // nil ignored
	}
	h := a.CalculateRunHealth("run1", "wf1", "type1", 5, results, 1)
	if h.SuccessCount != 1 || h.FailCount != 1 || h.ErrorCount != 1 {
		t.Errorf("counts wrong: %+v", h)
	}
	// pending = total - banned - completed = 5 - 1 - 3 = 1
	if h.PendingCount != 1 {
		t.Errorf("pending wrong: %d", h.PendingCount)
	}
}

func TestHealthAggregator_AggregateWorkflowHealth(t *testing.T) {
	a := NewHealthAggregator()
	out := a.AggregateWorkflowHealth("t", nil, 5)
	if out.RunsConsidered != 0 {
		t.Error("empty")
	}
	if out.Trend != TrendStable {
		t.Error("trend")
	}

	r1 := &RunHealth{TotalClients: 2, SuccessCount: 2} // 100%
	r2 := &RunHealth{TotalClients: 2, SuccessCount: 1, FailCount: 1} // 50%
	r3 := &RunHealth{TotalClients: 2, SuccessCount: 0, FailCount: 2} // 0%
	out = a.AggregateWorkflowHealth("t", []*RunHealth{r1, r2, r3}, 10)
	if out.RunsConsidered != 3 {
		t.Errorf("runs: %d", out.RunsConsidered)
	}
	if out.SuccessPercentageAvg != 50 {
		t.Errorf("avg: %v", out.SuccessPercentageAvg)
	}
	if out.Trend != TrendDegrading {
		t.Errorf("expected degrading: %v", out.Trend)
	}

	// Improving
	out = a.AggregateWorkflowHealth("t", []*RunHealth{r3, r2, r1}, 10)
	if out.Trend != TrendImproving {
		t.Errorf("expected improving: %v", out.Trend)
	}

	// Window truncation
	out = a.AggregateWorkflowHealth("t", []*RunHealth{r1, r2, r3, r1}, 2)
	if out.RunsConsidered != 2 {
		t.Errorf("window: %d", out.RunsConsidered)
	}
}

func TestWorkflowTypeHealth_IsHealthy(t *testing.T) {
	h := &WorkflowTypeHealth{RunsConsidered: 0, SuccessPercentageAvg: 0}
	if !h.IsHealthy(80) {
		t.Error("0 runs => healthy")
	}
	h.RunsConsidered = 5
	h.SuccessPercentageAvg = 90
	if !h.IsHealthy(80) {
		t.Error("90%")
	}
	h.SuccessPercentageAvg = 50
	if h.IsHealthy(80) {
		t.Error("50% should not pass")
	}
}

func TestHealthThreshold_Validate(t *testing.T) {
	if err := (&HealthThreshold{SuccessThreshold: 80, WindowSize: 5}).Validate(); err != nil {
		t.Error("good threshold")
	}
	if err := (&HealthThreshold{SuccessThreshold: -1}).Validate(); err == nil {
		t.Error("neg")
	}
	if err := (&HealthThreshold{SuccessThreshold: 200}).Validate(); err == nil {
		t.Error(">100")
	}
	if err := (&HealthThreshold{SuccessThreshold: 50, WindowSize: -1}).Validate(); err == nil {
		t.Error("neg window")
	}
}
