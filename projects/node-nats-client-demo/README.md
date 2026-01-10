# Node.js NATS Client Demo with fs-jetstream

This recipe demonstrates **polyglot microservices interoperability** between Node.js clients and Go-based Mono applications via NATS messaging. It showcases file storage operations using the built-in `fs-jetstream` plugin and real-time object store watching.

## What This Demonstrates

- **Node.js + Go Interoperability**: Seamless interaction between Node.js clients and Mono services via NATS
- **fs-jetstream Plugin**: File storage with JetStream object store
- **Service Patterns**: RequestReplyService (file.save) and QueueGroupService (file.archive)
- **Real-time Monitoring**: Object store watching from Node.js to monitor bucket changes
- **UsePluginModule**: Proper plugin dependency injection pattern

## Why Node.js NATS Client?

### The Challenge
Modern applications often need to leverage the strengths of multiple languages:
- Go for high-performance backend services
- Node.js for rapid client development and scripting

### The Solution
NATS provides a language-agnostic messaging bus that enables:
- Zero HTTP overhead with native async patterns
- Natural event-driven architecture
- Load balancing and fault tolerance built-in

### Benefits Over REST APIs
- **Lower Latency**: Direct messaging vs HTTP request/response
- **Simpler Architecture**: No need for HTTP servers, load balancers
- **Better Scalability**: Built-in queue groups for horizontal scaling
- **Real-time Updates**: Native support for pub/sub and streaming

## Project Structure

```
projects/node-nats-client-demo/
├── main.go                      # Go application entry point
├── go.mod                       # Go module definition
├── README.md                    # This file
├── demo.js                      # Interactive demo workflow
├── package.json                 # Node.js dependencies
├── modules/fileops/             # Go Mono module
│   ├── module.go               # Module registration (UsePluginModule)
│   ├── service.go              # File save + archive handlers
│   ├── service_test.go         # Go unit tests
│   └── types.go                # Request/response types
├── client/                      # Node.js client library
│   ├── index.js                # Main exports
│   ├── client.js               # NATS connection manager
│   ├── file-service.js         # RequestReply client (file.save)
│   ├── archive-service.js      # QueueGroup publisher (file.archive)
│   └── watcher.js              # Object store bucket watcher
└── bin/                         # Compiled Go binary
```

## Prerequisites

- **Go 1.21+** - For the Mono backend service
- **Node.js 18+** - For the Node.js client and demo
- **No external NATS required** - Mono embeds NATS server

## Quick Start

### 1. Start the Go Server

```bash
# Build and run
go run .

# Or build binary first
go build -o bin/node-nats-client-demo .
./bin/node-nats-client-demo
```

You should see:
```
=== Node.js NATS Client Demo ===
NATS available at nats://localhost:4222
Services:
  services.fileops.save    - RequestReplyService: Save JSON file to bucket
  services.fileops.archive - QueueGroupService: Archive JSON file as ZIP
```

### 2. Install Node.js Dependencies

```bash
npm install
```

### 3. Run the Demo

```bash
node demo.js
```

The demo will:
1. Connect to NATS
2. Start watching the "user-settings" bucket
3. Save 3 user setting files
4. Archive them as ZIP files
5. Show all events in real-time

## Service Patterns

### RequestReplyService (file.save)

**When to use**: Operations that need immediate response with result data

```javascript
const fileService = new FileService(nc);
const result = await fileService.saveFile('settings.json', { theme: 'dark' });
console.log(result.file_id, result.size, result.digest);
```

**Characteristics**:
- Synchronous request/response
- Client waits for result
- Returns success/error data
- One-to-one communication

### QueueGroupService (file.archive)

**When to use**: Background tasks that don't need immediate response

```javascript
const archiveService = new ArchiveService(nc);
await archiveService.archiveFile(fileId);  // Fire-and-forget
```

