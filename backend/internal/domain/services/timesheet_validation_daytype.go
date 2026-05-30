package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"
)

// BulkDaytypeEntry represents the minimal info needed for daytype validation
type BulkDaytypeEntry struct {
	ProjectID   uint
	EmployeeID  uint
	Date        string
	DayType     string
	HoursWorked float64 // Used to distinguish deletions (0) from creates/updates
}

// ValidateDaytypeConsistency checks if the daytype is consistent for the date
// One date can only have one daytype (e.g., "Ngày thường" or "Ngày nghỉ") per employee
func (s *TimesheetValidationService) ValidateDaytypeConsistency(ctx context.Context, timesheet *domain.Timesheet) error {
	// Create a temporary context for single-call usage
	vctx := newValidationContext()
	return s.validateDaytypeConsistencyWithContext(ctx, vctx, timesheet)
}

// validateDaytypeConsistencyWithContext checks daytype consistency with caching support
func (s *TimesheetValidationService) validateDaytypeConsistencyWithContext(ctx context.Context, vctx *validationContext, timesheet *domain.Timesheet) error {
	// Parse daytype from new timesheet
	newDayType, _ := s.parsePaytype(timesheet.PayType)

	// Get all existing timesheets for this employee/project/date (cached)
	existingTimesheets, err := s.getExistingTimesheets(ctx, vctx, timesheet.ProjectID, timesheet.EmployeeID, timesheet.Date)
	if err != nil {
		return domain.NewInternalError("Lỗi kiểm tra tính nhất quán loại ngày", err)
	}

	// Check each existing timesheet for daytype conflicts
	for _, existing := range existingTimesheets {
		if existing.ID == timesheet.ID {
			continue
		}

		existingDayType, _ := s.parsePaytype(existing.PayType)

		if existingDayType != newDayType {
			return domain.NewConflictError(fmt.Sprintf("Đã có công '%s', hãy xóa trước khi chuyển sang '%s'",
				existingDayType, newDayType))
		}
	}

	return nil
}

// ValidateBulkDaytypeConsistency validates daytype consistency for bulk operations
// Ensures one date can only have one daytype per employee across batch and existing entries.
// Entries with HoursWorked==0 are deletions and are excluded from conflict checks.
func (s *TimesheetValidationService) ValidateBulkDaytypeConsistency(ctx context.Context, entries []BulkDaytypeEntry) []PreviewError {
	var errors []PreviewError

	if len(entries) == 0 {
		return errors
	}

	type dateKey struct {
		ProjectID  uint
		EmployeeID uint
		Date       string
	}

	// Track which day types are being deleted vs created/updated per date
	deletedDaytypes := make(map[dateKey]map[string]bool) // daytypes being removed (hoursWorked=0)
	activeDaytypes := make(map[dateKey]map[string]bool)  // daytypes being created/updated

	for _, entry := range entries {
		key := dateKey{
			ProjectID:  entry.ProjectID,
			EmployeeID: entry.EmployeeID,
			Date:       entry.Date,
		}
		if entry.HoursWorked == 0 {
			if deletedDaytypes[key] == nil {
				deletedDaytypes[key] = make(map[string]bool)
			}
			deletedDaytypes[key][entry.DayType] = true
		} else {
			if activeDaytypes[key] == nil {
				activeDaytypes[key] = make(map[string]bool)
			}
			activeDaytypes[key][entry.DayType] = true
		}
	}

	// Check for multiple active daytypes within the batch (excluding deletions)
	for key, daytypes := range activeDaytypes {
		if len(daytypes) > 1 {
			var daytypeList []string
			for dt := range daytypes {
				daytypeList = append(daytypeList, dt)
			}
			errors = append(errors, PreviewError{
				EmployeeID: key.EmployeeID,
				Date:       key.Date,
				Message: fmt.Sprintf("Có nhiều loại công khác nhau trong yêu cầu: %s",
					strings.Join(daytypeList, ", ")),
			})
		}
	}

	// Build combos for bulk DB query (only for keys with active entries)
	var combos []domain.EmployeeDateCombo
	for key := range activeDaytypes {
		date, err := time.ParseInLocation("2006-01-02", key.Date, time.Local)
		if err != nil {
			errors = append(errors, PreviewError{
				EmployeeID: key.EmployeeID,
				Date:       key.Date,
				Message:    fmt.Sprintf("Định dạng ngày không hợp lệ: %s", key.Date),
			})
			continue
		}
		combos = append(combos, domain.EmployeeDateCombo{
			EmployeeID: key.EmployeeID,
			ProjectID:  key.ProjectID,
			Date:       date,
		})
	}

	if len(combos) == 0 {
		return errors
	}

	existingTimesheets, err := s.timesheetRepo.GetByEmployeeDateCombos(ctx, combos)
	if err != nil {
		errors = append(errors, PreviewError{
			EmployeeID: 0,
			Message:    fmt.Sprintf("Lỗi kiểm tra tính nhất quán loại ngày: %v", err),
		})
		return errors
	}
	// Group existing timesheets by date key
	existingDaytypes := make(map[dateKey]map[string]bool)
	for _, ts := range existingTimesheets {
		key := dateKey{
			ProjectID:  ts.ProjectID,
			EmployeeID: ts.EmployeeID,
			Date:       ts.Date.Format("2006-01-02"),
		}
		existingDayType, _ := s.parsePaytype(ts.PayType)
		if existingDaytypes[key] == nil {
			existingDaytypes[key] = make(map[string]bool)
		}
		existingDaytypes[key][existingDayType] = true
	}

	// Check active batch daytypes against existing daytypes,
	// but skip if the existing daytype is being deleted in this same batch.
	for key, batchDaytypes := range activeDaytypes {
		existingDaytypesForDate, hasExisting := existingDaytypes[key]
		if !hasExisting {
			continue
		}

		deletedForKey := deletedDaytypes[key]

		for batchDaytype := range batchDaytypes {
			for existingDaytype := range existingDaytypesForDate {
				if batchDaytype == existingDaytype {
					continue // Same day type — no conflict
				}
				// If the existing day type is being deleted in this batch, it's an intentional
				// day-type migration — not a conflict.
				if deletedForKey != nil && deletedForKey[existingDaytype] {
					continue
				}
				errors = append(errors, PreviewError{
					EmployeeID: key.EmployeeID,
					Date:       key.Date,
					Message: fmt.Sprintf("Đã có công %s, không thể thêm công %s",
						existingDaytype, batchDaytype),
				})
				break
			}
		}
	}

	return errors
}
