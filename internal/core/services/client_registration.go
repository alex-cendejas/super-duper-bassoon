package services

import (
	"context"
	"log"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
)

// ClientRegistrationService registers or updates clients from incoming result messages.
type ClientRegistrationService struct {
	clients ports.ClientRepository
	logger  *log.Logger
}

func NewClientRegistrationService(clients ports.ClientRepository, logger *log.Logger) *ClientRegistrationService {
	if logger == nil {
		logger = log.Default()
	}
	return &ClientRegistrationService{clients: clients, logger: logger}
}

func (s *ClientRegistrationService) Name() string  { return "client_registration" }
func (s *ClientRegistrationService) Priority() int { return 0 } // highest priority — runs before loop detection

func (s *ClientRegistrationService) HandleResult(ctx context.Context, r *domain.Result) error {
	if r == nil || r.ClientID == "" {
		return nil
	}
	innerState := map[string]interface{}{}
	if r.InnerState != nil {
		innerState = r.InnerState
	}
	client := &domain.ClientMetadata{
		ClientID:   r.ClientID,
		Active:     true,
		Labels:     map[string]string{},
		InnerState: innerState,
		LastSeenAt: r.ReceivedAt,
	}
	if client.LastSeenAt.IsZero() {
		client.LastSeenAt = time.Now().UTC()
	}
	if err := s.clients.SaveClient(ctx, client); err != nil {
		s.logger.Printf("client_registration: save client %s: %v", r.ClientID, err)
	}
	return nil
}
