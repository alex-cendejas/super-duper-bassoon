# Loop Detection & Ban Service - Hexagonal Architecture Plan

## Overview
The Loop Detection & Ban Service processes result messages from clients to detect loop conditions (same client executing same workflow multiple times within a threshold window), immediately bans offending clients, persists ban records, and prevents future dispatches to banned clients. It is a critical safety mechanism.

## Step 1: Core/Domain Layer

### Responsibilities
Define pure business logic for loop detection, ban management, and enforcement.

### Data Models & Methods

**LoopDetectionRecord Domain Model**
- `ClientID`: which client looped
- `WorkflowType`: the workflow type (derived from workflow_id)
- `CurrentRunID`: the result that triggered detection
- `PreviousRunID`: the earlier run that matches current run
- `DetectedAt`: timestamp of detection
- `Result`: activity result that triggered detection (for audit)
- Methods:
  - `IsValid()`: check required fields
  - `TimeWindow()`: duration between previous and current run

**BanRecord Domain Model**
- `ClientID`: which client is banned
- `WorkflowType`: which workflow type (null = all workflows)
- `RunIDEvidence`: run_id where loop/ban was triggered
- `ResultEvidence`: the result that triggered ban
- `BannedAt`: timestamp
- `BannedUntil`: null (permanent) or timestamp (temporary)
- `BanReason`: enum (loop_detected, manual, explicit_failure, etc.)
- `BannedBy`: who triggered the ban (system or admin)
- Methods:
  - `IsActive()`: check if currently banned
  - `IsPermanent()`: BannedUntil is null
  - `DaysRemaining()`: compute from BannedUntil
  - `CanUnban()`: false if permanent
  - `Describe()`: human-readable ban description

**LoopDetector Domain Model** (stateless detector)
- Methods:
  - `DetectLoop(clientID, workflowType, currentRunID, previousRunHistory, loopThreshold) (*LoopDetectionRecord, bool)`: 
    - Query run history for same workflow_type on this client
    - Check if previous run is within loopThreshold time window
    - Return detection record if loop found, else nil
  - `CalculateLoopWindow(loopThresholdMs)`: time window for lookback

**BanManager Domain Model** (enforces ban policy)
- Methods:
  - `CanDispatchToClient(clientID, workflowType, banRecords []*BanRecord) bool`:
    - Check if any active ban exists for (clientID, workflowType) or (clientID, null)
    - Return false if banned
  - `ApplyBan(clientID, workflowType, runID, reason) *BanRecord`:
    - Create ban record with current timestamp
    - Set permanent flag
  - `UnbanClient(clientID, workflowType, reason) error`:
    - Only allows if ban is temporary
    - Raises error if permanent
  - `ComputeBanStatus(bans []*BanRecord) enum`: active, inactive, expired

**LoopAlert Domain Model**
- `ClientID`, `WorkflowType`, `RunID`, `Timestamp`, `Severity`
- Methods:
  - `Describe()`: human-readable alert message
  - `IsEscalated()`: true if multiple loops detected for same client in time window

**BanAlert Domain Model**
- `ClientID`, `WorkflowType`, `Reason`, `BannedAt`, `PermanentFlag`, `AdminContactInfo`
- Methods:
  - `Describe()`: alert message

### File Structure
```
internal/core/domain/
  loop_detection_record.go    # LoopDetectionRecord model
  loop_detection_record_test.go
  ban_record.go               # BanRecord model
  ban_record_test.go
  loop_detector.go            # LoopDetector logic
  loop_detector_test.go
  ban_manager.go              # BanManager logic
  ban_manager_test.go
  alerts.go                   # Alert models
  alerts_test.go
  errors.go                   # domain-specific errors
```

---

## Step 2: Core/Ports Layer

### Responsibilities
Define generic interfaces for data access and notification.

### Port Interfaces

**BanRepository Port**
- `SaveBan(ctx, *BanRecord) error`: persist a ban
- `GetBans(ctx, clientID) ([]*BanRecord, error)`: fetch all bans for client
- `GetActiveBans(ctx, clientID) ([]*BanRecord, error)`: only active bans
- `GetBansByWorkflow(ctx, workflowType) ([]*BanRecord, error)`: all bans for workflow_type
- `UnbanClient(ctx, clientID, workflowType) error`: mark ban as inactive/expired
- `ListAllBans(ctx) ([]*BanRecord, error)`: admin query

**ResultRepository Port** (read-only)
- `GetResult(ctx, runID, clientID) (*Result, error)`: fetch single result
- `GetRunResults(ctx, runID) ([]*Result, error)`: all results for a run
- `ListClientResults(ctx, clientID, workflowType) ([]*Result, error)`: run history for client/workflow

