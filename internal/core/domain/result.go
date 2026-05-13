package domain

import (
	"encoding/json"
	"time"
)

const (
	StatusSuccess ResultStatus = "success"
	StatusFail    ResultStatus = "fail"
	StatusError   ResultStatus = "error"
)

type Result struct {
	RunID      string                 `json:"run_id"`
	WorkflowID string                 `json:"wf_id"`
	ClientID   string                 `json:"client_id"`
	Status     ResultStatus           `json:"status"`
	InnerState map[string]interface{} `json:"inner_state,omitempty"`
	ErrorMsg   string                 `json:"error_msg,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	ReceivedAt time.Time              `json:"received_at"`
}

func (r *Result) IsValid() bool {
	if r.RunID == "" || r.ClientID == "" || r.WorkflowID == "" {
		return false
	}
	if !r.Status.IsValid() {
		return false
	}
	return true
}

func ParseResult(data []byte) (*Result, error) {
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.ReceivedAt.IsZero() {
		r.ReceivedAt = time.Now().UTC()
	}
	return &r, nil
}
