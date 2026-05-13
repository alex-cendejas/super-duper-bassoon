# API Service - Hexagonal Architecture Plan

## Overview
The API Service exposes a REST API for interacting with the automation engine. It provides endpoints for triggering workflows, querying clients/runs/health, managing bans, and system status. It is the primary user-facing interface.

## Step 1: Core/Domain Layer

### Responsibilities
Define pure business logic for API request/response handling and validation.

### Data Models & Methods

**APIRequest Domain Model** (generic request wrapper)
- `Method`: HTTP method (GET, POST, PUT)
- `Path`: request path
- `Headers`: request headers (map)
- `Query`: query parameters (map)
- `Body`: raw body (JSON)
- `UserID`: (optional) authenticated user
- Methods:
  - `Validate()`: basic structural validation
  - `HasRequiredHeaders()`: check content-type, etc.
  - `GetQueryParam(name)`: helper
  - `GetHeader(name)`: helper

**APIResponse Domain Model** (generic response wrapper)
- `StatusCode`: HTTP status (200, 400, 404, 500, etc.)
- `Headers`: response headers (map)
- `Body`: response body (JSON-serializable)
- Methods:
  - `SetStatus(code int)`: builder
  - `SetBody(data interface{})`: builder
  - `ToJSON() string`: serialize

**APIError Domain Model** (error representation)
- `Code`: error code string (e.g., "INVALID_FILTER", "WORKFLOW_NOT_FOUND")
- `Message`: human-readable error message
- `Details`: additional context (map)
- `Timestamp`: when error occurred
- Methods:
  - `Serialize()`: to JSON
  - `GetHTTPStatus()`: map error code to HTTP status
  - `Describe()`: human-readable

**QueryFilter Domain Model** (for list endpoints)
- `Limit`: max results (default 10, max 1000)
- `Offset`: pagination offset
- `SortBy`: field to sort (e.g., "created_at")
- `SortOrder`: asc/desc
- `Fields`: which fields to return (projection)
- Methods:
  - `Validate()`: check limits, reasonable values
  - `GetOffset()`: compute SQL offset
  - `GetLimit()`: return validated limit

**TriggerWorkflowRequest Domain Model**
- `WorkflowID`: which workflow to trigger
- `Reason`: (optional) admin reason for manual trigger
- Methods:
  - `Validate()`: check workflow_id is provided

**ClientStatusResponse Domain Model**
- `ClientID`, `OS`, `Labels`, `InnerState`, `Active`, `BannedFromWorkflows`, `LastSeenAt`
- Methods:
  - `FilterFields(projection []string)`: return only requested fields

**RunStatusResponse Domain Model**
- `RunID`, `WorkflowID`, `TriggeredAt`, `DispatchedAt`, `State`, `Health`, `ClientCount`, `SuccessCount`, `FailCount`
- Methods:
  - `Describe()`: summary

**HealthStatusResponse Domain Model**
- `WorkflowType`, `SuccessPercentage`, `FailPercentage`, `ErrorPercentage`, `Trend`, `HealthyFlag`, `LastCalculatedAt`
- Methods:
  - `ToJSON()`: serialization

**BanListResponse Domain Model**
- `ClientID`, `WorkflowType`, `BannedAt`, `BannedUntil`, `Reason`, `RunIDEvidence`
- Methods:
  - `FilterBans(clientID)`: for querying single client
  - `IsActive()`: check if current ban is active

**UnbanRequest Domain Model**
- `ClientID`: which client to unban
- `WorkflowType`: (optional) which workflow type, null = all bans
- `AdminID`: who is unbanning
- `Reason`: why unban is requested
- Methods:
  - `Validate()`: check required fields

**CreateWorkflowRequest Domain Model**
- `Name`: workflow name (required)
- `Description`: workflow description
- `WorkflowDefinition`: the workflow definition/logic (JSON)
- `Enabled`: initial enabled state (default: true)
- Methods:
  - `Validate()`: check name is not empty, definition is valid JSON

**EditWorkflowRequest Domain Model**
- `Name`: (optional) new name
- `Description`: (optional) new description
- `WorkflowDefinition`: (optional) updated workflow definition
- `Enabled`: (optional) change enabled state
- Methods:
  - `Validate()`: check at least one field is provided, definition if provided is valid JSON
  - `HasChanges()`: check if any updates were specified

