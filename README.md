# Automation Engine Server

A lightweight workflow automation engine for managing and coordinating activities across a fleet of remote devices via asynchronous messaging.

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Services](#services)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Usage](#usage)
- [Filter Expression Language](#filter-expression-language)
- [Client Messaging Protocol](#client-messaging-protocol)
- [Safety Mechanisms](#safety-mechanisms)
- [API Reference](#api-reference)
- [Database Schema](#database-schema)
- [Development](#development)

---

## Overview

The Automation Engine Server is a hub-and-spoke orchestration system that dispatches activity commands to remote client devices, collects their results, and enforces safety policies automatically.

**Core capabilities:**

- Define workflows that target specific subsets of clients using a flexible filter expression language
- Trigger workflows on a schedule (cron), in response to events, or manually via API
- Dispatch activity commands to matching clients over NATS messaging
- Collect and persist execution results per client per run
- Monitor health trends across workflow types with configurable sliding windows
- Automatically open circuit breakers when health degrades below configured thresholds
- Detect and ban clients that re-enter the same workflow type too rapidly (loop detection)
- Expose all state and controls via a REST API

**Supported activities:** `reboot`, `install_package`, `upgrade_package`, `remove_package`, `apply_config`, `validate_config`, `run_script`

---

## Architecture

The server follows a **hexagonal (ports and adapters)** architecture, separating pure business logic from infrastructure concerns.

```
┌─────────────────────────────────────────────────────────┐
│                        HTTP API                         │
│                  (chi router, REST/JSON)                 │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│                    Service Layer                         │
│  WorkflowOrchestration │ DispatchCoordination           │
│  TriggerCoordination   │ HealthMonitoring                │
│  CircuitBreaker        │ LoopDetection                   │
│  BanEnforcement        │ DynamicGrouping                 │
└──────────┬─────────────────────────────┬────────────────┘
           │                             │
┌──────────▼──────────┐   ┌─────────────▼───────────────┐
│   Domain / Ports    │   │         Adapters             │
│  (pure Go, no deps) │   │  SQLite │ NATS │ Event Bus   │
└─────────────────────┘   └─────────────────────────────┘
```

### Layer responsibilities

| Layer | Location | Responsibility |
|---|---|---|
| Domain | `internal/core/domain/` | Value objects, aggregate types, filter AST, error types |
| Ports | `internal/core/ports/` | Repository and dispatcher interfaces |
| Services | `internal/core/services/` | All business logic |
| Adapters | `internal/adapters/` | SQLite persistence, NATS messaging, HTTP routing, alert publishing |
| Entry point | `cmd/` | Configuration loading, dependency wiring, startup/shutdown |

### Internal event bus

Services communicate through an in-process pub/sub event bus (`messaging.InMemoryEventBus`). For example, when the health monitoring service finishes aggregating a workflow type, it publishes a `health.updated` event that the circuit breaker service subscribes to for evaluation.

---

## Services

### WorkflowOrchestrationService

The central service for workflow lifecycle management. Handles creating, editing, deleting, activating, deactivating, and triggering workflows. When a workflow is triggered (manually or by the trigger coordinator), this service evaluates the target filter against all registered clients, creates a `Run` record, and delegates to `DispatchCoordinationService` to send activity commands.

### DynamicGroupingService

Evaluates a workflow's `target_filter` expression against the current set of registered clients, returning the subset that matches. Uses the filter expression parser to build an AST and evaluate each client's metadata against it.

### DispatchCoordinationService

Generates individual `Dispatch` messages for each eligible client in a run, applies the ban filter to exclude blocked clients, and sends the messages through the configured dispatcher (NATS in production, an in-process channel dispatcher when NATS is unavailable).

### DispatchFilterService

A thin wrapper around `BanEnforcementService` that determines which clients should be excluded from a given dispatch batch. Banned clients are tracked both in-memory for fast lookups and in the database for durability.

### BanEnforcementService

Manages the lifecycle of ban records. On startup, warms an in-memory cache (`InMemoryDispatchBlocker`) from the database so ban checks require no database round-trips during normal operation. Bans are created automatically by `LoopDetectionService` and can be removed by administrators via the unban API endpoint.

### LoopDetectionService

Processes every incoming result message and checks whether the reporting client re-entered the same workflow type faster than the configured `loop_threshold_ms`. If the previous result for that client/type arrived within the threshold window, the client is immediately banned from that workflow type and an alert is published.

### HealthMonitoringService

Aggregates per-run health metrics (success, fail, error, pending, banned counts) into sliding-window summaries per workflow type. Calculates a trend (improving, degrading, stable) by comparing the most recent half of the window to the earlier half. Publishes `health.updated` events after each aggregation.

### CircuitBreakerService

Subscribes to `health.updated` events and evaluates whether the aggregated success percentage for a workflow type falls below the configured threshold. When it does, the service deactivates the affected workflow(s) and opens their circuit breaker state. After the configured cooldown period, the circuit transitions to `half_open` and allows re-evaluation on the next health update.

### TriggerCoordinationService

Runs a background loop at `TRIGGER_CHECK_INTERVAL_MS` that evaluates scheduled (cron) triggers by comparing each active workflow's next-fire time against the current time. Also processes event-based triggers by subscribing to the internal event bus for `workflow.completed` events and chaining the configured `on_complete` workflow.

### ResultMessageDispatcher

Subscribes to the NATS `result.>` subject and fans out each incoming result byte slice to all registered handlers. Currently registered handlers are `LoopDetectionService` and `HealthMonitoringService`. Runs in a dedicated goroutine and stops cleanly on context cancellation.

### APIHandlerService

A thin facade over the other services, exposing a single unified API used by the HTTP handler layer. Avoids coupling HTTP handlers directly to multiple services.

---

## Configuration

All configuration is provided via environment variables. No configuration files are required.

| Variable | Default | Description |
|---|---|---|
| `HTTP_HOST` | `0.0.0.0` | Interface the HTTP server binds to |
| `HTTP_PORT` | `8080` | Port the HTTP server listens on (1–65535) |
| `HTTP_READ_TIMEOUT_MS` | `30000` | HTTP read and write timeout in milliseconds |
| `DB_PATH` | `./data/server.db` | Path to the SQLite database file. Parent directory is created automatically. |
| `NATS_URL` | `nats://localhost:4222` | NATS server connection URL |
| `START_NATS` | `true` | Set to `false` to skip NATS connection entirely (useful in testing) |
| `TRIGGER_CHECK_INTERVAL_MS` | `5000` | How often the trigger coordinator evaluates scheduled workflows |
| `HEALTH_AGGREGATION_INTERVAL_MS` | `5000` | How often health metrics are aggregated across active workflows |
| `HEALTH_WINDOW_SIZE` | `10` | Number of recent runs to include in the sliding health window |
| `HEALTH_SUCCESS_THRESHOLD` | `80` | Minimum success percentage (0–100) before the circuit breaker considers opening |
| `CIRCUIT_BREAKER_CHECK_INTERVAL_MS` | `10000` | How often circuit breaker evaluation runs independent of health events |
| `CIRCUIT_BREAKER_SUCCESS_THRESHOLD` | `80` | Success percentage threshold that triggers a circuit open |
| `CIRCUIT_BREAKER_COOLDOWN_MS` | `300000` | Cooldown duration (ms) before a circuit transitions from open to half_open |
| `LOOP_THRESHOLD_MS` | `5000` | If a client returns a result within this window of its previous result for the same workflow type, it is banned |
| `LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, or `error` |
| `SHUTDOWN_TIMEOUT_MS` | `30000` | Grace period for in-flight requests during shutdown |

---

## Deployment

### Prerequisites

- Go 1.25 or later
- A NATS server (optional — the server degrades gracefully without one, but no dispatches will be sent)
- Write access to the filesystem path specified by `DB_PATH`

### Build and run from source

```bash
# Build
go build -o automation-server ./cmd

# Run with defaults
./automation-server

# Run with custom configuration
HTTP_PORT=9090 \
DB_PATH=/var/lib/automation/server.db \
NATS_URL=nats://nats.internal:4222 \
LOG_LEVEL=debug \
./automation-server
```

### Run with `go run`

```bash
NATS_URL=nats://localhost:4222 \
HTTP_PORT=8080 \
DB_PATH=./data/server.db \
go run ./cmd
```

### Run without NATS (local/testing)

```bash
START_NATS=false go run ./cmd
```

Dispatches will be routed through an in-process channel dispatcher. Clients cannot receive or respond to activities, but the REST API and all other services function normally.

### Docker (example)

No official `Dockerfile` is provided. A minimal example:

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY . .
RUN go build -o /automation-server ./cmd

FROM alpine:3.20
COPY --from=builder /automation-server /usr/local/bin/automation-server
ENTRYPOINT ["/usr/local/bin/automation-server"]
```

```bash
docker build -t automation-server .
docker run -p 8080:8080 \
  -e NATS_URL=nats://host.docker.internal:4222 \
  -e DB_PATH=/data/server.db \
  -v automation-data:/data \
  automation-server
```

### Kubernetes readiness and liveness

The server exposes health probes at:

- `GET /health/liveness` — always returns `200 OK` if the process is running
- `GET /health/readiness` — returns `200 OK` only when both the SQLite database and NATS connection are healthy; returns `503` otherwise

```yaml
livenessProbe:
  httpGet:
    path: /health/liveness
    port: 8080
  initialDelaySeconds: 5
readinessProbe:
  httpGet:
    path: /health/readiness
    port: 8080
  initialDelaySeconds: 10
```

### Graceful shutdown

The server handles `SIGINT` and `SIGTERM`. On receipt, it:

1. Stops accepting new HTTP connections (with a `SHUTDOWN_TIMEOUT_MS` grace period)
2. Stops the trigger coordinator
3. Stops the result message dispatcher
4. Cancels all background goroutines
5. Closes the NATS connection and SQLite database

---

## Usage

### Registering clients

Clients self-register by publishing result messages to NATS. A client's record is created or updated in the `clients` table the first time a result message containing its `client_id` is received.

Clients carry metadata used for filter matching:

- `os` — operating system string (e.g. `"ubuntu-22.04"`)
- `labels` — arbitrary key/value map (e.g. `{"region": "us-east", "env": "production"}`)
- `inner_state` — mutable state the client can carry forward between runs

### Creating a workflow

```bash
curl -X POST http://localhost:8080/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Nightly package upgrade",
    "workflow_type": "upgrade",
    "activity": "upgrade_package",
    "params": {"package": "nginx"},
    "target_filter": "labels.env == \"production\" AND os CONTAINS \"ubuntu\"",
    "trigger": {
      "kind": "scheduled",
      "cron": "0 2 * * *"
    },
    "success_threshold": 80,
    "loop_threshold_ms": 10000,
    "timeout_ms": 60000
  }'
```

### Triggering a workflow manually

```bash
curl -X POST http://localhost:8080/workflows/{id}/trigger \
  -H "Content-Type: application/json" \
  -d '{"reason": "emergency patch"}'
```

### Checking run results

```bash
curl http://localhost:8080/runs/{run_id}/results
```

### Unbanning a client

```bash
curl -X PUT http://localhost:8080/bans/{client_id}/unban \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_type": "upgrade",
    "admin_id": "ops-team",
    "reason": "false positive, client behaviour verified"
  }'
```

---

## Filter Expression Language

Workflows target clients through a filter expression evaluated against each client's metadata. An empty filter matches all clients.

### Fields available

| Field | Type | Description |
|---|---|---|
| `id` or `client_id` | string | Client identifier |
| `os` | string | Client operating system |
| `active` | bool | Whether the client is active |
| `labels.<key>` | string / number | Any label key on the client |
| `inner_state.<key>` or `state.<key>` | any | Any inner state field from the client's last result |

### Operators

| Operator | Example |
|---|---|
| `==` | `os == "ubuntu-22.04"` |
| `!=` | `labels.env != "staging"` |
| `<`, `>`, `<=`, `>=` | `inner_state.disk_free_gb > 10` |
| `IN` | `labels.region IN ["us-east", "us-west"]` |
| `NOT_IN` | `labels.tier NOT_IN ["dev", "test"]` |
| `CONTAINS` | `os CONTAINS "ubuntu"` |
| `NOT_CONTAINS` | `os NOT_CONTAINS "windows"` |

### Logical operators

`AND`, `OR`, `NOT` — standard boolean logic with parentheses for grouping.

### Examples

```
labels.env == "production"

os CONTAINS "ubuntu" AND labels.region IN ["us-east", "eu-west"]

NOT labels.tier == "canary" AND inner_state.disk_free_gb > 5

(labels.env == "production" OR labels.env == "staging") AND os != "windows"
```

---

## Client Messaging Protocol

Clients communicate with the server over NATS subjects. The server dispatches activities and clients respond with results.

### Dispatch — server to client

Subject: `dispatch.<client_id>`

```json
{
  "run_id": "uuid",
  "wf_id": "uuid",
  "activity": "upgrade_package",
  "params": {
    "package": "nginx"
  }
}
```

### Result — client to server

Subject: `result.<client_id>` (matched by the server's `result.>` subscription)

```json
{
  "run_id": "uuid",
  "wf_id": "uuid",
  "client_id": "string",
  "status": "success",
  "inner_state": {
    "disk_free_gb": 42.5
  },
  "payload": {
    "installed_version": "1.26.0"
  },
  "error_msg": "",
  "received_at": "2026-05-13T02:00:05Z"
}
```

**Status values:** `success`, `fail`, `error`

- `success` — the activity completed as expected
- `fail` — the activity ran but the outcome was not successful (e.g. validation failure)
- `error` — the client encountered an unrecoverable error running the activity

The `inner_state` object is persisted and replaces the client's stored state, making it available in filter expressions for subsequent workflows.

---

## Safety Mechanisms

### Loop detection

If a client submits a result for a given workflow type and the **previous** result for the same client and workflow type arrived within `LOOP_THRESHOLD_MS`, the client is automatically banned from that workflow type. A ban record is written to the database with the triggering run ID and result as evidence, an alert is published, and the in-memory ban cache is updated immediately.

A banned client is excluded from future dispatches for the banned workflow type but is not blocked from other workflow types. Bans are permanent until lifted by an administrator via `PUT /bans/{client_id}/unban`.

### Circuit breaker

Each workflow has an associated circuit breaker with three states:

| State | Meaning |
|---|---|
| `closed` | Normal operation — dispatches proceed |
| `open` | Workflow is deactivated — no dispatches; cooldown in progress |
| `half_open` | Cooldown expired — next health evaluation determines whether to close or re-open |

The circuit breaker opens when the aggregated success percentage across the health window falls below `CIRCUIT_BREAKER_SUCCESS_THRESHOLD`. The workflow is deactivated automatically. After `CIRCUIT_BREAKER_COOLDOWN_MS`, the circuit moves to `half_open`. When the next health evaluation produces a success percentage at or above the threshold, the circuit closes and the workflow is reactivated.

### Health window

Health is tracked per workflow type using a sliding window of the most recent `HEALTH_WINDOW_SIZE` runs. The trend is calculated by comparing the average success rate of the second half of the window to the first half:

- **improving** — later half success rate > earlier half
- **degrading** — later half success rate < earlier half
- **stable** — no significant difference

---

## API Reference

All responses use `Content-Type: application/json`. Error responses use the following shape:

```json
{
  "code": "NOT_FOUND",
  "message": "workflow not found"
}
```

**Error codes:**

| Code | HTTP Status | Description |
|---|---|---|
| `NOT_FOUND` | 404 | Requested resource does not exist |
| `VALIDATION_ERROR` | 400 | Request body failed validation |
| `BAD_REQUEST` | 400 | Malformed JSON or missing required fields |
| `CONFLICT` | 409 | Operation cannot be performed in the current state (e.g. triggering an inactive workflow) |
| `INTERNAL` | 500 | Unexpected server error |

---

### Workflows

#### `POST /workflows`

Create a new workflow.

**Request body:**

```json
{
  "name": "string (required)",
  "description": "string",
  "workflow_type": "string (required) — logical grouping identifier",
  "activity": "reboot | install_package | upgrade_package | remove_package | apply_config | validate_config | run_script",
  "params": {},
  "target_filter": "string — filter expression; empty matches all clients",
  "trigger": {
    "kind": "scheduled | event | state_change | manual",
    "cron": "cron expression (when kind=scheduled)",
    "on_complete": "workflow_type to chain (when kind=event)",
    "condition": "string",
    "params": {}
  },
  "success_threshold": 80,
  "loop_threshold_ms": 5000,
  "timeout_ms": 30000,
  "enabled": true
}
```

**Response:** `201 Created` — the created `Workflow` object.

---

#### `GET /workflows`

List all workflows.

**Response:** `200 OK`
```json
{
  "items": [ /* Workflow objects */ ],
  "total": 3
}
```

---

#### `GET /workflows/{id}`

Get a single workflow by ID.

**Response:** `200 OK` — `Workflow` object, or `404`.

---

#### `PUT /workflows/{id}`

Update a workflow. All fields are optional — only provided fields are changed.

**Request body:**

```json
{
  "name": "string",
  "description": "string",
  "params": {},
  "target_filter": "string",
  "success_threshold": 90,
  "loop_threshold_ms": 10000,
  "timeout_ms": 60000,
  "enabled": false
}
```

**Response:** `200 OK` — updated `Workflow` object.

---

#### `DELETE /workflows/{id}`

Delete a workflow and all associated data.

**Response:** `204 No Content`.

---

#### `POST /workflows/{id}/trigger`

Manually trigger a workflow execution.

**Request body:**

```json
{
  "reason": "string — optional description shown in the run record"
}
```

**Response:** `200 OK` — the created `Run` object. Returns `409` if the workflow is inactive.

---

#### `POST /workflows/{id}/activate`

Activate a previously deactivated workflow.

**Response:** `200 OK` — updated `Workflow` object.

---

#### `POST /workflows/{id}/deactivate`

Deactivate a workflow. While deactivated, scheduled triggers are skipped and manual triggers return `409`.

**Response:** `200 OK` — updated `Workflow` object.

---

#### `GET /workflows/{id}/runs`

List recent runs for a workflow.

**Query parameters:**

| Parameter | Default | Description |
|---|---|---|
| `limit` | `50` | Maximum number of runs to return |

**Response:** `200 OK`
```json
{
  "items": [ /* Run objects */ ],
  "total": 12
}
```

---

### Clients

#### `GET /clients`

List all registered clients.

**Response:** `200 OK`
```json
{
  "items": [ /* Client objects */ ],
  "total": 47
}
```

---

#### `GET /clients/{id}`

Get a single client by ID.

**Response:** `200 OK` — `Client` object, or `404`.

**Client object:**
```json
{
  "client_id": "string",
  "os": "ubuntu-22.04",
  "labels": { "region": "us-east", "env": "production" },
  "inner_state": {},
  "active": true,
  "last_seen_at": "2026-05-13T02:00:00Z"
}
```

---

### Runs

#### `GET /runs/{id}`

Get a run by ID.

**Run object:**
```json
{
  "run_id": "uuid",
  "workflow_id": "uuid",
  "workflow_type": "string",
  "triggered_at": "2026-05-13T02:00:00Z",
  "dispatched_at": "2026-05-13T02:00:01Z",
  "state": "pending | in_progress | completed | failed",
  "reason": "scheduled",
  "participating_clients": ["client-a", "client-b"]
}
```

---

#### `GET /runs/{id}/results`

Get all results for a run.

**Response:** `200 OK`
```json
{
  "items": [
    {
      "run_id": "uuid",
      "client_id": "string",
      "workflow_id": "uuid",
      "workflow_type": "string",
      "status": "success | fail | error",
      "inner_state": {},
      "error_msg": "",
      "payload": {},
      "received_at": "2026-05-13T02:00:05Z"
    }
  ],
  "total": 2
}
```

---

### Health

#### `GET /health`

List aggregated health for all workflow types.

**Response:** `200 OK`
```json
{
  "items": [
    {
      "workflow_type": "upgrade",
      "runs_considered": 10,
      "window_size": 10,
      "success_percentage_avg": 92.5,
      "fail_percentage_avg": 5.0,
      "error_percentage_avg": 2.5,
      "trend": "improving | degrading | stable",
      "calculated_at": "2026-05-13T02:00:00Z"
    }
  ],
  "total": 1
}
```

---

#### `GET /health/{workflow_type}`

Get health for a specific workflow type.

**Response:** `200 OK` — single health object (see above), or `404`.

---

#### `GET /health/liveness`

Kubernetes liveness probe.

**Response:** `200 OK`
```json
{"status": "alive"}
```

---

#### `GET /health/readiness`

Kubernetes readiness probe. Returns `503` if the database or NATS connection is unavailable.

**Response:** `200 OK`
```json
{"status": "ready"}
```

**Response:** `503 Service Unavailable`
```json
{"db": true, "nats": false}
```

---

### Bans

#### `GET /bans`

List all active ban records.

**Response:** `200 OK`
```json
{
  "items": [
    {
      "id": 1,
      "client_id": "string",
      "workflow_type": "upgrade",
      "run_id_evidence": "uuid",
      "result_evidence": "...",
      "banned_at": "2026-05-13T01:55:00Z",
      "banned_until": null,
      "reason": "loop_detected | manual | admin_unban_failed",
      "banned_by": "system",
      "active": true
    }
  ],
  "total": 1
}
```

---

#### `GET /bans/{client_id}`

Get all bans for a specific client.

**Response:** `200 OK` — paginated ban list, or `404` if the client has no bans.

---

#### `PUT /bans/{client_id}/unban`

Remove bans for a client. An `admin_id` and `reason` are required for audit purposes.

**Request body:**

```json
{
  "workflow_type": "upgrade",
  "admin_id": "ops-team (required)",
  "reason": "verified safe (required)"
}
```

Omit `workflow_type` to unban the client from all workflow types.

**Response:** `200 OK`
```json
{"success": true, "client_id": "string"}
```

---

### Circuit Breaker

#### `GET /circuits`

List all circuit breaker states.

**Response:** `200 OK`
```json
{
  "items": [
    {
      "workflow_id": "uuid",
      "workflow_type": "upgrade",
      "state": "closed | open | half_open",
      "opened_at": null,
      "last_evaluated_at": "2026-05-13T02:00:00Z",
      "opened_reason": "",
      "evaluation_count": 42
    }
  ],
  "total": 1
}
```

---

#### `GET /circuits/{workflow_id}`

Get the circuit breaker state for a specific workflow.

**Response:** `200 OK` — single circuit state object, or `404`.

---

### System

#### `GET /status`

System-level status including uptime, goroutine count, and component health.

**Response:** `200 OK`
```json
{
  "uptime": 3600000000000,
  "uptime_seconds": 3600,
  "db_status": "ok",
  "nats_status": "ok",
  "started_at": "2026-05-13T01:00:00Z",
  "goroutines": 18
}
```

---

## Database Schema

The server manages its own SQLite schema. Tables are created automatically on first startup.

| Table | Description |
|---|---|
| `workflows` | Workflow definitions including trigger spec, activity, thresholds, and active state |
| `clients` | Registered client metadata (OS, labels, inner state) |
| `runs` | Execution records linking a workflow to the clients that participated |
| `runs_clients` | Junction table for run ↔ client relationships |
| `results` | Per-client activity results for each run |
| `bans` | Ban records with evidence, reason, and active flag |
| `run_health` | Per-run health metrics (success/fail/error/pending/banned counts) |
| `workflow_type_health` | Aggregated sliding-window health per workflow type |
| `circuit_breaker_states` | Circuit state (closed/open/half_open) per workflow |

The schema is located at `migrations/schema.sql`. It is applied once at startup using `CREATE TABLE IF NOT EXISTS` statements, so the file is safe to re-apply.

---

## Development

### Project layout

```
cmd/                  Entry point, config loading, dependency wiring
internal/
  core/
    domain/           Value objects, aggregate types, filter AST, error definitions
    ports/            Repository and dispatcher interfaces
    services/         All business logic
  adapters/
    alert/            Alert publisher (stdout)
    config/           Configuration adapters
    enforcement/      In-memory ban cache
    http/             chi router and HTTP handlers
    messaging/        NATS dispatcher, in-process event bus, channel dispatcher
    parser/           Additional parser implementations
    repository/       SQLite repository implementations
    trigger/          Cron evaluator
migrations/           SQL schema
tests/integration/    Integration test suites
```

### Running tests

```bash
# All tests
go test ./...

# Unit tests only
go test ./internal/...

# Integration tests (requires a writable temp directory)
go test ./tests/...

# With race detector
go test -race ./...
```

### Key dependencies

| Package | Purpose |
|---|---|
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/nats-io/nats.go` | NATS client |
| `github.com/nats-io/nats-server/v2` | Embedded NATS server (test use) |
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGo required) |
| `github.com/google/uuid` | UUID generation |
| `github.com/robfig/cron/v3` | Cron expression parsing |

### Adding a new activity type

1. Add the constant to `ActivityType` in `internal/core/domain/workflow.go`
2. Add it to the `IsValid()` switch statement in the same file
3. Update client implementations to handle the new activity

### Adding a new service

Follow the ports and adapters pattern:

1. Define the interface in `internal/core/ports/ports.go`
2. Implement business logic in `internal/core/services/`
3. Wire the new service in `cmd/main.go`
4. Add any required adapter (repository, external client) in `internal/adapters/`
