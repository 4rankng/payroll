package user

import (
	"api-server/internal/pkg/clock"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"api-server/internal/app/dto"
	infraServices "api-server/internal/app/services/infrastructure"
	"api-server/internal/infra/observability"
)

var (
	ErrJobAlreadyRunning = errors.New("job đang chạy, vui lòng thử lại sau")
	ErrJobNotFound       = errors.New("không tìm thấy job")
)

const (
	passwordResetJobKey = "password_reset_job"
	jobResultTTL        = 15 * time.Minute
)

type PasswordResetJob struct {
	mu        sync.RWMutex
	status    dto.JobStatus
	result    *dto.ResetFirstTimeLoginPasswordResponse
	startedAt time.Time
}

type PasswordResetJobManager struct {
	job         *PasswordResetJob
	userService *UserService
	cache       *infraServices.CacheService
}

func NewPasswordResetJobManager(userService *UserService, cache *infraServices.CacheService) *PasswordResetJobManager {
	return &PasswordResetJobManager{
		userService: userService,
		cache:       cache,
		job: &PasswordResetJob{
			status: dto.JobStatusCompleted, // Start as completed with nil result (no job run yet)
		},
	}
}

// StartJob starts the password reset job in background
// Returns error if a job is already running
func (m *PasswordResetJobManager) StartJob(ctx context.Context) error {
	m.job.mu.Lock()
	defer m.job.mu.Unlock()

	// Check if job is already running (in-memory)
	if m.job.status == dto.JobStatusProcessing {
		return ErrJobAlreadyRunning
	}

	// Reset job state
	m.job.status = dto.JobStatusProcessing
	m.job.result = nil
	m.job.startedAt = clock.Now()

	// Store processing status in Redis
	if err := m.storeJobStatus(ctx, dto.JobStatusProcessing, nil); err != nil {
		observability.GetLogger().Error("Failed to store job status in Redis", "error", err)
	}

	// Start job in background
	go m.runJob(context.Background())

	return nil
}

// GetStatus returns the current job status
func (m *PasswordResetJobManager) GetStatus() dto.JobStatus {
	// First try to load from Redis
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status, _, err := m.loadJobStatus(ctx)
	if err == nil && status != "" {
		return status
	}

	// Fallback to in-memory status
	m.job.mu.RLock()
	defer m.job.mu.RUnlock()
	return m.job.status
}

// GetResult returns the job result (nil if still processing or not found)
func (m *PasswordResetJobManager) GetResult() *dto.ResetFirstTimeLoginPasswordResponse {
	// First try to load from Redis
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, result, err := m.loadJobStatus(ctx)
	if err == nil && result != nil {
		return result
	}

	// Fallback to in-memory result
	m.job.mu.RLock()
	defer m.job.mu.RUnlock()
	return m.job.result
}

// JobExists checks if a job result exists in Redis or in memory
func (m *PasswordResetJobManager) JobExists() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	exists, _ := m.cache.Exists(ctx, passwordResetJobKey)
	if exists {
		return true
	}

	// Check in-memory if job has a result
	m.job.mu.RLock()
	defer m.job.mu.RUnlock()
	return m.job.result != nil
}

// storeJobStatus stores the job status and result in Redis
func (m *PasswordResetJobManager) storeJobStatus(ctx context.Context, status dto.JobStatus, result *dto.ResetFirstTimeLoginPasswordResponse) error {
	jobData := map[string]interface{}{
		"status": status,
	}

	if result != nil {
		jobData["total"] = result.Total
		jobData["usernames"] = result.Usernames
	}

	return m.cache.Set(ctx, passwordResetJobKey, jobData, jobResultTTL)
}

// loadJobStatus loads the job status and result from Redis
func (m *PasswordResetJobManager) loadJobStatus(ctx context.Context) (dto.JobStatus, *dto.ResetFirstTimeLoginPasswordResponse, error) {
	var jobData struct {
		Status    dto.JobStatus `json:"status"`
		Total     int           `json:"total"`
		Usernames []string      `json:"usernames"`
	}

	err := m.cache.Get(ctx, passwordResetJobKey, &jobData)
	if err != nil {
		return "", nil, fmt.Errorf("job not found in cache")
	}

	result := &dto.ResetFirstTimeLoginPasswordResponse{
		Total:     jobData.Total,
		Usernames: jobData.Usernames,
	}

	return jobData.Status, result, nil
}

// runJob executes the password reset operation
func (m *PasswordResetJobManager) runJob(ctx context.Context) {
	logger := observability.GetLogger()
	logger.Info("Starting password reset job for first-time users")

	defaultPassword := "Vfic@1234"

	result, err := m.userService.ResetFirstTimeLoginPasswords(ctx, defaultPassword)
	if err != nil {
		logger.Error("Password reset job failed", "error", err)
		// Mark as completed with empty result on error
		m.job.mu.Lock()
		m.job.status = dto.JobStatusCompleted
		m.job.result = &dto.ResetFirstTimeLoginPasswordResponse{
			Total:     0,
			Usernames: []string{},
		}
		m.job.mu.Unlock()

		// Store error result in Redis
		if storeErr := m.storeJobStatus(context.Background(), dto.JobStatusCompleted, m.job.result); storeErr != nil {
			logger.Error("Failed to store job error result in Redis", "error", storeErr)
		}
		return
	}

	logger.Info("Password reset job completed", "total", result.Total, "usernames", result.Usernames)

	// Mark as completed with result
	m.job.mu.Lock()
	m.job.status = dto.JobStatusCompleted
	m.job.result = result
	m.job.mu.Unlock()

	// Store success result in Redis
	if storeErr := m.storeJobStatus(context.Background(), dto.JobStatusCompleted, result); storeErr != nil {
		logger.Error("Failed to store job result in Redis", "error", storeErr)
	}
}
