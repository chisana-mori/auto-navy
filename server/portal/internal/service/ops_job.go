// Package service provides the business logic for the portal module.
package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"navy-ng/models/portal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// OpsJobService provides operations for OpsJob.
type OpsJobService struct {
	db            *gorm.DB
	activeJobs    map[int64]*JobExecution
	activeJobsMux sync.Mutex
}

// JobExecution represents an active job execution
type JobExecution struct {
	JobID     int64
	Clients   map[*websocket.Conn]bool
	ClientMux sync.Mutex
	Cancel    context.CancelFunc
}

// NewOpsJobService creates a new OpsJobService.
func NewOpsJobService(db *gorm.DB) *OpsJobService {
	return &OpsJobService{
		db:         db,
		activeJobs: make(map[int64]*JobExecution),
	}
}

// Constants for OpsJob service
const (
	// ErrOpsJobNotFoundMsg is the error message for record not found errors.
	ErrOpsJobNotFoundMsg = "Operation job with id %d not found"

	// Job status constants
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// GetOpsJob retrieves a single OpsJob by ID.
// @Summary Get operation job information by ID
// @Description Get detailed information of an operation job by its ID
// @Tags OpsJob
// @Accept json
// @Produce json
// @Param id path int true "Operation Job ID"
// @Success 200 {object} OpsJobResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ops/job/{id} [get]
func (s *OpsJobService) GetOpsJob(ctx context.Context, id int64) (*OpsJobResponse, error) {
	var model portal.OpsJob
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = ?", id, emptyString).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf(ErrOpsJobNotFoundMsg, id)
		}
		return nil, fmt.Errorf("failed to get operation job: %w", err)
	}

	return toOpsJobResponse(&model), nil
}

// ListOpsJobs retrieves a list of OpsJobs based on query parameters.
// @Summary List operation jobs
// @Description Get a list of operation jobs with filtering and pagination
// @Tags OpsJob
// @Accept json
// @Produce json
// @Param query query OpsJobQuery true "Query parameters"
// @Success 200 {object} OpsJobListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ops/job [get]
func (s *OpsJobService) ListOpsJobs(ctx context.Context, query *OpsJobQuery) (*OpsJobListResponse, error) {
	var models []portal.OpsJob
	var total int64

	db := s.db.WithContext(ctx).Model(&portal.OpsJob{}).Where("deleted = ?", emptyString)

	// Apply filters
	if query.Name != emptyString {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Status != emptyString {
		db = db.Where("status = ?", query.Status)
	}

	// Count total records
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count operation jobs: %w", err)
	}

	// Adjust pagination
	if query.Page <= 0 {
		query.Page = defaultPage
	}
	if query.Size <= 0 || query.Size > maxSize {
		query.Size = defaultSize
	}

	// Fetch data with pagination
	err := db.Order("id DESC").
		Offset((query.Page - 1) * query.Size).Limit(query.Size).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list operation jobs: %w", err)
	}

	list := make([]*OpsJobResponse, 0, len(models))
	for i := range models {
		list = append(list, toOpsJobResponse(&models[i]))
	}

	return &OpsJobListResponse{
		List:  list,
		Page:  query.Page,
		Size:  query.Size,
		Total: total,
	}, nil
}

// CreateOpsJob creates a new OpsJob.
// @Summary Create a new operation job
// @Description Create a new operation job with the provided information
// @Tags OpsJob
// @Accept json
// @Produce json
// @Param job body OpsJobCreateDTO true "Operation Job information"
// @Success 200 {object} OpsJobResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ops/job [post]
func (s *OpsJobService) CreateOpsJob(ctx context.Context, dto *OpsJobCreateDTO) (*OpsJobResponse, error) {
	// 确保使用数据库自增ID，忽略前端可能传递的ID字段
	now := time.Now()
	model := &portal.OpsJob{
		BaseModel:   portal.BaseModel{ID: 0}, // 显式设置ID为0，确保使用数据库自增ID
		Name:        dto.Name,
		Description: dto.Description,
		Status:      StatusPending,
		Progress:    0,
		StartTime:   now,
		EndTime:     now,
		LogContent:  "Job created and waiting to start...\n",
	}

	if err := s.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, fmt.Errorf("failed to create operation job: %w", err)
	}

	return toOpsJobResponse(model), nil
}

// StartJob starts the execution of a job and broadcasts updates via WebSocket.
func (s *OpsJobService) StartJob(jobID int64, client *websocket.Conn) error {
	// Check if job exists
	var job portal.OpsJob
	if err := s.db.Where("id = ? AND deleted = ?", jobID, emptyString).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf(ErrOpsJobNotFoundMsg, jobID)
		}
		return fmt.Errorf("failed to get operation job: %w", err)
	}

	// Check if job is already running
	s.activeJobsMux.Lock()
	jobExec, exists := s.activeJobs[jobID]
	if exists {
		// Add this client to existing job
		jobExec.ClientMux.Lock()
		jobExec.Clients[client] = true
		jobExec.ClientMux.Unlock()
		s.activeJobsMux.Unlock()
		return nil
	}

	// Create a new job execution context
	ctx, cancel := context.WithCancel(context.Background())
	jobExec = &JobExecution{
		JobID:   jobID,
		Clients: make(map[*websocket.Conn]bool),
		Cancel:  cancel,
	}
	jobExec.Clients[client] = true
	s.activeJobs[jobID] = jobExec
	s.activeJobsMux.Unlock()

	// Update job status to running
	job.Status = StatusRunning
	job.LogContent += "Job execution started...\n"
	s.db.Save(&job)

	// Start job execution in a goroutine
	go s.executeJob(ctx, jobID)

	return nil
}

