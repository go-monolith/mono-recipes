# Implementation Plan

Goal: [Use Latest Framework Version](./goal.md)
Constraints: [constraints.md](./constraints.md)
Current Version: v0.0.3
Target Version: v0.0.4 (released January 10, 2026)

## Framework Changes in v0.0.4

**Key Changes from v0.0.3 to v0.0.4:**
- Enhanced StreamConsumer error handling
- Improved test coverage for StreamConsumer component
- No breaking API changes identified
- All performance benchmarks passing (54 benchmarks, 12 framework targets)

**Reference:** https://github.com/go-monolith/mono/releases/tag/v0.0.4

## Task List

- [x] 1. Upgrade rate-limiting-middleware to mono v0.0.4

  - Update `projects/rate-limiting-middleware/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/rate-limiting-middleware ./cmd/api/main.go`
  - Run all existing tests: `go test ./...` (5 test files must pass)
  - Verify no deprecation warnings or compiler errors
  - Update `projects/rate-limiting-middleware/README.md` to reference v0.0.4 instead of v0.0.3
  - Commit changes after all tests pass: "chore: upgrade rate-limiting-middleware to mono v0.0.4"
  - _Success Criteria:_ All 5 tests pass, binary builds successfully, no errors/warnings
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

- [x] 2. Upgrade jwt-auth-demo to mono v0.0.4

  - Update `projects/jwt-auth-demo/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/jwt-auth-demo ./cmd/api/main.go`
  - Run all existing tests: `go test ./...` (7 tests in 4 test files must pass)
  - Check for any authentication service error handling changes (RequestReplyService pattern)
  - Verify no deprecation warnings or compiler errors
  - Update `projects/jwt-auth-demo/README.md` to reference v0.0.4
  - Commit changes after all tests pass: "chore: upgrade jwt-auth-demo to mono v0.0.4"
  - _Success Criteria:_ All 7 tests pass, authentication flows work correctly, binary builds
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

- [x] 3. Upgrade background-jobs-demo to mono v0.0.4

  - Update `projects/background-jobs-demo/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/background-jobs-demo ./cmd/api/main.go`
  - Run all existing tests: `go test ./...` (7 tests in 2 test files must pass)
  - Verify QueueGroupService error handling (improved in v0.0.3, should remain compatible in v0.0.4)
  - Verify no deprecation warnings or compiler errors
  - Update `projects/background-jobs-demo/README.md` to reference v0.0.4
  - Commit changes after all tests pass: "chore: upgrade background-jobs-demo to mono v0.0.4"
  - _Success Criteria:_ All 7 tests pass, background job patterns work correctly, binary builds
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

- [x] 4. Upgrade redis-caching-plugin to mono v0.0.4

  - Update `projects/redis-caching-plugin/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/redis-caching-plugin ./cmd/api/main.go`
  - Run all existing tests: `go test ./...` (11 tests in 2 test files must pass)
  - Verify PluginModule pattern compatibility with v0.0.4
  - Verify no deprecation warnings or compiler errors
  - Update `projects/redis-caching-plugin/README.md` to reference v0.0.4
  - Commit changes after all tests pass: "chore: upgrade redis-caching-plugin to mono v0.0.4"
  - _Success Criteria:_ All 11 tests pass, plugin module loads correctly, binary builds
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

- [x] 5. Upgrade url-shortener-demo to mono v0.0.4

  - Update `projects/url-shortener-demo/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/url-shortener-demo ./cmd/api/main.go`
  - Run all existing tests: `go test ./...` (2 tests in 2 test files must pass)
  - Verify kv-jetstream plugin (PluginModule) compatibility with v0.0.4
  - Check EventBus and EventEmitterModule patterns for any changes
  - Verify no deprecation warnings or compiler errors
  - Update `projects/url-shortener-demo/README.md` to reference v0.0.4
  - Commit changes after all tests pass: "chore: upgrade url-shortener-demo to mono v0.0.4"
  - _Success Criteria:_ All 2 tests pass, kv-jetstream plugin works, event handling intact, binary builds
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

<!-- New tasks added on 2026-01-10 - Batch 2: Next 5 projects -->

- [x] 6. Upgrade file-upload-demo to mono v0.0.4

  - Update `projects/file-upload-demo/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/file-upload-demo .`
  - Run all existing tests: `go test ./...` (2 test files: handlers_test.go, service_test.go)
  - Verify file upload service patterns work correctly with v0.0.4
  - Verify no deprecation warnings or compiler errors
  - Commit changes after all tests pass: "chore: upgrade file-upload-demo to mono v0.0.4"
  - _Success Criteria:_ All tests pass, file upload handlers work correctly, binary builds
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

