package api

import (
	"github.com/example/hexagonal-architecture/modules/task"
	"github.com/gofiber/fiber/v2"
)

func (m *APIModule) setupRoutes() {
	m.app.Get("/health", m.healthHandler)

	api := m.app.Group("/api/v1")
	tasks := api.Group("/tasks")
	tasks.Post("/", m.createTask)
	tasks.Get("/", m.listTasks)
	tasks.Get("/:id", m.getTask)
	tasks.Put("/:id", m.updateTask)
	tasks.Delete("/:id", m.deleteTask)
	tasks.Post("/:id/complete", m.completeTask)
}

func (m *APIModule) healthHandler(c *fiber.Ctx) error {
	return c.JSON(HealthResponse{
		Status: "healthy",
		Details: map[string]any{
			"module": "api",
			"port":   3000,
		},
	})
}

func (m *APIModule) createTask(c *fiber.Ctx) error {
	var req CreateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid_request", "Invalid request body")
	}
	if req.Title == "" {
		return validationError(c, "Title is required")
	}
	if req.UserID == "" {
		return validationError(c, "User ID is required")
	}

	resp, err := m.taskAdapter.CreateTask(c.Context(), &task.CreateTaskRequest{
		Title:       req.Title,
		Description: req.Description,
		UserID:      req.UserID,
	})
	if err != nil {
		return serverError(c, "create_failed", err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(TaskResponse{
		ID:        resp.ID,
		Title:     resp.Title,
		Status:    resp.Status,
		CreatedAt: resp.CreatedAt,
	})
}

func (m *APIModule) getTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	if taskID == "" {
		return validationError(c, "Task ID is required")
	}

	resp, err := m.taskAdapter.GetTask(c.Context(), taskID)
	if err != nil {
		return notFound(c, "Task not found")
	}

	return c.JSON(taskResponseFromService(resp))
}

func (m *APIModule) listTasks(c *fiber.Ctx) error {
	userID := c.Query("user_id")

	resp, err := m.taskAdapter.ListTasks(c.Context(), userID)
	if err != nil {
		return serverError(c, "list_failed", err.Error())
	}

	tasks := make([]TaskResponse, 0, len(resp.Tasks))
	for _, t := range resp.Tasks {
		tasks = append(tasks, taskResponseFromService(&t))
	}

	return c.JSON(ListTasksResponse{
		Tasks: tasks,
		Total: resp.Total,
	})
}

func (m *APIModule) updateTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	if taskID == "" {
		return validationError(c, "Task ID is required")
	}

	var req UpdateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "invalid_request", "Invalid request body")
	}

	resp, err := m.taskAdapter.UpdateTask(c.Context(), &task.UpdateTaskRequest{
		TaskID:      taskID,
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		return notFound(c, "Task not found")
	}

	return c.JSON(taskResponseFromService(resp))
}

func (m *APIModule) deleteTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	if taskID == "" {
		return validationError(c, "Task ID is required")
	}

	userID := c.Query("user_id")
	if err := m.taskAdapter.DeleteTask(c.Context(), taskID, userID); err != nil {
		return notFound(c, "Task not found")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (m *APIModule) completeTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	if taskID == "" {
		return validationError(c, "Task ID is required")
	}

	resp, err := m.taskAdapter.CompleteTask(c.Context(), taskID)
	if err != nil {
		return notFound(c, "Task not found")
	}

	return c.JSON(taskResponseFromService(resp))
}

// taskResponseFromService converts a task service response to an API response.
func taskResponseFromService(t *task.TaskResponse) TaskResponse {
	return TaskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		UserID:      t.UserID,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
	}
}

// Error response helpers

func badRequest(c *fiber.Ctx, errorCode, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}

func validationError(c *fiber.Ctx, message string) error {
	return badRequest(c, "validation_error", message)
}

func notFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
		Error:   "not_found",
		Message: message,
	})
}

func serverError(c *fiber.Ctx, errorCode, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}
