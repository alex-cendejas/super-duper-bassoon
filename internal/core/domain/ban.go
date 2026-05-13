package domain

import "time"

type BanReason string

const (
	ReasonLoopDetected   BanReason = "loop_detected"
	ReasonManual         BanReason = "manual"
	ReasonAdminUnbanFail BanReason = "admin_unban_failed"
)

type BanRecord struct {
	ID             int64     `json:"id"`
	ClientID       string    `json:"client_id"`
	WorkflowType   string    `json:"workflow_type"`
	RunIDEvidence  string    `json:"run_id_evidence"`
	ResultEvidence string    `json:"result_evidence"`
	BannedAt       time.Time `json:"banned_at"`
	BannedUntil    *time.Time `json:"banned_until,omitempty"`
	Reason         BanReason `json:"reason"`
	BannedBy       string    `json:"banned_by"`
	Active         bool      `json:"active"`
}

func (b *BanRecord) IsActive() bool {
	if !b.Active {
		return false
	}
	if b.BannedUntil == nil {
		return true
	}
	return b.BannedUntil.After(time.Now())
}

func (b *BanRecord) IsPermanent() bool { return b.BannedUntil == nil }

func (b *BanRecord) CanUnban() bool {
	// All bans are unbannable (admin override)
	return b.Active
}

type LoopDetectionRecord struct {
	ClientID       string
	WorkflowType   string
	CurrentRunID   string
	PreviousRunID  string
	DetectedAt     time.Time
	ResultEvidence string
	TimeBetween    time.Duration
}

func (r *LoopDetectionRecord) IsValid() bool {
	return r.ClientID != "" && r.WorkflowType != "" && r.CurrentRunID != ""
}

type LoopDetector struct {
	defaultThreshold time.Duration
}

func NewLoopDetector(defaultThreshold time.Duration) *LoopDetector {
	if defaultThreshold <= 0 {
		defaultThreshold = 5 * time.Second
	}
	return &LoopDetector{defaultThreshold: defaultThreshold}
}

// DetectLoop returns a loop record when result.RunID is different from prevRun.RunID
// AND prevRun.TriggeredAt is within threshold of current's run-receive time.
func (d *LoopDetector) DetectLoop(clientID, workflowType, currentRunID string, currentRunTime time.Time, prevRunID string, prevRunTime time.Time, threshold time.Duration, evidence string) *LoopDetectionRecord {
	if threshold <= 0 {
		threshold = d.defaultThreshold
	}
	if prevRunID == "" || prevRunID == currentRunID {
		return nil
	}
	if currentRunTime.Before(prevRunTime) {
		return nil
	}
	delta := currentRunTime.Sub(prevRunTime)
	if delta > threshold {
		return nil
	}
	return &LoopDetectionRecord{
		ClientID:       clientID,
		WorkflowType:   workflowType,
		CurrentRunID:   currentRunID,
		PreviousRunID:  prevRunID,
		DetectedAt:     time.Now().UTC(),
		ResultEvidence: evidence,
		TimeBetween:    delta,
	}
}

type BanManager struct{}

func NewBanManager() *BanManager { return &BanManager{} }

func (m *BanManager) CanDispatchToClient(clientID, workflowType string, bans []*BanRecord) bool {
	for _, b := range bans {
		if b.ClientID != clientID {
			continue
		}
		if b.WorkflowType != "" && b.WorkflowType != workflowType {
			continue
		}
		if b.IsActive() {
			return false
		}
	}
	return true
}

func (m *BanManager) ApplyBan(clientID, workflowType, runID, evidence string, reason BanReason) *BanRecord {
	return &BanRecord{
		ClientID:       clientID,
		WorkflowType:   workflowType,
		RunIDEvidence:  runID,
		ResultEvidence: evidence,
		BannedAt:       time.Now().UTC(),
		Reason:         reason,
		BannedBy:       "system",
		Active:         true,
	}
}
