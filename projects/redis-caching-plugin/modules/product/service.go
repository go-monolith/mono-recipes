// Package product provides the product service with caching support.
package product

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/example/redis-caching-demo/domain/product"
	"github.com/example/redis-caching-demo/modules/cache"
	"golang.org/x/sync/singleflight"
)

// Service provides product operations with caching.
type Service struct {
	repo    *product.Repository
	cache   cache.CacheService
	sfGroup singleflight.Group // Prevents cache stampede
}

// NewService creates a new product service.
func NewService(repo *product.Repository, c cache.CacheService) *Service {
	return &Service{
		repo:  repo,
		cache: c,
	}
}

// cacheKeyByID returns the cache key for a product by ID.
func cacheKeyByID(id uint) string {
	return "id:" + strconv.FormatUint(uint64(id), 10)
}

// cacheKeyList returns the cache key for the product list.
func cacheKeyList(offset, limit int) string {
	return fmt.Sprintf("list:%d:%d", offset, limit)
}

// Create creates a new product (no caching, invalidates cache).
func (s *Service) Create(ctx context.Context, req *product.CreateProductRequest) (*product.Product, error) {
	p := &product.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}

	// Invalidate all cache since we added a new product
	if err := s.cache.InvalidateAll(ctx); err != nil {
		log.Printf("[product] Warning: failed to invalidate cache: %v", err)
	}

	log.Printf("[product] Created product ID=%d, cache invalidated", p.ID)
	return p, nil
}

// GetByID retrieves a product by ID with caching (cache-aside pattern).
// Uses singleflight to prevent cache stampede on concurrent cache misses.
func (s *Service) GetByID(ctx context.Context, id uint) (*product.Product, bool, error) {
	cacheKey := cacheKeyByID(id)

	// Step 1: Check cache first
	var cached product.Product
	found, err := s.cache.Get(ctx, cacheKey, &cached)
	if err != nil {
		log.Printf("[product] Cache error for ID=%d: %v", id, err)
		// Continue to database on cache error
	}

	if found {
		log.Printf("[product] Cache HIT for ID=%d", id)
		return &cached, true, nil
	}

	// Step 2: Cache miss - query database using singleflight to prevent stampede
	log.Printf("[product] Cache MISS for ID=%d, querying database", id)
	sfKey := fmt.Sprintf("product:%d", id)
	val, err, _ := s.sfGroup.Do(sfKey, func() (any, error) {
		return s.repo.GetByID(ctx, id)
	})
	if err != nil {
		return nil, false, err
	}

	p, ok := val.(*product.Product)
	if !ok || p == nil {
		return nil, false, nil // Not found
	}

	// Step 3: Populate cache
	if err := s.cache.Set(ctx, cacheKey, p); err != nil {
		log.Printf("[product] Warning: failed to cache product ID=%d: %v", id, err)
	} else {
		log.Printf("[product] Cached product ID=%d", id)
	}

	return p, false, nil
}

// List retrieves all products with caching (cache-aside pattern).
func (s *Service) List(ctx context.Context, offset, limit int) ([]product.Product, int64, bool, error) {
	cacheKey := cacheKeyList(offset, limit)

	// Step 1: Check cache first
	var cached struct {
		Products []product.Product `json:"products"`
		Total    int64             `json:"total"`
	}

	found, err := s.cache.Get(ctx, cacheKey, &cached)
	if err != nil {
		log.Printf("[product] Cache error for list: %v", err)
	}

	if found {
		log.Printf("[product] Cache HIT for list (offset=%d, limit=%d)", offset, limit)
		return cached.Products, cached.Total, true, nil
	}

	// Step 2: Cache miss - query database
	log.Printf("[product] Cache MISS for list, querying database")
	products, total, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, false, err
	}

	// Step 3: Populate cache
	cacheData := struct {
		Products []product.Product `json:"products"`
		Total    int64             `json:"total"`
	}{
		Products: products,
		Total:    total,
	}

	if err := s.cache.Set(ctx, cacheKey, cacheData); err != nil {
		log.Printf("[product] Warning: failed to cache list: %v", err)
	} else {
		log.Printf("[product] Cached list (offset=%d, limit=%d, count=%d)", offset, limit, len(products))
	}

	return products, total, false, nil
}

// Update updates a product and invalidates cache.
func (s *Service) Update(ctx context.Context, id uint, req *product.UpdateProductRequest) (*product.Product, error) {
	// Get existing product
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil // Not found
	}

	// Apply updates
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Price != nil {
		p.Price = *req.Price
	}
	if req.Stock != nil {
		p.Stock = *req.Stock
	}
	if req.Category != nil {
		p.Category = *req.Category
	}

	// Save to database
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}

	// Invalidate caches
	s.invalidateCaches(ctx, id)

	log.Printf("[product] Updated product ID=%d, caches invalidated", id)
	return p, nil
}

// Delete deletes a product and invalidates cache.
func (s *Service) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate caches
	s.invalidateCaches(ctx, id)

	log.Printf("[product] Deleted product ID=%d, caches invalidated", id)
	return nil
}

// invalidateCaches removes the product from cache and invalidates all cache.
func (s *Service) invalidateCaches(ctx context.Context, id uint) {
	// Invalidate individual product cache
	cacheKey := cacheKeyByID(id)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		log.Printf("[product] Warning: failed to invalidate cache for ID=%d: %v", id, err)
	}

	// Invalidate all cache (replaces pattern-based deletion)
	if err := s.cache.InvalidateAll(ctx); err != nil {
		log.Printf("[product] Warning: failed to invalidate all cache: %v", err)
	}
}
