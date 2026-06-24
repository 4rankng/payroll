package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"api-server/internal/domain"
)

// ValidateTotalDailyHours checks if total hours worked in a day (across all entries) don't exceed 24
func (s *TimesheetValidationService) ValidateTotalDailyHours(ctx context.Context, newTimesheet *domain.Timesheet) error {
	// Create a temporary context for single-call usage
	vctx := newValidationContext()
	return s.validateTotalDailyHoursWithContext(ctx, vctx, newTimesheet)
}

// validateTotalDailyHoursWithContext checks total hours with caching support
func (s *TimesheetValidationService) validateTotalDailyHoursWithContext(ctx context.Context, vctx *validationContext, newTimesheet *domain.Timesheet) error {
	// Get all existing timesheets for this employee/project/date (cached)
	existingTimesheets, err := s.getExistingTimesheets(ctx, vctx, newTimesheet.ProjectID, newTimesheet.EmployeeID, newTimesheet.Date)
	if err != nil {
		return domain.NewInternalError("Lỗi kiểm tra tổng giờ làm việc hàng ngày", err)
	}

	// Calculate total hours (existing + new)
	var totalHours = newTimesheet.HoursWorked
	for _, ts := range existingTimesheets {
		// Skip this timesheet if we're updating an existing one
		if newTimesheet.ID != 0 && ts.ID == newTimesheet.ID {
			continue
		}
		totalHours += ts.HoursWorked
	}

	// Check if total exceeds 24 hours
	if totalHours > 24 {
		// Get employee name for proper error format
		employee, err := s.employeeRepo.GetByID(ctx, newTimesheet.EmployeeID)
		employeeName := "Unknown"
		if err == nil && employee != nil {
			employeeName = employee.Fullname
		}

		// Build error message in required format: "%s: Tổng giờ làm việc ngày YYYY-MM-DD vượt quá 24 giờ. Hiện tại: %.1f giờ"
		return domain.NewValidationError(fmt.Sprintf("%s: Tổng giờ làm việc ngày %s vượt quá 24 giờ. Hiện tại: %.1f giờ",
			employeeName, newTimesheet.Date.Format("2006-01-02"), totalHours))
	}

	return nil
}

// ValidateBulkDailyHours checks if total hours worked in a day (across all entries including batch) don't exceed 24
// This method is specifically for bulk operations and considers other entries in the same batch
func (s *TimesheetValidationService) ValidateBulkDailyHours(ctx context.Context, employeeID, projectID uint, date time.Time, batchEntries []float64) error {
	// Get all existing timesheets for this employee/project/date
	existingTimesheets, err := s.timesheetReader.GetByProjectEmployeeDate(ctx, projectID, employeeID, date)
	if err != nil {
		return domain.NewInternalError("Lỗi kiểm tra tổng giờ làm việc hàng ngày", err)
	}

	// Calculate total hours (existing + batch)
	var totalHours float64
	for _, ts := range existingTimesheets {
		totalHours += ts.HoursWorked
	}

	// Add hours from batch entries
	for _, batchHours := range batchEntries {
		totalHours += batchHours
	}

	// Check if total exceeds 24 hours
	if totalHours > 24 {
		// Get employee name for proper error format
		employee, err := s.employeeRepo.GetByID(ctx, employeeID)
		employeeName := "Unknown"
		if err == nil && employee != nil {
			employeeName = employee.Fullname
		}

		// Build error message in required format: "%s: Tổng giờ làm việc ngày YYYY-MM-DD vượt quá 24 giờ. Hiện tại: %.1f giờ"
		return domain.NewValidationError(fmt.Sprintf("%s: Tổng giờ làm việc ngày %s vượt quá 24 giờ. Hiện tại: %.1f giờ",
			employeeName, date.Format("2006-01-02"), totalHours))
	}

	return nil
}

