package domain

import (
	"testing"
	"time"
)

func TestBanRecord_IsActive(t *testing.T) {
	b := &BanRecord{Active: true}
	if !b.IsActive() {
		t.Error("permanent => active")
	}
	if !b.IsPermanent() {
		t.Error("nil until => permanent")
	}
	future := time.Now().Add(time.Hour)
	b.BannedUntil = &future
	if !b.IsActive() {
		t.Error("future => active")
	}
	if b.IsPermanent() {
		t.Error("with until => not permanent")
	}
	past := time.Now().Add(-time.Hour)
	b.BannedUntil = &past
	if b.IsActive() {
		t.Error("past => not active")
	}
	b.Active = false
	b.BannedUntil = nil
	if b.IsActive() {
		t.Error("inactive flag")
	}
	if !b.CanUnban() == b.Active {
		// CanUnban returns b.Active so check
	}
}

func TestLoopDetector_DetectLoop(t *testing.T) {
	d := NewLoopDetector(0)
	now := time.Now()
	// no prev run
	if got := d.DetectLoop("c", "t", "r2", now, "", time.Time{}, 5*time.Second, "ev"); got != nil {
		t.Error("no prev => no loop")
	}
	// same run
	if got := d.DetectLoop("c", "t", "r1", now, "r1", now, 5*time.Second, "ev"); got != nil {
		t.Error("same run => no loop")
	}
	// outside threshold
	if got := d.DetectLoop("c", "t", "r2", now, "r1", now.Add(-time.Minute), time.Second, "ev"); got != nil {
		t.Error("outside threshold")
	}
	// within threshold
	rec := d.DetectLoop("c", "t", "r2", now, "r1", now.Add(-time.Second), 5*time.Second, "ev")
	if rec == nil {
		t.Fatal("expected loop")
	}
	if !rec.IsValid() {
		t.Error("rec invalid")
	}
	if rec.TimeBetween < time.Second-100*time.Millisecond || rec.TimeBetween > time.Second+time.Second {
		t.Errorf("delta wrong: %v", rec.TimeBetween)
	}
	// prev > current (clock skew) => no loop
	if got := d.DetectLoop("c", "t", "r2", now, "r1", now.Add(time.Second), 5*time.Second, "ev"); got != nil {
		t.Error("prev > cur => no loop")
	}
}

func TestBanManager_CanDispatch(t *testing.T) {
	m := NewBanManager()
	bans := []*BanRecord{
		{ClientID: "c1", WorkflowType: "t1", Active: true},
		{ClientID: "c2", WorkflowType: "", Active: true}, // global ban
		{ClientID: "c3", WorkflowType: "t1", Active: false},
	}
	if m.CanDispatchToClient("c1", "t1", bans) {
		t.Error("c1 banned for t1")
	}
	if !m.CanDispatchToClient("c1", "t2", bans) {
		t.Error("c1 not banned for t2")
	}
	if m.CanDispatchToClient("c2", "anything", bans) {
		t.Error("c2 globally banned")
	}
	if !m.CanDispatchToClient("c3", "t1", bans) {
		t.Error("c3 ban inactive")
	}
	if !m.CanDispatchToClient("cX", "t1", bans) {
		t.Error("unknown client")
	}
}

func TestBanManager_ApplyBan(t *testing.T) {
	m := NewBanManager()
	b := m.ApplyBan("c", "t", "r", "evidence", ReasonLoopDetected)
	if !b.Active {
		t.Error("active")
	}
	if !b.IsPermanent() {
		t.Error("permanent")
	}
	if b.Reason != ReasonLoopDetected {
		t.Error("reason")
	}
}
