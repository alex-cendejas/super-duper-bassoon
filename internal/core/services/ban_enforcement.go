package services

import (
	"context"
	"fmt"
	"log"

	"github.com/super-duper-bassoon/server/internal/core/domain"
	"github.com/super-duper-bassoon/server/internal/core/ports"
)

type BanEnforcementService struct {
	manager *domain.BanManager
	bans    ports.BanRepository
	alerts  ports.AlertPublisher
	blocker ports.DispatchBlocker
	logger  *log.Logger
}

func NewBanEnforcementService(bans ports.BanRepository, alerts ports.AlertPublisher, blocker ports.DispatchBlocker, logger *log.Logger) *BanEnforcementService {
	if logger == nil {
		logger = log.Default()
	}
	return &BanEnforcementService{
		manager: domain.NewBanManager(),
		bans:    bans,
		alerts:  alerts,
		blocker: blocker,
		logger:  logger,
	}
}

func (s *BanEnforcementService) BanClient(ctx context.Context, clientID, workflowType, runID, evidence string, reason domain.BanReason) (*domain.BanRecord, error) {
	ban := s.manager.ApplyBan(clientID, workflowType, runID, evidence, reason)
	if err := s.bans.SaveBan(ctx, ban); err != nil {
		return nil, err
	}
	if s.blocker != nil {
		s.blocker.Add(ban)
	}
	if s.alerts != nil {
		alert := domain.NewAlert(domain.AlertClientBanned, domain.SeverityCritical, fmt.Sprintf("client %s banned from %s", clientID, workflowType))
		alert.Details["client_id"] = clientID
		alert.Details["workflow_type"] = workflowType
		alert.Details["run_id_evidence"] = runID
		alert.Details["reason"] = string(reason)
		_ = s.alerts.PublishAlert(ctx, alert)
	}
	s.logger.Printf("banned client=%s wf_type=%s reason=%s", clientID, workflowType, reason)
	return ban, nil
}

func (s *BanEnforcementService) UnbanClient(ctx context.Context, clientID, workflowType, adminID, reason string) error {
	if err := s.bans.UnbanClient(ctx, clientID, workflowType); err != nil {
		return err
	}
	if s.blocker != nil {
		s.blocker.Remove(clientID, workflowType)
	}
	if s.alerts != nil {
		alert := domain.NewAlert(domain.AlertClientUnbanned, domain.SeverityInfo, fmt.Sprintf("client %s unbanned from %s by %s", clientID, workflowType, adminID))
		alert.Details["client_id"] = clientID
		alert.Details["workflow_type"] = workflowType
		alert.Details["admin_id"] = adminID
		alert.Details["reason"] = reason
		_ = s.alerts.PublishAlert(ctx, alert)
	}
	return nil
}

func (s *BanEnforcementService) IsBanned(ctx context.Context, clientID, workflowType string) (bool, error) {
	if s.blocker != nil && s.blocker.ShouldBlockDispatch(ctx, clientID, workflowType) {
		return true, nil
	}
	bans, err := s.bans.GetActiveBans(ctx, clientID)
	if err != nil {
		return false, err
	}
	return !s.manager.CanDispatchToClient(clientID, workflowType, bans), nil
}

func (s *BanEnforcementService) WarmCache(ctx context.Context) error {
	if s.blocker == nil {
		return nil
	}
	bans, err := s.bans.ListAllBans(ctx)
	if err != nil {
		return err
	}
	for _, b := range bans {
		if b.IsActive() {
			s.blocker.Add(b)
		}
	}
	return nil
}
