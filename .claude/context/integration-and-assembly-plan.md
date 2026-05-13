# Service Integration & Assembly Plan

This document addresses integration challenges across the 6 services and establishes the assembly pattern for the all-in-one server.

---

## Core Integration Decisions

### 1. Event Flow Architecture

**Health Metrics → Circuit Breaker (Event-Based)**
- HealthMonitoringService publishes `HealthUpdatedEvent` via EventPublisher
- CircuitBreakerService subscribes to `HealthUpdatedEvent` in its `OnHealthUpdate()` callback
- **Benefit:** Loose coupling; CircuitBreaker doesn't depend on Health internals
- **Implementation:** InMemoryEventBus (pub/sub registry in core/services)

**Result Messages → Multiple Services (Central Dispatcher)**
- NATSMessageDispatcher receives result messages from NATS
- Routes to a central `ResultMessageHandler` that dispatches to:
  1. LoopDetectionService.ProcessResult() (for ban detection)
  2. HealthMonitoringService.OnResultReceived() (for incremental health)
- **Benefit:** Single NATS subscription, ordered processing, transaction-like semantics
- **Implementation:** ResultMessageDispatcher service (new core/services layer)

### 2. Dispatch Filtering Integration

**Loop Detection Ban Enforcement at Dispatch Time**
- DispatchCoordinationService calls DispatchFilterService.FilterDispatchList() BEFORE sending
- DispatchFilterService calls BanEnforcementService.IsBanned() for each client
- Banned clients are filtered out; safe dispatch list is sent to NATS
- **Benefit:** Prevents dispatch to banned clients; ban enforcement is immediate
- **Integration point:** DispatchCoordinationService (workflow-orchestration) → DispatchFilterService (loop-detection-ban)

### 3. Repository Consolidation

**Single Shared Repository Layer**
- Create `RepositoryRegistry` that owns all DB connections and repository instances
- All services receive RepositoryRegistry, not individual repositories
- Benefits:
  - Single DB connection pool (SQLite connection pooling via WAL mode)
  - Consistent transaction handling
  - Centralized schema initialization
  - Simplified testing (mock registry)

**Repository Organization:**
```
RepositoryRegistry provides:
  - WorkflowRepository (read-write)
  - ClientRepository (read-only)
  - RunRepository (read-write)
  - ResultRepository (read-write)
  - BanRepository (read-write)
  - HealthRepository (read-write)
  - CircuitBreakerStateRepository (read-write)
```

### 4. Service Dependency Graph

```
                        API Service
                            ↑
                (reads from all services)
                            
    ┌─────────────────────────────────────────────┐
    │                                             │
    ↓                                             ↓
WorkflowOrchestration      CircuitBreaker     Health
    ↑                            ↑                ↑
    │                            │                │
DynamicGrouping         HealthMonitoring    LoopDetection
    │                            ↑                ↑
    └────────────────────────────┼────────────────┘
                                 │
                        ResultMessageDispatcher
                                 ↑
                           (NATS results)
```

**Dependency Levels:**
- Level 0 (no dependencies): DynamicGrouping, LoopDetection, HealthMonitoring (domain logic)
- Level 1 (depends on Level 0): WorkflowOrchestration (uses DynamicGrouping), CircuitBreaker (uses HealthMonitoring via events)
- Level 2 (depends on Level 1): API Service (reads from all)

---

## Revised Service Architecture

### Shared Core Components

**EventBus** (in core/services)
- Generic pub/sub for inter-service communication
- Supports: HealthUpdatedEvent, CircuitBreakerStateChangedEvent, WorkflowCompletionEvent
- In-process, no external dependencies (can be replaced with Redis pub/sub later)

**ResultMessageDispatcher** (in core/services)
- Routes NATS result messages to registered handlers
- Ensures single subscription, ordered processing
- Handlers: LoopDetectionService, HealthMonitoringService

**RepositoryRegistry** (in core/services)
- Singleton that provides all repository instances
- Owns DB connection
- Initialized at startup, used by all services

