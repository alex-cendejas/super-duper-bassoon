package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
)

type SQLiteHealthRepo struct{ db *sql.DB }

func NewSQLiteHealthRepo(db *sql.DB) *SQLiteHealthRepo { return &SQLiteHealthRepo{db: db} }

func (r *SQLiteHealthRepo) SaveRunHealth(ctx context.Context, h *domain.RunHealth) error {
	if h.CalculatedAt.IsZero() {
		h.CalculatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO run_health (run_id,workflow_id,workflow_type,total_clients,success_count,fail_count,error_count,pending_count,banned_count,calculated_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(run_id) DO UPDATE SET workflow_id=excluded.workflow_id, workflow_type=excluded.workflow_type, total_clients=excluded.total_clients, success_count=excluded.success_count, fail_count=excluded.fail_count, error_count=excluded.error_count, pending_count=excluded.pending_count, banned_count=excluded.banned_count, calculated_at=excluded.calculated_at
`, h.RunID, h.WorkflowID, h.WorkflowType, h.TotalClients, h.SuccessCount, h.FailCount, h.ErrorCount, h.PendingCount, h.BannedClientCount, h.CalculatedAt)
	return err
}

func (r *SQLiteHealthRepo) GetRunHealth(ctx context.Context, runID string) (*domain.RunHealth, error) {
	row := r.db.QueryRowContext(ctx, `SELECT run_id,workflow_id,workflow_type,total_clients,success_count,fail_count,error_count,pending_count,banned_count,calculated_at FROM run_health WHERE run_id=?`, runID)
	return scanRunHealth(row)
}

func (r *SQLiteHealthRepo) ListRunHealths(ctx context.Context, workflowType string, limit int) ([]*domain.RunHealth, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT run_id,workflow_id,workflow_type,total_clients,success_count,fail_count,error_count,pending_count,banned_count,calculated_at FROM run_health WHERE workflow_type=? ORDER BY calculated_at DESC LIMIT ?`, workflowType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.RunHealth
	for rows.Next() {
		h, err := scanRunHealth(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	// Reverse so callers get chronological order (oldest first).
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}

func (r *SQLiteHealthRepo) SaveWorkflowTypeHealth(ctx context.Context, h *domain.WorkflowTypeHealth) error {
	if h.CalculatedAt.IsZero() {
		h.CalculatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO workflow_type_health (workflow_type, runs_considered, window_size, success_pct_avg, fail_pct_avg, error_pct_avg, trend, calculated_at)
VALUES (?,?,?,?,?,?,?,?)
ON CONFLICT(workflow_type) DO UPDATE SET runs_considered=excluded.runs_considered, window_size=excluded.window_size, success_pct_avg=excluded.success_pct_avg, fail_pct_avg=excluded.fail_pct_avg, error_pct_avg=excluded.error_pct_avg, trend=excluded.trend, calculated_at=excluded.calculated_at
`, h.WorkflowType, h.RunsConsidered, h.WindowSize, h.SuccessPercentageAvg, h.FailPercentageAvg, h.ErrorPercentageAvg, string(h.Trend), h.CalculatedAt)
	return err
}

func (r *SQLiteHealthRepo) GetWorkflowTypeHealth(ctx context.Context, workflowType string) (*domain.WorkflowTypeHealth, error) {
	row := r.db.QueryRowContext(ctx, `SELECT workflow_type, runs_considered, window_size, success_pct_avg, fail_pct_avg, error_pct_avg, trend, calculated_at FROM workflow_type_health WHERE workflow_type=?`, workflowType)
	var h domain.WorkflowTypeHealth
	var trend string
	err := row.Scan(&h.WorkflowType, &h.RunsConsidered, &h.WindowSize, &h.SuccessPercentageAvg, &h.FailPercentageAvg, &h.ErrorPercentageAvg, &trend, &h.CalculatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("workflow_type_health not found")
		}
		return nil, err
	}
	h.Trend = domain.TrendDirection(trend)
	return &h, nil
}

func (r *SQLiteHealthRepo) ListAllWorkflowTypeHealths(ctx context.Context) ([]*domain.WorkflowTypeHealth, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT workflow_type, runs_considered, window_size, success_pct_avg, fail_pct_avg, error_pct_avg, trend, calculated_at FROM workflow_type_health ORDER BY workflow_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.WorkflowTypeHealth
	for rows.Next() {
		var h domain.WorkflowTypeHealth
		var trend string
		err := rows.Scan(&h.WorkflowType, &h.RunsConsidered, &h.WindowSize, &h.SuccessPercentageAvg, &h.FailPercentageAvg, &h.ErrorPercentageAvg, &trend, &h.CalculatedAt)
		if err != nil {
			return nil, err
		}
		h.Trend = domain.TrendDirection(trend)
		out = append(out, &h)
	}
	return out, rows.Err()
}

func scanRunHealth(row rowScanner) (*domain.RunHealth, error) {
	var h domain.RunHealth
	err := row.Scan(&h.RunID, &h.WorkflowID, &h.WorkflowType, &h.TotalClients, &h.SuccessCount, &h.FailCount, &h.ErrorCount, &h.PendingCount, &h.BannedClientCount, &h.CalculatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("run_health not found")
		}
		return nil, err
	}
	return &h, nil
}
