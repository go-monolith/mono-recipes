# Implementation Plan

Goal: [mono-cookbook](./goal.md)
Constraints: [constraints.md](./constraints.md)

**Custom Instructions**: Focus on these specific recipes:
1. File Upload (Gin + fs-jetstream plugin)
2. URL Shortener (Fiber + kv-jetstream plugin)
3. GORM + SQLite
4. JWT Authentication (Echo + GORM + SQLite)
5. WebSocket Chat (Fiber + EventBus pubsub)

**Codebase Analysis Summary**:
- Current state: 2 existing sample projects (graceful-shutdown-demo, hexagonal-architecture)
- Gap: Need 18+ more recipes to reach the 20-recipe goal
- Reference patterns: hexagonal-architecture provides excellent template with demo.sh, README.md structure
- Mono framework version: v0.0.2

## Milestones

1. Milestone 1 - Database Integration Recipe
   - Tasks: 1
   - Notes: Establish database pattern using GORM

2. Milestone 2 - Authentication Recipe
   - Tasks: 2
   - Notes: JWT authentication combining Echo, GORM, and SQLite

3. Milestone 3 - Event-Driven Architecture Recipes
   - Tasks: 3-5
   - Notes: File storage, key-value patterns, and real-time communication

4. Milestone 4 - API Patterns and Caching Recipes
   - Tasks: 6-10
   - Notes: Rate limiting, Redis caching, background jobs, validation, health checks

## Task List

- [x] 1. Create GORM + SQLite recipe demonstrating ORM-based database integration
  - Create new recipe directory: `projects/gorm-sqlite-demo/`
  - Implement mono module with GORM repository pattern for SQLite
  - Create domain entity (e.g., `Product` with CRUD operations)
  - Implement `ServiceProviderModule` exposing request-reply services for CRUD operations:
    - `product.create` - Create a new product
    - `product.get` - Get product by ID
    - `product.list` - List all products
    - `product.update` - Update product by ID
    - `product.delete` - Delete product by ID
  - No REST API endpoints - module only exposes services via mono's ServiceContainer
  - Add automatic SQLite database migration on startup
  - Include comprehensive `README.md` explaining:
    - Why use GORM with SQLite (rapid prototyping, embedded DB, zero config)
    - Trade-offs vs raw SQL and other ORMs
    - When to choose SQLite vs PostgreSQL
    - How request-reply services work in mono framework
  - Create executable `demo.sh` script using `nats` CLI to:
    - Send request messages directly to EventBus for CRUD operations
    - Demonstrate `nats request` commands for each service endpoint
    - Show JSON request/response payloads
    - Include examples: create product, list products, get by ID, update, delete
  - Add unit tests for repository layer
  - Success Criteria: `go run .` starts app, `demo.sh` performs full CRUD via `nats` CLI
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

- [x] 2. Create JWT Authentication recipe with Echo + GORM + SQLite
  - _Dependencies: 1_
  - Create new recipe directory: `projects/jwt-auth-demo/`
  - Implement user registration and login endpoints using Echo framework
  - Create `User` entity with GORM (email, hashed password, created_at)
  - Implement secure password hashing using bcrypt
  - Generate and validate JWT tokens with configurable expiry
  - Create authentication middleware for protected routes
  - Implement `ServiceProviderModule` for auth service
  - Add token refresh endpoint
  - Create protected endpoint example (`GET /api/v1/profile`)
  - Include comprehensive `README.md` explaining:
    - Why JWT for stateless authentication
    - Security considerations (token storage, expiry, refresh strategy)
    - When to use JWT vs session-based auth
  - Create executable `demo.sh` demonstrating:
    - User registration
    - Login and token retrieval
    - Accessing protected routes with/without token
    - Token refresh flow
  - Add unit tests for auth service and middleware
  - Success Criteria: Full auth flow works via `demo.sh`, invalid tokens are rejected
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

