package api

import (
	"strconv"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/gofiber/fiber/v2"
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
	Type     string      `json:"type"`
	Payload  interface{} `json:"payload"`
	Priority int         `json:"priority,omitempty"`
}

// JobResponse is the response for a single job.
type JobResponse struct {
	ID              string      `json:"id"`
	Type            string      `json:"type"`
	Status          string      `json:"status"`
	Priority        int         `json:"priority"`
	Payload         interface{} `json:"payload"`
	Result          interface{} `json:"result,omitempty"`
	Error           string      `json:"error,omitempty"`
	Progress        int         `json:"progress"`
	ProgressMessage string      `json:"progress_message,omitempty"`
	RetryCount      int         `json:"retry_count"`
	MaxRetries      int         `json:"max_retries"`
	WorkerID        string      `json:"worker_id,omitempty"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
	StartedAt       string      `json:"started_at,omitempty"`
	CompletedAt     string      `json:"completed_at,omitempty"`
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

	// Convert to domain request
	jobReq := &job.CreateJobRequest{
		Type:     job.JobType(req.Type),
		Payload:  req.Payload,
		Priority: req.Priority,
	}

	// Create job
	j, err := h.service.CreateJob(c.Context(), jobReq)
	if err != nil {
		switch err {
		case job.ErrInvalidJobType:
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "invalid_job_type",
				Message: "Invalid job type. Valid types: email, image_processing, report_generation",
			})
		case job.ErrInvalidPayload:
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "invalid_payload",
				Message: err.Error(),
			})
		case job.ErrQueueUnavailable:
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

	j, err := h.service.GetJob(id)
	if err != nil {
		if err == job.ErrJobNotFound {
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
	status := job.JobStatus(c.Query("status"))
	jobType := job.JobType(c.Query("type"))

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	// List jobs
	jobs := h.service.ListJobs(status, jobType, limit, offset)

	// Convert to response
	response := make([]*JobResponse, len(jobs))
	for i, j := range jobs {
		response[i] = h.toJobResponse(j)
	}

	return c.JSON(fiber.Map{
		"jobs":   response,
		"count":  len(response),
		"limit":  limit,
		"offset": offset,
	})
}

// toJobResponse converts a job to its response format.
func (h *Handler) toJobResponse(j *job.Job) *JobResponse {
	resp := &JobResponse{
		ID:              j.ID,
		Type:            string(j.Type),
		Status:          string(j.Status),
		Priority:        j.Priority,
		Payload:         j.Payload,
		Result:          j.Result,
		Progress:        j.Progress,
		ProgressMessage: j.ProgressMessage,
		RetryCount:      j.RetryCount,
		MaxRetries:      j.MaxRetries,
		WorkerID:        j.WorkerID,
		CreatedAt:       j.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       j.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if j.Error != "" {
		resp.Error = j.Error
	}

	if j.StartedAt != nil && !j.StartedAt.IsZero() {
		resp.StartedAt = j.StartedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	if j.CompletedAt != nil && !j.CompletedAt.IsZero() {
		resp.CompletedAt = j.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return resp
}
