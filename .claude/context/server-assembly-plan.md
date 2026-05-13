# Server Assembly Plan - All-in-One Application

This document specifies how to assemble all 6 services (API Service, Health Monitoring, Circuit Breaker, Loop Detection & Ban, Workflow Orchestration, Dynamic Grouping) into a single production-ready server application with correct initialization order, dependency wiring, and shutdown coordination.

---

## Architecture Overview

```
┌────────────────────────────────────────────────────────────────┐
│                      HTTP Server (chi)                         │
│  ├─ API Service Handlers (read-only REST API)                │
│  └─ Middleware (logging, error handling, CORS)               │
└────────────────────────────────────────────────────────────────┘
                              ↑
                         (queries)
┌────────────────────────────────────────────────────────────────┐
│                    Service Layer                              │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ WorkflowOrchestrationService                            │  │
│  │  ├─ DynamicGroupingService (filter evaluation)         │  │
│  │  ├─ DispatchCoordinationService                        │  │
│  │  │  └─ DispatchFilterService (ban filtering)           │  │
│  │  │     └─ BanEnforcementService                        │  │
│  │  └─ TriggerCoordinationService (trigger evaluation)    │  │
│  └─────────────────────────────────────────────────────────┘  │
│                              ↓                                  │
│                    (dispatches via NATS)                       │
│                              ↓                                  │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ ResultMessageDispatcher (routes result messages)        │  │
│  │  ├─ LoopDetectionService (priority 1)                  │  │
│  │  │  └─ BanEnforcementService (applies bans)            │  │
│  │  └─ HealthMonitoringService (priority 2)               │  │
│  │     └─ publishes HealthUpdatedEvent via EventBus       │  │
│  └─────────────────────────────────────────────────────────┘  │
│                              ↓                                  │
│                        (EventBus)                              │
│                              ↓                                  │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │ CircuitBreakerService                                   │  │
│  │  ├─ Subscribes to HealthUpdatedEvent (reactive)        │  │
│  │  └─ Periodic evaluation (fallback)                     │  │
│  └─────────────────────────────────────────────────────────┘  │
│                              ↓                                  │
│              (deactivates/activates workflows)                 │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
                              ↓
                    ┌──────────────────┐
                    │   Repositories   │
                    ├──────────────────┤
                    │  Workflow (RW)   │
                    │  Client (RO)     │
                    │  Run (RW)        │
                    │  Result (RW)     │
                    │  Ban (RW)        │
                    │  Health (RW)     │
                    │  CircuitBreaker (RW)
                    └──────────────────┘
                              ↓
                        ┌──────────┐
                        │ SQLite   │
                        │ Database │
                        └──────────┘
                              ↑
                        ┌──────────────┐
                        │ NATS Broker  │
                        │ (messaging)  │
                        └──────────────┘
```

---

## Initialization Sequence

### Phase 1: Foundation (Sequential - dependencies are required)

**1.1 Configure Logging**
```go
// cmd/main.go - Initialize at entry point
logLevel := os.Getenv("LOG_LEVEL") // "debug", "info", "warn", "error"
logger := initializeLogger(logLevel)
```

**1.2 Load Configuration**
```go
// cmd/config.go
config, err := LoadConfig()
if err != nil {
    logger.Fatal("Failed to load config", err)
}
// Validates all required env vars, sets defaults
```

**1.3 Initialize Database**
```go
// cmd/main.go
db, err := sql.Open("sqlite", config.DBPath)
if err != nil {
    logger.Fatal("Failed to open database", err)
}
defer db.Close()

// Run migrations
migrations := LoadMigrations()
err = RunMigrations(db, migrations)
if err != nil {
    logger.Fatal("Failed to run migrations", err)
}

// Configure connection pool
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Minute * 5)

// Enable WAL mode for concurrency
_, err = db.Exec("PRAGMA journal_mode=WAL")
```

**1.4 Initialize NATS Connection**
```go
// cmd/main.go
natsConn, err := nats.Connect(config.NatsURL)
if err != nil {
    logger.Fatal("Failed to connect to NATS", err)
}
defer natsConn.Close()

// Configure connection options
natsConn.SetErrorHandler(func(conn *nats.Conn, sub *nats.Subscription, err error) {
    logger.Error("NATS error", err)
})
```

### Phase 2: Repository & Shared Services (Sequential - repos need DB, others need repos)

