Project super-duper-bassoon: Automation Engine PoC

Project super-duper-bassoon is a lightweight automation workflow engine designed to manage remote devices (toy clients) via asynchronous messaging. It focuses on safety rails, specifically loop detection, health-based circuit breaking, and permanent client banning.
Architectural Overview

super-duper-bassoon follows a Hub-and-Spoke architecture using NATS/MQTT as the central message bus.

    Server: The central authority managing state, scheduling, and safety logic.

    Broker: Handles asynchronous communication via orbit/cmd/{id} and orbit/results.

    Toy Clients: State-aware agents that simulate real-world execution, random failure modes, and spontaneous state drift.

    Database: SQLite for persistent storage of client metadata, workflow definitions, and run history.

Core Services
1. Messaging and Protocol

    Standard Payload: JSON containing run_id, wf_id, activity, and params.

    Result Payload: Includes status (success, fail, or error), the client's updated inner_state, and error_msg.

    Activity Types: reboot, install package, upgrade package, remove package, apply config, validate config, and run script.

2. Workflow and Orchestration Service

    Workflow Definition: Targets a group via filter, specifies an activity, and sets success_threshold and loop_threshold.

    Triggers: Scheduled (cron-like) or Event-driven (chained to another workflow's completion).

    Execution: On trigger, the service takes a Static Snapshot of the fleet matching the filter. It generates a unique run_id and dispatches commands to each client.

3. Dynamic Grouping Service

    Evaluates client metadata and inner state against workflow filters. Example: os == 'linux' AND state.config_version < 2.

    Resolves filters into a concrete list of client_ids only at the moment of workflow initiation.

4. Loop-Detection and Ban Service

    Concurrency Guard: Tracks client_id, workflow_type, and timestamp for every dispatch.

    Violation: If a client enters a second run_id for the same workflow_type within the loop_threshold window, it is flagged as a loop.

    The Ban: Banned clients are permanently excluded from that specific workflow_type. This state is persisted in the DB and requires a manual administrative Unban command to clear.

5. Health Monitoring and Circuit Breaker

    Run Health: Tracks real-time status including Total, Success percentage, Fail percentage, and Pending percentage.

    Type Health: Aggregates the last n runs of a workflow_type.

    Circuit Breaker: If the aggregated success rate falls below the workflow's success_threshold, the workflow is Deactivated (Active=False) and an alert is raised.

6. The Toy Client (Simulation)

    Internal State: Manages a virtual manifest of packages, config versions, and power state.

    Chaos Engine: Randomly determines activity outcome. On failure, the client may enter an ERROR state.

    Spontaneous Drift: Occasionally, the client will randomly modify its own state (e.g., removing a package or changing a config version) independent of server commands. This simulates "shadow IT" or local manual changes.

    Impact: While crippled, package and config activities fail automatically; only reboot or run script can restore functionality.

Safety Scenarios

    Rapid Retriggering: Handled by the Loop Detector via a Permanent Client Ban.

    Mass Client Failure: Handled by the Circuit Breaker via Deactivating the Workflow Type.

    Client Logic Lock: Handled by the Toy Client (Chaos) by entering a Crippled State.

    Stale Targets: Handled by the Dynamic Grouper using a Static Snapshot at the time of the trigger.

    Configuration Drift: Detected by the Dynamic Grouper during the next workflow run via the Spontaneous Drift feature.
