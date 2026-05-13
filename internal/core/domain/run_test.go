package domain

import (
	"testing"
	"time"
)

func TestRun_IsExpired(t *testing.T) {
	now := time.Now()
	r := &Run{TriggeredAt: now.Add(-2 * time.Second)}
	if !r.IsExpired(time.Second, now) {
		t.Error("expired")
	}
	if r.IsExpired(10*time.Second, now) {
		t.Error("not expired")
	}
	zero := &Run{}
	if zero.IsExpired(time.Second, now) {
		t.Error("zero triggered_at should not expire")
	}
}

func TestRun_CanComplete(t *testing.T) {
	r := &Run{ParticipatingClients: []string{"a", "b", "c"}}
	if r.CanComplete(2) {
		t.Error("2 < 3")
	}
	if !r.CanComplete(3) {
		t.Error("3 of 3")
	}
}

func TestAPIError(t *testing.T) {
	e := NewAPIError("NOT_FOUND", "x")
	if e.GetHTTPStatus() != 404 {
		t.Error("not_found should be 404")
	}
	if e.Error() != "x" {
		t.Error("message")
	}
	e = NewAPIError("VALIDATION_ERROR", "x")
	if e.GetHTTPStatus() != 400 {
		t.Error("400")
	}
	e = NewAPIError("CONFLICT", "x")
	if e.GetHTTPStatus() != 409 {
		t.Error("409")
	}
	e = NewAPIError("FORBIDDEN", "x")
	if e.GetHTTPStatus() != 403 {
		t.Error("403")
	}
	e = NewAPIError("OTHER", "x")
	if e.GetHTTPStatus() != 500 {
		t.Error("500")
	}
}

func TestQueryFilter(t *testing.T) {
	q := &QueryFilter{}
	if q.GetLimit() != 10 {
		t.Error("default limit")
	}
	if q.GetOffset() != 0 {
		t.Error("default offset")
	}
	if err := q.Validate(); err != nil {
		t.Error("ok")
	}
	q.Limit = -1
	if err := q.Validate(); err == nil {
		t.Error("neg limit")
	}
	q.Limit = 100000
	if err := q.Validate(); err == nil {
		t.Error(">1000")
	}
	q.Limit = 50
	q.Offset = -1
	if err := q.Validate(); err == nil {
		t.Error("neg offset")
	}
	q.Offset = 5
	if q.GetOffset() != 5 || q.GetLimit() != 50 {
		t.Error("values")
	}
}

func TestCreateWorkflowRequest_Validate(t *testing.T) {
	r := &CreateWorkflowRequest{Name: "x", WorkflowType: "t", Activity: ActivityReboot}
	if err := r.Validate(); err != nil {
		t.Error("ok")
	}
	for _, bad := range []*CreateWorkflowRequest{
		{},
		{Name: "x"},
		{Name: "x", WorkflowType: "t"},
	} {
		if err := bad.Validate(); err == nil {
			t.Error("expected error")
		}
	}
}

func TestEditWorkflowRequest(t *testing.T) {
	r := &EditWorkflowRequest{}
	if r.HasChanges() {
		t.Error("empty has no changes")
	}
	if err := r.Validate(); err == nil {
		t.Error("validation should fail")
	}
	name := "n"
	r.Name = &name
	if !r.HasChanges() {
		t.Error("changes")
	}
	if err := r.Validate(); err != nil {
		t.Error("ok")
	}
}

func TestUnbanRequest_Validate(t *testing.T) {
	r := &UnbanRequest{}
	if err := r.Validate(); err == nil {
		t.Error("missing admin_id")
	}
	r.AdminID = "a"
	if err := r.Validate(); err == nil {
		t.Error("missing reason")
	}
	r.Reason = "x"
	if err := r.Validate(); err != nil {
		t.Error("ok")
	}
}
