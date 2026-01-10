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

- [ ] 1. Upgrade rate-limiting-middleware to mono v0.0.4

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

- [ ] 2. Upgrade jwt-auth-demo to mono v0.0.4

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

- [ ] 3. Upgrade background-jobs-demo to mono v0.0.4

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

- [ ] 4. Upgrade redis-caching-plugin to mono v0.0.4

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

- [ ] 5. Upgrade url-shortener-demo to mono v0.0.4

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

---

**Progress:** 0/5 tasks completed (0%)

**Notes:**
- All tasks follow the "Definition of Safe Change" from constraints.md
- Each task updates one project independently as required by constraints
- Test coverage validated: all selected projects have passing tests
- No "Don't Touch" areas violated (no modules with failing tests selected)
- Remaining 8 projects will be upgraded in subsequent task batches after these 5 complete successfully
