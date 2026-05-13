# Hexagonal Architecture Plan: Super-Client (Toy Simulation)

## 1. Core/Domain Layer

The domain layer captures pure business logic without external dependencies.

### Key Domain Models

**InnerClient State:**
- `client_id` (unique identifier)
- `state: ClientState`
  - `packages: map[string]string` (package → version)
  - `config_version: int`
  - `power_state: PowerState` (on/off/restarting)
  - `is_crippled: bool`
  - `cripple_mode: string` ("fail_package_ops" | "fail_config" | "silent" | none)
  - `cripple_recovery_attempts: int`

**Activity Execution Model:**
- Activity types: `reboot`, `install_package`, `upgrade_package`, `remove_package`, `apply_config`, `validate_config`, `run_script`
- Per-activity result: `status` (success|fail|error), `payload` (activity-specific), `error_msg`

**Chaos Simulation Logic** (pure functions):
- `ShouldActivityFail()` → bool (10% base failure rate)
- `ShouldCrippleClient(didFail bool)` → bool (3% given failure)
- `SelectCrippleMode()` → CrippleMode (randomly choose behavior)
- `ShouldDriftState()` → bool (5-10% spontaneous change)
- `ApplyDrift(state ClientState)` → ClientState (modify state independently)

**Activity Semantics** (state transition rules):
- `ExecuteActivity(activity Activity, state ClientState) → (newState ClientState, result ActivityResult, error)`
  - Package ops: atomic (all succeed or all fail)
  - Reboot: resets is_crippled, clears cripple_mode
  - Config validate: compare current vs expected
  - Script: execute and capture exit code/output

**Domain Errors:**
- `ErrClientNotFound`
- `ErrInvalidActivity`
- `ErrCrippledClient`
- `ErrStateConflict`

### File Structure

```
internal/core/domain/
├── client.go              # InnerClient, ClientState, PowerState models
├── client_test.go
├── activity.go            # Activity types, ActivityResult, semantics
├── activity_test.go
├── chaos.go               # Chaos simulation logic (pure functions)
├── chaos_test.go
└── errors.go              # Domain error types
```

---

## 2. Core/Ports Layer

Generic interfaces for driving/driven adapters (no NATS, no specific storage).

### Driven Adapter Ports (Core → External)

**MessageBroker** - Abstract messaging (NATS, gRPC, etc.)
```go
type MessageBroker interface {
    // Subscribe to dispatch messages for this super-client pool
    SubscribeDispatch(ctx context.Context) (<-chan DispatchMessage, error)
    
    // Publish result message back to server
    PublishResult(ctx context.Context, result ResultMessage) error
    
    // Close connection
    Close(ctx context.Context) error
}
```

**StateStore** - Abstract state persistence (in-memory, SQLite, etc.)
```go
type StateStore interface {
    // Get current state of a client
    GetState(ctx context.Context, clientID string) (*ClientState, error)
    
    // Update state atomically
    UpdateState(ctx context.Context, clientID string, state *ClientState) error
    
    // Batch get for pool operations
    GetAllStates(ctx context.Context) (map[string]*ClientState, error)
}
```

**ActivityExecutor** - Abstract activity execution
```go
type ActivityExecutor interface {
    // Execute a single activity on a client's state
    Execute(ctx context.Context, clientID string, activity Activity, state ClientState) (*ClientState, *ActivityResult, error)
}
```

**Clock** - Time provider (for testing, chaos timing)
```go
type Clock interface {
    Now() time.Time
    Sleep(d time.Duration)
}
```

### File Structure

```
internal/core/ports/
├── messaging.go           # MessageBroker interface
├── state.go               # StateStore interface
├── activity.go            # ActivityExecutor interface
└── clock.go               # Clock interface
```

---

## 3. Core/Services Layer

Services orchestrate domain logic using ports.

### ClientPoolManager
- **Responsibility:** Manage lifecycle of all inner clients in this super-client
- **Methods:**
  - `Initialize(ctx, poolSize int) error` - create N clients with initial state
  - `HandleDispatch(ctx, dispatch DispatchMessage) error` - route dispatch to target client
  - `PublishResults(ctx) error` - batch-publish accumulated results
  - `GetClientState(ctx, clientID) (*ClientState, error)` - query state
  - `Shutdown(ctx) error` - graceful shutdown
