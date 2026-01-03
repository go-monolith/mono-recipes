# GORM + SQLite Recipe

A demonstration project showcasing how to integrate GORM (Go ORM) with SQLite in a modular monolith application using the [go-monolith/mono](https://github.com/go-monolith/mono) framework.

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

### Service Registration

```go
// In RegisterServices():
helper.RegisterTypedRequestReplyService(
    container,
    "product.create",     // NATS subject
    json.Unmarshal,       // Request decoder
    json.Marshal,         // Response encoder
    m.createProduct,      // Handler function
)
```

### Service Handler

```go
func (m *ProductModule) createProduct(
    ctx context.Context,
    req CreateProductRequest,
    msg *mono.Msg,
) (CreateProductResponse, error) {
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
│       ├── module.go               # ServiceProviderModule implementation
│       ├── entity.go               # GORM model definition
│       ├── repository.go           # Data access layer
│       ├── service.go              # Service handlers
│       └── types.go                # Request/response types
├── go.mod
├── demo.sh                         # NATS CLI demo script
└── README.md
```

## Available Services

| Service | Subject | Description |
|---------|---------|-------------|
| Create Product | `product.create` | Create a new product |
| Get Product | `product.get` | Get product by ID |
| List Products | `product.list` | List all products |
| Update Product | `product.update` | Update product by ID |
| Delete Product | `product.delete` | Delete product by ID |

## Running the Application

### Prerequisites

- Go 1.21 or later
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
go build -o bin/gorm-sqlite-demo
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

func (r *Repository) Create(product *Product) error {
    return r.db.Create(product).Error
}

func (r *Repository) FindByID(id string) (*Product, error) {
    var product Product
    err := r.db.First(&product, "id = ?", id).Error
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
    return helper.RegisterTypedRequestReplyService(
        container,
        "product.create",
        json.Unmarshal,
        json.Marshal,
        m.createProduct,
    )
}
```

## Dependencies

- [github.com/go-monolith/mono](https://github.com/go-monolith/mono) - Modular monolith framework
- [gorm.io/gorm](https://gorm.io) - Go ORM library
- [gorm.io/driver/sqlite](https://gorm.io/docs/connecting_to_the_database.html#SQLite) - SQLite driver for GORM

## License

This is a demonstration project for educational purposes.
