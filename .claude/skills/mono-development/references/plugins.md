# Plugin Module Patterns

Detailed reference for implementing and using plugins in the Mono Framework.

## What is a Plugin?

Plugins are specialized modules that extend the framework with infrastructure services. Unlike regular modules, plugins:

- **Start first, stop last**: Ensures services are available to all modules
- **Receive ServiceContainer**: Manage internal services
- **Are excluded from dependency graph**: Modules don't declare plugins as dependencies
- **Are injected via SetPlugin**: Modules receive plugins by alias

## Plugin Lifecycle

```
1. Framework Start
   │
   ├─→ NATS initializes
   │
   ├─→ Plugins start (in registration order)
   │   ├─ SetContainer() called
   │   └─ Start() called
   │
   ├─→ SetPlugin() called on UsePluginModule modules
   │
   ├─→ Middleware starts
   │
   └─→ Regular modules start (dependency order)

2. Framework Stop
   │
   ├─→ Regular modules stop (reverse dependency order)
   │
   ├─→ Middleware stops
   │
   └─→ Plugins stop (reverse registration order)
```

## Plugin Interfaces

### PluginModule Interface

Every plugin implements:

```go
type PluginModule interface {
    Module  // Name(), Start(), Stop()

    SetContainer(container ServiceContainer)
    Container() ServiceContainer
}
```

### UsePluginModule Interface

Modules that consume plugins implement:

```go
type UsePluginModule interface {
    Module

    SetPlugin(alias string, plugin PluginModule)
}
```

## Creating a Custom Plugin

### Step 1: Define Structure

```go
package myplugin

import (
    "context"
    "github.com/go-monolith/mono/pkg/types"
)

type PluginModule struct {
    name      string
    container types.ServiceContainer
    config    Config
    // Your resources
    conn      *Connection
}

type Config struct {
    Endpoint   string
    MaxConns   int
    Timeout    time.Duration
}

func New(config Config) (*PluginModule, error) {
    if config.Endpoint == "" {
        return nil, fmt.Errorf("endpoint required")
    }
    return &PluginModule{
        name:   "my-plugin",
        config: config,
    }, nil
}
```

### Step 2: Implement Module Interface

```go
func (p *PluginModule) Name() string {
    return p.name
}

func (p *PluginModule) Start(ctx context.Context) error {
    // Initialize resources
    conn, err := Connect(ctx, p.config.Endpoint)
    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }
    p.conn = conn
    return nil
}

func (p *PluginModule) Stop(ctx context.Context) error {
    if p.conn != nil {
        return p.conn.Close(ctx)
    }
    return nil
}
```

### Step 3: Implement PluginModule Interface

```go
func (p *PluginModule) SetContainer(container types.ServiceContainer) {
    p.container = container
}

func (p *PluginModule) Container() types.ServiceContainer {
    return p.container
}
```

### Step 4: Define Public API (Port)

```go
// Public interface for consumers
type DataPort interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte) error
    Delete(ctx context.Context, key string) error
}

// Expose the port
func (p *PluginModule) Port() DataPort {
    return &adapter{conn: p.conn}
}

type adapter struct {
    conn *Connection
}

func (a *adapter) Get(ctx context.Context, key string) ([]byte, error) {
    return a.conn.Fetch(ctx, key)
}

func (a *adapter) Set(ctx context.Context, key string, value []byte) error {
    return a.conn.Store(ctx, key, value)
}

func (a *adapter) Delete(ctx context.Context, key string) error {
    return a.conn.Remove(ctx, key)
}
```

### Step 5: Register and Use

```go
// In main.go
func main() {
    app, _ := mono.NewMonoApplication()

    plugin, _ := myplugin.New(myplugin.Config{
        Endpoint: "localhost:9000",
        MaxConns: 10,
    })
    app.RegisterPlugin(plugin, "my-plugin")

    app.Register(&MyModule{})
    app.Start(context.Background())
}

// In module
type MyModule struct {
    plugin *myplugin.PluginModule
    data   myplugin.DataPort
}

func (m *MyModule) SetPlugin(alias string, plugin mono.PluginModule) {
    if alias == "my-plugin" {
        m.plugin = plugin.(*myplugin.PluginModule)
    }
}

func (m *MyModule) Start(ctx context.Context) error {
    if m.plugin == nil {
        return fmt.Errorf("required plugin not registered")
    }
    m.data = m.plugin.Port()
    return nil
}
```

## Built-in Plugins

### kv-jetstream (Key-Value Storage)

Fast, distributed key-value storage using NATS JetStream KV Store.

**Use Cases:**
- Caching
- Sessions
- Configuration
- Application state
- Distributed locks

**Key Features:**
- Revision-based optimistic locking
- Watch API for real-time notifications
- TTL support for auto-expiration
- Multiple storage backends (memory/disk)