- **Dependencies:** StateStore, ActivityExecutor, MessageBroker, Clock

### DispatchHandler
- **Responsibility:** Process incoming dispatch messages, validate, route to executor
- **Methods:**
  - `Handle(ctx, dispatch DispatchMessage) (*ActivityResult, error)` - validate dispatch, check client exists, return result immediately
  - `ValidateDispatch(dispatch DispatchMessage) error` - check run_id, wf_id, activity type
- **Dependencies:** StateStore (read), ActivityExecutor

### StateOrchestrator
- **Responsibility:** Coordinate state mutations, apply chaos, detect crippling recovery
- **Methods:**
  - `ApplyActivityResult(ctx, clientID, activity Activity, result ActivityResult) error` - update state based on activity result
  - `ApplyChaosAfterFailure(ctx, clientID) error` - check if should cripple, apply if yes
  - `ApplyDriftIfNeeded(ctx, clientID) error` - spontaneous state change
  - `CheckRecoveryFromCripple(ctx, clientID, activity Activity) bool` - did reboot succeed?
- **Dependencies:** StateStore, Clock, domain chaos logic

### ResultCollector
- **Responsibility:** Accumulate results and publish to server
- **Methods:**
  - `Collect(runID, wfID, clientID, result ActivityResult) error` - queue result
  - `FlushResults(ctx) error` - batch-publish all queued results
  - `GetPendingCount() int` - metrics
- **Dependencies:** MessageBroker

### File Structure

```
internal/core/services/
├── client_pool.go         # ClientPoolManager
├── client_pool_test.go
├── dispatch.go            # DispatchHandler
├── dispatch_test.go
├── state_orchestrator.go  # StateOrchestrator
├── state_orchestrator_test.go
├── result_collector.go    # ResultCollector
└── result_collector_test.go
```

---

## 4. Adapters Layer

Concrete implementations of ports.

### Message Broker Adapter

**NATS Implementation** (`nats_broker.go`)
- Subscribe to subject: `super-client.{client-id}.dispatch`
- Publish results to: `server.results`
- Handle NATS connection lifecycle
- Implement reconnection logic (optional, or fail fast)

### State Store Adapters

**In-Memory Store** (`memory_store.go`) - for PoC
- `sync.Map` or `sync.RWMutex` + map for client states
- Atomic updates via read-modify-write under lock

**SQLite Store** (`sqlite_store.go`) - optional for persistence
- Single table: `client_states` (client_id, state_json, updated_at)
- Use transactions for atomic updates

### Activity Executor Adapters

**Standard Executor** (`executor.go`)
- Route activity type to handler function
- Apply chaos (10% fail chance via domain logic)
- Apply state mutations (packages, config, power)
- Return result with status/payload/error

**Chaos-Aware Executor** (`chaos_executor.go`) - optional wrapper
- Wraps standard executor
- Injects chaos decisions
- Logs chaos events for debugging

### File Structure

```
internal/adapters/
├── messaging/
│   ├── nats_broker.go
│   └── nats_broker_test.go
├── storage/
│   ├── memory_store.go
│   ├── memory_store_test.go
│   ├── sqlite_store.go        # optional
│   └── sqlite_store_test.go    # optional
├── activity/
│   ├── executor.go
│   ├── executor_test.go
│   └── chaos_executor.go       # optional
└── clock/
    ├── system_clock.go
    ├── mock_clock.go
    └── mock_clock_test.go
```

---

## 5. Binary/Deployment Layer

CLI, configuration, instantiation, lifecycle.

