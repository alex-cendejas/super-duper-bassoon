package domain

import (
	"encoding/json"
	"time"
)

type Dispatch struct {
	RunID        string                 `json:"run_id"`
	WorkflowID   string                 `json:"wf_id"`
	ClientID     string                 `json:"client_id,omitempty"`
	Activity     ActivityType           `json:"activity"`
	Params       map[string]interface{} `json:"params"`
	DispatchedAt time.Time              `json:"dispatched_at,omitempty"`
}

func (d *Dispatch) IsValid() bool {
	if d.RunID == "" || d.WorkflowID == "" {
		return false
	}
	if !d.Activity.IsValid() {
		return false
	}
	return true
}

func (d *Dispatch) GetPayload() ([]byte, error) {
	type wire struct {
		RunID    string                 `json:"run_id"`
		WfID     string                 `json:"wf_id"`
		Activity ActivityType           `json:"activity"`
		Params   map[string]interface{} `json:"params"`
	}
	w := wire{
		RunID:    d.RunID,
		WfID:     d.WorkflowID,
		Activity: d.Activity,
		Params:   d.Params,
	}
	return json.Marshal(w)
}