**2.1 Create RepositoryRegistry**
```go
// internal/core/services/repository_registry.go
registry := NewRepositoryRegistry(db)

// Registry provides:
workflowRepo := registry.WorkflowRepository()      // RW
clientRepo := registry.ClientRepository()          // RO
runRepo := registry.RunRepository()                // RW
resultRepo := registry.ResultRepository()          // RW
banRepo := registry.BanRepository()                // RW
healthRepo := registry.HealthRepository()          // RW
circuitRepo := registry.CircuitBreakerStateRepository() // RW

// Validate database connection
err := registry.HealthCheck(ctx)
if err != nil {
    logger.Fatal("Database health check failed", err)
}
```

**2.2 Create EventBus (for inter-service communication)**
```go
// internal/adapters/messaging/inmemory_event_bus.go
eventBus := NewInMemoryEventBus(logger)

// RegisterEventTypes
eventBus.RegisterEventType("health.updated", func() Event { return &HealthUpdatedEvent{} })
eventBus.RegisterEventType("circuit.state.changed", func() Event { return &CircuitBreakerStateChangedEvent{} })
eventBus.RegisterEventType("workflow.completed", func() Event { return &WorkflowCompletionEvent{} })
```

**2.3 Create NATSMessageDispatcher (for client messaging)**
```go
// internal/adapters/messaging/nats_dispatcher.go
natsDispatcher := NewNATSMessageDispatcher(natsConn, logger)

// Configure result channels
resultChan := natsDispatcher.SubscribeToResults(ctx, "result.>")
```

**2.4 Create Graceful Shutdown Manager**
```go
// internal/core/services/shutdown_manager.go
shutdownMgr := NewGracefulShutdownManager(logger, config.ShutdownTimeoutMS)

// Will be used to coordinate shutdown of all services
// defer shutdownMgr.HandleShutdown() at end of main()
```

### Phase 3: Domain Services (No Dependencies)

**3.1-3.7 Create Stateless Domain Services**
```go
// All in internal/core/services or internal/adapters

healthAggregator := NewHealthAggregator(logger)
circuitBreakerLogic := NewCircuitBreakerLogic(logger)
loopDetector := NewLoopDetector(logger, config.LoopThresholdMS)
banManager := NewBanManager(logger)
filterEvaluationService := NewFilterEvaluationService(logger)
fieldResolver := NewFieldResolver(logger)
```

### Phase 4: Core Application Services (Sequential - dependencies on repos)

**4.1 DynamicGroupingService** (reads clients)
```go
dynamicGrouping := NewDynamicGroupingService(
    clientRepo,
    filterEvaluationService,
    fieldResolver,
    logger,
)
```

**4.2 LoopDetectionService** (reads/writes bans, reads runs)
```go
loopDetection := NewLoopDetectionService(
    resultRepo,
    runRepo,
    workflowRepo,
    loopDetector,
    banRepo,
    nil, // AlertPublisher (created next)
    logger,
)
```

**4.3 BanEnforcementService** (manages bans, publishes alerts)
```go
alertPublisher := NewStdoutAlertPublisher(logger) // can replace later

banEnforcement := NewBanEnforcementService(
    banManager,
    banRepo,
    alertPublisher,
    logger,
)

// Link to LoopDetectionService
loopDetection.SetBanEnforcementService(banEnforcement)
```

**4.4 DispatchFilterService** (filters banned clients)
```go
inMemoryDispatchBlocker := NewInMemoryDispatchBlocker(logger)

// Warm cache with active bans from DB
activeBans, err := banRepo.ListAllBans(ctx)
if err != nil {
    logger.Error("Failed to load active bans", err)
}
for _, ban := range activeBans {
    if ban.IsActive() {
        inMemoryDispatchBlocker.Add(ban)
    }
}

dispatchFilter := NewDispatchFilterService(
    banEnforcement,
    inMemoryDispatchBlocker,
    logger,
)
```

**4.5 DispatchCoordinationService** (generates and sends dispatches)
```go
dispatchCoordination := NewDispatchCoordinationService(
    runRepo,
    natsDispatcher,
    clientRepo,
    dispatchFilter,
    logger,
)
```

