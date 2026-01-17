.PHONY: all prisma-dev prisma-dev-detach prisma-dev-stop prisma-dev-ls prisma-migrate sqlc-generate generate build run clean test help

# Default target
all: help

# Prisma local development server
prisma-dev:
	npx prisma dev --name=prisma-postgres-demo

prisma-dev-detach:
	npx prisma dev --name=prisma-postgres-demo --detach

prisma-dev-start:
	npx prisma dev start prisma-postgres-demo

prisma-dev-stop:
	npx prisma dev stop prisma-postgres-demo

prisma-dev-ls:
	npx prisma dev ls --debug

# Database migrations
prisma-migrate:
	npx prisma migrate dev
	mkdir -p ./modules/article/db/prisma/
	mkdir -p ./modules/blog/db/prisma/
	npx prisma migrate diff --from-empty --to-schema-datamodel ./prisma/schema.prisma --script > ./modules/article/db/prisma/full_schema.sql
	npx prisma migrate diff --from-empty --to-schema-datamodel ./prisma/schema.prisma --script > ./modules/blog/db/prisma/full_schema.sql

prisma-migrate-reset:
	npx prisma migrate reset prisma-postgres-demo

# sqlc code generation
sqlc-generate:
	cd modules/article/db && sqlc generate
	cd modules/blog/db && sqlc generate

# Combined generation (migrations + sqlc)
generate: prisma-migrate sqlc-generate

# Build the application
build:
	go build -o bin/prisma-postgres-demo .

# Run the application
run: build
	./bin/prisma-postgres-demo

# Clean build artifacts
clean:
	rm -rf bin/

# Run tests
test:
	go test ./... -v

# Help
help:
	@echo "Prisma + sqlc PostgreSQL Recipe"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Node.js v20+ (for Prisma CLI)"
	@echo "  - sqlc CLI (go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest)"
	@echo ""
	@echo "Development Workflow:"
	@echo "  1. make prisma-dev-detach  # Start local Prisma Postgres in background"
	@echo "  2. make prisma-migrate     # Create/apply database migrations"
	@echo "  3. make sqlc-generate      # Generate type-safe Go code"
	@echo "  4. make run                # Build and run the application"
	@echo "  5. make prisma-dev-stop    # Stop local Prisma Postgres"
	@echo ""
	@echo "Targets:"
	@echo "  prisma-dev         Start local Prisma Postgres server (interactive)"
	@echo "  prisma-dev-detach  Start local Prisma Postgres server (background)"
	@echo "  prisma-dev-stop    Stop local Prisma Postgres server"
	@echo "  prisma-dev-ls      List local Prisma Postgres instances"
	@echo "  prisma-migrate     Run database migrations"
	@echo "  sqlc-generate      Generate sqlc Go code from SQL queries"
	@echo "  generate           Run migrations and sqlc generation"
	@echo "  build              Build the Go application"
	@echo "  run                Build and run the application"
	@echo "  test               Run all tests"
	@echo "  clean              Remove build artifacts"
