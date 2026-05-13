CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    workflow_type TEXT NOT NULL,
    activity TEXT NOT NULL,
    params_json TEXT,
    target_filter TEXT,
    trigger_json TEXT,
    success_threshold REAL NOT NULL DEFAULT 80,
    loop_threshold_ms INTEGER NOT NULL DEFAULT 5000,
    timeout_ms INTEGER NOT NULL DEFAULT 30000,
    active INTEGER NOT NULL DEFAULT 1,
    deactivated_reason TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_workflows_type ON workflows(workflow_type);
CREATE INDEX IF NOT EXISTS idx_workflows_active ON workflows(active);

CREATE TABLE IF NOT EXISTS clients (
    id TEXT PRIMARY KEY,
    os TEXT,
    labels_json TEXT,
    inner_state_json TEXT,
    active INTEGER NOT NULL DEFAULT 1,
    last_seen_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS runs (
    run_id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    workflow_type TEXT NOT NULL,
    triggered_at TIMESTAMP NOT NULL,
    dispatched_at TIMESTAMP,
    state TEXT NOT NULL,
    reason TEXT,
    participating_clients_json TEXT
);
CREATE INDEX IF NOT EXISTS idx_runs_workflow_id ON runs(workflow_id);
CREATE INDEX IF NOT EXISTS idx_runs_workflow_type ON runs(workflow_type);
CREATE INDEX IF NOT EXISTS idx_runs_triggered_at ON runs(triggered_at);

CREATE TABLE IF NOT EXISTS runs_clients (
    run_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    PRIMARY KEY (run_id, client_id)
);
CREATE INDEX IF NOT EXISTS idx_runs_clients_client ON runs_clients(client_id);

CREATE TABLE IF NOT EXISTS results (
    run_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    workflow_id TEXT NOT NULL,
    workflow_type TEXT NOT NULL,
    status TEXT NOT NULL,
    inner_state_json TEXT,
    error_msg TEXT,
    payload_json TEXT,
    received_at TIMESTAMP NOT NULL,
    PRIMARY KEY (run_id, client_id)
);
CREATE INDEX IF NOT EXISTS idx_results_run ON results(run_id);
CREATE INDEX IF NOT EXISTS idx_results_client_workflow ON results(client_id, workflow_type);

CREATE TABLE IF NOT EXISTS bans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id TEXT NOT NULL,
    workflow_type TEXT NOT NULL DEFAULT '',
    run_id_evidence TEXT,
    result_evidence TEXT,
    banned_at TIMESTAMP NOT NULL,
    banned_until TIMESTAMP,
    reason TEXT NOT NULL,
    banned_by TEXT,
    active INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_bans_client ON bans(client_id);
CREATE INDEX IF NOT EXISTS idx_bans_workflow_type ON bans(workflow_type);
CREATE INDEX IF NOT EXISTS idx_bans_client_wf ON bans(client_id, workflow_type);

CREATE TABLE IF NOT EXISTS run_health (
    run_id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    workflow_type TEXT NOT NULL,
    total_clients INTEGER NOT NULL,
    success_count INTEGER NOT NULL,
    fail_count INTEGER NOT NULL,
    error_count INTEGER NOT NULL,
    pending_count INTEGER NOT NULL,
    banned_count INTEGER NOT NULL,
    calculated_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_run_health_type ON run_health(workflow_type);

CREATE TABLE IF NOT EXISTS workflow_type_health (
    workflow_type TEXT PRIMARY KEY,
    runs_considered INTEGER NOT NULL,
    window_size INTEGER NOT NULL,
    success_pct_avg REAL NOT NULL,
    fail_pct_avg REAL NOT NULL,
    error_pct_avg REAL NOT NULL,
    trend TEXT NOT NULL,
    calculated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS circuit_breaker_states (
    workflow_id TEXT PRIMARY KEY,
    workflow_type TEXT NOT NULL,
    state TEXT NOT NULL,
    opened_at TIMESTAMP,
    last_evaluated_at TIMESTAMP NOT NULL,
    opened_reason TEXT,
    evaluation_count INTEGER NOT NULL DEFAULT 0
);
