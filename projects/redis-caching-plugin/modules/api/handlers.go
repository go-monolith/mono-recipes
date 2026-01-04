package api

import (
	"log"
	"strconv"
	"time"

	"github.com/example/redis-caching-demo/domain/product"
	productmod "github.com/example/redis-caching-demo/modules/product"
	"github.com/gofiber/fiber/v2"
)

// Handlers provides HTTP handlers for the API.
type Handlers struct {
	productService *productmod.Service
}

// NewHandlers creates a new handlers instance.
func NewHandlers(productService *productmod.Service) *Handlers {
	return &Handlers{
		productService: productService,
	}
}

// ListProducts handles GET /api/v1/products.
func (h *Handlers) ListProducts(c *fiber.Ctx) error {
	offset := c.QueryInt("offset", 0)
	limit := c.QueryInt("limit", 20)

	// Validate pagination parameters
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Cap at 100
	}

	start := time.Now()
	products, total, fromCache, err := h.productService.List(c.Context(), offset, limit)
	duration := time.Since(start)

	if err != nil {
		log.Printf("[api] Error listing products: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve products",
		})
	}

	return c.JSON(fiber.Map{
		"products":    products,
		"total":       total,
		"offset":      offset,
		"limit":       limit,
		"from_cache":  fromCache,
		"duration_ms": duration.Milliseconds(),
	})
}

// GetProduct handles GET /api/v1/products/:id.
func (h *Handlers) GetProduct(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "Invalid product ID",
		})
	}

	start := time.Now()
	p, fromCache, err := h.productService.GetByID(c.Context(), uint(id))
	duration := time.Since(start)

	if err != nil {
		log.Printf("[api] Error getting product ID=%d: %v", id, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve product",
		})
	}

	if p == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Not Found",
			"message": "Product not found",
		})
	}

	return c.JSON(fiber.Map{
		"product":     p,
		"from_cache":  fromCache,
		"duration_ms": duration.Milliseconds(),
	})
}

// CreateProduct handles POST /api/v1/products.
func (h *Handlers) CreateProduct(c *fiber.Ctx) error {
	var req product.CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "Invalid request body",
		})
	}

	// Basic validation
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "Name is required",
		})
	}
	if req.Price <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "Price must be greater than 0",
		})
	}

	p, err := h.productService.Create(c.Context(), &req)
	if err != nil {
		log.Printf("[api] Error creating product: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal Server Error",
			"message": "Failed to create product",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"product": p,
		"message": "Product created successfully",
	})
}

// UpdateProduct handles PUT /api/v1/products/:id.
func (h *Handlers) UpdateProduct(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "Invalid product ID",
		})
	}

	var req product.UpdateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "Invalid request body",
		})
	}

	p, err := h.productService.Update(c.Context(), uint(id), &req)
	if err != nil {
		log.Printf("[api] Error updating product ID=%d: %v", id, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal Server Error",
			"message": "Failed to update product",
		})
	}

	if p == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Not Found",
			"message": "Product not found",
		})
	}

	return c.JSON(fiber.Map{
		"product": p,
		"message": "Product updated successfully, cache invalidated",
	})
}

// DeleteProduct handles DELETE /api/v1/products/:id.
func (h *Handlers) DeleteProduct(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "Invalid product ID",
		})
	}

	if err := h.productService.Delete(c.Context(), uint(id)); err != nil {
		log.Printf("[api] Error deleting product ID=%d: %v", id, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal Server Error",
			"message": "Failed to delete product",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Product deleted successfully, cache invalidated",
	})
}

// HealthCheck handles GET /health.
func (h *Handlers) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"service":   "redis-caching-demo",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
