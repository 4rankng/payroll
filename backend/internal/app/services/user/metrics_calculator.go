package user

import (
	"context"
	"log/slog"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

// MetricsCalculator handles all business metrics calculations for user activities
type MetricsCalculator struct {
	auditRepo domain.AuditLogRepository
	logger    *slog.Logger
}

// NewMetricsCalculator creates a new metrics calculator with dependencies
func NewMetricsCalculator(auditRepo domain.AuditLogRepository, logger *slog.Logger) *MetricsCalculator {
	return &MetricsCalculator{
		auditRepo: auditRepo,
		logger:    logger,
	}
}

// CalculateAuthenticationMetrics computes authentication-related metrics
func (mc *MetricsCalculator) CalculateAuthenticationMetrics(
	ctx context.Context,
	userID uint,
	periodStart time.Time,
	lastLogin *time.Time,
	auditLogs []*domain.AuditLog,
) (dto.AuthenticationMetrics, error) {
	mc.logger.Info("Calculating authentication metrics", "user_id", userID, "period_start", periodStart)

	// Count successful logins using repository (legacy filters no longer work)
	totalLogins := int64(0)

	// Analyze audit logs for authentication metrics
	for _, log := range auditLogs {
		message := log.Message
		if contains(message, []string{"đã đăng nhập", "đăng nhập vào hệ thống"}) {
			totalLogins++
		}
	}

	metrics := dto.AuthenticationMetrics{
		TotalLogins: totalLogins,
		LastLogin:   lastLogin,
	}

	mc.logger.Info("Authentication metrics calculated",
		"user_id", userID,
		"total_logins", totalLogins)

	return metrics, nil
}

// CalculatePayrollOperationMetrics computes payroll-specific operation metrics
func (mc *MetricsCalculator) CalculatePayrollOperationMetrics(auditLogs []*domain.AuditLog) dto.PayrollOperationMetrics {
	timesheetsManaged := int64(0)

	for _, log := range auditLogs {
		message := log.Message
		if contains(message, []string{"chấm công", "bản ghi chấm công"}) {
			timesheetsManaged++
		}
	}

	return dto.PayrollOperationMetrics{
		TimesheetsManaged: timesheetsManaged,
	}
}
