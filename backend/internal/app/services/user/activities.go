package user

import (
	"api-server/internal/constants"
	"api-server/internal/pkg/clock"
	"context"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

// GetUserActivities returns comprehensive payroll-specific user activity summary (Admin only)
func (s *UserService) GetUserActivities(ctx context.Context, userID uint, days int) (*dto.UserActivitiesResponse, error) {
	s.logger.Info("Getting user activities", "user_id", userID, "period_days", days)

	// Check if user exists
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Info("User not found for activities", "user_id", userID, "error", err)
		return nil, err
	}

	// Calculate date range
	periodStart := clock.Now().AddDate(0, 0, -days)

	// Get all audit logs for analysis
	filters := domain.AuditFilters{
		UserID:    &userID,
		FromDate:  &periodStart,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	auditLogs, err := s.AuditRepo.List(ctx, filters)
	if err != nil {
		s.logger.Error("Failed to get user audit logs", "error", err, "user_id", userID)
		return nil, domain.NewInternalError(constants.MsgFailedToGetUserActivitiesVN, err)
	}

	s.logger.Info("Retrieved audit logs for analysis",
		"user_id", userID,
		"logs_count", len(auditLogs),
		"period_start", periodStart)

	// Calculate authentication metrics
	authMetrics, err := s.metricsCalculator.CalculateAuthenticationMetrics(ctx, userID, periodStart, user.LastLogin, auditLogs)
	if err != nil {
		s.logger.Error("Failed to calculate authentication metrics", "error", err, "user_id", userID)
		return nil, err
	}

	// Calculate payroll metrics
	payrollMetrics := s.metricsCalculator.CalculatePayrollOperationMetrics(auditLogs)

	// Format recent activities with business context (limit to 50)
	recentActivities := s.formatRecentActivitiesWithContext(auditLogs, 50)

	response := &dto.UserActivitiesResponse{
		UserID:            userID,
		PeriodDays:        days,
		Authentication:    authMetrics,
		PayrollOperations: payrollMetrics,
		RecentActivities:  recentActivities,
	}

	s.logger.Info("User activities calculated successfully",
		"user_id", userID,
		"period_days", days,
		"total_logins", authMetrics.TotalLogins,
		"timesheets_managed", payrollMetrics.TimesheetsManaged)

	return response, nil
}

// formatRecentActivitiesWithContext formats recent activities with business context using efficient map lookups
func (s *UserService) formatRecentActivitiesWithContext(auditLogs []*domain.AuditLog, limit int) []dto.UserActivityDetailResponse {
	s.logger.Info("Formatting recent activities with context", "total_logs", len(auditLogs), "limit", limit)

	activities := make([]dto.UserActivityDetailResponse, 0, limit)

	count := 0
	for _, log := range auditLogs {
		if count >= limit {
			break
		}

		// Use efficient map-based business context generation
		_ = s.businessContext.GetBusinessContext(log) // Business context is now in log.Message
		_ = s.businessContext.GetImpactLevel(log)     // Impact level is no longer used

		activities = append(activities, dto.UserActivityDetailResponse{
			ID:        log.ID,
			Message:   log.Message,
			CreatedAt: log.CreatedAt,
		})
		count++
	}

	s.logger.Info("Recent activities formatted successfully",
		"formatted_count", len(activities),
		"requested_limit", limit)

	return activities
}
