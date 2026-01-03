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
                    │  Uses TaskPort interface         │
                    └──────────────┬───────────────────┘
                                   │ TaskPort
                    ┌──────────────▼───────────────────┐
                    │        Task Module               │
                    │       [Core Domain]              │
                    │  Business logic, publishes       │
                    │  events to EventBus              │
                    └───────┬──────────────┬───────────┘
                            │              │
               UserPort     │              │ Events
                            │              │
    ┌───────────────────────▼──┐    ┌──────▼───────────────────┐
    │      User Module         │    │   Notification Module    │
    │  User validation and     │    │     [Driven Adapter]     │
    │  storage                 │    │   Subscribes to events   │
    └──────────────────────────┘    └──────────────────────────┘
```

## Hexagonal Architecture Concepts

### Ports (Interfaces)
- **TaskPort**: Defines contract for task operations (`modules/task/types.go`)
- **UserPort**: Defines contract for user validation (`modules/user/adapter.go`)

### Adapters
- **Driving Adapter**: API module receives HTTP requests and calls the core domain via TaskPort
- **Driven Adapter**: Notification module reacts to domain events

### Domain Isolation
The Task module (core domain) has no knowledge of HTTP or notification systems. It only depends on abstract interfaces (ports).

## Project Structure

```
hexagonal-architecture/
├── main.go                    # Application bootstrap (framework handles DI)
├── go.mod                     # Go module definition
├── events/
│   └── task_events.go         # Typed event definitions using helper.EventDefinition
├── domain/
│   └── task/
│       └── entity.go          # Task domain entity
└── modules/
    ├── api/                   # Driving Adapter (HTTP)
    │   ├── module.go          # DependentModule (depends on task)
    │   ├── handlers.go        # REST endpoint handlers
    │   └── dto.go             # Request/Response DTOs
    ├── task/                  # Core Domain
    │   ├── module.go          # ServiceProviderModule, DependentModule, EventEmitterModule
    │   ├── service.go         # Business logic handlers
    │   ├── repository.go      # In-memory storage
    │   ├── adapter.go         # TaskAdapter using helper.CallRequestReplyService
    │   └── types.go           # Types and TaskPort interface
    ├── user/                  # User Module
    │   ├── module.go          # ServiceProviderModule
    │   ├── repository.go      # In-memory storage with demo users
    │   ├── adapter.go         # UserAdapter using helper.CallRequestReplyService
    │   └── types.go           # User types
    └── notification/          # Driven Adapter (Event Consumer)
        └── module.go          # EventConsumerModule
```

## Mono Framework Interfaces

This project demonstrates proper use of mono framework interfaces:

| Interface | Purpose | Implemented By |
|-----------|---------|----------------|
| `ServiceProviderModule` | Register request-reply services | user, task |
| `DependentModule` | Declare and receive dependencies | api, task |
| `EventEmitterModule` | Declare emitted events | task |
| `EventBusAwareModule` | Receive event bus for publishing | task |
| `EventConsumerModule` | Subscribe to events | notification |
| `HealthCheckableModule` | Health checks | api |

## Dependency Wiring

The mono framework automatically handles dependency injection:

```go
// main.go - Just register modules, framework handles the rest
app.Register(user.NewModule())         // Independent module
app.Register(notification.NewModule()) // Event consumer
app.Register(task.NewModule())         // Depends on user, emits events
app.Register(api.NewModule())          // Depends on task

// Framework automatically calls:
// - ServiceProviderModule.RegisterServices() for request-reply services
// - DependentModule.SetDependencyServiceContainer() for cross-module calls
// - EventBusAwareModule.SetEventBus() for event publishing
// - EventConsumerModule.RegisterEventConsumers() for event subscriptions
```

### Cross-Module Communication

```go
// Task module declares dependency on user module
func (m *TaskModule) Dependencies() []string {
    return []string{"user"}
}

// Framework injects user module's ServiceContainer
func (m *TaskModule) SetDependencyServiceContainer(dep string, container mono.ServiceContainer) {
    switch dep {
    case "user":
        m.userPort = user.NewUserAdapter(container)  // Create adapter from container
    }
}
```

## Communication Patterns

1. **Ports and Adapters (Synchronous)**
   - API → Task: Via TaskPort interface (CreateTask, GetTask, etc.)
   - Task → User: Via UserPort interface (ValidateUser)

2. **Events (Asynchronous)**
   - Task publishes: `TaskCreatedEvent`, `TaskCompletedEvent`, `TaskDeletedEvent`
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

# Run the application
go run main.go
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

1. **Ports (Interfaces)**: `TaskPort` and `UserPort` define contracts between modules
2. **Adapters**: `TaskAdapter` and `UserAdapter` implement port interfaces
3. **Driving Adapter**: API module calls into core domain via TaskPort
4. **Driven Adapter**: Notification module reacts to domain events
5. **Domain Isolation**: Task business logic has no HTTP or notification dependencies
6. **Dependency Inversion**: Core domain depends on abstractions (ports), not implementations

## Graceful Shutdown

The application handles `SIGINT` and `SIGTERM` signals for graceful shutdown:
- HTTP server stops accepting new requests
- In-flight requests are allowed to complete
- All modules shut down in reverse registration order

## Dependencies

- [go-monolith/mono](https://github.com/go-monolith/mono) v0.0.2 - Modular monolith framework
- [gofiber/fiber](https://github.com/gofiber/fiber) v2 - Fast HTTP server
- [gelmium/graceful-shutdown](https://github.com/gelmium/graceful-shutdown) - Signal handling
- [google/uuid](https://github.com/google/uuid) - UUID generation

## License

MIT
