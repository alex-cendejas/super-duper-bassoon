package services

import (
	"context"
	"log"
	"sort"
	"sync"

	"github.com/super-duper-bassoon/server/internal/core/domain"
	"github.com/super-duper-bassoon/server/internal/core/ports"
)

type ResultMessageDispatcher struct {
	handlers []ports.ResultHandler
	logger   *log.Logger
	mu       sync.Mutex
	stopped  bool
}

func NewResultMessageDispatcher(logger *log.Logger) *ResultMessageDispatcher {
	if logger == nil {
		logger = log.Default()
	}
	return &ResultMessageDispatcher{logger: logger}
}

func (d *ResultMessageDispatcher) RegisterHandler(h ports.ResultHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers = append(d.handlers, h)
	sort.SliceStable(d.handlers, func(i, j int) bool {
		return d.handlers[i].Priority() < d.handlers[j].Priority()
	})
}

func (d *ResultMessageDispatcher) Dispatch(ctx context.Context, r *domain.Result) error {
	if r == nil {
		return nil
	}
	if !r.IsValid() {
		d.logger.Printf("result dispatcher: ignoring malformed result run=%s client=%s", r.RunID, r.ClientID)
		return nil
	}
	d.mu.Lock()
	handlers := append([]ports.ResultHandler(nil), d.handlers...)
	d.mu.Unlock()
	for _, h := range handlers {
		if err := h.HandleResult(ctx, r); err != nil {
			d.logger.Printf("result handler %s error: %v", h.Name(), err)
		}
	}
	return nil
}

// Start reads result payloads from `ch` and dispatches them. Returns when ctx is done.
func (d *ResultMessageDispatcher) Start(ctx context.Context, ch <-chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			r, err := domain.ParseResult(data)
			if err != nil {
				d.logger.Printf("invalid result payload: %v", err)
				continue
			}
			_ = d.Dispatch(ctx, r)
		}
	}
}

func (d *ResultMessageDispatcher) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopped = true
}
