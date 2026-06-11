package services

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	dbhelper "api-server/internal/pkg/db"
	"api-server/internal/pkg/retry"
	"api-server/internal/pkg/timeutil"

	"gorm.io/gorm"
)

// ProjectEmployeeTemporalService provides temporal data management for project employee assignments
type ProjectEmployeeTemporalService struct {
	db                  *gorm.DB
	projectEmployeeRepo domain.ProjectEmployeeRepository
	dbHelper            *dbhelper.DatabaseHelper
	retryConfig         retry.DatabaseOperationConfig
}

// NewProjectEmployeeTemporalService creates a new project employee temporal service
func NewProjectEmployeeTemporalService(db *gorm.DB, projectEmployeeRepo domain.ProjectEmployeeRepository) *ProjectEmployeeTemporalService {
	dbHelper := dbhelper.NewDatabaseHelper(db)
	return &ProjectEmployeeTemporalService{
		db:                  db,
		projectEmployeeRepo: projectEmployeeRepo,
		dbHelper:            dbHelper,
		retryConfig:         retry.DefaultDatabaseConfig(),
	}
}

// CreateEffectiveDatedAssignment implements the temporal algorithm for project employee assignments
func (s *ProjectEmployeeTemporalService) CreateEffectiveDatedAssignment(ctx context.Context, assignment *domain.ProjectEmployee) error {
	if err := s.validateNotInPast(assignment.StartDate, "start_date"); err != nil {
		return err
	}

	// Use database helper with retry logic and transaction management
	return s.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		// Validate no conflicts for this employee-project combination
		if err := s.validateNoConflicts(tx, assignment.EmployeeID, assignment.ProjectID,
			assignment.StartDate, assignment.LastDate, 0); err != nil {
			return err
		}

		// Insert new assignment
		if err := tx.Create(assignment).Error; err != nil {
			return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to create new assignment: %w", err))
		}

		return nil
	})
}

// CreateEffectiveDatedAssignmentWithAutoEnd creates assignment and automatically ends previous one
// This strategy follows the payrate pattern of auto-closing previous records
func (s *ProjectEmployeeTemporalService) CreateEffectiveDatedAssignmentWithAutoEnd(ctx context.Context, assignment *domain.ProjectEmployee) error {
	if err := s.validateNotInPast(assignment.StartDate, "start_date"); err != nil {
		return err
	}

	// Use database helper with retry logic and transaction management
	return s.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		// Close existing active assignments for this employee-project
		if err := s.closeActiveAssignments(tx, assignment.EmployeeID, assignment.ProjectID, assignment.StartDate); err != nil {
			return err
		}

		// Insert new assignment as active with open-ended last_date
		assignment.LastDate = nil

		if err := tx.Create(assignment).Error; err != nil {
			return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to create new assignment: %w", err))
		}

		return nil
	})
}

// GetActiveAssignmentForEmployeeProject retrieves the currently active assignment
func (s *ProjectEmployeeTemporalService) GetActiveAssignmentForEmployeeProject(ctx context.Context, employeeID, projectID uint) (*domain.ProjectEmployee, error) {
	var assignment *domain.ProjectEmployee
	err := s.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		var err error
		assignment, err = s.findActiveAssignmentForEmployeeProject(db, employeeID, projectID)
		return err
	})
	return assignment, err
}

// GetActiveAssignmentOnDate retrieves the active assignment for an employee-project on a specific date
func (s *ProjectEmployeeTemporalService) GetActiveAssignmentOnDate(ctx context.Context, employeeID, projectID uint, date time.Time) (*domain.ProjectEmployee, error) {
	date = timeutil.StartOfDay(date)

	var assignment domain.ProjectEmployee
	err := s.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		return db.Where("employee_id = ? AND project_id = ? AND start_date <= ? AND (last_date IS NULL OR last_date >= ?)",
			employeeID, projectID, date, date).
			Order("start_date DESC").
			First(&assignment).Error
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError(fmt.Sprintf("không tìm thấy phân công hoạt động cho nhân viên %d, dự án %d vào ngày %s",
				employeeID, projectID, date.Format("2006-01-02")))
		}
		return nil, s.dbHelper.WrapDatabaseError(err)
	}

	return &assignment, nil
}

