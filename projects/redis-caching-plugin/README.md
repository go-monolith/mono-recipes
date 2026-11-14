# Redis Caching Demo

A demonstration of Redis caching with the cache-aside pattern using the Mono framework (v0.1.1), Fiber HTTP framework, and GORM ORM. The Redis cache is a custom mono **plugin**; the HTTP API talks to the product module only through its typed service contracts.

## Overview

This recipe demonstrates:

- **Cache-Aside Pattern**: Check cache first, query database on miss, populate cache after database read
- **Custom Plugin**: A Redis cache implemented as a `mono.PluginModule` and injected with `SetPlugin`
- **Redis Integration**: Using `gofiber/storage/redis` (go-redis) behind the `mono/pkg/storage` interface
- **GORM ORM**: SQLite database with GORM for data persistence
- **Cache Invalidation**: Automatic cache invalidation on create, update, and delete operations
- **Module Boundaries**: `product/contract` typed request-reply contracts, a consumer-owned port and adapter in the API module, `ConsumeServices()`, strict mode, and `monotest` conformance tests

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Request                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Fiber HTTP Server (API Module)               │
│  Routes: /api/v1/products, /health                              │
│  ProductPort ──► productAdapter                                 │
└─────────────────────────────────────────────────────────────────┘
                                │  productcontract.X.Call(...)
                                │  services.product.<service> (NATS request-reply)
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Product Service (Product Module)             │
│   contract.X.Register(...) ──► Service (cache-aside pattern)    │
└─────────────────────────────────────────────────────────────────┘
                    │ SetPlugin("cache")    │
                    ▼                       ▼
┌──────────────────────────┐   ┌──────────────────────────────────┐
│    Redis Cache           │   │    SQLite Database               │
│    (Cache Plugin)        │   │    (GORM Repository)             │
│                          │   │                                  │
│  - Get/Set/Delete        │   │  - Product CRUD                  │
│  - InvalidateAll         │   │  - Auto Migration                │
│  - Health check          │   │                                  │
└──────────────────────────┘   └──────────────────────────────────┘
```

## Cache-Aside Pattern

The cache-aside pattern implemented in this demo:

### Read Operation (GetByID, List)
1. **Check Cache**: First, check if data exists in Redis cache
2. **Cache Hit**: If found, return cached data immediately
3. **Cache Miss**: Query SQLite database
4. **Populate Cache**: Store result in Redis with TTL
5. **Return Data**: Return data to client

### Write Operations (Create, Update, Delete)
1. **Execute Database Operation**: Perform the write to SQLite
2. **Invalidate Cache**: Clear the cache (`InvalidateAll`), so product and list keys are both refreshed on the next read

## Project Structure

```
redis-caching-plugin/
├── main.go                    # Application entry point (strict module boundaries)
├── go.mod                     # Go module definition
├── docker-compose.yml         # Redis container setup
├── demo.sh                    # Interactive demo script
├── README.md                  # This file
├── domain/
│   └── product/
│       ├── entity.go          # Product domain entity and input types
│       └── repository.go      # GORM repository
└── modules/
    ├── api/
    │   ├── module.go          # Fiber HTTP server, Dependencies(), ConsumeServices()
    │   ├── ports.go           # ProductPort: what the handlers need (consumer-owned)
    │   ├── adapter.go         # productAdapter: ProductPort over product/contract
    │   ├── handlers.go        # HTTP request handlers
    │   └── module_test.go     # monotest.CheckModule with a stand-in product module
    ├── cache/
    │   ├── module.go          # Cache plugin (mono.PluginModule)
    │   ├── service.go         # CacheService port over mono/pkg/storage
    │   └── cache_test.go      # CacheService tests (need Redis, skipped otherwise)
    └── product/
        ├── contract/
        │   └── contract.go    # Published port: DTOs + typed request-reply contracts
        ├── module.go          # Product module: SetPlugin, RegisterServices, Start/Stop
        ├── handlers.go        # Typed service handlers (contract DTOs <-> domain)
        ├── service.go         # Product service with cache-aside caching
        ├── service_test.go    # Service tests (need Redis, skipped otherwise)
        └── module_test.go     # monotest.CheckModule with the cache plugin (in-memory storage)
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/products` | List all products (cached) |
| GET | `/api/v1/products/:id` | Get product by ID (cached) |
| POST | `/api/v1/products` | Create new product |
| PUT | `/api/v1/products/:id` | Update product |
| DELETE | `/api/v1/products/:id` | Delete product |

### Response Format

All product responses include caching metadata:

```json
{
  "product": { ... },
  "from_cache": true,
  "duration_ms": 1
}
```

Cache hit/miss counters are not exposed over HTTP; use `redis-cli INFO stats` (`keyspace_hits`, `keyspace_misses`) or the `[cache] Cache Hit!/Miss!` log lines.

## Prerequisites

- Go 1.25 or later
- Docker and Docker Compose (for Redis)
- curl (for testing)

## Quick Start

1. **Start Redis**:
   ```bash
   docker-compose up -d
   ```

2. **Build and run the application**:
   ```bash
   go build -o bin/redis-caching-plugin .
   ./bin/redis-caching-plugin
   ```

3. **Run the demo** (in another terminal):
   ```bash
   ./demo.sh
   ```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `localhost:6379` | Redis server address |
| `DB_PATH` | `./products.db` | SQLite database path |
| `HTTP_PORT` | `3000` | HTTP server port |
| `CACHE_TTL` | `5m` | Cache entry TTL |
| `CACHE_PREFIX` | `product:` | Redis key prefix |

Example:
```bash
REDIS_ADDR=localhost:6379 CACHE_TTL=10m HTTP_PORT=8080 go run main.go
```

## Example Usage

### Create a Product
```bash
curl -X POST http://localhost:3000/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptop",
    "description": "High-performance laptop",
    "price": 999.99,
    "stock": 50,
    "category": "Electronics"
  }'
