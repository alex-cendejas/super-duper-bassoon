package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/super-duper-bassoon/server/internal/core/domain"
)

type SQLiteWorkflowRepo struct{ db *sql.DB }

func NewSQLiteWorkflowRepo(db *sql.DB) *SQLiteWorkflowRepo { return &SQLiteWorkflowRepo{db: db} }

func (r *SQLiteWorkflowRepo) SaveWorkflow(ctx context.Context, w *domain.Workflow) error {
	params, _ := json.Marshal(w.Params)
	trig, _ := json.Marshal(w.Trigger)
	if w.CreatedAt.IsZero() {
		w.CreatedAt = time.Now().UTC()
	}
	w.UpdatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
INSERT INTO workflows (id,name,description,workflow_type,activity,params_json,target_filter,trigger_json,success_threshold,loop_threshold_ms,timeout_ms,active,deactivated_reason,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET name=excluded.name, description=excluded.description, workflow_type=excluded.workflow_type, activity=excluded.activity, params_json=excluded.params_json, target_filter=excluded.target_filter, trigger_json=excluded.trigger_json, success_threshold=excluded.success_threshold, loop_threshold_ms=excluded.loop_threshold_ms, timeout_ms=excluded.timeout_ms, active=excluded.active, deactivated_reason=excluded.deactivated_reason, updated_at=excluded.updated_at
`, w.ID, w.Name, w.Description, w.WorkflowType, string(w.Activity), string(params), w.TargetFilter, string(trig), w.SuccessThreshold, w.LoopThresholdMS, w.TimeoutMS, boolInt(w.Active), w.DeactivatedReason, w.CreatedAt, w.UpdatedAt)
	return err
}

func (r *SQLiteWorkflowRepo) GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,name,description,workflow_type,activity,params_json,target_filter,trigger_json,success_threshold,loop_threshold_ms,timeout_ms,active,deactivated_reason,created_at,updated_at FROM workflows WHERE id=?`, id)
	return scanWorkflow(row)
}

func (r *SQLiteWorkflowRepo) ListAllWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	return r.queryWorkflows(ctx, `SELECT id,name,description,workflow_type,activity,params_json,target_filter,trigger_json,success_threshold,loop_threshold_ms,timeout_ms,active,deactivated_reason,created_at,updated_at FROM workflows ORDER BY created_at DESC`)
}

func (r *SQLiteWorkflowRepo) ListActiveWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	return r.queryWorkflows(ctx, `SELECT id,name,description,workflow_type,activity,params_json,target_filter,trigger_json,success_threshold,loop_threshold_ms,timeout_ms,active,deactivated_reason,created_at,updated_at FROM workflows WHERE active=1 ORDER BY created_at DESC`)
}

func (r *SQLiteWorkflowRepo) ListWorkflowsByType(ctx context.Context, workflowType string) ([]*domain.Workflow, error) {
	return r.queryWorkflows(ctx, `SELECT id,name,description,workflow_type,activity,params_json,target_filter,trigger_json,success_threshold,loop_threshold_ms,timeout_ms,active,deactivated_reason,created_at,updated_at FROM workflows WHERE workflow_type=?`, workflowType)
}

func (r *SQLiteWorkflowRepo) UpdateWorkflowState(ctx context.Context, id string, active bool, reason string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE workflows SET active=?, deactivated_reason=?, updated_at=? WHERE id=?`, boolInt(active), reason, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrWorkflowNotFound
	}
	return nil
}

func (r *SQLiteWorkflowRepo) DeleteWorkflow(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM workflows WHERE id=?`, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrWorkflowNotFound
	}
	return nil
}

func (r *SQLiteWorkflowRepo) queryWorkflows(ctx context.Context, q string, args ...interface{}) ([]*domain.Workflow, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Workflow
	for rows.Next() {
		w, err := scanWorkflow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanWorkflow(row rowScanner) (*domain.Workflow, error) {
	var w domain.Workflow
	var activeInt int
	var paramsJSON, triggerJSON sql.NullString
	var deactReason sql.NullString
	err := row.Scan(&w.ID, &w.Name, &w.Description, &w.WorkflowType, &w.Activity, &paramsJSON, &w.TargetFilter, &triggerJSON, &w.SuccessThreshold, &w.LoopThresholdMS, &w.TimeoutMS, &activeInt, &deactReason, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWorkflowNotFound
		}
		return nil, err
	}
	w.Active = activeInt != 0
	if paramsJSON.Valid && paramsJSON.String != "" {
		_ = json.Unmarshal([]byte(paramsJSON.String), &w.Params)
	}
	if triggerJSON.Valid && triggerJSON.String != "" {
		_ = json.Unmarshal([]byte(triggerJSON.String), &w.Trigger)
	}
	if deactReason.Valid {
		w.DeactivatedReason = deactReason.String
	}
	return &w, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
