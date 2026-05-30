package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"api-server/internal/domain"
)

// BulkUpdateTimesheetEntry represents a single timesheet update request for bulk operations
type BulkUpdateTimesheetEntry struct {
	ID          uint
	HoursWorked float64
	PayType     string
}

// BulkUpdateTimesheetResult represents the result of bulk timesheet updates
type BulkUpdateTimesheetResult struct {
	UpdatedTimesheets []*domain.Timesheet
	FailedEntries     []BulkUpdateFailure
}

// BulkUpdateFailure represents a failed timesheet update in bulk operation
type BulkUpdateFailure struct {
	ID    uint
	Error string
}

// BulkUpdateTimesheets handles bulk timesheet updates with complete business logic
func (s *TimesheetDomainService) BulkUpdateTimesheets(ctx context.Context, requests []BulkUpdateTimesheetEntry, updatedBy uint, userRole string) (*BulkUpdateTimesheetResult, error) {
	var updatedTimesheets []*domain.Timesheet
	var failedEntries []BulkUpdateFailure

	// Extract timesheet IDs for batch loading
	timesheetIDs := make([]uint, len(requests))
	for i, req := range requests {
		timesheetIDs[i] = req.ID
	}

	// Load all timesheets in a single query without relationships (optimization for validation)
	// Relationships are not needed for update validation, only basic fields
	timesheets, err := s.timesheetRepo.GetByIDsWithoutRelations(ctx, timesheetIDs)
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup
	timesheetMap := make(map[uint]*domain.Timesheet)
	for _, ts := range timesheets {
		timesheetMap[ts.ID] = ts
	}

	// Prepare for daily hours validation
	dailyGroups := make(map[string][]float64) // key: "projectID-employeeID-date" -> new hours

	// Process each update request
	for _, req := range requests {
		timesheet, exists := timesheetMap[req.ID]
		if !exists {
			failedEntries = append(failedEntries, BulkUpdateFailure{
				ID:    req.ID,
				Error: "Không tìm thấy bảng chấm công",
			})
			continue
		}

		// Validate timesheet is not paid
		if timesheet.IsPaid() {
			detailedError := s.validationService.GenerateDetailedPaidTimesheetError(ctx, timesheet)
			failedEntries = append(failedEntries, BulkUpdateFailure{
				ID:    req.ID,
				Error: detailedError,
			})
			continue
		}

		// Validate role-based edit permissions
		if timesheet.IsApproved() && userRole == "partner" {
			// Partners can edit approved timesheets, but it resets to pending
			timesheet.Status = domain.TimesheetStatusPendingApproval
			timesheet.ApprovedBy = nil
			timesheet.ApprovedAt = nil
		}

		// Update the fields
		oldHoursWorked := timesheet.HoursWorked
		oldPayType := timesheet.PayType
		timesheet.HoursWorked = req.HoursWorked
		timesheet.PayType = req.PayType

		// Validate updated timesheet
		if err := timesheet.ValidateHoursWorked(); err != nil {
			failedEntries = append(failedEntries, BulkUpdateFailure{
				ID:    req.ID,
				Error: err.Error(),
			})
			// Revert changes
			timesheet.HoursWorked = oldHoursWorked
			timesheet.PayType = oldPayType
			continue
		}

		if err := timesheet.ValidatePayType(); err != nil {
			failedEntries = append(failedEntries, BulkUpdateFailure{
				ID:    req.ID,
				Error: err.Error(),
			})
			// Revert changes
			timesheet.HoursWorked = oldHoursWorked
			timesheet.PayType = oldPayType
			continue
		}

		// Validate daytype consistency if paytype changed
		if oldPayType != req.PayType {
			if err := s.validationService.ValidateDaytypeConsistency(ctx, timesheet); err != nil {
				failedEntries = append(failedEntries, BulkUpdateFailure{
					ID:    req.ID,
					Error: err.Error(),
				})
				// Revert changes
				timesheet.HoursWorked = oldHoursWorked
				timesheet.PayType = oldPayType
				continue
			}
		}

		// Zero-rate check — reject if the payrate bucket is 0
		if req.HoursWorked > 0 {
			if zeroRateErr := s.validationService.ValidateZeroRate(ctx, timesheet.ProjectID, timesheet.Date, req.PayType, timesheet.EmployeeID); zeroRateErr != nil {
				failedEntries = append(failedEntries, BulkUpdateFailure{
					ID:    req.ID,
					Error: zeroRateErr.Message,
				})
				timesheet.HoursWorked = oldHoursWorked
				timesheet.PayType = oldPayType
				continue
			}
		}

		// Collect for daily hours validation
		dateStr := timesheet.Date.Format("2006-01-02")
		key := fmt.Sprintf("%d-%d-%s", timesheet.ProjectID, timesheet.EmployeeID, dateStr)

		// Calculate the difference in hours for this update
		hoursDiff := req.HoursWorked - oldHoursWorked
		if _, ok := dailyGroups[key]; !ok {
			dailyGroups[key] = []float64{}
		}
		dailyGroups[key] = append(dailyGroups[key], hoursDiff)

		updatedTimesheets = append(updatedTimesheets, timesheet)
	}

	// Validate total daily hours after updates
	if len(dailyGroups) > 0 {
		// Check daily limits with the changes
		for key, hoursDiffs := range dailyGroups {
			parts := strings.Split(key, "-")
			if len(parts) < 3 {
				continue
			}

			projectID, _ := strconv.Atoi(parts[0])
			employeeID, _ := strconv.Atoi(parts[1])
			dateStr := strings.Join(parts[2:], "-")
			date, _ := time.ParseInLocation("2006-01-02", dateStr, time.Local)

			// Get all existing entries for this date
			existingEntries, err := s.timesheetRepo.GetByProjectEmployeeDate(ctx, uint(projectID), uint(employeeID), date)
			if err != nil {
				// Mark all updates for this date as failed
				for _, ts := range updatedTimesheets {
					if ts.ProjectID == uint(projectID) && ts.EmployeeID == uint(employeeID) && ts.Date.Format("2006-01-02") == dateStr {
						failedEntries = append(failedEntries, BulkUpdateFailure{
							ID:    ts.ID,
							Error: fmt.Sprintf("Lỗi kiểm tra giờ làm việc cho ngày %s", dateStr),
						})
					}
				}
				continue
			}

			// Calculate total hours after update
			var totalHours float64
			for _, ts := range existingEntries {
				totalHours += ts.HoursWorked
			}

			// Add the difference from updates
			for _, diff := range hoursDiffs {
				totalHours += diff
			}

			// Check if total exceeds 24 hours
			if totalHours > 24 {
				employee, _ := s.employeeRepo.GetByID(ctx, uint(employeeID))
				employeeName := "Unknown"
				if employee != nil {
					employeeName = employee.Fullname
				}

				// Mark all updates for this date as failed
				for i, ts := range updatedTimesheets {
					if ts.ProjectID == uint(projectID) && ts.EmployeeID == uint(employeeID) && ts.Date.Format("2006-01-02") == dateStr {
						failedEntries = append(failedEntries, BulkUpdateFailure{
							ID:    ts.ID,
							Error: fmt.Sprintf("%s: Tổng giờ làm việc ngày %s vượt quá 24 giờ. Tổng sau cập nhật: %.1f giờ", employeeName, dateStr, totalHours),
						})
						// Remove from updated list
						updatedTimesheets = append(updatedTimesheets[:i], updatedTimesheets[i+1:]...)
					}
				}
			}
		}
	}

	// Recalculate amounts for successfully validated timesheets
	var finalUpdatedTimesheets []*domain.Timesheet
	for _, ts := range updatedTimesheets {
		// Check if this timesheet was marked as failed during daily hours validation
		isFailed := false
		for _, failed := range failedEntries {
			if failed.ID == ts.ID {
				isFailed = true
				break
			}
		}

		if isFailed {
			continue
		}

		// Recalculate amount based on new hours_worked
		amount, err := s.calculationService.CalculateTimesheetAmount(ctx, ts)
		if err != nil {
			failedEntries = append(failedEntries, BulkUpdateFailure{
				ID:    ts.ID,
				Error: fmt.Sprintf("Lỗi tính toán số tiền: %s", err.Error()),
			})
			continue
		}
		ts.Amount = amount

		// Update timesheet in repository
		if err := s.timesheetRepo.Update(ctx, ts); err != nil {
			failedEntries = append(failedEntries, BulkUpdateFailure{
				ID:    ts.ID,
				Error: fmt.Sprintf("Lỗi cập nhật: %s", err.Error()),
			})
			continue
		}

		finalUpdatedTimesheets = append(finalUpdatedTimesheets, ts)
	}

	return &BulkUpdateTimesheetResult{
		UpdatedTimesheets: finalUpdatedTimesheets,
		FailedEntries:     failedEntries,
	}, nil
}
