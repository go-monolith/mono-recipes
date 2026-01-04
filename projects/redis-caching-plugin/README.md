# Redis Caching Demo

A demonstration of Redis caching with the cache-aside pattern using the Mono framework, Fiber HTTP framework, and GORM ORM.

## Overview

This recipe demonstrates:

- **Cache-Aside Pattern**: Check cache first, query database on miss, populate cache after database read
- **Redis Integration**: Using go-redis for high-performance caching
- **GORM ORM**: SQLite database with GORM for data persistence
- **Cache Invalidation**: Automatic cache invalidation on create, update, and delete operations
- **Cache Statistics**: Real-time monitoring of cache hit/miss rates
- **Mono Framework**: Modular architecture with dependency injection

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Request                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Fiber HTTP Server (API Module)               │
│  Routes: /api/v1/products, /api/v1/cache/stats, /health         │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Product Service (Product Module)             │
│              Implements Cache-Aside Pattern                     │
└─────────────────────────────────────────────────────────────────┘
                    │                       │
                    ▼                       ▼
┌──────────────────────────┐   ┌──────────────────────────────────┐
│    Redis Cache           │   │    SQLite Database               │
│    (Cache Module)        │   │    (GORM Repository)             │
│                          │   │                                  │
│  - Get/Set/Delete        │   │  - Product CRUD                  │
│  - Pattern Delete        │   │  - Auto Migration                │
│  - Statistics Tracking   │   │                                  │
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
2. **Invalidate Cache**: Remove affected keys from Redis
   - Individual product key (for update/delete)
   - List cache keys (for create/update/delete)

## Project Structure

```
redis-caching-demo/
├── main.go                    # Application entry point
├── go.mod                     # Go module definition
├── docker-compose.yml         # Redis container setup
├── demo.sh                    # Interactive demo script
├── README.md                  # This file
├── domain/
│   └── product/
│       ├── entity.go          # Product domain entity
│       └── repository.go      # GORM repository
└── modules/
    ├── api/
    │   ├── module.go          # Fiber HTTP server module
    │   └── handlers.go        # HTTP request handlers
    ├── cache/
    │   ├── module.go          # Cache mono module
    │   └── cache.go           # Redis cache implementation
    └── product/
        ├── module.go          # Product mono module
        └── service.go         # Product service with caching
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
| GET | `/api/v1/cache/stats` | Get cache statistics |
| POST | `/api/v1/cache/stats/reset` | Reset cache statistics |

### Response Format

All product responses include caching metadata:

```json
{
  "product": { ... },
  "from_cache": true,
  "duration_ms": 1
}
```

### Cache Statistics

```json
{
  "cache_stats": {
    "hits": 150,
    "misses": 30,
    "sets": 30,
    "deletes": 5,
    "errors": 0,
    "hit_rate": 0.833
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose (for Redis)
- curl (for testing)

## Quick Start

1. **Start Redis**:
   ```bash
   docker-compose up -d
   ```

2. **Run the application**:
   ```bash
   go run main.go
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

### View Cache Statistics
```bash
curl http://localhost:3000/api/v1/cache/stats
```

### Reset Cache Statistics
```bash
curl -X POST http://localhost:3000/api/v1/cache/stats/reset
```

## Key Implementation Details

### Cache Key Strategy

- Individual products: `product:id:<id>` (e.g., `product:id:1`)
- Product lists: `product:list:<offset>:<limit>` (e.g., `product:list:0:20`)

### Cache Invalidation Strategy

- **Create**: Invalidates all list caches (pattern: `product:list:*`)
- **Update**: Invalidates specific product key + all list caches
- **Delete**: Invalidates specific product key + all list caches

### Thread Safety

- Cache statistics use atomic operations for thread-safe updates
- Redis operations are inherently thread-safe

### Error Handling

- Cache errors are logged but don't fail requests (graceful degradation)
- Database errors are propagated to the client
- Invalid requests return appropriate HTTP status codes

## Testing the Cache

1. **Create a product** and note the ID
2. **Get the product twice** - first request should show `"from_cache": false`, second shows `"from_cache": true`
3. **Check cache stats** - should show 1 miss, 1 hit
4. **Update the product** - cache is invalidated
5. **Get the product again** - `"from_cache": false` (cache was invalidated)
6. **Check cache stats** - should show 2 misses, 1 hit

## Mono Framework Features Used

- **Module Lifecycle**: Init, Start, Stop methods for proper startup/shutdown
- **Dependency Injection**: Modules wired together via setter methods
- **Service Container**: Modules registered with the mono app
- **Graceful Shutdown**: Clean shutdown with resource cleanup

## License

MIT
