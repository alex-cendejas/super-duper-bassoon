# super-client

A toy simulation component of the **super-duper-bassoon** workflow automation engine. `super-client` spawns a pool of inner clients that communicate with the automation server over NATS, simulate realistic device behavior, and inject configurable chaos to validate the server's safety mechanisms (loop detection, permanent banning, circuit breaking).

---

## Table of Contents

- [System Context](#system-context)
- [Architecture](#architecture)
  - [Hexagonal Layout](#hexagonal-layout)
  - [Domain Layer](#domain-layer)
  - [Ports Layer](#ports-layer)
  - [Services Layer](#services-layer)
  - [Adapters Layer](#adapters-layer)
  - [Application Layer](#application-layer)
  - [Message Flow](#message-flow)
- [Building](#building)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Usage](#usage)
  - [Message Protocol](#message-protocol)
  - [Client IDs](#client-ids)
  - [Activities](#activities)
  - [Client State](#client-state)
- [Chaos Simulation](#chaos-simulation)
- [Testing](#testing)
- [Project Structure](#project-structure)

---

## System Context

`super-client` is one binary in a larger hub-and-spoke system:

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
    │     super-client    │  ← this binary
    │  ┌───┐ ┌───┐ ┌───┐  │
    │  │C-1│ │C-2│ │C-N│  │  inner clients
    │  └───┘ └───┘ └───┘  │
    └─────────────────────┘
```

The server dispatches workflow activities to individual inner clients over NATS. Each client executes the activity against its simulated state, then publishes a result back to the server. The chaos layer introduces realistic failure rates that stress-test the server's safety mechanisms.

---

## Architecture

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

`internal/core/ports/` defines the four abstract interfaces that services depend on.

```go
// MessageBroker — asynchronous messaging
type MessageBroker interface {
    SubscribeDispatch(ctx context.Context) (<-chan DispatchMessage, error)
    PublishResult(ctx context.Context, result ResultMessage) error
    Close(ctx context.Context) error
}

// StateStore — client state persistence
type StateStore interface {
    GetState(ctx context.Context, clientID string) (*ClientState, error)
    UpdateState(ctx context.Context, clientID string, state *ClientState) error
    GetAllStates(ctx context.Context) (map[string]*ClientState, error)
}

// ActivityExecutor — activity execution against client state
type ActivityExecutor interface {
    Execute(ctx context.Context, clientID string, activity Activity, state ClientState) (*ClientState, *ActivityResult, error)
}

// Clock — injectable time source
type Clock interface {
    Now() time.Time
    Sleep(d time.Duration)
}
```

### Services Layer

`internal/core/services/` contains four services that compose ports to implement business logic.

**ClientPoolManager** — owns the main dispatch loop.
- `Initialize(ctx, poolSize)` — creates N inner clients with default state.
- `Run(ctx)` — subscribes to NATS dispatch channel, routes each message, flushes results; blocks until `ctx` is cancelled.
- `Shutdown(ctx)` — drains pending results and closes the broker.

**DispatchHandler** — validates and routes a single dispatch message.
- `Handle(ctx, dispatch)` — confirms the target client exists, calls the executor, returns the result.
- `ValidateDispatch(dispatch)` — checks `run_id`, `wf_id`, and activity type are non-empty and recognized.

**StateOrchestrator** — applies post-execution side effects.
- `ApplyActivityResult(ctx, clientID, activity, result)` — updates the state store based on what the activity did.
- `ApplyChaosAfterFailure(ctx, clientID)` — potentially cripples the client.
- `ApplyDriftIfNeeded(ctx, clientID)` — potentially applies spontaneous state drift.
- `CheckRecoveryFromCripple(ctx, clientID, activity)` — a successful reboot clears the crippled flag.

**ResultCollector** — batches and publishes results.
- `Collect(runID, wfID, clientID, result)` — enqueues a result in memory.
- `FlushResults(ctx)` — publishes all enqueued results to NATS and clears the queue.
- `GetPendingCount()` — returns the current queue depth.

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

`internal/app/app.go` wires every component together and exposes a two-method lifecycle:

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
        │         │
        │         ├─► StateStore.GetState()
        │         └─► ChaosExecutor.Execute()
        │                   └─► StandardExecutor.Execute()
        │
        ├─► StateOrchestrator.ApplyActivityResult()
        │         └─► StateStore.UpdateState()
        │
        ├─► StateOrchestrator.ApplyChaosAfterFailure() / ApplyDriftIfNeeded()
        │
        └─► ResultCollector.Collect()  ──► FlushResults()
                                                │
                                                ▼
                                    NATS "server.results"
```

---

## Building

Requires Go 1.25+.

```bash
go build -o super-client ./cmd/super-client
```

Run tests:

```bash
go test ./...
```

---

## Configuration

All configuration is through environment variables. There are no config files.

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS broker connection URL |
| `POOL_SIZE` | `5` | Number of inner clients to spawn. Must be a positive integer. |
| `CLIENT_PREFIX` | `client` | Prefix used when generating client IDs (e.g. `client-0001`) |
| `LOG_LEVEL` | `info` | Logging verbosity: `debug`, `info`, `warn`, or `error` |

Logs are written to stdout as JSON (via `log/slog`).

---

## Deployment

### Bare metal / VM

```bash
export NATS_URL=nats://broker.internal:4222
export POOL_SIZE=20
export CLIENT_PREFIX=edge
export LOG_LEVEL=info

./super-client
```

### Docker (example)

No `Dockerfile` is committed — this is a PoC. A minimal example:

```dockerfile
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /super-client ./cmd/super-client

FROM alpine:3.21
COPY --from=build /super-client /super-client
ENTRYPOINT ["/super-client"]
```

```bash
docker build -t super-client .
docker run --rm \
  -e NATS_URL=nats://broker:4222 \
  -e POOL_SIZE=10 \
  super-client
```

### Multiple instances

Each `super-client` process is independent. Run as many as needed; each will register its own pool of clients. Make sure `CLIENT_PREFIX` differs per instance to avoid ID collisions:

```bash
CLIENT_PREFIX=east POOL_SIZE=5 ./super-client &
CLIENT_PREFIX=west POOL_SIZE=5 ./super-client &
```

### Graceful shutdown

The process handles `SIGTERM` and `SIGINT`. On receipt it cancels the dispatch loop, flushes all pending results to NATS, then exits cleanly.

---

## Usage

### Message Protocol

Communication with the server is JSON over NATS pub/sub.

**Dispatch** (server → client)

Subject: `super-client.<client-id>.dispatch`

```json
{
  "run_id":   "run-abc123",
  "wf_id":    "wf-upgrade-packages",
  "client_id": "client-0001",
  "activity": {
    "type":   "install_package",
    "params": { "name": "nginx", "version": "1.26.0" }
  }
}
```

**Result** (client → server)

Subject: `server.results`

```json
{
  "run_id":    "run-abc123",
  "wf_id":     "wf-upgrade-packages",
  "client_id": "client-0001",
  "status":    "success",
  "inner_state": {
    "packages":       { "nginx": "1.26.0" },
    "config_version": 3,
    "power_state":    "on",
    "is_crippled":    false
  },
  "error_msg": "",
  "payload":   {}
}
```

`status` is one of `success`, `fail`, or `error`.

Results with the same `(client_id, wf_id, run_id)` triple are idempotent — duplicate publishes are safe. Malformed dispatch messages are silently dropped.

### Client IDs

Client IDs are generated at startup as `<CLIENT_PREFIX>-<zero-padded-index>`:

```
client-0001
client-0002
...
client-NNNN
```

### Activities

| Activity | Params | State Effect |
|----------|--------|-------------|
| `reboot` | — | Sets `power_state` to `restarting` then `on`; clears `is_crippled` |
| `install_package` | `name`, `version` | Adds entry to `packages` map (atomic: all-or-nothing) |
| `upgrade_package` | `name`, `version` | Updates entry in `packages` map |
| `remove_package` | `name` | Removes entry from `packages` map |
| `apply_config` | `version` | Sets `config_version` |
| `validate_config` | `expected_version` | Compares `config_version` to `expected_version`; no state change |
| `run_script` | `script`, `args` | No state change; result payload includes `exit_code`, `stdout`, `stderr` |

Package operations are atomic: if any step fails, no state change is committed.

### Client State

Each inner client holds an independent in-memory state:

```go
type ClientState struct {
    Packages               map[string]string // package → version
    ConfigVersion          int
    PowerState             PowerState        // "on" | "off" | "restarting"
    IsCrippled             bool
    CrippleMode            string            // "fail_package_ops" | "fail_config" | "silent" | ""
    CrippleRecoveryAttempts int
}
```

State is initialized with an empty package map, `config_version=0`, `power_state=on`, and `is_crippled=false`. It is held in memory only and resets on process restart.

---

## Chaos Simulation

Chaos is injected by `ChaosExecutor` wrapping `StandardExecutor`. It is always active and not configurable at runtime — the rates are baked into the domain logic.

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

`silent` mode is particularly insidious — the client reports success while lying about its state. This is intentional: it surfaces scenarios where the server's health metrics look fine but the client is actually broken.

---

## Testing

The project has ~2,200 lines of test code covering every layer. Tests use manual mocks (no mocking library) defined in `internal/core/services/mocks_test.go`.

```bash
# run all tests
go test ./...

# run with race detector
go test -race ./...

# run a specific package
go test ./internal/core/services/...

# verbose output
go test -v ./internal/core/domain/...
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

## Project Structure

```
super-client/
├── cmd/
│   └── super-client/
│       ├── main.go          # entry point, signal handling, logger init
│       └── config.go        # env var parsing
├── internal/
│   ├── app/
│   │   └── app.go           # wires all dependencies, exposes Start/Shutdown
│   ├── core/
│   │   ├── domain/
│   │   │   ├── client.go    # InnerClient, ClientState, PowerState
│   │   │   ├── activity.go  # Activity types, ActivityResult, execution semantics
│   │   │   ├── chaos.go     # pure chaos simulation functions
│   │   │   └── errors.go    # domain error types
│   │   ├── ports/
│   │   │   ├── messaging.go # MessageBroker interface
│   │   │   ├── state.go     # StateStore interface
│   │   │   ├── activity.go  # ActivityExecutor interface
│   │   │   └── clock.go     # Clock interface
│   │   └── services/
│   │       ├── client_pool.go         # ClientPoolManager
│   │       ├── dispatch.go            # DispatchHandler
│   │       ├── state_orchestrator.go  # StateOrchestrator
│   │       └── result_collector.go    # ResultCollector
│   └── adapters/
│       ├── messaging/
│       │   └── nats_broker.go   # NATS implementation of MessageBroker
│       ├── storage/
│       │   └── memory_store.go  # in-memory implementation of StateStore
│       ├── activity/
│       │   ├── executor.go      # StandardExecutor
│       │   └── chaos_executor.go # ChaosExecutor (wraps StandardExecutor)
│       └── clock/
│           ├── system_clock.go  # real time
│           └── mock_clock.go    # deterministic time for tests
├── go.mod
├── go.sum
└── README.md
```
