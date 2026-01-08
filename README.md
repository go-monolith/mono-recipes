# 🍳 Recipes for [Mono](https://github.com/go-monolith/mono)

**Welcome to the official Mono cookbook**!

Here you can find the most **delicious** recipes to cook delicious meals using our Monolith Framework.

*All examples presented here are built with `github.com/go-monolith/mono` framework **v0.0.3***

> **Note:** All recipes have been updated to use mono framework v0.0.3, which includes improved error handling in RequestReplyService and QueueGroupService patterns.

## 🌽 Table of contents

### Core Patterns & Architecture
- [Graceful Shutdown Demo](./projects/graceful-shutdown-demo/README.md) - Demonstrates graceful shutdown of a HTTP server module (using **Fiber**) and a background worker module.
- [Hexagonal Architecture](./projects/hexagonal-architecture/README.md) - Example of hexagonal architecture with modules, services, and event handling.
- [Background Jobs Demo](./projects/background-jobs-demo/README.md) - Background job processing system using **QueueGroupService** pattern with load-balanced queue groups.

### Data & Storage
- [GORM SQLite Demo](./projects/gorm-sqlite-demo/README.md) - **GORM** ORM with SQLite integration demonstrating the **ServiceProviderModule** pattern for CRUD services.
- [File Upload Demo](./projects/file-upload-demo/README.md) - File upload/download functionality using the built-in **fs-jetstream** plugin with **Gin** HTTP framework.
- [Redis Caching Demo](./projects/redis-caching-plugin/README.md) - **Redis** caching with cache-aside pattern, automatic cache invalidation, and real-time statistics using **Fiber** and **GORM**.

### Authentication & Security
- [JWT Auth Demo](./projects/jwt-auth-demo/README.md) - Complete JWT authentication with access/refresh tokens, bcrypt password hashing, and authentication middleware using **Echo** and **GORM**.
- [Rate Limiting Middleware](./projects/rate-limiting-middleware/README.md) - **Redis**-based sliding window rate limiting using the **MiddlewareModule** pattern to protect services.

### Real-time & Communication
- [WebSocket Chat Demo](./projects/websocket-chat-demo/README.md) - Real-time chat application with WebSocket communication, multi-room support, and **EventBus** pubsub pattern for message broadcasting.
- [URL Shortener Demo](./projects/url-shortener-demo/README.md) - URL shortening service using the built-in **kv-jetstream** plugin with event-driven analytics and TTL support.