**4.6 HealthMonitoringService** (calculates and publishes health)
```go
configRepo := NewDefaultConfigRepository(
    workflowRepo,
    config.HealthSuccessThreshold,
    config.HealthWindowSize,
)

healthMonitoring := NewHealthMonitoringService(
    runRepo,
    resultRepo,
    banRepo,
    healthRepo,
    eventBus, // EventPublisher (publishes HealthUpdatedEvent)
    configRepo,
    healthAggregator,
    logger,
)
```

**4.7 CircuitBreakerService** (monitors health, deactivates workflows)**
```go
policyRepo := NewDefaultPolicyRepository(
    workflowRepo,
    config.CircuitBreakerSuccessThreshold,
    config.CircuitBreakerCooldownMS,
)

workflowStateManager := NewWorkflowRepositoryStateManager(
    workflowRepo,
    alertPublisher,
    logger,
)

circuitBreaker := NewCircuitBreakerService(
    healthRepo,
    circuitRepo,
    policyRepo,
    alertPublisher,
    workflowStateManager,
    circuitBreakerLogic,
    eventBus, // EventPublisher (subscribes to HealthUpdatedEvent)
    logger,
)

// Subscribe to health updates
eventBus.Subscribe("health.updated", circuitBreaker.OnHealthUpdatedEvent)
```

**4.8 WorkflowOrchestrationService** (main orchestrator)
```go
workflowOrchestration := NewWorkflowOrchestrationService(
    workflowRepo,
    clientRepo,
    natsDispatcher,
    runRepo,
    dynamicGrouping,
    eventBus, // EventPublisher (publishes WorkflowCompletionEvent)
    logger,
)
```

**4.9 TriggerCoordinationService** (evaluates and fires triggers)**
```go
triggerEvaluator := NewCronAndEventTriggerEvaluator(
    config.TriggerCheckIntervalMS,
    logger,
)

triggerCoordination := NewTriggerCoordinationService(
    triggerEvaluator,
    workflowOrchestration,
    eventBus, // EventPublisher
    logger,
)
```

### Phase 5: Routing & Message Handling (Sequential - needs all services)

**5.1 Create ResultMessageDispatcher**
```go
resultDispatcher := NewResultMessageDispatcher(logger)

// Register handlers in priority order
resultDispatcher.RegisterHandler(loopDetection, 1)      // Priority 1: ban detection
resultDispatcher.RegisterHandler(healthMonitoring, 2)   // Priority 2: health calculation

// Start consuming NATS result messages
go resultDispatcher.Start(ctx, resultChan)
shutdownMgr.RegisterStopFunc("result_dispatcher", resultDispatcher.Stop)
```

**5.2 Create API Service (reads from all services)**
```go
apiHandler := NewAPIHandlerService(
    workflowOrchestration,
    clientRepo,
    runRepo,
    healthMonitoring,
    banEnforcement,
    circuitBreaker,
    alertPublisher, // for system status
    logger,
)

requestValidation := NewRequestValidationService(logger)
responseFormatting := NewResponseFormattingService(logger)
```

**5.3 Create HTTP Server**
```go
httpServer := chi.NewRouter()

// Middleware
httpServer.Use(middleware.Logger(logger))
httpServer.Use(middleware.Recoverer())
httpServer.Use(middleware.RequestID)

// Workflow routes
httpServer.Post("/workflows", apiHandler.HandleCreateWorkflow)
httpServer.Get("/workflows", apiHandler.HandleListWorkflows)
httpServer.Get("/workflows/{id}", apiHandler.HandleGetWorkflow)
httpServer.Put("/workflows/{id}", apiHandler.HandleEditWorkflow)
httpServer.Delete("/workflows/{id}", apiHandler.HandleDeleteWorkflow)
httpServer.Post("/workflows/{id}/trigger", apiHandler.HandleTriggerWorkflow)
httpServer.Post("/workflows/{id}/activate", apiHandler.HandleActivateWorkflow)
httpServer.Post("/workflows/{id}/deactivate", apiHandler.HandleDeactivateWorkflow)

// Client routes
httpServer.Get("/clients", apiHandler.HandleListClients)
httpServer.Get("/clients/{id}", apiHandler.HandleGetClient)

// Run routes
httpServer.Get("/runs/{id}", apiHandler.HandleGetRun)
httpServer.Get("/workflows/{id}/runs", apiHandler.HandleListRuns)

// Health routes
httpServer.Get("/health", apiHandler.HandleListAllHealth)
httpServer.Get("/health/{workflow_type}", apiHandler.HandleGetHealth)

// Ban routes
httpServer.Get("/bans", apiHandler.HandleListAllBans)
httpServer.Get("/bans/{client_id}", apiHandler.HandleGetBans)
httpServer.Put("/bans/{client_id}/unban", apiHandler.HandleUnbanClient)

// Circuit breaker routes
httpServer.Get("/circuits/{workflow_id}", apiHandler.HandleGetCircuitState)
httpServer.Get("/circuits", apiHandler.HandleListAllCircuitStates)

// System routes
httpServer.Get("/status", apiHandler.HandleGetSystemStatus)
httpServer.Get("/health/liveness", apiHandler.HandleLivenessProbe)
httpServer.Get("/health/readiness", apiHandler.HandleReadinessProbe)
```

