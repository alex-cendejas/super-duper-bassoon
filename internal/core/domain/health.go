package domain

import "time"

type TrendDirection string

const (
	TrendImproving TrendDirection = "improving"
	TrendDegrading TrendDirection = "degrading"
	TrendStable    TrendDirection = "stable"
)

type RunHealth struct {
	RunID             string    `json:"run_id"`
	WorkflowID        string    `json:"workflow_id"`
	WorkflowType      string    `json:"workflow_type"`
	TotalClients      int       `json:"total_clients"`
	SuccessCount      int       `json:"success_count"`
	FailCount         int       `json:"fail_count"`
	ErrorCount        int       `json:"error_count"`
	PendingCount      int       `json:"pending_count"`
	BannedClientCount int       `json:"banned_client_count"`
	CalculatedAt      time.Time `json:"calculated_at"`
}

func (h *RunHealth) effective() int {
	d := h.TotalClients - h.BannedClientCount
	if d <= 0 {
		return 0
	}
	return d
}

func (h *RunHealth) CompletedCount() int {
	return h.SuccessCount + h.FailCount + h.ErrorCount
}

func (h *RunHealth) SuccessPercentage() float64 {
	d := h.effective()
	if d == 0 {
		return 0
	}
	return 100.0 * float64(h.SuccessCount) / float64(d)
}

func (h *RunHealth) FailPercentage() float64 {
	d := h.effective()
	if d == 0 {
		return 0
	}
	return 100.0 * float64(h.FailCount) / float64(d)
}

func (h *RunHealth) ErrorPercentage() float64 {
	d := h.effective()
	if d == 0 {
		return 0
	}
	return 100.0 * float64(h.ErrorCount) / float64(d)
}

func (h *RunHealth) PendingPercentage() float64 {
	d := h.effective()
	if d == 0 {
		return 0
	}
	return 100.0 * float64(h.PendingCount) / float64(d)
}

func (h *RunHealth) IsComplete() bool { return h.PendingCount == 0 }

type WorkflowTypeHealth struct {
	WorkflowType         string         `json:"workflow_type"`
	WindowSize           int            `json:"window_size"`
	RunsConsidered       int            `json:"runs_considered"`
	SuccessPercentageAvg float64        `json:"success_percentage_avg"`
	FailPercentageAvg    float64        `json:"fail_percentage_avg"`
	ErrorPercentageAvg   float64        `json:"error_percentage_avg"`
	Trend                TrendDirection `json:"trend"`
	CalculatedAt         time.Time      `json:"calculated_at"`
}

func (w *WorkflowTypeHealth) IsHealthy(threshold float64) bool {
	if w.RunsConsidered == 0 {
		return true
	}
	return w.SuccessPercentageAvg >= threshold
}

type HealthThreshold struct {
	SuccessThreshold float64
	WindowSize       int
}

func (t *HealthThreshold) Validate() error {
	if t.SuccessThreshold < 0 || t.SuccessThreshold > 100 {
		return ErrInvalidPolicy
	}
	if t.WindowSize < 0 {
		return ErrInvalidPolicy
	}
	return nil
}

type HealthAggregator struct{}

func NewHealthAggregator() *HealthAggregator { return &HealthAggregator{} }

func (a *HealthAggregator) CalculateRunHealth(runID, workflowID, workflowType string, totalClients int, results []*Result, bannedCount int) *RunHealth {
	h := &RunHealth{
		RunID:             runID,
		WorkflowID:        workflowID,
		WorkflowType:      workflowType,
		TotalClients:      totalClients,
		BannedClientCount: bannedCount,
		CalculatedAt:      time.Now().UTC(),
	}
	seen := make(map[string]ResultStatus)
	for _, r := range results {
		if r == nil {
			continue
		}
		if _, ok := seen[r.ClientID]; ok {
			continue
		}
		seen[r.ClientID] = r.Status
	}
	for _, s := range seen {
		switch s {
		case StatusSuccess:
			h.SuccessCount++
		case StatusFail:
			h.FailCount++
		case StatusError:
			h.ErrorCount++
		}
	}
	completed := h.CompletedCount()
	pending := totalClients - bannedCount - completed
	if pending < 0 {
		pending = 0
	}
	h.PendingCount = pending
	return h
}

func (a *HealthAggregator) AggregateWorkflowHealth(workflowType string, runs []*RunHealth, windowSize int) *WorkflowTypeHealth {
	out := &WorkflowTypeHealth{
		WorkflowType: workflowType,
		WindowSize:   windowSize,
		CalculatedAt: time.Now().UTC(),
		Trend:        TrendStable,
	}
	if len(runs) == 0 {
		return out
	}
	if windowSize > 0 && len(runs) > windowSize {
		runs = runs[len(runs)-windowSize:]
	}
	var sumS, sumF, sumE float64
	for _, r := range runs {
		sumS += r.SuccessPercentage()
		sumF += r.FailPercentage()
		sumE += r.ErrorPercentage()
	}
	n := float64(len(runs))
	out.RunsConsidered = len(runs)
	out.SuccessPercentageAvg = sumS / n
	out.FailPercentageAvg = sumF / n
	out.ErrorPercentageAvg = sumE / n
	out.Trend = a.CalculateTrend(runs)
	return out
}

func (a *HealthAggregator) CalculateTrend(runs []*RunHealth) TrendDirection {
	if len(runs) < 2 {
		return TrendStable
	}
	half := len(runs) / 2
	if half == 0 {
		return TrendStable
	}
	var older, newer float64
	for _, r := range runs[:half] {
		older += r.SuccessPercentage()
	}
	for _, r := range runs[len(runs)-half:] {
		newer += r.SuccessPercentage()
	}
	older /= float64(half)
	newer /= float64(half)
	delta := newer - older
	switch {
	case delta > 1:
		return TrendImproving
	case delta < -1:
		return TrendDegrading
	default:
		return TrendStable
	}
}