- [+] 3. Create File Upload recipe with Gin + builtin `fs-jetstream` plugin
  - Create new recipe directory: `projects/file-upload-demo/`
  - Implement HTTP server using Gin framework (alternative to Fiber)
  - Use mono framework's builtin `fs-jetstream` plugin via `UsePluginModule` interface
  - No external NATS required - uses mono framework's embedded NATS with JetStream
  - Create file upload endpoint (`POST /api/v1/files`)
  - Create file metadata module with `ServiceProviderModule`
  - Implement file download endpoint (`GET /api/v1/files/:id`)
  - Add file listing endpoint (`GET /api/v1/files`)
  - Implement file deletion endpoint (`DELETE /api/v1/files/:id`)
  - Include comprehensive `README.md` explaining:
    - Why use `fs-jetstream` plugin for file storage
    - How `UsePluginModule` interface provides plugin instances
    - Benefits of JetStream object store vs local filesystem
    - Scalability considerations and use cases
  - Create executable `demo.sh` demonstrating:
    - File upload (single and multiple files)
    - File download
    - File listing and deletion
  - Add unit tests for file service
  - Success Criteria: Files can be uploaded/downloaded via `demo.sh`, stored in embedded JetStream
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

- [ ] 4. Create URL Shortener recipe with Fiber + builtin `kv-jetstream` plugin
  - Create new recipe directory: `projects/url-shortener-demo/`
  - Implement HTTP server using Fiber framework
  - Use mono framework's builtin `kv-jetstream` plugin via `UsePluginModule` interface
  - No external NATS required - uses mono framework's embedded NATS with JetStream
  - Create URL shortening endpoint (`POST /api/v1/shorten`)
  - Generate short codes (base62 encoding or nanoid)
  - Create redirect endpoint (`GET /:shortCode`)
  - Add URL statistics endpoint (`GET /api/v1/stats/:shortCode`)
  - Implement `EventEmitterModule` to publish URL created/accessed events
  - Create analytics consumer module using `EventConsumerModule`
  - Include comprehensive `README.md` explaining:
    - Why use `kv-jetstream` plugin for URL mappings
    - How `UsePluginModule` interface provides plugin instances
    - Event-driven analytics pattern with EventBus
    - Scalability and TTL considerations
  - Create executable `demo.sh` demonstrating:
    - URL shortening
    - Redirect verification
    - Statistics retrieval
  - Add unit tests for shortener service
  - Success Criteria: URLs shortened and redirected via `demo.sh`, events published to analytics
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

- [ ] 5. Create WebSocket Chat recipe with Fiber + EventBus pubsub
  - Create new recipe directory: `projects/websocket-chat-demo/`
  - Implement HTTP server using Fiber framework with WebSocket support
  - Create WebSocket endpoint (`/ws`) for real-time bidirectional communication
  - Implement chat room module with `ServiceProviderModule`
  - Use mono `EventBus` for internal pubsub between connected clients
  - Implement `EventEmitterModule` to publish chat messages as events
  - Create `EventConsumerModule` to broadcast messages to WebSocket connections
  - Support multiple chat rooms with room-based message routing
  - Create REST endpoints for room management:
    - `GET /api/v1/rooms` - List active rooms
    - `POST /api/v1/rooms` - Create new room
    - `GET /api/v1/rooms/:id/history` - Get message history
  - Include comprehensive `README.md` explaining:
    - Why use WebSockets for real-time communication
    - EventBus pubsub pattern for message broadcasting
    - Scaling considerations (sticky sessions, external pubsub for multi-instance)
  - Create executable `demo.py` Python script (more powerful than bash) demonstrating:
    - Multiple concurrent WebSocket clients using asyncio and websockets library
    - Simulated chat conversation between multiple users
    - Room creation, joining, and message broadcasting
    - Color-coded output for different users
    - Command-line arguments for custom scenarios (e.g., `--users 3 --messages 10`)
  - Add `requirements.txt` with demo dependencies (websockets, asyncio, colorama)
  - Add unit tests for chat service and message handling
  - Success Criteria: `python demo.py` simulates multi-user chat, messages broadcast via EventBus
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

<!-- New tasks added on 2026-01-03 - Milestone 4: API Patterns and Caching -->