### Configuration
- **Source:** Environment variables only
- **Variables:**
  - `NATS_URL` (default: nats://localhost:4222)
  - `POOL_SIZE` (number of inner clients to spawn)
  - `CLIENT_PREFIX` (prefix for client IDs)
  - `STATE_STORAGE` (memory|sqlite)
  - `SQLITE_PATH` (if sqlite)
  - `LOG_LEVEL` (debug|info|warn|error)

### Main Binary
- Parse config from env vars
- Instantiate adapters (NATS, StateStore, Clock)
- Create services (ClientPoolManager, DispatchHandler, StateOrchestrator, ResultCollector)
- Wire together via dependency injection
- Start ClientPoolManager
- Run signal handler (graceful shutdown on SIGTERM/SIGINT)

### Lifecycle Management
- Initialize client pool on startup
- Long-running dispatch loop: receive → handle → collect
- Periodic result flush (e.g., every 5s or on batch threshold)
- Graceful shutdown: drain pending results, close NATS, exit

### File Structure

```
cmd/super-client/
├── main.go                # Entry point, signal handling
├── config.go              # Config parsing from env
└── wire.go                # Dependency injection (or use manual wiring)

internal/
└── app/
    ├── app.go             # Application orchestrator (all services together)
    └── app_test.go
```

---

## Implementation Steps

### Phase 1: Core Domain + In-Memory Adapters
1. Implement domain models (Client, Activity, Result, State)
2. Implement chaos simulation logic (pure functions)
3. Implement activity semantics (state transitions)
4. Implement in-memory StateStore
5. Implement NATS MessageBroker adapter
6. Implement standard ActivityExecutor
7. Implement services (ClientPoolManager, DispatchHandler, StateOrchestrator, ResultCollector)
8. Wire in main.go, test with integration tests

### Phase 2: Polish + Optional SQLite
1. Add SQLite StateStore adapter
2. Add metrics/observability
3. Add structured logging
4. Integration tests with chaos scenarios
5. Validate loop detection (server side), circuit breaker (server side)

### Phase 3: Optional Enhancements
1. Graceful connection handling (reconnect logic)
2. Persistence of pending results
3. Metrics export (Prometheus)

---

## Key Design Decisions

1. **No external dependencies in domain:** Chaos logic, activity semantics, and state models are pure. Easy to test, understand, and modify.

2. **Generic ports:** MessageBroker is abstraction over NATS; StateStore is abstraction over storage. Easy to swap implementations for testing.

3. **Service layer owns orchestration:** ClientPoolManager coordinates everything; DispatchHandler validates and routes; StateOrchestrator manages chaos side effects.

4. **Result batching:** ResultCollector accumulates results and flushes periodically, reducing NATS overhead.

5. **In-memory default:** PoC starts with in-memory storage; SQLite adapter is optional for persistence later.

6. **Clock injection:** Enables deterministic testing of chaos timing and drift.

---

## File Structure Summary

```
super-client/
├── cmd/
│   └── super-client/
│       ├── main.go
│       └── config.go
├── internal/
│   ├── adapters/
│   │   ├── messaging/
│   │   │   ├── nats_broker.go
│   │   │   └── nats_broker_test.go
│   │   ├── storage/
│   │   │   ├── memory_store.go
│   │   │   ├── memory_store_test.go
│   │   │   ├── sqlite_store.go
│   │   │   └── sqlite_store_test.go
│   │   ├── activity/
│   │   │   ├── executor.go
│   │   │   ├── executor_test.go
│   │   │   └── chaos_executor.go
│   │   └── clock/
│   │       ├── system_clock.go
│   │       ├── mock_clock.go
│   │       └── mock_clock_test.go
│   ├── app/
│   │   ├── app.go
│   │   └── app_test.go
│   └── core/
│       ├── domain/
│       │   ├── client.go
│       │   ├── client_test.go
│       │   ├── activity.go
│       │   ├── activity_test.go
│       │   ├── chaos.go
│       │   ├── chaos_test.go
│       │   └── errors.go
│       ├── ports/
│       │   ├── messaging.go
│       │   ├── state.go
│       │   ├── activity.go
│       │   └── clock.go
│       └── services/
│           ├── client_pool.go
│           ├── client_pool_test.go
│           ├── dispatch.go
│           ├── dispatch_test.go
│           ├── state_orchestrator.go
│           ├── state_orchestrator_test.go
│           ├── result_collector.go
│           └── result_collector_test.go
├── go.mod
├── go.sum
└── README.md
```

This architecture ensures **separation of concerns**, **testability**, **and flexibility** to evolve adapters without touching domain logic or services.
