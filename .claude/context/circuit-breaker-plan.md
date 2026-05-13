# Circuit Breaker Service - Hexagonal Architecture Plan

## Overview
The Circuit Breaker Service monitors aggregated health metrics per workflow_type and automatically deactivates workflows when health falls below a configured success threshold. This prevents cascading failures across the client fleet. It is a critical safety mechanism.

## Step 1: Core/Domain Layer

### Responsibilities
Define pure business logic for circuit breaker state management and policy evaluation.

### Data Models & Methods

**CircuitState Domain Model** (enum-like)
- `Closed`: workflow is active, dispatches are allowed
- `Open`: workflow is deactivated, no dispatches
- `HalfOpen`: temporary state for recovery testing (optional advanced feature)
- Methods:
  - `CanDispatch()`: bool, true only if Closed
  - `Description()`: human-readable state name

**CircuitBreakerPolicy Domain Model**
- `SuccessThreshold`: minimum success % to keep workflow Open (e.g., 80%)
- `EvaluationWindow`: number of runs or time window to aggregate (e.g., last 10 runs)
- `CooldownPeriod`: minimum time before attempting to reopen (e.g., 5 min)
- Methods:
  - `Validate()`: check thresholds are reasonable
  - `IsHealthyEnough(workflowHealth)`: bool, applies success_threshold
  - `Describe()`: policy summary

**WorkflowCircuitBreaker Domain Model** (per-workflow state)
- `WorkflowID`: which workflow this breaker monitors
- `State`: CircuitState (Closed/Open/HalfOpen)
- `OpenedAt`: timestamp when state changed to Open
- `LastEvaluatedAt`: timestamp of last health evaluation
- `OpenedReason`: why the circuit opened (health metrics snapshot)
- `EvaluationCount`: number of times health has been checked
- Methods:
  - `CanDispatch()`: delegate to state
  - `IsRecoveryReady(cooldownPeriod)`: bool, check if enough time passed since opened
  - `Describe()`: circuit status summary

**CircuitBreakerLogic Domain Model** (stateless decision logic)
- Methods:
  - `EvaluateHealth(policy, workflowHealth) CircuitState`:
    - If health.SuccessPercentageAvg >= policy.SuccessThreshold → Closed
    - If health.SuccessPercentageAvg < policy.SuccessThreshold → Open
    - (HalfOpen logic optional for advanced recovery)
  - `ShouldAttemptRecovery(state, openedAt, cooldownPeriod) bool`:
    - Check if state is Open and cooldownPeriod has elapsed
    - Return true if recovery should be attempted
  - `EvaluateRecoveryRun(policy, newHealth) CircuitState`:
    - In HalfOpen state, evaluate single recovery run
    - Return Closed if successful, Open if failed

**CircuitBreakerAlert Domain Model**
- `WorkflowID`, `WorkflowType`, `Event`: enum (opened, closed, recovery_attempted, recovery_failed)
- `Reason`: snapshot of health metrics that triggered state change
- `Timestamp`: when event occurred
- `Severity`: critical (opened), warning (recovery_failed), info (closed)
- Methods:
  - `Describe()`: alert message with health details

**CircuitBreakerEvent Domain Model** (publishable)
- `WorkflowID`, `WorkflowType`, `OldState`, `NewState`
- `Timestamp`, `Health`: the health metrics that caused transition
- Methods:
  - `Describe()`: event summary

### File Structure
```
internal/core/domain/
  circuit_state.go           # CircuitState enum
  circuit_state_test.go
  circuit_breaker_policy.go  # CircuitBreakerPolicy model
  circuit_breaker_policy_test.go
  workflow_circuit_breaker.go # WorkflowCircuitBreaker model
  workflow_circuit_breaker_test.go
  circuit_breaker_logic.go   # Decision logic (stateless)
  circuit_breaker_logic_test.go
  alerts.go                  # Alert/Event models
  alerts_test.go
  errors.go                  # domain-specific errors
```

---

## Step 2: Core/Ports Layer

### Responsibilities
Define generic interfaces for health data and workflow state management.

### Port Interfaces

