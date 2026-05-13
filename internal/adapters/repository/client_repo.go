package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/super-duper-bassoon/internal/core/domain"
)

type SQLiteClientRepo struct{ db *sql.DB }

func NewSQLiteClientRepo(db *sql.DB) *SQLiteClientRepo { return &SQLiteClientRepo{db: db} }

func (r *SQLiteClientRepo) SaveClient(ctx context.Context, c *domain.ClientMetadata) error {
	labels, _ := json.Marshal(c.Labels)
	state, _ := json.Marshal(c.InnerState)
	if c.LastSeenAt.IsZero() {
		c.LastSeenAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO clients (id,os,labels_json,inner_state_json,active,last_seen_at)
VALUES (?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET os=excluded.os, labels_json=excluded.labels_json, inner_state_json=excluded.inner_state_json, active=excluded.active, last_seen_at=excluded.last_seen_at
`, c.ClientID, c.OS, string(labels), string(state), boolInt(c.Active), c.LastSeenAt)
	return err
}

func (r *SQLiteClientRepo) GetClientByID(ctx context.Context, id string) (*domain.ClientMetadata, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,os,labels_json,inner_state_json,active,last_seen_at FROM clients WHERE id=?`, id)
	return scanClient(row)
}

func (r *SQLiteClientRepo) ListClients(ctx context.Context) ([]*domain.ClientMetadata, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,os,labels_json,inner_state_json,active,last_seen_at FROM clients`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*domain.ClientMetadata, 0)
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *SQLiteClientRepo) GetClientsByIDs(ctx context.Context, ids []string) ([]*domain.ClientMetadata, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,os,labels_json,inner_state_json,active,last_seen_at FROM clients WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*domain.ClientMetadata, 0)
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanClient(row rowScanner) (*domain.ClientMetadata, error) {
	var c domain.ClientMetadata
	var activeInt int
	var labelsJSON, stateJSON sql.NullString
	var lastSeen sql.NullTime
	err := row.Scan(&c.ClientID, &c.OS, &labelsJSON, &stateJSON, &activeInt, &lastSeen)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrClientNotFound
		}
		return nil, err
	}
	c.Active = activeInt != 0
	if lastSeen.Valid {
		c.LastSeenAt = lastSeen.Time
	}
	c.Labels = map[string]string{}
	if labelsJSON.Valid && labelsJSON.String != "" {
		_ = json.Unmarshal([]byte(labelsJSON.String), &c.Labels)
	}
	c.InnerState = map[string]interface{}{}
	if stateJSON.Valid && stateJSON.String != "" {
		_ = json.Unmarshal([]byte(stateJSON.String), &c.InnerState)
	}
	return &c, nil
}
