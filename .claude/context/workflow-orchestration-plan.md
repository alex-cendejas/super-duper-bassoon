# Workflow & Orchestration Service - Hexagonal Architecture Plan

## Overview
The Workflow & Orchestration Service is responsible for executing workflows on trigger, managing dispatches to clients, and coordinating the overall workflow lifecycle. It must handle multiple trigger types (scheduled, event-driven, state-change), generate unique run_ids, and track run completion.

## Step 1: Core/Domain Layer

### Responsibilities
Define pure business logic for workflow execution without external dependencies.

### Data Models & Methods

**Workflow Domain Model**
- `WorkflowID`: unique identifier
- `Definition`: the workflow specification
  - `Activity`: action type (reboot, install, upgrade, remove, apply_config, validate, run_script)
  - `Trigger`: trigger configuration (scheduled/event/state-change)
  - `TargetFilter`: filter expression for client selection
  - `SuccessThreshold`: circuit breaker threshold (%)
  - `LoopThreshold`: time window for loop detection (ms)
  - `Timeout`: activity timeout (ms)
  - `Active`: bool flag (can be deactivated by circuit breaker)
- Methods:
  - `IsActive()`: check if workflow can dispatch
  - `ValidateDefinition()`: business rule validation
  - `GetActivityTimeout()`: return timeout for activity

**Run Domain Model**
- `RunID`: unique per trigger cycle
- `WorkflowID`: reference to workflow
- `TriggeredAt`: timestamp
- `ParticipatingClients`: snapshot of client list at trigger time
- `DispatchedAt`: timestamp when dispatch completed
- `Health`: current run health (populated incrementally)
- `State`: enum (pending, in_progress, completed, failed)
- Methods:
  - `CanComplete()`: check if enough results received
  - `IsExpired()`: check against timeout
  - `CalculateHealth()`: compute health from partial results

**Dispatch Domain Model**
- `RunID`: which run this dispatch belongs to
- `WorkflowID`: which workflow
- `ClientID`: target client
- `Activity`: action to perform
- `Params`: activity-specific parameters
- `DispatchedAt`: timestamp
- Methods:
  - `IsValid()`: structural validation
  - `GetPayload()`: serialize for transmission

**WorkflowTrigger Domain Models**
- `ScheduledTrigger`: cron expression + evaluation logic
- `EventDrivenTrigger`: event source + event type + workflow chaining
- `StateChangeTrigger`: client state path + condition
- Methods per trigger type:
  - `ShouldExecute(context)`: bool indicating if trigger fires

**TriggerResult Domain Model**
- `TriggeredWorkflows`: list of WorkflowIDs to execute
- `TriggeredAt`: timestamp
- `Reason`: what caused the trigger (for audit)

### File Structure
```
internal/core/domain/
  workflow.go              # Workflow struct + methods
  workflow_test.go
  run.go                   # Run struct + methods
  run_test.go
  dispatch.go              # Dispatch struct + methods
  dispatch_test.go
  trigger.go               # Trigger interfaces + implementations
  trigger_test.go
  errors.go                # domain-specific errors
```

---

## Step 2: Core/Ports Layer

### Responsibilities
Define generic interfaces for external dependencies needed by the orchestration logic.

### Port Interfaces

**WorkflowRepository Port**
- `GetWorkflow(ctx, workflowID) (*Workflow, error)`
- `ListActiveWorkflows(ctx) ([]*Workflow, error)`
- `SaveWorkflow(ctx, *Workflow) error`
- `UpdateWorkflowState(ctx, workflowID, active bool) error`

**ClientRepository Port** (used to resolve filter expression)
- `GetClientByID(ctx, clientID) (*Client, error)`
- `ListClients(ctx) ([]*Client, error)`
- `GetClientsByIDs(ctx, []ClientID) ([]*Client, error)`

**MessageDispatcher Port** (sends to NATS)
- `SendDispatch(ctx, *Dispatch) error`
- `SendBatchDispatches(ctx, []*Dispatch) error`
- `SubscribeToResults(ctx, handler ResultHandler) error`

**RunRepository Port**
- `CreateRun(ctx, *Run) error`
- `GetRun(ctx, runID) (*Run, error)`
- `UpdateRun(ctx, *Run) error`
- `ListRuns(ctx, workflowID) ([]*Run, error)`

