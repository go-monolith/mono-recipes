package product

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/example/redis-caching-demo/domain/product"
	"github.com/example/redis-caching-demo/modules/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test configuration
const testRedisAddr = "localhost:6379"

// testSetup creates a test environment with database and cache.
type testSetup struct {
	db      *gorm.DB
	repo    *product.Repository
	cache   *cache.Cache
	service *Service
	cleanup func()
}

func setupTest(t *testing.T) *testSetup {
	t.Helper()

	// Create a temporary SQLite database
	dbPath := "test_products_" + t.Name() + ".db"

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create repository and run migrations
	repo := product.NewRepository(db)
	if err := repo.Migrate(); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: testRedisAddr,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s: %v", testRedisAddr, err)
	}

	// Create cache with unique prefix for this test
	prefix := "test:" + t.Name() + ":"
	cleanupKeys(ctx, client, prefix+"*")
	c := cache.New(client, prefix, 5*time.Minute)

	// Create service
	service := NewService(repo, c)

	cleanup := func() {
		cleanupKeys(ctx, client, prefix+"*")
		client.Close()
		sqlDB, _ := db.DB()
		sqlDB.Close()
		os.Remove(dbPath)
	}

	return &testSetup{
		db:      db,
		repo:    repo,
		cache:   c,
		service: service,
		cleanup: cleanup,
	}
}

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

func TestService_Create(t *testing.T) {
	ts := setupTest(t)
	defer ts.cleanup()

	ctx := context.Background()

	req := &product.CreateProductRequest{
		Name:        "Test Product",
		Description: "A test product",
		Price:       99.99,
		Stock:       10,
		Category:    "Test",
	}

	p, err := ts.service.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if p.ID == 0 {
		t.Error("Created product should have non-zero ID")
	}
	if p.Name != req.Name {
		t.Errorf("Name = %q, want %q", p.Name, req.Name)
	}
	if p.Price != req.Price {
		t.Errorf("Price = %f, want %f", p.Price, req.Price)
	}

	// Verify it's in the database
	dbProduct, err := ts.repo.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if dbProduct == nil {
		t.Fatal("Product should exist in database")
	}
}

func TestService_GetByID_CacheAside(t *testing.T) {
	ts := setupTest(t)
	defer ts.cleanup()

	ctx := context.Background()
	ts.cache.ResetStats()

	// Create a product directly in DB
	p := &product.Product{
		Name:        "Cache Test",
		Description: "Testing cache-aside pattern",
		Price:       50.00,
		Stock:       5,
		Category:    "Test",
	}
	if err := ts.repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// First get - should be cache miss
	result1, fromCache1, err := ts.service.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID() first call error = %v", err)
	}
	if fromCache1 {
		t.Error("First GetByID() should be cache miss (fromCache=false)")
	}
	if result1.ID != p.ID {
		t.Errorf("Product ID = %d, want %d", result1.ID, p.ID)
	}

	// Second get - should be cache hit
	result2, fromCache2, err := ts.service.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID() second call error = %v", err)
	}
	if !fromCache2 {
		t.Error("Second GetByID() should be cache hit (fromCache=true)")
	}
	if result2.ID != p.ID {
		t.Errorf("Product ID = %d, want %d", result2.ID, p.ID)
	}

	// Verify stats
	stats := ts.cache.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Cache hits = %d, want 1", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Cache misses = %d, want 1", stats.Misses)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	ts := setupTest(t)
	defer ts.cleanup()

	ctx := context.Background()

	result, fromCache, err := ts.service.GetByID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if result != nil {
		t.Error("GetByID() for nonexistent product should return nil")
	}
	if fromCache {
		t.Error("fromCache should be false for not found")
	}
}

func TestService_List_CacheAside(t *testing.T) {
	ts := setupTest(t)
	defer ts.cleanup()

	ctx := context.Background()
	ts.cache.ResetStats()

	// Create some products
	for i := 1; i <= 3; i++ {
		req := &product.CreateProductRequest{
			Name:     "Product " + string(rune('A'+i-1)),
			Price:    float64(i) * 10,
			Stock:    i,
			Category: "Test",
		}
		ts.service.Create(ctx, req)
	}

	// Reset stats after creates (which may have cache operations)
	ts.cache.ResetStats()

	// First list - cache miss
	products1, total1, fromCache1, err := ts.service.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("List() first call error = %v", err)
	}
	if fromCache1 {
		t.Error("First List() should be cache miss")
	}
	if len(products1) != 3 {
		t.Errorf("Products count = %d, want 3", len(products1))
	}
	if total1 != 3 {
		t.Errorf("Total = %d, want 3", total1)
	}

	// Second list - cache hit
	products2, total2, fromCache2, err := ts.service.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("List() second call error = %v", err)
	}
	if !fromCache2 {
		t.Error("Second List() should be cache hit")
	}
	if len(products2) != 3 {
		t.Errorf("Products count = %d, want 3", len(products2))
	}
	if total2 != 3 {
		t.Errorf("Total = %d, want 3", total2)
	}
}

