package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/super-client/internal/core/domain"
	"github.com/super-client/internal/core/ports"
)

// pendingResult holds a queued result awaiting publication.
type pendingResult struct {
	msg domain.ResultMessage
}

// ResultCollector accumulates activity results and publishes them to the broker.
type ResultCollector struct {
	broker  ports.MessageBroker
	mu      sync.Mutex
	pending []pendingResult
}

// NewResultCollector creates a ResultCollector backed by the given broker.
func NewResultCollector(broker ports.MessageBroker) *ResultCollector {
	return &ResultCollector{broker: broker}
}

// Collect queues a result for later publication.
func (rc *ResultCollector) Collect(
	runID, wfID, clientID string,
	result domain.ActivityResult,
	state *domain.ClientState,
) error {
	if runID == "" {
		return fmt.Errorf("empty run_id")
	}
	if wfID == "" {
		return fmt.Errorf("empty wf_id")
	}
	if clientID == "" {
		return fmt.Errorf("empty client_id")
	}
	msg := domain.ResultMessage{
		RunID:      runID,
		WfID:       wfID,
		ClientID:   clientID,
		Status:     result.Status,
		InnerState: state,
		ErrorMsg:   result.ErrorMsg,
		Payload:    result.Payload,
	}
	rc.mu.Lock()
	rc.pending = append(rc.pending, pendingResult{msg: msg})
	rc.mu.Unlock()
	return nil
}

// FlushResults publishes all queued results and clears the queue.
// Returns the first error encountered; remaining results stay queued.
func (rc *ResultCollector) FlushResults(ctx context.Context) error {
	rc.mu.Lock()
	toFlush := rc.pending
	rc.pending = nil
	rc.mu.Unlock()

	var failed []pendingResult
	var firstErr error
	for _, p := range toFlush {
		if err := rc.broker.PublishResult(ctx, p.msg); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed = append(failed, p)
		}
	}
	if len(failed) > 0 {
		rc.mu.Lock()
		rc.pending = append(failed, rc.pending...)
		rc.mu.Unlock()
	}
	return firstErr
}

// GetPendingCount returns the number of queued results.
func (rc *ResultCollector) GetPendingCount() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return len(rc.pending)
}
