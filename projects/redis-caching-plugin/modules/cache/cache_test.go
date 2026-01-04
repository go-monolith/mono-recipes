package cache

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gofiber/storage/redis/v3"
)

// TestConfig for unit tests - requires Redis running on localhost:6379
const testRedisAddr = "localhost:6379"

// checkRedisAvailable checks if Redis is reachable before creating storage.
// gofiber/storage/redis panics on connection failure, so we check first.
func checkRedisAvailable(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", testRedisAddr, 2*time.Second)
	if err != nil {
		t.Skipf("Redis not available at %s: %v", testRedisAddr, err)
	}
	conn.Close()
}

// setupTestCacheService creates a CacheService instance for testing.
// Returns the service and a cleanup function.
func setupTestCacheService(t *testing.T, prefix string) (CacheService, func()) {
	t.Helper()
	checkRedisAvailable(t)

	storage := redis.New(redis.Config{
		Host: "localhost",
		Port: 6379,
	})

	svc := NewCacheService(storage, prefix, 5*time.Minute)

	cleanup := func() {
		storage.Reset()
		storage.Close()
	}

	return svc, cleanup
}

func TestNewCacheService(t *testing.T) {
	checkRedisAvailable(t)

	storage := redis.New(redis.Config{
		Host: "localhost",
		Port: 6379,
	})
	defer storage.Close()

	svc := NewCacheService(storage, "test:", 10*time.Minute)

	if svc == nil {
		t.Fatal("NewCacheService() returned nil")
	}
}

