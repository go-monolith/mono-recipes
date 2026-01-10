# Mono Dev Plugin

A Claude Code plugin providing best practices for developing modular monolith applications with the [go-monolith/mono](https://github.com/go-monolith/mono) framework.

## Features

- **Framework guidance**: Core concepts, patterns, and best practices for Mono applications
- **Module development**: Creating modules, services, and event-driven communication
- **Service patterns**: RequestReply, QueueGroup, Channel, and Stream services
- **Plugin system**: Using kv-jetstream, fs-jetstream, and creating custom plugins
- **Middleware**: Built-in and custom middleware patterns
- **HTTP integration**: Fiber and Gin server modules
- **Database integration**: GORM and sqlc patterns
- **Polyglot clients**: Python and Node.js client integration

## Installation

### Local Installation

Add to your project's `.claude/settings.json`:

```json
{
  "plugins": [
    ".claude/plugins/mono-dev"
  ]
}
```

### Global Installation

Add to your global `~/.claude/settings.json`:

```json
{
  "plugins": [
    "/path/to/mono-dev"
  ]
}
```

## Usage

The skill activates automatically when you ask about:

- Creating Mono applications or modules
- Mono plugins (kv-jetstream, fs-jetstream)
- Service patterns (RequestReply, QueueGroup, etc.)
- Event-driven communication
- Middleware development
- Python or Node.js clients for Mono apps

### Example Prompts

- "Create a new Mono module with RequestReply service"
- "Add kv-jetstream plugin for session storage"
- "Create a Python client to connect to my Mono app"
- "Add custom middleware for rate limiting"
- "Set up QueueGroup workers for background jobs"

## Contents

### Skill

The main skill provides comprehensive guidance on:

- MonoApplication setup and configuration
- Module lifecycle and interfaces
- Service communication patterns
- Event emitter and consumer patterns
- Plugin and middleware systems

### References

Detailed documentation in `skills/mono-dev/references/`:

- `modules.md` - Module interface patterns and lifecycle
- `services.md` - Service registration and consumption
- `events.md` - Event emitter and consumer patterns
- `plugins.md` - Plugin creation and built-in plugins
- `middleware.md` - Middleware hooks and built-in middleware
- `http-servers.md` - HTTP server integration (Fiber, Gin)
- `databases.md` - Database integration (GORM, sqlc)
- `graceful-shutdown.md` - Graceful shutdown patterns
- `polyglot-clients.md` - Python and Node.js client patterns

### Examples

Working code examples in `skills/mono-dev/examples/`:

- `basic-module.go` - Minimal module implementation
- `service-provider.go` - Service provider with RequestReply
- `queue-group-service.go` - QueueGroupService with QGHP pattern
- `multi-module.go` - Multi-module with dependencies
- `event-emitter.go` - Event emitter with consumer
- `plugin-module.go` - Custom plugin implementation
- `kv-jetstream-usecase.go` - Sessions, counters, distributed locks
- `fs-jetstream-usecase.go` - Document storage, uploads, media
- `middleware-module.go` - Custom metrics middleware
- `middleware-usecases.go` - Using requestid, accesslog, audit
- `http-fiber-module.go` - Fiber HTTP server integration
- `http-gin-module.go` - Gin HTTP server integration
- `gorm-sqlite-module.go` - GORM with SQLite integration
- `sqlc-postgres-module.go` - sqlc with PostgreSQL integration

## Resources

- [Mono Framework Documentation](https://gelmium.gitbook.io/monolith-framework)
- [Mono GitHub Repository](https://github.com/go-monolith/mono)
- [Mono Recipes (Example Projects)](https://github.com/go-monolith/mono-recipes)

## License

MIT
