package user

import (
	"api-server/internal/pkg/clock"
	"context"
	"log/slog"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/password"
)

// UserService provides user management functionality with comprehensive business operations
type UserService struct {
	// Repositories
	UserRepo  domain.UserRepository
	AuditRepo domain.AuditLogRepository

	// Infrastructure
	EventBus domain.EventBus

	// Configuration
	hashConfig HashConfig
	hashSecret string
	hashSalt   string

	// Utilities
	passwordValidator *password.PasswordValidator
	businessContext   *BusinessContextMapper
	metricsCalculator *MetricsCalculator

	// Logger
	logger *slog.Logger
}

// HashConfig defines password hashing parameters
type HashConfig struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// NewUserService creates a new UserService instance with all dependencies
func NewUserService(
	userRepo domain.UserRepository,
	auditRepo domain.AuditLogRepository,
	eventBus domain.EventBus,
	hashSecret, hashSalt string,
) *UserService {
	logger := observability.GetLogger()

	hashConfig := DefaultHashConfig()

	businessContext := NewBusinessContextMapper()
	metricsCalculator := NewMetricsCalculator(auditRepo, logger)

	service := &UserService{
		UserRepo:          userRepo,
		AuditRepo:         auditRepo,
		EventBus:          eventBus,
		hashConfig:        hashConfig,
		hashSecret:        hashSecret,
		hashSalt:          hashSalt,
		passwordValidator: password.NewPasswordValidator(),
		businessContext:   businessContext,
		metricsCalculator: metricsCalculator,
		logger:            logger,
	}

	logger.Info("UserService initialized",
		"hash_memory", hashConfig.Memory,
		"hash_iterations", hashConfig.Iterations,
		"hash_parallelism", hashConfig.Parallelism)

	return service
}

// GetUserSummary returns user statistics summary for admin dashboard
func (s *UserService) GetUserSummary(ctx context.Context) (*dto.UserSummaryResponse, error) {
	s.logger.Info("Getting user summary")

	// Get total user count
	totalUsers, err := s.UserRepo.Count(ctx)
	if err != nil {
		s.logger.Error("Failed to count total users", "error", err)
		return nil, err
	}

	// Get admin count
	totalAdmins, err := s.UserRepo.CountByRole(ctx, domain.RoleAdmin)
	if err != nil {
		s.logger.Error("Failed to count admin users", "error", err)
		return nil, err
	}

	// Get partner count
	totalPartners, err := s.UserRepo.CountByRole(ctx, domain.RolePartner)
	if err != nil {
		s.logger.Error("Failed to count partner users", "error", err)
		return nil, err
	}

	// Get recent logins today (since start of today)
	startOfToday := clock.NowUTC().Truncate(24 * time.Hour)
	recentLoginsToday, err := s.UserRepo.CountRecentLogins(ctx, startOfToday)
	if err != nil {
		s.logger.Error("Failed to count recent logins", "error", err)
		return nil, err
	}

	// Get total employees (users with role 'employee')
	totalEmployees, err := s.UserRepo.CountByRole(ctx, domain.RoleEmployee)
	if err != nil {
		s.logger.Error("Failed to count employee users", "error", err)
		return nil, err
	}

	response := &dto.UserSummaryResponse{
		TotalUsers:        totalUsers,
		TotalAdmins:       totalAdmins,
		TotalPartners:     totalPartners,
		TotalEmployees:    totalEmployees,
		RecentLoginsToday: recentLoginsToday,
	}

	s.logger.Info("User summary retrieved successfully",
		"total_users", totalUsers,
		"total_admins", totalAdmins,
		"total_partners", totalPartners,
		"total_employees", totalEmployees,
		"recent_logins_today", recentLoginsToday)

	return response, nil
}
