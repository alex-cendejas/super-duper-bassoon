package services

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

type orderedHandler struct {
	name string
	pri  int
	mu   *sync.Mutex
	seq  *[]string
	fail bool
}

func (h *orderedHandler) Name() string  { return h.name }
func (h *orderedHandler) Priority() int { return h.pri }
func (h *orderedHandler) HandleResult(ctx context.Context, r *domain.Result) error {
	h.mu.Lock()
	*h.seq = append(*h.seq, h.name)
	h.mu.Unlock()
	if h.fail {
		return errors.New("boom")
	}
	return nil
}

func TestResultDispatcher_Order(t *testing.T) {
	var mu sync.Mutex
	var seq []string
	d := NewResultMessageDispatcher(nil)
	d.RegisterHandler(&orderedHandler{name: "low", pri: 2, mu: &mu, seq: &seq})
	d.RegisterHandler(&orderedHandler{name: "high", pri: 1, mu: &mu, seq: &seq})
	_ = d.Dispatch(context.Background(), &domain.Result{RunID: "r", WorkflowID: "w", ClientID: "c", Status: domain.StatusSuccess})
	if len(seq) != 2 || seq[0] != "high" || seq[1] != "low" {
		t.Errorf("order: %v", seq)
	}
}

func TestResultDispatcher_HandlersIsolatedFromErrors(t *testing.T) {
	var mu sync.Mutex
	var seq []string
	d := NewResultMessageDispatcher(nil)
	d.RegisterHandler(&orderedHandler{name: "a", pri: 1, mu: &mu, seq: &seq, fail: true})
	d.RegisterHandler(&orderedHandler{name: "b", pri: 2, mu: &mu, seq: &seq})
	_ = d.Dispatch(context.Background(), &domain.Result{RunID: "r", WorkflowID: "w", ClientID: "c", Status: domain.StatusSuccess})
	if len(seq) != 2 {
		t.Errorf("both handlers should be called: %v", seq)
	}
}

func TestResultDispatcher_IgnoresMalformed(t *testing.T) {
	var mu sync.Mutex
	var seq []string
	d := NewResultMessageDispatcher(nil)
	d.RegisterHandler(&orderedHandler{name: "a", pri: 1, mu: &mu, seq: &seq})
	_ = d.Dispatch(context.Background(), &domain.Result{})
	_ = d.Dispatch(context.Background(), nil)
	if len(seq) != 0 {
		t.Errorf("expected no calls")
	}
}

func TestResultDispatcher_Stop(t *testing.T) {
	d := NewResultMessageDispatcher(nil)
	d.Stop()
}

func TestResultDispatcher_Start(t *testing.T) {
	d := NewResultMessageDispatcher(nil)
	var mu sync.Mutex
	var seq []string
	d.RegisterHandler(&orderedHandler{name: "a", pri: 1, mu: &mu, seq: &seq})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan []byte, 2)
	good := []byte(`{"run_id":"r","wf_id":"w","client_id":"c","status":"success"}`)
	bad := []byte(`not json`)
	ch <- good
	ch <- bad
	done := make(chan struct{})
	go func() {
		d.Start(ctx, ch)
		close(done)
	}()
	// Allow workers to process
	// Sleep is awkward, but ResultDispatcher reads from ch synchronously
	// Let's cancel after we expect both messages to be drained.
	for i := 0; i < 10; i++ {
		mu.Lock()
		got := len(seq)
		mu.Unlock()
		if got == 1 {
			break
		}
	}
	cancel()
	<-done
}
