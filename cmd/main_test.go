package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/super-duper-bassoon/server/internal/adapters/messaging"
)

func TestRunPeriodic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var count int32
	go runPeriodic(ctx, 10*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
	if atomic.LoadInt32(&count) < 2 {
		t.Errorf("expected periodic fires, got %d", count)
	}
	// Zero interval: should return immediately, no-op
	runPeriodic(context.Background(), 0, func() { t.Error("should not be called") })
}

func TestPickDispatcher(t *testing.T) {
	if pickDispatcher(nil) == nil {
		t.Error("expected fallback dispatcher")
	}
	natsDisp := messaging.NewNATSMessageDispatcher(nil, nil)
	if pickDispatcher(natsDisp) == nil {
		t.Error("expected nats dispatcher")
	}
}
