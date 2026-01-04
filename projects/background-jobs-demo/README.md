# Background Jobs with NATS JetStream and Worker Pools

This recipe demonstrates how to build a robust background job processing system using NATS JetStream as a message broker and worker pools for concurrent job execution. It showcases asynchronous task processing, retry strategies, dead-letter queues, and job progress tracking.

## Why Use Message Queues for Background Processing?

Message queues decouple job producers from consumers, enabling:

- **Scalability**: Process jobs asynchronously without blocking the main application
- **Reliability**: Jobs persist in the queue even if workers crash or restart
- **Load Distribution**: Multiple workers can process jobs concurrently from the same queue
- **Resilience**: Failed jobs can be retried automatically with exponential backoff
- **Observability**: Track job progress, failures, and processing metrics in real-time

## Architecture Overview

```
┌──────────────┐       ┌────────────────┐       ┌──────────────┐
│              │       │                │       │              │
│  REST API    ├──────►│  NATS          ├──────►│  Worker Pool │
│  (Producer)  │       │  JetStream     │       │  (Consumer)  │
│              │       │                │       │              │
└──────────────┘       └────────┬───────┘       └──────┬───────┘
                                │                      │
                                │                      │
                        ┌───────▼────────┐    ┌────────▼────────┐
                        │                │    │                 │
                        │  Dead-Letter   │    │    EventBus     │
                        │     Queue      │    │  (Progress)     │
                        │                │    │                 │
                        └────────────────┘    └─────────────────┘
```

### Components

1. **REST API**: Accepts job requests and enqueues them to NATS JetStream
2. **NATS JetStream**: Message broker that stores jobs persistently and delivers them to workers
3. **Worker Pool**: Concurrent workers that pull jobs from the queue and process them
4. **Dead-Letter Queue**: Captures jobs that exceed maximum retry attempts
5. **EventBus**: Publishes real-time job progress events for monitoring

## Worker Pool Pattern and Concurrency

### Worker Pool Design

The worker pool pattern allows you to:

- Control concurrency by limiting the number of concurrent workers
- Share a message subscription across multiple workers for load balancing
- Process jobs in parallel while respecting resource constraints
- Gracefully shut down workers without losing in-flight jobs

### Configuration

```go
workerConfig := worker.PoolConfig{
    NumWorkers:     3,               // Number of concurrent workers
    MaxRetries:     5,               // Maximum retry attempts per job
    BaseRetryDelay: time.Second,     // Initial retry delay
    MaxRetryDelay:  time.Minute,     // Maximum retry delay
    ProcessTimeout: 5 * time.Minute, // Job processing timeout
}
```

### How It Works

1. Each worker runs in its own goroutine
2. Workers share a single NATS subscription (pull consumer)
3. NATS JetStream distributes jobs across workers automatically
4. Workers acknowledge successful jobs or negatively acknowledge failures
5. Failed jobs are retried with exponential backoff

## Retry Strategies and Dead-Letter Queues

### Exponential Backoff

When a job fails, it's retried with increasing delays:

```
Retry 1: 1 second
Retry 2: 2 seconds
Retry 3: 4 seconds
Retry 4: 8 seconds
Retry 5: 16 seconds
```

This prevents overwhelming external services and gives transient issues time to resolve.

### Dead-Letter Queue (DLQ)

Jobs that exceed the maximum retry count are moved to a dead-letter queue for manual inspection:

- **Purpose**: Prevent infinite retries of permanently failed jobs
- **Benefits**:
  - Preserve failing jobs for debugging
  - Prevent queue backlog from bad jobs
  - Enable manual reprocessing after fixes
- **Implementation**: Separate NATS stream (`jobs-dlq`) for failed jobs

## Idempotency and Exactly-Once Processing

### Idempotency

Idempotent operations produce the same result when executed multiple times:

```go
// Good: Idempotent
SET user:123:email = "user@example.com"

// Bad: Not idempotent
INCREMENT user:123:login_count
```

### Achieving Idempotency

1. **Job IDs**: Each job has a unique ID (UUID) for deduplication
2. **State Checks**: Check current state before applying changes
3. **Atomic Operations**: Use database transactions or atomic primitives
4. **Result Storage**: Store processing results to detect duplicates

### Exactly-Once Semantics

While NATS JetStream provides at-least-once delivery, exactly-once processing requires:

- **Deduplication**: Track processed job IDs in a database or cache
- **Acknowledgment**: Only acknowledge jobs after successful completion
- **Retry Logic**: Ensure retries don't cause duplicate side effects

## Project Structure

```
background-jobs-demo/
├── domain/
│   └── job/
│       ├── types.go       # Job types, statuses, and payloads
│       ├── errors.go      # Domain errors
│       ├── events.go      # Job lifecycle events
│       └── store.go       # In-memory job storage
├── modules/
│   ├── nats/
│   │   ├── client.go      # NATS JetStream client
│   │   └── module.go      # NATS mono module
│   ├── eventbus/
│   │   ├── eventbus.go    # In-memory event bus
│   │   └── module.go      # EventBus mono module
│   ├── worker/
│   │   ├── processor.go   # Job processing logic
│   │   ├── pool.go        # Worker pool implementation
│   │   └── module.go      # Worker mono module
│   └── api/
│       ├── service.go     # Job service layer
│       ├── handlers.go    # HTTP handlers
│       └── module.go      # API mono module
├── main.go
├── docker-compose.yml
├── demo.py
└── README.md
```

## Job Types

This demo implements three types of background jobs:

