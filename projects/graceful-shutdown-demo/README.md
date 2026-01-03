# Graceful Shutdown Demo

A demonstration project showcasing how to build a modular monolith application using the [go-monolith/mono](https://github.com/go-monolith/mono) framework with graceful shutdown capabilities powered by [gelmium/graceful-shutdown](https://github.com/gelmium/graceful-shutdown).

## Features

- **Modular Architecture**: Built with the mono framework for clear module boundaries
- **Graceful Shutdown**: Handles OS signals (SIGINT, SIGTERM) and ensures clean application shutdown
- **In-Flight Request Handling**: HTTP server waits for ongoing requests to complete before shutting down
- **Background Worker**: Demonstrates graceful shutdown of background tasks
- **Context-Aware Operations**: All blocking operations respect cancellation signals

## Project Structure

```
graceful-shutdown-demo/
├── main.go                    # Application entry point
├── modules/
│   ├── httpserver/
│   │   └── module.go          # HTTP server module (Fiber-based)
│   └── worker/
│       └── module.go          # Background worker module
├── go.mod
└── README.md
```

## Modules

### HTTP Server Module
- Runs a Fiber HTTP server on port 3000
- Provides three endpoints:
  - `GET /` - Basic hello endpoint
  - `GET /health` - Health check endpoint
  - `GET /slow` - Slow endpoint (5 seconds) to demonstrate in-flight request handling
- Implements context-aware request processing

### Background Worker Module
- Runs periodic tasks every 2 seconds
- Demonstrates graceful task interruption during shutdown
- Uses sync.Once to prevent panic from double-close

## Running the Application

### Build

```bash
go build -o bin/graceful-shutdown-demo
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
4. Observe that the application waits for the request to complete before shutting down

### Test 3: Quick Requests

```bash
# Health check
curl http://localhost:3000/health

# Hello endpoint
curl http://localhost:3000/
```

## How It Works

1. **Application Startup**:
   - Creates a mono application with a 30-second shutdown timeout
   - Registers HTTP server and background worker modules
   - Starts all modules

2. **Graceful Shutdown Integration**:
   - `gfshutdown.GracefulShutdown()` monitors OS signals (SIGINT, SIGTERM, SIGHUP)
   - On signal receipt, it calls the mono app's `Stop()` method
   - The mono framework stops all modules in reverse order
   - Each module performs its cleanup (HTTP server waits for requests, worker completes tasks)

3. **Exit Code Handling**:
   - Returns exit code 0 if shutdown completes within timeout
   - Returns exit code 1 if timeout is exceeded

## Dependencies

- [github.com/go-monolith/mono](https://github.com/go-monolith/mono) - Modular monolith framework
- [github.com/gelmium/graceful-shutdown](https://github.com/gelmium/graceful-shutdown) - Graceful shutdown library
- [github.com/gofiber/fiber/v2](https://github.com/gofiber/fiber) - HTTP web framework

## Code Quality

This project includes:
- Proper error handling with error wrapping
- Context-aware operations for cancellation support
- Thread-safe shutdown with sync.Once
- Startup error detection for the HTTP server
- Defensive programming against edge cases

## License

This is a demonstration project for educational purposes.
