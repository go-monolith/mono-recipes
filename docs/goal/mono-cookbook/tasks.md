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

## Task List

- [ ] 1. Create GORM + SQLite recipe demonstrating ORM-based database integration
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

- [ ] 2. Create JWT Authentication recipe with Echo + GORM + SQLite
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

- [ ] 3. Create File Upload recipe with Gin + fs-jetstream plugin
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

- [ ] 4. Create URL Shortener recipe with Fiber + kv-jetstream plugin
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
  - _Requirements: Success Criterion 1, Success Criterion 3, Success Criterion 4, Success Criterion 5_
