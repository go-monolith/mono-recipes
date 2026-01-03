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
  - _Requirements: Success Criterion 1 (20 recipes), Success Criterion 3 (self-contained), Success Criterion 4 (working examples), Success Criterion 5 (explains "why")_

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
  - _Requirements: Success Criterion 1, Success Criterion 3, Success Criterion 4, Success Criterion 5_

- [x] 3. Create File Upload recipe with Gin + fs-jetstream plugin
  - Create new recipe directory: `projects/file-upload-demo/`
  - Implement HTTP server using Gin framework (alternative to Fiber)
  - Create file upload endpoint (`POST /api/v1/files`)
  - Implement fs-jetstream plugin for file storage backend (NATS JetStream object store)
  - Create file metadata module with `ServiceProviderModule`
  - Implement file download endpoint (`GET /api/v1/files/:id`)
  - Add file listing endpoint (`GET /api/v1/files`)
  - Implement file deletion endpoint (`DELETE /api/v1/files/:id`)
  - Include `docker-compose.yml` for NATS JetStream container
  - Include comprehensive `README.md` explaining:
    - Why use JetStream object store for file storage
    - Benefits of distributed file storage vs local filesystem
    - Scalability considerations and use cases
  - Create executable `demo.sh` demonstrating:
    - File upload (single and multiple files)
    - File download
    - File listing and deletion
  - Add unit tests for file service
  - Success Criteria: Files can be uploaded/downloaded via `demo.sh`, stored in JetStream
  - _Requirements: Success Criterion 1, Success Criterion 3, Success Criterion 4, Success Criterion 5_

- [x] 4. Create URL Shortener recipe with Fiber + kv-jetstream plugin
  - Create new recipe directory: `projects/url-shortener-demo/`
  - Implement HTTP server using Fiber framework
  - Create URL shortening endpoint (`POST /api/v1/shorten`)
  - Implement kv-jetstream plugin for key-value storage (NATS JetStream KV store)
  - Generate short codes (base62 encoding or nanoid)
  - Create redirect endpoint (`GET /:shortCode`)
  - Add URL statistics endpoint (`GET /api/v1/stats/:shortCode`)
  - Implement `EventEmitterModule` to publish URL created/accessed events
  - Create analytics consumer module using `EventConsumerModule`
  - Include `docker-compose.yml` for NATS JetStream container
  - Include comprehensive `README.md` explaining:
    - Why use JetStream KV for URL mappings
    - Event-driven analytics pattern
    - Scalability and TTL considerations
  - Create executable `demo.sh` demonstrating:
    - URL shortening
    - Redirect verification
    - Statistics retrieval
  - Add unit tests for shortener service
  - Success Criteria: URLs shortened and redirected via `demo.sh`, events published to analytics
  - _Requirements: Success Criterion 1, Success Criterion 3, Success Criterion 4, Success Criterion 5_

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
  - _Requirements: Success Criterion 1, Success Criterion 3, Success Criterion 4, Success Criterion 5_

<!-- New tasks added on 2026-01-03 - Milestone 4: API Patterns and Caching -->

- [x] 6. Create Rate Limiting recipe with Fiber + Redis
  - Create new recipe directory: `projects/rate-limiting-demo/`
  - Implement HTTP server using Fiber framework
  - Integrate Redis for distributed rate limit counters (using go-redis)
  - Implement sliding window rate limiting algorithm
  - Create rate limiting middleware with configurable limits:
    - Per-IP rate limiting
    - Per-user/API-key rate limiting (authenticated routes)
    - Global rate limiting fallback
  - Add endpoints demonstrating rate limiting:
    - `GET /api/v1/public` - Rate limited by IP (100 req/min)
    - `GET /api/v1/premium` - Higher limits for authenticated users (1000 req/min)
  - Return proper 429 Too Many Requests responses with Retry-After header
  - Include `docker-compose.yml` for Redis container
  - Include comprehensive `README.md` explaining:
    - Why use distributed rate limiting (vs in-memory)
    - Sliding window vs fixed window algorithms
    - Trade-offs and production considerations
  - Create executable `demo.sh` demonstrating:
    - Normal requests within rate limit
    - Exceeding rate limit and receiving 429 responses
    - Retry-After header behavior
  - Add unit tests for rate limiting middleware
  - Success Criteria: Rate limits enforced correctly, 429 returned when exceeded
  - _Requirements: Success Criterion 1, Success Criterion 3, Success Criterion 4, Success Criterion 5_

