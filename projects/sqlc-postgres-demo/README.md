# sqlc + PostgreSQL Recipe

A demonstration project showcasing how to integrate **sqlc** (type-safe SQL code generation) with PostgreSQL in a modular monolith application using the [go-monolith/mono](https://github.com/go-monolith/mono) framework (v0.1.1).

This recipe demonstrates the **ServiceProviderModule** pattern for exposing CRUD services via the mono framework's request-reply mechanism, without any HTTP endpoints.

## Why Use sqlc with PostgreSQL?

### Benefits of sqlc

1. **Compile-Time Safety**: SQL queries are validated at code generation time, not runtime. Typos, missing columns, and type mismatches are caught before your code runs.

2. **Zero Runtime Overhead**: Generated code is pure Go with no reflection or query building. It's as fast as hand-written code.

3. **SQL First**: Write natural SQL, get type-safe Go. No learning curve for SQL experts, and no ORM query language to master.

4. **IDE Support**: Generated types enable full autocomplete and refactoring support in your editor.

5. **Database Schema as Truth**: The schema.sql file is the single source of truth. Changes to the schema immediately surface as compile errors.

### sqlc vs GORM Comparison

| Aspect | sqlc | GORM |
|--------|------|------|
| **Query Style** | Write raw SQL | ORM DSL/method chaining |
| **Type Safety** | Compile-time | Runtime |
| **Learning Curve** | Just SQL | ORM concepts required |
| **Performance** | Direct SQL, zero overhead | Query building overhead |
| **Flexibility** | Full SQL power | May need raw SQL for complex queries |
| **Migrations** | Manual (schema.sql) | Auto-migrations available |
| **Code Generation** | Required (sqlc generate) | Not required |
| **Complex Queries** | Native SQL support | Can be awkward |

### When to Choose sqlc

Choose sqlc when:
- Your team is comfortable with SQL
- You need maximum performance
- You have complex queries (joins, CTEs, window functions)
- You want compile-time query validation
- You prefer explicit over implicit behavior

### When to Choose GORM

Choose GORM when:
- Rapid prototyping is priority
- You need auto-migrations
- Simple CRUD is sufficient
- Team prefers ORM abstractions
- You want soft deletes, hooks, associations out of the box

### PostgreSQL vs SQLite Considerations

| Use Case | PostgreSQL | SQLite |
|----------|------------|--------|
| Multi-instance deployment | Best choice | Not suitable |
| Write-heavy concurrent | Excellent | Limited |
| Full-text search | Powerful built-in | Basic |
| JSON operations | Rich support | Limited |
| Geospatial data | PostGIS extension | Not available |
| Single-user/embedded | Overkill | Perfect |
| Development simplicity | Requires server | Zero config |

## How Request-Reply Services Work in Mono

The mono framework uses NATS as its message bus. The `ServiceProviderModule` interface allows modules to register request-reply services that respond to NATS request messages.

### Architecture

```
┌─────────────────┐     NATS Request      ┌─────────────────┐
│   NATS Client   │ ──────────────────────▶│    UserModule   │
│  (nats CLI)     │                        │                 │
│                 │     NATS Reply         │  ┌───────────┐  │
│                 │ ◀──────────────────────│  │Repository │  │
└─────────────────┘                        │  └─────┬─────┘  │
                                           │        │        │
                                           │  ┌─────▼─────┐  │
                                           │  │   sqlc    │  │
                                           │  │ Queries   │  │
                                           │  └─────┬─────┘  │
                                           │        │        │
                                           │  ┌─────▼─────┐  │
                                           │  │PostgreSQL │  │
                                           │  └───────────┘  │
                                           └─────────────────┘
```

### Service Contracts and Registration

Each service is declared once, as a typed contract in `modules/user/contract`:

```go
// modules/user/contract/contract.go
var (
    CreateUser = helper.RequestReplyService[CreateUserRequest, UserResponse]("user", "create")
    GetUser    = helper.RequestReplyService[GetUserRequest, UserResponse]("user", "get")
    ListUsers  = helper.RequestReplyService[ListUsersRequest, ListUsersResponse]("user", "list")
    UpdateUser = helper.RequestReplyService[UpdateUserRequest, UserResponse]("user", "update")
    DeleteUser = helper.RequestReplyService[DeleteUserRequest, DeleteUserResponse]("user", "delete")
)
```

The `UserModule` implements `ServiceProviderModule` and registers its handlers
through those contracts:

```go
func (m *UserModule) RegisterServices(container mono.ServiceContainer) error {
    if err := contract.CreateUser.Register(container, m.handleCreate); err != nil {
        return err
    }
    // ... GetUser, ListUsers, UpdateUser, DeleteUser
    return nil
}
```

The subject is `services.<module>.<service>`, so `create` becomes
`services.user.create`. Callers use the same contract value:
`contract.CreateUser.Call(ctx, container, contract.CreateUserRequest{...})`.

## Project Structure

```
sqlc-postgres-demo/
├── main.go                         # Application entry point
├── go.mod                          # Go module definition
├── docker-compose.yml              # PostgreSQL container
├── modules/
│   └── user/
│       ├── contract/
│       │   └── contract.go         # Published port: DTOs + typed service contracts
│       ├── db/
│       │   ├── sqlc.yaml           # sqlc configuration
│       │   ├── schema.sql          # Database schema
│       │   ├── query.sql           # SQL queries with annotations
│       │   └── generated/          # sqlc generated code (private to the module)
│       │       ├── db.go
│       │       ├── models.go
│       │       └── query.sql.go
│       ├── module.go               # ServiceProviderModule
│       ├── module_test.go          # monotest.CheckModule conformance tests
│       ├── repository.go           # Wraps sqlc queries
│       ├── repository_test.go      # Repository tests (need PostgreSQL)
│       ├── service.go              # Business logic, maps sqlc rows to contract DTOs
│       └── service_test.go         # Service tests with a mock repository
├── demo.sh                         # NATS CLI demo script
└── README.md
```

## Available Services

| Service | Subject | Description |
|---------|---------|-------------|
| Create User | `services.user.create` | Create a new user |
| Get User | `services.user.get` | Get user by ID |
| List Users | `services.user.list` | List users with pagination |
| Update User | `services.user.update` | Update user by ID |
| Delete User | `services.user.delete` | Delete user by ID |

## Running the Application

### Prerequisites

- Go 1.25 or later
- Docker and Docker Compose
- sqlc CLI (for code generation)
- NATS CLI (`nats` command) for the demo script
- jq (for JSON formatting in demo)

### Install sqlc

```bash
# macOS
brew install sqlc

# Linux/Go
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Or download from releases
# https://github.com/sqlc-dev/sqlc/releases
```

### Start PostgreSQL

```bash
cd projects/sqlc-postgres-demo
docker compose up -d
```

### Generate sqlc Code

Only needed if you modify `modules/user/db/query.sql` or `modules/user/db/schema.sql`:

```bash
cd modules/user/db && sqlc generate
```

### Build

```bash
go build -o bin/sqlc-postgres-demo .
```

### Run

```bash
./bin/sqlc-postgres-demo
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://demo:demo123@localhost:5432/users_db?sslmode=disable` |

## Testing with NATS CLI

### Create a User

```bash
nats request services.user.create '{"name":"Alice","email":"alice@example.com"}'
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Alice",
  "email": "alice@example.com",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Get a User

```bash
nats request services.user.get '{"id":"550e8400-e29b-41d4-a716-446655440000"}'
```

### List Users with Pagination

```bash
# Default (limit=10, offset=0)
nats request services.user.list '{}'

# Custom pagination
nats request services.user.list '{"limit":5,"offset":10}'
```

**Response:**
```json
{
  "users": [...],
  "total": 100,
  "limit": 5,
  "offset": 10
}
```

### Update a User

```bash
nats request services.user.update '{"id":"...","name":"Alice Updated"}'
```

### Delete a User

```bash
nats request services.user.delete '{"id":"..."}'
```

## Running the Demo

The demo script performs full CRUD operations with direct PostgreSQL verification:

```bash
./demo.sh
```

This script demonstrates:
- Creating multiple users
- Listing with pagination
- Getting by ID
- Updating user data
- Deleting users
- Direct `psql` verification of data
- Input validation error handling

## Running Tests

```bash
# Make sure PostgreSQL is running
docker compose up -d

# Run tests
go test ./...
```

Tests that need PostgreSQL (`repository_test.go`, `TestUserModuleConformancePostgres`)
skip gracefully if the database at `TEST_DATABASE_URL` (default: the
docker-compose database) is unavailable. The other tests, including the
`monotest.CheckModule` conformance test with an in-memory service, always run.

## Module Boundaries

The user module follows mono's Ports and Adapters rules:

- `modules/user/contract` is the module's published port: request/response
  DTOs and the typed contracts (`helper.RequestReplyService`). It imports only
  the standard library and `mono/pkg/helper`, never the module package or the
  sqlc-generated `db/generated` package; `service.go` maps `generated.User`
  rows to the contract DTOs. It is the only package of the module other
  modules may import.
- `RegisterServices` registers every handler through its contract, so no
  service name is written as a string in module code. Service names and
  subjects (`services.user.<service>`) are unchanged.
- `main.go` runs with `mono.WithStrictModuleBoundaries()`, which rejects
  boundary violations instead of only reporting them when the app stops.
- `modules/user/module_test.go` runs `monotest.CheckModule` and calls every
  contract, once with an in-memory `UserService` injected through
  `user.NewModuleWithService` (no database needed) and once against
  PostgreSQL when it is reachable:

```bash
go test ./modules/user/ -run Conformance -v
```

## sqlc Configuration

The `modules/user/db/sqlc.yaml` configures code generation:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "query.sql"
    schema: "schema.sql"
    gen:
      go:
        package: "generated"
        out: "generated"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
```

Key options:
- `sql_package: "pgx/v5"` - Uses modern pgx driver for better performance
- `emit_json_tags: true` - Adds JSON tags for API responses
- `emit_empty_slices: true` - Returns `[]` instead of `nil` for empty results

## Dependencies

- [github.com/go-monolith/mono](https://github.com/go-monolith/mono) - Modular monolith framework
- [github.com/jackc/pgx/v5](https://github.com/jackc/pgx) - PostgreSQL driver
- [github.com/sqlc-dev/sqlc](https://github.com/sqlc-dev/sqlc) - SQL compiler

## Cleanup

```bash
# Stop and remove PostgreSQL container and data
docker compose down -v
```

## License

This is a demonstration project for educational purposes.