func TestService_List_Pagination(t *testing.T) {
	ts := setupTest(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create 5 products
	for i := 1; i <= 5; i++ {
		req := &product.CreateProductRequest{
			Name:     "Product " + string(rune('A'+i-1)),
			Price:    float64(i) * 10,
			Stock:    i,
			Category: "Test",
		}
		ts.service.Create(ctx, req)
	}

	ts.cache.ResetStats()

	// Different pagination params should have different cache keys
	list1, _, fromCache1, _ := ts.service.List(ctx, 0, 2)
	list2, _, fromCache2, _ := ts.service.List(ctx, 2, 2)

	if fromCache1 || fromCache2 {
		t.Error("First requests with different pagination should both be cache misses")
	}

	if len(list1) != 2 {
		t.Errorf("list1 length = %d, want 2", len(list1))
	}
	if len(list2) != 2 {
		t.Errorf("list2 length = %d, want 2", len(list2))
	}

	// Stats should show 2 misses
	stats := ts.cache.GetStats()
	if stats.Misses != 2 {
		t.Errorf("Misses = %d, want 2", stats.Misses)
	}
}

func TestService_Update_InvalidatesCache(t *testing.T) {
	ts := setupTest(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create product
	req := &product.CreateProductRequest{
		Name:     "Update Test",
		Price:    100.00,
		Stock:    10,
		Category: "Test",
	}
	created, _ := ts.service.Create(ctx, req)

	// Populate cache
	ts.service.GetByID(ctx, created.ID)

	// Verify it's cached
	_, fromCache, _ := ts.service.GetByID(ctx, created.ID)
	if !fromCache {
		t.Fatal("Product should be cached before update")
	}

	// Update
	newPrice := 150.00
	updateReq := &product.UpdateProductRequest{
		Price: &newPrice,
	}
	updated, err := ts.service.Update(ctx, created.ID, updateReq)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Price != newPrice {
		t.Errorf("Updated price = %f, want %f", updated.Price, newPrice)
	}

	// After update, cache should be invalidated - next get should be miss
	result, fromCache, _ := ts.service.GetByID(ctx, created.ID)
	if fromCache {
		t.Error("GetByID() after update should be cache miss (cache should be invalidated)")
	}
	if result.Price != newPrice {
		t.Errorf("Price after update = %f, want %f", result.Price, newPrice)
	}
}

func TestService_Delete_InvalidatesCache(t *testing.T) {
	ts := setupTest(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create product
	req := &product.CreateProductRequest{
		Name:     "Delete Test",
		Price:    100.00,
		Stock:    10,
		Category: "Test",
	}
	created, _ := ts.service.Create(ctx, req)

	// Populate cache
	ts.service.GetByID(ctx, created.ID)

	// Verify it's cached
	_, fromCache, _ := ts.service.GetByID(ctx, created.ID)
	if !fromCache {
		t.Fatal("Product should be cached before delete")
	}

	// Delete
	err := ts.service.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// After delete, product should not exist
	result, _, _ := ts.service.GetByID(ctx, created.ID)
	if result != nil {
		t.Error("GetByID() after delete should return nil")
	}
}

func TestService_CacheStats(t *testing.T) {
	ts := setupTest(t)
	defer ts.cleanup()

	ctx := context.Background()
	ts.cache.ResetStats()

	// Create and fetch to generate cache activity
	req := &product.CreateProductRequest{
		Name:     "Stats Test",
		Price:    100.00,
		Stock:    10,
		Category: "Test",
	}
	created, _ := ts.service.Create(ctx, req)

	// Get twice - 1 miss, 1 hit
	ts.service.GetByID(ctx, created.ID)
	ts.service.GetByID(ctx, created.ID)

	stats := ts.service.GetCacheStats()
	if stats.Hits < 1 {
		t.Errorf("Hits = %d, want >= 1", stats.Hits)
	}
	if stats.Misses < 1 {
		t.Errorf("Misses = %d, want >= 1", stats.Misses)
	}
}

func TestService_ResetCacheStats(t *testing.T) {
	ts := setupTest(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create and fetch to generate cache activity
	req := &product.CreateProductRequest{
		Name:     "Reset Test",
		Price:    100.00,
		Stock:    10,
		Category: "Test",
	}
	created, _ := ts.service.Create(ctx, req)
	ts.service.GetByID(ctx, created.ID)
	ts.service.GetByID(ctx, created.ID)

	// Reset
	ts.service.ResetCacheStats()

	stats := ts.service.GetCacheStats()
	if stats.Hits != 0 {
		t.Errorf("Hits after reset = %d, want 0", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("Misses after reset = %d, want 0", stats.Misses)
	}
}

func TestCacheKeyByID(t *testing.T) {
	tests := []struct {
		id   uint
		want string
	}{
		{1, "id:1"},
		{100, "id:100"},
		{0, "id:0"},
	}

	for _, tc := range tests {
		got := cacheKeyByID(tc.id)
		if got != tc.want {
			t.Errorf("cacheKeyByID(%d) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestCacheKeyList(t *testing.T) {
	tests := []struct {
		offset int
		limit  int
		want   string
	}{
		{0, 10, "list:0:10"},
		{10, 20, "list:10:20"},
		{0, 0, "list:0:0"},
	}

	for _, tc := range tests {
		got := cacheKeyList(tc.offset, tc.limit)
		if got != tc.want {
			t.Errorf("cacheKeyList(%d, %d) = %q, want %q", tc.offset, tc.limit, got, tc.want)
		}
	}
}
