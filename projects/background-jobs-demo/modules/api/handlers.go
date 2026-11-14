package api

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	workercontract "github.com/example/background-jobs-demo/modules/worker/contract"
)

// Handler handles HTTP requests for job management.
type Handler struct {
	service *Service
}

// NewHandler creates a new API handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateJobRequest is the request body for creating a job.
type CreateJobRequest struct {
	Type     string `json:"type"`
	Payload  any    `json:"payload"`
	Priority int    `json:"priority,omitempty"`
}

// JobResponse is the response for a single job.
type JobResponse struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	Status          string `json:"status"`
	Priority        int    `json:"priority"`
	Payload         any    `json:"payload"`
	Result          any    `json:"result,omitempty"`
	Error           string `json:"error,omitempty"`
	Progress        int    `json:"progress"`
	ProgressMessage string `json:"progress_message,omitempty"`
	RetryCount      int    `json:"retry_count"`
	MaxRetries      int    `json:"max_retries"`
	WorkerID        string `json:"worker_id,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	StartedAt       string `json:"started_at,omitempty"`
	CompletedAt     string `json:"completed_at,omitempty"`
}

// ErrorResponse is the response for errors.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// RegisterRoutes registers the API routes.
func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// Job endpoints
	api.Post("/jobs", h.CreateJob)
	api.Get("/jobs", h.ListJobs)
	api.Get("/jobs/:id", h.GetJob)
}

// CreateJob handles POST /api/v1/jobs
func (h *Handler) CreateJob(c *fiber.Ctx) error {
	var req CreateJobRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Create job (recorded and enqueued by the worker module)
	j, err := h.service.CreateJob(c.UserContext(), req.Type, req.Payload, req.Priority)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidJobType):
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "invalid_job_type",
				Message: "Invalid job type. Valid types: email, image_processing, report_generation",
			})
		case errors.Is(err, ErrInvalidPayload):
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "invalid_payload",
				Message: err.Error(),
			})
		case errors.Is(err, ErrQueueUnavailable):
			return c.Status(fiber.StatusServiceUnavailable).JSON(ErrorResponse{
				Error:   "queue_unavailable",
				Message: "Job queue is currently unavailable",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to create job",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(h.toJobResponse(j))
}

// GetJob handles GET /api/v1/jobs/:id
func (h *Handler) GetJob(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "invalid_request",
			Message: "Job ID is required",
		})
	}

	j, err := h.service.GetJob(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error:   "not_found",
				Message: "Job not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve job",
		})
	}

	return c.JSON(h.toJobResponse(j))
}

// ListJobs handles GET /api/v1/jobs
func (h *Handler) ListJobs(c *fiber.Ctx) error {
	// Parse query parameters
	status := c.Query("status")
	jobType := c.Query("type")

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	// List jobs
	jobs, err := h.service.ListJobs(c.UserContext(), status, jobType, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list jobs",
		})
	}

	// Convert to response
	response := make([]*JobResponse, len(jobs))
	for i := range jobs {
		response[i] = h.toJobResponse(&jobs[i])
	}

	return c.JSON(fiber.Map{
		"jobs":   response,
		"count":  len(response),
		"limit":  limit,
		"offset": offset,
	})
}

// RFC3339 is the standard time format used for API responses.
const RFC3339 = "2006-01-02T15:04:05Z07:00"

// toJobResponse converts a job to its response format.
func (h *Handler) toJobResponse(j *workercontract.Job) *JobResponse {
	resp := &JobResponse{
		ID:              j.ID,
		Type:            j.Type,
		Status:          j.Status,
		Priority:        j.Priority,
		Payload:         j.Payload,
		Result:          j.Result,
		Error:           j.Error,
		Progress:        j.Progress,
		ProgressMessage: j.ProgressMessage,
		RetryCount:      j.RetryCount,
		MaxRetries:      j.MaxRetries,
		WorkerID:        j.WorkerID,
		CreatedAt:       j.CreatedAt.Format(RFC3339),
		UpdatedAt:       j.UpdatedAt.Format(RFC3339),
	}

	if j.StartedAt != nil && !j.StartedAt.IsZero() {
		resp.StartedAt = j.StartedAt.Format(RFC3339)
	}

	if j.CompletedAt != nil && !j.CompletedAt.IsZero() {
		resp.CompletedAt = j.CompletedAt.Format(RFC3339)
	}

	return resp
}