**RunRepository Port** (read-only)
- `GetRun(ctx, runID) (*Run, error)`: fetch run details
- `GetPreviousRun(ctx, clientID, workflowType, beforeTime) (*Run, error)`: lookback for loop detection
- `ListClientRuns(ctx, clientID, workflowType) ([]*Run, error)`: client run history

**WorkflowRepository Port** (read-only, for workflow_type validation)
- `GetWorkflow(ctx, workflowID) (*Workflow, error)`: fetch to derive workflow_type

**AlertPublisher Port** (for notifications)
- `PublishAlert(ctx, alert Alert) error`: send loop or ban alert
- `PublishBulkAlerts(ctx, []Alert) error`: batch send

**DispatchBlocker Port** (prevents dispatch to banned clients)
- `ShouldBlockDispatch(ctx, clientID, workflowType) (bool, error)`: check bans before dispatch
- `RegisterBanCheckCallback(callback BanCheckCallback) error`: reactive update on new ban

### File Structure
```
internal/core/ports/
  ban_repository.go         # BanRepository interface
  result_repository.go      # ResultRepository interface
  run_repository.go         # RunRepository interface (read-only)
  workflow_repository.go    # WorkflowRepository interface (read-only)
  alert_publisher.go        # AlertPublisher interface
  dispatch_blocker.go       # DispatchBlocker interface
```

---

## Step 3: Core/Services Layer

### Responsibilities
Orchestrate domain logic using ports to implement loop detection and ban enforcement.

### Services

**LoopDetectionService** (primary service)
- Dependencies: ResultRepository, RunRepository, WorkflowRepository, LoopDetector, BanRepository, AlertPublisher
- Implements: ResultHandler interface (for ResultMessageDispatcher)
- Methods:
  - `ProcessResult(ctx, result *Result) error`: main entrypoint (called by ResultMessageDispatcher)
    - Fetch workflow_type from workflow_id
    - Fetch previous runs for (clientID, workflow_type)
    - Call LoopDetector.DetectLoop()
    - If loop detected: call BanEnforcementService.BanClient()
    - Publish LoopAlert
    - Return result processing status
  - `Priority() int`: returns 1 (higher priority than HealthMonitoring)
    - Ensures bans are applied before health is calculated
  - `GetLoopThreshold(ctx, workflowID) (time.Duration, error)`: query workflow config

**BanEnforcementService** (enforces bans)
- Dependencies: BanManager, BanRepository, AlertPublisher, DispatchBlocker
- Methods:
  - `BanClient(ctx, clientID, workflowType, runID, reason) error`: apply a ban
    - Create BanRecord via BanManager
    - Save to BanRepository
    - Publish BanAlert
    - Register with DispatchBlocker to prevent future dispatches
    - Log for audit
  - `UnbanClient(ctx, clientID, workflowType, adminID, reason) error`: unban a client
    - Check if ban is temporary (reject if permanent)
    - Mark as inactive in BanRepository
    - Notify DispatchBlocker
    - Log audit
  - `IsBanned(ctx, clientID, workflowType) (bool, error)`: check current ban status
    - Query BanRepository
    - Filter by active bans
    - Return bool

**DispatchFilterService** (blocks bans at dispatch time)
- Dependencies: BanEnforcementService
- Methods:
  - `FilterDispatchList(ctx, runID, candidateClients []*Client) ([]*Client, error)`: remove banned clients
    - For each candidate: check IsBanned()
    - Filter out banned clients
    - Log filtered-out clients
    - Return safe dispatch list
  - `OnBanApplied(ctx, ban *BanRecord)`: reactive callback
    - Notify any in-progress dispatches to abort for this client
    - (Optional, for safety)

### File Structure
```
internal/core/services/
  loop_detection.go           # LoopDetectionService
  loop_detection_test.go
  ban_enforcement.go          # BanEnforcementService
  ban_enforcement_test.go
  dispatch_filter.go          # DispatchFilterService
  dispatch_filter_test.go
```

---

## Step 4: Adapters Layer

### Responsibilities
Implement the port interfaces with concrete technologies.

### Adapter Implementations

**SQLiteBanRepository** → BanRepository port
- Table: bans (client_id, workflow_type, run_id_evidence, result_evidence_json, banned_at, banned_until, reason, banned_by)
- Indexes: (client_id), (workflow_type), (client_id, workflow_type)
- Implementation: all BanRepository methods
- Active ban check: `banned_until IS NULL OR banned_until > now()`

**SQLiteResultRepository** → ResultRepository port
- Table: results (run_id, client_id, status, inner_state_json, error_msg, payload_json, created_at)
- Implementation: `GetResult`, `GetRunResults`, `ListClientResults`

**SQLiteRunRepository** → RunRepository port (read-only)
- Reads existing runs table (created by WorkflowOrchestrationService)
- Implementation: `GetRun`, `GetPreviousRun`, `ListClientRuns`
- GetPreviousRun query: `SELECT * FROM runs WHERE client_id=? AND workflow_id=? AND created_at < ? ORDER BY created_at DESC LIMIT 1`

