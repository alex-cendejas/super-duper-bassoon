package storage_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/super-duper-bassoon/internal/adapters/storage"
	"github.com/super-duper-bassoon/internal/core/domain"
)

func sampleState(version int) *domain.ClientState {
	return &domain.ClientState{
		Packages:      map[string]string{"vim": "9.0"},
		ConfigVersion: version,
		PowerState:    domain.PowerStateOn,
	}
}

func TestMemoryStore_GetState_NotFound(t *testing.T) {
	s := storage.NewMemoryStore()
	_, err := s.GetState(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent client")
	}
	if !errors.Is(err, domain.ErrInnerClientNotFound) {
		t.Errorf("expected ErrInnerClientNotFound, got %v", err)
	}
}

func TestMemoryStore_UpdateAndGet(t *testing.T) {
	s := storage.NewMemoryStore()
	state := sampleState(3)

	if err := s.UpdateState(context.Background(), "client-1", state); err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	got, err := s.GetState(context.Background(), "client-1")
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if got.ConfigVersion != 3 {
		t.Errorf("expected ConfigVersion=3, got %d", got.ConfigVersion)
	}
}

func TestMemoryStore_GetState_ReturnsCopy(t *testing.T) {
	s := storage.NewMemoryStore()
	state := sampleState(1)
	s.UpdateState(context.Background(), "client-1", state)

	got, _ := s.GetState(context.Background(), "client-1")
	got.ConfigVersion = 999 // mutate the returned copy

	// Original in store should be unchanged
	got2, _ := s.GetState(context.Background(), "client-1")
	if got2.ConfigVersion == 999 {
		t.Error("GetState did not return a deep copy; mutation leaked into store")
	}
}

func TestMemoryStore_UpdateState_NilState(t *testing.T) {
	s := storage.NewMemoryStore()
	if err := s.UpdateState(context.Background(), "client-1", nil); err == nil {
		t.Error("expected error for nil state")
	}
}

func TestMemoryStore_GetAllStates(t *testing.T) {
	s := storage.NewMemoryStore()
	s.UpdateState(context.Background(), "c1", sampleState(1))
	s.UpdateState(context.Background(), "c2", sampleState(2))
	s.UpdateState(context.Background(), "c3", sampleState(3))

	all, err := s.GetAllStates(context.Background())
	if err != nil {
		t.Fatalf("GetAllStates failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 states, got %d", len(all))
	}
	if all["c2"].ConfigVersion != 2 {
		t.Errorf("expected c2 ConfigVersion=2, got %d", all["c2"].ConfigVersion)
	}
}

func TestMemoryStore_GetAllStates_ReturnsCopies(t *testing.T) {
	s := storage.NewMemoryStore()
	s.UpdateState(context.Background(), "c1", sampleState(10))

	all, _ := s.GetAllStates(context.Background())
	all["c1"].ConfigVersion = 999

	// Store should be unaffected
	got, _ := s.GetState(context.Background(), "c1")
	if got.ConfigVersion == 999 {
		t.Error("GetAllStates did not return deep copies")
	}
}

func TestMemoryStore_ListClientIDs(t *testing.T) {
	s := storage.NewMemoryStore()
	s.UpdateState(context.Background(), "a", sampleState(1))
	s.UpdateState(context.Background(), "b", sampleState(1))

	ids, err := s.ListClientIDs(context.Background())
	if err != nil {
		t.Fatalf("ListClientIDs failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs, got %d", len(ids))
	}
}

func TestMemoryStore_ListClientIDs_Empty(t *testing.T) {
	s := storage.NewMemoryStore()
	ids, err := s.ListClientIDs(context.Background())
	if err != nil {
		t.Fatalf("ListClientIDs failed: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs, got %d", len(ids))
	}
}

func TestMemoryStore_Concurrent(t *testing.T) {
	s := storage.NewMemoryStore()
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "client"
			state := sampleState(n)
			s.UpdateState(context.Background(), id, state)
			s.GetState(context.Background(), id)
		}(i)
	}
	wg.Wait()
}

func TestMemoryStore_Overwrite(t *testing.T) {
	s := storage.NewMemoryStore()
	s.UpdateState(context.Background(), "c1", sampleState(1))
	s.UpdateState(context.Background(), "c1", sampleState(99))

	got, _ := s.GetState(context.Background(), "c1")
	if got.ConfigVersion != 99 {
		t.Errorf("expected ConfigVersion=99 after overwrite, got %d", got.ConfigVersion)
	}
}
