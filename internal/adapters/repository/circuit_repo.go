package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
)

type SQLiteCircuitRepo struct{ db *sql.DB }

func NewSQLiteCircuitRepo(db *sql.DB) *SQLiteCircuitRepo { return &SQLiteCircuitRepo{db: db} }

func (r *SQLiteCircuitRepo) SaveCircuitState(ctx context.Context, s *domain.WorkflowCircuitBreaker) error {
	if s.LastEvaluatedAt.IsZero() {
		s.LastEvaluatedAt = time.Now().UTC()
	}
	var openedAt interface{}
	if !s.OpenedAt.IsZero() {
		openedAt = s.OpenedAt
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO circuit_breaker_states (workflow_id, workflow_type, state, opened_at, last_evaluated_at, opened_reason, evaluation_count)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(workflow_id) DO UPDATE SET workflow_type=excluded.workflow_type, state=excluded.state, opened_at=excluded.opened_at, last_evaluated_at=excluded.last_evaluated_at, opened_reason=excluded.opened_reason, evaluation_count=excluded.evaluation_count
`, s.WorkflowID, s.WorkflowType, string(s.State), openedAt, s.LastEvaluatedAt, s.OpenedReason, s.EvaluationCount)
	return err
}

func (r *SQLiteCircuitRepo) GetCircuitState(ctx context.Context, workflowID string) (*domain.WorkflowCircuitBreaker, error) {
	row := r.db.QueryRowContext(ctx, `SELECT workflow_id, workflow_type, state, opened_at, last_evaluated_at, opened_reason, evaluation_count FROM circuit_breaker_states WHERE workflow_id=?`, workflowID)
	return scanCircuit(row)
}

func (r *SQLiteCircuitRepo) ListCircuitStates(ctx context.Context) ([]*domain.WorkflowCircuitBreaker, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT workflow_id, workflow_type, state, opened_at, last_evaluated_at, opened_reason, evaluation_count FROM circuit_breaker_states`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.WorkflowCircuitBreaker
	for rows.Next() {
		s, err := scanCircuit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanCircuit(row rowScanner) (*domain.WorkflowCircuitBreaker, error) {
	var s domain.WorkflowCircuitBreaker
	var state string
	var openedAt sql.NullTime
	var reason sql.NullString
	err := row.Scan(&s.WorkflowID, &s.WorkflowType, &state, &openedAt, &s.LastEvaluatedAt, &reason, &s.EvaluationCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCircuitStateNotFound
		}
		return nil, err
	}
	s.State = domain.CircuitState(state)
	if openedAt.Valid {
		s.OpenedAt = openedAt.Time
	}
	if reason.Valid {
		s.OpenedReason = reason.String
	}
	return &s, nil
}
