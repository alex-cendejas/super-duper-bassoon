package services

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
	"github.com/super-duper-bassoon/internal/core/ports"
)

// memoryWorkflowRepo is an in-memory ports.WorkflowRepository.
type memoryWorkflowRepo struct {
	mu  sync.Mutex
	all map[string]*domain.Workflow
}

func newMemWorkflowRepo() *memoryWorkflowRepo { return &memoryWorkflowRepo{all: map[string]*domain.Workflow{}} }

func (m *memoryWorkflowRepo) SaveWorkflow(ctx context.Context, w *domain.Workflow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *w
	m.all[w.ID] = &cp
	return nil
}

func (m *memoryWorkflowRepo) GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if w, ok := m.all[id]; ok {
		cp := *w
		return &cp, nil
	}
	return nil, domain.ErrWorkflowNotFound
}

func (m *memoryWorkflowRepo) ListAllWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*domain.Workflow, 0, len(m.all))
	for _, w := range m.all {
		cp := *w
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *memoryWorkflowRepo) ListActiveWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	all, _ := m.ListAllWorkflows(ctx)
	var out []*domain.Workflow
	for _, w := range all {
		if w.Active {
			out = append(out, w)
		}
	}
	return out, nil
}

func (m *memoryWorkflowRepo) ListWorkflowsByType(ctx context.Context, workflowType string) ([]*domain.Workflow, error) {
	all, _ := m.ListAllWorkflows(ctx)
	var out []*domain.Workflow
	for _, w := range all {
		if w.WorkflowType == workflowType {
			out = append(out, w)
		}
	}
	return out, nil
}

func (m *memoryWorkflowRepo) UpdateWorkflowState(ctx context.Context, id string, active bool, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.all[id]
	if !ok {
		return domain.ErrWorkflowNotFound
	}
	w.Active = active
	w.DeactivatedReason = reason
	w.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *memoryWorkflowRepo) DeleteWorkflow(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.all[id]; !ok {
		return domain.ErrWorkflowNotFound
	}
	delete(m.all, id)
	return nil
}

// memoryClientRepo
type memoryClientRepo struct {
	mu  sync.Mutex
	all map[string]*domain.ClientMetadata
}

func newMemClientRepo() *memoryClientRepo {
	return &memoryClientRepo{all: map[string]*domain.ClientMetadata{}}
}

func (m *memoryClientRepo) SaveClient(ctx context.Context, c *domain.ClientMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *c
	m.all[c.ClientID] = &cp
	return nil
}

func (m *memoryClientRepo) GetClientByID(ctx context.Context, id string) (*domain.ClientMetadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.all[id]; ok {
		cp := *c
		return &cp, nil
	}
	return nil, domain.ErrClientNotFound
}

func (m *memoryClientRepo) ListClients(ctx context.Context) ([]*domain.ClientMetadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*domain.ClientMetadata, 0, len(m.all))
	for _, c := range m.all {
		cp := *c
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ClientID < out[j].ClientID })
	return out, nil
}

