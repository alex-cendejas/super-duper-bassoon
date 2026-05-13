# Dynamic Grouping Service - Hexagonal Architecture Plan

## Overview
The Dynamic Grouping Service evaluates filter expressions against client metadata and state to resolve a concrete list of target clients for workflow execution. It must handle complex filter syntax, match against both static metadata and dynamic state, and return results at trigger time.

## Step 1: Core/Domain Layer

### Responsibilities
Define pure business logic for filter expression parsing, validation, and evaluation.

### Data Models & Methods

**FilterExpression Domain Model**
- `Expression`: string representation of the filter (e.g., `os == 'linux' AND state.config_version < 2`)
- `ParsedExpression`: structured AST or parsed form
- Methods:
  - `Validate()`: syntax/semantic validation
  - `ExtractRequiredFields()`: list of client fields needed for evaluation
  - `IsSimple()`: heuristic for single-table vs complex evaluation

**FilterOperator Domain Model** (enum-like)
- Operators: `==`, `!=`, `<`, `>`, `<=`, `>=`, `IN`, `NOT_IN`, `CONTAINS`, `NOT_CONTAINS`
- Methods:
  - `Evaluate(left, right)`: apply operator to operands
  - `RequiresType()`: what data type is expected

**FilterCondition Domain Model**
- `FieldPath`: dot-notation path (e.g., `state.config_version`)
- `Operator`: comparison operator
- `Value`: right-hand side value
- Methods:
  - `Matches(clientMetadata, clientState)`: bool result
  - `Describe()`: human-readable condition text

**FilterLogicalExpression Domain Model** (AST node)
- `Left`: condition or sub-expression
- `Operator`: AND, OR, NOT
- `Right`: condition or sub-expression (null for NOT)
- Methods:
  - `Evaluate(clientMetadata, clientState)`: recursive evaluation
  - `Optimize()`: simplification/short-circuit hints

**ClientMetadata Domain Model** (immutable value)
- `ClientID`: unique identifier
- `OS`: operating system string
- `Labels`: map[string]string for grouping metadata
- `InnerState`: map for dynamic state fields
- Methods:
  - `GetField(path string)`: retrieve value by dot-notation path
  - `MatchesCondition(cond Condition)`: bool

**FilterResult Domain Model**
- `MatchingClientIDs`: list of ClientID that matched
- `TotalEvaluated`: count of clients evaluated
- `MatchCount`: count of matches
- `EvaluatedAt`: timestamp
- Methods:
  - `GetMatchPercentage()`: % of clients that matched
  - `IsEmpty()`: no matches found

**GroupingError Domain Model**
- `InvalidSyntax`: unparseable filter expression
- `UnknownField`: referenced field doesn't exist
- `TypeMismatch`: operand type incompatible with operator
- `EvaluationError`: runtime error during evaluation

### File Structure
```
internal/core/domain/
  filter_expression.go     # FilterExpression struct + parsing
  filter_expression_test.go
  filter_operator.go       # Operator enum + evaluation
  filter_operator_test.go
  filter_condition.go      # Condition struct + evaluation
  filter_condition_test.go
  filter_ast.go            # AST node types
  filter_ast_test.go
  client_metadata.go       # ClientMetadata + field resolution
  client_metadata_test.go
  filter_result.go         # FilterResult model
  grouping_error.go        # Error types
```

---

## Step 2: Core/Ports Layer

### Responsibilities
Define generic interfaces for data sources and evaluation strategies.

### Port Interfaces

**ClientRepository Port** (read-only view)
- `ListAllClients(ctx) ([]*ClientMetadata, error)`: retrieve all clients and their metadata/state
- `GetClientByID(ctx, clientID) (*ClientMetadata, error)`: fetch single client
- `GetClientsByIDs(ctx, []ClientID) ([]*ClientMetadata, error)`: batch fetch

**FilterValidator Port** (optional optimization)
- `ValidateExpression(ctx, expr string) error`: pre-flight validation
- `GetSupportedFields(ctx) []string`: advertise available fields

**CachingPort** (optional, for repeated filter evaluation)
- `GetCachedResult(ctx, filterExpr string) (*FilterResult, error)`
- `SetCachedResult(ctx, filterExpr string, result *FilterResult) error`
- `InvalidateCache(ctx, clientID) error`: on client state change

### File Structure
```
internal/core/ports/
  client_repository.go     # ClientRepository interface
  filter_validator.go      # FilterValidator interface
  cache.go                 # CachingPort interface
```

---

## Step 3: Core/Services Layer

### Responsibilities
Orchestrate domain logic using ports to provide filter evaluation capabilities.

### Services

**DynamicGroupingService**
- Dependencies: ClientRepository, FilterValidator (optional), CachingPort (optional)
- Methods:
  - `ResolveClients(ctx, filterExpr string) (*FilterResult, error)`: main entry point
    - Validate expression (use FilterValidator if available)
    - Fetch all clients from ClientRepository
    - Parse expression into AST
    - Evaluate AST against each client
    - Collect matching client IDs
    - Return FilterResult
  - `ResolveClientsByIDs(ctx, filterExpr string, clientIDs []ClientID) (*FilterResult, error)`: scoped version
    - Fetch only specified clients
    - Evaluate same way as above
  - `PreloadClients(ctx) error`: optional warming of client cache

