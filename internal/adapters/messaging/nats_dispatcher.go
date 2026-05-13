package messaging

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

type NATSMessageDispatcher struct {
	conn       *nats.Conn
	logger     *log.Logger
	subjectFn  func(clientID string) string
	resultSubj string
	subMu      sync.Mutex
	sub        *nats.Subscription
}

func NewNATSMessageDispatcher(conn *nats.Conn, logger *log.Logger) *NATSMessageDispatcher {
	if logger == nil {
		logger = log.Default()
	}
	return &NATSMessageDispatcher{
		conn:       conn,
		logger:     logger,
		subjectFn:  defaultDispatchSubject,
		resultSubj: "result.>",
	}
}

func defaultDispatchSubject(clientID string) string {
	return fmt.Sprintf("dispatch.%s", clientID)
}

func (n *NATSMessageDispatcher) SendDispatch(ctx context.Context, d *domain.Dispatch) error {
	payload, err := d.GetPayload()
	if err != nil {
		return fmt.Errorf("encode dispatch: %w", err)
	}
	subj := n.subjectFn(d.ClientID)
	if err := n.conn.Publish(subj, payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	if _, ok := ctx.Deadline(); !ok {
		return n.conn.FlushTimeout(2 * time.Second)
	}
	return n.conn.FlushWithContext(ctx)
}

func (n *NATSMessageDispatcher) SendBatchDispatches(ctx context.Context, list []*domain.Dispatch) error {
	for _, d := range list {
		if err := n.SendDispatch(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// SubscribeToResults returns a channel of incoming result bytes from NATS.
func (n *NATSMessageDispatcher) SubscribeToResults(subject string, bufSize int) (<-chan []byte, error) {
	if subject == "" {
		subject = n.resultSubj
	}
	if bufSize <= 0 {
		bufSize = 256
	}
	ch := make(chan []byte, bufSize)
	sub, err := n.conn.Subscribe(subject, func(m *nats.Msg) {
		select {
		case ch <- m.Data:
		default:
			n.logger.Printf("result channel full; dropping message")
		}
	})
	if err != nil {
		return nil, err
	}
	n.subMu.Lock()
	n.sub = sub
	n.subMu.Unlock()
	return ch, nil
}

func (n *NATSMessageDispatcher) Close() error {
	n.subMu.Lock()
	defer n.subMu.Unlock()
	if n.sub != nil {
		_ = n.sub.Unsubscribe()
	}
	return nil
}

// ChannelDispatcher is an in-memory dispatcher used for tests; it publishes dispatches
// onto a buffered channel.
type ChannelDispatcher struct {
	mu      sync.Mutex
	Sent    []*domain.Dispatch
	OnSend  func(d *domain.Dispatch) error
}

func NewChannelDispatcher() *ChannelDispatcher { return &ChannelDispatcher{} }

func (c *ChannelDispatcher) SendDispatch(ctx context.Context, d *domain.Dispatch) error {
	if c.OnSend != nil {
		if err := c.OnSend(d); err != nil {
			return err
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Sent = append(c.Sent, d)
	return nil
}

func (c *ChannelDispatcher) SendBatchDispatches(ctx context.Context, list []*domain.Dispatch) error {
	for _, d := range list {
		if err := c.SendDispatch(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

func (c *ChannelDispatcher) Snapshot() []*domain.Dispatch {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]*domain.Dispatch, len(c.Sent))
	copy(cp, c.Sent)
	return cp
}