**EventPublisher Port** (subscribe to health updates)
- `SubscribeToHealthUpdatedEvent(ctx, handler EventHandler) error`: react to health changes
  - Handler receives HealthUpdatedEvent with current health and workflow_type

**HealthRepository Port** (read-only for queries)
- `GetWorkflowTypeHealth(ctx, workflowType) (*WorkflowTypeHealth, error)`: fetch latest aggregated health
- `GetWorkflowTypeHealthHistory(ctx, workflowType, limit) ([]*WorkflowTypeHealth, error)`: trend data

**WorkflowRepository Port** (read-write for state changes)
- `GetWorkflow(ctx, workflowID) (*Workflow, error)`: fetch for policy and current state
- `UpdateWorkflowState(ctx, workflowID, active bool, reason string) error`: deactivate workflow
- `ListWorkflowsByType(ctx, workflowType) ([]*Workflow, error)`: all workflows of a type

**CircuitBreakerStateRepository Port** (state persistence)
- `SaveCircuitState(ctx, *WorkflowCircuitBreaker) error`: persist breaker state
- `GetCircuitState(ctx, workflowID) (*WorkflowCircuitBreaker, error)`: fetch current state
- `ListCircuitStates(ctx) ([]*WorkflowCircuitBreaker, error)`: all breakers

**PolicyRepository Port** (read-only, policy configuration)
- `GetPolicy(ctx, workflowID) (*CircuitBreakerPolicy, error)`: fetch policy for workflow
- `GetDefaultPolicy(ctx) (*CircuitBreakerPolicy, error)`: fallback default policy

**AlertPublisher Port** (notifications)
- `PublishAlert(ctx, *CircuitBreakerAlert) error`: publish alert
- `PublishEvent(ctx, *CircuitBreakerEvent) error`: publish state change event

**WorkflowStateManager Port** (state change enforcement)
- `DeactivateWorkflow(ctx, workflowID, reason) error`: disable workflow
- `ActivateWorkflow(ctx, workflowID, reason) error`: re-enable workflow

### File Structure
```
internal/core/ports/
  health_repository.go      # HealthRepository interface
  workflow_repository.go    # WorkflowRepository interface
  circuit_state_repository.go # CircuitBreakerStateRepository interface
  policy_repository.go      # PolicyRepository interface
  alert_publisher.go        # AlertPublisher interface
  workflow_state_manager.go # WorkflowStateManager interface
```

---

## Step 3: Core/Services Layer

### Responsibilities
Orchestrate domain logic using ports to implement circuit breaker monitoring.

### Services

**CircuitBreakerService** (main service)
- Dependencies: HealthRepository, CircuitBreakerStateRepository, PolicyRepository, AlertPublisher, WorkflowStateManager, CircuitBreakerLogic, EventPublisher
- Methods:
  - `EvaluateWorkflowHealth(ctx, workflowType) error`: main entrypoint
    - Fetch all workflows of this type
    - For each workflow:
      - Get current circuit state from repository
      - Fetch latest health from HealthRepository
      - Fetch policy from PolicyRepository
      - Call CircuitBreakerLogic.EvaluateHealth(policy, health)
      - If state changed (Open → Closed or vice versa):
        - Save new state to CircuitBreakerStateRepository
        - Call WorkflowStateManager to deactivate/activate
        - Publish CircuitBreakerAlert and CircuitBreakerEvent
      - Return result
  - `OnHealthUpdatedEvent(ctx, event HealthUpdatedEvent) error`: reactive callback from HealthMonitoringService
    - Extract workflowType from event
    - Trigger EvaluateWorkflowHealth() for that type
  - `StartPeriodicEvaluation(ctx, interval time.Duration) error`: background periodic evaluation
    - Tick at configured interval
    - Call EvaluateAllWorkflows() to catch any health changes that weren't published
  - `EvaluateAllWorkflows(ctx) error`: iterate all active workflows
    - For each unique workflow_type: call EvaluateWorkflowHealth()
  - `GetCircuitState(ctx, workflowID) (*WorkflowCircuitBreaker, error)`: query interface
    - Fetch current state from repository
    - Return state + health info

