package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// PaytypeConstructionService handles business logic for constructing paytype paths
type PaytypeConstructionService struct {
	projectEmployeeRepo domain.ProjectEmployeeRepository
}

// NewPaytypeConstructionService creates a new paytype construction service
func NewPaytypeConstructionService(
	projectEmployeeRepo domain.ProjectEmployeeRepository,
) *PaytypeConstructionService {
	return &PaytypeConstructionService{
		projectEmployeeRepo: projectEmployeeRepo,
	}
}

// ConstructPaytypeFromAssignment constructs the paytype path from a pre-fetched assignment.
// This is a pure computation — no DB call — intended for use in bulk operations
// where assignments have already been batch-loaded into a cache.
// Returns: position.dayType.hourType (all lowercase)
func (s *PaytypeConstructionService) ConstructPaytypeFromAssignment(assignment *domain.ProjectEmployee, date time.Time, hourType, dayType string) (string, error) {
	position := assignment.Position
	if position == "" {
		position = "phổ thông"
	}

	if dayType == "" {
		switch date.Weekday() {
		case time.Sunday, time.Saturday:
			dayType = "ngày nghỉ"
		default:
			dayType = "ngày thường"
		}
	}

	return strings.ToLower(position) + "." + strings.ToLower(dayType) + "." + strings.ToLower(hourType), nil
}

// ConstructPaytype constructs the paytype path for a timesheet entry
// Returns: position.dayType.hourType (all lowercase)
func (s *PaytypeConstructionService) ConstructPaytype(ctx context.Context, projectID uint, employeeID uint, date time.Time, hourType string, dayType string) (string, error) {
	logger := observability.GetLogger()
	logger.Info("PaytypeConstruction: ConstructPaytype called",
		"project_id", projectID,
		"employee_id", employeeID,
		"date", date.Format("2006-01-02"),
		"hour_type", hourType,
		"day_type", dayType)

	// 1. Get employee assignment to determine position (skill level)
	logger.Info("PaytypeConstruction: Getting employee assignment",
		"project_id", projectID,
		"employee_id", employeeID)
	assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, projectID, employeeID)
	if err != nil {
		logger.Error("PaytypeConstruction: Failed to get employee assignment",
			"error", err.Error(),
			"error_type", fmt.Sprintf("%T", err))
		return "", domain.NewValidationError("nhân viên chưa được phân công vào dự án này")
	}

	lastDate := "nil"
	if assignment.LastDate != nil {
		lastDate = assignment.LastDate.Format("2006-01-02")
	}
	logger.Info("PaytypeConstruction: Employee assignment found",
		"position", assignment.Position,
		"start_date", assignment.StartDate.Format("2006-01-02"),
		"last_date", lastDate)

	// 2. Determine position (skill level) - use position from assignment or default
	position := assignment.Position
	if position == "" {
		logger.Info("PaytypeConstruction: No position found, using default", "default_position", "phổ thông")
		position = "phổ thông" // Default position
	} else {
		logger.Info("PaytypeConstruction: Using position from assignment", "position", position)
	}

	// 3. Determine day type if not provided
	if dayType == "" {
		// Simple day type determination logic
		weekday := date.Weekday()
		switch weekday {
		case time.Sunday, time.Saturday:
			dayType = "ngày nghỉ"
		default:
			dayType = "ngày thường"
		}
		logger.Info("PaytypeConstruction: Day type determined from weekday",
			"weekday", weekday,
			"day_type", dayType)
	} else {
		logger.Info("PaytypeConstruction: Using provided day type", "day_type", dayType)
	}

	// 4. Construct paytype: position.dayType.hourType (all lowercase for consistency)
	payType := strings.ToLower(position) + "." + strings.ToLower(dayType) + "." + strings.ToLower(hourType)

	logger.Info("PaytypeConstruction: Constructed paytype",
		"paytype", payType,
		"position", position,
		"daytype", dayType,
		"hourtype", hourType)

	logger.Info("PaytypeConstruction: ConstructPaytype completed successfully")
	return payType, nil
}