- [ ] 6. Create Rate Limiting Middleware recipe with mono framework + Redis
  - Create new recipe directory: `projects/rate-limiting-middleware/`
  - Implement `ServiceProviderModule` exposing request-reply services for API operations:
    - `api.getData` - Get data with rate limiting applied
    - `api.createOrder` - Create order with rate limiting applied
    - `api.getStatus` - Get status with rate limiting applied
  - Integrate Redis for distributed rate limit counters (using go-redis)
  - Implement sliding window rate limiting algorithm
  - Create rate limiting middleware as `mono.MiddlewareModule` applied to request-reply services:
    - Intercepts incoming service requests before reaching handlers
    - Per-client rate limiting based on request metadata (client ID, IP from headers)
    - Per-service configurable limits (e.g., `api.getData`: 100 req/min, `api.createOrder`: 50 req/min)
    - Global rate limiting fallback
  - Middleware returns error response when limit exceeded (similar to 429 semantics)
  - Include `docker-compose.yml` for Redis container
  - Include comprehensive `README.md` explaining:
    - Why use Redis for distributed rate limiting (atomic counters, TTL)
    - Sliding window vs fixed window algorithms
    - How `mono.MiddlewareModule` intercepts request-reply service calls
    - Trade-offs and production considerations
  - Create executable `demo.sh` using `nats` CLI demonstrating:
    - Normal service requests within rate limit
    - Exceeding rate limit and receiving error responses
    - Rate limit reset behavior over time
  - Add unit tests for rate limiting middleware
  - Success Criteria: Rate limits enforced correctly on request-reply services, error returned when exceeded
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

- [x] 7. Create Redis Caching Plugin recipe with Fiber + GORM + PluginModule pattern
  - Create new recipe directory: `projects/redis-caching-demo/`
  - Implement HTTP server using Fiber framework
  - Create Product entity with GORM and SQLite database
  - Implement cache as `PluginModule` using `gofiber/storage/redis/v3` with `mono.Storage` interface:
    - `CacheService` interface with JSON marshaling for Get/Set/Delete operations
    - Plugin exposes `Port()` method returning `CacheService` to consumers
    - Plugin starts first and stops last (cross-cutting concern)
  - Product module implements `UsePluginModule` to receive cache plugin via `SetPlugin()`:
    - Framework automatically injects cache plugin before `Start()`
    - No manual dependency wiring required in `main.go`
  - Implement cache-aside pattern for database queries:
    - Check cache first, return if hit (logs "Cache Hit!")
    - Query database if cache miss (logs "Cache Miss!"), then populate cache
    - Automatic cache invalidation on updates/deletes using `InvalidateAll()`
  - Create REST endpoints with caching:
    - `GET /api/v1/products` - List with cache
    - `GET /api/v1/products/:id` - Get with cache
    - `POST /api/v1/products` - Create (invalidates cache)
    - `PUT /api/v1/products/:id` - Update with cache invalidation
    - `DELETE /api/v1/products/:id` - Delete with cache invalidation
  - Cache statistics via Redis monitoring tools (no custom stats endpoint)
  - Include comprehensive `README.md` explaining:
    - Why use Redis for caching (performance, distributed)
    - Cache-aside pattern explained
    - PluginModule pattern for cross-cutting concerns
    - `UsePluginModule` interface for dependency injection
    - Cache invalidation strategies and TTL considerations
  - Create executable `demo.sh` demonstrating:
    - Cache miss → database query → cache population
    - Cache hit on subsequent requests (faster response, `from_cache: true`)
    - Cache invalidation on update/delete
  - Add unit tests for caching layer with Redis skip when unavailable
  - Success Criteria: Cache hits/misses work correctly, PluginModule pattern demonstrates framework best practices
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

