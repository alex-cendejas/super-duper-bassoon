package repository

import (
	"context"
	"testing"
	"time"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

func openTestDB(t *testing.T) *Registry {
	t.Helper()
	db, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewRegistry(db)
}

func TestWorkflowRepo_CRUD(t *testing.T) {
	r := openTestDB(t)
	ctx := context.Background()
	wf := &domain.Workflow{
		ID: "w1", Name: "n", WorkflowType: "t", Activity: domain.ActivityReboot,
		Active: true, SuccessThreshold: 90, LoopThresholdMS: 1000, TimeoutMS: 5000,
		Params: map[string]interface{}{"k": "v"},
		Trigger: domain.TriggerSpec{Kind: domain.TriggerScheduled, Cron: "* * * * *"},
	}
	if err := r.Workflow().SaveWorkflow(ctx, wf); err != nil {
		t.Fatal(err)
	}
	got, err := r.Workflow().GetWorkflow(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "n" || got.WorkflowType != "t" || got.Trigger.Cron != "* * * * *" {
		t.Errorf("got: %+v", got)
	}
	// Update via Save (upsert)
	wf.Name = "n2"
	if err := r.Workflow().SaveWorkflow(ctx, wf); err != nil {
		t.Fatal(err)
	}
	got, _ = r.Workflow().GetWorkflow(ctx, "w1")
	if got.Name != "n2" {
		t.Errorf("not upserted: %v", got.Name)
	}
	// State updates
	if err := r.Workflow().UpdateWorkflowState(ctx, "w1", false, "test"); err != nil {
		t.Fatal(err)
	}
	got, _ = r.Workflow().GetWorkflow(ctx, "w1")
	if got.Active {
		t.Error("expected inactive")
	}
	// List
	list, _ := r.Workflow().ListAllWorkflows(ctx)
	if len(list) != 1 {
		t.Error("list count")
	}
	listActive, _ := r.Workflow().ListActiveWorkflows(ctx)
	if len(listActive) != 0 {
		t.Error("no active workflows expected")
	}
	byType, _ := r.Workflow().ListWorkflowsByType(ctx, "t")
	if len(byType) != 1 {
		t.Error("by type")
	}
	// Delete
	if err := r.Workflow().DeleteWorkflow(ctx, "w1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Workflow().GetWorkflow(ctx, "w1"); err == nil {
		t.Error("should be deleted")
	}
	if err := r.Workflow().DeleteWorkflow(ctx, "w1"); err == nil {
		t.Error("expected not found")
	}
	if err := r.Workflow().UpdateWorkflowState(ctx, "missing", false, ""); err == nil {
		t.Error("expected not found")
	}
}

func TestClientRepo_CRUD(t *testing.T) {
	r := openTestDB(t)
	ctx := context.Background()
	c := &domain.ClientMetadata{ClientID: "c1", OS: "linux", Active: true, Labels: map[string]string{"env": "prod"}, InnerState: map[string]interface{}{"version": float64(2)}}
	if err := r.Client().SaveClient(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, err := r.Client().GetClientByID(ctx, "c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.OS != "linux" || got.Labels["env"] != "prod" {
		t.Errorf("got: %+v", got)
	}
	if got.LastSeenAt.IsZero() {
		t.Error("last_seen_at not set")
	}
	// Upsert
	c.OS = "darwin"
	_ = r.Client().SaveClient(ctx, c)
	got, _ = r.Client().GetClientByID(ctx, "c1")
	if got.OS != "darwin" {
		t.Error("upsert")
	}
	// List
	list, _ := r.Client().ListClients(ctx)
	if len(list) != 1 {
		t.Error("list")
	}
	// Multi-fetch
	multi, _ := r.Client().GetClientsByIDs(ctx, []string{"c1", "missing"})
	if len(multi) != 1 {
		t.Errorf("multi: %d", len(multi))
	}
	if _, err := r.Client().GetClientByID(ctx, "missing"); err == nil {
		t.Error("expected not found")
	}
	// Empty IDs
	out, _ := r.Client().GetClientsByIDs(ctx, nil)
	if out != nil {
		t.Error("expected nil")
	}
}

func TestRunResultRepo(t *testing.T) {
	r := openTestDB(t)
	ctx := context.Background()
	run := &domain.Run{RunID: "r1", WorkflowID: "w1", WorkflowType: "t", TriggeredAt: time.Now(), ParticipatingClients: []string{"c1", "c2"}, State: domain.RunPending}
	if err := r.Run().CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	got, err := r.Run().GetRun(ctx, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ParticipatingClients) != 2 {
		t.Error("clients")
	}
	got.State = domain.RunInProgress
	got.DispatchedAt = time.Now()
	if err := r.Run().UpdateRun(ctx, got); err != nil {
		t.Fatal(err)
	}
	upd, _ := r.Run().GetRun(ctx, "r1")
	if upd.State != domain.RunInProgress {
		t.Error("state not updated")
	}
	// List by workflow
	list, _ := r.Run().ListRuns(ctx, "w1", 0)
	if len(list) != 1 {
		t.Error("count")
	}
	byType, _ := r.Run().ListRunsByWorkflowType(ctx, "t", 0)
	if len(byType) != 1 {
		t.Error("byType")
	}

	// Save results
	res := &domain.Result{RunID: "r1", ClientID: "c1", WorkflowID: "w1", Status: domain.StatusSuccess, ReceivedAt: time.Now()}
	if err := r.Result().(*SQLiteResultRepo).SaveResultWithType(ctx, res, "t"); err != nil {
		t.Fatal(err)
	}
	// Idempotency: re-save should not error
	_ = r.Result().(*SQLiteResultRepo).SaveResultWithType(ctx, res, "t")
	all, _ := r.Result().GetRunResults(ctx, "r1")
	if len(all) != 1 {
		t.Errorf("expected idempotent 1 result, got %d", len(all))
	}
	getOne, err := r.Result().GetResult(ctx, "r1", "c1")
	if err != nil || getOne.Status != domain.StatusSuccess {
		t.Errorf("get one: %v %v", err, getOne)
	}
	clientHist, _ := r.Result().ListClientResults(ctx, "c1", "t", 10)
	if len(clientHist) != 1 {
		t.Error("client results")
	}

	// Previous run lookup
	run2 := &domain.Run{RunID: "r2", WorkflowID: "w1", WorkflowType: "t", TriggeredAt: time.Now().Add(time.Second), ParticipatingClients: []string{"c1"}, State: domain.RunPending}
	_ = r.Run().CreateRun(ctx, run2)
	prev, err := r.Run().GetPreviousRun(ctx, "c1", "t", "r2", time.Now().Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if prev.RunID != "r1" {
		t.Errorf("prev: %v", prev.RunID)
	}
	// Missing
	if _, err := r.Run().GetRun(ctx, "missing"); err == nil {
		t.Error("expected not found")
	}
	if err := r.Run().UpdateRun(ctx, &domain.Run{RunID: "missing"}); err == nil {
		t.Error("expected not found")
	}
}

func TestBanRepo(t *testing.T) {
	r := openTestDB(t)
	ctx := context.Background()
	b := &domain.BanRecord{ClientID: "c1", WorkflowType: "t1", Reason: domain.ReasonLoopDetected, Active: true, RunIDEvidence: "r1", ResultEvidence: "ev", BannedBy: "system"}
	if err := r.Ban().SaveBan(ctx, b); err != nil {
		t.Fatal(err)
	}
	if b.ID == 0 {
		t.Error("id not set")
	}
	// Active by client
	active, _ := r.Ban().GetActiveBans(ctx, "c1")
	if len(active) != 1 {
		t.Error("active count")
	}
	// By type
	byType, _ := r.Ban().GetActiveBansByWorkflowType(ctx, "t1")
	if len(byType) != 1 {
		t.Errorf("type count: %d", len(byType))
	}
	// Get all bans for client
	all, _ := r.Ban().GetBans(ctx, "c1")
	if len(all) != 1 {
		t.Error("getbans count")
	}
	// Unban
	if err := r.Ban().UnbanClient(ctx, "c1", "t1"); err != nil {
		t.Fatal(err)
	}
	active, _ = r.Ban().GetActiveBans(ctx, "c1")
	if len(active) != 0 {
		t.Error("should be inactive")
	}
	// Unban missing => error
	if err := r.Ban().UnbanClient(ctx, "missing", "t"); err == nil {
		t.Error("expected error")
	}
	// Temporal ban
	until := time.Now().Add(time.Hour)
	temp := &domain.BanRecord{ClientID: "c2", WorkflowType: "t1", Reason: domain.ReasonManual, Active: true, BannedUntil: &until}
	_ = r.Ban().SaveBan(ctx, temp)
	all, _ = r.Ban().ListAllBans(ctx)
	if len(all) != 2 {
		t.Errorf("all count: %d", len(all))
	}
}

func TestHealthRepo(t *testing.T) {
	r := openTestDB(t)
	ctx := context.Background()
	h := &domain.RunHealth{RunID: "r1", WorkflowID: "w1", WorkflowType: "t", TotalClients: 2, SuccessCount: 2}
	if err := r.Health().SaveRunHealth(ctx, h); err != nil {
		t.Fatal(err)
	}
	got, _ := r.Health().GetRunHealth(ctx, "r1")
	if got == nil || got.TotalClients != 2 {
		t.Errorf("got: %+v", got)
	}
	// Upsert
	h.SuccessCount = 1
	_ = r.Health().SaveRunHealth(ctx, h)
	got, _ = r.Health().GetRunHealth(ctx, "r1")
	if got.SuccessCount != 1 {
		t.Error("upsert")
	}
	list, _ := r.Health().ListRunHealths(ctx, "t", 10)
	if len(list) != 1 {
		t.Error("list count")
	}

	// Workflow type
	w := &domain.WorkflowTypeHealth{WorkflowType: "t", RunsConsidered: 2, WindowSize: 10, SuccessPercentageAvg: 90, Trend: domain.TrendStable}
	if err := r.Health().SaveWorkflowTypeHealth(ctx, w); err != nil {
		t.Fatal(err)
	}
	got2, _ := r.Health().GetWorkflowTypeHealth(ctx, "t")
	if got2.RunsConsidered != 2 {
		t.Error("workflow health")
	}
	listAll, _ := r.Health().ListAllWorkflowTypeHealths(ctx)
	if len(listAll) != 1 {
		t.Error("list all")
	}
	if _, err := r.Health().GetWorkflowTypeHealth(ctx, "missing"); err == nil {
		t.Error("expected error")
	}
}

func TestCircuitRepo(t *testing.T) {
	r := openTestDB(t)
	ctx := context.Background()
	s := &domain.WorkflowCircuitBreaker{WorkflowID: "w", WorkflowType: "t", State: domain.CircuitClosed}
	if err := r.Circuit().SaveCircuitState(ctx, s); err != nil {
		t.Fatal(err)
	}
	got, _ := r.Circuit().GetCircuitState(ctx, "w")
	if got.State != domain.CircuitClosed {
		t.Error("state")
	}
	// Open and persist OpenedAt
	s.State = domain.CircuitOpen
	s.OpenedAt = time.Now().UTC()
	_ = r.Circuit().SaveCircuitState(ctx, s)
	got, _ = r.Circuit().GetCircuitState(ctx, "w")
	if got.State != domain.CircuitOpen || got.OpenedAt.IsZero() {
		t.Errorf("open state: %+v", got)
	}
	list, _ := r.Circuit().ListCircuitStates(ctx)
	if len(list) != 1 {
		t.Error("count")
	}
	if _, err := r.Circuit().GetCircuitState(ctx, "missing"); err == nil {
		t.Error("expected not found")
	}
}

func TestRegistry_HealthCheck(t *testing.T) {
	r := openTestDB(t)
	if err := r.HealthCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestResultRepo_SaveResult(t *testing.T) {
	r := openTestDB(t)
	ctx := context.Background()
	res := &domain.Result{RunID: "r", ClientID: "c", WorkflowID: "w", Status: domain.StatusSuccess, ReceivedAt: time.Now()}
	if err := r.Result().SaveResult(ctx, res); err != nil {
		t.Fatal(err)
	}
	got, err := r.Result().GetResult(ctx, "r", "c")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusSuccess {
		t.Error("status")
	}
}
