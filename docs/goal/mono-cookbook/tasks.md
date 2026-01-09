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

5. Milestone 5 - Advanced Patterns
   - Tasks: 11-15
   - Notes: CRUD generation, resilience patterns, pagination, audit trails

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

- [x] 3. Create File Upload recipe with Gin + builtin `fs-jetstream` plugin
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

- [x] 4. Create URL Shortener recipe with Fiber + builtin `kv-jetstream` plugin
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

- [x] 5. Create WebSocket Chat recipe with Fiber + EventBus pubsub
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

- [x] 6. Create Rate Limiting Middleware recipe with mono framework + Redis
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

- [x] 9. Create sqlc + PostgreSQL CRUD recipe with RequestReplyService
  - Create new recipe directory: `projects/sqlc-postgres-demo/`
  - Implement type-safe SQL with sqlc code generation (not ORM-based)
  - Use PostgreSQL database via `docker-compose.yml` for easy setup
  - Create domain entity (e.g., `User` with id, name, email, created_at, updated_at)
  - Implement `ServiceProviderModule` exposing `RequestReplyService` for CRUD operations:
    - `user.create` - Create a new user (returns created user with ID)
    - `user.get` - Get user by ID
    - `user.list` - List all users with pagination support
    - `user.update` - Update user by ID
    - `user.delete` - Delete user by ID
  - No REST API endpoints - module only exposes services via mono's ServiceContainer
  - Include `docker-compose.yml` with:
    - PostgreSQL container with health check
    - Volume for data persistence
    - Environment variables for connection config
  - Include `schema.sql` for table creation and `query.sql` for sqlc queries
  - Include `sqlc.yaml` configuration file
  - Run `sqlc generate` to produce type-safe Go code
  - Include comprehensive `README.md` explaining:
    - Why use sqlc vs GORM (type-safety, performance, SQL familiarity)
    - Trade-offs: compile-time SQL validation vs runtime ORM flexibility
    - When to choose sqlc vs ORM approaches
    - PostgreSQL vs SQLite considerations for production
    - How RequestReplyService works in mono framework
  - Create executable `demo.sh` script demonstrating:
    - Start PostgreSQL via `docker compose up -d`
    - Wait for database ready using health check
    - Send request messages via `nats request` for CRUD operations
    - JSON request/response payloads for each operation
    - Verify data directly via `psql` commands showing table contents
    - Examples: create user, list users, get by ID, update, delete, final psql verification
  - Add unit tests for repository layer with test database
  - Use `code-simplifier` subagent to simplify and refine the code for clarity and maintainability
  - Re-run all unit tests after code simplification to ensure they still pass
  - Success Criteria: `docker compose up -d && go run .` starts app, `demo.sh` performs full CRUD via `nats` CLI, `psql` verification shows correct data, all unit tests pass after code simplification
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

