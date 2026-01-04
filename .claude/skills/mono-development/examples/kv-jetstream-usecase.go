// Example: Using kv-jetstream Plugin
//
// This example demonstrates:
// - Configuring and registering the kv-jetstream plugin
// - Basic CRUD operations (Get, Set, Delete)
// - Optimistic locking with revisions (Create, Update, GetEntry)
// - Watching for real-time changes
// - Session management use case
// - Distributed locking use case

package kvjetstreamusecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-monolith/mono"
	kvjetstream "github.com/go-monolith/mono/plugin/kv-jetstream"
)

// ============================================================
// Domain Types
// ============================================================

type Session struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Counter struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// ============================================================
// Session Module - Using KV for Session Management
// ============================================================

type SessionModule struct {
	kv       *kvjetstream.PluginModule
	sessions kvjetstream.KVStoragePort
}

var (
	_ mono.Module          = (*SessionModule)(nil)
	_ mono.UsePluginModule = (*SessionModule)(nil)
)

func NewSessionModule() *SessionModule {
	return &SessionModule{}
}

func (m *SessionModule) Name() string { return "sessions" }

func (m *SessionModule) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias == "kv" {
		m.kv = plugin.(*kvjetstream.PluginModule)
	}
}

func (m *SessionModule) Start(ctx context.Context) error {
	if m.kv == nil {
		return fmt.Errorf("required plugin 'kv' not registered")
	}

	m.sessions = m.kv.Bucket("sessions")
	if m.sessions == nil {
		return fmt.Errorf("bucket 'sessions' not found")
	}

	slog.Info("Session module started")
	return nil
}

func (m *SessionModule) Stop(ctx context.Context) error {
	slog.Info("Session module stopped")
	return nil
}

// CreateSession creates a new session
func (m *SessionModule) CreateSession(userID, email, role string, ttl time.Duration) (string, error) {
	sessionID := fmt.Sprintf("sess:%d", time.Now().UnixNano())

	session := Session{
		UserID:    userID,
		Email:     email,
		Role:      role,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session: %w", err)
	}

	// Use Set for simple storage
	err = m.sessions.Set(sessionID, data, ttl)
	if err != nil {
		return "", fmt.Errorf("failed to store session: %w", err)
	}

	slog.Info("Session created", "sessionID", sessionID, "userID", userID)
	return sessionID, nil
}