func (m *memoryClientRepo) GetClientsByIDs(ctx context.Context, ids []string) ([]*domain.ClientMetadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []*domain.ClientMetadata{}
	for _, id := range ids {
		if c, ok := m.all[id]; ok {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out, nil
}

// memoryRunRepo
type memoryRunRepo struct {
	mu   sync.Mutex
	runs map[string]*domain.Run
}

func newMemRunRepo() *memoryRunRepo { return &memoryRunRepo{runs: map[string]*domain.Run{}} }

func (m *memoryRunRepo) CreateRun(ctx context.Context, r *domain.Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *r
	cp.ParticipatingClients = append([]string{}, r.ParticipatingClients...)
	m.runs[r.RunID] = &cp
	return nil
}

func (m *memoryRunRepo) GetRun(ctx context.Context, runID string) (*domain.Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.runs[runID]; ok {
		cp := *r
		cp.ParticipatingClients = append([]string{}, r.ParticipatingClients...)
		return &cp, nil
	}
	return nil, domain.ErrRunNotFound
}

func (m *memoryRunRepo) UpdateRun(ctx context.Context, r *domain.Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.runs[r.RunID]; !ok {
		return domain.ErrRunNotFound
	}
	cp := *r
	cp.ParticipatingClients = append([]string{}, r.ParticipatingClients...)
	m.runs[r.RunID] = &cp
	return nil
}

func (m *memoryRunRepo) ListRuns(ctx context.Context, workflowID string, limit int) ([]*domain.Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []*domain.Run{}
	for _, r := range m.runs {
		if r.WorkflowID == workflowID {
			cp := *r
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TriggeredAt.After(out[j].TriggeredAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryRunRepo) ListRunsByWorkflowType(ctx context.Context, workflowType string, limit int) ([]*domain.Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []*domain.Run{}
	for _, r := range m.runs {
		if r.WorkflowType == workflowType {
			cp := *r
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TriggeredAt.After(out[j].TriggeredAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memoryRunRepo) GetPreviousRun(ctx context.Context, clientID, workflowType, currentRunID string, before time.Time) (*domain.Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var best *domain.Run
	for _, r := range m.runs {
		if r.RunID == currentRunID || r.WorkflowType != workflowType {
			continue
		}
		if r.TriggeredAt.After(before) {
			continue
		}
		contains := false
		for _, cid := range r.ParticipatingClients {
			if cid == clientID {
				contains = true
				break
			}
		}
		if !contains {
			continue
		}
		if best == nil || r.TriggeredAt.After(best.TriggeredAt) {
			cp := *r
			best = &cp
		}
	}
	if best == nil {
		return nil, errors.New("not found")
	}
	return best, nil
}

// memoryResultRepo
type memoryResultRepo struct {
	mu      sync.Mutex
	results map[string]map[string]*domain.Result // runID -> clientID -> result
}

func newMemResultRepo() *memoryResultRepo {
	return &memoryResultRepo{results: map[string]map[string]*domain.Result{}}
}

func (m *memoryResultRepo) SaveResult(ctx context.Context, r *domain.Result) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.results[r.RunID]; !ok {
		m.results[r.RunID] = map[string]*domain.Result{}
	}
	if _, exists := m.results[r.RunID][r.ClientID]; exists {
		return nil
	}
	cp := *r
	m.results[r.RunID][r.ClientID] = &cp
	return nil
}

func (m *memoryResultRepo) SaveResultWithType(ctx context.Context, r *domain.Result, workflowType string) error {
	return m.SaveResult(ctx, r)
}

func (m *memoryResultRepo) GetResult(ctx context.Context, runID, clientID string) (*domain.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rm, ok := m.results[runID]; ok {
		if r, ok := rm[clientID]; ok {
			cp := *r
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *memoryResultRepo) GetRunResults(ctx context.Context, runID string) ([]*domain.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rm, ok := m.results[runID]
	if !ok {
		return nil, nil
	}
	out := make([]*domain.Result, 0, len(rm))
	for _, r := range rm {
		cp := *r
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memoryResultRepo) ListClientResults(ctx context.Context, clientID, workflowType string, limit int) ([]*domain.Result, error) {
	return nil, nil
}

// memoryBanRepo
type memoryBanRepo struct {
	mu   sync.Mutex
	bans []*domain.BanRecord
}

func newMemBanRepo() *memoryBanRepo { return &memoryBanRepo{} }

func (m *memoryBanRepo) SaveBan(ctx context.Context, b *domain.BanRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *b
	cp.ID = int64(len(m.bans) + 1)
	m.bans = append(m.bans, &cp)
	b.ID = cp.ID
	return nil
}

func (m *memoryBanRepo) GetBans(ctx context.Context, clientID string) ([]*domain.BanRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*domain.BanRecord
	for _, b := range m.bans {
		if b.ClientID == clientID {
			cp := *b
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *memoryBanRepo) GetActiveBans(ctx context.Context, clientID string) ([]*domain.BanRecord, error) {
	all, _ := m.GetBans(ctx, clientID)
	var out []*domain.BanRecord
	for _, b := range all {
		if b.IsActive() {
			out = append(out, b)
		}
	}
	return out, nil
}

func (m *memoryBanRepo) GetActiveBansByWorkflowType(ctx context.Context, workflowType string) ([]*domain.BanRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*domain.BanRecord
	for _, b := range m.bans {
		if b.WorkflowType == workflowType && b.IsActive() {
			cp := *b
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (m *memoryBanRepo) UnbanClient(ctx context.Context, clientID, workflowType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	changed := false
	for _, b := range m.bans {
		if b.ClientID == clientID && b.WorkflowType == workflowType && b.Active {
			b.Active = false
			changed = true
		}
	}
	if !changed {
		return domain.ErrBanNotFound
	}
	return nil
}

func (m *memoryBanRepo) ListAllBans(ctx context.Context) ([]*domain.BanRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*domain.BanRecord, 0, len(m.bans))
	for _, b := range m.bans {
		cp := *b
		out = append(out, &cp)
	}
	return out, nil
}

// memoryHealthRepo
type memoryHealthRepo struct {
	mu       sync.Mutex
	runs     map[string]*domain.RunHealth
	wfHealth map[string]*domain.WorkflowTypeHealth
}

func newMemHealthRepo() *memoryHealthRepo {
	return &memoryHealthRepo{
		runs:     map[string]*domain.RunHealth{},
		wfHealth: map[string]*domain.WorkflowTypeHealth{},
	}
}

func (m *memoryHealthRepo) SaveRunHealth(ctx context.Context, h *domain.RunHealth) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *h
	m.runs[h.RunID] = &cp
	return nil
}

func (m *memoryHealthRepo) GetRunHealth(ctx context.Context, runID string) (*domain.RunHealth, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.runs[runID]; ok {
		cp := *h
		return &cp, nil
	}
	return nil, errors.New("not found")
}

func (m *memoryHealthRepo) ListRunHealths(ctx context.Context, workflowType string, limit int) ([]*domain.RunHealth, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []*domain.RunHealth{}
	for _, h := range m.runs {
		if h.WorkflowType == workflowType {
			cp := *h
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CalculatedAt.Before(out[j].CalculatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func (m *memoryHealthRepo) SaveWorkflowTypeHealth(ctx context.Context, h *domain.WorkflowTypeHealth) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *h
	m.wfHealth[h.WorkflowType] = &cp
	return nil
}

func (m *memoryHealthRepo) GetWorkflowTypeHealth(ctx context.Context, workflowType string) (*domain.WorkflowTypeHealth, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.wfHealth[workflowType]; ok {
		cp := *h
		return &cp, nil
	}
	return nil, errors.New("not found")
}

func (m *memoryHealthRepo) ListAllWorkflowTypeHealths(ctx context.Context) ([]*domain.WorkflowTypeHealth, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*domain.WorkflowTypeHealth, 0, len(m.wfHealth))
	for _, h := range m.wfHealth {
		cp := *h
		out = append(out, &cp)
	}
	return out, nil
}

// memoryCircuitRepo
type memoryCircuitRepo struct {
	mu     sync.Mutex
	states map[string]*domain.WorkflowCircuitBreaker
}

func newMemCircuitRepo() *memoryCircuitRepo {
	return &memoryCircuitRepo{states: map[string]*domain.WorkflowCircuitBreaker{}}
}

func (m *memoryCircuitRepo) SaveCircuitState(ctx context.Context, s *domain.WorkflowCircuitBreaker) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *s
	m.states[s.WorkflowID] = &cp
	return nil
}

func (m *memoryCircuitRepo) GetCircuitState(ctx context.Context, workflowID string) (*domain.WorkflowCircuitBreaker, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.states[workflowID]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, domain.ErrCircuitStateNotFound
}

func (m *memoryCircuitRepo) ListCircuitStates(ctx context.Context) ([]*domain.WorkflowCircuitBreaker, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []*domain.WorkflowCircuitBreaker{}
	for _, s := range m.states {
		cp := *s
		out = append(out, &cp)
	}
	return out, nil
}

// stubBlocker
type stubBlocker struct {
	mu   sync.Mutex
	bans map[string]map[string]bool
}

func newStubBlocker() *stubBlocker { return &stubBlocker{bans: map[string]map[string]bool{}} }
func (s *stubBlocker) ShouldBlockDispatch(ctx context.Context, clientID, workflowType string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.bans[workflowType]; ok && m[clientID] {
		return true
	}
	if m, ok := s.bans[""]; ok && m[clientID] {
		return true
	}
	return false
}
func (s *stubBlocker) Add(b *domain.BanRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !b.IsActive() {
		return
	}
	if s.bans[b.WorkflowType] == nil {
		s.bans[b.WorkflowType] = map[string]bool{}
	}
	s.bans[b.WorkflowType][b.ClientID] = true
}
func (s *stubBlocker) Remove(clientID, workflowType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.bans[workflowType]; ok {
		delete(m, clientID)
	}
}

// stubAlerter
type stubAlerter struct {
	mu     sync.Mutex
	Alerts []*domain.Alert
}

func newStubAlerter() *stubAlerter { return &stubAlerter{} }
func (s *stubAlerter) PublishAlert(ctx context.Context, a *domain.Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Alerts = append(s.Alerts, a)
	return nil
}
func (s *stubAlerter) Count(kind domain.AlertKind) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := 0
	for _, a := range s.Alerts {
		if a.Kind == kind {
			c++
		}
	}
	return c
}

// stubDispatcher
type stubDispatcher struct {
	mu     sync.Mutex
	Sent   []*domain.Dispatch
	OnSend func(d *domain.Dispatch) error
}

func newStubDispatcher() *stubDispatcher { return &stubDispatcher{} }
func (s *stubDispatcher) SendDispatch(ctx context.Context, d *domain.Dispatch) error {
	if s.OnSend != nil {
		if err := s.OnSend(d); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *d
	s.Sent = append(s.Sent, &cp)
	return nil
}
func (s *stubDispatcher) SendBatchDispatches(ctx context.Context, list []*domain.Dispatch) error {
	for _, d := range list {
		if err := s.SendDispatch(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// stubEventBus
type stubEventBus struct {
	mu     sync.Mutex
	subs   map[string][]ports.EventHandler
	Events []domain.Event
}

func newStubEventBus() *stubEventBus {
	return &stubEventBus{subs: map[string][]ports.EventHandler{}}
}
func (s *stubEventBus) Publish(ctx context.Context, event domain.Event) error {
	s.mu.Lock()
	handlers := append([]ports.EventHandler(nil), s.subs[event.EventType()]...)
	s.Events = append(s.Events, event)
	s.mu.Unlock()
	for _, h := range handlers {
		_ = h(ctx, event)
	}
	return nil
}
func (s *stubEventBus) Subscribe(eventType string, handler ports.EventHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[eventType] = append(s.subs[eventType], handler)
	return nil
}