**TriggerEvaluator Port** (determines if/when workflows should execute)
- `EvaluateTriggers(ctx, []*Trigger) ([]*TriggerResult, error)`
- `RegisterTriggerHandler(trigger Trigger, handler TriggerHandler) error`

**EventPublisher Port** (for workflow completion events)
- `PublishEvent(ctx, event WorkflowCompletionEvent) error`
- `SubscribeToEvents(ctx, eventType string, handler EventHandler) error`

### File Structure
```
internal/core/ports/
  workflow_repository.go      # WorkflowRepository interface
  client_repository.go        # ClientRepository interface
  message_dispatcher.go       # MessageDispatcher interface
  run_repository.go           # RunRepository interface
  trigger_evaluator.go        # TriggerEvaluator interface
  event_publisher.go          # EventPublisher interface
```

---

## Step 3: Core/Services Layer

### Responsibilities
Orchestrate domain logic using ports to provide high-level workflow execution capabilities.

### Services

**WorkflowOrchestrationService**
- Dependencies: WorkflowRepository, ClientRepository, MessageDispatcher, RunRepository, DynamicGroupingService, EventPublisher
- Methods:
  - `TriggerWorkflow(ctx, workflowID, reason string) (*Run, error)`: main entry point
    - Fetch workflow definition
    - Check if active
    - Evaluate filter to get participating clients
    - Create run record
    - Generate dispatches
    - Send to MessageDispatcher
    - Return run
  - `HandleTrigger(ctx, trigger Trigger) error`: react to trigger events
  - `ProcessResults(ctx, runID, results []Result) error`: update run health (delegates to HealthMonitoringService)

**TriggerCoordinationService**
- Dependencies: TriggerEvaluator, WorkflowOrchestrationService, EventPublisher
- Methods:
  - `StartTriggerMonitoring(ctx)`: goroutine to continuously evaluate triggers
  - `OnTriggerFire(ctx, trigger Trigger, workflowID WorkflowID) error`: callback
  - `HandleWorkflowCompletion(ctx, event WorkflowCompletionEvent) error`: execute event-driven workflows

**DispatchCoordinationService**
- Dependencies: RunRepository, MessageDispatcher, ClientRepository, DispatchFilterService
- Methods:
  - `GenerateDispatches(ctx, run *Run, clients []*Client) ([]*Dispatch, error)`
    - Create dispatch per client in run
    - Validate dispatch structure
    - Return list
  - `SendDispatches(ctx, []*Dispatch) error`
    - Filter dispatch list via DispatchFilterService (removes banned clients)
    - Batch send via MessageDispatcher
    - Handle NATS errors
    - Log dispatch metrics (including count of filtered clients)

### File Structure
```
internal/core/services/
  workflow_orchestration.go       # WorkflowOrchestrationService
  workflow_orchestration_test.go
  trigger_coordination.go         # TriggerCoordinationService
  trigger_coordination_test.go
  dispatch_coordination.go        # DispatchCoordinationService
  dispatch_coordination_test.go
```

---

## Step 4: Adapters Layer

### Responsibilities
Implement the port interfaces with concrete technologies.

### Adapter Implementations

**SQLiteWorkflowRepository** → WorkflowRepository port
- Uses database/sql with sqlc or raw SQL
- Tables: workflows (id, definition_json, active, created_at, updated_at)
- Implementation: `GetWorkflow`, `ListActiveWorkflows`, `SaveWorkflow`, `UpdateWorkflowState`

**SQLiteClientRepository** → ClientRepository port
- Tables: clients (id, os, labels_json, inner_state_json, created_at, updated_at)
- Implementation: `GetClientByID`, `ListClients`, `GetClientsByIDs`

**NATSMessageDispatcher** → MessageDispatcher port
- Connects to NATS broker
- Publishes dispatch messages to `dispatch.{client_id}` subject
- Subscribes to `result.{server_id}` subject to receive results
- Implementation: `SendDispatch`, `SendBatchDispatches`, `SubscribeToResults`

**SQLiteRunRepository** → RunRepository port
- Tables: runs (run_id, workflow_id, triggered_at, dispatched_at, state, health_json, created_at)
  - runs_clients (run_id, client_id) - join table for participating clients
- Implementation: `CreateRun`, `GetRun`, `UpdateRun`, `ListRuns`