// GetSession retrieves a session
func (m *SessionModule) GetSession(sessionID string) (*Session, error) {
	data, err := m.sessions.Get(sessionID)
	if err != nil {
		if errors.Is(err, kvjetstream.ErrKeyNotFound) {
			return nil, nil // Session not found or expired
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// DeleteSession removes a session
func (m *SessionModule) DeleteSession(sessionID string) error {
	return m.sessions.Delete(sessionID)
}

// ListSessions lists all active session keys
func (m *SessionModule) ListSessions() ([]string, error) {
	return m.sessions.Keys()
}

// ============================================================
// Counter Module - Optimistic Locking Example
// ============================================================

type CounterModule struct {
	kv       *kvjetstream.PluginModule
	counters kvjetstream.KVStoragePort
}

var (
	_ mono.Module          = (*CounterModule)(nil)
	_ mono.UsePluginModule = (*CounterModule)(nil)
)

func NewCounterModule() *CounterModule {
	return &CounterModule{}
}

func (m *CounterModule) Name() string { return "counters" }

func (m *CounterModule) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias == "kv" {
		m.kv = plugin.(*kvjetstream.PluginModule)
	}
}

func (m *CounterModule) Start(ctx context.Context) error {
	if m.kv == nil {
		return fmt.Errorf("required plugin 'kv' not registered")
	}

	m.counters = m.kv.Bucket("counters")
	if m.counters == nil {
		return fmt.Errorf("bucket 'counters' not found")
	}

	slog.Info("Counter module started")
	return nil
}

func (m *CounterModule) Stop(ctx context.Context) error {
	slog.Info("Counter module stopped")
	return nil
}

// CreateCounter creates a new counter (fails if exists)
func (m *CounterModule) CreateCounter(name string, initialValue int) error {
	counter := Counter{Name: name, Value: initialValue}
	data, _ := json.Marshal(counter)

	// Create only succeeds if key doesn't exist
	_, err := m.counters.Create(name, data, 0)
	if errors.Is(err, kvjetstream.ErrKeyExists) {
		return fmt.Errorf("counter '%s' already exists", name)
	}
	return err
}

// IncrementCounter atomically increments a counter using optimistic locking
func (m *CounterModule) IncrementCounter(name string) (int, error) {
	maxRetries := 5

	for i := 0; i < maxRetries; i++ {
		// Get current value with revision
		entry, err := m.counters.GetEntry(name)
		if err != nil {
			return 0, fmt.Errorf("failed to get counter: %w", err)
		}

		var counter Counter
		if err := json.Unmarshal(entry.Value, &counter); err != nil {
			return 0, fmt.Errorf("failed to unmarshal counter: %w", err)
		}

		// Increment
		counter.Value++
		newData, _ := json.Marshal(counter)

		// Update with revision check (optimistic locking)
		_, err = m.counters.Update(name, newData, 0, entry.Revision)
		if err == nil {
			return counter.Value, nil
		}

		if errors.Is(err, kvjetstream.ErrRevisionMismatch) {
			// Concurrent modification, retry
			slog.Debug("Revision mismatch, retrying", "attempt", i+1)
			continue
		}

		return 0, fmt.Errorf("failed to update counter: %w", err)
	}

	return 0, fmt.Errorf("max retries exceeded for counter increment")
}

// GetCounter gets the current counter value
func (m *CounterModule) GetCounter(name string) (int, error) {
	data, err := m.counters.Get(name)
	if err != nil {
		return 0, err
	}

	var counter Counter
	if err := json.Unmarshal(data, &counter); err != nil {
		return 0, err
	}

	return counter.Value, nil
}

// ============================================================
// Lock Module - Distributed Locking Example
// ============================================================

type LockModule struct {
	kv    *kvjetstream.PluginModule
	locks kvjetstream.KVStoragePort
}

var (
	_ mono.Module          = (*LockModule)(nil)
	_ mono.UsePluginModule = (*LockModule)(nil)
)

func NewLockModule() *LockModule {
	return &LockModule{}
}

func (m *LockModule) Name() string { return "locks" }

func (m *LockModule) SetPlugin(alias string, plugin mono.PluginModule) {
	if alias == "kv" {
		m.kv = plugin.(*kvjetstream.PluginModule)
	}
}

func (m *LockModule) Start(ctx context.Context) error {
	if m.kv == nil {
		return fmt.Errorf("required plugin 'kv' not registered")
	}

	m.locks = m.kv.Bucket("locks")
	if m.locks == nil {
		return fmt.Errorf("bucket 'locks' not found")
	}

	slog.Info("Lock module started")
	return nil
}

func (m *LockModule) Stop(ctx context.Context) error {
	slog.Info("Lock module stopped")
	return nil
}

// AcquireLock tries to acquire a distributed lock
func (m *LockModule) AcquireLock(resource, owner string, ttl time.Duration) (bool, error) {
	key := "lock:" + resource
	data := []byte(owner)

	// Create only succeeds if key doesn't exist
	_, err := m.locks.Create(key, data, ttl)
	if err == nil {
		slog.Info("Lock acquired", "resource", resource, "owner", owner)
		return true, nil
	}

	if errors.Is(err, kvjetstream.ErrKeyExists) {
		// Lock already held
		return false, nil
	}

	return false, fmt.Errorf("failed to acquire lock: %w", err)
}

// ReleaseLock releases a distributed lock
func (m *LockModule) ReleaseLock(resource, owner string) error {
	key := "lock:" + resource

	// Check if we own the lock
	data, err := m.locks.Get(key)
	if err != nil {
		if errors.Is(err, kvjetstream.ErrKeyNotFound) {
			return nil // Lock already released
		}
		return err
	}

	if string(data) != owner {
		return fmt.Errorf("cannot release lock: owned by %s, not %s", string(data), owner)
	}

	// Use Purge to completely remove the key
	return m.locks.Purge(key)
}

// ============================================================
// Application Entry Point
// ============================================================

func main() {
	fmt.Println("=== Mono Framework kv-jetstream Plugin Example ===")
	fmt.Println("Demonstrates: Sessions, Counters with optimistic locking, Distributed locks")
	fmt.Println()

	// Create temp directory for JetStream storage
	jsDir := "/tmp/mono-kv-example"

	// Create application
	app, err := mono.NewMonoApplication(
		mono.WithLogLevel(mono.LogLevelInfo),
		mono.WithLogFormat(mono.LogFormatText),
		mono.WithShutdownTimeout(10*time.Second),
		mono.WithJetStreamStorageDir(jsDir),
	)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Create kv-jetstream plugin with multiple buckets
	kvStore, err := kvjetstream.New(kvjetstream.Config{
		Buckets: []kvjetstream.BucketConfig{
			{
				Name:        "sessions",
				Description: "User sessions",
				TTL:         time.Hour, // Sessions expire after 1 hour
				Storage:     kvjetstream.MemoryStorage,
			},
			{
				Name:        "counters",
				Description: "Application counters",
				Storage:     kvjetstream.FileStorage, // Persistent
			},
			{
				Name:        "locks",
				Description: "Distributed locks",
				TTL:         time.Minute, // Locks auto-expire after 1 minute
				Storage:     kvjetstream.MemoryStorage,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create KV plugin: %v", err)
	}

	// Register plugin
	if err := app.RegisterPlugin(kvStore, "kv"); err != nil {
		log.Fatalf("Failed to register plugin: %v", err)
	}
	fmt.Println("KV plugin registered with 3 buckets")

	// Create modules
	sessionModule := NewSessionModule()
	counterModule := NewCounterModule()
	lockModule := NewLockModule()

	// Register modules
	app.Register(sessionModule)
	app.Register(counterModule)
	app.Register(lockModule)

	// Start application
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	fmt.Println("App started successfully")
	fmt.Println()

	// Demo: Session Management
	fmt.Println("=== Session Management Demo ===")
	sessionID, _ := sessionModule.CreateSession("user123", "alice@example.com", "admin", time.Hour)
	fmt.Printf("Created session: %s\n", sessionID)

	session, _ := sessionModule.GetSession(sessionID)
	fmt.Printf("Retrieved session: %+v\n", session)

	sessions, _ := sessionModule.ListSessions()
	fmt.Printf("Active sessions: %v\n", sessions)
	fmt.Println()

	// Demo: Counter with Optimistic Locking
	fmt.Println("=== Counter Demo (Optimistic Locking) ===")
	counterModule.CreateCounter("page-views", 0)
	fmt.Println("Created counter: page-views")

	for i := 0; i < 5; i++ {
		val, _ := counterModule.IncrementCounter("page-views")
		fmt.Printf("Incremented to: %d\n", val)
	}

	finalVal, _ := counterModule.GetCounter("page-views")
	fmt.Printf("Final value: %d\n", finalVal)
	fmt.Println()

	// Demo: Distributed Locking
	fmt.Println("=== Distributed Lock Demo ===")
	acquired, _ := lockModule.AcquireLock("resource-1", "worker-A", time.Minute)
	fmt.Printf("Worker-A acquired lock: %v\n", acquired)

	acquired, _ = lockModule.AcquireLock("resource-1", "worker-B", time.Minute)
	fmt.Printf("Worker-B acquired lock: %v (expected false)\n", acquired)

	lockModule.ReleaseLock("resource-1", "worker-A")
	fmt.Println("Worker-A released lock")

	acquired, _ = lockModule.AcquireLock("resource-1", "worker-B", time.Minute)
	fmt.Printf("Worker-B acquired lock: %v\n", acquired)

	// Wait for shutdown signal
	fmt.Println("\nPress Ctrl+C to shutdown...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	fmt.Println("\nShutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to stop app: %v", err)
	}

	fmt.Println("App stopped successfully")
}
