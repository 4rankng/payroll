package services

import (
	"context"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// EmployeeAssignmentService handles business logic for employee project assignments
type EmployeeAssignmentService struct {
	projectEmployeeRepo domain.ProjectEmployeeRepository
	employeeRepo        domain.EmployeeRepository
	projectRepo         domain.ProjectRepository
	timesheetRepo       domain.TimesheetRepository
}

// NewEmployeeAssignmentService creates a new employee assignment service
func NewEmployeeAssignmentService(
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	employeeRepo domain.EmployeeRepository,
	projectRepo domain.ProjectRepository,
	timesheetRepo domain.TimesheetRepository,
) *EmployeeAssignmentService {
	return &EmployeeAssignmentService{
		projectEmployeeRepo: projectEmployeeRepo,
		employeeRepo:        employeeRepo,
		projectRepo:         projectRepo,
		timesheetRepo:       timesheetRepo,
	}
}

// ValidateAssignment validates business rules for employee assignment
func (s *EmployeeAssignmentService) ValidateAssignment(ctx context.Context, assignment *domain.ProjectEmployee) error {
	// Check if employee exists
	_, err := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
	if err != nil {
		return domain.NewNotFoundError(constants.MsgEmployeeNotFoundVN)
	}

	// Check if project exists
	_, err = s.projectRepo.GetByID(ctx, assignment.ProjectID)
	if err != nil {
		return domain.NewNotFoundError(constants.MsgProjectNotFoundVN)
	}

	// Check for date overlaps with existing assignments
	return s.ValidateDateOverlap(ctx, assignment.EmployeeID, assignment.ProjectID, assignment.StartDate, assignment.LastDate)
}

// ValidateAssignmentForUpdate validates business rules for updating an existing assignment
// This is a convenience method that loads data and delegates to the pure validation method
func (s *EmployeeAssignmentService) ValidateAssignmentForUpdate(ctx context.Context, assignment *domain.ProjectEmployee) error {
	// Check if employee exists
	_, err := s.employeeRepo.GetByID(ctx, assignment.EmployeeID)
	if err != nil {
		return domain.NewNotFoundError(constants.MsgEmployeeNotFoundVN)
	}

	// Check if project exists
	_, err = s.projectRepo.GetByID(ctx, assignment.ProjectID)
	if err != nil {
		return domain.NewNotFoundError(constants.MsgProjectNotFoundVN)
	}

	// Validate dates (last_date must be after start_date)
	if err := assignment.ValidateDates(); err != nil {
		return err
	}

	// Load overlapping assignments
	overlapping, err := s.projectEmployeeRepo.GetOverlappingAssignments(ctx, assignment.EmployeeID, &assignment.StartDate, assignment.LastDate)
	if err != nil {
		return err
	}

	// Load counts for overlapping assignments
	overlappingCounts := make(map[uint]int64)
	for _, other := range overlapping {
		if other.ID != assignment.ID && other.ProjectID == assignment.ProjectID {
			count, err := s.timesheetRepo.CountTimesheets(ctx, other.ProjectID, other.EmployeeID)
			if err != nil {
				return err
			}
			overlappingCounts[other.ID] = count
		}
	}

	// Check for non-editable timesheets
	hasNonEditable, err := s.timesheetRepo.HasNonEditableTimesheetsAfterDate(ctx, assignment.ProjectID, assignment.EmployeeID, assignment.StartDate)
	if err != nil {
		return err
	}

	// Delegate to pure validation method
	return s.ValidateAssignmentUpdateWithData(assignment, overlapping, overlappingCounts, hasNonEditable)
}

// ValidateAssignmentUpdateWithData performs pure domain validation with pre-loaded data
// This method has no repository dependencies and can be tested easily
func (s *EmployeeAssignmentService) ValidateAssignmentUpdateWithData(
	assignment *domain.ProjectEmployee,
	overlappingAssignments []*domain.ProjectEmployee,
	overlappingTimesheetCounts map[uint]int64,
	hasNonEditableTimesheets bool,
) error {
	// Check for overlaps with OTHER assignments for the same project
	for _, other := range overlappingAssignments {
		// Skip the current assignment being updated
		if other.ID == assignment.ID {
			continue
		}

		// Only check overlaps within the same project
		if other.ProjectID == assignment.ProjectID {
			// Check if the other assignment has timesheets
			if count, exists := overlappingTimesheetCounts[other.ID]; exists && count > 0 {
				return domain.NewConflictError(constants.MsgAssignmentOverlapVN)
			}
		}
	}

	// Check if there are non-editable timesheets on or after the new start date
	if hasNonEditableTimesheets {
		return domain.NewValidationError("Không thể cập nhật phân công vì có bảng chấm công đã được duyệt hoặc thanh toán từ ngày bắt đầu mới")
	}

	return nil
}

// ValidateDateOverlap validates that new assignment doesn't overlap with existing ones for the same project
func (s *EmployeeAssignmentService) ValidateDateOverlap(ctx context.Context, employeeID, projectID uint, startDate time.Time, endDate *time.Time) error {
	overlapping, err := s.projectEmployeeRepo.GetOverlappingAssignments(ctx, employeeID, &startDate, endDate)
	if err != nil {
		return err
	}

	// Only check for overlaps within the same project - employees can work on multiple projects simultaneously
	for _, assignment := range overlapping {
		if assignment.ProjectID == projectID {
			// Check if there are any timesheets for this assignment
			timesheetCount, err := s.timesheetRepo.CountTimesheets(ctx, assignment.ProjectID, assignment.EmployeeID)
			if err != nil {
				return err
			}

			// Only reject overlap if there are timesheets attached
			if timesheetCount > 0 {
				// Calculate suggested next available start date (day after existing assignment ends)
				var suggestedDate string
				if assignment.LastDate != nil {
					suggestedDate = assignment.LastDate.AddDate(0, 0, 1).Format("2006-01-02")
				} else {
					// If existing assignment has no end date, suggest tomorrow
					suggestedDate = clock.Now().AddDate(0, 0, 1).Format("2006-01-02")
				}

				// Build helpful error message with context
				existingPeriod := assignment.StartDate.Format("2006-01-02")
				if assignment.LastDate != nil {
					existingPeriod += " đến " + assignment.LastDate.Format("2006-01-02")
				} else {
					existingPeriod += " (không có ngày kết thúc)"
				}

				message := constants.MsgAssignmentOverlapVN + ". Phân công hiện tại: " + existingPeriod + ". Vui lòng chọn ngày bắt đầu từ " + suggestedDate + " trở đi."

				return domain.NewConflictError(message).
					WithContext("existing_start_date", assignment.StartDate.Format("2006-01-02")).
					WithContext("existing_end_date", assignment.LastDate).
					WithContext("suggested_start_date", suggestedDate).
					WithContext("timesheet_count", timesheetCount)
			}
		}
	}

	return nil
}

// GetConflictingAssignments returns all overlapping assignments for an employee (for reference purposes)
func (s *EmployeeAssignmentService) GetConflictingAssignments(ctx context.Context, employeeID uint, startDate time.Time, endDate *time.Time) ([]*domain.ProjectEmployee, error) {
	return s.projectEmployeeRepo.GetOverlappingAssignments(ctx, employeeID, &startDate, endDate)
}