**WorkflowStateManagementService** (manages state changes)
- Dependencies: WorkflowRepository, CircuitBreakerStateRepository, AlertPublisher
- Methods:
  - `DeactivateWorkflow(ctx, workflowID, reason) error`:
    - Update workflow.active = false in repository
    - Save reason to CircuitBreakerStateRepository
    - Publish alert with reason
    - Log audit event
  - `ActivateWorkflow(ctx, workflowID, reason) error`:
    - Update workflow.active = true in repository
    - Clear OpenedAt in CircuitBreakerStateRepository
    - Publish alert with recovery info
    - Log audit event
  - `IsWorkflowActive(ctx, workflowID) bool`: query helper

### File Structure
```
internal/core/services/
  circuit_breaker.go        # CircuitBreakerService
  circuit_breaker_test.go
  workflow_state_management.go # WorkflowStateManagementService
  workflow_state_management_test.go
```

---

## Step 4: Adapters Layer

### Responsibilities
Implement the port interfaces with concrete technologies.

### Adapter Implementations

**SQLiteHealthRepository** → HealthRepository port (read-only)
- Reads existing workflow_type_health table
- Implementation: `GetWorkflowTypeHealth`, `GetWorkflowTypeHealthHistory`
- Latest health fetched by most recent calculated_at

**SQLiteWorkflowRepository** → WorkflowRepository port (read-write)
- Reads/updates workflows table
- Implementation: `GetWorkflow`, `UpdateWorkflowState`, `ListWorkflowsByType`
- State update: `UPDATE workflows SET active=false, deactivated_at=now(), deactivation_reason=? WHERE id=?`

**SQLiteCircuitBreakerStateRepository** → CircuitBreakerStateRepository port
- Table: circuit_breaker_states (workflow_id, state, opened_at, last_evaluated_at, opened_reason_json, evaluation_count, created_at, updated_at)
- Indexes: (workflow_id)
- Implementation: all CircuitBreakerStateRepository methods
- State enum: closed, open, half_open (stored as string)

**SQLitePolicyRepository** → PolicyRepository port (read-only)
- Reads from workflows table (success_threshold) or dedicated policies table
- Implementation: `GetPolicy`, `GetDefaultPolicy`
- Fallback: hard-coded default (success_threshold=80%, cooldown=5min)

**StdoutAlertPublisher** → AlertPublisher port
- Logs alerts/events to stdout with severity
- Implementation: `PublishAlert`, `PublishEvent`
- (Can be replaced with Slack/email/webhook later)

**WorkflowRepositoryStateManager** → WorkflowStateManager port
- Delegates to SQLiteWorkflowRepository
- Implementation: `DeactivateWorkflow`, `ActivateWorkflow`
- Ensures audit logging of state changes

### File Structure
```
internal/adapters/
  repository/
    sqlite_health_repo.go (read-only wrapper)
    sqlite_workflow_repo.go (read-write wrapper)
    sqlite_circuit_state_repo.go
    sqlite_circuit_state_repo_test.go
    sqlite_policy_repo.go
    sqlite_policy_repo_test.go
  alert/
    stdout_alert_publisher.go
    stdout_alert_publisher_test.go
  workflow/
    workflow_state_manager.go
    workflow_state_manager_test.go
```

---

## Step 5: Binary/Deployment Layer

### Responsibilities
Wire together components, load configuration, run monitoring loop.

### Configuration
- Environment variables:
  - `DB_PATH`: SQLite database file
  - `CIRCUIT_BREAKER_CHECK_INTERVAL_MS`: how often to evaluate all workflows (e.g., 10000 ms)
  - `CIRCUIT_BREAKER_SUCCESS_THRESHOLD`: default success % threshold (e.g., 80)
  - `CIRCUIT_BREAKER_COOLDOWN_MS`: min time before recovery attempt (e.g., 300000 ms = 5 min)
  - `CIRCUIT_BREAKER_POLICY_MODE`: enum (default=threshold, advanced=halfopen_recovery)

### Initialization & Wiring

