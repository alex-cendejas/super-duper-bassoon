package domain

import (
	"time"

	"github.com/google/uuid"
)

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

type AlertKind string

const (
	AlertLoopDetected      AlertKind = "loop_detected"
	AlertClientBanned      AlertKind = "client_banned"
	AlertClientUnbanned    AlertKind = "client_unbanned"
	AlertCircuitOpened     AlertKind = "circuit_opened"
	AlertCircuitClosed     AlertKind = "circuit_closed"
	AlertWorkflowDeactivated AlertKind = "workflow_deactivated"
	AlertWorkflowActivated AlertKind = "workflow_activated"
)

type Alert struct {
	ID        string                 `json:"id"`
	Kind      AlertKind              `json:"kind"`
	Severity  AlertSeverity          `json:"severity"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

func NewAlert(kind AlertKind, severity AlertSeverity, message string) *Alert {
	return &Alert{
		ID:        uuid.NewString(),
		Kind:      kind,
		Severity:  severity,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Details:   map[string]interface{}{},
	}
}
