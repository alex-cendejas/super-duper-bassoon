package repository

import (
	"testing"
)

func TestRegistry_DB(t *testing.T) {
	r := openTestDB(t)
	if r.DB() == nil {
		t.Error("DB() should be non-nil")
	}
}
