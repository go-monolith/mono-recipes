# GORM + SQLite Recipe

A demonstration project showcasing how to integrate GORM (Go ORM) with SQLite in a modular monolith application using the [go-monolith/mono](https://github.com/go-monolith/mono) framework (v0.1.1).

This recipe demonstrates the **ServiceProviderModule** pattern for exposing CRUD services via the mono framework's request-reply mechanism, without any HTTP endpoints.

## Why Use GORM with SQLite?

### Benefits

1. **Rapid Prototyping**: SQLite requires zero configuration - no server setup, no connection strings to external databases. Start building immediately.

2. **Embedded Database**: The database is a single file that ships with your application. Perfect for:
   - Desktop applications
   - Edge/IoT devices
   - Development and testing environments
   - Serverless functions with persistent storage

3. **Zero Dependencies**: No need to install database servers. SQLite is compiled directly into your Go binary via CGO.

4. **ACID Compliant**: Full transaction support with rollback capabilities.

5. **GORM Features**: Auto-migrations, soft deletes, associations, hooks, and a rich query builder.

### Trade-offs vs Raw SQL

| Aspect | GORM | Raw SQL |
|--------|------|---------|
| **Learning Curve** | Higher (ORM concepts) | Lower (just SQL) |
| **Development Speed** | Faster (auto-migrations, helpers) | Slower (manual SQL) |
| **Performance** | Slight overhead | Direct execution |
| **Flexibility** | May need raw SQL for complex queries | Full control |
| **Type Safety** | Struct-based, compile-time checks | String-based, runtime errors |

### When to Choose SQLite vs PostgreSQL

| Use Case | SQLite | PostgreSQL |
|----------|--------|------------|
| Single-user applications | ✅ | Overkill |
| Embedded/edge devices | ✅ | Not suitable |
| Prototyping/development | ✅ | Setup overhead |
| Read-heavy workloads | ✅ | ✅ |
| Write-heavy concurrent | ❌ (limited) | ✅ |
| Multi-instance deployment | ❌ | ✅ |
| Full-text search | Basic | ✅ |
| JSON operations | Basic | ✅ |

## How Request-Reply Services Work in Mono

The mono framework uses NATS as its message bus. The `ServiceProviderModule` interface allows modules to register request-reply services that respond to NATS request messages.

### Architecture

```
┌─────────────────┐     NATS Request      ┌─────────────────┐
│   NATS Client   │ ──────────────────────▶│  ProductModule  │
│  (nats CLI)     │                        │                 │
│                 │     NATS Reply         │  ┌───────────┐  │
│                 │ ◀──────────────────────│  │Repository │  │
└─────────────────┘                        │  └───────────┘  │
                                           │        │        │
                                           │  ┌─────▼─────┐  │
                                           │  │  SQLite   │  │
                                           │  └───────────┘  │
                                           └─────────────────┘
```

### Service Contracts

Each service is declared once, as a typed contract in `modules/product/contract`:

```go
// modules/product/contract/contract.go
var CreateProduct = helper.RequestReplyService[CreateProductRequest, CreateProductResponse]("product", "create")
```

The module registers its handler through the contract (subject
`services.product.create`, JSON encoding included):

```go
// In RegisterServices():
contract.CreateProduct.Register(container, m.createProduct)
```

and any caller (another module's adapter, `main.go`, a test) calls through the
same value:

```go
resp, err := contract.CreateProduct.Call(ctx, container, contract.CreateProductRequest{Name: "Widget", Price: 9.99})
```

### Service Handler

```go
func (m *ProductModule) createProduct(
    ctx context.Context,
    req contract.CreateProductRequest,
    msg *mono.Msg,
) (contract.CreateProductResponse, error) {
    // Business logic here
    return response, nil
}
```

## Project Structure

```
gorm-sqlite-demo/
├── main.go                         # Application entry point
├── modules/
│   └── product/
│       ├── contract/
│       │   └── contract.go         # Published port: DTOs + typed service contracts
│       ├── module.go               # ServiceProviderModule implementation
│       ├── module_test.go          # monotest.CheckModule conformance test
│       ├── entity.go               # GORM model definition (private to the module)
│       ├── repository.go           # Data access layer
│       ├── repository_test.go      # Repository tests (in-memory SQLite)
│       └── service.go              # Service handlers
├── go.mod
├── demo.sh                         # NATS CLI demo script
└── README.md
```

## Available Services

| Service | Subject | Description |
|---------|---------|-------------|
| Create Product | `services.product.create` | Create a new product |
| Get Product | `services.product.get` | Get product by ID |
| List Products | `services.product.list` | List all products |
| Update Product | `services.product.update` | Update product by ID |
| Delete Product | `services.product.delete` | Delete product by ID |

## Running the Application

### Prerequisites

- Go 1.25 or later
- CGO enabled (required for SQLite)
- NATS CLI (`nats` command) for the demo script

### Install NATS CLI

```bash
# macOS
brew install nats-io/nats-tools/nats

# Linux
curl -sf https://binaries.nats.dev/nats-io/natscli/nats@latest | sh

# Or download from GitHub releases
# https://github.com/nats-io/natscli/releases
```

### Build

```bash
go build -o bin/gorm-sqlite-demo .
```

### Run

```bash
./bin/gorm-sqlite-demo
```

Or directly with Go:

```bash
go run .
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_PATH` | SQLite database file path | `products.db` |
| `DB_DEBUG` | Enable GORM SQL logging | `false` |

## Testing with NATS CLI

The mono framework starts an embedded NATS server. Use the `nats` CLI to send requests:

### Create a Product

```bash
nats request services.product.create '{"name":"Widget","description":"A useful widget","price":9.99,"stock":100}'
```

### Get a Product

```bash
nats request services.product.get '{"id":"<product-id>"}'
```

### List All Products

```bash
nats request services.product.list '{}'
```

### Update a Product

```bash
nats request services.product.update '{"id":"<product-id>","price":12.99,"stock":50}'
```

### Delete a Product

```bash
nats request services.product.delete '{"id":"<product-id>"}'
```

## Running the Demo

```bash
./demo.sh
```

This script demonstrates the full CRUD workflow using NATS CLI commands.

## Code Patterns

### GORM Model Definition

```go
type Product struct {
    ID          string         `gorm:"primarykey;size:36" json:"id"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
    Name        string         `gorm:"size:100;not null" json:"name"`
    Description string         `gorm:"size:500" json:"description"`
    Price       float64        `gorm:"not null" json:"price"`
    Stock       int            `gorm:"not null;default:0" json:"stock"`
}
```

### Repository Pattern

```go
type Repository struct {
    db *gorm.DB
}

// The handler's context is passed down so a query stops when the caller's
// request deadline expires.
func (r *Repository) Create(ctx context.Context, product *Product) error {
    return r.db.WithContext(ctx).Create(product).Error
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Product, error) {
    var product Product
    err := r.db.WithContext(ctx).First(&product, "id = ?", id).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, ErrNotFound
    }
    return &product, err
}
```

### ServiceProviderModule Implementation

```go
var _ mono.ServiceProviderModule = (*ProductModule)(nil)