**SQLiteWorkflowRepository** → WorkflowRepository port (read-only)
- Reads existing workflows table
- Implementation: `GetWorkflow` to fetch and extract workflow_type

**StdoutAlertPublisher** → AlertPublisher port
- Logs alerts to stdout with severity levels
- Implementation: `PublishAlert`, `PublishBulkAlerts`
- (Can be replaced with Slack/email/webhook later)

**InMemoryDispatchBlocker** → DispatchBlocker port
- In-process cache of active bans (loaded from DB on startup)
- Receives updates via BanEnforcementService callbacks
- Implementation: `ShouldBlockDispatch`, `RegisterBanCheckCallback`
- Query: check in-memory cache before querying DB

### File Structure
```
internal/adapters/
  repository/
    sqlite_ban_repo.go
    sqlite_ban_repo_test.go
    sqlite_result_repo.go
    sqlite_result_repo_test.go
    sqlite_run_repo.go (read-only wrapper)
    sqlite_workflow_repo.go (read-only wrapper)
  alert/
    stdout_alert_publisher.go
    stdout_alert_publisher_test.go
  enforcement/
    inmemory_dispatch_blocker.go
    inmemory_dispatch_blocker_test.go
```

---

## Step 5: Binary/Deployment Layer

### Responsibilities
Wire together components, load configuration, run result-processing pipeline.

### Configuration
- Environment variables:
  - `DB_PATH`: SQLite database file
  - `LOOP_THRESHOLD_MS`: time window for loop detection (default: 5000 ms)
  - `ENABLE_PERMANENT_BAN`: bool, default true
  - `BAN_ESCALATION_COUNT`: number of loops before escalated alert (e.g., 3)
  - `BAN_ESCALATION_WINDOW_MS`: time window for escalation counting

### Initialization & Wiring

**cmd/main.go** (integrated into server)
```
1. Parse env variables for loop detection config
2. Initialize DB connection (ban, result, run, workflow repos)
3. Create port implementations (repos, alert publisher, blocker)
4. Create domain services (LoopDetector, BanManager)
5. Create orchestration services (LoopDetectionService, BanEnforcementService, DispatchFilterService)
6. Wire dependencies (services get ports as constructor params)
7. Register LoopDetectionService as a ResultHandler with ResultMessageDispatcher
   - Priority: 1 (runs before HealthMonitoring at priority 2)
8. Create in-memory dispatch blocker and warm cache with active bans from DB
9. Register dispatch blocker with DispatchCoordinationService
   - DispatchCoordinationService calls FilterDispatchList before sending
10. Setup graceful shutdown: finalize pending result processing, close connections
```

**cmd/config.go**
- `LoadConfig()`: read env vars, set defaults, validate
- Env vars: DB_PATH, LOOP_THRESHOLD_MS, ENABLE_PERMANENT_BAN, BAN_ESCALATION_COUNT, BAN_ESCALATION_WINDOW_MS, LOG_LEVEL

### File Structure
```
cmd/
  main.go          # loop detection service wiring (integrated)
  config.go        # config loading
```

### Runtime Behavior
1. On startup: load active bans from DB, warm InMemoryDispatchBlocker cache
2. On result message from client:
   - LoopDetectionService.ProcessResult() is called
   - Queries previous runs to check for loop
   - If loop detected: BanEnforcementService.BanClient()
   - Ban is saved to DB and published as alert
3. On new dispatch generation:
   - DispatchFilterService.FilterDispatchList() checks bans
   - Banned clients are excluded from dispatch
4. On admin unban request:
   - BanEnforcementService.UnbanClient() is called
   - Ban marked as inactive
   - Blocker cache invalidated
5. On shutdown: finalize any pending detections, close DB

### Integration Points
- **Driven by:** ResultMessageDispatcher (processes results with priority 1)
- **Reads:** RunRepository, ResultRepository, WorkflowRepository (run history)
- **Writes:** BanRepository (bans are immutable, new bans appended)
- **Publishes:** AlertPublisher (loop and ban events)
- **Blocks:** DispatchCoordinationService uses DispatchFilterService (filters dispatch lists before sending)
- **Notified by:** DispatchFilterService caches ban data from LoopDetectionService

---

## Implementation Dependencies

- **Depends on:** ResultRepository (results), RunRepository (run history), WorkflowRepository (workflow_type)
- **Used by:** DispatchCoordinationService (filters dispatch), HealthMonitoringService (excludes banned), API Service (query bans/unban)
- **Database:** SQLite for bans, results, runs, workflows
- **Messaging:** NATS for result subscriptions (driven by NATSMessageDispatcher)
- **Timing:** Critical path in result processing pipeline, must process quickly to prevent repeated loops
