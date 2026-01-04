package job

import (
	"encoding/json"
	"sync"
	"time"
)

// Store provides in-memory storage for jobs.
type Store struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

// NewStore creates a new job store.
func NewStore() *Store {
	return &Store{
		jobs: make(map[string]*Job),
	}
}

// Create stores a new job.
func (s *Store) Create(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[job.ID] = job
	return nil
}

// GetByID retrieves a job by its ID.
func (s *Store) GetByID(id string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[id]
	if !exists {
		return nil, ErrJobNotFound
	}
	// Return a copy to prevent external modifications
	jobCopy := *job
	return &jobCopy, nil
}

// List retrieves jobs with optional filtering and pagination.
func (s *Store) List(status JobStatus, jobType JobType, limit, offset int) []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert map to slice and apply filters
	allJobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		// Filter by status if provided
		if status != "" && job.Status != status {
			continue
		}
		// Filter by type if provided
		if jobType != "" && job.Type != jobType {
			continue
		}
		allJobs = append(allJobs, job)
	}

	// Apply pagination
	if offset >= len(allJobs) {
		return []*Job{}
	}

	end := offset + limit
	if end > len(allJobs) {
		end = len(allJobs)
	}

	return allJobs[offset:end]
}

// Update updates an existing job.
func (s *Store) Update(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.ID]; !exists {
		return ErrJobNotFound
	}

	job.UpdatedAt = time.Now()
	s.jobs[job.ID] = job
	return nil
}

// UpdateStatus updates the status of a job.
func (s *Store) UpdateStatus(id string, status JobStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	job.Status = status
	job.UpdatedAt = time.Now()
	return nil
}

// UpdateProgress updates the progress of a job.
func (s *Store) UpdateProgress(id string, progress int, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	job.Progress = progress
	job.ProgressMessage = message
	job.UpdatedAt = time.Now()
	return nil
}

// SetStarted marks a job as started.
func (s *Store) SetStarted(id string, workerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	now := time.Now()
	job.Status = JobStatusProcessing
	job.StartedAt = &now
	job.WorkerID = workerID
	job.UpdatedAt = now
	return nil
}

// SetCompleted marks a job as completed.
func (s *Store) SetCompleted(id string, result any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	now := time.Now()
	job.Status = JobStatusCompleted
	job.Progress = 100
	if result != nil {
		job.Result, _ = json.Marshal(result)
	}
	job.CompletedAt = &now
	job.UpdatedAt = now
	return nil
}

// SetFailed marks a job as failed.
func (s *Store) SetFailed(id string, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	now := time.Now()
	job.Status = JobStatusFailed
	job.Error = errMsg
	job.UpdatedAt = now
	return nil
}

// IncrementRetry increments the retry count for a job.
func (s *Store) IncrementRetry(id string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return 0, ErrJobNotFound
	}

	job.RetryCount++
	job.Status = JobStatusPending
	job.UpdatedAt = time.Now()
	return job.RetryCount, nil
}

// SetDeadLetter marks a job as dead-letter.
func (s *Store) SetDeadLetter(id string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return ErrJobNotFound
	}

	now := time.Now()
	job.Status = JobStatusDeadLetter
	job.Error = reason
	job.CompletedAt = &now
	job.UpdatedAt = now
	return nil
}

// ListByStatus retrieves jobs with a specific status.
func (s *Store) ListByStatus(status JobStatus, offset, limit int) ([]Job, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]Job, 0)
	for _, job := range s.jobs {
		if job.Status == status {
			filtered = append(filtered, *job)
		}
	}

	total := len(filtered)

	if offset >= total {
		return []Job{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return filtered[offset:end], total, nil
}
