package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/super-duper-bassoon/server/internal/core/domain"
	"github.com/super-duper-bassoon/server/internal/core/ports"
)

type LoopDetectionService struct {
	runs        ports.RunRepository
	workflows   ports.WorkflowRepository
	bans        ports.BanRepository
	enforcer    *BanEnforcementService
	alerts      ports.AlertPublisher
	detector    *domain.LoopDetector
	defaultMS   int64
	logger      *log.Logger
}

func NewLoopDetectionService(runs ports.RunRepository, workflows ports.WorkflowRepository, bans ports.BanRepository, enforcer *BanEnforcementService, alerts ports.AlertPublisher, defaultThresholdMS int64, logger *log.Logger) *LoopDetectionService {
	if logger == nil {
		logger = log.Default()
	}
	dur := time.Duration(defaultThresholdMS) * time.Millisecond
	return &LoopDetectionService{
		runs:      runs,
		workflows: workflows,
		bans:      bans,
		enforcer:  enforcer,
		alerts:    alerts,
		detector:  domain.NewLoopDetector(dur),
		defaultMS: defaultThresholdMS,
		logger:    logger,
	}
}

func (s *LoopDetectionService) Name() string  { return "loop_detection" }
func (s *LoopDetectionService) Priority() int { return 1 }

func (s *LoopDetectionService) HandleResult(ctx context.Context, r *domain.Result) error {
	if r == nil || !r.IsValid() {
		return nil
	}
	wf, err := s.workflows.GetWorkflow(ctx, r.WorkflowID)
	if err != nil {
		s.logger.Printf("loop_detection: workflow not found %s: %v", r.WorkflowID, err)
		return nil
	}
	threshold := wf.LoopThreshold()
	if threshold <= 0 {
		threshold = time.Duration(s.defaultMS) * time.Millisecond
	}
	currentRun, err := s.runs.GetRun(ctx, r.RunID)
	if err != nil {
		s.logger.Printf("loop_detection: current run not found %s: %v", r.RunID, err)
		return nil
	}
	prev, err := s.runs.GetPreviousRun(ctx, r.ClientID, wf.WorkflowType, r.RunID, currentRun.TriggeredAt)
	if err != nil {
		// no previous run => no loop
		return nil
	}
	if prev == nil {
		return nil
	}
	evidence, _ := json.Marshal(r)
	rec := s.detector.DetectLoop(r.ClientID, wf.WorkflowType, r.RunID, currentRun.TriggeredAt, prev.RunID, prev.TriggeredAt, threshold, string(evidence))
	if rec == nil {
		return nil
	}
	if s.alerts != nil {
		alert := domain.NewAlert(domain.AlertLoopDetected, domain.SeverityCritical, fmt.Sprintf("loop detected client=%s wf_type=%s", r.ClientID, wf.WorkflowType))
		alert.Details["client_id"] = r.ClientID
		alert.Details["workflow_type"] = wf.WorkflowType
		alert.Details["current_run"] = rec.CurrentRunID
		alert.Details["previous_run"] = rec.PreviousRunID
		alert.Details["time_between_ms"] = rec.TimeBetween.Milliseconds()
		_ = s.alerts.PublishAlert(ctx, alert)
	}
	if s.enforcer != nil {
		if _, banErr := s.enforcer.BanClient(ctx, r.ClientID, wf.WorkflowType, r.RunID, string(evidence), domain.ReasonLoopDetected); banErr != nil {
			s.logger.Printf("loop_detection: ban failed: %v", banErr)
		}
	}
	return nil
}