### Phase 6: Start Background Goroutines (Parallel - no dependencies)

**6.1 Start Trigger Coordination** (evaluates triggers continuously)
```go
go func() {
    err := triggerCoordination.Start(ctx)
    if err != nil {
        logger.Error("Trigger coordination stopped", err)
    }
}()

shutdownMgr.RegisterStopFunc("trigger_coordination", triggerCoordination.Stop)
```

**6.2 Start Health Aggregation** (periodic health recalculation)
```go
go func() {
    ticker := time.NewTicker(time.Duration(config.HealthAggregationIntervalMS) * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Recalculate all workflow health
            workflows, err := workflowRepo.ListActiveWorkflows(ctx)
            if err != nil {
                logger.Error("Failed to list workflows", err)
                continue
            }
            
            for _, wf := range workflows {
                _, err := healthMonitoring.AggregateWorkflowTypeHealth(ctx, wf.WorkflowType())
                if err != nil {
                    logger.Error("Failed to aggregate health", err)
                }
            }
        }
    }
}()

shutdownMgr.RegisterStopFunc("health_aggregation", func(ctx context.Context) error {
    // Ticker stopped above
    return nil
})
```

**6.3 Start Circuit Breaker Periodic Evaluation** (fallback evaluation)
```go
go func() {
    ticker := time.NewTicker(time.Duration(config.CircuitBreakerCheckIntervalMS) * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            err := circuitBreaker.EvaluateAllWorkflows(ctx)
            if err != nil {
                logger.Error("Circuit breaker evaluation failed", err)
            }
        }
    }
}()

shutdownMgr.RegisterStopFunc("circuit_breaker_evaluation", func(ctx context.Context) error {
    // Ticker stopped above
    return nil
})
```

### Phase 7: Start HTTP Server

**7.1 Create Listener**
```go
addr := fmt.Sprintf("%s:%d", config.HTTPHost, config.HTTPPort)
listener, err := net.Listen("tcp", addr)
if err != nil {
    logger.Fatal("Failed to create listener", err)
}

logger.Info("Server listening", map[string]interface{}{
    "host": config.HTTPHost,
    "port": config.HTTPPort,
})
```

**7.2 Start Server in Goroutine**
```go
go func() {
    // Configure HTTP server with timeouts
    server := &http.Server{
        Addr:           addr,
        Handler:        httpServer,
        ReadTimeout:    time.Duration(config.HTTPReadTimeoutMS) * time.Millisecond,
        WriteTimeout:   time.Duration(config.HTTPReadTimeoutMS) * time.Millisecond,
        MaxHeaderBytes: 1 << 20, // 1MB
    }
    
    err := server.Serve(listener)
    if err != nil && err != http.ErrServerClosed {
        logger.Error("HTTP server error", err)
    }
}()

shutdownMgr.RegisterHTTPServer(listener)
```

### Phase 8: Wait for Shutdown Signal

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

logger.Info("Server started, waiting for shutdown signal...")
<-sigChan
logger.Info("Received shutdown signal, starting graceful shutdown...")
```

---

## Graceful Shutdown Sequence

**Coordinated by GracefulShutdownManager**

```
1. Stop accepting new HTTP connections
   - Close listener
   - HTTP handler returns 503 Service Unavailable

2. Drain in-flight HTTP requests (30 second timeout)
   - Each HTTP handler completes or times out
   - No new goroutines spawned

