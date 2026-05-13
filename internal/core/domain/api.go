package domain

import "time"

type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *APIError) Error() string { return e.Message }

func (e *APIError) GetHTTPStatus() int {
	switch e.Code {
	case "NOT_FOUND":
		return 404
	case "BAD_REQUEST", "INVALID_FILTER", "VALIDATION_ERROR":
		return 400
	case "CONFLICT":
		return 409
	case "FORBIDDEN":
		return 403
	}
	return 500
}

func NewAPIError(code, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

type QueryFilter struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

func (q *QueryFilter) Validate() error {
	if q.Limit < 0 || q.Limit > 1000 {
		return ErrInvalidRequest
	}
	if q.Offset < 0 {
		return ErrInvalidRequest
	}
	return nil
}

func (q *QueryFilter) GetLimit() int {
	if q.Limit <= 0 {
		return 10
	}
	return q.Limit
}

func (q *QueryFilter) GetOffset() int {
	if q.Offset < 0 {
		return 0
	}
	return q.Offset
}

type CreateWorkflowRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	WorkflowType     string                 `json:"workflow_type"`
	Activity         ActivityType           `json:"activity"`
	Params           map[string]interface{} `json:"params"`
	TargetFilter     string                 `json:"target_filter"`
	Trigger          TriggerSpec            `json:"trigger"`
	SuccessThreshold float64                `json:"success_threshold"`
	LoopThresholdMS  int64                  `json:"loop_threshold_ms"`
	TimeoutMS        int64                  `json:"timeout_ms"`
	Enabled          *bool                  `json:"enabled,omitempty"`
}

func (r *CreateWorkflowRequest) Validate() error {
	if r.Name == "" {
		return NewAPIError("VALIDATION_ERROR", "name required")
	}
	if r.WorkflowType == "" {
		return NewAPIError("VALIDATION_ERROR", "workflow_type required")
	}
	if !r.Activity.IsValid() {
		return NewAPIError("VALIDATION_ERROR", "invalid activity")
	}
	return nil
}

type EditWorkflowRequest struct {
	Name             *string                `json:"name,omitempty"`
	Description      *string                `json:"description,omitempty"`
	Params           map[string]interface{} `json:"params,omitempty"`
	TargetFilter     *string                `json:"target_filter,omitempty"`
	SuccessThreshold *float64               `json:"success_threshold,omitempty"`
	LoopThresholdMS  *int64                 `json:"loop_threshold_ms,omitempty"`
	TimeoutMS        *int64                 `json:"timeout_ms,omitempty"`
	Enabled          *bool                  `json:"enabled,omitempty"`
}

func (r *EditWorkflowRequest) HasChanges() bool {
	return r.Name != nil || r.Description != nil || r.Params != nil ||
		r.TargetFilter != nil || r.SuccessThreshold != nil ||
		r.LoopThresholdMS != nil || r.TimeoutMS != nil || r.Enabled != nil
}

func (r *EditWorkflowRequest) Validate() error {
	if !r.HasChanges() {
		return NewAPIError("VALIDATION_ERROR", "at least one field required")
	}
	return nil
}

type TriggerWorkflowRequest struct {
	Reason string `json:"reason"`
}

type UnbanRequest struct {
	WorkflowType string `json:"workflow_type"`
	AdminID      string `json:"admin_id"`
	Reason       string `json:"reason"`
}

func (r *UnbanRequest) Validate() error {
	if r.AdminID == "" {
		return NewAPIError("VALIDATION_ERROR", "admin_id required")
	}
	if r.Reason == "" {
		return NewAPIError("VALIDATION_ERROR", "reason required")
	}
	return nil
}

type SystemStatus struct {
	Uptime       time.Duration `json:"uptime"`
	UptimeSec    int64         `json:"uptime_seconds"`
	DBStatus     string        `json:"db_status"`
	NATSStatus   string        `json:"nats_status"`
	StartedAt    time.Time     `json:"started_at"`
	Goroutines   int           `json:"goroutines"`
}