- [x] 7. Upgrade gorm-sqlite-demo to mono v0.0.4

  - Update `projects/gorm-sqlite-demo/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/gorm-sqlite-demo .`
  - Run all existing tests: `go test ./...` (1 test file: repository_test.go)
  - Verify GORM integration and repository patterns work with v0.0.4
  - Verify no deprecation warnings or compiler errors
  - Commit changes after all tests pass: "chore: upgrade gorm-sqlite-demo to mono v0.0.4"
  - _Success Criteria:_ All tests pass, GORM repository works correctly, binary builds
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

- [x] 8. Upgrade graceful-shutdown-demo to mono v0.0.4

  - Update `projects/graceful-shutdown-demo/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/graceful-shutdown-demo .`
  - Note: This project has no test files - verify manually that it compiles and builds
  - Verify graceful shutdown patterns and signal handling work with v0.0.4
  - Verify no deprecation warnings or compiler errors
  - Commit changes after build succeeds: "chore: upgrade graceful-shutdown-demo to mono v0.0.4"
  - _Success Criteria:_ Binary builds successfully, no compilation errors
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

- [x] 9. Upgrade hexagonal-architecture to mono v0.0.4

  - Update `projects/hexagonal-architecture/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/hexagonal-architecture .`
  - Note: This project has no test files - verify manually that it compiles and builds
  - Verify hexagonal architecture patterns (ports/adapters) work with v0.0.4
  - Verify no deprecation warnings or compiler errors
  - Commit changes after build succeeds: "chore: upgrade hexagonal-architecture to mono v0.0.4"
  - _Success Criteria:_ Binary builds successfully, no compilation errors
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

- [x] 10. Upgrade websocket-chat-demo to mono v0.0.4

  - Update `projects/websocket-chat-demo/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/websocket-chat-demo .`
  - Run all existing tests: `go test ./...` (1 test file: types_test.go)
  - Verify WebSocket service and chat patterns work with v0.0.4
  - Verify no deprecation warnings or compiler errors
  - Commit changes after all tests pass: "chore: upgrade websocket-chat-demo to mono v0.0.4"
  - _Success Criteria:_ All tests pass, WebSocket handlers work correctly, binary builds
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

<!-- New tasks added on 2026-01-10 - Batch 3: Final 3 projects -->

- [x] 11. Upgrade node-nats-client-demo to mono v0.0.4

  - Update `projects/node-nats-client-demo/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/node-nats-client-demo .`
  - Run all existing tests: `go test ./...` (1 test file: modules/fileops/service_test.go)
  - Verify polyglot client patterns work with v0.0.4
  - Verify no deprecation warnings or compiler errors
  - Commit changes after all tests pass: "chore: upgrade node-nats-client-demo to mono v0.0.4"
  - _Success Criteria:_ All tests pass, NATS client patterns work correctly, binary builds
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

- [x] 12. Upgrade python-nats-client-demo to mono v0.0.4

  - Update `projects/python-nats-client-demo/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/python-nats-client-demo .`
  - Run all existing tests: `go test ./...` (2 test files: modules/payment/service_test.go, modules/math/service_test.go)
  - Verify polyglot client patterns work with v0.0.4
  - Verify no deprecation warnings or compiler errors
  - Commit changes after all tests pass: "chore: upgrade python-nats-client-demo to mono v0.0.4"
  - _Success Criteria:_ All tests pass, NATS client patterns work correctly, binary builds
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

- [x] 13. Upgrade sqlc-postgres-demo to mono v0.0.4

  - Update `projects/sqlc-postgres-demo/go.mod` from `github.com/go-monolith/mono v0.0.3` to `v0.0.4`
  - Run `go mod tidy` to update dependencies
  - Build the project: `go build -o ./bin/sqlc-postgres-demo .`
  - Run all existing tests: `go test ./...` (2 test files: modules/user/repository_test.go, modules/user/service_test.go)
  - Verify SQLC and PostgreSQL integration patterns work with v0.0.4
  - Verify no deprecation warnings or compiler errors
  - Commit changes after all tests pass: "chore: upgrade sqlc-postgres-demo to mono v0.0.4"
  - _Success Criteria:_ All tests pass, SQLC patterns work correctly, binary builds
  - _Estimated Complexity:_ S (Small)
  - _Addresses Success Criterion:_ "All modules use the latest compatible mono framework version"

---

**Progress:** 13/13 tasks completed (100%) - GOAL ACHIEVED

**Completion Summary:**
- All 13 projects upgraded from mono v0.0.3 to v0.0.4
- All projects build successfully
- All unit tests pass (where available)
- Note: node-nats-client-demo has pre-existing integration test failures (JetStream infrastructure required) - not caused by the upgrade

**Notes:**
- All tasks follow the "Definition of Safe Change" from constraints.md
- Each task updates one project independently as required by constraints
- No "Don't Touch" areas violated