**Interface: `KVStoragePort`**

```go
type KVStoragePort interface {
    // Basic operations
    Get(key string) ([]byte, error)
    Set(key string, val []byte, exp time.Duration) error
    Delete(key string) error

    // Revision-based operations (optimistic locking)
    Create(key string, val []byte, exp time.Duration) (uint64, error)
    Update(key string, val []byte, exp time.Duration, revision uint64) (uint64, error)
    GetEntry(key string) (*Entry, error)
    PutWithRevision(key string, val []byte, exp time.Duration) (uint64, error)
    Purge(key string) error

    // Enumeration
    Keys() ([]string, error)

    // Real-time notifications
    Watch(pattern string, opts ...WatchOption) (KeyWatcher, error)
    WatchAll(ctx context.Context, opts ...WatchOption) (KeyWatcher, error)

    // Status
    Status() (*BucketStatus, error)
}
```

**Usage:**

```go
// Create plugin
kv, _ := kvjetstream.New(kvjetstream.Config{
    Buckets: []kvjetstream.BucketConfig{
        {
            Name:    "cache",
            TTL:     time.Hour,
            Storage: kvjetstream.MemoryStorage,
        },
        {
            Name:     "sessions",
            MaxBytes: 100 * 1024 * 1024, // 100MB
        },
    },
})
app.RegisterPlugin(kv, "kv")

// In module
cache := m.kv.Bucket("cache")
cache.Set("key", []byte("value"), time.Hour)

// Optimistic locking
entry, _ := cache.GetEntry("counter")
newVal := incrementCounter(entry.Value)
_, err := cache.Update("counter", newVal, 0, entry.Revision)
if errors.Is(err, kvjetstream.ErrRevisionMismatch) {
    // Concurrent modification, retry
}
```

### fs-jetstream (File Storage)

Persistent file/object storage using NATS JetStream ObjectStore.

**Use Cases:**
- Documents and PDFs
- Media files (images, videos)
- Large binary objects
- Temporary file uploads
- Backup storage

**Key Features:**
- Streaming API for large files
- Custom metadata headers
- TTL-based expiration
- Compression support
- Object listing with prefix filtering

**Interface: `FileStoragePort`**

```go
type FileStoragePort interface {
    // Basic operations
    Get(key string) ([]byte, error)
    Set(key string, val []byte, exp time.Duration) error
    Delete(key string) error

    // File-specific (returns ObjectInfo)
    Put(ctx context.Context, key string, data []byte, opts ...PutOption) (*ObjectInfo, error)

    // Streaming for large files
    GetReader(key string) (io.ReadCloser, *ObjectInfo, error)
    PutReader(key string, reader io.Reader, exp time.Duration, opts ...PutOption) (*ObjectInfo, error)

    // Listing and metadata
    List(opts ...ListOption) ([]ObjectInfo, error)
    Stat(key string) (*ObjectInfo, error)
}
```

**Usage:**

```go
// Create plugin
storage, _ := fsjetstream.New(fsjetstream.Config{
    Buckets: []fsjetstream.BucketConfig{
        {
            Name:        "documents",
            MaxBytes:    1_000_000_000, // 1GB
            Compression: true,
        },
        {
            Name:    "uploads",
            TTL:     24 * time.Hour,
            Storage: fsjetstream.MemoryStorage,
        },
    },
})
app.RegisterPlugin(storage, "storage")

// In module
docs := m.storage.Bucket("documents")

// Store file
info, _ := docs.Put(ctx, "report.pdf", pdfData,
    fsjetstream.WithDescription("Q4 Report"),
    fsjetstream.WithHeaders(map[string]string{
        "Content-Type": "application/pdf",
    }),
)

// Stream large file
file, _ := os.Open("large-video.mp4")
info, _ = docs.PutReader("video.mp4", file, 0)

// Read with streaming
reader, info, _ := docs.GetReader("video.mp4")
defer reader.Close()
io.Copy(outputFile, reader)

// List files
files, _ := docs.List(fsjetstream.WithPrefix("reports/"))
```

## kv-jetstream vs fs-jetstream

| Aspect | kv-jetstream | fs-jetstream |
|--------|-------------|--------------|
| **Data Model** | Key-value pairs | Binary objects |
| **Max Value Size** | ~1MB recommended | Unlimited |
| **Concurrency** | Revision-based locking | None |
| **Notifications** | Watch API | Not supported |
| **Streaming** | No | Yes (Reader/Writer) |
| **Use Case** | Cache, sessions, config | Files, documents, media |

**Decision Guide:**

```
Is the data small (<1MB)?
├── YES → Is real-time notification needed?
│   ├── YES → kv-jetstream (Watch API)
│   └── NO → kv-jetstream (simpler)
│
└── NO (large files) → fs-jetstream (streaming)
```

