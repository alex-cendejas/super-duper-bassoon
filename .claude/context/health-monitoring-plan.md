# Health Monitoring Service - Hexagonal Architecture Plan

## Overview
The Health Monitoring Service calculates per-run health metrics (success %, fail %, pending %) as results arrive, and aggregates health across recent runs per workflow_type. Health metrics are used by the Circuit Breaker Service to deactivate failing workflows, and by the API Service for observability.

## Step 1: Core/Domain Layer

### Responsibilities
Define pure business logic for health calculation and aggregation.

### Data Models & Methods

**RunHealth Domain Model** (calculated per-run)
- `RunID`: which run
- `WorkflowID`: which workflow
- `TotalClients`: count of clients participating in run
- `SuccessCount`: clients with status='success'
- `FailCount`: clients with status='fail'
- `ErrorCount`: clients with status='error'
- `PendingCount`: clients with no result yet
- `CompletedCount`: SuccessCount + FailCount + ErrorCount
- `CalculatedAt`: timestamp
- `BannedClientCount`: clients banned from this workflow_type (excluded from totals)
- Methods:
  - `SuccessPercentage()`: 100 * SuccessCount / (TotalClients - BannedClientCount)
  - `FailPercentage()`: 100 * FailCount / (TotalClients - BannedClientCount)
  - `ErrorPercentage()`: 100 * ErrorCount / (TotalClients - BannedClientCount)
  - `PendingPercentage()`: 100 * PendingCount / (TotalClients - BannedClientCount)
  - `IsComplete()`: PendingCount == 0
  - `Describe()`: human-readable health summary
  - `GetTrend()`: improving/degrading relative to previous run

**WorkflowTypeHealth Domain Model** (aggregated across runs)
- `WorkflowType`: workflow identifier (derived from workflow_id)
- `Window`: last n runs to aggregate
- `SuccessPercentageAvg`: average success % across window
- `FailPercentageAvg`: average fail % across window
- `ErrorPercentageAvg`: average error % across window
- `SuccessPercentageTrend`: increasing/decreasing over window
- `Runs`: list of RunHealth within window (chronological)
- `CalculatedAt`: timestamp
- Methods:
  - `IsHealthy(threshold)`: SuccessPercentageAvg >= threshold
  - `IsHealthy()`: uses default threshold from workflow config
  - `GetTrendDirection()`: is success trend positive
  - `GetWorstRun()`: run with lowest success %
  - `Describe()`: summary

**HealthMetric Domain Model** (publishable event)
- `WorkflowID`/`WorkflowType`: which workflow
- `RunID`: specific run (if run-level) or null (if aggregated)
- `MetricType`: enum (run_health, workflow_health)
- `Health`: the RunHealth or WorkflowTypeHealth value
- `PublishedAt`: timestamp
- Methods:
  - `Serialize()`: to JSON
  - `Describe()`: human-readable

**HealthThreshold Domain Model**
- `SuccessThreshold`: minimum success % to consider workflow healthy
- `WindowSize`: number of runs to aggregate (e.g., last 10)
- Methods:
  - `Validate()`: check ranges
  - `IsMetForWorkflow(workflowHealth)`: boolean decision

**HealthAggregator Domain Model** (stateless aggregation logic)
- Methods:
  - `CalculateRunHealth(runID, totalClients, results []*Result, bannedCount) *RunHealth`:
    - Count results by status
    - Exclude banned clients from denominators
    - Return calculated health
  - `AggregateWorkflowHealth(workflowType, []*RunHealth, windowSize) *WorkflowTypeHealth`:
    - Average success % across window
    - Calculate trend (compare recent runs to older)
    - Return aggregated health
  - `CalculateTrend(current, previous) TrendDirection`: improving/degrading/stable

