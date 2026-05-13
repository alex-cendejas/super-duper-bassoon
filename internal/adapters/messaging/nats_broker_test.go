package messaging_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/super-client/internal/adapters/messaging"
	"github.com/super-client/internal/core/domain"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// startEmbeddedNATS launches an in-process NATS server on a random port and
// returns its URL. The server is stopped when the test ends.
func startEmbeddedNATS(t *testing.T) string {
	t.Helper()
	opts := &natsserver.Options{
		Host:           "127.0.0.1",
		Port:           -1, // random port
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 4096,
	}
	srv, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatalf("failed to create embedded NATS server: %v", err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("embedded NATS server did not become ready")
	}
	t.Cleanup(func() { srv.Shutdown() })
	return srv.ClientURL()
}

func TestNATSBroker_PublishResult(t *testing.T) {
	url := startEmbeddedNATS(t)
	broker, err := messaging.NewNATSBroker(url, testLogger())
	if err != nil {
		t.Fatalf("NewNATSBroker failed: %v", err)
	}
	defer broker.Close(context.Background())

	// Subscribe directly via NATS to verify the message arrives.
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("direct nats connect failed: %v", err)
	}
	defer nc.Close()

	received := make(chan []byte, 1)
	sub, err := nc.Subscribe(messaging.ResultSubject, func(msg *nats.Msg) {
		received <- msg.Data
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()
	// Flush ensures the subscription is registered on the server before we publish.
	if err := nc.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	result := domain.ResultMessage{
		RunID:    "run-42",
		WfID:     "wf-99",
		ClientID: "client-1",
		Status:   domain.ResultSuccess,
	}
	if err := broker.PublishResult(context.Background(), result); err != nil {
		t.Fatalf("PublishResult failed: %v", err)
	}

	select {
	case data := <-received:
		var got domain.ResultMessage
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if got.RunID != "run-42" {
			t.Errorf("expected run_id=run-42, got %s", got.RunID)
		}
		if got.Status != domain.ResultSuccess {
			t.Errorf("expected status=success, got %s", got.Status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published result")
	}
}

func TestNATSBroker_SubscribeDispatch(t *testing.T) {
	url := startEmbeddedNATS(t)
	broker, err := messaging.NewNATSBroker(url, testLogger())
	if err != nil {
		t.Fatalf("NewNATSBroker failed: %v", err)
	}
	defer broker.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	clientIDs := []string{"client-001", "client-002"}
	dispatches, err := broker.SubscribeDispatch(ctx, clientIDs)
	if err != nil {
		t.Fatalf("SubscribeDispatch failed: %v", err)
	}

	// Publish a dispatch from a separate NATS connection (simulates server).
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("direct nats connect failed: %v", err)
	}
	defer nc.Close()

	dispatchMsg := domain.DispatchMessage{
		RunID:  "run-001",
		WfID:   "wf-001",
		Activity: domain.Activity{Type: domain.ActivityReboot},
	}
	data, _ := json.Marshal(dispatchMsg)
	// Publish to client-001 subject
	nc.Publish("super-client.client-001.dispatch", data)

	select {
	case msg, ok := <-dispatches:
		if !ok {
			t.Fatal("dispatch channel closed unexpectedly")
		}
		if msg.RunID != "run-001" {
			t.Errorf("expected run_id=run-001, got %s", msg.RunID)
		}
		if msg.ClientID != "client-001" {
			t.Errorf("expected client_id=client-001, got %s", msg.ClientID)
		}
		if msg.Activity.Type != domain.ActivityReboot {
			t.Errorf("expected activity=reboot, got %s", msg.Activity.Type)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for dispatch message")
	}
}

func TestNATSBroker_SubscribeDispatch_MalformedMessage(t *testing.T) {
	url := startEmbeddedNATS(t)
	broker, err := messaging.NewNATSBroker(url, testLogger())
	if err != nil {
		t.Fatalf("NewNATSBroker failed: %v", err)
	}
	defer broker.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	clientIDs := []string{"client-001"}
	dispatches, err := broker.SubscribeDispatch(ctx, clientIDs)
	if err != nil {
		t.Fatalf("SubscribeDispatch failed: %v", err)
	}

	nc, _ := nats.Connect(url)
	defer nc.Close()

	// Send malformed JSON
	nc.Publish("super-client.client-001.dispatch", []byte("{invalid json"))

	// Malformed message should be dropped silently; channel should remain open.
	select {
	case msg, ok := <-dispatches:
		if ok {
			t.Errorf("expected no message for malformed JSON, got %+v", msg)
		}
		// channel closed by ctx.Done - ok
	case <-ctx.Done():
		// expected: no message received, context timed out
	}
}

func TestNATSBroker_MultipleClients(t *testing.T) {
	url := startEmbeddedNATS(t)
	broker, err := messaging.NewNATSBroker(url, testLogger())
	if err != nil {
		t.Fatalf("NewNATSBroker failed: %v", err)
	}
	defer broker.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	clientIDs := []string{"c1", "c2", "c3"}
	dispatches, err := broker.SubscribeDispatch(ctx, clientIDs)
	if err != nil {
		t.Fatalf("SubscribeDispatch failed: %v", err)
	}

	nc, _ := nats.Connect(url)
	defer nc.Close()
	// Flush to ensure broker subscriptions are registered before we publish.
	nc.Flush()

	// Send one dispatch to each client
	for _, id := range clientIDs {
		msg := domain.DispatchMessage{RunID: "r-" + id, WfID: "wf-1", Activity: domain.Activity{Type: domain.ActivityReboot}}
		data, _ := json.Marshal(msg)
		nc.Publish("super-client."+id+".dispatch", data)
	}

	received := make(map[string]bool)
	deadline := time.After(2 * time.Second)
	for len(received) < 3 {
		select {
		case msg := <-dispatches:
			received[msg.ClientID] = true
		case <-deadline:
			t.Fatalf("timed out; received dispatches for: %v", received)
		}
	}
	for _, id := range clientIDs {
		if !received[id] {
			t.Errorf("no dispatch received for client %s", id)
		}
	}
}

func TestNATSBroker_Close(t *testing.T) {
	url := startEmbeddedNATS(t)
	broker, err := messaging.NewNATSBroker(url, testLogger())
	if err != nil {
		t.Fatalf("NewNATSBroker failed: %v", err)
	}

	if err := broker.Close(context.Background()); err != nil {
		t.Errorf("Close returned unexpected error: %v", err)
	}
}

func TestNATSBroker_ConnectionFailed(t *testing.T) {
	_, err := messaging.NewNATSBroker("nats://127.0.0.1:19999", testLogger())
	if err == nil {
		t.Error("expected error connecting to nonexistent NATS server")
	}
}
