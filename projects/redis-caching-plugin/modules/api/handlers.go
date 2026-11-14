package api

import (
	"context"
	"log"
	"strconv"
	"time"

	productcontract "github.com/example/redis-caching-demo/modules/product/contract"
	"github.com/gofiber/fiber/v2"
)

// requestTimeout bounds each call to the product module. The deadline is
// propagated to the product service handler.
const requestTimeout = 5 * time.Second

// Handlers provides HTTP handlers for the API.
type Handlers struct {
	products ProductPort
}

// NewHandlers creates a new handlers instance on top of the product port.
func NewHandlers(products ProductPort) *Handlers {
	return &Handlers{
		products: products,
	}
}

// callContext derives the context for one product call from the request.
func callContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), requestTimeout)
}

// ListProducts handles GET /api/v1/products.
func (h *Handlers) ListProducts(c *fiber.Ctx) error {
	offset := max(c.QueryInt("offset", 0), 0)
	limit := min(max(c.QueryInt("limit", 20), 1), 100)

	ctx, cancel := callContext(c)
	defer cancel()

	start := time.Now()
	resp, err := h.products.ListProducts(ctx, productcontract.ListProductsRequest{Offset: offset, Limit: limit})
	duration := time.Since(start)

	if err != nil {
		log.Printf("[api] Error listing products: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve products",
		})
	}

	return c.JSON(fiber.Map{
		"products":    resp.Products,
		"total":       resp.Total,
		"offset":      offset,
		"limit":       limit,
		"from_cache":  resp.FromCache,
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

	ctx, cancel := callContext(c)
	defer cancel()

	start := time.Now()
	resp, err := h.products.GetProduct(ctx, productcontract.GetProductRequest{ID: uint(id)})
	duration := time.Since(start)

	if err != nil {
		log.Printf("[api] Error getting product ID=%d: %v", id, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve product",
		})
	}

	if !resp.Found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Not Found",
			"message": "Product not found",
		})
	}

	return c.JSON(fiber.Map{
		"product":     resp.Product,
		"from_cache":  resp.FromCache,
		"duration_ms": duration.Milliseconds(),
	})
}

// CreateProduct handles POST /api/v1/products.
func (h *Handlers) CreateProduct(c *fiber.Ctx) error {
	var req productcontract.CreateProductRequest
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

	ctx, cancel := callContext(c)
	defer cancel()

	resp, err := h.products.CreateProduct(ctx, req)
	if err != nil {
		log.Printf("[api] Error creating product: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal Server Error",
			"message": "Failed to create product",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"product": resp.Product,
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

	var req productcontract.UpdateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Bad Request",
			"message": "Invalid request body",
		})
	}

	req.ID = uint(id) // the path parameter wins over any "id" in the body

	ctx, cancel := callContext(c)
	defer cancel()

	resp, err := h.products.UpdateProduct(ctx, req)
	if err != nil {
		log.Printf("[api] Error updating product ID=%d: %v", id, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal Server Error",
			"message": "Failed to update product",
		})
	}

	if !resp.Found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Not Found",
			"message": "Product not found",
		})
	}

	return c.JSON(fiber.Map{
		"product": resp.Product,
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

	ctx, cancel := callContext(c)
	defer cancel()

	if _, err := h.products.DeleteProduct(ctx, productcontract.DeleteProductRequest{ID: uint(id)}); err != nil {
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
