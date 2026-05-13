package services

import (
	"context"
	"log"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

type DispatchFilterService struct {
	ban    *BanEnforcementService
	logger *log.Logger
}

func NewDispatchFilterService(ban *BanEnforcementService, logger *log.Logger) *DispatchFilterService {
	if logger == nil {
		logger = log.Default()
	}
	return &DispatchFilterService{ban: ban, logger: logger}
}

func (s *DispatchFilterService) FilterDispatchList(ctx context.Context, workflowType string, clientIDs []string) ([]string, []string, error) {
	allowed := make([]string, 0, len(clientIDs))
	filtered := make([]string, 0)
	for _, cid := range clientIDs {
		blocked, err := s.ban.IsBanned(ctx, cid, workflowType)
		if err != nil {
			return nil, nil, err
		}
		if blocked {
			filtered = append(filtered, cid)
			continue
		}
		allowed = append(allowed, cid)
	}
	if len(filtered) > 0 {
		s.logger.Printf("dispatch_filter: filtered %d banned client(s) from workflow_type=%s", len(filtered), workflowType)
	}
	return allowed, filtered, nil
}

// FilterDispatches removes dispatches targeting banned clients.
func (s *DispatchFilterService) FilterDispatches(ctx context.Context, workflowType string, ds []*domain.Dispatch) ([]*domain.Dispatch, []*domain.Dispatch, error) {
	allowed := make([]*domain.Dispatch, 0, len(ds))
	filtered := make([]*domain.Dispatch, 0)
	for _, d := range ds {
		blocked, err := s.ban.IsBanned(ctx, d.ClientID, workflowType)
		if err != nil {
			return nil, nil, err
		}
		if blocked {
			filtered = append(filtered, d)
			continue
		}
		allowed = append(allowed, d)
	}
	return allowed, filtered, nil
}
