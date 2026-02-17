# Prisma + sqlc PostgreSQL Demo

A mono framework recipe demonstrating how to combine **Prisma for infrastructure** (local development, migrations) with **sqlc for database access** (type-safe Go code generation).

## Why Prisma + sqlc?

This hybrid approach gives you the **best of both worlds**:

| Tool | Responsibility | Benefits |
|------|---------------|----------|
| **Prisma** | Local development, migrations | PGlite-powered local dev (no Docker!), declarative schema, automatic migrations |
| **sqlc** | Database access | Type-safe Go code, compile-time SQL validation, native Go ecosystem integration |

### Why NOT use Prisma as an ORM?

Prisma's ORM generates TypeScript/JavaScript client code, not Go. While there are community Go clients for Prisma, the Go ecosystem strongly favors sqlc for type-safe database access because:

1. **Native Go code** - sqlc generates idiomatic Go structs and functions
2. **Compile-time validation** - SQL syntax errors are caught at code generation time
3. **No runtime overhead** - Direct SQL queries without ORM abstraction layers
4. **Full SQL power** - Write any SQL query, no ORM limitations

## Architecture

This recipe implements a 3-layer architecture following SOLID principles:

```
┌─────────────────────────────────────────────────────────────┐
│                    ArticleModule                             │
│  (Framework integration: lifecycle, service registration)    │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                    ArticleService                            │
│  (Business logic: validation, pagination, type conversion)   │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                  ArticleRepository                           │
│  (Data access: wraps sqlc-generated queries)                 │
└─────────────────────────────────────────────────────────────┘
```

## Domain Entity

**Article** with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Auto-generated primary key |
| title | varchar(255) | Article title (required) |
| content | text | Article body (required) |
| slug | varchar(255) | URL-friendly unique identifier |
| published | boolean | Draft (false) or published (true) |
| created_at | timestamptz | Creation timestamp |
| updated_at | timestamptz | Last update timestamp |

## Services

All services are exposed via NATS request-reply pattern:

| Service | Description |
|---------|-------------|
| `services.article.create` | Create a new article |
| `services.article.get` | Get article by ID or slug |
| `services.article.list` | List articles with pagination and published filter |
| `services.article.update` | Update article fields |
| `services.article.delete` | Delete article by ID |
| `services.article.publish` | Publish a draft article |

## Prerequisites

- **Node.js v20+** - Required for Prisma CLI (PGlite feature)
- **Go 1.25+** - For building the application
- **sqlc CLI** - For generating type-safe Go code
- **nats CLI** - For interacting with services (demo)

Install sqlc:
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

Install nats CLI:
```bash
# macOS
brew install nats-io/nats-tools/nats

# Linux
curl -sf https://binaries.nats.dev/nats-io/natscli/nats@latest | sh
```

## Quick Start

```bash
# Install dependencies
npm install

# Start local Prisma Postgres (PGlite) and list instances
make prisma-dev-detach prisma-dev-ls
```

The `prisma-dev-ls` command outputs server information in JSON format:
```json
{
  "name": "prisma-postgres-demo",
  "port": 51213,
  "databasePort": 51214,
  "exports": {
    "ppg": {
      "url": "prisma+postgres://localhost:51213/?api_key=eyJ..."
    }
  }
}
```

Copy the `exports.ppg.url` value and set it as the `PRISMA_DB_URL` environment variable in `.env` or export it in your shell:
```bash
# Set the Prisma Postgres URL (copy from ppg.url above)
export PRISMA_DB_URL="prisma+postgres://localhost:51213/?api_key=eyJ..."

# Apply migrations
make prisma-migrate

# Generate sqlc code
make sqlc-generate

# Build and run application
make run
```

To run the complete demo (require DB to be migrated & running), execute:
```bash
./demo.sh
```

## Development Workflow

The recommended development workflow combines Prisma's migration tooling with sqlc's code generation. Prisma handles schema migrations while sqlc reads from a `full_schema.sql` file that mirrors the database schema with proper DEFAULT values for Go compatibility.

```
┌─────────────────────────────────────────────────────────────┐
│  1. Edit prisma/schema.prisma                               │
│     - Define models with Prisma schema syntax               │
│     - Use @db directives for PostgreSQL types               │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│  2. Run: npx prisma migrate dev                             │
│     - Generates SQL migration in prisma/migrations/         │
│     - Applies migrations to local PGlite database           │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│  3. Update modules/article/db/prisma/full_schema.sql        │
│     - Keep in sync with Prisma migrations                   │
│     - Include DEFAULT values for UUID generation            │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│  4. Write queries in modules/article/db/query.sql           │
│     - Use sqlc annotations (-- name: QueryName :type)       │
│     - Full SQL power with type safety                       │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│  5. Run: sqlc generate                                      │
│     - Reads schema from full_schema.sql                     │
│     - Generates type-safe Go code in db/generated/          │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│  6. Run: go run .                                           │
│     - Start the mono application                            │
│     - Services available via NATS request-reply             │
└─────────────────────────────────────────────────────────────┘
```

