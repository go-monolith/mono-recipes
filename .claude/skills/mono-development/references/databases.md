# Database Integration

Mono applications can integrate with various databases. This reference covers GORM (ORM) and sqlc (code generation) patterns.

## Framework Selection

| Framework | Approach | Use Cases |
|-----------|----------|-----------|
| **GORM** | ORM with migrations | Rapid development, complex relationships |
| **sqlc** | SQL-first code generation | Type-safe queries, performance-critical |

## GORM with SQLite Module

### Basic Module Structure

```go
package product

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/go-monolith/mono"
    "github.com/go-monolith/mono/pkg/helper"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

type ProductModule struct {
    db     *gorm.DB
    repo   *Repository
    dbPath string
}

var (
    _ mono.Module                = (*ProductModule)(nil)
    _ mono.ServiceProviderModule = (*ProductModule)(nil)
    _ mono.HealthCheckableModule = (*ProductModule)(nil)
)

func NewModule() *ProductModule {
    dbPath := os.Getenv("DB_PATH")
    if dbPath == "" {
        dbPath = "products.db"
    }
    return &ProductModule{
        dbPath: dbPath,
    }
}

func (m *ProductModule) Name() string { return "product" }
```

### Entity Definition

```go
package product

import (
    "time"

    "gorm.io/gorm"
)

type Product struct {
    ID          uint           `gorm:"primaryKey" json:"id"`
    Name        string         `gorm:"size:255;not null" json:"name"`
    Description string         `gorm:"size:1000" json:"description"`
    Price       float64        `gorm:"not null" json:"price"`
    Stock       int            `gorm:"default:0" json:"stock"`
    Active      bool           `gorm:"default:true" json:"active"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
```

### Module Start with Migration

```go
func (m *ProductModule) Start(_ context.Context) error {
    log.Printf("[product] Connecting to SQLite database: %s", m.dbPath)

    // Configure GORM logger
    logLevel := logger.Silent
    if os.Getenv("DB_DEBUG") == "true" {
        logLevel = logger.Info
    }

    db, err := gorm.Open(sqlite.Open(m.dbPath), &gorm.Config{
        Logger: logger.Default.LogMode(logLevel),
    })
    if err != nil {
        return fmt.Errorf("failed to connect to database: %w", err)
    }

    m.db = db

    // Auto-migrate models
    if err := m.db.AutoMigrate(&Product{}); err != nil {
        return fmt.Errorf("failed to run migrations: %w", err)
    }

    // Initialize repository
    m.repo = NewRepository(m.db)

    log.Println("[product] Module started successfully")
    return nil
}
```

### Health Check

```go
func (m *ProductModule) Health(ctx context.Context) mono.HealthStatus {
    if m.db == nil {
        return mono.HealthStatus{
            Healthy: false,
            Message: "database not initialized",
        }
    }

    sqlDB, err := m.db.DB()
    if err != nil {
        return mono.HealthStatus{
            Healthy: false,
            Message: fmt.Sprintf("failed to get sql.DB: %v", err),
        }
    }

    if err := sqlDB.PingContext(ctx); err != nil {
        return mono.HealthStatus{
            Healthy: false,
            Message: fmt.Sprintf("database ping failed: %v", err),
        }
    }

    return mono.HealthStatus{
        Healthy: true,
        Message: "operational",
        Details: map[string]any{
            "driver": "sqlite",
            "path":   m.dbPath,
        },
    }
}
```

### Graceful Shutdown

```go
func (m *ProductModule) Stop(_ context.Context) error {
    if m.db == nil {
        return nil
    }

    log.Println("[product] Closing database connection...")

    sqlDB, err := m.db.DB()
    if err != nil {
        return fmt.Errorf("failed to get sql.DB: %w", err)
    }

    if err := sqlDB.Close(); err != nil {
        return fmt.Errorf("failed to close database: %w", err)
    }

    log.Println("[product] Database connection closed")
    return nil
}
```

### Repository Pattern

```go
package product

import "gorm.io/gorm"

type Repository struct {
    db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) Create(product *Product) error {
    return r.db.Create(product).Error
}

func (r *Repository) FindByID(id uint) (*Product, error) {
    var product Product
    if err := r.db.First(&product, id).Error; err != nil {
        return nil, err
    }
    return &product, nil
}

func (r *Repository) FindAll(limit, offset int) ([]Product, error) {
    var products []Product
    err := r.db.Limit(limit).Offset(offset).Find(&products).Error
    return products, err
}

func (r *Repository) Update(product *Product) error {
    return r.db.Save(product).Error
}

func (r *Repository) Delete(id uint) error {
    return r.db.Delete(&Product{}, id).Error
}
```

## sqlc with PostgreSQL Module

### Basic Module Structure

```go
package user

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/go-monolith/mono"
    "github.com/go-monolith/mono/pkg/helper"
    "github.com/jackc/pgx/v5/pgxpool"
)

type UserModule struct {
    pool  *pgxpool.Pool
    repo  *Repository
    dbURL string
}

var (
    _ mono.Module                = (*UserModule)(nil)
    _ mono.ServiceProviderModule = (*UserModule)(nil)
    _ mono.HealthCheckableModule = (*UserModule)(nil)
)