// EndActiveAssignmentForEmployeeProject manually ends the currently active assignment
func (s *ProjectEmployeeTemporalService) EndActiveAssignmentForEmployeeProject(ctx context.Context, employeeID, projectID uint, endDate time.Time) error {
	endDate = timeutil.StartOfDay(endDate)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		activeAssignment, err := s.findActiveAssignmentForEmployeeProject(tx, employeeID, projectID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return domain.NewNotFoundError("không tìm thấy phân công hoạt động để kết thúc")
			}
			return fmt.Errorf("failed to find active assignment: %w", err)
		}

		// Validate end date is after start date
		if endDate.Before(activeAssignment.StartDate) || endDate.Equal(activeAssignment.StartDate) {
			return domain.NewValidationError(fmt.Sprintf("ngày kết thúc (%s) phải sau ngày bắt đầu (%s)",
				endDate.Format("2006-01-02"), activeAssignment.StartDate.Format("2006-01-02")))
		}

		activeAssignment.LastDate = &endDate

		return tx.Save(activeAssignment).Error
	})
}

// GetEmployeeActiveAssignments retrieves all active assignments for an employee
func (s *ProjectEmployeeTemporalService) GetEmployeeActiveAssignments(ctx context.Context, employeeID uint) ([]*domain.ProjectEmployee, error) {
	var assignments []*domain.ProjectEmployee
	err := s.db.WithContext(ctx).
		Where("employee_id = ? AND last_date IS NULL",
			employeeID).
		Find(&assignments).Error

	return assignments, err
}

// ValidateEmployeeTemporalIntegrity performs comprehensive validation of employee assignment temporal data
func (s *ProjectEmployeeTemporalService) ValidateEmployeeTemporalIntegrity(ctx context.Context, employeeID uint) error {
	var assignments []domain.ProjectEmployee
	err := s.db.WithContext(ctx).
		Where("employee_id = ?", employeeID).
		Order("start_date ASC").
		Find(&assignments).Error
	if err != nil {
		return fmt.Errorf("failed to fetch assignments for validation: %w", err)
	}

	// Group assignments by project
	projectAssignments := make(map[uint][]domain.ProjectEmployee)
	for _, assignment := range assignments {
		projectAssignments[assignment.ProjectID] = append(projectAssignments[assignment.ProjectID], assignment)
	}

	// Validate each project's assignment timeline
	for projectID, projectAssigns := range projectAssignments {
		if err := s.validateProjectAssignmentTimeline(projectAssigns, employeeID, projectID); err != nil {
			return err
		}
	}

	// Check for overlapping assignments across different projects
	if err := s.validateNoOverlappingAssignments(assignments, employeeID); err != nil {
		return err
	}

	return nil
}

// Private helper methods