```

### Get a Product (demonstrates caching)
```bash
# First request - cache miss, queries database
curl http://localhost:3000/api/v1/products/1

# Second request - cache hit, returns from Redis
curl http://localhost:3000/api/v1/products/1
```

### List Products with Pagination
```bash
curl "http://localhost:3000/api/v1/products?offset=0&limit=10"
```

### Update a Product (invalidates cache)
```bash
curl -X PUT http://localhost:3000/api/v1/products/1 \
  -H "Content-Type: application/json" \
  -d '{"price": 899.99}'
```

### Delete a Product
```bash
curl -X DELETE http://localhost:3000/api/v1/products/1
```

## Key Implementation Details

### Cache Key Strategy

- Individual products: `product:id:<id>` (e.g., `product:id:1`)
- Product lists: `product:list:<offset>:<limit>` (e.g., `product:list:0:20`)

### Cache Invalidation Strategy

- **Create / Update / Delete**: `InvalidateAll` resets the Redis database, so every product and list key is re-read from SQLite on the next request. (This also clears keys outside the prefix; use a dedicated Redis database in shared environments.)

### Thread Safety

- Concurrent cache misses for the same product are collapsed with `singleflight` (no cache stampede)
- Redis operations are inherently thread-safe

### Error Handling

- Cache errors are logged but don't fail requests (graceful degradation)
- Database errors are propagated to the client
- Invalid requests return appropriate HTTP status codes

## Testing the Cache

1. **Create a product** and note the ID
2. **Get the product twice** - first request should show `"from_cache": false`, second shows `"from_cache": true`
3. **Update the product** - cache is invalidated
4. **Get the product again** - `"from_cache": false` (cache was invalidated)

## Module Boundaries

The API module never touches the product module's Go types. The product module publishes its services in `modules/product/contract`, the only package of it other modules import (it imports only the standard library and `mono/pkg/helper`):

```go
// modules/product/contract/contract.go
var GetProduct = helper.RequestReplyService[GetProductRequest, GetProductResponse]("product", "get-product")

// modules/product/module.go (provider, driving adapter)
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
    if err := contract.GetProduct.Register(container, m.getProduct); err != nil {
        return err
    }
    // ... list-products, create-product, update-product, delete-product
}
```

The API module owns a `ProductPort` (`ports.go`), implemented by `productAdapter` (`adapter.go`) over the read-only container it receives for its declared dependency, and lists every contract it calls in `ConsumeServices()`, so a missing or mismatched service fails at startup:

```go
func (m *Module) Dependencies() []string { return []string{"product"} }

func (m *Module) SetDependencyServiceContainer(dep string, c mono.ServiceContainer) {
    if dep == "product" {
        m.products = productAdapter{container: c}
    }
}

func (m *Module) ConsumeServices() []types.BaseServiceDefinition {
    return []types.BaseServiceDefinition{
        productcontract.ListProducts.ToBase(), productcontract.GetProduct.ToBase(),
        productcontract.CreateProduct.ToBase(), productcontract.UpdateProduct.ToBase(),
        productcontract.DeleteProduct.ToBase(),
    }
}
```

The cache plugin is not a dependency: the framework injects it with `SetPlugin("cache", plugin)`. The product module keeps the plugin and resolves `Port()` in its own `Start`, after the plugin has started (the Redis connection is opened in the plugin's `Start`).

`main.go` creates the app with `mono.WithStrictModuleBoundaries()`, so a boundary violation (for example a raw request on another module's `services.*` subject) fails instead of being logged at `Stop`.

### Conformance tests

```bash
go test ./modules/api/ ./modules/product/ -run Conformance -v
```

- `modules/api/module_test.go`: `monotest.CheckModule` with a stand-in `product` container whose services are registered through `product/contract`; the exercise drives every HTTP route, so every consumed contract is called.
- `modules/product/module_test.go`: `monotest.CheckModule` with the real cache plugin injected via `monotest.WithPlugin`, backed by in-memory storage (no Redis needed); the exercise calls every provided contract.

The `CacheService` and product `Service` tests need Redis on `localhost:6379` and are skipped when it is not reachable.

## Mono Framework Features Used

- **Plugin Module**: Redis cache as a `PluginModule`, injected via `UsePluginModule.SetPlugin`
- **Typed Service Contracts**: `helper.RequestReplyService` with `Register` / `Call`
- **Dependencies & ServiceConsumerModule**: declared dependency plus startup-validated `ConsumeServices()`
- **Strict Module Boundaries**: `mono.WithStrictModuleBoundaries()`
- **monotest**: NATS-free `CheckModule` conformance tests
- **Graceful Shutdown**: `gelmium/graceful-shutdown` calling `app.Stop` with a timeout

## License

MIT
