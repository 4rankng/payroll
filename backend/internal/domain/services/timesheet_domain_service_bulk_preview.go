package services

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
)

// PreviewTimesheetResult represents the result of timesheet preview validation
type PreviewTimesheetResult struct {
	Success bool
	Errors  []PreviewError
}

// PreviewTimesheets performs dry-run validation for bulk timesheet upsert (create or update)
func (s *TimesheetDomainService) PreviewTimesheets(ctx context.Context, requests []BulkCreateTimesheetEntry, createdBy uint, userRole string) (*PreviewTimesheetResult, error) {
	// Validate for duplicates within the request batch
	seen := make(map[string][]BulkCreateTimesheetEntry)
	for _, req := range requests {
		dayType := ""
		if req.DayType != nil {
			dayType = *req.DayType
		}
		key := fmt.Sprintf("%d-%d-%s-%s-%s", req.ProjectID, req.EmployeeID, req.Date, req.HourType, dayType)
		seen[key] = append(seen[key], req)
	}

	var errors []PreviewError

	// Check for duplicates within the batch
	for _, batchEntries := range seen {
		if len(batchEntries) > 1 {
			errors = append(errors, PreviewError{
				EmployeeID: batchEntries[0].EmployeeID,
				Date:       batchEntries[0].Date,
				Message: fmt.Sprintf("Trùng lặp mục nhập trong yêu cầu: Nhân viên ID %d, Dự án ID %d, Ngày %s, Loại giờ '%s'",
					batchEntries[0].EmployeeID, batchEntries[0].ProjectID, batchEntries[0].Date, batchEntries[0].HourType),
			})
		}
	}

	// Validate total daily hours — only count non-deletion entries (hoursWorked=0 are deletions)
	dailyGroups := make(map[string][]float64)
	deletionGroups := make(map[string]float64)
	for _, req := range requests {
		key := fmt.Sprintf("%d-%d-%s", req.ProjectID, req.EmployeeID, req.Date)
		if req.HoursWorked == 0 {
			deletionGroups[key] = -1
			continue
		}
		dailyGroups[key] = append(dailyGroups[key], req.HoursWorked)
	}
	bulkValidationErrors := s.validationService.ValidateBulkDailyHoursOptimized(ctx, dailyGroups, deletionGroups)
	errors = append(errors, bulkValidationErrors...)

	// Validate daytype consistency
	var daytypeEntries []BulkDaytypeEntry
	for _, req := range requests {
		dayType := "Ngày thường"
		if req.DayType != nil && *req.DayType != "" {
			dayType = *req.DayType
		}
		daytypeEntries = append(daytypeEntries, BulkDaytypeEntry{
			ProjectID:   req.ProjectID,
			EmployeeID:  req.EmployeeID,
			Date:        req.Date,
			DayType:     dayType,
			HoursWorked: req.HoursWorked,
		})
	}
	errors = append(errors, s.validationService.ValidateBulkDaytypeConsistency(ctx, daytypeEntries)...)

	// Prefetch existing timesheets for upsert detection
	var combos []domain.EmployeeDateCombo
	existingByPaytypeMap := make(map[string]*domain.Timesheet)

	for _, req := range requests {
		date, err := time.ParseInLocation("2006-01-02", req.Date, time.Local)
		if err == nil {
			combos = append(combos, domain.EmployeeDateCombo{
				EmployeeID: req.EmployeeID,
				ProjectID:  req.ProjectID,
				Date:       date,
			})
		}
	}

	if len(combos) > 0 {
		existingTimesheets, err := s.timesheetRepo.GetByEmployeeDateCombos(ctx, combos)
		if err == nil {
			for _, ts := range existingTimesheets {
				paytypeKey := fmt.Sprintf("%d-%d-%s-%s", ts.ProjectID, ts.EmployeeID, ts.Date.Format("2006-01-02"), ts.PayType)
				existingByPaytypeMap[paytypeKey] = ts
			}
		}
	}

	// Collect non-deletion entries for bulk zero-rate validation
	var zeroRateEntries []BulkZeroRateEntry

	// Validate each entry for upsert-specific logic
	for _, req := range requests {
		date, err := time.ParseInLocation("2006-01-02", req.Date, time.Local)
		if err != nil {
			errors = append(errors, PreviewError{
				EmployeeID: req.EmployeeID,
				Date:       req.Date,
				Message:    "Định dạng ngày không hợp lệ. Sử dụng định dạng YYYY-MM-DD",
			})
			continue
		}

		dayType := "Ngày thường"
		if req.DayType != nil && *req.DayType != "" {
			dayType = *req.DayType
		}

		payType, err := s.paytypeConstructionService.ConstructPaytype(ctx, req.ProjectID, req.EmployeeID, date, req.HourType, dayType)
		if err != nil {
			errors = append(errors, PreviewError{
				EmployeeID: req.EmployeeID,
				Date:       req.Date,
				Message:    fmt.Sprintf("Kết hợp loại giờ '%s' không hợp lệ với loại ngày '%s'", req.HourType, dayType),
			})
			continue
		}

		paytypeKey := fmt.Sprintf("%d-%d-%s-%s", req.ProjectID, req.EmployeeID, req.Date, payType)
		existingTimesheet, exists := existingByPaytypeMap[paytypeKey]

		if exists {
			if req.HoursWorked == 0 {
				if existingTimesheet.IsPaid() {
					detailedError := s.validationService.GenerateDetailedPaidTimesheetError(ctx, existingTimesheet)
					errors = append(errors, PreviewError{EmployeeID: req.EmployeeID, Date: req.Date, Message: detailedError})
					continue
				}
				if existingTimesheet.IsApproved() && userRole == "partner" {
					errors = append(errors, PreviewError{
						EmployeeID: req.EmployeeID,
						Date:       req.Date,
						Message:    "Không thể xóa bảng chấm công đã được phê duyệt",
					})
					continue
				}
				continue
			}

			if existingTimesheet.IsPaid() {
				detailedError := s.validationService.GenerateDetailedPaidTimesheetError(ctx, existingTimesheet)
				errors = append(errors, PreviewError{EmployeeID: req.EmployeeID, Date: req.Date, Message: detailedError})
				continue
			}
			if existingTimesheet.IsApproved() && userRole == "partner" {
				errors = append(errors, PreviewError{
					EmployeeID: req.EmployeeID,
					Date:       req.Date,
					Message:    "Không thể chỉnh sửa bảng chấm công đã được phê duyệt",
				})
				continue
			}
			if req.HoursWorked < 0 {
				errors = append(errors, PreviewError{
					EmployeeID: req.EmployeeID,
					Date:       req.Date,
					Message:    "Số giờ làm việc không thể là số âm",
				})
				continue
			}

			tempTimesheet := &domain.Timesheet{
				ProjectID:   req.ProjectID,
				EmployeeID:  req.EmployeeID,
				Date:        date,
				HoursWorked: req.HoursWorked,
				PayType:     payType,
			}
			if err := s.validationService.ValidateDaytypeConsistency(ctx, tempTimesheet); err != nil {
				errors = append(errors, PreviewError{EmployeeID: req.EmployeeID, Date: req.Date, Message: err.Error()})
				continue
			}
		} else {
			if req.HoursWorked == 0 {
				continue
			}
			timesheet := &domain.Timesheet{
				ProjectID:   req.ProjectID,
				EmployeeID:  req.EmployeeID,
				Date:        date,
				HoursWorked: req.HoursWorked,
				PayType:     payType,
			}
			if err := s.validationService.ValidateTimesheet(ctx, timesheet); err != nil {
				errors = append(errors, PreviewError{EmployeeID: req.EmployeeID, Date: req.Date, Message: err.Error()})
				continue
			}
		}

		zeroRateEntries = append(zeroRateEntries, BulkZeroRateEntry{
			ProjectID:  req.ProjectID,
			EmployeeID: req.EmployeeID,
			Date:       date,
			PayType:    payType,
		})
	}

	// Bulk zero-rate validation
	errors = append(errors, s.validationService.ValidateBulkZeroRates(ctx, zeroRateEntries)...)

	// Deduplicate errors
	errors = deduplicatePreviewErrors(errors)

	result := &PreviewTimesheetResult{
		Success: len(errors) == 0,
		Errors:  errors,
	}

	return result, nil
}

