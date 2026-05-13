package services

import (
	"context"
	"log"
	"time"

	"github.com/super-duper-bassoon/server/internal/core/domain"
	"github.com/super-duper-bassoon/server/internal/core/ports"
)

type DispatchCoordinationService struct {
	runs    ports.RunRepository
	disp    ports.MessageDispatcher
	clients ports.ClientRepository
	filter  *DispatchFilterService
	logger  *log.Logger
}

func NewDispatchCoordinationService(runs ports.RunRepository, disp ports.MessageDispatcher, clients ports.ClientRepository, filter *DispatchFilterService, logger *log.Logger) *DispatchCoordinationService {
	if logger == nil {
		logger = log.Default()
	}
	return &DispatchCoordinationService{
		runs:    runs,
		disp:    disp,
		clients: clients,
		filter:  filter,
		logger:  logger,
	}
}

func (s *DispatchCoordinationService) GenerateDispatches(run *domain.Run, wf *domain.Workflow, clientIDs []string) []*domain.Dispatch {
	out := make([]*domain.Dispatch, 0, len(clientIDs))
	now := time.Now().UTC()
	for _, cid := range clientIDs {
		out = append(out, &domain.Dispatch{
			RunID:        run.RunID,
			WorkflowID:   wf.ID,
			ClientID:     cid,
			Activity:     wf.Activity,
			Params:       wf.Params,
			DispatchedAt: now,
		})
	}
	return out
}

func (s *DispatchCoordinationService) SendDispatches(ctx context.Context, run *domain.Run, wf *domain.Workflow, clientIDs []string) (sent []*domain.Dispatch, filteredIDs []string, err error) {
	allowed, filtered, err := s.filter.FilterDispatchList(ctx, wf.WorkflowType, clientIDs)
	if err != nil {
		return nil, nil, err
	}
	// Persist participation as the filtered list
	run.ParticipatingClients = allowed
	dispatches := s.GenerateDispatches(run, wf, allowed)
	for _, d := range dispatches {
		if !d.IsValid() {
			s.logger.Printf("invalid dispatch run=%s client=%s", d.RunID, d.ClientID)
			continue
		}
		if err := s.disp.SendDispatch(ctx, d); err != nil {
			return sent, filtered, err
		}
		sent = append(sent, d)
	}
	run.DispatchedAt = time.Now().UTC()
	run.State = domain.RunInProgress
	if updErr := s.runs.UpdateRun(ctx, run); updErr != nil {
		s.logger.Printf("failed to update run %s after dispatch: %v", run.RunID, updErr)
	}
	return sent, filtered, nil
}