func (m *ProductModule) RegisterServices(container mono.ServiceContainer) error {
    if err := contract.CreateProduct.Register(container, m.createProduct); err != nil {
        return err
    }
    // ... get, list, update, delete
    return nil
}
```

### SQLite and Concurrent Requests

Since mono v0.1.1, request-reply handlers run concurrently on a worker pool.
SQLite allows a single writer at a time, so `Start` limits the GORM connection
pool to one connection (`sqlDB.SetMaxOpenConns(1)`): concurrent requests queue
in `database/sql` instead of contending for the database file lock.
`TestProductModuleConcurrentRequests` covers this.

## Module Boundaries

The product module follows mono's Ports and Adapters rules:

- `modules/product/contract` is the module's published port: request/response
  DTOs and the typed contracts (`helper.RequestReplyService`). It imports only
  the standard library and `mono/pkg/helper`; the GORM entity stays private.
  It is the only package of the module other modules may import.
- `RegisterServices` registers every handler through its contract, so no
  service name is written as a string in module code. Service names and
  subjects (`services.product.<service>`) are unchanged.
- `main.go` runs with `mono.WithStrictModuleBoundaries()`, which rejects
  boundary violations instead of only reporting them when the app stops.
- `modules/product/module_test.go` runs `monotest.CheckModule` against a
  temporary SQLite file and calls every contract:

```bash
go test ./modules/product/ -run Conformance -v
```

## Dependencies

- [github.com/go-monolith/mono](https://github.com/go-monolith/mono) - Modular monolith framework
- [gorm.io/gorm](https://gorm.io) - Go ORM library
- [gorm.io/driver/sqlite](https://gorm.io/docs/connecting_to_the_database.html#SQLite) - SQLite driver for GORM

## License

This is a demonstration project for educational purposes.