**DeleteWorkflowRequest Domain Model**
- `WorkflowID`: which workflow to delete
- `Force`: (optional) force delete even if runs exist
- Methods:
  - `Validate()`: check workflow_id is provided

**ActivateWorkflowRequest Domain Model**
- `WorkflowID`: which workflow to activate
- Methods:
  - `Validate()`: check workflow_id is provided

**DeactivateWorkflowRequest Domain Model**
- `WorkflowID`: which workflow to deactivate
- Methods:
  - `Validate()`: check workflow_id is provided

### File Structure
```
internal/core/domain/
  api_request.go             # APIRequest model
  api_request_test.go
  api_response.go            # APIResponse model
  api_response_test.go
  api_error.go               # APIError model
  api_error_test.go
  query_filter.go            # QueryFilter model
  query_filter_test.go
  request_models.go          # TriggerWorkflowRequest, etc.
  request_models_test.go
  response_models.go         # Response models
  response_models_test.go
```

---

## Step 2: Core/Ports Layer

### Responsibilities
Define generic interfaces for core business operations exposed via API.

### Port Interfaces

**WorkflowService Port** (create, read, update, delete, and trigger workflows)
- `CreateWorkflow(ctx, req *CreateWorkflowRequest) (*Workflow, error)`
- `EditWorkflow(ctx, workflowID string, req *EditWorkflowRequest) (*Workflow, error)`
- `DeleteWorkflow(ctx, workflowID string) error`
- `ActivateWorkflow(ctx, workflowID string) error`
- `DeactivateWorkflow(ctx, workflowID string) error`
- `TriggerWorkflow(ctx, workflowID, reason string) (*Run, error)`
- `GetWorkflow(ctx, workflowID) (*Workflow, error)`
- `ListWorkflows(ctx, filter *QueryFilter) ([]*Workflow, error)`
- `GetWorkflowHealth(ctx, workflowType) (*WorkflowTypeHealth, error)`

**ClientService Port** (query clients)
- `GetClient(ctx, clientID) (*ClientMetadata, error)`
- `ListClients(ctx, filter *QueryFilter) ([]*ClientMetadata, error)`
- `GetClientStatus(ctx, clientID) (*ClientStatus, error)`: includes bans

**RunService Port** (query runs)
- `GetRun(ctx, runID) (*Run, error)`
- `ListRuns(ctx, workflowID, filter *QueryFilter) ([]*Run, error)`
- `ListRunsByStatus(ctx, status string, filter *QueryFilter) ([]*Run, error)`
- `GetRunResults(ctx, runID) ([]*Result, error)`: detailed run results

**HealthService Port** (query health)
- `GetWorkflowTypeHealth(ctx, workflowType) (*WorkflowTypeHealth, error)`
- `ListAllHealths(ctx) ([]*WorkflowTypeHealth, error)`
- `GetHealthTrend(ctx, workflowType, timeWindow) (*HealthTrend, error)`: historical data

**BanService Port** (manage bans)
- `GetBans(ctx, clientID) ([]*BanRecord, error)`
- `ListAllBans(ctx, filter *QueryFilter) ([]*BanRecord, error)`
- `UnbanClient(ctx, clientID, workflowType, reason) error`

**CircuitBreakerService Port** (query circuit state)
- `GetCircuitState(ctx, workflowID) (*WorkflowCircuitBreaker, error)`
- `ListAllCircuitStates(ctx) ([]*WorkflowCircuitBreaker, error)`

**SystemService Port** (system status and config)
- `GetSystemStatus(ctx) (*SystemStatus, error)`: uptime, DB status, NATS status
- `GetConfig(ctx) (*SystemConfig, error)`: active configuration values

### File Structure
```
internal/core/ports/
  workflow_service.go        # WorkflowService interface
  client_service.go          # ClientService interface
  run_service.go             # RunService interface
  health_service.go          # HealthService interface
  ban_service.go             # BanService interface
  circuit_breaker_service.go # CircuitBreakerService interface
  system_service.go          # SystemService interface
```

---

## Step 3: Core/Services Layer

### Responsibilities
Orchestrate ports to implement API business logic.

### Services