func (s *ProjectEmployeeTemporalService) findActiveAssignmentForEmployeeProject(db *gorm.DB, employeeID, projectID uint) (*domain.ProjectEmployee, error) {
	var assignment domain.ProjectEmployee
	err := db.Where("employee_id = ? AND project_id = ? AND last_date IS NULL",
		employeeID, projectID).First(&assignment).Error
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

// validateNotInPast checks if a date is not in the past
func (s *ProjectEmployeeTemporalService) validateNotInPast(date time.Time, fieldName string) error {
	// Use UTC date-only comparison to avoid timezone issues
	now := clock.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	if dateOnly.Before(today) {
		return domain.NewValidationError(fmt.Sprintf("không thể thêm phân công với %s trong quá khứ: %s < %s",
			fieldName, dateOnly.Format("2006-01-02"), today.Format("2006-01-02")))
	}

	return nil
}

// validateNoConflicts checks for assignment conflicts using optimized query
func (s *ProjectEmployeeTemporalService) validateNoConflicts(db *gorm.DB, employeeID, projectID uint, startDate time.Time, endDate *time.Time, excludeID uint) error {
	query := db.Model(&domain.ProjectEmployee{}).
		Where("employee_id = ? AND project_id = ?", employeeID, projectID)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	// Check for temporal overlaps using optimized logic
	if endDate != nil {
		query = query.Where("(last_date IS NULL AND start_date < ?) OR (last_date IS NOT NULL AND start_date < ? AND last_date > ?)",
			*endDate, *endDate, startDate)
	} else {
		query = query.Where("last_date IS NULL OR (last_date IS NOT NULL AND last_date > ?)", startDate)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check for conflicts: %w", err)
	}

	if count > 0 {
		return domain.NewConflictError(fmt.Sprintf("nhân viên %d đã có phân công xung đột với dự án %d", employeeID, projectID))
	}

	return nil
}

// closeActiveAssignments closes existing active assignments for employee-project combination
func (s *ProjectEmployeeTemporalService) closeActiveAssignments(db *gorm.DB, employeeID, projectID uint, newStartDate time.Time) error {
	// Optimized: Single query to check max start date
	var maxStartDate time.Time
	err := db.Model(&domain.ProjectEmployee{}).
		Where("employee_id = ? AND project_id = ? AND last_date IS NULL",
			employeeID, projectID).
		Select("COALESCE(MAX(start_date), '1900-01-01')").
		Scan(&maxStartDate).Error
	if err != nil {
		return fmt.Errorf("failed to check active assignments: %w", err)
	}

	// No active assignments if we get the default date
	if maxStartDate.Year() == 1900 {
		return nil
	}

	// Validate new start_date is after existing
	if !newStartDate.After(maxStartDate) {
		return domain.NewValidationError(fmt.Sprintf("ngày bắt đầu mới (%s) phải sau ngày bắt đầu phân công hiện tại (%s)",
			newStartDate.Format("2006-01-02"), maxStartDate.Format("2006-01-02")))
	}

	// Single bulk update leveraging indexes
	lastDate := newStartDate.AddDate(0, 0, -1)
	result := db.Model(&domain.ProjectEmployee{}).
		Where("employee_id = ? AND project_id = ? AND last_date IS NULL",
			employeeID, projectID).
		Updates(map[string]interface{}{
			"last_date": lastDate,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to close active assignments: %w", result.Error)
	}

	return nil
}

func (s *ProjectEmployeeTemporalService) validateProjectAssignmentTimeline(assignments []domain.ProjectEmployee, employeeID, projectID uint) error {
	activeCount := 0

	for i, assignment := range assignments {
		// Validate date ordering
		if assignment.LastDate != nil && !assignment.LastDate.After(assignment.StartDate) {
			return domain.NewValidationError(fmt.Sprintf("phân công %d: khoảng ngày không hợp lệ", assignment.ID))
		}

		// Count active assignments (active means LastDate is nil)
		if assignment.LastDate == nil {
			activeCount++
		}

		// Check overlap with next assignment
		if i < len(assignments)-1 {
			next := assignments[i+1]
			if assignment.LastDate != nil && !next.StartDate.After(*assignment.LastDate) {
				return domain.NewValidationError(fmt.Sprintf("phân công chồng lấp: %d (kết thúc %s) và %d (bắt đầu %s)",
					assignment.ID, assignment.LastDate.Format("2006-01-02"),
					next.ID, next.StartDate.Format("2006-01-02")))
			}
			// If current assignment is open-ended (LastDate == nil), next must start after current start
			if assignment.LastDate == nil && !next.StartDate.After(assignment.StartDate) {
				return domain.NewValidationError(fmt.Sprintf("trình tự không hợp lệ: phân công không giới hạn %d (bắt đầu %s) xung đột với phân công %d (bắt đầu %s)",
					assignment.ID, assignment.StartDate.Format("2006-01-02"),
					next.ID, next.StartDate.Format("2006-01-02")))
			}
		}
	}

	if activeCount > 1 {
		return domain.NewValidationError(fmt.Sprintf("nhiều phân công hoạt động cho nhân viên %d, dự án %d",
			employeeID, projectID))
	}

	return nil
}

func (s *ProjectEmployeeTemporalService) validateNoOverlappingAssignments(assignments []domain.ProjectEmployee, employeeID uint) error {
	// Group assignments by project first, then validate cross-project overlaps
	var periods []assignmentPeriod
	for _, assignment := range assignments {
		periods = append(periods, assignmentPeriod{
			projectID: assignment.ProjectID,
			start:     assignment.StartDate,
			end:       assignment.LastDate,
			id:        assignment.ID,
		})
	}

	// Check for overlaps between different projects
	for i := 0; i < len(periods); i++ {
		for j := i + 1; j < len(periods); j++ {
			if periods[i].projectID == periods[j].projectID {
				continue // Same project assignments are validated elsewhere
			}

			// Check if periods overlap
			if periodsOverlap(periods[i], periods[j]) {
				return domain.NewValidationError(fmt.Sprintf("nhân viên %d có phân công chồng lấp giữa dự án %d và %d (ID phân công: %d, %d)",
					employeeID, periods[i].projectID, periods[j].projectID, periods[i].id, periods[j].id))
			}
		}
	}

	return nil
}

// assignmentPeriod represents a time period for an assignment
type assignmentPeriod struct {
	projectID uint
	start     time.Time
	end       *time.Time
	id        uint
}

// periodsOverlap checks if two assignment periods overlap
func periodsOverlap(a, b assignmentPeriod) bool {
	// If either period is open-ended, check if they start at the same time or one starts before the other ends
	if a.end == nil && b.end == nil {
		// Both open-ended - overlap if they have any common time
		return true
	}
	if a.end == nil {
		// a is open-ended, overlaps if a starts before or on b's end
		return b.end == nil || !a.start.After(*b.end)
	}
	if b.end == nil {
		// b is open-ended, overlaps if b starts before or on a's end
		return !b.start.After(*a.end)
	}
	// Both have end dates - standard interval overlap check
	return a.start.Before(*b.end) && b.start.Before(*a.end)
}
