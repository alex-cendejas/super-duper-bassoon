package repository

import (
	"context"
	"database/sql"

	"github.com/super-duper-bassoon/server/internal/core/ports"
)

type Registry struct {
	db          *sql.DB
	workflows   *SQLiteWorkflowRepo
	clients     *SQLiteClientRepo
	runs        *SQLiteRunRepo
	results     *SQLiteResultRepo
	bans        *SQLiteBanRepo
	health      *SQLiteHealthRepo
	circuits    *SQLiteCircuitRepo
}

func NewRegistry(db *sql.DB) *Registry {
	return &Registry{
		db:        db,
		workflows: NewSQLiteWorkflowRepo(db),
		clients:   NewSQLiteClientRepo(db),
		runs:      NewSQLiteRunRepo(db),
		results:   NewSQLiteResultRepo(db),
		bans:      NewSQLiteBanRepo(db),
		health:    NewSQLiteHealthRepo(db),
		circuits:  NewSQLiteCircuitRepo(db),
	}
}

func (r *Registry) Workflow() ports.WorkflowRepository             { return r.workflows }
func (r *Registry) Client() ports.ClientRepository                  { return r.clients }
func (r *Registry) Run() ports.RunRepository                        { return r.runs }
func (r *Registry) Result() ports.ResultRepository                  { return r.results }
func (r *Registry) Ban() ports.BanRepository                        { return r.bans }
func (r *Registry) Health() ports.HealthRepository                  { return r.health }
func (r *Registry) Circuit() ports.CircuitBreakerStateRepository    { return r.circuits }
func (r *Registry) DB() *sql.DB                                     { return r.db }

func (r *Registry) HealthCheck(ctx context.Context) error {
	return r.db.PingContext(ctx)
}