**APIHandlerService** (routes and handles HTTP requests)
- Dependencies: all service ports, RequestValidationService, ResponseFormattingService
- Methods:
  - `HandleCreateWorkflow(ctx, req *APIRequest) *APIResponse`:
    - Validate request (parse JSON, check required fields)
    - Call WorkflowService.CreateWorkflow()
    - Format and return WorkflowResponse
    - Handle errors with APIError
  - `HandleEditWorkflow(ctx, req *APIRequest) *APIResponse`:
    - Extract workflow_id from path
    - Validate request (parse JSON, check at least one field)
    - Call WorkflowService.EditWorkflow()
    - Format and return WorkflowResponse
  - `HandleDeleteWorkflow(ctx, req *APIRequest) *APIResponse`:
    - Extract workflow_id from path
    - Validate request (check required fields)
    - Call WorkflowService.DeleteWorkflow()
    - Return 204 No Content or success response
  - `HandleActivateWorkflow(ctx, req *APIRequest) *APIResponse`:
    - Extract workflow_id from path
    - Call WorkflowService.ActivateWorkflow()
    - Return updated WorkflowResponse
  - `HandleDeactivateWorkflow(ctx, req *APIRequest) *APIResponse`:
    - Extract workflow_id from path
    - Call WorkflowService.DeactivateWorkflow()
    - Return updated WorkflowResponse
  - `HandleTriggerWorkflow(ctx, req *APIRequest) *APIResponse`:
    - Validate request (parse JSON, check required fields)
    - Call WorkflowService.TriggerWorkflow()
    - Format and return RunStatusResponse
    - Handle errors with APIError
  - `HandleGetWorkflow(ctx, req *APIRequest) *APIResponse`
  - `HandleListWorkflows(ctx, req *APIRequest) *APIResponse`
  - `HandleGetClient(ctx, req *APIRequest) *APIResponse`
  - `HandleListClients(ctx, req *APIRequest) *APIResponse`
  - `HandleGetRun(ctx, req *APIRequest) *APIResponse`
  - `HandleListRuns(ctx, req *APIRequest) *APIResponse`
  - `HandleGetHealth(ctx, req *APIRequest) *APIResponse`: query workflow_type health
  - `HandleListAllHealth(ctx, req *APIRequest) *APIResponse`
  - `HandleGetBans(ctx, req *APIRequest) *APIResponse`: query client bans
  - `HandleListAllBans(ctx, req *APIRequest) *APIResponse`
  - `HandleUnbanClient(ctx, req *APIRequest) *APIResponse`
  - `HandleGetCircuitState(ctx, req *APIRequest) *APIResponse`
  - `HandleGetSystemStatus(ctx, req *APIRequest) *APIResponse`

**RequestValidationService** (validates incoming requests)
- Dependencies: none
- Methods:
  - `ValidateCreateWorkflowRequest(req *APIRequest) error`: check name, definition validity
  - `ValidateEditWorkflowRequest(req *APIRequest) error`: check at least one field, definition if provided
  - `ValidateDeleteWorkflowRequest(req *APIRequest) error`: check workflow_id
  - `ValidateActivateWorkflowRequest(req *APIRequest) error`: check workflow_id
  - `ValidateDeactivateWorkflowRequest(req *APIRequest) error`: check workflow_id
  - `ValidateTriggerWorkflowRequest(req *APIRequest) error`: check workflow_id
  - `ValidateQueryFilter(req *APIRequest) (*QueryFilter, error)`: parse limit, offset, sort
  - `ValidateUnbanRequest(req *APIRequest) error`: check required fields
  - `ValidatePathParam(name, value string) error`: check path params

**ResponseFormattingService** (formats outgoing responses)
- Dependencies: none
- Methods:
  - `FormatWorkflowResponse(workflow *Workflow) interface{}`
  - `FormatClientResponse(client *ClientMetadata) interface{}`
  - `FormatRunResponse(run *Run) interface{}`
  - `FormatHealthResponse(health *WorkflowTypeHealth) interface{}`
  - `FormatBanResponse(ban *BanRecord) interface{}`
  - `FormatErrorResponse(err *APIError) *APIResponse`
  - `FormatListResponse(items []interface{}, total, limit, offset int) interface{}`

### File Structure
```
internal/core/services/
  api_handler.go            # APIHandlerService
  api_handler_test.go
  request_validation.go     # RequestValidationService
  request_validation_test.go
  response_formatting.go    # ResponseFormattingService
  response_formatting_test.go
```

---

## Step 4: Adapters Layer

### Responsibilities
Implement the port interfaces and HTTP server integration.

### Adapter Implementations

