package ports

import (
	"context"

	"github.com/super-client/internal/core/domain"
)

// MessageBroker abstracts the messaging transport between super-client and server.
type MessageBroker interface {
	// SubscribeDispatch subscribes to dispatch messages for the given client IDs.
	// Returns a channel that receives incoming dispatch messages.
	SubscribeDispatch(ctx context.Context, clientIDs []string) (<-chan domain.DispatchMessage, error)

	// PublishResult sends a result message back to the server.
	PublishResult(ctx context.Context, result domain.ResultMessage) error

	// Close tears down the broker connection.
	Close(ctx context.Context) error
}
