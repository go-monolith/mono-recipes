package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestConfig for unit tests - requires Redis running on localhost:6379
const testRedisAddr = "localhost:6379"

// setupTestCache creates a cache instance for testing.
// Returns the cache and a cleanup function.
func setupTestCache(t *testing.T, prefix string) (*Cache, func()) {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: testRedisAddr,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s: %v", testRedisAddr, err)
	}

	// Clean up any existing keys with this prefix
	cleanupKeys(ctx, client, prefix+"*")

	cache := New(client, prefix, 5*time.Minute)

	cleanup := func() {
		cleanupKeys(ctx, client, prefix+"*")
		client.Close()
	}

	return cache, cleanup
}

// cleanupKeys removes all keys matching the pattern.
func cleanupKeys(ctx context.Context, client *redis.Client, pattern string) {
	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

func TestNew(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: testRedisAddr})
	defer client.Close()

	cache := New(client, "test:", 10*time.Minute)

	if cache == nil {
		t.Fatal("New() returned nil")
	}
	if cache.prefix != "test:" {
		t.Errorf("prefix = %q, want %q", cache.prefix, "test:")
	}
	if cache.ttl != 10*time.Minute {
		t.Errorf("ttl = %v, want %v", cache.ttl, 10*time.Minute)
	}
	if cache.stats == nil {
		t.Error("stats is nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("RedisAddr = %q, want %q", cfg.RedisAddr, "localhost:6379")
	}
	if cfg.Prefix != "cache:" {
		t.Errorf("Prefix = %q, want %q", cfg.Prefix, "cache:")
	}
	if cfg.TTL != 5*time.Minute {
		t.Errorf("TTL = %v, want %v", cfg.TTL, 5*time.Minute)
	}
}