**HTTPServer Adapter** (implements HTTP routing and serving)
- Framework: stdlib `net/http` or chi/gin (chi recommended for minimal deps)
- Routes:
  - `POST /workflows` → HandleCreateWorkflow
  - `GET /workflows` → HandleListWorkflows
  - `GET /workflows/{id}` → HandleGetWorkflow
  - `PUT /workflows/{id}` → HandleEditWorkflow
  - `DELETE /workflows/{id}` → HandleDeleteWorkflow
  - `POST /workflows/{id}/trigger` → HandleTriggerWorkflow
  - `POST /workflows/{id}/activate` → HandleActivateWorkflow
  - `POST /workflows/{id}/deactivate` → HandleDeactivateWorkflow
  - `GET /clients/{id}` → HandleGetClient
  - `GET /clients` → HandleListClients
  - `GET /runs/{id}` → HandleGetRun
  - `GET /workflows/{id}/runs` → HandleListRuns
  - `GET /health/{workflow_type}` → HandleGetHealth
  - `GET /health` → HandleListAllHealth
  - `GET /bans/{client_id}` → HandleGetBans
  - `GET /bans` → HandleListAllBans
  - `PUT /bans/{client_id}/unban` → HandleUnbanClient
  - `GET /circuits/{workflow_id}` → HandleGetCircuitState
  - `GET /circuits` → HandleListAllCircuitStates (optional)
  - `GET /status` → HandleGetSystemStatus
- Middleware:
  - Logging (request/response)
  - Error handling (convert APIError to HTTP response)
  - CORS (if needed)

**ServicePortAdapters** (delegate to actual services)
- WorkflowServiceAdapter → WorkflowOrchestrationService
- ClientServiceAdapter → ClientRepository + DynamicGroupingService
- RunServiceAdapter → RunRepository + HealthMonitoringService
- HealthServiceAdapter → HealthMonitoringService
- BanServiceAdapter → BanEnforcementService
- CircuitBreakerServiceAdapter → CircuitBreakerService
- SystemServiceAdapter → DB/NATS health checks + config

**JSONMarshalerAdapter** (serialization)
- Implement custom JSON marshaling for domain models
- Encode health percentages, timestamps, enums
- Hide internal implementation details

### File Structure
```
internal/adapters/
  http/
    server.go              # HTTPServer setup + routes
    server_test.go
    middleware.go          # logging, error handling
  service/
    workflow_adapter.go    # WorkflowService implementation
    client_adapter.go      # ClientService implementation
    run_adapter.go         # RunService implementation
    health_adapter.go      # HealthService implementation
    ban_adapter.go         # BanService implementation
    circuit_adapter.go     # CircuitBreakerService implementation
    system_adapter.go      # SystemService implementation
  marshaling/
    json_marshaler.go      # JSON serialization
```

---

## Step 5: Binary/Deployment Layer

### Responsibilities
Wire together HTTP server and services, handle startup/shutdown.

### Configuration
- Environment variables:
  - `HTTP_HOST`: bind address (default: 0.0.0.0)
  - `HTTP_PORT`: port number (default: 8080)
  - `HTTP_READ_TIMEOUT_MS`: request timeout (default: 30000)
  - `HTTP_MAX_BODY_SIZE_BYTES`: max request body (default: 1MB)
  - `DB_PATH`: SQLite database
  - `NATS_URL`: NATS broker
  - `LOG_LEVEL`: debug/info/warn/error (default: info)

### Initialization & Wiring

**cmd/main.go** (integrated into server)
```
1. Parse env variables for HTTP config
2. Initialize DB connections + NATS (reuse existing connections from other services)
3. Create all service ports/adapters:
   - WorkflowOrchestrationService → WorkflowServiceAdapter
   - ClientRepository + DynamicGroupingService → ClientServiceAdapter
   - RunRepository + HealthMonitoringService → RunServiceAdapter
   - HealthMonitoringService → HealthServiceAdapter
   - BanEnforcementService → BanServiceAdapter
   - CircuitBreakerService → CircuitBreakerServiceAdapter
   - SystemStatus checker → SystemServiceAdapter
4. Create core services:
   - APIHandlerService (with all adapters)
   - RequestValidationService
   - ResponseFormattingService
5. Create HTTPServer with:
   - APIHandlerService
   - Response formatter
   - Error handler
   - Request logger middleware
6. Bind to configured host:port
7. Start HTTP server in goroutine
8. Setup graceful shutdown: drain in-flight requests, close listeners, close DB/NATS
```

