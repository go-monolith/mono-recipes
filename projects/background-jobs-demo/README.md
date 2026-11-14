# Background Jobs with QueueGroupService

This recipe demonstrates how to build a background job processing system using the [mono framework](https://github.com/go-monolith/mono) v0.1.1 `QueueGroupService` pattern. It showcases asynchronous task processing with load-balanced queue groups.

## Why Use QueueGroupService for Background Processing?

The mono framework's `QueueGroupService` pattern provides:

- **Fire-and-Forget Semantics**: Submit jobs without waiting for responses
- **Load Balancing**: Framework automatically distributes jobs across workers
- **Embedded NATS**: No external message broker required
- **Typed contracts**: the worker's services are declared once in `worker/contract` and shared by provider and consumer
- **Simple Module Dependencies**: Declared via `DependentModule` and `ConsumeServices()`

## Architecture Overview

```
┌──────────────┐  create-job / get-job /   ┌──────────────────────────────────────┐
│              │  list-jobs (request-reply) │            Worker Module             │
│  REST API    ├───────────────────────────►│  owns the job store (in-memory)      │
│  (no state)  │      worker/contract       │                                      │
│              │                            │  create-job ──► process-job (QG) ──► │
└──────────────┘                            │    email-worker                      │
                                            │    image-processing-worker           │
                                            │    report-generation-worker          │
                                            └──────────────────────────────────────┘
```

### Components

1. **REST API**: Accepts HTTP job requests and calls the worker module through
   the `worker/contract` request-reply contracts (`CreateJob`, `GetJob`,
   `ListJobs`). It holds no job state.
2. **Worker Module**: The single owner of job state. `create-job` validates and
   records a job, then enqueues it on its `process-job` QueueGroupService, which
   has 3 QueueGroups, each handling a specific job type:
   - `email-worker` - handles email jobs
   - `image-processing-worker` - handles image processing jobs
   - `report-generation-worker` - handles report generation jobs
3. **Job Store**: In-memory status tracking, private to the worker module.

## QueueGroupService Pattern

Every worker service is declared once, as a typed contract in the worker
module's `contract` package. The worker registers through it, the API calls
through it; neither side writes a service name or does JSON by hand.

### Contract (Published Port)

```go
// modules/worker/contract/contract.go (DTOs abridged)
var CreateJob  = helper.RequestReplyService[CreateJobRequest, CreateJobResponse]("worker", "create-job") // services.worker.create-job
var GetJob     = helper.RequestReplyService[GetJobRequest, GetJobResponse]("worker", "get-job")             // services.worker.get-job
var ListJobs   = helper.RequestReplyService[ListJobsRequest, ListJobsResponse]("worker", "list-jobs")       // services.worker.list-jobs
var ProcessJob = helper.QueueGroupService[JobMessage]("worker", "process-job")                              // services.worker.process-job
```

`CreateJobResponse` reports a rejected job (invalid type or payload, queue
unavailable) with an `ErrorCode`, as a response rather than a transport error,
so the API can map it to the right HTTP status.

### Worker Module (Service Provider)

This demo registers **3 QueueGroups on the same service**, each handling a specific job type:

```go
func (m *Module) RegisterServices(container mono.ServiceContainer) error {
    m.processor = NewProcessor()
    m.container = container // own container: create-job enqueues on process-job
    // contract.CreateJob / GetJob / ListJobs .Register(container, ...) likewise
    return contract.ProcessJob.Register(container,
        types.TypedQGHP[contract.JobMessage]{QueueGroup: "email-worker", Handler: m.handleJobTypeEmail},
        types.TypedQGHP[contract.JobMessage]{QueueGroup: "image-processing-worker", Handler: m.handleJobTypeImageProcessing},
        types.TypedQGHP[contract.JobMessage]{QueueGroup: "report-generation-worker", Handler: m.handleJobTypeReportGeneration},
    )
}

// create-job: validate, record, then enqueue (fire-and-forget once accepted)
if err := contract.ProcessJob.Send(ctx, m.container, msg); err != nil {
    _ = m.jobStore.UpdateStatus(j.ID, job.JobStatusFailed)
    return contract.CreateJobResponse{ErrorCode: contract.ErrorCodeQueueUnavailable, Error: job.ErrQueueUnavailable.Error()}, nil
}

// Each queue group handler receives the decoded message and filters for its job type
func (m *Module) handleJobTypeEmail(ctx context.Context, msg contract.JobMessage, _ *mono.Msg) error {
    return m.handleJobOfType(ctx, msg, job.JobTypeEmail, QueueGroupEmail) // ignores other types
}
```

### API Module (Service Consumer)

The API owns a small port and an adapter that implements it over the contracts:

```go
// modules/api/ports.go
type JobsPort interface {
    CreateJob(ctx context.Context, req workercontract.CreateJobRequest) (workercontract.CreateJobResponse, error)
    GetJob(ctx context.Context, req workercontract.GetJobRequest) (workercontract.GetJobResponse, error)
    ListJobs(ctx context.Context, req workercontract.ListJobsRequest) (workercontract.ListJobsResponse, error)
}

// modules/api/adapter.go
func (a workerAdapter) CreateJob(ctx context.Context, req workercontract.CreateJobRequest) (workercontract.CreateJobResponse, error) {
    return workercontract.CreateJob.Call(ctx, a.container, req)
}

// modules/api/module.go
func (m *Module) Dependencies() []string { return []string{"worker"} }

func (m *Module) ConsumeServices() []types.BaseServiceDefinition {
    return []types.BaseServiceDefinition{
        workercontract.CreateJob.ToBase(), workercontract.GetJob.ToBase(), workercontract.ListJobs.ToBase(),
    }
}

func (m *Module) SetDependencyServiceContainer(module string, container mono.ServiceContainer) {
    if module == "worker" {
        m.jobs = workerAdapter{container: container}
    }
}
```

## Module Boundaries

- **Contract package**: `modules/worker/contract` is the only package of the
  worker module the API imports. It holds the job DTOs and the `CreateJob`,
  `GetJob`, `ListJobs` and `ProcessJob` contracts.
- **Single owner of state**: the job store lives inside the worker module;
  the API reads and creates jobs only through the contracts. No state is
  shared between modules.
- **`ConsumeServices()`**: the API declares the three request-reply contracts
  it calls; startup fails fast if the worker does not provide one of them.
- **Strict mode**: `main.go` enables `mono.WithStrictModuleBoundaries()`, so a
  boundary violation (registering on a dependency's container, calling an
  undeclared module's `services.*` subject) is rejected.
- **Conformance tests**: `modules/*/module_test.go` run `monotest.CheckModule`
  (no NATS): the worker test calls all four worker contracts and waits for one
  job of each type to finish; the API test drives every HTTP route against a
  stand-in worker registered through the same contracts. `boundaries_test.go`
  starts the whole app and asserts `app.ModuleBoundaries()` has no violations
  and no unused dependencies.

```bash
go test ./modules/... -run Conformance -v
```

## Project Structure

```
background-jobs-demo/
├── domain/
│   └── job/
│       ├── types.go       # Job types, statuses, and payloads (used by the worker)
│       ├── errors.go      # Domain errors
│       └── store.go       # In-memory job storage (owned by the worker module)
├── modules/
│   ├── worker/
│   │   ├── contract/
│   │   │   └── contract.go # Job DTOs, create/get/list-job and process-job contracts
│   │   ├── jobs.go        # create-job, get-job, list-jobs handlers + validation
│   │   ├── processor.go   # Job processing logic
│   │   ├── processor_test.go
│   │   ├── module.go      # Job state owner + QueueGroupService provider
│   │   └── module_test.go # monotest.CheckModule conformance + validation tests
│   └── api/
│       ├── ports.go       # JobsPort (driven port owned by api)
│       ├── adapter.go     # workerAdapter: JobsPort over worker/contract
│       ├── service.go     # Job service layer (maps worker responses to errors)
│       ├── errors.go      # Service errors mapped to HTTP status codes
│       ├── service_test.go # Service tests against a monotest stand-in worker
│       ├── handlers.go    # HTTP handlers
│       ├── module.go      # API module (depends on worker, ConsumeServices)
│       └── module_test.go # monotest.CheckModule conformance test
├── boundaries_test.go     # Whole-app boundary report test (in-process NATS)
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

- Go 1.25+
- Python 3.7+ (for demo script)

### Run the Application

```bash
go build -o bin/background-jobs-demo .
./bin/background-jobs-demo
```

The application will:
- Start with embedded NATS (no external setup needed)
- Register the worker module (job state owner) with its `create-job`, `get-job`, `list-jobs` services and the `process-job` QueueGroupService
- Enable strict module boundaries
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

### Unit and Conformance Tests

```bash
go test ./... -v
```

Covers the processor, the API service (against a `monotest` stand-in worker,
no hand-written container mocks), one `monotest.CheckModule` conformance test
per module, and the whole-app boundary report.

## Key Differences from Complex Implementation

| Aspect | Complex Version | This Version |
|--------|-----------------|--------------|
| Message Broker | External NATS JetStream | Embedded NATS |
| Worker Pool | Manual goroutine pool | Framework-managed |
| Retry Logic | Exponential backoff, DLQ | Fire-and-forget |
| Event Bus | Custom in-memory pub/sub | Removed |
| Setup | docker-compose required | Just `go build` and run |
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
- Easier testing with `monotest` stand-in containers

## Key Takeaways

1. **QueueGroupService** provides fire-and-forget messaging with automatic load balancing
2. **DependentModule** interface enables explicit dependency declaration
3. **Embedded NATS** eliminates external infrastructure requirements
4. **Typed contracts** (`helper.QueueGroupService`) keep the service name, subject and encoding in one place
5. **Simpler code** is easier to understand, test, and maintain

## Further Reading

- [Mono Framework Documentation](https://github.com/go-monolith/mono)
- [NATS Queue Groups](https://docs.nats.io/nats-concepts/core-nats/queue)
