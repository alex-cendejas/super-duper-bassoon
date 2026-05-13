package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
)

type SQLiteBanRepo struct{ db *sql.DB }

func NewSQLiteBanRepo(db *sql.DB) *SQLiteBanRepo { return &SQLiteBanRepo{db: db} }

func (r *SQLiteBanRepo) SaveBan(ctx context.Context, b *domain.BanRecord) error {
	if b.BannedAt.IsZero() {
		b.BannedAt = time.Now().UTC()
	}
	var until interface{}
	if b.BannedUntil != nil {
		until = *b.BannedUntil
	}
	res, err := r.db.ExecContext(ctx, `
INSERT INTO bans (client_id, workflow_type, run_id_evidence, result_evidence, banned_at, banned_until, reason, banned_by, active)
VALUES (?,?,?,?,?,?,?,?,?)
`, b.ClientID, b.WorkflowType, b.RunIDEvidence, b.ResultEvidence, b.BannedAt, until, string(b.Reason), b.BannedBy, boolInt(b.Active))
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	b.ID = id
	return nil
}

func (r *SQLiteBanRepo) GetBans(ctx context.Context, clientID string) ([]*domain.BanRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,client_id,workflow_type,run_id_evidence,result_evidence,banned_at,banned_until,reason,banned_by,active FROM bans WHERE client_id=? ORDER BY banned_at DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBans(rows)
}

func (r *SQLiteBanRepo) GetActiveBans(ctx context.Context, clientID string) ([]*domain.BanRecord, error) {
	now := time.Now().UTC()
	rows, err := r.db.QueryContext(ctx, `SELECT id,client_id,workflow_type,run_id_evidence,result_evidence,banned_at,banned_until,reason,banned_by,active FROM bans WHERE client_id=? AND active=1 AND (banned_until IS NULL OR banned_until > ?)`, clientID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBans(rows)
}

func (r *SQLiteBanRepo) GetActiveBansByWorkflowType(ctx context.Context, workflowType string) ([]*domain.BanRecord, error) {
	now := time.Now().UTC()
	rows, err := r.db.QueryContext(ctx, `SELECT id,client_id,workflow_type,run_id_evidence,result_evidence,banned_at,banned_until,reason,banned_by,active FROM bans WHERE workflow_type=? AND active=1 AND (banned_until IS NULL OR banned_until > ?)`, workflowType, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBans(rows)
}

func (r *SQLiteBanRepo) UnbanClient(ctx context.Context, clientID, workflowType string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE bans SET active=0 WHERE client_id=? AND workflow_type=? AND active=1`, clientID, workflowType)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrBanNotFound
	}
	return nil
}

func (r *SQLiteBanRepo) ListAllBans(ctx context.Context) ([]*domain.BanRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,client_id,workflow_type,run_id_evidence,result_evidence,banned_at,banned_until,reason,banned_by,active FROM bans ORDER BY banned_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBans(rows)
}

func collectBans(rows *sql.Rows) ([]*domain.BanRecord, error) {
	out := make([]*domain.BanRecord, 0)
	for rows.Next() {
		var b domain.BanRecord
		var until sql.NullTime
		var activeInt int
		var reason string
		var evidence sql.NullString
		var bannedBy sql.NullString
		var runEvidence sql.NullString
		if err := rows.Scan(&b.ID, &b.ClientID, &b.WorkflowType, &runEvidence, &evidence, &b.BannedAt, &until, &reason, &bannedBy, &activeInt); err != nil {
			return nil, err
		}
		b.Active = activeInt != 0
		b.Reason = domain.BanReason(reason)
		if runEvidence.Valid {
			b.RunIDEvidence = runEvidence.String
		}
		if evidence.Valid {
			b.ResultEvidence = evidence.String
		}
		if bannedBy.Valid {
			b.BannedBy = bannedBy.String
		}
		if until.Valid {
			t := until.Time
			b.BannedUntil = &t
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}