func TestCacheService_SetAndGet(t *testing.T) {
	svc, cleanup := setupTestCacheService(t, "test:setget:")
	defer cleanup()

	ctx := context.Background()

	type TestData struct {
		ID    int     `json:"id"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	testCases := []struct {
		name  string
		key   string
		value TestData
	}{
		{
			name:  "simple data",
			key:   "item1",
			value: TestData{ID: 1, Name: "Product A", Price: 99.99},
		},
		{
			name:  "data with special characters",
			key:   "item:2:special",
			value: TestData{ID: 2, Name: "Product B & C", Price: 149.50},
		},
		{
			name:  "data with zero values",
			key:   "item3",
			value: TestData{ID: 0, Name: "", Price: 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the value
			err := svc.Set(ctx, tc.key, tc.value)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			// Get the value
			var result TestData
			found, err := svc.Get(ctx, tc.key, &result)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if !found {
				t.Fatal("Get() returned found = false, want true")
			}

			// Verify the data
			if result.ID != tc.value.ID {
				t.Errorf("ID = %d, want %d", result.ID, tc.value.ID)
			}
			if result.Name != tc.value.Name {
				t.Errorf("Name = %q, want %q", result.Name, tc.value.Name)
			}
			if result.Price != tc.value.Price {
				t.Errorf("Price = %f, want %f", result.Price, tc.value.Price)
			}
		})
	}
}

func TestCacheService_GetMiss(t *testing.T) {
	svc, cleanup := setupTestCacheService(t, "test:miss:")
	defer cleanup()

	ctx := context.Background()

	var result string
	found, err := svc.Get(ctx, "nonexistent", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if found {
		t.Error("Get() returned found = true for nonexistent key, want false")
	}
}

func TestCacheService_SetWithTTL(t *testing.T) {
	svc, cleanup := setupTestCacheService(t, "test:ttl:")
	defer cleanup()

	ctx := context.Background()

	// Set with very short TTL
	err := svc.SetWithTTL(ctx, "expiring", "test value", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("SetWithTTL() error = %v", err)
	}

	// Should exist immediately
	var result string
	found, err := svc.Get(ctx, "expiring", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Fatal("Get() immediately after Set should find the key")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should be expired
	found, err = svc.Get(ctx, "expiring", &result)
	if err != nil {
		t.Fatalf("Get() after expiration error = %v", err)
	}
	if found {
		t.Error("Get() after TTL expiration should return found = false")
	}
}

func TestCacheService_Delete(t *testing.T) {
	svc, cleanup := setupTestCacheService(t, "test:delete:")
	defer cleanup()

	ctx := context.Background()

	// Set a value
	err := svc.Set(ctx, "to-delete", "some value")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify it exists
	var result string
	found, _ := svc.Get(ctx, "to-delete", &result)
	if !found {
		t.Fatal("Key should exist before deletion")
	}

	// Delete it
	err = svc.Delete(ctx, "to-delete")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	found, _ = svc.Get(ctx, "to-delete", &result)
	if found {
		t.Error("Key should not exist after deletion")
	}
}

func TestCacheService_InvalidateAll(t *testing.T) {
	svc, cleanup := setupTestCacheService(t, "test:invalidate:")
	defer cleanup()

	ctx := context.Background()

	// Set multiple values
	for i := 1; i <= 5; i++ {
		key := "item" + string(rune('a'+i-1))
		err := svc.Set(ctx, key, i)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	// Verify values exist
	var result int
	found, _ := svc.Get(ctx, "itema", &result)
	if !found {
		t.Fatal("Keys should exist before invalidation")
	}

	// Invalidate all
	err := svc.InvalidateAll(ctx)
	if err != nil {
		t.Fatalf("InvalidateAll() error = %v", err)
	}

	// Verify all keys are gone
	for i := 1; i <= 5; i++ {
		key := "item" + string(rune('a'+i-1))
		found, _ := svc.Get(ctx, key, &result)
		if found {
			t.Errorf("Key %q should have been invalidated", key)
		}
	}
}

func TestCacheService_KeyPrefix(t *testing.T) {
	checkRedisAvailable(t)

	// Create storage directly for this test
	storage := redis.New(redis.Config{
		Host: "localhost",
		Port: 6379,
	})
	defer func() {
		storage.Reset()
		storage.Close()
	}()

	svc := NewCacheService(storage, "myprefix:", 5*time.Minute)
	ctx := context.Background()

	// Set a value
	err := svc.Set(ctx, "mykey", "myvalue")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify the key is stored with prefix using the underlying storage
	result, err := storage.Get("myprefix:mykey")
	if err != nil {
		t.Fatalf("Direct storage Get error = %v", err)
	}
	if string(result) != `"myvalue"` { // JSON encoded string
		t.Errorf("Stored value = %q, want %q", string(result), `"myvalue"`)
	}
}

func TestCacheService_ComplexDataTypes(t *testing.T) {
	svc, cleanup := setupTestCacheService(t, "test:complex:")
	defer cleanup()

	ctx := context.Background()

	t.Run("slice", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		err := svc.Set(ctx, "slice", input)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		var result []string
		found, err := svc.Get(ctx, "slice", &result)
		if err != nil || !found {
			t.Fatalf("Get() error = %v, found = %v", err, found)
		}
		if len(result) != 3 {
			t.Errorf("len(result) = %d, want 3", len(result))
		}
	})

	t.Run("map", func(t *testing.T) {
		input := map[string]int{"one": 1, "two": 2}
		err := svc.Set(ctx, "map", input)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		var result map[string]int
		found, err := svc.Get(ctx, "map", &result)
		if err != nil || !found {
			t.Fatalf("Get() error = %v, found = %v", err, found)
		}
		if result["one"] != 1 || result["two"] != 2 {
			t.Errorf("result = %v, want map[one:1 two:2]", result)
		}
	})

	t.Run("nested struct", func(t *testing.T) {
		type Inner struct {
			Value int `json:"value"`
		}
		type Outer struct {
			Name  string `json:"name"`
			Inner Inner  `json:"inner"`
		}

		input := Outer{Name: "test", Inner: Inner{Value: 42}}
		err := svc.Set(ctx, "nested", input)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		var result Outer
		found, err := svc.Get(ctx, "nested", &result)
		if err != nil || !found {
			t.Fatalf("Get() error = %v, found = %v", err, found)
		}
		if result.Name != "test" || result.Inner.Value != 42 {
			t.Errorf("result = %+v, want {Name:test Inner:{Value:42}}", result)
		}
	})
}

func TestCacheService_Close(t *testing.T) {
	checkRedisAvailable(t)

	storage := redis.New(redis.Config{
		Host: "localhost",
		Port: 6379,
	})

	svc := NewCacheService(storage, "test:", 5*time.Minute)

	// Close should not error
	err := svc.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
