package domain

import (
	"encoding/json"
	"testing"
)

func TestDispatch_IsValid(t *testing.T) {
	d := &Dispatch{RunID: "r", WorkflowID: "w", Activity: ActivityReboot}
	if !d.IsValid() {
		t.Errorf("expected valid")
	}
	if (&Dispatch{}).IsValid() {
		t.Errorf("empty dispatch should be invalid")
	}
	if (&Dispatch{RunID: "r", WorkflowID: "w", Activity: ActivityType("?")}).IsValid() {
		t.Errorf("bad activity should be invalid")
	}
}

func TestDispatch_GetPayload(t *testing.T) {
	d := &Dispatch{RunID: "r1", WorkflowID: "w1", Activity: ActivityReboot, Params: map[string]interface{}{"k": "v"}}
	b, err := d.GetPayload()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(b, &got)
	if got["run_id"] != "r1" || got["wf_id"] != "w1" || got["activity"] != "reboot" {
		t.Errorf("unexpected payload: %v", got)
	}
}

func TestParseResult_OK(t *testing.T) {
	data := []byte(`{"run_id":"r","wf_id":"w","client_id":"c","status":"success"}`)
	r, err := ParseResult(data)
	if err != nil {
		t.Fatal(err)
	}
	if !r.IsValid() {
		t.Error("expected valid")
	}
	if r.ReceivedAt.IsZero() {
		t.Error("ReceivedAt should be set")
	}
}

func TestParseResult_Invalid(t *testing.T) {
	if _, err := ParseResult([]byte("not json")); err == nil {
		t.Error("expected error")
	}
}

func TestResultStatus_IsValid(t *testing.T) {
	for _, s := range []ResultStatus{StatusSuccess, StatusFail, StatusError} {
		if !s.IsValid() {
			t.Errorf("%s should be valid", s)
		}
	}
	if ResultStatus("nope").IsValid() {
		t.Error("invalid status passed")
	}
}

func TestResult_IsValid(t *testing.T) {
	tests := []struct {
		r    Result
		want bool
	}{
		{Result{RunID: "r", WorkflowID: "w", ClientID: "c", Status: StatusSuccess}, true},
		{Result{}, false},
		{Result{RunID: "r", WorkflowID: "w", ClientID: "c", Status: ResultStatus("?")}, false},
	}
	for _, tc := range tests {
		if got := tc.r.IsValid(); got != tc.want {
			t.Errorf("IsValid=%v want %v", got, tc.want)
		}
	}
}
