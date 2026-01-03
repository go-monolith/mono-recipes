package api

import (
	"github.com/example/hexagonal-architecture/modules/task"
	"github.com/gofiber/fiber/v2"
)

// setupRoutes configures all HTTP routes.
func (m *APIModule) setupRoutes() {
	// Health check endpoint
	m.app.Get("/health", m.healthHandler)

	// API v1 routes
	api := m.app.Group("/api/v1")

	// Task endpoints
	tasks := api.Group("/tasks")
	tasks.Post("/", m.createTask)
	tasks.Get("/", m.listTasks)
	tasks.Get("/:id", m.getTask)
	tasks.Put("/:id", m.updateTask)
	tasks.Delete("/:id", m.deleteTask)
	tasks.Post("/:id/complete", m.completeTask)
}

// healthHandler handles GET /health.
func (m *APIModule) healthHandler(c *fiber.Ctx) error {
	return c.JSON(HealthResponse{
		Status: "healthy",
		Details: map[string]any{
			"module": "api",
			"port":   3000,
		},
	})
}

// createTask handles POST /api/v1/tasks.
func (m *APIModule) createTask(c *fiber.Ctx) error {
	var req CreateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate required fields
	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Title is required",
		})
	}
	if req.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "User ID is required",
		})
	}

	// Call task service via adapter (driving adapter -> core domain)
	resp, err := m.taskAdapter.CreateTask(c.Context(), &task.CreateTaskRequest{
		Title:       req.Title,
		Description: req.Description,
		UserID:      req.UserID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "create_failed",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(TaskResponse{
		ID:        resp.ID,
		Title:     resp.Title,
		Status:    resp.Status,
		CreatedAt: resp.CreatedAt,
	})
}

// getTask handles GET /api/v1/tasks/:id.
func (m *APIModule) getTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	if taskID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Task ID is required",
		})
	}

	resp, err := m.taskAdapter.GetTask(c.Context(), taskID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "not_found",
			Message: "Task not found",
		})
	}

	return c.JSON(TaskResponse{
		ID:          resp.ID,
		Title:       resp.Title,
		Description: resp.Description,
		Status:      resp.Status,
		UserID:      resp.UserID,
		CreatedAt:   resp.CreatedAt,
		UpdatedAt:   resp.UpdatedAt,
		CompletedAt: resp.CompletedAt,
	})
}

// listTasks handles GET /api/v1/tasks.
func (m *APIModule) listTasks(c *fiber.Ctx) error {
	userID := c.Query("user_id", "")

	resp, err := m.taskAdapter.ListTasks(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "list_failed",
			Message: err.Error(),
		})
	}

	tasks := make([]TaskResponse, 0, len(resp.Tasks))
	for _, t := range resp.Tasks {
		tasks = append(tasks, TaskResponse{
			ID:          t.ID,
			Title:       t.Title,
			Description: t.Description,
			Status:      t.Status,
			UserID:      t.UserID,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
			CompletedAt: t.CompletedAt,
		})
	}

	return c.JSON(ListTasksResponse{
		Tasks: tasks,
		Total: resp.Total,
	})
}

// updateTask handles PUT /api/v1/tasks/:id.
func (m *APIModule) updateTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	if taskID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Task ID is required",
		})
	}

	var req UpdateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	resp, err := m.taskAdapter.UpdateTask(c.Context(), &task.UpdateTaskRequest{
		TaskID:      taskID,
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "not_found",
			Message: "Task not found",
		})
	}

	return c.JSON(TaskResponse{
		ID:          resp.ID,
		Title:       resp.Title,
		Description: resp.Description,
		Status:      resp.Status,
		UserID:      resp.UserID,
		CreatedAt:   resp.CreatedAt,
		UpdatedAt:   resp.UpdatedAt,
		CompletedAt: resp.CompletedAt,
	})
}

// deleteTask handles DELETE /api/v1/tasks/:id.
func (m *APIModule) deleteTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	if taskID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Task ID is required",
		})
	}

	// Get user_id from query parameter for authorization
	userID := c.Query("user_id", "")

	err := m.taskAdapter.DeleteTask(c.Context(), taskID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "not_found",
			Message: "Task not found",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// completeTask handles POST /api/v1/tasks/:id/complete.
func (m *APIModule) completeTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	if taskID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "validation_error",
			Message: "Task ID is required",
		})
	}

	resp, err := m.taskAdapter.CompleteTask(c.Context(), taskID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "not_found",
			Message: "Task not found",
		})
	}

	return c.JSON(TaskResponse{
		ID:          resp.ID,
		Title:       resp.Title,
		Description: resp.Description,
		Status:      resp.Status,
		UserID:      resp.UserID,
		CreatedAt:   resp.CreatedAt,
		UpdatedAt:   resp.UpdatedAt,
		CompletedAt: resp.CompletedAt,
	})
}
