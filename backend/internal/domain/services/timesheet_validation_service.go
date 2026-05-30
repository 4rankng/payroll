package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"api-server/internal/domain"
	infrastructureports "api-server/internal/domain/ports/infrastructure"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
)

// TimesheetValidationService handles business logic for timesheet validation
type TimesheetValidationService struct {
	timesheetRepo       domain.TimesheetRepository
	projectEmployeeRepo domain.ProjectEmployeeRepository
	payrateRepo         domain.PayrateRepository
	employeeRepo        domain.EmployeeRepository
	projectRepo         domain.ProjectRepository
	cache               infrastructureports.CachePort
}

// validationContext caches query results within a single validation request
// to avoid duplicate database queries. Thread-safe for concurrent validation.
type validationContext struct {
	mu                 sync.RWMutex
	existingTimesheets map[string][]*domain.Timesheet // key: "projectID-employeeID-date"
}

// NewTimesheetValidationService creates a new timesheet validation service
func NewTimesheetValidationService(
	timesheetRepo domain.TimesheetRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	payrateRepo domain.PayrateRepository,
	employeeRepo domain.EmployeeRepository,
	projectRepo domain.ProjectRepository,
	cache infrastructureports.CachePort,
) *TimesheetValidationService {
	return &TimesheetValidationService{
		timesheetRepo:       timesheetRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		payrateRepo:         payrateRepo,
		employeeRepo:        employeeRepo,
		projectRepo:         projectRepo,
		cache:               cache,
	}
}

// newValidationContext creates a new validation context for caching
func newValidationContext() *validationContext {
	return &validationContext{
		existingTimesheets: make(map[string][]*domain.Timesheet),
	}
}

// getExistingTimesheets retrieves existing timesheets with thread-safe caching
func (s *TimesheetValidationService) getExistingTimesheets(ctx context.Context, vctx *validationContext, projectID, employeeID uint, date time.Time) ([]*domain.Timesheet, error) {
	key := fmt.Sprintf("%d-%d-%s", projectID, employeeID, date.Format("2006-01-02"))

	// Check cache first with read lock
	vctx.mu.RLock()
	if cached, ok := vctx.existingTimesheets[key]; ok {
		vctx.mu.RUnlock()
		return cached, nil
	}
	vctx.mu.RUnlock()

	// Query database and cache result with write lock
	vctx.mu.Lock()
	defer vctx.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have populated it)
	if cached, ok := vctx.existingTimesheets[key]; ok {
		return cached, nil
	}

	timesheets, err := s.timesheetRepo.GetByProjectEmployeeDate(ctx, projectID, employeeID, date)
	if err != nil {
		return nil, err
	}

	vctx.existingTimesheets[key] = timesheets
	return timesheets, nil
}