func NewModule() *UserModule {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        dbURL = "postgres://demo:demo123@localhost:5432/users_db?sslmode=disable"
    }
    return &UserModule{
        dbURL: dbURL,
    }
}

func (m *UserModule) Name() string { return "user" }
```

### Module Start with Connection Pool

```go
func (m *UserModule) Start(ctx context.Context) error {
    log.Printf("[user] Connecting to PostgreSQL...")

    pool, err := pgxpool.New(ctx, m.dbURL)
    if err != nil {
        return fmt.Errorf("failed to create connection pool: %w", err)
    }

    // Verify connection
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return fmt.Errorf("failed to ping database: %w", err)
    }

    m.pool = pool
    m.repo = NewRepository(pool)

    log.Println("[user] Module started successfully")
    return nil
}
```

### Health Check

```go
func (m *UserModule) Health(ctx context.Context) mono.HealthStatus {
    if m.pool == nil {
        return mono.HealthStatus{
            Healthy: false,
            Message: "database pool not initialized",
        }
    }

    if err := m.pool.Ping(ctx); err != nil {
        return mono.HealthStatus{
            Healthy: false,
            Message: fmt.Sprintf("database ping failed: %v", err),
        }
    }

    return mono.HealthStatus{
        Healthy: true,
        Message: "operational",
        Details: map[string]any{
            "driver": "pgx/v5",
            "pool":   "postgresql",
        },
    }
}
```

### Graceful Shutdown

```go
func (m *UserModule) Stop(_ context.Context) error {
    if m.pool == nil {
        return nil
    }

    log.Println("[user] Closing database connection pool...")
    m.pool.Close()
    log.Println("[user] Database connection pool closed")
    return nil
}
```

### Module-Owned Database Structure (SOLID)

Each module owns its database schema, queries, and generated code:

```
modules/user/
├── module.go
├── service.go
├── repository.go          # Repository interface
└── db/                    # Module-owned database
    ├── sqlc.yaml          # sqlc config for this module
    ├── schema.sql         # Schema owned by this module
    ├── queries/
    │   └── users.sql
    └── generated/         # sqlc generated code
        ├── db.go
        ├── models.go
        └── query.sql.go
```

### sqlc Configuration (modules/user/db/sqlc.yaml)

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "schema.sql"
    gen:
      go:
        package: "generated"
        out: "generated"
        sql_package: "pgx/v5"
        emit_json_tags: true
```

Run from module's db directory: `cd modules/user/db && sqlc generate`

### SQL Schema (modules/user/db/schema.sql)

```sql
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### SQL Queries (modules/user/db/queries/users.sql)

```sql
-- name: CreateUser :one
INSERT INTO users (email, name, active)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY id LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET name = $2, active = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
```

### Repository with sqlc

```go
package user

import (
    "context"

    "github.com/example/my-app/modules/user/db/generated"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
    pool    *pgxpool.Pool
    queries *generated.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
    return &Repository{
        pool:    pool,
        queries: generated.New(pool),
    }
}

func (r *Repository) Create(ctx context.Context, email, name string) (*generated.User, error) {
    return r.queries.CreateUser(ctx, generated.CreateUserParams{
        Email:  email,
        Name:   name,
        Active: true,
    })
}

func (r *Repository) FindByID(ctx context.Context, id int32) (*generated.User, error) {
    return r.queries.GetUser(ctx, id)
}
```

## Service Registration Pattern

Both GORM and sqlc modules follow the same service registration pattern:

```go
func (m *UserModule) RegisterServices(container mono.ServiceContainer) error {
    services := []struct {
        name    string
        handler func(ctx context.Context, req RequestType, msg *mono.Msg) (ResponseType, error)
    }{
        {"create", m.createUser},
        {"get", m.getUser},
        {"list", m.listUsers},
        {"update", m.updateUser},
        {"delete", m.deleteUser},
    }

    for _, svc := range services {
        if err := helper.RegisterTypedRequestReplyService(
            container, svc.name, json.Unmarshal, json.Marshal, svc.handler,
        ); err != nil {
            return fmt.Errorf("failed to register %s service: %w", svc.name, err)
        }
    }

    log.Printf("[user] Registered services: services.user.{create,get,list,update,delete}")
    return nil
}
```

## Example Projects

| Project | Framework | Database |
|---------|-----------|----------|
| `gorm-sqlite-demo` | GORM | SQLite |
| `sqlc-postgres-demo` | sqlc | PostgreSQL |

## Best Practices

1. **Use connection pooling** - pgxpool for PostgreSQL, GORM manages internally for SQLite
2. **Verify connections in Start()** - Ping the database before proceeding
3. **Implement health checks** - Use HealthCheckableModule interface
4. **Close connections in Stop()** - Always clean up resources
5. **Use environment variables** - DATABASE_URL or DB_PATH for configuration
6. **Log database operations** - Enable debug logging conditionally
7. **Use repository pattern** - Separate database logic from module logic
8. **Run migrations in Start()** - AutoMigrate for GORM, external tools for sqlc
