# Project super-duper-bassoon: Automation Engine PoC

A lightweight workflow automation engine for managing remote devices (toy clients) via asynchronous messaging. The PoC validates three core safety mechanisms: **loop detection** (prevents clients from re-entering the same workflow), **permanent banning** (isolates problem clients), and **circuit breaking** (stops cascading failures).

## Architecture Overview

Hub-and-spoke pattern with internal service coordination:

- **Server (Go all-in-one binary):** Orchestration, safety logic, state management, REST API. Internal services communicate via Go channels.
- **NATS Broker:** Asynchronous messaging between server and clients.
- **SQLite Database:** Persistent storage (client metadata, workflow definitions, run history, bans, loop detection records).
- **Super-Client (toy simulation):** Single binary spawning multiple inner clients, each with simulated state and chaos behavior.

## Core Concepts

### Run
One dispatch cycle from the server to all matching clients at a moment in time. Health is calculated per-run in real-time (does not wait for completion).

### Health
- **Run Health:** Per-run metrics (total clients, success %, fail %, pending %).
- **Type Health:** Aggregated across last n runs of a workflow_type, used by circuit breaker.

### Workflow Definition
- **Trigger:** Scheduled (cron), event-driven (on another workflow's completion), or state change (on client state event).
- **Target:** Filter expression (e.g., `os == 'linux' AND state.config_version < 2`).
- **Activity:** One of: `reboot`, `install package`, `upgrade package`, `remove package`, `apply config`, `validate config`, `run script` (with parameters inline).
- **Thresholds:** `success_threshold` (circuit breaker trigger), `loop_threshold` (time window for loop detection).
- **Timeout:** Per-workflow activity timeout.
- **State:** `active` flag (set to false by circuit breaker on deactivation).

### Activity Semantics
- **Package operations:** Atomic (all-or-nothing).
- **Validate config:** Client confirms current config matches expected state.
- **Run script:** Returns exit code and stdout/stderr in result payload.

## Services

### 1. Workflow & Orchestration Service
Executes workflows on trigger. Takes a static snapshot of matching clients at trigger time. Generates unique `run_id` per run and dispatches commands. Chains workflows via event-driven triggers.

### 2. Dynamic Grouping Service
Evaluates filter expressions against client metadata + state. Resolves to concrete client list only at workflow initiation.

### 3. Loop Detection & Ban Service
Processes result messages to detect loops. **Detection:** Client reports result for run_id N of workflow_type X while still within loop_threshold of run_id N-1 for same workflow_type. **Action:** Immediately ban client from that workflow_type (no more dispatches). Trigger alert. Persist ban record with run_id and result evidence. **Recovery:** Manual admin unban only.

### 4. Health Monitoring Service
Streams health metrics per run and aggregates across last n runs per workflow_type.

### 5. Circuit Breaker Service
Monitors aggregated health. When workflow_type health falls below success_threshold, deactivate workflow (set active=false, stop new dispatches) and alert.

### 6. API Service
REST API for: trigger workflow, query clients, query runs, query health, unban client, manage workflow state.

## Message Protocol

### Dispatch (server → client)
```json
{
  "run_id": "string",
  "wf_id": "string",
  "activity": "string",
  "params": {}
}
```

### Result (client → server)
```json
{
  "run_id": "string",
  "wf_id": "string",
  "status": "success|fail|error",
  "inner_state": {},
  "error_msg": "string?",
  "payload": {}
}
```

Repeated results with same (client_id, wf_id, run_id) are idempotent. Malformed results are ignored.

## Persistence

### Client Metadata
- `client_id`, `os`, `labels` (for grouping), `inner_state` (packages, config version, power state).

### Workflows
- Definition, active flag, trigger config, activity, thresholds, timeout.

### Run History
- `run_id`, `wf_id`, timestamp, list of participating clients, health stats, completion status.

### Loop/Ban Records
- `client_id`, `workflow_type`, `run_id` (evidence of loop), `result` (activity result that triggered), `banned_at`, `banned_until` (null = permanent).

All records are immutable audit trail. No pruning.

## Toy Client Behavior

Internal state machine with packages, config versions, power states. Chaos simulation:
- **10% of activities fail randomly.**
- **3% of failures cripple the client.**
- **Crippled state:** Client randomly decides to either fail all activities of a certain type OR stop responding catastrophically.
- **Crippling persists until reboot activity succeeds.**
- **Spontaneous drift:** 5-10% chance per run that client modifies its own state independently (simulates manual changes).

## Safety Mechanisms Working Together

1. **Loop Detection → Banning:** Client loops (enters run N while still in run N-1), server detects via result, immediately bans, alerts. Prevents infinite retry cycles.
2. **Circuit Breaking:** Workflow degrades (success % drops), server deactivates workflow, alerts. Prevents cascading damage across all clients.
3. **Banned Clients in Health:** Banned clients excluded from run totals (don't poison health calculation), but their completed activities still persisted (forensics).

## Implementation Approach

**Phase 1 (Core):** Build toy client with NATS comms → server core (loop/ban + circuit logic) with NATS comms → integration tests.

**Phase 2 (Polish):** Dynamic grouping, health aggregation, REST API.

**Phase 3 (Optional):** Web UI.

## Configuration

Environment variables only. No config files. Examples: `NATS_URL`, `DB_PATH`, `LOOP_THRESHOLD_MS`, `SUCCESS_THRESHOLD`, `HEALTH_WINDOW_SIZE`.

## Error Handling

- Malformed results: ignored (idempotent).
- DB unavailable: graceful shutdown with error.
- NATS unavailable: graceful shutdown with error.
- Client sends stale result (run_id doesn't exist): ignored.

## Success Criteria

**Phase 1 validation:**
1. Loop detection works: Client loops, gets banned, no further dispatches.
2. Circuit breaker works: Health drops, workflow deactivates.
3. Bans persist: Banned client stays banned across restarts.
4. Unban works: Admin unbans, client dispatchable again.
5. Toy client chaos: 10% failures, 3% crippling, reboot recovery.

**Phase 2 validation:** Dynamic grouping accuracy, health aggregation correctness, full REST API.
