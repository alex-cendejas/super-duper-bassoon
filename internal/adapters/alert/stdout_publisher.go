package alert

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/super-duper-bassoon/internal/core/domain"
)

type StdoutAlertPublisher struct {
	logger *log.Logger
	mu     sync.Mutex
	Alerts []*domain.Alert
}

func NewStdoutAlertPublisher(logger *log.Logger) *StdoutAlertPublisher {
	if logger == nil {
		logger = log.Default()
	}
	return &StdoutAlertPublisher{logger: logger}
}

func (s *StdoutAlertPublisher) PublishAlert(ctx context.Context, alert *domain.Alert) error {
	if alert == nil {
		return nil
	}
	s.mu.Lock()
	s.Alerts = append(s.Alerts, alert)
	s.mu.Unlock()
	b, _ := json.Marshal(alert)
	s.logger.Printf("ALERT [%s/%s] %s %s", alert.Severity, alert.Kind, alert.Message, string(b))
	return nil
}

func (s *StdoutAlertPublisher) Snapshot() []*domain.Alert {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]*domain.Alert, len(s.Alerts))
	copy(cp, s.Alerts)
	return cp
}