### 1. Email Sending (Async Task)

Simulates sending emails asynchronously:

```json
{
  "type": "email",
  "payload": {
    "to": "user@example.com",
    "subject": "Welcome!",
    "body": "Thanks for signing up"
  }
}
```

### 2. Image Processing (Long-Running Task)

Simulates CPU-intensive image processing:

```json
{
  "type": "image_processing",
  "payload": {
    "image_url": "https://example.com/image.jpg",
    "operations": ["resize", "watermark"],
    "output_path": "/output/processed.jpg"
  }
}
```

### 3. Report Generation (Batch Task)

Simulates generating reports from large datasets:

```json
{
  "type": "report_generation",
  "payload": {
    "report_type": "monthly_sales",
    "format": "pdf",
    "date_range": {
      "start": "2024-01-01",
      "end": "2024-01-31"
    }
  }
}
```

## API Endpoints

### Create Job

```bash
POST /api/v1/jobs
Content-Type: application/json

{
  "type": "email",
  "payload": {
    "to": "user@example.com",
    "subject": "Test Email",
    "body": "This is a test"
  },
  "priority": 1
}
```

### Get Job Status

```bash
GET /api/v1/jobs/:id
```

Response:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "email",
  "status": "processing",
  "progress": 50,
  "progress_message": "Sending email...",
  "retry_count": 0,
  "max_retries": 5,
  "worker_id": "worker-1",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:05Z",
  "started_at": "2024-01-15T10:00:02Z"
}
```

### List Jobs

```bash
GET /api/v1/jobs?status=completed&type=email&limit=50&offset=0
```

## Running the Demo

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Python 3.7+ (for demo script)

### Start NATS JetStream

```bash
docker-compose up -d
```

This starts NATS JetStream with:
- Client port: 4222
- Monitoring dashboard: http://localhost:8222
- Persistent storage for job durability

### Run the Application

```bash
go run main.go
```

The application will:
- Connect to NATS JetStream
- Start 3 worker goroutines
- Launch the REST API on port 8080

### Run the Demo Script

```bash
python3 demo.py
```

The demo script will:
1. Enqueue jobs of different types
2. Poll job status in real-time
3. Display progress updates
4. Show completed, failed, and dead-letter queue jobs

## Job Lifecycle Events

The EventBus publishes events for job state changes:

- **JobStarted**: Worker begins processing a job
- **JobProgress**: Job reports progress (0-100%)
- **JobCompleted**: Job finishes successfully
- **JobFailed**: Job fails (will retry if retries remain)
- **JobDeadLetter**: Job moved to DLQ after max retries

Subscribe to events:

```go
eventBus.SubscribeAll(func(event *eventbus.Event) {
    switch event.Type {
    case eventbus.EventTypeJobCompleted:
        data := event.Data.(*eventbus.JobCompletedData)
        log.Printf("Job %s completed in %dms", data.JobID, data.DurationMs)
    }
})
```

## Monitoring and Observability

### NATS Monitoring

Access the NATS monitoring dashboard at http://localhost:8222:

- Stream statistics
- Consumer acknowledgment rates
- Message counts and delivery statistics
- JetStream memory and disk usage

### Application Logs

The application logs job lifecycle events:

```
[event] Job started: 550e8400... (type=email, worker=worker-1)
[event] Job progress: 550e8400... (progress=25%, message=Connecting to SMTP...)
[event] Job progress: 550e8400... (progress=75%, message=Sending email...)
[event] Job completed: 550e8400... (type=email, duration=3245ms)
```

## Testing

### Unit Tests

Run unit tests for job service and worker:

```bash
go test ./... -v
```

### Integration Tests

Test end-to-end job processing:

```bash
# Start NATS
docker-compose up -d

# Run integration tests
go test ./... -tags=integration -v
```

## Production Considerations

### Scaling Workers

Increase worker count for higher throughput:

```go
workerConfig := worker.PoolConfig{
    NumWorkers: 10, // Scale based on workload
}
```

### Resource Limits

Set appropriate timeouts and limits:

```go
ProcessTimeout:  5 * time.Minute,  // Job processing timeout
MaxRetries:      5,                 // Max retry attempts
MaxRetryDelay:   5 * time.Minute,  // Cap retry delays
```

### Error Handling

- **Transient Errors**: Retry automatically (network timeouts, rate limits)
- **Permanent Errors**: Move to DLQ immediately (invalid payloads, auth failures)
- **Partial Failures**: Save progress and resume on retry

### Dead-Letter Queue Management

Monitor and process DLQ jobs:

```bash
# View DLQ messages via NATS CLI
nats stream view jobs-dlq
```

### Monitoring Alerts

Set up alerts for:
- High job failure rates
- DLQ queue depth exceeding threshold
- Worker pool saturation
- Processing latency spikes

## Key Takeaways

1. **Message queues** enable asynchronous processing and improve system resilience
2. **Worker pools** provide controlled concurrency and resource management
3. **Exponential backoff** prevents overwhelming services during failures
4. **Dead-letter queues** capture permanently failed jobs for manual intervention
5. **Idempotency** is crucial for exactly-once processing semantics
6. **NATS JetStream** provides persistence, replay, and exactly-once delivery guarantees
7. **Event-driven architecture** enables real-time monitoring and observability

## Further Reading

- [NATS JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
- [Worker Pool Pattern](https://gobyexample.com/worker-pools)
- [Exponential Backoff and Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
- [Idempotency in Distributed Systems](https://www.2ndquadrant.com/en/blog/idempotency/)
