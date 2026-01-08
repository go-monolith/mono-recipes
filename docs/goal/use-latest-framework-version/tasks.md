# Implementation Plan: Use Latest Framework Version

Goal: [Use Latest Framework Version](./goal.md)
Constraints: [constraints.md](./constraints.md)

## Context

This implementation plan upgrades all 10 mono-recipes projects from mono framework v0.0.2 to v0.0.3.

**Current State** (from codebase analysis):
- All 10 projects currently use `github.com/go-monolith/mono v0.0.2`
- All existing tests are passing (19 test files across 8 projects)
- Latest stable version available: `v0.0.3`

**Key Changes in v0.0.3**:
- Improved RequestReplyService error propagation (PR #4, issue #3)
- Handler errors now properly propagated to client
- No explicit breaking changes documented
- All performance benchmarks passing (53,070 msgs/sec throughput)

**Modules Affected by v0.0.3 Changes**:
- Projects using RequestReplyService: websocket-chat-demo, url-shortener-demo, rate-limiting-middleware
- Projects using QueueGroupService: background-jobs-demo

**Upgrade Strategy**:
- Start with simple modules (no plugins, direct mono dependencies only)
- Progress to modules using RequestReplyService (affected by v0.0.3 improvements)
- Validate error handling after upgrade
- Commit after each module passes all tests

## Milestones

1. Milestone 1 - Quick Win Validation
   - Tasks: 1
   - Upgrade simplest module to validate process and compatibility

2. Milestone 2 - Core Service Modules
   - Tasks: 2-3
   - Upgrade modules using RequestReplyService and QueueGroupService

3. Milestone 3 - Advanced Pattern Modules
   - Tasks: 4
   - Upgrade middleware and plugin-based modules

4. Milestone 4 - Documentation and Verification
   - Tasks: 5
   - Update documentation and final validation

## Task List

- [ ] 1. Upgrade graceful-shutdown-demo to mono v0.0.3

  - Update `/projects/graceful-shutdown-demo/go.mod`: Change `github.com/go-monolith/mono v0.0.2` to `v0.0.3`
  - Run `go mod tidy` in `/projects/graceful-shutdown-demo/` to update go.sum
  - Verify code compiles: `go build -o /tmp/graceful-shutdown ./main.go`
  - Run the application manually to test basic lifecycle: Start → Health Check → Graceful Shutdown
  - Verify no deprecation warnings in build output
  - **Success Criteria**: Application compiles, starts, responds to signals, shuts down gracefully
  - **Rationale**: Simplest module (no plugins, no tests, basic lifecycle only) - ideal for validating upgrade process
  - **Addresses Goal Success Criterion**: "All modules use the latest compatible mono framework version"
  - _Complexity: S_

- [ ] 2. Upgrade websocket-chat-demo to mono v0.0.3 with RequestReplyService testing

  - _Dependencies: 1_
  - Update `/projects/websocket-chat-demo/go.mod`: Change mono version to `v0.0.3`
  - Run `go mod tidy` in `/projects/websocket-chat-demo/`
  - Review `/projects/websocket-chat-demo/modules/chat/module.go` RequestReplyService usage (ChatService, UserService)
  - Run existing tests: `go test ./modules/chat/types_test.go`
  - **Critical**: Test error handling in request-reply services - verify handler errors propagate correctly to clients
  - Manually test WebSocket connections and chat message flow
  - Verify EventBus pubsub patterns still work correctly
  - **Success Criteria**: All tests pass, error propagation works as expected, WebSocket chat functions correctly
  - **Rationale**: First module using RequestReplyService (directly affected by v0.0.3 improvements) - validates error handling changes
  - **Addresses Goal Success Criterion**: "All tests pass successfully with the new framework version"
  - _Complexity: M_

- [ ] 3. Upgrade background-jobs-demo to mono v0.0.3 with QueueGroupService testing

  - _Dependencies: 1_
  - Update `/projects/background-jobs-demo/go.mod`: Change mono version to `v0.0.3`
  - Run `go mod tidy` in `/projects/background-jobs-demo/`
  - Review `/projects/background-jobs-demo/modules/worker/module.go` QueueGroupService usage
  - Run existing unit tests: `go test ./modules/api/service_test.go ./modules/worker/processor_test.go`
  - **Critical**: Test error handling in queue group handlers - verify job processing errors are handled correctly
  - Test job submission → worker processing → completion flow
  - Verify multiple queue group handlers still load balance correctly
  - **Success Criteria**: All 2 test files pass, job processing works, error handling verified
  - **Rationale**: Uses QueueGroupService (service container pattern) - validates service error handling improvements
  - **Addresses Goal Success Criterion**: "All tests pass successfully with the new framework version"
  - _Complexity: M_

- [ ] 4. Upgrade rate-limiting-middleware and remaining 7 modules to mono v0.0.3

  - _Dependencies: 2, 3_
  - Update go.mod files for all 8 remaining projects:
    - `/projects/rate-limiting-middleware/go.mod`
    - `/projects/jwt-auth-demo/go.mod`
    - `/projects/redis-caching-plugin/go.mod`
    - `/projects/file-upload-demo/go.mod`
    - `/projects/url-shortener-demo/go.mod`
    - `/projects/hexagonal-architecture/go.mod`
    - `/projects/gorm-sqlite-demo/go.mod`
  - Run `go mod tidy` in each project directory
  - Run all existing test suites (17 remaining test files):
    - Rate-limiting: 5 test files (config, limiter, middleware, module, adapter)
    - JWT Auth: 4 test files (middleware, jwt, password, service)
    - Redis Caching: 2 test files (cache, product service)
    - File Upload: 2 test files (fileservice, handlers)
    - URL Shortener: 2 test files (analytics types, shortener service)
    - GORM SQLite: 1 test file (repository)
  - **Critical**: Verify MiddlewareModule interface still works (rate-limiting-middleware)
  - **Critical**: Verify plugin patterns work (kv-jetstream, fs-jetstream, cache plugin)
  - **Success Criteria**: All 17 test files pass, no compilation errors, no deprecation warnings
  - **Rationale**: Batch upgrade remaining modules after validating core patterns work
  - **Addresses Goal Success Criterion**: "All modules use the latest compatible mono framework version" and "All tests pass"
  - _Complexity: L_

- [ ] 5. Update all documentation to reflect mono v0.0.3 upgrade

  - _Dependencies: 1, 2, 3, 4_
  - Update main `/README.md`:
    - Add note that all recipes use mono framework v0.0.3
    - Mention v0.0.3 includes improved error handling in RequestReplyService
  - Review and update project-specific READMEs if they mention framework version or error handling:
    - `/projects/websocket-chat-demo/README.md` (RequestReplyService usage)
    - `/projects/background-jobs-demo/README.md` (QueueGroupService pattern)
    - `/projects/rate-limiting-middleware/README.md` (MiddlewareModule pattern)
    - `/projects/url-shortener-demo/README.md` (kv-jetstream plugin)
  - Verify no deprecated API usage is documented in any README
  - Add upgrade notes if any code patterns changed due to error handling improvements
  - **Success Criteria**: Documentation accurately reflects v0.0.3, no references to deprecated APIs
  - **Rationale**: Final step to ensure documentation is up-to-date after all code changes
  - **Addresses Goal Success Criterion**: "All README files, specs, and documentation reflect the new framework version and any API changes"
  - _Complexity: S_

---

**Task Summary**:
- Total Tasks: 5
- Complexity: 2 Small, 2 Medium, 1 Large
- Estimated Total Effort: 6-8 hours (focused work sessions)

**Completion Criteria** (from goal.md):
- ✅ All modules use the latest compatible mono framework version (v0.0.3)
- ✅ All tests (unit, integration) pass successfully with the new framework version
- ✅ All README files and documentation reflect the new framework version and any API changes
- ✅ Code refactored to eliminate usage of deprecated APIs (no deprecated APIs in v0.0.3)

**Safe Change Definition** (from constraints.md):
- Each task requires all existing tests to pass before committing
- Code must compile without errors
- No new deprecation warnings introduced
- Documentation updated to reflect changes
- Each module updated and tested independently