// RegisterClient registers a WebSocket client for job updates
func (s *OpsJobService) RegisterClient(jobID int64, client *websocket.Conn) error {
	s.activeJobsMux.Lock()
	defer s.activeJobsMux.Unlock()

	jobExec, exists := s.activeJobs[jobID]
	if !exists {
		// Job is not running, check if it exists
		var job portal.OpsJob
		if err := s.db.Where("id = ? AND deleted = ?", jobID, emptyString).First(&job).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf(ErrOpsJobNotFoundMsg, jobID)
			}
			return fmt.Errorf("failed to get operation job: %w", err)
		}

		// Send the current status immediately
		update := OpsJobStatusUpdate{
			ID:       jobID,
			Status:   job.Status,
			Progress: job.Progress,
			Message:  "Connected to job monitoring",
		}
		client.WriteJSON(update)
		return nil
	}

	// Add client to active job
	jobExec.ClientMux.Lock()
	jobExec.Clients[client] = true
	jobExec.ClientMux.Unlock()

	return nil
}

// UnregisterClient removes a WebSocket client
func (s *OpsJobService) UnregisterClient(jobID int64, client *websocket.Conn) {
	s.activeJobsMux.Lock()
	defer s.activeJobsMux.Unlock()

	jobExec, exists := s.activeJobs[jobID]
	if !exists {
		return
	}

	jobExec.ClientMux.Lock()
	delete(jobExec.Clients, client)
	jobExec.ClientMux.Unlock()
}

// executeJob simulates a job execution with progress updates
func (s *OpsJobService) executeJob(ctx context.Context, jobID int64) {
	// Simulate job execution with random progress updates
	totalSteps := 10
	logLines := []string{
		"Initializing job execution...",
		"Connecting to AWX server...",
		"Preparing deployment environment...",
		"Validating configuration...",
		"Starting deployment process...",
		"Deploying application components...",
		"Running database migrations...",
		"Configuring network settings...",
		"Running post-deployment checks...",
		"Finalizing deployment...",
	}

	for step := 0; step < totalSteps; step++ {
		select {
		case <-ctx.Done():
			// Job was cancelled
			s.updateJobStatus(jobID, StatusFailed, (step*100)/totalSteps, "Job execution cancelled")
			s.cleanupJob(jobID)
			return
		default:
			// Update progress
			progress := ((step + 1) * 100) / totalSteps
			logLine := logLines[step]
			s.updateJobStatus(jobID, StatusRunning, progress, logLine)

			// Simulate work
			sleepTime := time.Duration(2+rand.Intn(3)) * time.Second
			time.Sleep(sleepTime)
		}
	}

	// Job completed successfully
	s.updateJobStatus(jobID, StatusCompleted, 100, "Job execution completed successfully")
	s.cleanupJob(jobID)
}

// updateJobStatus updates the job status and notifies all connected clients
func (s *OpsJobService) updateJobStatus(jobID int64, status string, progress int, logLine string) {
	// Update database
	var job portal.OpsJob
	if err := s.db.First(&job, jobID).Error; err != nil {
		return
	}

	job.Status = status
	job.Progress = progress
	job.LogContent += logLine + "\n"
	if status == StatusCompleted || status == StatusFailed {
		job.EndTime = time.Now()
	}
	s.db.Save(&job)

	// Notify clients
	update := OpsJobStatusUpdate{
		ID:       jobID,
		Status:   status,
		Progress: progress,
		Message:  fmt.Sprintf("Job %s - Progress: %d%%", status, progress),
		LogLine:  logLine,
	}

	s.activeJobsMux.Lock()
	jobExec, exists := s.activeJobs[jobID]
	s.activeJobsMux.Unlock()

	if !exists {
		return
	}

	jobExec.ClientMux.Lock()
	for client := range jobExec.Clients {
		// Non-blocking send to avoid deadlocks if client is slow
		go func(c *websocket.Conn, u OpsJobStatusUpdate) {
			c.WriteJSON(u)
		}(client, update)
	}
	jobExec.ClientMux.Unlock()
}

// cleanupJob removes the job from active jobs map
func (s *OpsJobService) cleanupJob(jobID int64) {
	s.activeJobsMux.Lock()
	defer s.activeJobsMux.Unlock()

	if jobExec, exists := s.activeJobs[jobID]; exists {
		jobExec.ClientMux.Lock()
		for client := range jobExec.Clients {
			// Send final message
			client.WriteJSON(OpsJobStatusUpdate{
				ID:       jobID,
				Status:   "completed",
				Progress: 100,
				Message:  "Job execution finished",
			})
		}
		jobExec.ClientMux.Unlock()
		delete(s.activeJobs, jobID)
	}
}
