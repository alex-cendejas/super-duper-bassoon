package alert

import (
	"context"
	"testing"

	"github.com/super-duper-bassoon/internal/core/domain"
)

func TestStdoutAlertPublisher(t *testing.T) {
	p := NewStdoutAlertPublisher(nil)
	ctx := context.Background()
	if err := p.PublishAlert(ctx, nil); err != nil {
		t.Error("nil alert should be no-op")
	}
	a := domain.NewAlert(domain.AlertClientBanned, domain.SeverityCritical, "msg")
	if err := p.PublishAlert(ctx, a); err != nil {
		t.Fatal(err)
	}
	if len(p.Snapshot()) != 1 {
		t.Errorf("expected 1")
	}
}
