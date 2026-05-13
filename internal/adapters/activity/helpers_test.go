package activity_test

import (
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/super-duper-bassoon/internal/core/domain"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// deterministicChaos always returns the same values.
type deterministicChaos struct {
	float64Val float64
	intnVal    int
}

func (d deterministicChaos) Float64() float64 { return d.float64Val }
func (d deterministicChaos) Intn(_ int) int   { return d.intnVal }

// sequentialChaos returns Float64 values from a slice in order, cycling when
// exhausted. Intn always returns intnVal.
type sequentialChaos struct {
	values  []float64
	idx     atomic.Int32
	intnVal int
}

func (s *sequentialChaos) Float64() float64 {
	i := int(s.idx.Add(1)) - 1
	return s.values[i%len(s.values)]
}

func (s *sequentialChaos) Intn(_ int) int { return s.intnVal }

// Ensure sequentialChaos implements domain.ChaosSource.
var _ domain.ChaosSource = (*sequentialChaos)(nil)