**Characteristics**:
- Asynchronous fire-and-forget
- Load balanced across workers
- No response expected
- Scalable horizontally

### Object Store Watching

**When to use**: Real-time monitoring of file changes

```javascript
const watcher = new BucketWatcher(nc, 'user-settings');
await watcher.initialize();
await watcher.watch((event) => {
  console.log(event.name, event.deleted, event.size);
});
```

**Use Cases**:
- File audit logging
- Real-time dashboards
- Change detection systems
- Webhook notifications

## How It Works

### 1. File Save Workflow

```
Node.js Client → services.fileops.save → Go Handler → fs-jetstream → JetStream Object Store
                                              ↓
                        Response ← ← ← ← file_id, size, digest
```

1. Client sends JSON request with filename and content
2. Go handler generates UUID, creates storage key `{uuid}/{filename}.json`
3. Saves to fs-jetstream bucket with metadata headers
4. Returns file_id, size, and digest to client

### 2. Archive Workflow

```
Node.js Client → services.fileops.archive (queue) → Go Worker
                                                        ↓
                                              1. Get original file
                                              2. Create ZIP in memory
                                              3. Save ZIP to bucket
                                              4. Delete original
```

This is fire-and-forget - client doesn't wait for completion. Multiple workers can process the queue concurrently.

### 3. Watch Workflow

```
Node.js Client → JetStream Object Store → Watch Stream
                         ↓
                   File Events (create, update, delete)
                         ↓
                   Callback Function
```

The watcher subscribes to the object store's metadata stream and receives events for all changes.

## API Reference

### FileService

```javascript
const fileService = new FileService(nc);

// Save a file
await fileService.saveFile(filename, content, timeout = 5000);

// Convenience method for user settings
await fileService.saveUserSettings(userId, settings);
```

### ArchiveService

```javascript
const archiveService = new ArchiveService(nc);

// Archive a single file
await archiveService.archiveFile(fileId);

// Archive multiple files
await archiveService.archiveMultiple([fileId1, fileId2]);
```

### BucketWatcher

```javascript
const watcher = new BucketWatcher(nc, bucketName);

// Initialize
await watcher.initialize();

// Watch with options
await watcher.watch(callback, {
  includeHistory: false,  // Don't replay existing files
  ignoreDeletes: false    // Include delete events
});

// Stop watching
await watcher.stop();
```

## Testing

```bash
# Run Go tests
go test ./modules/fileops -v

# Run Node.js tests (when implemented)
npm test
```

## When to Use This Pattern

✅ **Good fit for:**
- Polyglot microservices architectures
- File storage with NATS as the backbone
- Background job processing
- Real-time file monitoring and synchronization
- Distributed systems needing message-based communication

❌ **Not ideal for:**
- Simple REST APIs where HTTP is sufficient
- Applications that can't run embedded NATS
- Scenarios requiring strict message ordering guarantees
- Very high-throughput file transfers (use dedicated file servers)

## Key Implementation Details

### File Naming Strategy
Files are stored with UUID-based prefixes to avoid naming collisions:
```
{uuid}/{filename}.json  → Original file
{uuid}/{filename}.zip   → Archived file
```

### Archive Strategy
- Creates ZIP archive in memory using Go's `archive/zip` package
- Preserves original filename within ZIP
- Deletes original only after successful ZIP creation
- Handles missing files gracefully (logs warning, doesn't fail)

### Error Handling
- **Go Service**: Returns errors in response JSON (for visibility to clients)
- **Node.js Client**: Throws errors for connection issues, parses error field from responses
- **QueueGroup**: Fire-and-forget, logs errors but doesn't propagate them

## Learn More

- [Mono Framework Documentation](https://github.com/go-monolith/mono)
- [NATS Documentation](https://docs.nats.io/)
- [JetStream Object Store](https://docs.nats.io/nats-concepts/jetstream/obj_store)
- [nats.js Client](https://github.com/nats-io/nats.js)

## License

MIT