**FilterEvaluationService** (implements domain evaluation logic)
- Dependencies: none (purely functional)
- Methods:
  - `ParseExpression(expr string) (*FilterAST, error)`: syntax → AST
  - `ValidateAST(ast *FilterAST) error`: semantic validation
  - `EvaluateAST(ast *FilterAST, client *ClientMetadata) (bool, error)`: recursive evaluation
  - `OptimizeAST(ast *FilterAST) *FilterAST`: optional performance hint

**FieldResolver Service** (interprets dot-notation paths)
- Dependencies: none
- Methods:
  - `ResolveField(client *ClientMetadata, path string) (interface{}, error)`
    - Split path by `.`
    - Walk structure: `os` → static, `state.X` → dynamic map
    - Return value and type
  - `GetFieldType(path string) (Type, error)`: for type checking

### File Structure
```
internal/core/services/
  dynamic_grouping.go          # DynamicGroupingService
  dynamic_grouping_test.go
  filter_evaluation.go         # FilterEvaluationService
  filter_evaluation_test.go
  field_resolver.go            # FieldResolver
  field_resolver_test.go
```

---

## Step 4: Adapters Layer

### Responsibilities
Implement the port interfaces with concrete technologies.

### Adapter Implementations

**SQLiteClientRepository** → ClientRepository port
- Queries SQLite clients table + joins with inner_state
- Implementation: `ListAllClients`, `GetClientByID`, `GetClientsByIDs`
- Query: `SELECT id, os, labels, inner_state FROM clients WHERE active = true`
- Deserializes JSON labels and inner_state into maps

**SimpleFilterValidator** → FilterValidator port
- Regex-based syntax check + known field whitelist
- Implementation: `ValidateExpression`, `GetSupportedFields`
- Supported fields hardcoded: os, labels.*, state.*

**InMemoryFilterCache** → CachingPort
- Simple map[string]*FilterResult with TTL or LRU eviction
- Implementation: `GetCachedResult`, `SetCachedResult`, `InvalidateCache`
- (Can be replaced with Redis later for distributed caching)

**ExpressionParser** (internal utility)
- Tokenizer: split expression into tokens (operators, identifiers, values)
- Parser: recursive descent or similar to build AST
- Uses FilterOperator and FilterCondition domains for evaluation

### File Structure
```
internal/adapters/
  repository/
    sqlite_client_repo.go
    sqlite_client_repo_test.go
  validator/
    simple_filter_validator.go
    simple_filter_validator_test.go
  cache/
    inmemory_filter_cache.go
    inmemory_filter_cache_test.go
  parser/
    expression_parser.go
    expression_parser_test.go
    tokenizer.go
    tokenizer_test.go
```

---

## Step 5: Binary/Deployment Layer

### Responsibilities
Wire together components and expose grouping via API or internal service calls.

### Configuration
- Environment variables:
  - `DB_PATH`: SQLite database file
  - `FILTER_CACHE_ENABLED`: bool, enable caching
  - `FILTER_CACHE_TTL_MS`: time-to-live for cached results
  - `SUPPORTED_FILTER_FIELDS`: comma-separated list (or hardcoded)

### Initialization & Wiring

**cmd/main.go** (if standalone) OR **Integrated into orchestration main.go**
```
1. Parse env variables
2. Initialize DB connection (SQLiteClientRepository)
3. Create port implementations (repos, validators, cache)
4. Create domain services (DynamicGroupingService, FilterEvaluationService, FieldResolver)
5. Wire dependencies (services get ports)
6. If standalone: expose via gRPC or REST API
7. If integrated: register as singleton for WorkflowOrchestrationService
8. Setup graceful shutdown
```

**cmd/config.go**
- `LoadConfig()`: read env vars, validate defaults
- Env vars: DB_PATH, FILTER_CACHE_ENABLED, FILTER_CACHE_TTL_MS, LOG_LEVEL

### File Structure
```
cmd/
  main.go          # entry point (standalone or included in orchestration)
  config.go        # config loading
```

### Runtime Behavior
1. On startup: initialize DB connection, warm client cache (optional)
2. On filter evaluation request:
   - Check cache (if enabled)
   - Parse expression into AST
   - Validate AST
   - Fetch clients from DB
   - Evaluate each client against AST
   - Cache result (if enabled)
   - Return matching client IDs
3. On client state change: invalidate cache entries for affected filters
4. On shutdown: close DB connection

### Integration Points
- **Driven by:** WorkflowOrchestrationService calls `ResolveClients(filter)` at trigger time
- **Returns:** list of matching ClientIDs to be used in dispatch generation
- **Read-only:** does not modify any state, purely computational

---

## Implementation Dependencies

- **Depends on:** ClientRepository (for client data)
- **Used by:** WorkflowOrchestrationService (to resolve target clients), API Service (for filter queries)
- **Database:** SQLite for client metadata/state
- **No external messaging:** purely computational, no NATS dependency