## Multiple Plugin Instances

Register same plugin type with different configurations:

```go
// Primary and backup storage
primary, _ := fsjetstream.New(primaryConfig)
backup, _ := fsjetstream.New(backupConfig)

app.RegisterPlugin(primary, "primary-storage")
app.RegisterPlugin(backup, "backup-storage")

// In module
func (m *Module) SetPlugin(alias string, plugin mono.PluginModule) {
    switch alias {
    case "primary-storage":
        m.primary = plugin.(*fsjetstream.PluginModule)
    case "backup-storage":
        m.backup = plugin.(*fsjetstream.PluginModule)
    }
}
```

## Error Handling

### Sentinel Errors (kv-jetstream)

```go
var (
    ErrKeyNotFound      = errors.New("key not found")
    ErrKeyExists        = errors.New("key already exists")
    ErrRevisionMismatch = errors.New("revision mismatch")
    ErrBucketNotFound   = errors.New("bucket not found")
)

// Usage
_, err := bucket.Create("lock", data, 0)
if errors.Is(err, kvjetstream.ErrKeyExists) {
    // Key already exists
}

_, err = bucket.Update("key", data, 0, revision)
if errors.Is(err, kvjetstream.ErrRevisionMismatch) {
    // Concurrent modification
}
```

### Plugin Not Found

```go
func (m *Module) Start(ctx context.Context) error {
    if m.plugin == nil {
        return fmt.Errorf("required plugin 'storage' not registered")
    }
    return nil
}
```

### Bucket Not Found

```go
bucket := m.storage.Bucket("documents")
if bucket == nil {
    return fmt.Errorf("bucket 'documents' not configured")
}
```

## Best Practices

### Do

- Register plugins before regular modules
- Check for nil plugin references in Start()
- Use meaningful aliases for plugins
- Use appropriate plugin for data size
- Handle sentinel errors properly
- Set TTL for temporary data

### Don't

- Implement business logic in plugins
- Store large values in kv-jetstream
- Forget to close readers from GetReader()
- Ignore revision mismatch errors
- Mix different data types in one bucket
- Register plugins after app.Start()

## Complete Plugin Example

```go
package cacheplugin

import (
    "context"
    "sync"
    "time"
    "github.com/go-monolith/mono/pkg/types"
)

// Plugin implementation
type PluginModule struct {
    name      string
    container types.ServiceContainer
    cache     map[string]*cacheEntry
    mu        sync.RWMutex
    ttl       time.Duration
}

type cacheEntry struct {
    value     []byte
    expiresAt time.Time
}

type Config struct {
    DefaultTTL time.Duration
}

// Compile-time check
var _ mono.PluginModule = (*PluginModule)(nil)

func New(config Config) (*PluginModule, error) {
    ttl := config.DefaultTTL
    if ttl == 0 {
        ttl = time.Hour
    }
    return &PluginModule{
        name:  "cache",
        cache: make(map[string]*cacheEntry),
        ttl:   ttl,
    }, nil
}

func (p *PluginModule) Name() string { return p.name }

func (p *PluginModule) SetContainer(container types.ServiceContainer) {
    p.container = container
}

func (p *PluginModule) Container() types.ServiceContainer {
    return p.container
}

func (p *PluginModule) Start(ctx context.Context) error {
    // Start cleanup goroutine
    go p.cleanupLoop(ctx)
    return nil
}

func (p *PluginModule) Stop(ctx context.Context) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.cache = nil
    return nil
}

func (p *PluginModule) cleanupLoop(ctx context.Context) {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            p.cleanup()
        }
    }
}

func (p *PluginModule) cleanup() {
    p.mu.Lock()
    defer p.mu.Unlock()
    now := time.Now()
    for key, entry := range p.cache {
        if now.After(entry.expiresAt) {
            delete(p.cache, key)
        }
    }
}

// Public API
type CachePort interface {
    Get(key string) ([]byte, bool)
    Set(key string, value []byte, ttl time.Duration)
    Delete(key string)
}

func (p *PluginModule) Port() CachePort {
    return p
}

func (p *PluginModule) Get(key string) ([]byte, bool) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    entry, ok := p.cache[key]
    if !ok || time.Now().After(entry.expiresAt) {
        return nil, false
    }
    return entry.value, true
}

func (p *PluginModule) Set(key string, value []byte, ttl time.Duration) {
    if ttl == 0 {
        ttl = p.ttl
    }
    p.mu.Lock()
    defer p.mu.Unlock()
    p.cache[key] = &cacheEntry{
        value:     value,
        expiresAt: time.Now().Add(ttl),
    }
}

func (p *PluginModule) Delete(key string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    delete(p.cache, key)
}
```