**cmd/main.go** (integrated into server)
```
1. Parse env variables for circuit breaker config
2. Initialize DB connection (health, workflow, policy, circuit state repos)
3. Create port implementations (repos, alert publisher, state manager)
4. Create domain services (CircuitBreakerLogic, CircuitBreakerService, WorkflowStateManagementService)
5. Wire dependencies (services get ports as constructor params)
6. Subscribe to HealthUpdatedEvent (from EventBus)
   - Register CircuitBreakerService.OnHealthUpdatedEvent() handler
7. Start periodic evaluation goroutine (fallback, in case events are missed):
   - Tick at configured interval (e.g., 30 seconds)
   - Call CircuitBreakerService.EvaluateAllWorkflows()
   - Handles any state changes not caught by event-driven path
8. Setup graceful shutdown: unsubscribe from events, close connections
```

**cmd/config.go**
- `LoadConfig()`: read env vars, set defaults, validate
- Env vars: DB_PATH, CIRCUIT_BREAKER_CHECK_INTERVAL_MS, CIRCUIT_BREAKER_SUCCESS_THRESHOLD, CIRCUIT_BREAKER_COOLDOWN_MS, CIRCUIT_BREAKER_POLICY_MODE, LOG_LEVEL

### File Structure
```
cmd/
  main.go          # circuit breaker service wiring (integrated)
  config.go        # config loading
```

### Runtime Behavior
1. On startup: load all workflows and circuit states from DB
2. On health update event (from HealthMonitoringService):
   - CircuitBreakerService.OnHealthUpdate() is called
   - Health for that workflow_type is evaluated
   - If health < threshold and circuit is Closed:
     - State changes to Open
     - Workflow is deactivated (no more dispatches)
     - Alert published (critical severity)
     - No new runs can start for this workflow
   - If health >= threshold and circuit is Open:
     - State changes to Closed
     - Workflow is reactivated (dispatches resume)
     - Alert published (info/recovery severity)
3. Periodic (at configured interval):
   - CircuitBreakerService.EvaluateAllWorkflows() is called
   - All workflows are re-evaluated (health state change detection)
   - Alerts published for any state transitions
4. (Optional) Recovery mode (if HalfOpen feature enabled):
   - When Open, after cooldown expires: state → HalfOpen
   - Single dispatch is sent as test
   - If that run succeeds: state → Closed (recovery)
   - If it fails: state → Open again (longer cooldown)
5. On API request:
   - APIService calls CircuitBreakerService.GetCircuitState()
   - Returns current state + health + reason
6. On shutdown: persist final circuit states, close DB

### Integration Points
- **Subscribes to:** HealthUpdatedEvent from EventBus (from HealthMonitoringService)
- **Periodic fallback:** Evaluates all workflows periodically to catch missed health changes
- **Reads:** HealthRepository (latest health), WorkflowRepository (definitions), PolicyRepository (thresholds), CircuitBreakerStateRepository (current state)
- **Writes:** CircuitBreakerStateRepository (state changes), WorkflowRepository (active flag)
- **Publishes:** AlertPublisher (opened/closed/recovery events)
- **Affects:** DispatchCoordinationService reads workflow.active flag
- **Serves:** API Service (query circuit state)

---

## Implementation Dependencies

- **Depends on:** HealthRepository (aggregated health), WorkflowRepository (definitions/state), PolicyRepository (thresholds), CircuitBreakerStateRepository (persisted breaker state)
- **Used by:** DispatchCoordinationService (checks workflow.active before dispatching), API Service (query circuit state)
- **Database:** SQLite for workflow state, circuit breaker state, policies
- **Messaging:** Alert publishing (no NATS dependency for core logic)
- **Timing:** Decoupled evaluation from health monitoring, but reactive on health update events

---

## State Transition Diagram

```
                   health >= threshold
                  ┌────────────────────┐
                  │                    ▼
             ┌─────────────┐      ┌────────┐
             │    Open     │      │ Closed │◄─┐ (initial)
             └─────────────┘      └────────┘  │
                  ▲                      │     │
                  │ health < threshold   └──┬──┘
                  └────────────────────────┘
                                         
            (Optional HalfOpen feature adds third state)
            Closed →[cooldown + eval] HalfOpen →[success] Closed
            HalfOpen →[failure] Open
```
