# 🍳 Recipes for [Mono](https://github.com/go-monolith/mono)

**Welcome to the official Mono cookbook**!

Here you can find the most **delicious** recipes to cook delicious meals using our Monolith Framework.

*All examples presented here are built with `github.com/go-monolith/mono` framework **v0.1.1***

> **Note:** All recipes have been updated to mono framework v0.1.1 and follow the **module boundaries** (Ports and Adapters) path taught by the `mono-dev` skill: each module that provides services or emits events publishes a `contract` package with typed service contracts and the events it emits, consumers own their ports and adapters and declare what they call with `ConsumeServices()`, every module has a `monotest.CheckModule` conformance test (or a real-app boundary test for JetStream-backed modules), and every application runs with `mono.WithStrictModuleBoundaries()`. See the [Upgrading guide](https://github.com/go-monolith/mono/blob/main/docs/official/guides/upgrading.md) for the framework changes since v0.0.8.

## 🌽 Table of contents

### Core Patterns & Architecture
- [Graceful Shutdown Demo](./projects/graceful-shutdown-demo/README.md) - Demonstrates graceful shutdown of a HTTP server module (using **Fiber**) and a background worker module.
- [Hexagonal Architecture](./projects/hexagonal-architecture/README.md) - Ports and Adapters end to end: per-module `contract` packages, consumer-owned ports and adapters, `ConsumeServices()`, events owned by the emitter's contract, and a `monotest.CheckModule` test per module.
- [Background Jobs Demo](./projects/background-jobs-demo/README.md) - Background job processing system using **QueueGroupService** pattern with load-balanced queue groups.
- [Cron Scheduler Demo](./projects/cron-scheduler-demo/README.md) - Server-side scheduled jobs using the **CronService** pattern (JetStream message scheduler), demonstrating interval, cron-expression, and named-alias schedules with cluster-wide single firing.

### Data & Storage
- [GORM SQLite Demo](./projects/gorm-sqlite-demo/README.md) - **GORM** ORM with SQLite integration demonstrating the **ServiceProviderModule** pattern for CRUD services.
- [sqlc PostgreSQL Demo](./projects/sqlc-postgres-demo/README.md) - **sqlc** for type-safe SQL code generation with **PostgreSQL**, demonstrating compile-time query validation and the **ServiceProviderModule** pattern.
- [Prisma + sqlc PostgreSQL Demo](./projects/prisma-postgres-demo/README.md) - Hybrid approach using **Prisma** for local development and migrations (PGlite-powered) with **sqlc** for type-safe database access.
- [Prisma + sqlc Multiple Schemas Demo](./projects/prisma-postgres-multiple-schemas-demo/README.md) - Two modules (article, blog), each owning its own **PostgreSQL** schema, managed by **Prisma** multi-schema migrations and accessed via **sqlc**.
- [File Upload Demo](./projects/file-upload-demo/README.md) - File upload/download functionality using the built-in **fs-jetstream** plugin with **Gin** HTTP framework.
- [Redis Caching Demo](./projects/redis-caching-plugin/README.md) - **Redis** caching with cache-aside pattern, automatic cache invalidation, and a custom **PluginModule**, using **Fiber** and **GORM**; the API module reaches products only through typed contracts.

### Authentication & Security
- [JWT Auth Demo](./projects/jwt-auth-demo/README.md) - Complete JWT authentication with access/refresh tokens, bcrypt password hashing, and authentication middleware using **Fiber** and **GORM**.
- [Rate Limiting Middleware](./projects/rate-limiting-middleware/README.md) - **Redis**-based sliding window rate limiting using the **MiddlewareModule** pattern to protect services.

### Polyglot & Interoperability
- [Node.js NATS Client Demo](./projects/node-nats-client-demo/README.md) - **Polyglot microservices** with Node.js clients and Go Mono services via NATS, featuring **fs-jetstream** file storage and real-time object store watching.
- [Python NATS Client Demo](./projects/python-nats-client-demo/README.md) - **Polyglot microservices** with Python clients and Go Mono services via NATS, showcasing **RequestReplyService**, **QueueGroupService**, and **StreamConsumerService** patterns.

### Real-time & Communication
- [WebSocket Chat Demo](./projects/websocket-chat-demo/README.md) - Real-time chat application with WebSocket communication, multi-room support, and **EventBus** pubsub pattern for message broadcasting.
- [URL Shortener Demo](./projects/url-shortener-demo/README.md) - URL shortening service using the built-in **kv-jetstream** plugin with event-driven analytics and TTL support.
