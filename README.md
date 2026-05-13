# super-duper-bassoon

A hub-and-spoke workflow automation engine. The **server** orchestrates activities across a fleet of remote devices; the **super-client** is a simulation component that spawns a pool of inner clients connected over NATS to validate the server's safety mechanisms.

---

## System Overview

```
┌─────────────────────────────────────────────────────────┐
│                        Server                           │
│  (orchestration, safety logic, SQLite, REST API)        │
└──────────────┬──────────────────────────────────────────┘
               │  NATS
    ┌──────────┴──────────┐
    │     NATS Broker     │
    └──────────┬──────────┘
               │
    ┌──────────┴──────────┐
    │     super-client    │
    │  ┌───┐ ┌───┐ ┌───┐  │
    │  │C-1│ │C-2│ │C-N│  │  inner clients
    │  └───┘ └───┘ └───┘  │
    └─────────────────────┘
```

The server dispatches workflow activities to individual inner clients over NATS. Each client executes the activity against its simulated state, then publishes a result back to the server. The chaos layer introduces realistic failure rates that stress-test the server's safety mechanisms.

---

## Table of Contents

**Server**
- [Server Overview](#server-overview)
- [Server Architecture](#server-architecture)
- [Services](#services)
- [Server Configuration](#server-configuration)
- [Server Deployment](#server-deployment)
- [Server Usage](#server-usage)
- [Filter Expression Language](#filter-expression-language)
- [Client Messaging Protocol](#client-messaging-protocol)
- [Safety Mechanisms](#safety-mechanisms)
- [API Reference](#api-reference)
- [Database Schema](#database-schema)
- [Server Development](#server-development)

**super-client**
- [super-client Overview](#super-client-overview)
- [super-client Architecture](#super-client-architecture)
- [super-client Configuration](#super-client-configuration)
- [super-client Deployment](#super-client-deployment)
- [Chaos Simulation](#chaos-simulation)
- [super-client Testing](#super-client-testing)
- [super-client Project Structure](#super-client-project-structure)

---

# Automation Engine Server

## Server Overview

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

## Server Architecture

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

## Server Configuration

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

## Server Deployment

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

## Server Usage

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
  "items": [],
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

#### `POST /workflows/{id}/activate` / `POST /workflows/{id}/deactivate`

Activate or deactivate a workflow.

**Response:** `200 OK` — updated `Workflow` object.

---

#### `GET /workflows/{id}/runs`

List recent runs for a workflow.

**Query parameters:** `limit` (default 50)

**Response:** `200 OK` — paginated run list.

---

### Clients

#### `GET /clients`

List all registered clients.

---

#### `GET /clients/{id}`

Get a single client by ID.

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

#### `GET /runs/{id}` / `GET /runs/{id}/results`

Get a run or all its results.

---

### Health

#### `GET /health` / `GET /health/{workflow_type}`

Aggregated health for all workflow types or a specific one.

#### `GET /health/liveness` / `GET /health/readiness`

Kubernetes probes. Readiness returns `503` if database or NATS is unavailable.

---

### Bans

#### `GET /bans` / `GET /bans/{client_id}`

List all bans or bans for a specific client.

#### `PUT /bans/{client_id}/unban`

Remove bans for a client. Requires `admin_id` and `reason`.

---

### Circuit Breaker

#### `GET /circuits` / `GET /circuits/{workflow_id}`

List circuit breaker states or get a specific one.

---

### System

#### `GET /status`

System-level status including uptime, goroutine count, and component health.

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

## Server Development

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
    enforcement/      In-memory ban cache
    http/             chi router and HTTP handlers
    messaging/        NATS dispatcher, in-process event bus, channel dispatcher
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

---

# super-client

## super-client Overview

`super-client` is a toy simulation component of the **super-duper-bassoon** workflow automation engine. It spawns a pool of inner clients that communicate with the automation server over NATS, simulate realistic device behavior, and inject configurable chaos to validate the server's safety mechanisms (loop detection, permanent banning, circuit breaking).

---

## super-client Architecture

### Hexagonal Layout

The codebase follows a strict hexagonal (ports & adapters) architecture. Dependencies flow inward only: adapters depend on ports, ports are defined by the core, and the core domain has no external imports.

```
cmd/super-client/          ← binary entry point & config
internal/
├── app/                   ← dependency wiring & lifecycle
├── core/
│   ├── domain/            ← pure business logic (no I/O)
│   ├── ports/             ← abstract interfaces
│   └── services/          ← orchestration (uses ports)
└── adapters/
    ├── messaging/         ← NATS implementation of MessageBroker
    ├── storage/           ← in-memory implementation of StateStore
    ├── activity/          ← StandardExecutor + ChaosExecutor
    └── clock/             ← SystemClock + MockClock
```

### Domain Layer

`internal/core/domain/` contains pure Go — no imports beyond the standard library.

**Models:**

| Type | Fields |
|------|--------|
| `InnerClient` | `client_id`, `state ClientState` |
| `ClientState` | `packages map[string]string`, `config_version int`, `power_state PowerState`, `is_crippled bool`, `cripple_mode string`, `cripple_recovery_attempts int` |
| `Activity` | `type string`, `params map[string]any` |
| `ActivityResult` | `status string`, `payload map[string]any`, `error_msg string` |
| `DispatchMessage` | `run_id`, `wf_id`, `client_id`, `activity Activity` |
| `ResultMessage` | `run_id`, `wf_id`, `client_id`, `status`, `inner_state`, `error_msg`, `payload` |

**Chaos functions** (pure, deterministic-by-seed):

| Function | Behaviour |
|----------|-----------|
| `ShouldActivityFail()` | Returns true ~10% of the time |
| `ShouldCrippleClient(didFail)` | Returns true ~3% of the time when the activity failed |
| `SelectCrippleMode()` | Randomly picks `fail_package_ops`, `fail_config`, or `silent` |
| `ShouldDriftState()` | Returns true ~7.5% of the time |
| `ApplyDrift(state)` | Mutates state independently of any activity |
| `IsCrippledForActivity(mode, activity)` | Whether the current cripple mode blocks the given activity type |

### Ports Layer

`internal/core/ports/` defines the abstract interfaces that services depend on.

```go
type MessageBroker interface {
    SubscribeDispatch(ctx context.Context) (<-chan DispatchMessage, error)
    PublishResult(ctx context.Context, result ResultMessage) error
    Close(ctx context.Context) error
}

type StateStore interface {
    GetState(ctx context.Context, clientID string) (*ClientState, error)
    UpdateState(ctx context.Context, clientID string, state *ClientState) error
    GetAllStates(ctx context.Context) (map[string]*ClientState, error)
}

type ActivityExecutor interface {
    Execute(ctx context.Context, clientID string, activity Activity, state ClientState) (*ClientState, *ActivityResult, error)
}

type Clock interface {
    Now() time.Time
    Sleep(d time.Duration)
}
```

### Services Layer

**ClientPoolManager** — owns the main dispatch loop. Initializes N inner clients and routes each dispatch message until context cancellation.

**DispatchHandler** — validates and routes a single dispatch message against the executor.

**StateOrchestrator** — applies post-execution side effects (chaos, drift, cripple recovery).

**ResultCollector** — batches and publishes results to NATS.

### Adapters Layer

| Adapter | Interface | Notes |
|---------|-----------|-------|
| `NATSBroker` | `MessageBroker` | Subscribes on `super-client.<client-id>.dispatch`; publishes to `server.results` |
| `MemoryStore` | `StateStore` | `sync.RWMutex`-protected map; non-persistent across restarts |
| `StandardExecutor` | `ActivityExecutor` | Applies activity semantics: package installs, config versions, power state transitions |
| `ChaosExecutor` | `ActivityExecutor` | Wraps `StandardExecutor`; injects chaos decisions before delegating |
| `SystemClock` | `Clock` | Wraps `time.Now()` and `time.Sleep()` |
| `MockClock` | `Clock` | Deterministic time for tests |

### Application Layer

`internal/app/app.go` wires every component together:

```go
app, err := app.New(cfg, logger)  // connects to NATS, creates all services
app.Start(ctx, poolSize)          // blocks — runs until ctx is cancelled
app.Shutdown(ctx)                 // flushes results, closes NATS
```

### Message Flow

```
NATS "super-client.<id>.dispatch"
        │
        ▼
ClientPoolManager.Run()
        │
        ├─► DispatchHandler.Handle()
        │         └─► ChaosExecutor.Execute() → StandardExecutor.Execute()
        │
        ├─► StateOrchestrator.ApplyActivityResult()
        ├─► StateOrchestrator.ApplyChaosAfterFailure() / ApplyDriftIfNeeded()
        │
        └─► ResultCollector.Collect()  ──► FlushResults()
                                                │
                                                ▼
                                    NATS "server.results"
```

---

## super-client Configuration

All configuration is through environment variables.

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS broker connection URL |
| `POOL_SIZE` | `5` | Number of inner clients to spawn |
| `CLIENT_PREFIX` | `client` | Prefix used when generating client IDs (e.g. `client-0001`) |
| `LOG_LEVEL` | `info` | Logging verbosity: `debug`, `info`, `warn`, or `error` |

---

## super-client Deployment

### Bare metal / VM

```bash
export NATS_URL=nats://broker.internal:4222
export POOL_SIZE=20
export CLIENT_PREFIX=edge
./super-client
```

### Docker (example)

```dockerfile
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /super-client ./cmd/super-client

FROM alpine:3.21
COPY --from=build /super-client /super-client
ENTRYPOINT ["/super-client"]
```

### Multiple instances

Each `super-client` process is independent. Use distinct `CLIENT_PREFIX` values per instance to avoid ID collisions:

```bash
CLIENT_PREFIX=east POOL_SIZE=5 ./super-client &
CLIENT_PREFIX=west POOL_SIZE=5 ./super-client &
```

### Graceful shutdown

The process handles `SIGTERM` and `SIGINT`. On receipt it cancels the dispatch loop, flushes all pending results to NATS, then exits cleanly.

---

## Chaos Simulation

Chaos is injected by `ChaosExecutor` wrapping `StandardExecutor`. It is always active and not configurable at runtime.

| Event | Probability | Behaviour |
|-------|-------------|-----------|
| Activity failure | ~10% | Returns `status=fail` regardless of activity type |
| Client crippling | ~3% (given failure) | Sets `is_crippled=true` and assigns a cripple mode |
| Spontaneous drift | ~7.5% per dispatch | Modifies client state independently of the activity |

**Cripple modes:**

| Mode | Effect |
|------|--------|
| `fail_package_ops` | All package install/upgrade/remove activities fail |
| `fail_config` | All apply_config and validate_config activities fail |
| `silent` | All activities appear to succeed but no state change is committed |

Recovery: a successful `reboot` activity clears `is_crippled` and `cripple_mode`.

`silent` mode is particularly insidious — the client reports success while lying about its state. This surfaces scenarios where the server's health metrics look fine but the client is actually broken.

---

## super-client Testing

```bash
go test ./...
go test -race ./...
go test ./internal/core/services/...
```

Key test files:

| File | What it tests |
|------|---------------|
| `internal/core/domain/chaos_test.go` | Chaos probability functions |
| `internal/core/domain/activity_test.go` | Activity state transitions |
| `internal/core/services/dispatch_test.go` | Dispatch validation and routing |
| `internal/core/services/state_orchestrator_test.go` | Post-execution state mutations |
| `internal/core/services/result_collector_test.go` | Result batching and flushing |
| `internal/adapters/messaging/nats_broker_test.go` | NATS pub/sub integration |
| `internal/adapters/storage/memory_store_test.go` | Concurrent state store access |

---

## super-client Project Structure

```
cmd/super-client/
├── main.go          # entry point, signal handling, logger init
└── config.go        # env var parsing
internal/
├── app/
│   └── app.go       # wires all dependencies, exposes Start/Shutdown
├── core/
│   ├── domain/
│   │   ├── inner_client.go  # InnerClient, ClientState, PowerState
│   │   ├── activity.go      # Activity types, ActivityResult, execution semantics
│   │   ├── chaos.go         # pure chaos simulation functions
│   │   └── client_errors.go # super-client domain error types
│   ├── ports/
│   └── services/
│       ├── client_pool.go         # ClientPoolManager
│       ├── dispatch.go            # DispatchHandler
│       ├── state_orchestrator.go  # StateOrchestrator
│       └── result_collector.go    # ResultCollector
└── adapters/
    ├── messaging/  # NATS MessageBroker
    ├── storage/    # in-memory StateStore
    ├── activity/   # StandardExecutor + ChaosExecutor
    └── clock/      # SystemClock + MockClock
go.mod
go.sum
README.md
```