**CronAndEventTriggerEvaluator** → TriggerEvaluator port
- Uses robfig/cron for scheduled triggers
- Maintains in-memory state for event and state-change trigger subscriptions
- Implementation: `EvaluateTriggers`, `RegisterTriggerHandler`

**InMemoryEventPublisher** → EventPublisher port
- In-process pub/sub for workflow completion events
- (Can be replaced with Redis/NATS later)
- Implementation: `PublishEvent`, `SubscribeToEvents`

### File Structure
```
internal/adapters/
  repository/
    sqlite_workflow_repo.go
    sqlite_workflow_repo_test.go
    sqlite_client_repo.go
    sqlite_client_repo_test.go
    sqlite_run_repo.go
    sqlite_run_repo_test.go
  messaging/
    nats_dispatcher.go
    nats_dispatcher_test.go
  trigger/
    cron_evaluator.go
    cron_evaluator_test.go
    event_trigger_evaluator.go
  event/
    inmemory_publisher.go
    inmemory_publisher_test.go
```

---

## Step 5: Binary/Deployment Layer

### Responsibilities
Wire together all components, load configuration, run services.

### Configuration
- Environment variables:
  - `NATS_URL`: NATS broker connection string
  - `DB_PATH`: SQLite database file path
  - `TRIGGER_CHECK_INTERVAL_MS`: how often to evaluate triggers
  - `RUN_COMPLETION_TIMEOUT_MS`: timeout for run completion

### Initialization & Wiring

**cmd/main.go** (integrated into server)
```
1. Parse env variables
2. Initialize DB connection
3. Initialize NATS connection
4. Create RepositoryRegistry (shared repos)
5. Create EventBus (inter-service pub/sub)
6. Create domain services
7. Create LoopDetectionService and BanEnforcementService (priority 1 result handlers)
8. Create HealthMonitoringService (priority 2 result handler)
9. Create DispatchFilterService (uses BanEnforcementService)
10. Create DispatchCoordinationService (uses DispatchFilterService)
11. Create WorkflowOrchestrationService (uses DispatchCoordinationService)
12. Create ResultMessageDispatcher and register handlers
13. Create TriggerCoordinationService (uses WorkflowOrchestrationService)
14. Start all services and background goroutines
15. Setup graceful shutdown (close resources in reverse order)
```

**cmd/config.go**
- `LoadConfig()`: read env vars, validate, return config struct
- Env vars: NATS_URL, DB_PATH, LOG_LEVEL, TRIGGER_CHECK_INTERVAL_MS, etc.

### File Structure
```
cmd/
  main.go          # orchestration service entrypoint
  config.go        # env var loading
```

### Runtime Behavior
1. On startup: load all active workflows, initialize trigger monitoring
2. Continuous: TriggerCoordinationService evaluates triggers at configured interval
3. On trigger fire: WorkflowOrchestrationService.TriggerWorkflow()
   - Creates run record
   - Evaluates filter via DynamicGroupingService
   - Generates dispatches
   - Calls DispatchCoordinationService.SendDispatches()
4. In SendDispatches:
   - Calls DispatchFilterService.FilterDispatchList() (removes banned clients)
   - Sends filtered dispatch list via NATS
5. On result message from NATS:
   - ResultMessageDispatcher routes to registered handlers (priority order):
     1. LoopDetectionService.ProcessResult() (priority 1) - detects loops, applies bans
     2. HealthMonitoringService.OnResultReceived() (priority 2) - updates run health, publishes HealthUpdatedEvent
6. On HealthUpdatedEvent:
   - CircuitBreakerService.OnHealthUpdatedEvent() (subscribed to EventBus)
   - Evaluates circuit breaker policy, deactivates/activates workflow if needed
7. On completion or timeout: publish workflow completion event for downstream consumers
8. On shutdown: graceful shutdown coordinator stops all services in reverse order

---

## Implementation Dependencies

- **Depends on:** DynamicGroupingService (for filter evaluation), DispatchFilterService (for ban filtering)
- **Calls:** WorkflowRepository, RunRepository, ClientRepository, DispatchCoordinationService
- **Used by:** TriggerCoordinationService (drives workflow execution), API Service (query workflows/runs)
- **Messaging:** NATS for client dispatch/result via NATSMessageDispatcher, ResultMessageDispatcher for result routing
- **Database:** SQLite for workflow/run/client persistence via RepositoryRegistry