3. Stop background goroutines (reverse init order)
   a. TriggerCoordinationService.Stop()
      - Stop trigger evaluation loop
      - Drain any pending triggers
   
   b. CircuitBreakerService.Stop()
      - Unsubscribe from EventBus
      - Finalize pending evaluations
   
   c. ResultMessageDispatcher.Stop()
      - Stop consuming NATS messages
      - Unsubscribe from NATS
      - Wait for pending result processing
   
   d. HealthMonitoringService.Stop()
      - Finalize pending health calculations
      - Save any pending health records

   e. BanEnforcementService.Stop()
      - Flush any pending ban records
      - Close connections

4. Close EventBus
   - Unsubscribe all handlers
   - Drain any pending events

5. Close Database Connection
   - Commit any pending transactions
   - WAL checkpoint
   - Close connection pool

6. Close NATS Connection
   - Unsubscribe from all subscriptions
   - Close connection

7. Exit with code 0
```

---

## Complete Wiring Example (Pseudocode)

```go
// cmd/main.go

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Phase 1: Foundation
    logger := initLogger(os.Getenv("LOG_LEVEL"))
    config := loadConfig()
    db := initDB(config.DBPath)
    defer db.Close()
    natsConn := initNATS(config.NatsURL)
    defer natsConn.Close()
    
    // Phase 2: Shared Services
    registry := NewRepositoryRegistry(db)
    eventBus := NewInMemoryEventBus(logger)
    natsDispatcher := NewNATSMessageDispatcher(natsConn, logger)
    resultChan := natsDispatcher.SubscribeToResults(ctx, "result.>")
    shutdownMgr := NewGracefulShutdownManager(logger, config.ShutdownTimeoutMS)
    
    // Phase 3: Domain Services (stateless)
    healthAggregator := NewHealthAggregator(logger)
    // ... other domain services
    
    // Phase 4: Application Services (sequential)
    dynamicGrouping := NewDynamicGroupingService(...)
    loopDetection := NewLoopDetectionService(...)
    banEnforcement := NewBanEnforcementService(...)
    dispatchFilter := NewDispatchFilterService(...)
    dispatchCoordination := NewDispatchCoordinationService(...)
    healthMonitoring := NewHealthMonitoringService(...)
    circuitBreaker := NewCircuitBreakerService(...)
    workflowOrchestration := NewWorkflowOrchestrationService(...)
    triggerCoordination := NewTriggerCoordinationService(...)
    
    // Phase 5: Routing
    resultDispatcher := NewResultMessageDispatcher(logger)
    resultDispatcher.RegisterHandler(loopDetection, 1)
    resultDispatcher.RegisterHandler(healthMonitoring, 2)
    go resultDispatcher.Start(ctx, resultChan)
    
    apiHandler := NewAPIHandlerService(...)
    httpServer := setupHTTPRoutes(apiHandler)
    
    // Phase 6: Start Background Goroutines
    go triggerCoordination.Start(ctx)
    go runHealthAggregationLoop(ctx, healthMonitoring, ...)
    go runCircuitBreakerLoop(ctx, circuitBreaker, ...)
    
    // Phase 7: Start HTTP Server
    listener, _ := net.Listen("tcp", fmt.Sprintf("%s:%d", config.HTTPHost, config.HTTPPort))
    go func() {
        http.Serve(listener, httpServer)
    }()
    
    // Phase 8: Wait for Shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    // Graceful Shutdown
    shutdownMgr.GracefulShutdown(ctx)
}
```

---

## Testing Strategy

### Unit Tests
- All domain models and logic (health aggregation, loop detection, filter evaluation)
- All service methods with mocked ports
- All repository adapters with in-memory implementations

### Integration Tests
- RepositoryRegistry + SQLite (in-memory database for speed)
- EventBus pub/sub with multiple subscribers
- ResultMessageDispatcher with multiple handlers
- Full workflow: trigger → dispatch → result → health → circuit

### E2E Tests (Optional)
- Real SQLite database + NATS broker
- Trigger workflow → receive dispatch → send result → verify health/circuit
- Ban flow: detect loop → ban client → filter dispatch → verify exclusion

### Load Tests
- ResultMessageDispatcher throughput (messages/sec)
- Database write throughput (health + bans)
- HTTP API latency under load

---

## Configuration Validation

At startup (in LoadConfig):

```
Required Environment Variables:
  ✓ DB_PATH - SQLite database file path
  ✓ NATS_URL - NATS broker connection URL
  