- [-] 7. Create Redis Caching recipe with Fiber + GORM
  - Create new recipe directory: `projects/redis-caching-demo/`
  - Implement HTTP server using Fiber framework
  - Create Product entity with GORM and SQLite database
  - Integrate Redis for caching layer (using go-redis)
  - Implement cache-aside pattern for database queries:
    - Check cache first, return if hit
    - Query database if cache miss, then populate cache
    - Automatic cache invalidation on updates/deletes
  - Create REST endpoints with caching:
    - `GET /api/v1/products` - List with cache
    - `GET /api/v1/products/:id` - Get with cache
    - `POST /api/v1/products` - Create (no cache)
    - `PUT /api/v1/products/:id` - Update with cache invalidation
    - `DELETE /api/v1/products/:id` - Delete with cache invalidation
  - Add cache statistics endpoint (`GET /api/v1/cache/stats`)
  - Include `docker-compose.yml` for Redis container
  - Include comprehensive `README.md` explaining:
    - Why use Redis for caching (performance, distributed)
    - Cache-aside pattern explained with diagrams
    - Cache invalidation strategies and TTL considerations
    - When to cache vs when not to cache
  - Create executable `demo.sh` demonstrating:
    - Cache miss → database query → cache population
    - Cache hit on subsequent requests (faster response)
    - Cache invalidation on update/delete
  - Add unit tests for caching layer
  - Success Criteria: Cache hits/misses work correctly, invalidation on mutations
  - _Requirements: Success Criterion 1, Success Criterion 3, Success Criterion 4, Success Criterion 5_

- [ ] 8. Create Background Jobs recipe with NATS JetStream Workers
  - Create new recipe directory: `projects/background-jobs-demo/`
  - Implement job queue using NATS JetStream as message broker
  - Create worker pool module with `ServiceProviderModule`:
    - Configurable number of concurrent workers
    - Job acknowledgment after successful processing
    - Automatic retry with exponential backoff on failure
  - Implement job types:
    - Email sending simulation (async task)
    - Image processing simulation (long-running task)
    - Report generation simulation (batch task)
  - Create REST API for job management:
    - `POST /api/v1/jobs` - Enqueue new job
    - `GET /api/v1/jobs/:id` - Get job status
    - `GET /api/v1/jobs` - List jobs with pagination
  - Implement dead-letter queue for failed jobs (max retries exceeded)
  - Add job progress tracking via EventBus events:
    - `JobStartedEvent`, `JobProgressEvent`, `JobCompletedEvent`, `JobFailedEvent`
  - Include `docker-compose.yml` for NATS JetStream container
  - Include comprehensive `README.md` explaining:
    - Why use message queues for background processing
    - Worker pool pattern and concurrency considerations
    - Retry strategies and dead-letter queues
    - Idempotency and exactly-once processing
  - Create executable `demo.py` demonstrating:
    - Enqueueing multiple jobs of different types
    - Watching job progress in real-time
    - Simulating job failures and retries
  - Add unit tests for job service and worker
  - Success Criteria: Jobs processed asynchronously, retries work, dead-letter queue captures failures
  - _Requirements: Success Criterion 1, Success Criterion 3, Success Criterion 4, Success Criterion 5_

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
  - _Requirements: Success Criterion 1, Success Criterion 3, Success Criterion 4, Success Criterion 5_

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
  - _Requirements: Success Criterion 1, Success Criterion 3, Success Criterion 4, Success Criterion 5_