### File Structure
```
internal/core/domain/
  run_health.go              # RunHealth model
  run_health_test.go
  workflow_health.go         # WorkflowTypeHealth model
  workflow_health_test.go
  health_metric.go           # HealthMetric (publishable event)
  health_metric_test.go
  health_threshold.go        # HealthThreshold policy
  health_threshold_test.go
  health_aggregator.go       # Aggregation logic (stateless)
  health_aggregator_test.go
  errors.go                  # domain-specific errors
```

---

## Step 2: Core/Ports Layer

### Responsibilities
Define generic interfaces for data access and health publication.

### Port Interfaces

**RunRepository Port** (read-only)
- `GetRun(ctx, runID) (*Run, error)`: fetch run details
- `ListRuns(ctx, workflowID, limit, orderBy) ([]*Run, error)`: ordered run history
- `ListRecentRuns(ctx, workflowType, since time.Time) ([]*Run, error)`: runs in time window

**ResultRepository Port** (read-only)
- `GetRunResults(ctx, runID) ([]*Result, error)`: all results for a run
- `GetRunResultsByStatus(ctx, runID, status) ([]*Result, error)`: filtered by outcome

**BanRepository Port** (read-only)
- `GetActiveBans(ctx, workflowType) ([]*BanRecord, error)`: count of banned clients per workflow_type
- `GetBansForRun(ctx, runID) ([]*BanRecord, error)`: bans relevant to a run

**WorkflowRepository Port** (read-only)
- `GetWorkflow(ctx, workflowID) (*Workflow, error)`: fetch for success_threshold

**HealthRepository Port** (write)
- `SaveRunHealth(ctx, *RunHealth) error`: persist calculated health
- `GetRunHealth(ctx, runID) (*RunHealth, error)`: fetch cached health
- `SaveWorkflowTypeHealth(ctx, *WorkflowTypeHealth) error`: persist aggregated health
- `GetWorkflowTypeHealth(ctx, workflowType) (*WorkflowTypeHealth, error)`: fetch latest aggregation
- `ListWorkflowTypeHealthHistory(ctx, workflowType, limit) ([]*WorkflowTypeHealth, error)`: trend data

**EventPublisher Port** (publish health events)
- `PublishHealthUpdatedEvent(ctx, *HealthUpdatedEvent) error`: emit event when health changes
  - Event includes WorkflowType, RunHealth, WorkflowTypeHealth
  - Used by CircuitBreakerService to react to health changes

**ConfigRepository Port** (read-only, for thresholds)
- `GetHealthThreshold(ctx, workflowType) (*HealthThreshold, error)`: fetch success_threshold, window_size

### File Structure
```
internal/core/ports/
  run_repository.go          # RunRepository interface
  result_repository.go       # ResultRepository interface
  ban_repository.go          # BanRepository interface (read-only)
  workflow_repository.go     # WorkflowRepository interface (read-only)
  health_repository.go       # HealthRepository interface
  metrics_publisher.go       # MetricsPublisher interface
  config_repository.go       # ConfigRepository interface
```

---

## Step 3: Core/Services Layer

### Responsibilities
Orchestrate domain logic using ports to implement health monitoring.

### Services

**HealthMonitoringService** (main service)
- Dependencies: RunRepository, ResultRepository, BanRepository, HealthRepository, EventPublisher, ConfigRepository, HealthAggregator
- Methods:
  - `CalculateRunHealth(ctx, runID) (*RunHealth, error)`: main entrypoint
    - Fetch run from RunRepository
    - Fetch results from ResultRepository
    - Fetch banned client count from BanRepository
    - Call HealthAggregator.CalculateRunHealth()
    - Save to HealthRepository
    - Publish HealthUpdatedEvent via EventPublisher
    - Return health
  - `AggregateWorkflowTypeHealth(ctx, workflowType) (*WorkflowTypeHealth, error)`:
    - Fetch recent runs for workflow_type from RunRepository
    - Fetch cached run health values from HealthRepository
    - Fetch threshold config from ConfigRepository
    - Call HealthAggregator.AggregateWorkflowHealth()
    - Save to HealthRepository
    - Publish HealthUpdatedEvent via EventPublisher
    - Return aggregated health
  - `OnRunCompletion(ctx, runID)`: callback from orchestration service
    - Trigger CalculateRunHealth()
    - Trigger AggregateWorkflowTypeHealth() for the run's workflow_type
  - `OnResultReceived(ctx, runID, result *Result)`: handler from ResultMessageDispatcher
    - Recalculate run health (results changed)
    - Publish HealthUpdatedEvent if health changed
  - `GetCurrentHealth(ctx, workflowType) (*WorkflowTypeHealth, error)`: query interface
    - Fetch latest aggregated health from HealthRepository
    - Return (may be cached)

