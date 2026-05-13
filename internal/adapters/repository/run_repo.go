package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
)

type SQLiteRunRepo struct{ db *sql.DB }

func NewSQLiteRunRepo(db *sql.DB) *SQLiteRunRepo { return &SQLiteRunRepo{db: db} }

func (r *SQLiteRunRepo) CreateRun(ctx context.Context, run *domain.Run) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	clients, _ := json.Marshal(run.ParticipatingClients)
	_, err = tx.ExecContext(ctx, `
INSERT INTO runs (run_id,workflow_id,workflow_type,triggered_at,dispatched_at,state,reason,participating_clients_json)
VALUES (?,?,?,?,?,?,?,?)
`, run.RunID, run.WorkflowID, run.WorkflowType, run.TriggeredAt, nullableTime(run.DispatchedAt), string(run.State), run.Reason, string(clients))
	if err != nil {
		return err
	}
	for _, cid := range run.ParticipatingClients {
		_, err = tx.ExecContext(ctx, `INSERT INTO runs_clients (run_id, client_id) VALUES (?,?)`, run.RunID, cid)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *SQLiteRunRepo) GetRun(ctx context.Context, runID string) (*domain.Run, error) {
	row := r.db.QueryRowContext(ctx, `SELECT run_id,workflow_id,workflow_type,triggered_at,dispatched_at,state,reason,participating_clients_json FROM runs WHERE run_id=?`, runID)
	return scanRun(row)
}

func (r *SQLiteRunRepo) UpdateRun(ctx context.Context, run *domain.Run) error {
	clients, _ := json.Marshal(run.ParticipatingClients)
	res, err := r.db.ExecContext(ctx, `UPDATE runs SET workflow_id=?, workflow_type=?, triggered_at=?, dispatched_at=?, state=?, reason=?, participating_clients_json=? WHERE run_id=?`,
		run.WorkflowID, run.WorkflowType, run.TriggeredAt, nullableTime(run.DispatchedAt), string(run.State), run.Reason, string(clients), run.RunID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrRunNotFound
	}
	return nil
}

func (r *SQLiteRunRepo) ListRuns(ctx context.Context, workflowID string, limit int) ([]*domain.Run, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT run_id,workflow_id,workflow_type,triggered_at,dispatched_at,state,reason,participating_clients_json FROM runs WHERE workflow_id=? ORDER BY triggered_at DESC LIMIT ?`, workflowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRuns(rows)
}

func (r *SQLiteRunRepo) ListAllRuns(ctx context.Context, limit, offset int) ([]*domain.Run, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.QueryContext(ctx, `SELECT run_id,workflow_id,workflow_type,triggered_at,dispatched_at,state,reason,participating_clients_json FROM runs ORDER BY triggered_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRuns(rows)
}

func (r *SQLiteRunRepo) ListRunsByWorkflowType(ctx context.Context, workflowType string, limit int) ([]*domain.Run, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT run_id,workflow_id,workflow_type,triggered_at,dispatched_at,state,reason,participating_clients_json FROM runs WHERE workflow_type=? ORDER BY triggered_at DESC LIMIT ?`, workflowType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRuns(rows)
}

func (r *SQLiteRunRepo) GetPreviousRun(ctx context.Context, clientID, workflowType, currentRunID string, before time.Time) (*domain.Run, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT r.run_id,r.workflow_id,r.workflow_type,r.triggered_at,r.dispatched_at,r.state,r.reason,r.participating_clients_json
FROM runs r
JOIN runs_clients rc ON rc.run_id = r.run_id
WHERE rc.client_id = ? AND r.workflow_type = ? AND r.run_id != ? AND r.triggered_at <= ?
ORDER BY r.triggered_at DESC LIMIT 1
`, clientID, workflowType, currentRunID, before)
	return scanRun(row)
}

func collectRuns(rows *sql.Rows) ([]*domain.Run, error) {
	var out []*domain.Run
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

func scanRun(row rowScanner) (*domain.Run, error) {
	var r domain.Run
	var state string
	var triggered time.Time
	var dispatched sql.NullTime
	var clientsJSON sql.NullString
	var reason sql.NullString
	err := row.Scan(&r.RunID, &r.WorkflowID, &r.WorkflowType, &triggered, &dispatched, &state, &reason, &clientsJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRunNotFound
		}
		return nil, err
	}
	r.State = domain.RunState(state)
	r.TriggeredAt = triggered
	if dispatched.Valid {
		r.DispatchedAt = dispatched.Time
	}
	if reason.Valid {
		r.Reason = reason.String
	}
	if clientsJSON.Valid && clientsJSON.String != "" {
		_ = json.Unmarshal([]byte(clientsJSON.String), &r.ParticipatingClients)
	}
	return &r, nil
}

func nullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}