// ValidateBulkDailyHoursOptimized validates total daily hours for multiple combinations in a single query.
// dailyGroups:    key "projectID-employeeID-date" → hours being added/updated (hoursWorked > 0).
// deletionGroups: same key format → hours being deleted (hoursWorked = 0 entries, used to subtract from existing total).
// upsertDeltas:   same key format → sum of existing row hours that this batch's requests will overwrite
//
//	(an upsert of an existing paytype replaces that row, so its old hours must
//	be excluded from existingTotal — otherwise an upsert reads as if the row
//	were being added on top of itself, e.g. 2.0 + 10.5 + 12.5 = 25.0 instead of
//	the correct 2.0 + 12.5 = 14.5).
func (s *TimesheetValidationService) ValidateBulkDailyHoursOptimized(ctx context.Context, dailyGroups map[string][]float64, deletionGroups map[string]float64, upsertDeltas map[string]float64) []PreviewError {
	var errors []PreviewError

	if len(dailyGroups) == 0 {
		return errors
	}

	// Convert daily groups to combos for bulk query
	var combos []domain.EmployeeDateCombo
	for key := range dailyGroups {
		parts := strings.Split(key, "-")
		if len(parts) < 3 {
			continue
		}

		projectID, _ := strconv.Atoi(parts[0])
		employeeID, _ := strconv.Atoi(parts[1])
		dateStr := strings.Join(parts[2:], "-")
		date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			errors = append(errors, PreviewError{
				EmployeeID: uint(employeeID),
				Message:    fmt.Sprintf("Định dạng ngày không hợp lệ: %s", dateStr),
			})
			continue
		}

		combos = append(combos, domain.EmployeeDateCombo{
			EmployeeID: uint(employeeID),
			ProjectID:  uint(projectID),
			Date:       date,
		})
	}

	// Single query to get all existing timesheets
	existingTimesheets, err := s.timesheetValidator.GetByEmployeeDateCombos(ctx, combos)
	if err != nil {
		errors = append(errors, PreviewError{
			EmployeeID: 0,
			Message:    fmt.Sprintf("Lỗi kiểm tra tổng giờ làm việc hàng ngày: %v", err),
		})
		return errors
	}

	// Group existing timesheets by combo key for easy lookup
	existingMap := make(map[string][]*domain.Timesheet)
	for _, ts := range existingTimesheets {
		key := fmt.Sprintf("%d-%d-%s", ts.ProjectID, ts.EmployeeID, ts.Date.Format("2006-01-02"))
		existingMap[key] = append(existingMap[key], ts)
	}

	// Validate each day's total hours
	for key, batchHours := range dailyGroups {
		parts := strings.Split(key, "-")
		if len(parts) < 3 {
			continue
		}

		_, _ = strconv.Atoi(parts[0]) // projectID (not used in validation)
		employeeID, _ := strconv.Atoi(parts[1])
		dateStr := strings.Join(parts[2:], "-")
		_, _ = time.ParseInLocation("2006-01-02", dateStr, time.Local) // date (not used in validation)

		// Calculate post-batch daily total, treating upserts as replacements
		// (not additions) of their existing rows. See postBatchDailyTotal.
		deletionHours := deletionGroups[key]
		upsertHours := upsertDeltas[key]
		totalHours := postBatchDailyTotal(existingMap[key], batchHours, deletionHours, upsertHours)

		// Check if total exceeds 24 hours
		if totalHours > 24 {
			errors = append(errors, PreviewError{
				EmployeeID: uint(employeeID),
				Message: fmt.Sprintf("Tổng giờ làm việc ngày %s vượt quá 24 giờ. Hiện tại: %.1f giờ",
					dateStr, totalHours),
			})
		}
	}

	return errors
}

// postBatchDailyTotal returns the resulting daily total for one
// (projectID, employeeID, date) combo after applying a batch's additions,
// deletions, and upsert overwrites against the existing rows.
//
// Pure function — no DB, no time, no globals — so the upsert-overwrite rule
// can be regression-tested without the SQLite-vs-MySQL driver pitfalls that
// the day-bounds bug hit.
//
//   - existingTimesheets: rows currently stored for this combo (every row is
//     summed, so the caller must have already filtered soft-deletes).
//   - batchHours: hoursWorked of each non-deletion request in the batch for
//     this combo (deletion requests are represented by deletionHours, not here).
//   - deletionHours: <0 means "delete all existing rows for this combo" (sentinel);
//     otherwise, those existing hours are subtracted from the total.
//   - upsertHours: sum of hoursWorked of the existing rows that this batch's
//     requests will overwrite (e.g. updating ot200 10.5 → 12.5 contributes
//     10.5 here, so the total reads as 12.5, not 12.5 + 10.5).
//
// Clamped at 0 throughout — a deletion/upsert can't drive existingTotal
// negative, only zero it out.
func postBatchDailyTotal(existingTimesheets []*domain.Timesheet, batchHours []float64, deletionHours, upsertHours float64) float64 {
	var existingTotal float64
	for _, ts := range existingTimesheets {
		existingTotal += ts.HoursWorked
	}

	if deletionHours < 0 {
		existingTotal = 0
	} else if deletionHours > 0 {
		existingTotal -= deletionHours
		if existingTotal < 0 {
			existingTotal = 0
		}
	}

	if upsertHours > 0 {
		existingTotal -= upsertHours
		if existingTotal < 0 {
			existingTotal = 0
		}
	}

	var batchTotal float64
	for _, hours := range batchHours {
		batchTotal += hours
	}

	return existingTotal + batchTotal
}