### Service Ports Changes

**HealthMonitoringService**
- Add: `EventPublisher` port (to emit HealthUpdatedEvent)
- Remove: MetricsPublisher from this plan (handle separately)

**CircuitBreakerService**
- Add: `EventBus` subscription (instead of health repository polling)
- Reactive to HealthUpdatedEvent

**LoopDetectionService**
- Add: `ResultMessageHandler` interface (callback from ResultMessageDispatcher)

**DispatchCoordinationService** (in WorkflowOrchestration)
- Add: `DispatchFilterService` dependency (to filter banned clients)

---

## Initialization Order (Critical)

Startup sequence (in cmd/main.go):

```
1. Initialize logging
2. Parse environment configuration
3. Initialize Database
   - Open SQLite connection
   - Run schema migrations
4. Create RepositoryRegistry
   - Instantiate all repositories with shared DB connection
5. Create EventBus
6. Create Domain Services (stateless)
   - HealthAggregator
   - CircuitBreakerLogic
   - LoopDetector
   - BanManager
   - FilterEvaluationService
7. Create Application Services (in dependency order)
   - DynamicGroupingService (no deps)
   - LoopDetectionService (depends: ResultRepository, RunRepository, BanRepository)
   - BanEnforcementService (depends: BanRepository, EventBus)
   - DispatchFilterService (depends: BanEnforcementService)
   - HealthMonitoringService (depends: RepositoryRegistry, EventBus)
   - CircuitBreakerService (depends: RepositoryRegistry, EventBus)
   - WorkflowOrchestrationService (depends: RepositoryRegistry, DynamicGrouping, DispatchFilterService)
8. Create ResultMessageDispatcher
   - Register LoopDetectionService.ProcessResult as handler
   - Register HealthMonitoringService.OnResultReceived as handler
9. Create TriggerCoordinationService
   - Register trigger evaluator
10. Create API Service
    - Register all service ports as adapters
11. Create HTTP Server
    - Register all API handlers
    - Register middleware (logging, error handling)
12. Subscribe to NATS result messages
    - NATSMessageDispatcher.Subscribe → ResultMessageDispatcher.Handle
13. Start background goroutines
    a. TriggerCoordinationService (trigger evaluation loop)
    b. ResultMessageDispatcher (NATS message consumption)
    c. HealthAggregationService (periodic health recalculation)
    d. CircuitBreakerService (periodic evaluation loop)
14. Start HTTP server listener
15. Wait for SIGINT/SIGTERM
16. Graceful shutdown (see below)
```

---

## Graceful Shutdown Coordination

**GracefulShutdownManager** (new in core/services)

Responsibilities:
- Coordinates shutdown of all components
- Ensures in-order shutdown
- Handles timeouts

Shutdown sequence:

```
1. Stop accepting new HTTP requests
   - Close HTTP listener
2. Drain in-flight HTTP requests (timeout: 30 seconds)
3. Stop background goroutines (in reverse init order)
   - CircuitBreakerService.Stop()
   - HealthAggregationService.Stop()
   - ResultMessageDispatcher.Stop() → NATS unsubscribe
   - TriggerCoordinationService.Stop()
4. Finalize pending database operations
   - HealthMonitoringService.Finalize()
   - LoopDetectionService.Finalize()
   - BanEnforcementService.Finalize()
5. Close database connection (ensures WAL checkpoint)
6. Close NATS connection
7. Exit
```

---

## Data Flow Diagram

### Trigger → Dispatch → Result → Health → CircuitBreaker