func TestCache_SetAndGet(t *testing.T) {
	cache, cleanup := setupTestCache(t, "test:setget:")
	defer cleanup()

	ctx := context.Background()

	type TestData struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
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
			err := cache.Set(ctx, tc.key, tc.value)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			// Get the value
			var result TestData
			found, err := cache.Get(ctx, tc.key, &result)
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

func TestCache_GetMiss(t *testing.T) {
	cache, cleanup := setupTestCache(t, "test:miss:")
	defer cleanup()

	ctx := context.Background()

	var result string
	found, err := cache.Get(ctx, "nonexistent", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if found {
		t.Error("Get() returned found = true for nonexistent key, want false")
	}
}

func TestCache_SetWithTTL(t *testing.T) {
	cache, cleanup := setupTestCache(t, "test:ttl:")
	defer cleanup()

	ctx := context.Background()

	// Set with very short TTL
	err := cache.SetWithTTL(ctx, "expiring", "test value", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("SetWithTTL() error = %v", err)
	}

	// Should exist immediately
	var result string
	found, err := cache.Get(ctx, "expiring", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Fatal("Get() immediately after Set should find the key")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should be expired
	found, err = cache.Get(ctx, "expiring", &result)
	if err != nil {
		t.Fatalf("Get() after expiration error = %v", err)
	}
	if found {
		t.Error("Get() after TTL expiration should return found = false")
	}
}

func TestCache_Delete(t *testing.T) {
	cache, cleanup := setupTestCache(t, "test:delete:")
	defer cleanup()

	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "to-delete", "some value")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify it exists
	var result string
	found, _ := cache.Get(ctx, "to-delete", &result)
	if !found {
		t.Fatal("Key should exist before deletion")
	}

	// Delete it
	err = cache.Delete(ctx, "to-delete")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	found, _ = cache.Get(ctx, "to-delete", &result)
	if found {
		t.Error("Key should not exist after deletion")
	}
}

func TestCache_DeletePattern(t *testing.T) {
	cache, cleanup := setupTestCache(t, "test:pattern:")
	defer cleanup()

	ctx := context.Background()

	// Set multiple values with pattern
	for i := 1; i <= 5; i++ {
		key := "list:" + string(rune('a'+i-1))
		err := cache.Set(ctx, key, i)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	// Set a value that shouldn't match
	err := cache.Set(ctx, "other:key", "keep me")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Delete with pattern
	err = cache.DeletePattern(ctx, "list:*")
	if err != nil {
		t.Fatalf("DeletePattern() error = %v", err)
	}

	// Verify list keys are gone
	for i := 1; i <= 5; i++ {
		key := "list:" + string(rune('a'+i-1))
		var result int
		found, _ := cache.Get(ctx, key, &result)
		if found {
			t.Errorf("Key %q should have been deleted by pattern", key)
		}
	}

	// Verify other key still exists
	var result string
	found, _ := cache.Get(ctx, "other:key", &result)
	if !found {
		t.Error("Key 'other:key' should not have been deleted")
	}
}

func TestCache_Stats(t *testing.T) {
	cache, cleanup := setupTestCache(t, "test:stats:")
	defer cleanup()

	ctx := context.Background()

	// Reset stats
	cache.ResetStats()

	// Set a value
	cache.Set(ctx, "stats-test", "value")

	// Get - should be a hit
	var result string
	cache.Get(ctx, "stats-test", &result)

	// Get nonexistent - should be a miss
	cache.Get(ctx, "nonexistent", &result)

	// Get again - should be another hit
	cache.Get(ctx, "stats-test", &result)

	// Delete
	cache.Delete(ctx, "stats-test")

	// Check stats
	stats := cache.GetStats()

	if stats.Sets != 1 {
		t.Errorf("Sets = %d, want 1", stats.Sets)
	}
	if stats.Hits != 2 {
		t.Errorf("Hits = %d, want 2", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}
	if stats.Deletes != 1 {
		t.Errorf("Deletes = %d, want 1", stats.Deletes)
	}
	if stats.TotalGets != 3 {
		t.Errorf("TotalGets = %d, want 3", stats.TotalGets)
	}

	// Hit rate should be ~66.67% (2 hits out of 3 gets)
	expectedHitRate := float64(2) / float64(3) * 100
	if stats.HitRate < expectedHitRate-0.01 || stats.HitRate > expectedHitRate+0.01 {
		t.Errorf("HitRate = %f, want ~%f", stats.HitRate, expectedHitRate)
	}
}

func TestCache_ResetStats(t *testing.T) {
	cache, cleanup := setupTestCache(t, "test:reset:")
	defer cleanup()

	ctx := context.Background()

	// Generate some stats
	cache.Set(ctx, "key", "value")
	var result string
	cache.Get(ctx, "key", &result)
	cache.Get(ctx, "nonexistent", &result)
	cache.Delete(ctx, "key")

	// Verify stats are non-zero
	stats := cache.GetStats()
	if stats.Sets == 0 || stats.Hits == 0 || stats.Misses == 0 || stats.Deletes == 0 {
		t.Fatal("Stats should be non-zero before reset")
	}

	// Reset
	cache.ResetStats()

	// Verify all stats are zero
	stats = cache.GetStats()
	if stats.Sets != 0 {
		t.Errorf("Sets after reset = %d, want 0", stats.Sets)
	}
	if stats.Hits != 0 {
		t.Errorf("Hits after reset = %d, want 0", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("Misses after reset = %d, want 0", stats.Misses)
	}
	if stats.Deletes != 0 {
		t.Errorf("Deletes after reset = %d, want 0", stats.Deletes)
	}
	if stats.Errors != 0 {
		t.Errorf("Errors after reset = %d, want 0", stats.Errors)
	}
	if stats.HitRate != 0 {
		t.Errorf("HitRate after reset = %f, want 0", stats.HitRate)
	}
	if stats.TotalGets != 0 {
		t.Errorf("TotalGets after reset = %d, want 0", stats.TotalGets)
	}
}

func TestCache_Ping(t *testing.T) {
	cache, cleanup := setupTestCache(t, "test:ping:")
	defer cleanup()

	ctx := context.Background()

	err := cache.Ping(ctx)
	if err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestCache_GetClient(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: testRedisAddr})
	defer client.Close()

	cache := New(client, "test:", time.Minute)

	if cache.GetClient() != client {
		t.Error("GetClient() should return the same client passed to New()")
	}
}

func TestCache_KeyPrefix(t *testing.T) {
	cache, cleanup := setupTestCache(t, "myprefix:")
	defer cleanup()

	ctx := context.Background()

	// Set a value
	err := cache.Set(ctx, "mykey", "myvalue")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify the key is stored with prefix using the underlying client
	client := cache.GetClient()
	result, err := client.Get(ctx, "myprefix:mykey").Result()
	if err != nil {
		t.Fatalf("Direct Redis Get error = %v", err)
	}
	if result != `"myvalue"` { // JSON encoded string
		t.Errorf("Stored value = %q, want %q", result, `"myvalue"`)
	}
}

func TestCache_ComplexDataTypes(t *testing.T) {
	cache, cleanup := setupTestCache(t, "test:complex:")
	defer cleanup()

	ctx := context.Background()

	t.Run("slice", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		err := cache.Set(ctx, "slice", input)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		var result []string
		found, err := cache.Get(ctx, "slice", &result)
		if err != nil || !found {
			t.Fatalf("Get() error = %v, found = %v", err, found)
		}
		if len(result) != 3 {
			t.Errorf("len(result) = %d, want 3", len(result))
		}
	})

	t.Run("map", func(t *testing.T) {
		input := map[string]int{"one": 1, "two": 2}
		err := cache.Set(ctx, "map", input)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		var result map[string]int
		found, err := cache.Get(ctx, "map", &result)
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
		err := cache.Set(ctx, "nested", input)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		var result Outer
		found, err := cache.Get(ctx, "nested", &result)
		if err != nil || !found {
			t.Fatalf("Get() error = %v, found = %v", err, found)
		}
		if result.Name != "test" || result.Inner.Value != 42 {
			t.Errorf("result = %+v, want {Name:test Inner:{Value:42}}", result)
		}
	})
}

func TestCache_HitRateCalculation(t *testing.T) {
	cache, cleanup := setupTestCache(t, "test:hitrate:")
	defer cleanup()

	ctx := context.Background()
	cache.ResetStats()

	// No gets yet - hit rate should be 0
	stats := cache.GetStats()
	if stats.HitRate != 0 {
		t.Errorf("HitRate with no gets = %f, want 0", stats.HitRate)
	}

	// Set a value
	cache.Set(ctx, "key", "value")

	// 1 hit, 0 misses = 100% hit rate
	var result string
	cache.Get(ctx, "key", &result)
	stats = cache.GetStats()
	if stats.HitRate != 100 {
		t.Errorf("HitRate with 1 hit, 0 misses = %f, want 100", stats.HitRate)
	}

	// 1 hit, 1 miss = 50% hit rate
	cache.Get(ctx, "nonexistent", &result)
	stats = cache.GetStats()
	if stats.HitRate != 50 {
		t.Errorf("HitRate with 1 hit, 1 miss = %f, want 50", stats.HitRate)
	}

	// 2 hits, 1 miss = ~66.67% hit rate
	cache.Get(ctx, "key", &result)
	stats = cache.GetStats()
	expectedHitRate := float64(2) / float64(3) * 100
	if stats.HitRate < expectedHitRate-0.01 || stats.HitRate > expectedHitRate+0.01 {
		t.Errorf("HitRate with 2 hits, 1 miss = %f, want ~%f", stats.HitRate, expectedHitRate)
	}
}
