package converter

import (
	"context"
	"fmt"
	"strings"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

type Repositories struct {
	UserRepo     domain.UserRepository
	EmployeeRepo domain.EmployeeRepository
	ProjectRepo  domain.ProjectRepository
}

// ConvertAuditLogsToActivities converts multiple audit logs to ActivityItems efficiently
// using batch loading to avoid N+1 query problems.
func ConvertAuditLogsToActivities(logs []*domain.AuditLog, repos Repositories) ([]*dto.ActivityItem, error) {
	if len(logs) == 0 {
		return []*dto.ActivityItem{}, nil
	}

	// Collect unique user IDs
	userIDsMap := make(map[uint]bool)
	for _, log := range logs {
		userIDsMap[log.UserID] = true
	}

	// Convert map to slice
	userIDs := make([]uint, 0, len(userIDsMap))
	for id := range userIDsMap {
		userIDs = append(userIDs, id)
	}

	// Batch load all users in a single query
	userMap, err := repos.UserRepo.GetByIDs(context.Background(), userIDs)
	if err != nil {
		return nil, err
	}

	// Convert each audit log to activity item
	activities := make([]*dto.ActivityItem, 0, len(logs))
	for _, log := range logs {
		actorName := ""
		if user, exists := userMap[log.UserID]; exists && user != nil {
			actorName = strings.TrimSpace(user.Fullname)
			if actorName == "" {
				actorName = strings.TrimSpace(user.Username)
			}
		}

		// Build final human-readable message
		msg := strings.TrimSpace(log.Message)
		if actorName != "" && msg != "" {
			// Avoid duplicating actor name if it's already prefixed
			prefixed := actorName + " "
			if !strings.HasPrefix(msg, prefixed) {
				msg = fmt.Sprintf("%s %s", actorName, msg)
			}
		}

		activities = append(activities, &dto.ActivityItem{
			ID:        int(log.ID),
			Message:   msg,
			CreatedAt: log.CreatedAt,
		})
	}

	return activities, nil
}

// ConvertAuditLogToActivity converts an audit log to a minimal ActivityItem
// compatible with the dashboard recent activities API.
// Deprecated: Use ConvertAuditLogsToActivities for better performance
func ConvertAuditLogToActivity(log *domain.AuditLog, repos Repositories) *dto.ActivityItem {
	// Resolve actor name from user_id
	actorName := ""
	if user, err := repos.UserRepo.GetByID(context.Background(), log.UserID); err == nil && user != nil {
		actorName = strings.TrimSpace(user.Fullname)
		if actorName == "" {
			actorName = strings.TrimSpace(user.Username)
		}
	}

	// Build final human-readable message.
	// log.Message is a suffix like "đã tạo dự án mới".
	msg := strings.TrimSpace(log.Message)
	if actorName != "" && msg != "" {
		prefixed := actorName + " "
		if !strings.HasPrefix(msg, prefixed) {
			msg = fmt.Sprintf("%s %s", actorName, msg)
		}
	}

	return &dto.ActivityItem{
		ID:        int(log.ID),
		Message:   msg,
		CreatedAt: log.CreatedAt,
	}
}
