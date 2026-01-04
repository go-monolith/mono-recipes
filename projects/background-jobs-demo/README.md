# Background Jobs with QueueGroupService

This recipe demonstrates how to build a background job processing system using the mono framework's `QueueGroupService` pattern. It showcases asynchronous task processing with load-balanced queue groups.

## Why Use QueueGroupService for Background Processing?

The mono framework's `QueueGroupService` pattern provides:

- **Fire-and-Forget Semantics**: Submit jobs without waiting for responses
- **Load Balancing**: Framework automatically distributes jobs across workers
- **Embedded NATS**: No external message broker required
- **Simple Module Dependencies**: Declared via `DependentModule` interface

## Architecture Overview

```
┌──────────────┐       ┌────────────────────┐       ┌─────────────────────────────┐
│              │       │                    │       │        Worker Module         │
│  REST API    ├──────►│  QueueGroupService ├──────►├─────────────────────────────┤
│  (Producer)  │       │  (Embedded NATS)   │       │  email-worker               │
│              │       │                    │       │  image-processing-worker    │
└──────────────┘       └────────────────────┘       │  report-generation-worker   │
                                                    └──────────────┬──────────────┘
                                                                   │
                                                            ┌──────▼───────┐
                                                            │              │
                                                            │   Job Store  │
                                                            │  (In-Memory) │
                                                            │              │
                                                            └──────────────┘
```

### Components

1. **REST API**: Accepts job requests and sends them to the worker via `QueueGroupService`
2. **QueueGroupService**: Framework-managed NATS queue with automatic load balancing
3. **Worker Module**: 3 QueueGroups, each handling a specific job type:
   - `email-worker` - handles email jobs
   - `image-processing-worker` - handles image processing jobs
   - `report-generation-worker` - handles report generation jobs
4. **Job Store**: In-memory storage for job status tracking

## QueueGroupService Pattern

### Worker Module (Service Provider)

This demo registers **3 QueueGroups on the same service**, each handling a specific job type:

```go
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
    return container.RegisterQueueGroupService(
        "process-job",
        mono.QGHP{
            QueueGroup: "email-worker",
            Handler:    m.handleJobTypeEmail,
        },
        mono.QGHP{
            QueueGroup: "image-processing-worker",
            Handler:    m.handleJobTypeImageProcessing,
        },
        mono.QGHP{
            QueueGroup: "report-generation-worker",
            Handler:    m.handleJobTypeReportGeneration,
        },
    )
}

// Each handler filters for its specific job type
func (m *Module) handleJobTypeEmail(ctx context.Context, msg *mono.Msg) error {
    var j job.Job
    json.Unmarshal(msg.Data, &j)

    // Filter: only process email jobs
    if j.Type != job.JobTypeEmail {
        return nil  // Ignore other job types
    }

    return m.processJob(ctx, &j, "email-worker")
}
```

### API Module (Service Consumer)

```go
func (m *Module) Dependencies() []string {
    return []string{"worker"}
}

func (m *Module) SetDependencyServiceContainer(module string, container mono.ServiceContainer) {
    if module == "worker" {
        m.workerContainer = container
    }
}

// In CreateJob:
client, _ := s.workerContainer.GetQueueGroupService("process-job")
client.Send(ctx, jobData)  // Fire-and-forget
```

## Project Structure

```
background-jobs-demo/
├── domain/
│   └── job/
│       ├── types.go       # Job types, statuses, and payloads
│       ├── errors.go      # Domain errors
│       └── store.go       # In-memory job storage
├── modules/
│   ├── worker/
│   │   ├── processor.go   # Job processing logic
│   │   ├── processor_test.go
│   │   └── module.go      # QueueGroupService provider
│   └── api/
│       ├── service.go     # Job service layer
│       ├── service_test.go
│       ├── handlers.go    # HTTP handlers
│       └── module.go      # API module (depends on worker)
├── main.go
├── demo.py
└── README.md
```

## Job Types

This demo implements three types of background jobs:

### 1. Email Sending

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

### 2. Image Processing

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

### 3. Report Generation

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
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:05Z"
}
```

### List Jobs

```bash
GET /api/v1/jobs?status=completed&type=email&limit=50&offset=0
```

## Running the Demo

### Prerequisites

- Go 1.21+
- Python 3.7+ (for demo script)

### Run the Application

```bash
go run main.go
```

The application will:
- Start with embedded NATS (no external setup needed)
- Register the worker module with QueueGroupService
- Launch the REST API on port 8080

### Run the Demo Script

```bash
python3 demo.py
```

The demo script will:
1. Enqueue jobs of different types
2. Poll job status in real-time
3. Display progress updates
4. Show completed and failed jobs

## Testing

### Unit Tests

```bash
go test ./... -v
```

## Key Differences from Complex Implementation

| Aspect | Complex Version | This Version |
|--------|-----------------|--------------|
| Message Broker | External NATS JetStream | Embedded NATS |
| Worker Pool | Manual goroutine pool | Framework-managed |
| Retry Logic | Exponential backoff, DLQ | Fire-and-forget |
| Event Bus | Custom in-memory pub/sub | Removed |
| Setup | docker-compose required | Just `go run main.go` |
| Code Volume | ~1000 LOC | ~400 LOC |

## Trade-offs

### What This Version Doesn't Have:
- Retry with exponential backoff
- Dead-letter queue for failed jobs
- Job lifecycle events via EventBus
- Configurable worker pool size

### What This Version Provides:
- Simpler codebase (~60% less code)
- No external dependencies (no Docker needed)
- Framework handles NATS subscriptions
- Clear module boundaries with declared dependencies
- Easier testing with mock ServiceContainer

## Key Takeaways

1. **QueueGroupService** provides fire-and-forget messaging with automatic load balancing
2. **DependentModule** interface enables explicit dependency declaration
3. **Embedded NATS** eliminates external infrastructure requirements
4. **ServiceContainer** enables clean inter-module communication
5. **Simpler code** is easier to understand, test, and maintain

## Further Reading

- [Mono Framework Documentation](https://github.com/go-monolith/mono)
- [NATS Queue Groups](https://docs.nats.io/nats-concepts/core-nats/queue)
