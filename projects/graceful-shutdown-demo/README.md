# Graceful Shutdown Demo

A demonstration project showcasing how to build a modular monolith application using the [go-monolith/mono](https://github.com/go-monolith/mono) framework (v0.1.1) with graceful shutdown capabilities powered by [gelmium/graceful-shutdown](https://github.com/gelmium/graceful-shutdown).

## Features

- **Modular Architecture**: Built with the mono framework for clear module boundaries
- **Graceful Shutdown**: Handles OS signals (SIGINT, SIGTERM) and ensures clean application shutdown
- **In-Flight Request Handling**: HTTP server waits for ongoing requests to complete before shutting down, bounded by the shutdown deadline
- **Background Worker**: Demonstrates graceful shutdown of background tasks
- **Context-Aware Operations**: Every `Stop` returns within the deadline of the context it is given
- **Conformance Tests**: `monotest.CheckModule` per module, including the Stop-timeout rule

## Project Structure

```
graceful-shutdown-demo/
├── main.go                    # Application entry point (strict module boundaries)
├── modules/
│   ├── httpserver/
│   │   ├── module.go          # HTTP server module (Fiber-based)
│   │   └── module_test.go     # monotest.CheckModule + Stop-deadline test
│   └── worker/
│       ├── module.go          # Background worker module
│       └── module_test.go     # monotest.CheckModule (Stop interrupts a running task)
├── go.mod
└── README.md
```

## Modules

### HTTP Server Module
- Created with `httpserver.NewModule(httpserver.Config{Addr: ":3000"})`
- Runs a Fiber HTTP server on port 3000 (the listener is bound in `Start`, so a busy port fails startup)
- Provides three endpoints:
  - `GET /` - Basic hello endpoint
  - `GET /health` - Health check endpoint
  - `GET /slow` - Slow endpoint (5 seconds) to demonstrate in-flight request handling
- `Stop` calls `ShutdownWithContext(ctx)`: no new connections, in-flight requests drain until the shutdown deadline, and requests still running at the deadline are cancelled (`503`)

### Background Worker Module
- Created with `worker.NewModule(worker.Config{})` (defaults: a 500ms task every 2 seconds)
- Runs periodic tasks every 2 seconds
- Demonstrates graceful task interruption during shutdown
- Uses sync.Once to prevent panic from double-close

## Running the Application

### Build

```bash
go build -o bin/graceful-shutdown-demo .
```

### Run

```bash
./bin/graceful-shutdown-demo
```

Or directly with Go:

```bash
go run main.go
```

## Testing Graceful Shutdown

### Test 1: Basic Shutdown

1. Start the application
2. Press `Ctrl+C`
3. Observe the graceful shutdown sequence in the logs

### Test 2: In-Flight Request Handling

1. Start the application
2. In a terminal, make a slow request:
   ```bash
   curl http://localhost:3000/slow
   ```
3. While the request is processing, send a shutdown signal from another terminal:
   ```bash
   pkill -SIGTERM graceful-shutdown-demo
   ```
4. Observe that the application waits for the request to complete before shutting down (the `curl` still receives its `200` response)

### Test 3: Quick Requests

```bash
# Health check
curl http://localhost:3000/health

# Hello endpoint
curl http://localhost:3000/
```

## How It Works

1. **Application Startup**:
   - Creates a mono application with a 30-second shutdown timeout and `mono.WithStrictModuleBoundaries()`
   - Registers HTTP server and background worker modules
   - Starts all modules

2. **Graceful Shutdown Integration**:
   - `gfshutdown.GracefulShutdown()` monitors OS signals (SIGINT, SIGTERM, SIGHUP)
   - On signal receipt, it calls the mono app's `Stop()` method
   - The mono framework stops all modules in reverse order, passing a context with the shutdown deadline
   - Each module performs its cleanup within that deadline (HTTP server drains requests, worker interrupts its current task)

3. **Exit Code Handling**:
   - Returns exit code 0 if shutdown completes within timeout
   - Returns exit code 1 if timeout is exceeded

## Module Boundaries

The two modules provide and consume no services and do not know about each other, so there is no `contract` package: there is nothing to publish. (A module that exposes services or events gets a `modules/<name>/contract` package with typed contracts, and a module that calls them declares them in `ConsumeServices()`; see the other recipes.)

`main.go` still runs with `mono.WithStrictModuleBoundaries()`: should a module later reach into another one (for example a raw request on its `services.*` subject), startup or the call fails instead of only logging a warning at `Stop`.

### Conformance tests

```bash
go test ./... -v
```

Each module has a `monotest.CheckModule` test (no NATS). For this demo the interesting rule is rule 8: `Start` and `Stop` succeed, and `Stop` returns within `monotest.WithStopTimeout`:

- `httpserver`: the exercise leaves a `/slow` request in flight; `Stop` must drain it (the request still gets `200`) within the 2s stop timeout. A second test gives `Stop` a 300ms deadline with a 10s request in flight and checks that `Stop` returns at the deadline and the request is cancelled.
- `worker`: a task that would take 10s is running when `Stop` is called; `Stop` must interrupt it and return within 1s.

## Dependencies

- [github.com/go-monolith/mono](https://github.com/go-monolith/mono) - Modular monolith framework
- [github.com/gelmium/graceful-shutdown](https://github.com/gelmium/graceful-shutdown) - Graceful shutdown library
- [github.com/gofiber/fiber/v2](https://github.com/gofiber/fiber) - HTTP web framework

## Code Quality

This project includes:
- Proper error handling with error wrapping
- Context-aware operations for cancellation support
- Thread-safe shutdown with sync.Once
- Startup error detection for the HTTP server (synchronous listen)
- Defensive programming against edge cases

## License

This is a demonstration project for educational purposes.
