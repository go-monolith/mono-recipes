# Hexagonal Architecture Demo - Task/Todo System

This project demonstrates **Hexagonal Architecture** (Ports and Adapters) using the [mono framework](https://github.com/go-monolith/mono) for building modular monolith applications in Go.

## Architecture Overview

```
                    ┌──────────────────────────────────┐
                    │         HTTP Client              │
                    └──────────────┬───────────────────┘
                                   │
                    ┌──────────────▼───────────────────┐
                    │     API Module (Fiber HTTP)      │
                    │       [Driving Adapter]          │
                    │  owns TaskPort + taskAdapter     │
                    └──────────────┬───────────────────┘
                                   │ task/contract (CreateTask, GetTask, ...)
                    ┌──────────────▼───────────────────┐
                    │        Task Module               │
                    │       [Core Domain]              │
                    │  owns UserPort + userAdapter,    │
                    │  emits task/contract events      │
                    └───────┬──────────────┬───────────┘
                            │              │
     user/contract          │              │ task/contract events
     (ValidateUser)         │              │ (TaskCreatedV1, ...)
    ┌───────────────────────▼──┐    ┌──────▼───────────────────┐
    │      User Module         │    │   Notification Module    │
    │  User validation and     │    │     [Driven Adapter]     │
    │  storage                 │    │   Subscribes to events   │
    └──────────────────────────┘    └──────────────────────────┘
```

Every arrow crosses a module boundary through a **contract package** owned by
the provider. No module imports another module's package other than its
`contract`.

## Hexagonal Architecture Concepts

### Contracts (published ports)
Each module that offers services or events publishes them in a `contract`
sub-package: DTOs plus typed definitions declared once and shared by both sides.

- `modules/user/contract`: `GetUser`, `ValidateUser` (request-reply)
- `modules/task/contract`: `CreateTask`, `GetTask`, `UpdateTask`, `DeleteTask`,
  `ListTasks`, `CompleteTask` (request-reply) and the events the task module
  emits: `TaskCreatedV1`, `TaskCompletedV1`, `TaskDeletedV1`

A contract package imports only the standard library and `mono/pkg/helper`
(or `mono/pkg/types`), so any module can import it without cycles.

### Driven ports (owned by the consumer)
- **`api.TaskPort`** (`modules/api/ports.go`): what the HTTP adapter needs from the core domain
- **`task.UserPort`** (`modules/task/ports.go`): what the task domain needs from the user module

Ports speak the provider's contract DTOs and say nothing about NATS, so tests
can substitute a fake.

### Adapters
- **Driving adapters**: the API module receives HTTP requests; each module's
  `RegisterServices` binds its domain handlers to its own contracts.
- **Driven adapters**: `api/adapter.go` and `task/adapter.go` implement the
  ports by calling the provider's contract (`taskcontract.CreateTask.Call(...)`)
  through the container received in `SetDependencyServiceContainer`. The
  notification module reacts to domain events.

### Domain Isolation
The task module (core domain) has no knowledge of HTTP or notification
systems. It depends only on the `UserPort` interface it owns and publishes
events through its own contract.

## Project Structure

```
hexagonal-architecture/
├── main.go                    # Application bootstrap (strict module boundaries)
├── boundaries_test.go         # Whole-app boundary report test (in-process NATS)
├── go.mod                     # Go module definition
├── domain/
│   └── task/
│       └── entity.go          # Task domain entity
└── modules/
    ├── api/                   # Driving Adapter (HTTP)
    │   ├── module.go          # DependentModule + ServiceConsumerModule (depends on task)
    │   ├── ports.go           # TaskPort (driven port owned by api)
    │   ├── adapter.go         # taskAdapter: TaskPort over task/contract
    │   ├── handlers.go        # REST endpoint handlers
    │   ├── dto.go             # HTTP request/response DTOs
    │   └── module_test.go     # monotest.CheckModule conformance test
    ├── task/                  # Core Domain
    │   ├── contract/
    │   │   └── contract.go    # Task DTOs, service contracts, emitted events
    │   ├── module.go          # ServiceProvider, Dependent, EventEmitter, ServiceConsumer
    │   ├── ports.go           # UserPort (driven port owned by task)
    │   ├── adapter.go         # userAdapter: UserPort over user/contract
    │   ├── service.go         # Business logic handlers
    │   ├── repository.go      # In-memory storage
    │   └── module_test.go     # monotest.CheckModule conformance test
    ├── user/                  # User Module
    │   ├── contract/
    │   │   └── contract.go    # User DTOs and service contracts
    │   ├── module.go          # ServiceProviderModule
    │   ├── repository.go      # In-memory storage with demo users
    │   └── module_test.go     # monotest.CheckModule conformance test
    └── notification/          # Driven Adapter (Event Consumer)
        ├── module.go          # EventConsumerModule (uses task/contract events)
        └── module_test.go     # monotest.CheckModule conformance test
```

## Mono Framework Interfaces

| Interface | Purpose | Implemented By |
|-----------|---------|----------------|
| `ServiceProviderModule` | Register services through contracts | user, task |
| `DependentModule` | Declare and receive dependencies | api, task |
| `types.ServiceConsumerModule` | Declare consumed contracts (validated at startup) | api, task |
| `EventEmitterModule` | Declare emitted events | task |
| `EventBusAwareModule` | Receive event bus for publishing | task |
| `EventConsumerModule` | Subscribe to events | notification |
| `HealthCheckableModule` | Health checks | api |

## Dependency Wiring

```go
// main.go: register modules, the framework handles the wiring.
app, _ := mono.NewMonoApplication(
    mono.WithShutdownTimeout(shutdownTimeout),
    mono.WithStrictModuleBoundaries(),
)
for _, m := range []mono.Module{user.NewModule(), notification.NewModule(), task.NewModule(), api.NewModule()} {
    if err := app.Register(m); err != nil {
        log.Fatal(err)
    }
}
```

### Cross-Module Communication

The provider declares the contract once:

```go
// modules/user/contract/contract.go
var ValidateUser = helper.RequestReplyService[ValidateUserRequest, ValidateUserResponse]("user", "validate-user")

// modules/user/module.go: driving adapter
func (m *UserModule) RegisterServices(container mono.ServiceContainer) error {
    if err := contract.GetUser.Register(container, m.getUser); err != nil {
        return err
    }
    return contract.ValidateUser.Register(container, m.validateUser)
}
```

The consumer owns the port and the adapter, and declares what it consumes:

```go
// modules/task/ports.go
type UserPort interface {
    ValidateUser(ctx context.Context, req usercontract.ValidateUserRequest) (usercontract.ValidateUserResponse, error)
}

// modules/task/adapter.go
type userAdapter struct{ container mono.ServiceContainer }

func (a userAdapter) ValidateUser(ctx context.Context, req usercontract.ValidateUserRequest) (usercontract.ValidateUserResponse, error) {
    return usercontract.ValidateUser.Call(ctx, a.container, req)
}

// modules/task/module.go
func (m *TaskModule) Dependencies() []string { return []string{"user"} }

func (m *TaskModule) ConsumeServices() []types.BaseServiceDefinition {
    return []types.BaseServiceDefinition{usercontract.ValidateUser.ToBase()}
}

func (m *TaskModule) SetDependencyServiceContainer(dep string, container mono.ServiceContainer) {
    if dep == "user" {
        m.userPort = userAdapter{container: container}
    }
}
```

Event consumers use the emitter's definitions, never a name string:

```go
// modules/notification/module.go
helper.RegisterTypedEventConsumer(registry, taskcontract.TaskCreatedV1, m.handleTaskCreated, m)
```

## Module Boundaries

- **Contracts**: service names, subjects (`services.task.create-task`, ...) and
  JSON encoding live only in `<module>/contract`. Module code never writes a
  service name as a string.
- **`ConsumeServices()`**: api and task list every contract they call; startup
  fails fast if a dependency does not provide one.
- **Strict mode**: `main.go` enables `mono.WithStrictModuleBoundaries()`, so a
  module that registers on a dependency's container or calls an undeclared
  module's `services.*` subject is rejected.
- **Conformance tests**: each module has a `module_test.go` that runs
  `monotest.CheckModule` with stand-in dependency containers (no NATS) and an
  exercise that calls every provided and consumed contract.
  `boundaries_test.go` starts the whole app in recording mode and asserts that
  `app.ModuleBoundaries()` reports no violations and no unused dependencies.

```bash
go test ./...                                  # everything
go test ./modules/... -run Conformance -v      # per-module conformance only
```

## Communication Patterns

1. **Ports and Adapters (Synchronous)**
   - API → Task: `api.TaskPort` over `task/contract` (CreateTask, GetTask, ...)
   - Task → User: `task.UserPort` over `user/contract` (ValidateUser)

2. **Events (Asynchronous)**
   - Task publishes `TaskCreatedV1`, `TaskCompletedV1`, `TaskDeletedV1` (from `task/contract`)
   - Notification subscribes and logs/processes events

## REST API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/tasks | Create a new task |
| GET | /api/v1/tasks | List all tasks (optional: ?user_id=xxx) |
| GET | /api/v1/tasks/:id | Get a task by ID |
| PUT | /api/v1/tasks/:id | Update a task |
| DELETE | /api/v1/tasks/:id | Delete a task |
| POST | /api/v1/tasks/:id/complete | Mark task as completed |
| GET | /health | Health check |

## Running the Application

```bash
# Navigate to project directory
cd projects/hexagonal-architecture

# Download dependencies
go mod tidy

# Build and run the application
go build -o bin/hexagonal-architecture .
./bin/hexagonal-architecture

# Run the tests (module conformance + whole-app boundary report)
go test ./...
```

## Example Usage

### Create a Task
```bash
curl -X POST http://localhost:3000/api/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Buy groceries",
    "description": "Milk, eggs, bread",
    "user_id": "user-1"
  }'
```

### List Tasks
```bash
curl http://localhost:3000/api/v1/tasks
```

### Get a Task
```bash
curl http://localhost:3000/api/v1/tasks/{task-id}
```

### Complete a Task
```bash
curl -X POST http://localhost:3000/api/v1/tasks/{task-id}/complete
```

### Delete a Task
```bash
curl -X DELETE http://localhost:3000/api/v1/tasks/{task-id}
```

## Demo Users

The application comes with pre-seeded demo users:

| User ID | Name | Email |
|---------|------|-------|
| user-1 | Alice Johnson | alice@example.com |
| user-2 | Bob Smith | bob@example.com |
| user-3 | Charlie Brown | charlie@example.com |

## Key Hexagonal Patterns Demonstrated

1. **Contracts**: `task/contract` and `user/contract` are the published ports, shared by provider and consumer
2. **Consumer-owned ports**: `api.TaskPort` and `task.UserPort` describe only what each consumer needs
3. **Adapters**: `taskAdapter` and `userAdapter` implement those ports over the provider's contract
4. **Driving Adapter**: API module calls into core domain via TaskPort
5. **Driven Adapter**: Notification module reacts to domain events from `task/contract`
6. **Domain Isolation**: Task business logic has no HTTP or notification dependencies
7. **Checked boundaries**: `ConsumeServices()`, strict mode and `monotest.CheckModule` tests

## Graceful Shutdown

The application handles `SIGINT` and `SIGTERM` signals for graceful shutdown:
- HTTP server stops accepting new requests
- In-flight requests are allowed to complete
- All modules shut down in reverse dependency order

## Dependencies

- [go-monolith/mono](https://github.com/go-monolith/mono) v0.1.1 - Modular monolith framework (typed contracts, module boundaries, `monotest`)
- [gofiber/fiber](https://github.com/gofiber/fiber) v2 - Fast HTTP server
- [gelmium/graceful-shutdown](https://github.com/gelmium/graceful-shutdown) - Signal handling
- [google/uuid](https://github.com/google/uuid) - UUID generation

## License

MIT