```
TriggerCoordinationService
  ↓ (fires trigger)
WorkflowOrchestrationService
  ↓ (evaluates filter)
DynamicGroupingService → Returns [ClientID, ClientID, ...]
  ↓
DispatchCoordinationService
  ↓ (filters banned clients)
DispatchFilterService
  ├─ calls BanEnforcementService.IsBanned() for each client
  └─ returns safe dispatch list
    ↓
NATSMessageDispatcher
  ├─ sends dispatch.{client_id} messages to NATS
  └─ (clients receive and process)
    ↓ (clients send results)
NATS result.{server_id} channel
  ↓
ResultMessageDispatcher
  ├─ routes to LoopDetectionService.ProcessResult()
  │   ├─ detects loops
  │   └─ calls BanEnforcementService.BanClient() if loop found
  │
  ├─ routes to HealthMonitoringService.OnResultReceived()
  │   ├─ updates run health incrementally
  │   └─ publishes HealthUpdatedEvent via EventBus
  │
  └─ returns processed status
    ↓ (EventBus)
CircuitBreakerService.OnHealthUpdate()
  ├─ fetches latest health
  ├─ evaluates circuit breaker policy
  └─ if state changed: deactivates/activates workflow
    ↓
API queries → current state of health, circuit, bans
```

---

## Repository Access Patterns

### Read-Only vs Read-Write

**Read-Only** (no transactions needed):
- ClientRepository.ListClients()
- WorkflowRepository.GetWorkflow()
- HealthRepository.GetWorkflowTypeHealth()

**Read-Write** (with transaction boundaries):
- RunRepository.CreateRun() + UpdateRun() (coupled operation)
- HealthRepository.SaveRunHealth() + SaveWorkflowTypeHealth()
- BanRepository.SaveBan()
- CircuitBreakerStateRepository.SaveCircuitState()

**Transactions:**
- Use SQLite implicit transactions (WAL mode provides ACID)
- For multi-step updates, wrap in explicit transactions
- Example: CreateRun must be atomic (run record + runs_clients join)

---

## Configuration & Environment Variables

**Centralized config loading** (cmd/config.go):

```
HTTP Configuration:
  HTTP_HOST (default: 0.0.0.0)
  HTTP_PORT (default: 8080)
  HTTP_READ_TIMEOUT_MS (default: 30000)

Database:
  DB_PATH (default: ./data/server.db)

NATS:
  NATS_URL (default: nats://localhost:4222)

Trigger Evaluation:
  TRIGGER_CHECK_INTERVAL_MS (default: 5000)

Health Monitoring:
  HEALTH_AGGREGATION_INTERVAL_MS (default: 5000)
  HEALTH_WINDOW_SIZE (default: 10)
  HEALTH_SUCCESS_THRESHOLD (default: 80)

Circuit Breaker:
  CIRCUIT_BREAKER_CHECK_INTERVAL_MS (default: 10000)
  CIRCUIT_BREAKER_SUCCESS_THRESHOLD (default: 80)
  CIRCUIT_BREAKER_COOLDOWN_MS (default: 300000)

Loop Detection:
  LOOP_THRESHOLD_MS (default: 5000)
  ENABLE_PERMANENT_BAN (default: true)
  BAN_ESCALATION_COUNT (default: 3)

Logging:
  LOG_LEVEL (default: info)
```

---

## Testing Strategy

### Unit Tests
- All domain layer logic (health calculation, loop detection, filter evaluation)
- Service layer logic with mocked ports
- Repository adapters with in-memory implementations

### Integration Tests
- RepositoryRegistry with SQLite (in-memory database)
- EventBus pub/sub
- ResultMessageDispatcher with multiple handlers
- Full workflow: trigger → dispatch → result → health → circuit breaker

### E2E Tests
- Start full server with NATS
- Trigger workflow → send result → verify health/circuit state
- Ban client → verify dispatch exclusion
- Unban client → verify dispatch inclusion

---

## File Structure (Consolidated)