// ValidateTimesheet validates business rules for timesheet creation/update
// Uses concurrent validation for improved performance
// Note: This validator allows hoursWorked=0 because the bulk create flow uses zero hours
// as a deletion signal. The actual deletion logic is handled in BulkCreateTimesheets.
func (s *TimesheetValidationService) ValidateTimesheet(ctx context.Context, timesheet *domain.Timesheet) error {
	logger := observability.GetLogger()
	logger.Info("ValidateTimesheet: Starting validation",
		"project_id", timesheet.ProjectID,
		"employee_id", timesheet.EmployeeID,
		"date", timesheet.Date.Format("2006-01-02"),
		"hours_worked", timesheet.HoursWorked,
		"pay_type", timesheet.PayType)

	// Create validation context for caching
	vctx := newValidationContext()

	// Validate individual hours worked
	if timesheet.HoursWorked < 0 {
		logger.Error("ValidateTimesheet: Failed - negative hours worked",
			"hours_worked", timesheet.HoursWorked)
		return domain.NewValidationError("Số giờ làm việc không thể là số âm")
	}

	// IMPORTANT: Zero hours is allowed here because it's used as a deletion signal
	// in bulk operations (see BulkCreateTimesheets in timesheet_domain_service.go).
	// The actual deletion/skip logic is handled there, not in this validator.

	// Note: Individual 24-hour check removed - total daily hours validation handles this
	// and provides better error messages with employee names

	// Validate date is not in the future
	// Compare date-only (ignoring time and timezone) to avoid issues around midnight
	now := clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	timesheetDate := time.Date(timesheet.Date.Year(), timesheet.Date.Month(), timesheet.Date.Day(), 0, 0, 0, 0, now.Location())
	if timesheetDate.After(today) {
		logger.Error("ValidateTimesheet: Failed - future date",
			"timesheet_date", timesheetDate.Format("2006-01-02"),
			"current_date", today.Format("2006-01-02"))
		return domain.NewValidationError("Ngày không thể là ngày trong tương lai")
	}

	logger.Info("ValidateTimesheet: Passed basic validations, starting concurrent validations")

	// Run all validations concurrently for better performance
	var wg sync.WaitGroup
	errChan := make(chan error, 6)

	// Validation 1: Employee assignment
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger := observability.GetLogger()
		logger.Info("ValidateTimesheet: Checking employee assignment",
			"employee_id", timesheet.EmployeeID,
			"project_id", timesheet.ProjectID,
			"date", timesheet.Date.Format("2006-01-02"))
		if err := s.ValidateEmployeeAssignment(ctx, timesheet.EmployeeID, timesheet.ProjectID, timesheet.Date); err != nil {
			logger.Error("ValidateTimesheet: Employee assignment validation failed",
				"error", err.Error())
			errChan <- err
		} else {
			logger.Info("ValidateTimesheet: Employee assignment validation passed")
		}
	}()

	// Validation 2: Payrate exists
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger := observability.GetLogger()
		logger.Info("ValidateTimesheet: Checking payrate",
			"project_id", timesheet.ProjectID,
			"date", timesheet.Date.Format("2006-01-02"))
		if err := s.ValidatePayrate(ctx, timesheet.ProjectID, timesheet.Date); err != nil {
			logger.Error("ValidateTimesheet: Payrate validation failed",
				"error", err.Error())
			errChan <- err
		} else {
			logger.Info("ValidateTimesheet: Payrate validation passed")
		}
	}()

	// Validation 3: Total daily hours
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger := observability.GetLogger()
		logger.Info("ValidateTimesheet: Checking total daily hours")
		if err := s.validateTotalDailyHoursWithContext(ctx, vctx, timesheet); err != nil {
			logger.Error("ValidateTimesheet: Total daily hours validation failed",
				"error", err.Error())
			errChan <- err
		} else {
			logger.Info("ValidateTimesheet: Total daily hours validation passed")
		}
	}()

	// Validation 4: Daytype consistency
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger := observability.GetLogger()
		logger.Info("ValidateTimesheet: Checking daytype consistency")
		if err := s.validateDaytypeConsistencyWithContext(ctx, vctx, timesheet); err != nil {
			logger.Error("ValidateTimesheet: Daytype consistency validation failed",
				"error", err.Error())
			errChan <- err
		} else {
			logger.Info("ValidateTimesheet: Daytype consistency validation passed")
		}
	}()

	// Validation 5: Duplicate check
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger := observability.GetLogger()
		logger.Info("ValidateTimesheet: Checking for duplicate timesheets")
		if err := s.ValidateDuplicateTimesheet(ctx, timesheet); err != nil {
			logger.Error("ValidateTimesheet: Duplicate check validation failed",
				"error", err.Error())
			errChan <- err
		} else {
			logger.Info("ValidateTimesheet: Duplicate check validation passed")
		}
	}()

	// Wait for all validations to complete
	wg.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		if err != nil {
			logger.Error("ValidateTimesheet: Validation failed with error",
				"error", err.Error())
			return err
		}
	}

	logger.Info("ValidateTimesheet: All validations passed successfully")
	return nil
}

// parsePaytype extracts day type and hour type from a paytype string
// Format: "position.dayType.hourType"
func (s *TimesheetValidationService) parsePaytype(paytype string) (dayType, hourType string) {
	parts := strings.Split(paytype, ".")
	if len(parts) >= 3 {
		dayType = parts[1]
		hourType = strings.Join(parts[2:], ".") // Handle hour types with dots
	} else if len(parts) == 2 {
		// Fallback for old format
		dayType = parts[0]
		hourType = parts[1]
	} else {
		// Very old or invalid format
		dayType = "Ngày thường"
		hourType = paytype
	}
	return
}