**cmd/config.go**
- `LoadConfig()`: read env vars, set defaults, validate
- Env vars: HTTP_HOST, HTTP_PORT, HTTP_READ_TIMEOUT_MS, HTTP_MAX_BODY_SIZE_BYTES, LOG_LEVEL

### File Structure
```
cmd/
  main.go          # API service wiring (integrated into server)
  config.go        # config loading
```

### Runtime Behavior
1. On startup: initialize HTTP server and routes, listen on configured port
2. On incoming HTTP request:
   - Logging middleware logs method/path
   - Router dispatches to appropriate handler
   - Handler validates request (RequestValidationService)
   - Handler calls service port (e.g., WorkflowService)
   - Response is formatted (ResponseFormattingService)
   - Error handling converts errors to HTTP status codes + APIError JSON
   - Logging middleware logs response status
3. On API error:
   - APIError is caught by middleware
   - Converted to HTTP response with appropriate status code
   - Returned as JSON with error code, message, details
4. On graceful shutdown:
   - New connections are rejected
   - In-flight requests are given grace period to complete
   - HTTP listener closes

### Integration Points
- **Driven by:** HTTP clients (REST API consumers)
- **Calls:** All service ports (workflows, clients, runs, health, bans, circuits, system)
- **Reads:** All repositories (workflows, clients, runs, health, bans, circuits)
- **No writes:** except through service operations (e.g., unban)
- **No messaging:** API is synchronous request/response

---

## Example API Usage

```bash
# Create a workflow
POST /workflows
Body: {"name": "update_config", "description": "Deploy new configuration", "workflow_definition": {...}, "enabled": true}
Response: 201 Created, {id: "wf-123", name: "update_config", created_at: "...", ...}

# List workflows
GET /workflows?limit=10&offset=0&sort_by=created_at&sort_order=desc
Response: 200 OK, {items: [...], total: 5, limit: 10, offset: 0}

# Get workflow
GET /workflows/wf-123
Response: 200 OK, {id: "wf-123", name: "update_config", enabled: true, ...}

# Edit a workflow
PUT /workflows/wf-123
Body: {"name": "update_config_v2", "description": "Updated description"}
Response: 200 OK, {id: "wf-123", name: "update_config_v2", ...}

# Activate a workflow
POST /workflows/wf-123/activate
Response: 200 OK, {id: "wf-123", enabled: true, activated_at: "...", ...}

# Deactivate a workflow
POST /workflows/wf-123/deactivate
Response: 200 OK, {id: "wf-123", enabled: false, deactivated_at: "...", ...}

# Delete a workflow
DELETE /workflows/wf-123
Response: 204 No Content

# Trigger a workflow
POST /workflows/wf-123/trigger
Body: {"reason": "manual trigger for testing"}
Response: 200 OK, {run_id: "run-456", triggered_at: "...", ...}

# Get workflow health
GET /health/update_config
Response: 200 OK, {workflow_type: "update_config", success_percentage: 95.2, ...}

# List clients
GET /clients?limit=10&offset=0&sort_by=created_at&sort_order=desc
Response: 200 OK, {items: [...], total: 42, limit: 10, offset: 0}

# Get client status (includes bans)
GET /clients/client-789
Response: 200 OK, {client_id: "client-789", os: "linux", banned_from_workflows: ["wf-123"], ...}

# Unban client
PUT /bans/client-789/unban
Body: {"workflow_type": "wf-123", "admin_id": "admin-1", "reason": "manual unban after recovery"}
Response: 200 OK, {success: true, message: "..."}

# Get circuit breaker state
GET /circuits/wf-123
Response: 200 OK, {workflow_id: "wf-123", state: "open", opened_at: "...", reason: {...}, ...}

# System status
GET /status
Response: 200 OK, {uptime_seconds: 86400, db_status: "healthy", nats_status: "connected", ...}
```

---

## Implementation Dependencies

- **Depends on:** All service ports (workflows, clients, runs, health, bans, circuits, system)
- **Used by:** HTTP clients (curl, SDKs, web UI)
- **Database:** SQLite (read-only for queries, write through service ports)
- **Framework:** stdlib `net/http` or chi for routing/middleware
- **No NATS:** API doesn't directly use messaging, goes through services
