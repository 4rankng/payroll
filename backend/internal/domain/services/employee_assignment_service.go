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
	timesheetChecker    domain.TimesheetAssignmentChecker
}

// NewEmployeeAssignmentService creates a new employee assignment service
func NewEmployeeAssignmentService(
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	employeeRepo domain.EmployeeRepository,
	projectRepo domain.ProjectRepository,
	timesheetChecker domain.TimesheetAssignmentChecker,
) *EmployeeAssignmentService {
	return &EmployeeAssignmentService{
		projectEmployeeRepo: projectEmployeeRepo,
		employeeRepo:        employeeRepo,
		projectRepo:         projectRepo,
		timesheetChecker:    timesheetChecker,
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

	// Load the stored assignment so guards only fire on the boundary that actually shrinks.
	current, err := s.projectEmployeeRepo.GetByID(ctx, assignment.ID)
	if err != nil {
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
			count, err := s.timesheetChecker.CountTimesheets(ctx, other.ProjectID, other.EmployeeID)
			if err != nil {
				return err
			}
			overlappingCounts[other.ID] = count
		}
	}

	// The update may not leave an approved or paid timesheet outside the
	// resulting [start_date, last_date] window:
	//   - moving start later drops the strip [oldStart, newStart-1], which must
	//     hold no approved or paid timesheet;
	//   - keeping or setting a finite end date means any approved or paid
	//     timesheet after it falls outside the window — that also blocks an
	//     insufficient extension, since the timesheet stays uncovered;
	//   - clearing the end date (open-ended) always covers every existing
	//     timesheet, so it never blocks.
	var hasPaidBeforeNewStart, hasPaidAfterNewEnd bool
	if assignment.StartDate.After(current.StartDate) {
		windowEnd := assignment.StartDate.AddDate(0, 0, -1)
		hasPaidBeforeNewStart, err = s.timesheetChecker.HasNonEditableTimesheetsInRange(ctx, assignment.ProjectID, assignment.EmployeeID, &current.StartDate, &windowEnd)
		if err != nil {
			return err
		}
	}
	if assignment.LastDate != nil {
		windowStart := assignment.LastDate.AddDate(0, 0, 1)
		hasPaidAfterNewEnd, err = s.timesheetChecker.HasNonEditableTimesheetsInRange(ctx, assignment.ProjectID, assignment.EmployeeID, &windowStart, nil)
		if err != nil {
			return err
		}
	}

	// Delegate to pure validation method
	return s.ValidateAssignmentUpdateWithData(assignment, overlapping, overlappingCounts, hasPaidBeforeNewStart, hasPaidAfterNewEnd)
}

// ValidateAssignmentUpdateWithData performs pure domain validation with pre-loaded data
// This method has no repository dependencies and can be tested easily
func (s *EmployeeAssignmentService) ValidateAssignmentUpdateWithData(
	assignment *domain.ProjectEmployee,
	overlappingAssignments []*domain.ProjectEmployee,
	overlappingTimesheetCounts map[uint]int64,
	hasPaidBeforeNewStart bool,
	hasPaidAfterNewEnd bool,
) error {
	// Check for overlaps with OTHER assignments for the same project
	for _, other := range overlappingAssignments {
		// Skip the assignment being updated itself
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

	// The update may not leave an approved or paid timesheet outside the
	// resulting window: paid work before the new start or after the new finite
	// end stays uncovered, which blocks the update. Clearing the end date is
	// open-ended and always covers existing work.
	if hasPaidBeforeNewStart || hasPaidAfterNewEnd {
		return domain.NewValidationError(constants.MsgAssignmentUpdateExcludesPaidTimesheetsVN)
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
			timesheetCount, err := s.timesheetChecker.CountTimesheets(ctx, assignment.ProjectID, assignment.EmployeeID)
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

// SuggestAssignmentStart is the pure default-start rule shared by every
// assignment-creation path: last recorded timesheet + 1 day when the employee
// has history, else the 1st of the current month (uploaded BCC files cover the
// current month, so a mid-month "today" default would reject their entries).
func SuggestAssignmentStart(latest *time.Time, now time.Time) time.Time {
	if latest != nil {
		return latest.AddDate(0, 0, 1)
	}
	y, m, _ := now.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, now.Location())
}
