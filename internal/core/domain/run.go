package domain

import "time"

type RunState string

const (
	RunPending    RunState = "pending"
	RunInProgress RunState = "in_progress"
	RunCompleted  RunState = "completed"
	RunFailed     RunState = "failed"
)

type Run struct {
	RunID                string    `json:"run_id"`
	WorkflowID           string    `json:"workflow_id"`
	WorkflowType         string    `json:"workflow_type"`
	TriggeredAt          time.Time `json:"triggered_at"`
	DispatchedAt         time.Time `json:"dispatched_at"`
	ParticipatingClients []string  `json:"participating_clients"`
	State                RunState  `json:"state"`
	Reason               string    `json:"reason,omitempty"`
}

func (r *Run) IsExpired(timeout time.Duration, now time.Time) bool {
	if r.TriggeredAt.IsZero() {
		return false
	}
	return now.Sub(r.TriggeredAt) > timeout
}

func (r *Run) CanComplete(results int) bool {
	return results >= len(r.ParticipatingClients)
}