// ValidateBulkTimesheetEntries validates a batch of timesheet entries with complete business logic
func (s *TimesheetDomainService) ValidateBulkTimesheetEntries(ctx context.Context, entries []BulkCreateTimesheetEntry) []PreviewError {
	var errors []PreviewError

	seen := make(map[string][]BulkCreateTimesheetEntry)
	for _, req := range entries {
		dayType := ""
		if req.DayType != nil {
			dayType = *req.DayType
		}
		key := fmt.Sprintf("%d-%d-%s-%s-%s", req.ProjectID, req.EmployeeID, req.Date, req.HourType, dayType)
		seen[key] = append(seen[key], req)
	}

	for _, batchEntries := range seen {
		if len(batchEntries) > 1 {
			errors = append(errors, PreviewError{
				EmployeeID: batchEntries[0].EmployeeID,
				Date:       batchEntries[0].Date,
				Message: fmt.Sprintf("Trùng lặp mục nhập trong yêu cầu: Nhân viên ID %d, Dự án ID %d, Ngày %s, Loại giờ '%s'",
					batchEntries[0].EmployeeID, batchEntries[0].ProjectID, batchEntries[0].Date, batchEntries[0].HourType),
			})
		}
	}

	dailyGroups := make(map[string][]float64)
	deletionGroups := make(map[string]float64)
	for _, req := range entries {
		key := fmt.Sprintf("%d-%d-%s", req.ProjectID, req.EmployeeID, req.Date)
		if req.HoursWorked == 0 {
			deletionGroups[key] = -1
			continue
		}
		dailyGroups[key] = append(dailyGroups[key], req.HoursWorked)
	}

	bulkValidationErrors := s.validationService.ValidateBulkDailyHoursOptimized(ctx, dailyGroups, deletionGroups)
	errors = append(errors, bulkValidationErrors...)

	var daytypeEntries []BulkDaytypeEntry
	for _, req := range entries {
		dayType := "Ngày thường"
		if req.DayType != nil && *req.DayType != "" {
			dayType = *req.DayType
		}
		daytypeEntries = append(daytypeEntries, BulkDaytypeEntry{
			ProjectID:   req.ProjectID,
			EmployeeID:  req.EmployeeID,
			Date:        req.Date,
			DayType:     dayType,
			HoursWorked: req.HoursWorked,
		})
	}
	daytypeErrors := s.validationService.ValidateBulkDaytypeConsistency(ctx, daytypeEntries)
	errors = append(errors, daytypeErrors...)

	return errors
}

// deduplicatePreviewErrors removes duplicate PreviewErrors (same EmployeeID + Date + Message).
func deduplicatePreviewErrors(errors []PreviewError) []PreviewError {
	seen := make(map[string]bool, len(errors))
	out := errors[:0]
	for _, e := range errors {
		key := fmt.Sprintf("%d|%s|%s", e.EmployeeID, e.Date, e.Message)
		if !seen[key] {
			seen[key] = true
			out = append(out, e)
		}
	}
	return out
}
