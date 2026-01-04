# File Upload Demo

A demonstration of file upload/download functionality using the Mono framework with the built-in `fs-jetstream` plugin and Gin HTTP framework.

## Overview

This recipe demonstrates:

- **fs-jetstream Plugin**: Using the built-in file storage plugin for persistent file storage
- **UsePluginModule Interface**: How modules receive plugin instances from the framework
- **Gin HTTP Framework**: Building REST APIs with the Gin web framework
- **Embedded JetStream**: No external NATS server required - uses Mono's embedded NATS with JetStream
- **File Operations**: Upload, download, list, and delete files via REST API

## Why Use fs-jetstream for File Storage?

### Benefits Over Local Filesystem

1. **Distributed Storage**: Files are stored in JetStream object store, which can be replicated across nodes for high availability
2. **Automatic Persistence**: JetStream handles durable storage with configurable retention policies
3. **Compression Support**: Built-in compression reduces storage costs for compressible files
4. **Metadata Headers**: Attach custom metadata (content-type, upload time, etc.) to each file
5. **No External Dependencies**: Uses Mono's embedded NATS server - no external infrastructure needed
6. **Streaming API**: Efficient handling of large files without loading entire content into memory

### When to Use fs-jetstream

- Document storage (PDFs, reports, contracts)
- Media files (images, videos, audio)
- User-uploaded content
- Temporary file uploads with TTL-based expiration
- Any scenario requiring distributed file access

### When NOT to Use fs-jetstream

- High-frequency, small key-value data (use `kv-jetstream` instead)
- Static assets that need CDN distribution
- Files requiring complex querying (use a database with blob storage)

## How UsePluginModule Interface Works

The `UsePluginModule` interface allows modules to receive plugin instances injected by the framework:

```go
// Module declares it uses plugins by implementing UsePluginModule
type Module struct {
    storage *fsjetstream.PluginModule
}

var _ mono.UsePluginModule = (*Module)(nil)

// SetPlugin is called by the framework before Start()
func (m *Module) SetPlugin(alias string, plugin mono.PluginModule) {
    if alias == "storage" {
        m.storage = plugin.(*fsjetstream.PluginModule)
    }
}

func (m *Module) Start(ctx context.Context) error {
    // Access the bucket in Start() after plugin is injected
    m.bucket = m.storage.Bucket("files")
    return nil
}
```

The framework lifecycle:
1. Plugins start first (before regular modules)
2. `SetPlugin()` called on modules implementing `UsePluginModule`
3. Regular modules start in dependency order
4. On shutdown, regular modules stop first, then plugins

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Request                          │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Gin HTTP Server (httpserver module)          │
│  Routes: /api/v1/files, /health                                 │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    File Service (fileservice module)            │
│              Implements UsePluginModule                         │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    fs-jetstream Plugin                          │
│                    (Embedded JetStream)                         │
│                                                                 │
│  - Put/Get files with metadata                                  │
│  - Streaming for large files                                    │
│  - Compression support                                          │
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
file-upload-demo/
├── main.go                    # Application entry point
├── go.mod                     # Go module definition
├── demo.sh                    # Interactive demo script
├── README.md                  # This file
└── modules/
    ├── fileservice/
    │   ├── module.go          # File service mono module (UsePluginModule)
    │   ├── service.go         # File storage service implementation
    │   └── types.go           # Data types and DTOs
    └── httpserver/
        ├── module.go          # Gin HTTP server module
        └── handlers.go        # HTTP request handlers
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/files` | Upload a single file |
| POST | `/api/v1/files/batch` | Upload multiple files |
| GET | `/api/v1/files` | List all files |
| GET | `/api/v1/files/:id` | Download a file |
| GET | `/api/v1/files/:id/info` | Get file metadata |
| DELETE | `/api/v1/files/:id` | Delete a file |

### Response Format

Upload response:
```json
{
  "file": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "document.pdf",
    "size": 1048576,
    "content_type": "application/pdf",
    "digest": "SHA-256=abc123...",
    "created_at": "2024-01-15T10:30:00Z"
  },
  "message": "File uploaded successfully",
  "duration_ms": 45
}
```

List response:
```json
{
  "files": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "document.pdf",
      "size": 1048576,
      "content_type": "application/pdf"
    }
  ],
  "total": 1,
  "duration_ms": 5
}
```

## Prerequisites

- Go 1.21 or later
- curl (for testing)

No external services required! The demo uses Mono's embedded NATS with JetStream.

## Quick Start

1. **Run the application**:
   ```bash
   go run main.go
   ```

2. **Run the demo** (in another terminal):
   ```bash
   ./demo.sh
   ```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `3000` | HTTP server port |
| `MAX_UPLOAD_SIZE` | `104857600` | Max upload size in bytes (100MB) |
| `STORAGE_PATH` | `/tmp/file-upload-demo` | JetStream storage directory |

Example:
```bash
HTTP_PORT=8080 MAX_UPLOAD_SIZE=52428800 go run main.go
```

## Example Usage

### Upload a File
```bash
curl -X POST http://localhost:3000/api/v1/files \
  -F "file=@document.pdf"
```

### Upload Multiple Files
```bash
curl -X POST http://localhost:3000/api/v1/files/batch \
  -F "files=@file1.txt" \
  -F "files=@file2.txt"
```

### List Files
```bash
curl http://localhost:3000/api/v1/files
```

### Download a File
```bash
curl -O -J http://localhost:3000/api/v1/files/{file-id}
```

### Get File Metadata
```bash
curl http://localhost:3000/api/v1/files/{file-id}/info
```

### Delete a File
```bash
curl -X DELETE http://localhost:3000/api/v1/files/{file-id}
```

## Key Implementation Details

### File Storage Strategy

- Files are stored with a unique UUID prefix: `{uuid}/{filename}`
- This allows multiple files with the same name
- Original filename is preserved in metadata headers

### Content Type Detection

- Content type is detected from the file extension
- Can be overridden by the client via multipart header
- Falls back to `application/octet-stream` for unknown types

### Streaming Support

For large files, the service supports streaming:
```go
// Upload large file
result, err := service.UploadFileStream(filename, reader, contentType)

// Download large file
reader, info, err := service.GetFileStream(fileID)
defer reader.Close()
```

### Error Handling

- File not found returns 404
- Invalid requests return 400 with details
- Server errors return 500 with error message

## Mono Framework Features Used

- **Plugin System**: Using `fs-jetstream` plugin for file storage
- **UsePluginModule**: Receiving plugin instance via dependency injection
- **Module Lifecycle**: Proper Start/Stop for resource management
- **Embedded NATS**: No external infrastructure needed
- **Graceful Shutdown**: Clean shutdown with resource cleanup

## Scalability Considerations

1. **Horizontal Scaling**: Multiple instances can share the same JetStream cluster
2. **Storage Limits**: Configure `MaxBytes` per bucket to control storage usage
3. **Memory vs File Storage**: Choose storage type based on data persistence needs
4. **Compression**: Enable compression for text-based files to reduce storage

## License

MIT
