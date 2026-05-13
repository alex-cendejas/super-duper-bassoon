package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
	"github.com/super-duper-bassoon/internal/adapters/trigger"
)

// TriggerCoordinationService periodically evaluates triggers and fires workflows.
type TriggerCoordinationService struct {
	workflows ports.WorkflowRepository
	orch      *WorkflowOrchestrationService
	cron      *trigger.CronEvaluator
	events    ports.EventBus
	tick      time.Duration
	logger    *log.Logger
	wg        sync.WaitGroup
	cancelFn  context.CancelFunc
}

func NewTriggerCoordinationService(workflows ports.WorkflowRepository, orch *WorkflowOrchestrationService, cron *trigger.CronEvaluator, events ports.EventBus, tickMS int64, logger *log.Logger) *TriggerCoordinationService {
	if logger == nil {
		logger = log.Default()
	}
	if cron == nil {
		cron = trigger.NewCronEvaluator()
	}
	tick := time.Duration(tickMS) * time.Millisecond
	if tick <= 0 {
		tick = 5 * time.Second
	}
	return &TriggerCoordinationService{
		workflows: workflows,
		orch:      orch,
		cron:      cron,
		events:    events,
		tick:      tick,
		logger:    logger,
	}
}

// Start subscribes to workflow completion events for event-driven chains
// and starts the periodic evaluator goroutine.
func (s *TriggerCoordinationService) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	s.cancelFn = cancel
	if s.events != nil {
		_ = s.events.Subscribe("workflow.completed", s.onWorkflowCompleted)
	}
	s.wg.Add(1)
	go s.loop(ctx)
}

func (s *TriggerCoordinationService) Stop() {
	if s.cancelFn != nil {
		s.cancelFn()
	}
	s.wg.Wait()
}

func (s *TriggerCoordinationService) loop(ctx context.Context) {
	defer s.wg.Done()
	t := time.NewTicker(s.tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.evaluate(ctx, now)
		}
	}
}

func (s *TriggerCoordinationService) evaluate(ctx context.Context, now time.Time) {
	wfs, err := s.workflows.ListActiveWorkflows(ctx)
	if err != nil {
		s.logger.Printf("trigger: list workflows: %v", err)
		return
	}
	for _, wf := range wfs {
		if wf.Trigger.Kind != domain.TriggerScheduled || wf.Trigger.Cron == "" {
			continue
		}
		should, err := s.cron.ShouldFire(wf.ID, wf.Trigger, now)
		if err != nil {
			s.logger.Printf("trigger: cron parse error wf=%s: %v", wf.ID, err)
			continue
		}
		if !should {
			continue
		}
		s.fire(ctx, wf, "scheduled")
	}
}

func (s *TriggerCoordinationService) fire(ctx context.Context, wf *domain.Workflow, reason string) {
	_, err := s.orch.TriggerWorkflow(ctx, wf.ID, reason)
	if err != nil {
		s.logger.Printf("trigger: failed to fire workflow %s: %v", wf.ID, err)
	}
}

func (s *TriggerCoordinationService) onWorkflowCompleted(ctx context.Context, evt domain.Event) error {
	e, ok := evt.(*domain.WorkflowCompletionEvent)
	if !ok {
		return nil
	}
	wfs, err := s.workflows.ListActiveWorkflows(ctx)
	if err != nil {
		return err
	}
	for _, wf := range wfs {
		if wf.Trigger.Kind != domain.TriggerEvent {
			continue
		}
		if wf.Trigger.OnComplete == "" || wf.Trigger.OnComplete != e.WorkflowID {
			continue
		}
		s.fire(ctx, wf, "event:"+e.WorkflowID)
	}
	return nil
}