Optional (with defaults):
  ✓ HTTP_HOST (default: 0.0.0.0)
  ✓ HTTP_PORT (default: 8080)
  ✓ HTTP_READ_TIMEOUT_MS (default: 30000)
  ✓ TRIGGER_CHECK_INTERVAL_MS (default: 5000)
  ✓ HEALTH_AGGREGATION_INTERVAL_MS (default: 5000)
  ✓ HEALTH_WINDOW_SIZE (default: 10)
  ✓ HEALTH_SUCCESS_THRESHOLD (default: 80)
  ✓ CIRCUIT_BREAKER_CHECK_INTERVAL_MS (default: 10000)
  ✓ CIRCUIT_BREAKER_SUCCESS_THRESHOLD (default: 80)
  ✓ CIRCUIT_BREAKER_COOLDOWN_MS (default: 300000)
  ✓ LOOP_THRESHOLD_MS (default: 5000)
  ✓ LOG_LEVEL (default: info)
  ✓ SHUTDOWN_TIMEOUT_MS (default: 30000)

Validation:
  - HTTP_PORT is valid (1-65535)
  - NATS_URL is resolvable
  - DB_PATH directory exists and is writable
  - All numeric values are non-negative
  - LOG_LEVEL is one of: debug, info, warn, error
```

---

## Monitoring & Observability

### Metrics to Expose

**HTTP Endpoints:**
- `GET /status` → System status (uptime, connection counts, goroutine count)
- `GET /health/liveness` → Is server alive? (200 OK if yes)
- `GET /health/readiness` → Is server ready to serve? (200 OK if all services are ready)

**Logs:**
- All service initialization (debug level)
- Trigger evaluations (debug level)
- Workflow triggers and dispatches (info level)
- Result processing (debug level)
- Health calculations (debug level)
- Circuit breaker state changes (info level)
- Ban operations (info level)
- Errors and panics (error level)

**Optional (Future):**
- Prometheus metrics endpoint
- Distributed tracing (OpenTelemetry)
- Request performance profiling

---

## Security Considerations

### Database
- SQLite WAL mode for ACID guarantees
- Parameterized queries (prevent SQL injection)
- Connection pooling (prevent resource exhaustion)

### HTTP API
- All inputs validated before processing
- Error responses don't leak internal details
- No sensitive data in logs (e.g., ban reasons are logged but not full state)
- CORS headers configured (if multi-origin support needed)

### NATS Messaging
- No authentication configured (assumes private network)
- Consider adding TLS in production
- Consider adding NATS authentication/authorization

### Secrets Management
- Database path from environment variable
- NATS URL from environment variable
- (Consider Vault or similar for production)

---

## Performance Tuning Notes

### Database
- WAL mode enables concurrent reads
- Journal size can grow; consider periodic cleanup
- Index on (run_id, client_id) for result lookups
- Index on (workflow_type) for health lookups

### NATS
- Batch size for result messages (e.g., process 100 at a time)
- Consider queue groups for result consumption (if scaling to multiple servers)

### Memory
- EventBus events are in-memory (unbounded queue if not careful)
- Consider bounded channel for event bus
- HealthMonitoringService caches recent health (configurable window)

### CPU
- FilterEvaluationService: complex filter expressions can be slow
- CircuitBreakerLogic: O(n) evaluation per workflow type
- LoopDetectionService: O(m) lookback in run history

---

## Deployment Checklist

Before deploying to production:

- [ ] All 6 services have been integrated and tested
- [ ] Database schema is initialized (migrations run)
- [ ] NATS broker is accessible and responding
- [ ] All environment variables are set correctly
- [ ] SSL/TLS certificates are configured (if needed)
- [ ] Firewall rules allow HTTP traffic on configured port
- [ ] NATS is on private network or has authentication
- [ ] Database backups are configured
- [ ] Logging is configured and aggregated
- [ ] Monitoring/alerting is in place
- [ ] Graceful shutdown is tested (SIGTERM triggers clean shutdown)
- [ ] Load testing confirms acceptable latency
- [ ] Security review completed

---

## Summary

This assembly plan ensures:

1. **Correct Initialization Order** - Dependencies are initialized before dependents
2. **Clean Integration** - Services communicate via EventBus, repositories via RepositoryRegistry
3. **Proper Shutdown** - Graceful termination with timeout protection
4. **Observability** - Logging and status endpoints for monitoring
5. **Testability** - Clear separation of concerns allows unit/integration testing
6. **Scalability** - Architecture supports adding new services or replacing implementations

The all-in-one server is ready for deployment once all components are wired together following this plan.