**HealthAggregationService** (updates aggregations)
- Dependencies: HealthRepository, ConfigRepository, HealthAggregator
- Methods:
  - `UpdateAllWorkflowHealths(ctx) error`: periodic update all workflows
    - For each active workflow_type:
      - Call HealthMonitoringService.AggregateWorkflowTypeHealth()
  - `OnNewRun(ctx, runID, workflowType)`: reactive update
    - Trigger aggregation for that workflow_type
  - `ComputeHealthTrend(ctx, workflowType) (TrendDirection, error)`:
    - Compare recent health to historical health
    - Return trend assessment

### File Structure
```
internal/core/services/
  health_monitoring.go       # HealthMonitoringService
  health_monitoring_test.go
  health_aggregation.go      # HealthAggregationService
  health_aggregation_test.go
```

---

## Step 4: Adapters Layer

### Responsibilities
Implement the port interfaces with concrete technologies.

### Adapter Implementations

**SQLiteRunRepository** → RunRepository port (read-only)
- Reads existing runs table
- Implementation: `GetRun`, `ListRuns`, `ListRecentRuns`
- OrderBy: created_at DESC (most recent first)

**SQLiteResultRepository** → ResultRepository port (read-only)
- Reads existing results table
- Implementation: `GetRunResults`, `GetRunResultsByStatus`
- Index: (run_id) for fast lookups

**SQLiteBanRepository** → BanRepository port (read-only)
- Reads existing bans table
- Implementation: `GetActiveBans` (count per workflow_type), `GetBansForRun`
- Filters by active: `banned_until IS NULL OR banned_until > now()`

**SQLiteWorkflowRepository** → WorkflowRepository port (read-only)
- Reads existing workflows table
- Implementation: `GetWorkflow` to fetch thresholds

**SQLiteHealthRepository** → HealthRepository port
- Tables: 
  - run_health (run_id, workflow_id, total_clients, success_count, fail_count, error_count, pending_count, banned_count, calculated_at)
  - workflow_type_health (workflow_type, success_pct_avg, fail_pct_avg, error_pct_avg, trend, calculated_at)
- Indexes: (run_id), (workflow_type)
- Implementation: all HealthRepository methods

**InMemoryEventPublisher** → EventPublisher port
- Pub/sub event bus for inter-service communication
- Events: HealthUpdatedEvent (includes workflow_type, health metrics)
- Implementation: `PublishHealthUpdatedEvent`
- Subscribers: CircuitBreakerService (listens for health changes)

**DefaultConfigRepository** → ConfigRepository port
- Reads from Workflow definitions (success_threshold, loop_threshold)
- Implementation: `GetHealthThreshold`
- Fallback defaults: success_threshold=80%, window_size=10

### File Structure
```
internal/adapters/
  repository/
    sqlite_health_repo.go
    sqlite_health_repo_test.go
    sqlite_run_repo.go (read-only wrapper)
    sqlite_result_repo.go (read-only wrapper)
    sqlite_ban_repo.go (read-only wrapper)
    sqlite_workflow_repo.go (read-only wrapper)
  metrics/
    prometheus_publisher.go
    prometheus_publisher_test.go
  config/
    default_config_repo.go
    default_config_repo_test.go
```

---

## Step 5: Binary/Deployment Layer

### Responsibilities
Wire together components, load configuration, run health monitoring loop.

