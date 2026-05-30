package services

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

// ValidateEmployeeAssignment checks if employee is assigned to project on given date
// Uses Redis cache for frequently accessed assignments
func (s *TimesheetValidationService) ValidateEmployeeAssignment(ctx context.Context, employeeID, projectID uint, date time.Time) error {
	cacheKey := fmt.Sprintf("assignment:%d:%d", projectID, employeeID)

	// Try to get from cache first
	var assignment domain.ProjectEmployee
	err := s.cache.Get(ctx, cacheKey, &assignment)
	if err == nil {
		// Cache hit - use cached assignment
		if assignment.StartDate.After(date) {
			return domain.NewValidationError("thời gian phân công nhân viên bắt đầu sau ngày chấm công")
		}
		if assignment.LastDate != nil && assignment.LastDate.Before(date) {
			return domain.NewValidationError("thời gian phân công nhân viên kết thúc trước ngày chấm công")
		}
		return nil
	}

	// Cache miss - query database
	assignmentPtr, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, projectID, employeeID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return domain.NewValidationError("nhân viên chưa được phân công vào dự án này")
		}
		return domain.NewInternalError("Lỗi kiểm tra phân công nhân viên", err)
	}

	// Cache the result
	if assignmentPtr != nil {
		_ = s.cache.Set(ctx, cacheKey, *assignmentPtr, constants.EmployeeAssignmentCacheTTL)
	}

	// Check if assignment covers the timesheet date
	if assignmentPtr.StartDate.After(date) {
		return domain.NewValidationError("thời gian phân công nhân viên bắt đầu sau ngày chấm công")
	}

	if assignmentPtr.LastDate != nil && assignmentPtr.LastDate.Before(date) {
		return domain.NewValidationError("thời gian phân công nhân viên kết thúc trước ngày chấm công")
	}

	return nil
}

// ValidatePayrate checks if active payrate exists for project on given date
// Uses Redis cache for frequently accessed payrate configurations
func (s *TimesheetValidationService) ValidatePayrate(ctx context.Context, projectID uint, date time.Time) error {
	// Cache key includes date to handle different payrate periods
	dateStr := date.Format("2006-01-02")
	cacheKey := fmt.Sprintf("payrate:%d:%s", projectID, dateStr)

	// Try to get from cache first
	var exists bool
	err := s.cache.Get(ctx, cacheKey, &exists)
	if err == nil && exists {
		// Cache hit - payrate exists
		return nil
	}

	// Cache miss - query database
	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, projectID, date)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return domain.NewValidationError("không tìm thấy cấu hình mức lương hiệu lực cho dự án vào ngày này")
		}
		return domain.NewInternalError("Lỗi kiểm tra cấu hình mức lương", err)
	}

	// Cache the existence of payrate
	if payrate != nil {
		_ = s.cache.Set(ctx, cacheKey, true, constants.PayrateCacheTTL)
	}

	return nil
}

// InvalidateEmployeeAssignmentCache invalidates the cache for a specific employee-project assignment
// Should be called when:
// - Creating a new employee-project assignment
// - Updating an existing assignment (dates, position, etc.)
// - Deleting/unassigning an employee from a project
// Locations: internal/app/services/project/employee_service.go, internal/domain/services/assignment_lifecycle_manager.go
func (s *TimesheetValidationService) InvalidateEmployeeAssignmentCache(ctx context.Context, projectID, employeeID uint) error {
	cacheKey := fmt.Sprintf("assignment:%d:%d", projectID, employeeID)
	return s.cache.Delete(ctx, cacheKey)
}

// InvalidatePayrateCache invalidates all payrate caches for a specific project
// Since payrates can span multiple dates, we invalidate all payrate entries for the project
// Should be called when:
// - Creating a new payrate configuration
// - Updating an existing payrate (rates, dates, etc.)
// - Deleting a payrate configuration
// Locations: payrate service/repository where Create/Update/Delete operations occur
func (s *TimesheetValidationService) InvalidatePayrateCache(ctx context.Context, projectID uint) error {
	pattern := fmt.Sprintf("payrate:%d:*", projectID)
	return s.cache.InvalidatePattern(ctx, pattern)
}
