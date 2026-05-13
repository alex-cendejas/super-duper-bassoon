package services

import (
	"context"

	"github.com/super-duper-bassoon/server/internal/core/domain"
	"github.com/super-duper-bassoon/server/internal/core/ports"
)

type DynamicGroupingService struct {
	clients ports.ClientRepository
}

func NewDynamicGroupingService(clients ports.ClientRepository) *DynamicGroupingService {
	return &DynamicGroupingService{clients: clients}
}

func (s *DynamicGroupingService) ResolveClients(ctx context.Context, filterExpr string) (*domain.FilterResult, error) {
	node, err := domain.ParseFilter(filterExpr)
	if err != nil {
		return nil, err
	}
	clients, err := s.clients.ListClients(ctx)
	if err != nil {
		return nil, err
	}
	result := &domain.FilterResult{TotalEvaluated: len(clients)}
	for _, c := range clients {
		if !c.Active {
			continue
		}
		match, err := node.Evaluate(c)
		if err != nil {
			return nil, err
		}
		if match {
			result.MatchingClientIDs = append(result.MatchingClientIDs, c.ClientID)
			result.MatchCount++
		}
	}
	return result, nil
}
