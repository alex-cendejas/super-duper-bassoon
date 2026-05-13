package services

import (
	"context"
	"testing"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

func TestDynamicGroupingService_ResolveClients(t *testing.T) {
	clients := newMemClientRepo()
	ctx := context.Background()
	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "a", OS: "linux", Active: true, InnerState: map[string]interface{}{"config_version": 1}})
	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "b", OS: "darwin", Active: true})
	_ = clients.SaveClient(ctx, &domain.ClientMetadata{ClientID: "c", OS: "linux", Active: false})

	svc := NewDynamicGroupingService(clients)

	got, err := svc.ResolveClients(ctx, "os == 'linux'")
	if err != nil {
		t.Fatal(err)
	}
	if got.MatchCount != 1 || got.MatchingClientIDs[0] != "a" {
		t.Errorf("expected only 'a' (linux+active), got %v", got)
	}

	// Empty filter returns all active clients (per FilterNode nil == matches all)
	got, err = svc.ResolveClients(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.MatchCount != 2 {
		t.Errorf("empty filter: %d", got.MatchCount)
	}

	// Bad filter
	if _, err := svc.ResolveClients(ctx, "==="); err == nil {
		t.Error("expected parse error")
	}
}
