package trigger

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/super-duper-bassoon/internal/core/domain"
)

// CronEvaluator decides whether scheduled triggers should fire.
// It keeps an internal record of the "next fire time" per workflow.
type CronEvaluator struct {
	parser  cron.Parser
	mu      sync.Mutex
	next    map[string]time.Time
}

func NewCronEvaluator() *CronEvaluator {
	return &CronEvaluator{
		parser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
		next:   map[string]time.Time{},
	}
}

func (e *CronEvaluator) ShouldFire(workflowID string, spec domain.TriggerSpec, now time.Time) (bool, error) {
	if spec.Kind != domain.TriggerScheduled || spec.Cron == "" {
		return false, nil
	}
	sched, err := e.parser.Parse(spec.Cron)
	if err != nil {
		return false, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	next, ok := e.next[workflowID]
	if !ok {
		next = sched.Next(now)
		e.next[workflowID] = next
		return false, nil
	}
	if now.Before(next) {
		return false, nil
	}
	e.next[workflowID] = sched.Next(now)
	return true, nil
}

// Reset clears state for a workflow, e.g., when it is deactivated.
func (e *CronEvaluator) Reset(workflowID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.next, workflowID)
}

// Tick is a helper for tests/loops, ignored.
var _ = context.TODO