The key configuration enabling this workflow is in `prisma.config.ts`:
```typescript
export default defineConfig({
  schema: "prisma/schema.prisma",
  migrations: {
    path: "prisma/migrations",
  },
  datasource: {
    url: env("PRISMA_DB_URL"),  // Set from prisma-dev-ls ppg.url
  },
});
```

And `sqlc.yaml` reads from a consolidated schema file (kept in sync with Prisma migrations):
```yaml
schema: "prisma/full_schema.sql"
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make prisma-dev` | Start local Prisma Postgres (interactive) |
| `make prisma-dev-detach` | Start local Prisma Postgres (background) |
| `make prisma-dev-stop` | Stop local Prisma Postgres |
| `make prisma-dev-ls` | List local Prisma Postgres instances |
| `make prisma-migrate` | Run database migrations |
| `make sqlc-generate` | Generate type-safe Go code |
| `make generate` | Run migrations + sqlc generate |
| `make build` | Build the application |
| `make run` | Build and run |
| `make test` | Run all tests |
| `make clean` | Remove build artifacts |

## Project Structure

```
prisma-postgres-demo/
├── main.go                           # Application entry point
├── go.mod                            # Go dependencies
├── package.json                      # Prisma CLI dependency (includes dotenv)
├── prisma.config.ts                  # Prisma configuration (PRISMA_DB_URL)
├── Makefile                          # Build and workflow targets
├── demo.sh                           # NATS CLI demo script
├── README.md                         # This file
├── prisma/
│   ├── schema.prisma                 # Prisma schema definition
│   └── migrations/                   # Prisma-generated SQL migrations
│       └── 20260115.../
│           └── migration.sql
├── modules/
│   └── article/
│       ├── module.go                 # ArticleModule (ServiceProviderModule)
│       ├── service.go                # ArticleService business logic
│       ├── repository.go             # PostgresRepository using sqlc
│       ├── types.go                  # Request/Response DTOs
│       ├── service_test.go           # Unit tests with mock repository
│       ├── repository_test.go        # Integration tests
│       └── db/
│           ├── sqlc.yaml             # sqlc config
│           ├── query.sql             # SQL queries with sqlc annotations
│           ├── prisma/
│           │   └── full_schema.sql   # Consolidated schema for sqlc
│           └── generated/            # sqlc-generated Go code
└── bin/                              # Compiled binary
```

## Example Usage

### Create an Article

```bash
nats request services.article.create '{
  "title": "Getting Started with Go",
  "content": "Go is a statically typed, compiled language...",
  "slug": "getting-started-with-go",
  "published": false
}'
```

### Get Article by Slug

```bash
nats request services.article.get '{"slug": "getting-started-with-go"}'
```

### List Published Articles

```bash
nats request services.article.list '{"published": true, "limit": 10}'
```

### Publish a Draft

```bash
nats request services.article.publish '{"id": "article-uuid-here"}'
```

## Trade-offs

### Prisma Migrations vs Raw SQL Migrations

| Aspect | Prisma Migrations | Raw SQL (golang-migrate) |
|--------|------------------|--------------------------|
| Schema Definition | Declarative DSL | Raw SQL |
| Migration Generation | Automatic | Manual |
| Local Development | PGlite (no Docker) | Requires PostgreSQL |
| Production | Generates SQL | Direct SQL control |
| Learning Curve | Prisma-specific | Standard SQL |

### sqlc vs GORM (ORM)

| Aspect | sqlc | GORM |
|--------|------|------|
| Code Generation | At build time | At runtime (reflection) |
| SQL Validation | Compile-time | Runtime |
| Query Flexibility | Any SQL | Limited by ORM |
| Performance | Direct SQL | ORM overhead |
| Learning Curve | SQL knowledge | ORM API |

## When to Use This Pattern

**Choose Prisma + sqlc when:**
- You want easy local development (no Docker setup)
- You prefer declarative schema definitions
- You want compile-time SQL validation
- You're building a Go application
- You want full SQL power without ORM limitations

**Consider alternatives when:**
- You need a pure Go solution (use raw SQL + golang-migrate)
- Your team prefers ORM abstractions (use GORM)
- You don't want Node.js dependency (use Docker + raw SQL)

## Related Recipes

- [sqlc-postgres-demo](../sqlc-postgres-demo/) - Pure sqlc with Docker PostgreSQL
- [gorm-sqlite-demo](../gorm-sqlite-demo/) - GORM ORM with SQLite

## License

MIT
