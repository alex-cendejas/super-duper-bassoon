package services_test

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/super-duper-bassoon/internal/core/domain"
)

// --- MockStateStore ---

type MockStateStore struct {
	mu     sync.RWMutex
	states map[string]*domain.ClientState
	// Inject errors for testing
	GetErr    error
	UpdateErr error
}

func NewMockStateStore() *MockStateStore {
	return &MockStateStore{states: make(map[string]*domain.ClientState)}
}

func (m *MockStateStore) GetState(_ context.Context, clientID string) (*domain.ClientState, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	st, ok := m.states[clientID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrInnerClientNotFound, clientID)
	}
	clone := st.Clone()
	return &clone, nil
}

func (m *MockStateStore) UpdateState(_ context.Context, clientID string, state *domain.ClientState) error {
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := state.Clone()
	m.states[clientID] = &clone
	return nil
}

func (m *MockStateStore) GetAllStates(_ context.Context) (map[string]*domain.ClientState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]*domain.ClientState, len(m.states))
	for k, v := range m.states {
		clone := v.Clone()
		result[k] = &clone
	}
	return result, nil
}

func (m *MockStateStore) ListClientIDs(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.states))
	for k := range m.states {
		ids = append(ids, k)
	}
	return ids, nil
}

func (m *MockStateStore) Seed(clientID string, state domain.ClientState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := state.Clone()
	m.states[clientID] = &clone
}

// --- MockActivityExecutor ---

type MockActivityExecutor struct {
	// If set, always returns this error
	Err error
	// If set, returns this result regardless of input
	FixedResult *domain.ActivityResult
	// Tracks calls
	mu    sync.Mutex
	calls []executorCall
}

type executorCall struct {
	ClientID string
	Activity domain.Activity
}

func (m *MockActivityExecutor) Execute(
	_ context.Context,
	clientID string,
	activity domain.Activity,
	state domain.ClientState,
) (*domain.ClientState, *domain.ActivityResult, error) {
	m.mu.Lock()
	m.calls = append(m.calls, executorCall{ClientID: clientID, Activity: activity})
	m.mu.Unlock()

	if m.Err != nil {
		return nil, nil, m.Err
	}
	if m.FixedResult != nil {
		newState := state.Clone()
		return &newState, m.FixedResult, nil
	}
	// Default: delegate to domain logic
	newState, result := domain.ExecuteActivity(activity, state)
	return &newState, &result, nil
}

func (m *MockActivityExecutor) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// --- MockMessageBroker ---

type MockMessageBroker struct {
	published  []domain.ResultMessage
	Registered []domain.ClientMetadata
	mu         sync.Mutex
	PublishErr  error
	RegisterErr error
	dispatches chan domain.DispatchMessage
}

func NewMockMessageBroker() *MockMessageBroker {
	return &MockMessageBroker{
		dispatches: make(chan domain.DispatchMessage, 64),
	}
}

func (m *MockMessageBroker) SubscribeDispatch(_ context.Context, _ []string) (<-chan domain.DispatchMessage, error) {
	return m.dispatches, nil
}

func (m *MockMessageBroker) PublishResult(_ context.Context, result domain.ResultMessage) error {
	if m.PublishErr != nil {
		return m.PublishErr
	}
	m.mu.Lock()
	m.published = append(m.published, result)
	m.mu.Unlock()
	return nil
}

func (m *MockMessageBroker) RegisterClient(_ context.Context, client domain.ClientMetadata) error {
	if m.RegisterErr != nil {
		return m.RegisterErr
	}
	m.mu.Lock()
	m.Registered = append(m.Registered, client)
	m.mu.Unlock()
	return nil
}

func (m *MockMessageBroker) Close(_ context.Context) error { return nil }

func (m *MockMessageBroker) Published() []domain.ResultMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.ResultMessage, len(m.published))
	copy(result, m.published)
	return result
}

func (m *MockMessageBroker) SendDispatch(d domain.DispatchMessage) {
	m.dispatches <- d
}

// --- deterministicChaos ---

type deterministicChaos struct {
	float64Val float64
	intnVal    int
}

func (d deterministicChaos) Float64() float64 { return d.float64Val }
func (d deterministicChaos) Intn(_ int) int   { return d.intnVal }

// --- MockClock ---

type MockClock struct {
	current interface{ String() string }
}

// errForcedFailure is a sentinel for testing
var errForcedFailure = errors.New("forced failure")