- [x] 10. Create Python NATS Client integration recipe with multi-service Mono application
  - Create new recipe directory: `projects/python-nats-client-demo/`
  - Demonstrate interoperability between Python clients and Go-based Mono applications via NATS
  - Implement Mono application with 3 distinct service patterns:
    - **RequestReplyService** (`math.calculate`): Synchronous math calculation operations
      - Operations: add, subtract, multiply, divide, power, sqrt
      - Request: `{"operation": "add", "a": 10, "b": 5}`
      - Response: `{"result": 15, "operation": "add"}`
    - **QueueGroupService** (`email.send`): Simulate sending email to users (fire-and-forget)
      - Queue group ensures only one worker processes each email
      - Request: `{"to": "user@example.com", "subject": "Welcome", "body": "Hello!"}`
      - Simulates email sending with random delay and success/failure logging
    - **StreamConsumerService** (`payment.process`): Simulate subscription payment processing
      - Durable stream consumer for reliable payment processing
      - Message: `{"user_id": "user123", "subscription_id": "sub456", "amount": 9.99}`
      - Simulates payment gateway call with idempotency check
      - Acknowledges message only after successful processing
  - Create Python client using `nats-py` library (https://github.com/nats-io/nats.py):
    - `client.py` - Main client module with async NATS connection
    - `math_client.py` - RequestReply client for math operations
    - `email_client.py` - QueueGroup publisher for email jobs
    - `payment_client.py` - JetStream publisher for payment events
  - Include `requirements.txt` with `nats-py` and other dependencies
  - Include comprehensive `README.md` explaining:
    - Why use Python NATS client for polyglot microservices
    - RequestReplyService vs QueueGroupService vs StreamConsumerService patterns
    - When to use each service pattern (sync vs async, at-most-once vs at-least-once)
    - JetStream durability and acknowledgment semantics
    - Trade-offs of language interoperability via messaging
  - Create executable `demo.py` demonstrating:
    - Connect Python client to Mono application's embedded NATS
    - Call math operations via RequestReply and display results
    - Send multiple email jobs via QueueGroup and observe load balancing
    - Publish payment events to JetStream stream and verify processing
    - Color-coded output showing request/response flow
    - Command-line arguments for different demo scenarios
  - Add unit tests for Go service handlers
  - Add Python tests for client modules using pytest
  - Use `code-simplifier` subagent to simplify and refine the code for clarity and maintainability
  - Re-run all unit tests (Go and Python) after code simplification to ensure they still pass
  - Success Criteria: `go run .` starts Mono app, `python demo.py` demonstrates all 3 service patterns, messages flow correctly between Python and Go
  - _Dependencies: 8, 9_
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_

- [ ] 11. Create Node.js NATS Client integration recipe with fs-jetstream file storage
  - Create new recipe directory: `projects/node-nats-client-demo/`
  - Demonstrate interoperability between Node.js clients and Go-based Mono applications via NATS
  - Implement Mono application using builtin `fs-jetstream` plugin with 2 services:
    - **RequestReplyService** (`file.save`): Save JSON file to "user-setting" bucket
      - Request: `{"filename": "user123.json", "content": {"theme": "dark", "lang": "en"}}`
      - Response: `{"success": true, "filename": "user123.json", "size": 42}`
      - Uses `fs-jetstream` plugin's `FileStoragePort` to store file in JetStream object store
    - **QueueGroupService** (`file.archive`): Archive existing JSON file (zip and delete original)
      - Request: `{"filename": "user123.json"}`
      - Reads JSON file from "user-setting" bucket
      - Compresses content to ZIP format (e.g., `user123.zip`)
      - Saves ZIP file to same bucket
      - Deletes original JSON file
      - Fire-and-forget with logging for success/failure
  - Mono application uses `UsePluginModule` interface to receive `fs-jetstream` plugin
  - Create Node.js client using `nats.js` library (https://github.com/nats-io/nats.js):
    - `client.js` - Main NATS connection with async/await
    - `file-service.js` - RequestReply client for saving JSON files
    - `archive-service.js` - QueueGroup publisher for archive jobs
    - `watcher.js` - JetStream object store watcher for bucket changes
  - Include `package.json` with `nats` and other dependencies
  - Implement **watch capability** in Node.js client:
    - Subscribe to changes in "user-setting" bucket using JetStream object store watch
    - Detect file creation events (both JSON and ZIP files)
    - Print changes to stdout with timestamps and file metadata
    - Support filtering by file extension (`.json`, `.zip`)
  - Include comprehensive `README.md` explaining:
    - Why use Node.js NATS client for polyglot microservices
    - How `fs-jetstream` plugin provides file storage via JetStream object store
    - RequestReplyService vs QueueGroupService patterns for file operations
    - JetStream object store watch capability for real-time notifications
    - Use cases: user settings sync, config distribution, file processing pipelines
  - Create executable `demo.js` demonstrating:
    - Connect Node.js client to Mono application's embedded NATS
    - Start watcher to subscribe to bucket changes (runs in background)
    - Save multiple JSON files via RequestReply and observe watcher output
    - Archive files via QueueGroup and observe ZIP creation + JSON deletion in watcher
    - Color-coded console output showing request/response and watch events
    - Graceful shutdown with Ctrl+C
  - Add unit tests for Go service handlers
  - Add Node.js tests using Jest or Vitest
  - Use `code-simplifier` subagent to simplify and refine the code for clarity and maintainability
  - Re-run all unit tests (Go and Node.js) after code simplification to ensure they still pass
  - Success Criteria: `go run .` starts Mono app, `node demo.js` saves files, archives them, and watcher prints all bucket changes in real-time
  - _Dependencies: 3, 10_
  - _Requirements: self-contained, working example with a demo script, have README.md explains "why"_
