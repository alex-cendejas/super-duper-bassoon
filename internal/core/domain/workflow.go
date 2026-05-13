package domain

import (
	"time"
)

type TriggerKind string

const (
	TriggerScheduled   TriggerKind = "scheduled"
	TriggerEvent       TriggerKind = "event"
	TriggerStateChange TriggerKind = "state_change"
	TriggerManual      TriggerKind = "manual"
)

type TriggerSpec struct {
	Kind       TriggerKind            `json:"kind"`
	Cron       string                 `json:"cron,omitempty"`
	OnComplete string                 `json:"on_complete,omitempty"`
	Condition  string                 `json:"condition,omitempty"`
	Params     map[string]interface{} `json:"params,omitempty"`
}

type Workflow struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	WorkflowType      string                 `json:"workflow_type"`
	Activity          ActivityType           `json:"activity"`
	Params            map[string]interface{} `json:"params"`
	TargetFilter      string                 `json:"target_filter"`
	Trigger           TriggerSpec            `json:"trigger"`
	SuccessThreshold  float64                `json:"success_threshold"`
	LoopThresholdMS   int64                  `json:"loop_threshold_ms"`
	TimeoutMS         int64                  `json:"timeout_ms"`
	Active            bool                   `json:"active"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	DeactivatedReason string                 `json:"deactivated_reason,omitempty"`
}

func (w *Workflow) IsActive() bool { return w.Active }

func (w *Workflow) ValidateDefinition() error {
	if w.Name == "" {
		return ErrMissingRequiredField
	}
	if w.WorkflowType == "" {
		return ErrMissingRequiredField
	}
	if !w.Activity.IsValid() {
		return ErrInvalidWorkflow
	}
	if w.SuccessThreshold < 0 || w.SuccessThreshold > 100 {
		return ErrInvalidWorkflow
	}
	if w.LoopThresholdMS < 0 {
		return ErrInvalidWorkflow
	}
	if w.TimeoutMS < 0 {
		return ErrInvalidWorkflow
	}
	return nil
}

func (w *Workflow) GetActivityTimeout() time.Duration {
	if w.TimeoutMS <= 0 {
		return 30 * time.Second
	}
	return time.Duration(w.TimeoutMS) * time.Millisecond
}

func (w *Workflow) LoopThreshold() time.Duration {
	if w.LoopThresholdMS <= 0 {
		return 5 * time.Second
	}
	return time.Duration(w.LoopThresholdMS) * time.Millisecond
}