```
.
├── cmd/
│   ├── main.go                         # Server entry point with full wiring
│   ├── config.go                       # Configuration loading from env vars
│   └── wiring.go                       # Complex wiring logic extracted
│
├── internal/
│   ├── core/
│   │   ├── domain/                     # All domain models (from individual plans)
│   │   ├── ports/                      # All service interfaces (consolidated)
│   │   └── services/
│   │       ├── repository_registry.go  # NEW: Shared repository manager
│   │       ├── event_bus.go            # NEW: Inter-service pub/sub
│   │       ├── result_dispatcher.go    # NEW: Result message router
│   │       ├── shutdown_manager.go     # NEW: Graceful shutdown coordinator
│   │       ├── health_monitoring.go    # Existing
│   │       ├── circuit_breaker.go      # Existing
│   │       ├── loop_detection.go       # Existing
│   │       ├── ban_enforcement.go      # Existing
│   │       ├── dispatch_filter.go      # Existing (enhanced)
│   │       ├── workflow_orchestration.go # Existing
│   │       ├── dynamic_grouping.go     # Existing
│   │       └── api_handler.go          # Existing
│   │
│   └── adapters/
│       ├── repository/
│       │   ├── sqlite.go               # Shared SQLite setup
│       │   ├── workflow_repo.go
│       │   ├── client_repo.go
│       │   ├── run_repo.go
│       │   ├── result_repo.go
│       │   ├── ban_repo.go
│       │   ├── health_repo.go
│       │   └── circuit_breaker_repo.go
│       │
│       ├── messaging/
│       │   ├── nats_dispatcher.go      # NATS integration
│       │   └── inmemory_event_bus.go
│       │
│       ├── http/
│       │   ├── server.go               # HTTP routing
│       │   └── middleware.go
│       │
│       └── trigger/
│           └── cron_evaluator.go

├── migrations/
│   └── schema.sql                      # Combined schema for all services

└── README.md
```

---

## Key Implementation Notes

### EventBus Design
```go
type Event interface {
    EventType() string
    Timestamp() time.Time
}

type EventHandler func(ctx context.Context, event Event) error

type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(eventType string, handler EventHandler) error
    Unsubscribe(eventType string, handler EventHandler) error
}
```

### RepositoryRegistry Design
```go
type RepositoryRegistry interface {
    WorkflowRepository() WorkflowRepository
    ClientRepository() ClientRepository
    RunRepository() RunRepository
    ResultRepository() ResultRepository
    BanRepository() BanRepository
    HealthRepository() HealthRepository
    CircuitBreakerStateRepository() CircuitBreakerStateRepository
}
```

### ResultMessageDispatcher Design
```go
type ResultHandler interface {
    HandleResult(ctx context.Context, result *Result) error
    Priority() int  // Higher = runs first (LoopDetection → HealthMonitoring)
}

type ResultMessageDispatcher interface {
    RegisterHandler(handler ResultHandler) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

---

## Integration Checklist

When implementing all 6 services together:

- [ ] Database schema includes all tables (runs, results, bans, health, circuits, clients, workflows)
- [ ] RepositoryRegistry is created and available to all services
- [ ] EventBus is initialized before services that publish/subscribe
- [ ] ResultMessageDispatcher routes to LoopDetection (priority 1) then HealthMonitoring (priority 2)
- [ ] DispatchCoordinationService calls DispatchFilterService before sending
- [ ] CircuitBreakerService subscribes to HealthUpdatedEvent
- [ ] All services have consistent error handling
- [ ] Graceful shutdown stops in reverse init order
- [ ] Configuration validates all required env vars
- [ ] Schema migrations run at startup
- [ ] Integration tests cover the full data flow

---

## Summary: Clean Integration Achieved

**Separation of Concerns:**
- Each service has a single responsibility
- Services communicate via event bus (loose coupling)
- Domain logic is independent of infrastructure

**Clear Data Flow:**
- Trigger → Dispatch → Result → Health → Circuit (linear pipeline)
- Loop Detection handles bans in parallel to health updates
- API reads from all services (no writes except admin operations)

**Minimal Coupling:**
- Services don't import each other's code
- All dependencies go through ports/interfaces
- Easy to test, extend, or replace implementations

**Consistency:**
- Single RepositoryRegistry for all database access
- Shared EventBus for inter-service events
- Centralized configuration and shutdown

This design allows all 6 services to integrate cleanly in a single all-in-one server while maintaining their independence.