- [x] 8. Create Background Jobs recipe with QueueGroupService pattern
  - Create new recipe directory: `projects/background-jobs-demo/`
  - Implement job queue using mono framework's embedded NATS with `QueueGroupService` pattern
  - Create worker module with `ServiceProviderModule` registering 3 QueueGroups on the same service:
    - `email-worker` - handles email jobs
    - `image-processing-worker` - handles image processing jobs
    - `report-generation-worker` - handles report generation jobs
  - Each handler filters for its specific `JobType` and ignores other job types (returns nil)
  - Implement job types with simulated processing:
    - Email sending simulation (async task with progress updates)
    - Image processing simulation (long-running task with operations)
    - Report generation simulation (batch task with multiple steps)
  - Create API module with `DependentModule` interface to consume worker's QueueGroupService:
    - `POST /api/v1/jobs` - Create and enqueue new job via `QueueGroupService.Send()`
    - `GET /api/v1/jobs/:id` - Get job status with progress tracking
    - `GET /api/v1/jobs` - List jobs with filtering and pagination
  - In-memory job store for status tracking (pending, processing, completed, failed)
  - Fire-and-forget semantics with simulated random failures
  - No external NATS required - uses mono framework's embedded NATS
  - Include comprehensive `README.md` explaining:
    - Why use `QueueGroupService` for background processing
    - Multiple QueueGroups per service pattern
    - Module dependencies via `DependentModule` interface
    - Trade-offs vs complex worker pool implementations
  - Create executable `demo.py` demonstrating:
    - Enqueueing multiple jobs of different types
    - Watching job progress in real-time
    - Displaying completed and failed jobs
  - Add unit tests for processor and API service layers
  - Success Criteria: Jobs processed asynchronously by type-specific workers, `demo.py` shows full flow
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

- [ ] 9. Create Request Validation recipe with Fiber + go-playground/validator
  - Create new recipe directory: `projects/request-validation-demo/`
  - Implement HTTP server using Fiber framework
  - Integrate go-playground/validator for struct validation
  - Create custom validators for common patterns:
    - Email format validation
    - Phone number format validation
    - Password strength validation (min length, special chars)
    - UUID format validation
  - Implement validation middleware for automatic request body validation
  - Create REST endpoints demonstrating validation:
    - `POST /api/v1/users` - User registration with validation
    - `POST /api/v1/orders` - Order creation with complex validation
    - `PUT /api/v1/users/:id` - Partial update with conditional validation
  - Return structured validation errors in consistent JSON format:
    - Field name, error code, human-readable message
    - Support for multiple errors per request
  - Implement locale-aware error messages (en, es examples)
  - Include comprehensive `README.md` explaining:
    - Why use structured validation (security, UX)
    - Validation at API layer vs domain layer
    - Custom validator patterns
    - Error message localization
  - Create executable `demo.sh` demonstrating:
    - Valid requests passing validation
    - Invalid requests with detailed error responses
    - Multiple validation errors in single response
  - Add unit tests for validators and middleware
  - Success Criteria: Validation errors returned with proper format, custom validators work
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

- [ ] 10. Create Health Checks & Metrics recipe with Fiber + Prometheus
  - Create new recipe directory: `projects/health-metrics-demo/`
  - Implement HTTP server using Fiber framework
  - Create comprehensive health check endpoints:
    - `GET /health/live` - Liveness probe (is app running?)
    - `GET /health/ready` - Readiness probe (can app serve traffic?)
    - `GET /health/startup` - Startup probe (is app fully initialized?)
  - Implement health check dependencies:
    - Database connectivity check (SQLite with GORM)
    - NATS connectivity check
    - Custom health check registration via module interface
  - Integrate Prometheus metrics collection:
    - HTTP request duration histogram
    - Request count by endpoint and status code
    - Active connections gauge
    - Custom business metrics (orders processed, cache hits)
  - Expose Prometheus metrics endpoint (`GET /metrics`)
  - Implement `HealthCheckableModule` interface in example modules
  - Include `docker-compose.yml` for Prometheus + Grafana containers
  - Include comprehensive `README.md` explaining:
    - Why use health checks (Kubernetes, load balancers)
    - Liveness vs Readiness vs Startup probes
    - Prometheus metrics best practices
    - Grafana dashboard setup guide
  - Create executable `demo.sh` demonstrating:
    - Health check responses in different states
    - Prometheus metrics scraping
    - Sample Grafana dashboard import
  - Add unit tests for health check handlers
  - Success Criteria: Health probes work correctly, Prometheus can scrape metrics, Grafana displays dashboard
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_
