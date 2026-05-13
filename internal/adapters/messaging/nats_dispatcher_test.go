package messaging

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

func startTestNATS(t *testing.T) (*natsserver.Server, string) {
	t.Helper()
	opts := &natsserver.Options{
		Host:           "127.0.0.1",
		Port:           -1, // random port
		NoLog:          true,
		NoSigs:         true,
		Debug:          false,
		Trace:          false,
		JetStream:      false,
		MaxControlLine: 4096,
	}
	s, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatal(err)
	}
	go s.Start()
	if !s.ReadyForConnections(2 * time.Second) {
		t.Fatal("nats server not ready")
	}
	t.Cleanup(func() {
		s.Shutdown()
		s.WaitForShutdown()
	})
	return s, s.ClientURL()
}

func TestNATSDispatcher_RoundTrip(t *testing.T) {
	_, url := startTestNATS(t)
	conn, err := nats.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	d := NewNATSMessageDispatcher(conn, nil)
	ch, err := d.SubscribeToResults("result.>", 0)
	if err != nil {
		t.Fatal(err)
	}

	// Send dispatch
	disp := &domain.Dispatch{RunID: "r1", WorkflowID: "w1", ClientID: "c1", Activity: domain.ActivityReboot}
	if err := d.SendDispatch(context.Background(), disp); err != nil {
		t.Fatal(err)
	}

	// Publish a result back
	if err := conn.Publish("result.server", []byte(`{"run_id":"r","wf_id":"w","client_id":"c","status":"success"}`)); err != nil {
		t.Fatal(err)
	}
	_ = conn.Flush()

	select {
	case data := <-ch:
		r, err := domain.ParseResult(data)
		if err != nil {
			t.Fatal(err)
		}
		if r.ClientID != "c" {
			t.Errorf("got: %+v", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for result")
	}

	// Batch send
	if err := d.SendBatchDispatches(context.Background(), []*domain.Dispatch{disp, disp}); err != nil {
		t.Fatal(err)
	}

	// Close should be safe
	if err := d.Close(); err != nil {
		t.Error(err)
	}
}

func TestNATSDispatcher_BadEncode(t *testing.T) {
	_, url := startTestNATS(t)
	conn, _ := nats.Connect(url)
	defer conn.Close()
	d := NewNATSMessageDispatcher(conn, nil)
	// Invalid dispatch (activity not valid still serializes), but the encoder works.
	disp := &domain.Dispatch{RunID: "r", WorkflowID: "w", Activity: domain.ActivityReboot}
	if err := d.SendDispatch(context.Background(), disp); err != nil {
		t.Fatal(err)
	}
	_ = atomic.LoadInt32(new(int32))
}