### Configuration
- Environment variables:
  - `DB_PATH`: SQLite database file
  - `HEALTH_AGGREGATION_INTERVAL_MS`: how often to re-aggregate (e.g., 5000 ms)
  - `HEALTH_WINDOW_SIZE`: number of recent runs to aggregate (e.g., 10)
  - `HEALTH_SUCCESS_THRESHOLD`: default success % threshold (e.g., 80)
  - `METRICS_ENABLED`: bool, enable metrics publishing
  - `METRICS_EXPORT_PATH`: file path or endpoint for metrics

### Initialization & Wiring

**cmd/main.go** (integrated into server)
```
1. Parse env variables for health monitoring config
2. Initialize DB connection (run, result, ban, health, workflow repos)
3. Create port implementations (repos, metrics publisher, config repo)
4. Create domain services (HealthAggregator)
5. Create orchestration services (HealthMonitoringService, HealthAggregationService)
6. Wire dependencies (services get ports as constructor params)
7. Subscribe to run completion events (from WorkflowOrchestrationService)
   - Call HealthMonitoringService.CalculateRunHealth() on completion
   - Call HealthMonitoringService.AggregateWorkflowTypeHealth()
8. Subscribe to result messages (from result processing pipeline)
   - Call HealthMonitoringService.OnResultReceived() for incremental updates
9. Start health aggregation goroutine:
   - Tick at configured interval
   - Call HealthAggregationService.UpdateAllWorkflowHealths()
   - Publish metrics
10. Setup graceful shutdown: finalize pending calculations, close connections
```

**cmd/config.go**
- `LoadConfig()`: read env vars, set defaults, validate
- Env vars: DB_PATH, HEALTH_AGGREGATION_INTERVAL_MS, HEALTH_WINDOW_SIZE, HEALTH_SUCCESS_THRESHOLD, METRICS_ENABLED, METRICS_EXPORT_PATH, LOG_LEVEL

### File Structure
```
cmd/
  main.go          # health monitoring service wiring (integrated)
  config.go        # config loading
```

### Runtime Behavior
1. On startup: create health repository tables, initialize config
2. On run completion event:
   - HealthMonitoringService.CalculateRunHealth() is called
   - Run health is calculated and saved to DB
   - RunHealth metric is published
   - HealthAggregationService aggregates workflow_type health
   - WorkflowTypeHealth metric is published
3. On result message (incremental):
   - HealthMonitoringService.OnResultReceived() is called
   - Run health is recalculated (new result arrived)
   - Health metric is re-published (health changed)
4. Periodic (at configured interval):
   - HealthAggregationService.UpdateAllWorkflowHealths() is called
   - All workflow_type health values are recalculated
   - Metrics are published to metrics sink
5. On API request:
   - APIService calls HealthMonitoringService.GetCurrentHealth()
   - Returns cached/latest aggregated health
6. On shutdown: finalize pending health calculations, close DB

### Integration Points
- **Receives:** Result messages from ResultMessageDispatcher (incremental health updates)
- **Reads:** RunRepository, ResultRepository, BanRepository, WorkflowRepository
- **Writes:** HealthRepository (persists all health calculations)
- **Publishes:** EventPublisher → HealthUpdatedEvent
- **Notified by:** CircuitBreakerService subscribes to HealthUpdatedEvent
- **Serves:** API Service (queries for health data)

---

## Implementation Dependencies

- **Depends on:** RunRepository (run details), ResultRepository (results), BanRepository (banned client counts), WorkflowRepository (thresholds), ConfigRepository (window size, success threshold)
- **Used by:** CircuitBreakerService (monitors health for deactivation), API Service (query health), MetricsPublisher (observability)
- **Database:** SQLite for run_health, workflow_type_health, and existing runs/results/bans
- **No external messaging:** reads from internal event streams, publishes metrics independently
- **Real-time:** health is calculated incremental as results arrive, and re-aggregated on periodic tick
