// Package service provides the business logic for the portal module.
package service

import (
	"context"
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

// ClientConnection represents a WebSocket client connection with its own mutex
type ClientConnection struct {
	Conn     *websocket.Conn
	WriteMux sync.Mutex
}

// JobExecution represents an active job execution
type JobExecution struct {
	JobID   int64
	Manager *WebSocketManager
	Cancel  context.CancelFunc
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
)

// GetOpsJob retrieves a single OpsJob by ID.

func (s *OpsJobService) GetOpsJob(ctx context.Context, id int64) (*OpsJobResponse, error) {
	var model portal.OpsJob
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = ?", id, EmptyString).First(&model).Error
	if err != nil {
		return nil, HandleDBError(err, ResourceOpsJob, id)
	}
	return toOpsJobResponse(&model), nil
}

// ListOpsJobs retrieves a list of OpsJobs based on query parameters.

func (s *OpsJobService) ListOpsJobs(ctx context.Context, query *OpsJobQuery) (*OpsJobListResponse, error) {
	var models []portal.OpsJob
	var total int64

	db := s.db.WithContext(ctx).Model(&portal.OpsJob{}).Where("deleted = ?", EmptyString)

	// Apply filters
	if query.Name != EmptyString {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Status != EmptyString {
		db = db.Where("status = ?", query.Status)
	}

	// Count total records
	if err := db.Count(&total).Error; err != nil {
		return nil, NewServerError("failed to count operation jobs", err)
	}

	// Adjust pagination
	query.AdjustPagination()

	// Fetch data with pagination
	err := db.Order("id DESC").
		Offset(query.GetOffset()).Limit(query.Size).
		Find(&models).Error
	if err != nil {
		return nil, NewServerError("failed to list operation jobs", err)
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

func (s *OpsJobService) CreateOpsJob(ctx context.Context, dto *OpsJobCreateDTO) (*OpsJobResponse, error) {
	now := time.Now()
	model := &portal.OpsJob{
		BaseModel:   portal.BaseModel{ID: 0},
		Name:        dto.Name,
		Description: dto.Description,
		Status:      StatusPending,
		Progress:    0,
		StartTime:   now,
		EndTime:     now,
		LogContent:  "Job created and waiting to start...\n",
	}

	if err := s.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, NewServerError("failed to create operation job", err)
	}

	return toOpsJobResponse(model), nil
}

// StartJob starts the execution of a job and broadcasts updates via WebSocket.
func (s *OpsJobService) StartJob(jobID int64, conn *websocket.Conn) error {
	// Check if job exists
	var job portal.OpsJob
	if err := s.db.Where("id = ? AND deleted = ?", jobID, EmptyString).First(&job).Error; err != nil {
		return HandleDBError(err, ResourceOpsJob, jobID)
	}

	client := NewWebSocketClient(conn)

	// Check if job is already running
	s.activeJobsMux.Lock()
	jobExec, exists := s.activeJobs[jobID]
	if exists {
		// Add this client to existing job
		jobExec.Manager.AddClient(client)
		s.activeJobsMux.Unlock()
		return nil
	}

	// Create a new job execution context
	ctx, cancel := context.WithCancel(context.Background())
	jobExec = &JobExecution{
		JobID:   jobID,
		Manager: NewWebSocketManager(),
		Cancel:  cancel,
	}
	jobExec.Manager.AddClient(client)
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
func (s *OpsJobService) RegisterClient(jobID int64, conn *websocket.Conn) error {
	client := NewWebSocketClient(conn)

	s.activeJobsMux.Lock()
	defer s.activeJobsMux.Unlock()

	jobExec, exists := s.activeJobs[jobID]
	if !exists {
		// Job is not running, check if it exists
		var job portal.OpsJob
		if err := s.db.Where("id = ? AND deleted = ?", jobID, EmptyString).First(&job).Error; err != nil {
			return HandleDBError(err, ResourceOpsJob, jobID)
		}

		// Send the current status immediately
		update := OpsJobStatusUpdate{
			ID:       jobID,
			Status:   job.Status,
			Progress: job.Progress,
			Message:  "Connected to job monitoring",
		}
		return client.SafeWrite(update)
	}

	// Add client to active job
	jobExec.Manager.AddClient(client)
	return nil
}

// UnregisterClient removes a WebSocket client
func (s *OpsJobService) UnregisterClient(jobID int64, conn *websocket.Conn) {
	s.activeJobsMux.Lock()
	defer s.activeJobsMux.Unlock()

	jobExec, exists := s.activeJobs[jobID]
	if !exists {
		return
	}

	// Find and remove the client
	for client := range jobExec.Manager.Clients {
		if client.Conn == conn {
			jobExec.Manager.RemoveClient(client)
			break
		}
	}
}

// executeJob simulates a job execution with progress updates
func (s *OpsJobService) executeJob(ctx context.Context, jobID int64) {
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
			s.updateJobStatus(jobID, StatusFailed, (step*100)/totalSteps, "Job execution cancelled")
			s.cleanupJob(jobID)
			return
		default:
			progress := ((step + 1) * 100) / totalSteps
			logLine := logLines[step]
			s.updateJobStatus(jobID, StatusRunning, progress, logLine)

			sleepTime := time.Duration(2+rand.Intn(3)) * time.Second
			time.Sleep(sleepTime)
		}
	}

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

	if exists {
		jobExec.Manager.BroadcastMessage(update)
	}
}

// cleanupJob removes the job from active jobs map
func (s *OpsJobService) cleanupJob(jobID int64) {
	s.activeJobsMux.Lock()
	defer s.activeJobsMux.Unlock()

	if jobExec, exists := s.activeJobs[jobID]; exists {
		jobExec.Manager.BroadcastMessage(OpsJobStatusUpdate{
			ID:       jobID,
			Status:   StatusCompleted,
			Progress: 100,
			Message:  "Job execution finished",
		})
		delete(s.activeJobs, jobID)
	}
}
