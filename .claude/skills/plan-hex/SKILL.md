---
name: plan-hex
description: Plan the implementation of a project or component considering hexagonal architecture.
---

## Instructions

Structure planning based on the hexagonal architecture principles, for a Go project, in this order:

1. Plan the core/domain layer, which includes business logic data models and their methods without the use of any external components or specific implementations. These need to capture everything the business logic needs in order to accomplish the specified goal for the project.

2. Plan the core/ports layer, which includes interfaces for driving/driven adapters which will be needed to implement the business concepts using external components. However, these interfaces must be generic and not draw from a specific implementation of a component/service/library which could provide this functionality.

3. Plan the core/services layer, which should create services which use ports (the generic interfaces) in order to expose the services to driving adapters or the actual deployment.

4. Plan the adapters layer, which will contain specific implementations of the ports in core/ports which can be instantiated one layer above the core/services layer (binary/actual deployment) to enable the services.

5. Plan the top layer (binary/actual deployment), which will actually configure, instantiate, run, and manage the services.

## DO

- Provide details on the specific implementation steps.
- Provide an overview of the file structure for the project.
- Consider the previous step output as input for the next step.

## DON'T

- Mix layers, or change the order.
- Implement anything.
- Add layers.
- Remove layers.

## EXAMPLE FOR FILE STRUCTURE

- project
 - cmd
  - main.go
  - config.go
 - internal
  - adapters
   - repository
    - clients
     - sqlite_client.go
     - squlite_client_test.go
    - authz
     - permissions_store.go
     - permissions_store_test.go
   - api
    - json_api.go
    - json_api_test.go
    - grpc_api.go
    - grpc_api_test.go
  - core
   - domain
    - client.go
    - client_test.go
    - errors.go
    - permission.go
    - permission_test.go
   - ports
    - repository
     - clients.go
     - authz.go
   - services
     - client_service.go
     - client_service_test.go
     - permission_service.go
     - permission_service_test.go
