# File Upload Demo

A file upload service demonstrating the integration of [Gin](https://github.com/gin-gonic/gin) HTTP framework with [NATS JetStream Object Store](https://docs.nats.io/nats-concepts/jetstream/obj_store) for distributed file storage using the [mono](https://github.com/go-monolith/mono) framework.

## Why Gin + JetStream Object Store?

### Gin HTTP Framework

Gin is one of the most popular Go web frameworks, offering:

- **High Performance**: Radix tree-based routing, zero allocation router
- **Middleware Support**: Easy-to-use middleware chain for logging, recovery, CORS, etc.
- **JSON Validation**: Built-in request binding and validation
- **Error Management**: Convenient error handling with custom error types
- **Wide Adoption**: Large community, extensive documentation, and ecosystem

### JetStream Object Store

NATS JetStream Object Store provides:

- **Distributed Storage**: Files are stored in NATS, enabling multi-node access
- **Streaming Semantics**: Large files are chunked and streamed efficiently
- **Persistence**: Data survives restarts with configurable retention policies
- **Simple Operations**: Put, Get, Delete, List - no complex APIs
- **Metadata Support**: Store content-type and custom headers with files
- **Built-in Replication**: Data can be replicated across NATS cluster nodes

## When to Use This Pattern

### Good Use Cases

- **Microservices File Sharing**: Multiple services need access to the same files
- **Temporary File Storage**: Processing pipelines, job queues, caches
- **Development/Testing**: Quick setup without external dependencies
- **Small to Medium Files**: Files up to a few hundred MB
- **Event-Driven Architectures**: When you're already using NATS

### Consider Alternatives When

- **Very Large Files (>1GB)**: Consider S3, MinIO, or dedicated object storage
- **CDN Requirements**: Use S3 + CloudFront for global distribution
- **Complex Access Control**: Cloud providers offer better IAM integration
- **Long-term Archival**: Glacier, Azure Blob Archive for cold storage

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Clients                            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    API Module (Gin)                          │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌──────────┐ │
│  │  Upload   │  │   List    │  │    Get    │  │  Delete  │ │
│  │  Handler  │  │  Handler  │  │  Handler  │  │  Handler │ │
│  └─────┬─────┘  └─────┬─────┘  └─────┬─────┘  └────┬─────┘ │
│        └──────────────┴───────────────┴─────────────┘       │
│                              │                               │
│                     FilesAdapter (Port)                      │
└──────────────────────────────┼──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                   Files Module (Core Domain)                 │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    Service                           │    │
│  │  ┌──────────┐  ┌──────────┐  ┌────────┐  ┌────────┐ │    │
│  │  │  Upload  │  │   Get    │  │  List  │  │ Delete │ │    │
│  │  └────┬─────┘  └────┬─────┘  └───┬────┘  └───┬────┘ │    │
│  │       └─────────────┴────────────┴───────────┘      │    │
│  └──────────────────────────┬──────────────────────────┘    │
│                             │                                │
│              JetStreamObjectStore (Adapter)                  │
└─────────────────────────────┼───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  NATS JetStream Object Store                 │
│                                                              │
│    ┌─────────────────────────────────────────────────┐      │
│    │              Object Store Bucket                 │      │
│    │  ┌──────────┐ ┌──────────┐ ┌──────────┐        │      │
│    │  │  file-1  │ │  file-2  │ │  file-3  │  ...   │      │
│    │  │ metadata │ │ metadata │ │ metadata │        │      │
│    │  │  chunks  │ │  chunks  │ │  chunks  │        │      │
│    │  └──────────┘ └──────────┘ └──────────┘        │      │
│    └─────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose (for NATS)
- curl and jq (for demo script)

## Quick Start

1. **Start NATS JetStream**:
   ```bash
   docker-compose up -d
   ```

2. **Run the application**:
   ```bash
   go run .
   ```

3. **Run the demo**:
   ```bash
   ./demo.sh
   ```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/v1/files` | Upload a file |
| `GET` | `/api/v1/files` | List all files |
| `GET` | `/api/v1/files/:id` | Get file metadata |
| `GET` | `/api/v1/files/:id/download` | Download file content |
| `DELETE` | `/api/v1/files/:id` | Delete a file |

## Usage Examples

### Upload a file (multipart/form-data)

```bash
curl -X POST -F "file=@myfile.txt" http://localhost:3000/api/v1/files
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "myfile.txt",
  "size": 1234,
  "content_type": "text/plain",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Upload a file (JSON with base64)

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"data.json","data":"eyJrZXkiOiJ2YWx1ZSJ9","content_type":"application/json"}' \
  http://localhost:3000/api/v1/files
```

### List files with pagination

```bash
curl "http://localhost:3000/api/v1/files?limit=10&offset=0"
```

Response:
```json
{
  "files": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "myfile.txt",
      "size": 1234,
      "content_type": "text/plain",
      "created_at": "2024-01-15T10:30:00Z",
      "download_url": "/api/v1/files/550e8400-e29b-41d4-a716-446655440000/download"
    }
  ],
  "total": 1
}
```

### Download a file

```bash
curl -O http://localhost:3000/api/v1/files/550e8400-e29b-41d4-a716-446655440000/download
```

### Delete a file

```bash
curl -X DELETE http://localhost:3000/api/v1/files/550e8400-e29b-41d4-a716-446655440000
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | HTTP server port |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `NATS_BUCKET` | `files` | Object store bucket name |

## Mono Framework Patterns

This recipe demonstrates:

- **ServiceProviderModule**: Files module exposes `upload-file`, `get-file`, `list-files`, `delete-file` services
- **DependentModule**: API module depends on Files module via `FilesAdapter`
- **HealthCheckableModule**: Both modules report health status
- **Cross-module Communication**: Uses `helper.CallRequestReplyService()` for type-safe service calls

## Trade-offs

### Pros

- **Simple Setup**: Just NATS, no additional storage services
- **Integrated with Events**: Same infrastructure for files and messaging
- **Streaming**: Large files are chunked automatically
- **Multi-tenant Ready**: Use different buckets for isolation

### Cons

- **Not a CDN**: No edge caching or global distribution
- **Memory Consideration**: Files are loaded into memory for processing
- **Scaling Limits**: NATS cluster size determines storage capacity
- **No Pre-signed URLs**: All access goes through the API

## Learn More

- [NATS JetStream Object Store](https://docs.nats.io/nats-concepts/jetstream/obj_store)
- [Gin Web Framework](https://gin-gonic.com/)
- [Mono Framework](https://github.com/go-monolith/mono)
