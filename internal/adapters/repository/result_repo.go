package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/super-duper-bassoon/internal/core/domain"
)

type SQLiteResultRepo struct{ db *sql.DB }

func NewSQLiteResultRepo(db *sql.DB) *SQLiteResultRepo { return &SQLiteResultRepo{db: db} }

func (r *SQLiteResultRepo) SaveResult(ctx context.Context, res *domain.Result) error {
	innerState, _ := json.Marshal(res.InnerState)
	payload, _ := json.Marshal(res.Payload)
	workflowType := ""
	// workflowType is provided as a part of result's saved relations via the run lookup;
	// the caller may set Payload["workflow_type"] etc., here we accept the result.
	_, err := r.db.ExecContext(ctx, `
INSERT INTO results (run_id,client_id,workflow_id,workflow_type,status,inner_state_json,error_msg,payload_json,received_at)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT(run_id,client_id) DO NOTHING
`, res.RunID, res.ClientID, res.WorkflowID, workflowType, string(res.Status), string(innerState), res.ErrorMsg, string(payload), res.ReceivedAt)
	return err
}

// SaveResultWithType is a helper used in tests / services that know the workflow_type.
func (r *SQLiteResultRepo) SaveResultWithType(ctx context.Context, res *domain.Result, workflowType string) error {
	innerState, _ := json.Marshal(res.InnerState)
	payload, _ := json.Marshal(res.Payload)
	_, err := r.db.ExecContext(ctx, `
INSERT INTO results (run_id,client_id,workflow_id,workflow_type,status,inner_state_json,error_msg,payload_json,received_at)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT(run_id,client_id) DO NOTHING
`, res.RunID, res.ClientID, res.WorkflowID, workflowType, string(res.Status), string(innerState), res.ErrorMsg, string(payload), res.ReceivedAt)
	return err
}

func (r *SQLiteResultRepo) GetResult(ctx context.Context, runID, clientID string) (*domain.Result, error) {
	row := r.db.QueryRowContext(ctx, `SELECT run_id,client_id,workflow_id,status,inner_state_json,error_msg,payload_json,received_at FROM results WHERE run_id=? AND client_id=?`, runID, clientID)
	return scanResult(row)
}

func (r *SQLiteResultRepo) GetRunResults(ctx context.Context, runID string) ([]*domain.Result, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT run_id,client_id,workflow_id,status,inner_state_json,error_msg,payload_json,received_at FROM results WHERE run_id=?`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectResults(rows)
}

func (r *SQLiteResultRepo) ListClientResults(ctx context.Context, clientID, workflowType string, limit int) ([]*domain.Result, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT run_id,client_id,workflow_id,status,inner_state_json,error_msg,payload_json,received_at FROM results WHERE client_id=? AND workflow_type=? ORDER BY received_at DESC LIMIT ?`, clientID, workflowType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectResults(rows)
}

func collectResults(rows *sql.Rows) ([]*domain.Result, error) {
	var out []*domain.Result
	for rows.Next() {
		res, err := scanResult(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, rows.Err()
}

func scanResult(row rowScanner) (*domain.Result, error) {
	var r domain.Result
	var status string
	var innerJSON, payloadJSON sql.NullString
	var errMsg sql.NullString
	err := row.Scan(&r.RunID, &r.ClientID, &r.WorkflowID, &status, &innerJSON, &errMsg, &payloadJSON, &r.ReceivedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("result not found")
		}
		return nil, err
	}
	r.Status = domain.ResultStatus(status)
	if errMsg.Valid {
		r.ErrorMsg = errMsg.String
	}
	if innerJSON.Valid && innerJSON.String != "" {
		_ = json.Unmarshal([]byte(innerJSON.String), &r.InnerState)
	}
	if payloadJSON.Valid && payloadJSON.String != "" {
		_ = json.Unmarshal([]byte(payloadJSON.String), &r.Payload)
	}
	return &r, nil
}
