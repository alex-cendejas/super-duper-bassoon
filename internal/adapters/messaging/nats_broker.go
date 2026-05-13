package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/super-duper-bassoon/internal/core/domain"
)

const (
	// DispatchSubjectFmt is the NATS subject for dispatch messages per client.
	DispatchSubjectFmt = "dispatch.%s"
	// RegisterSubject is the NATS subject for client registration messages.
	RegisterSubject = "client.register"
)

// NATSBroker is the NATS-backed MessageBroker implementation.
type NATSBroker struct {
	conn   *nats.Conn
	logger *slog.Logger
	mu     sync.Mutex
	subs   []*nats.Subscription
}

// NewNATSBroker creates a NATSBroker connected to the given NATS URL.
func NewNATSBroker(url string, logger *slog.Logger) (*NATSBroker, error) {
	nc, err := nats.Connect(url,
		nats.Name("super-client"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(nats.DefaultReconnectWait),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &NATSBroker{conn: nc, logger: logger}, nil
}

// SubscribeDispatch subscribes to dispatch subjects for the given client IDs
// and returns a merged channel of incoming DispatchMessages.
func (b *NATSBroker) SubscribeDispatch(ctx context.Context, clientIDs []string) (<-chan domain.DispatchMessage, error) {
	out := make(chan domain.DispatchMessage, 256)
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, id := range clientIDs {
		subject := fmt.Sprintf(DispatchSubjectFmt, id)
		clientID := id // capture for closure
		sub, err := b.conn.Subscribe(subject, func(msg *nats.Msg) {
			var dispatch domain.DispatchMessage
			if err := json.Unmarshal(msg.Data, &dispatch); err != nil {
				b.logger.Warn("malformed dispatch message", "client_id", clientID, "err", err)
				return
			}
			dispatch.ClientID = clientID
			select {
			case out <- dispatch:
			case <-ctx.Done():
			}
		})
		if err != nil {
			return nil, fmt.Errorf("subscribe dispatch for %s: %w", id, err)
		}
		b.subs = append(b.subs, sub)
	}

	// Flush ensures all subscriptions are registered on the server before
	// the caller starts publishing dispatches.
	if err := b.conn.Flush(); err != nil {
		return nil, fmt.Errorf("flush subscriptions: %w", err)
	}

	// Close the channel when context is cancelled.
	go func() {
		<-ctx.Done()
		close(out)
	}()

	return out, nil
}

// PublishResult marshals and publishes a result message to a per-client subject.
func (b *NATSBroker) PublishResult(_ context.Context, result domain.ResultMessage) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	subject := fmt.Sprintf("result.%s", result.ClientID)
	if err := b.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish result: %w", err)
	}
	return nil
}

// RegisterClient publishes a registration message for a new client.
func (b *NATSBroker) RegisterClient(ctx context.Context, client domain.ClientMetadata) error {
	data, err := json.Marshal(client)
	if err != nil {
		return fmt.Errorf("marshal registration: %w", err)
	}
	return b.conn.Publish(RegisterSubject, data)
}

// Close drains subscriptions and the connection.
func (b *NATSBroker) Close(_ context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, sub := range b.subs {
		_ = sub.Unsubscribe()
	}
	b.conn.Close()
	return nil
}
