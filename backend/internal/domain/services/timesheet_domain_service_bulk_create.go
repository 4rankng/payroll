package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// BulkCreateTimesheetEntry represents a single timesheet creation request for bulk operations
type BulkCreateTimesheetEntry struct {
	ProjectID   uint    `json:"project_id"`
	EmployeeID  uint    `json:"employee_id"`
	Date        string  `json:"date"`
	HoursWorked float64 `json:"hours_worked"`
	HourType    string  `json:"hour_type"`
	DayType     *string `json:"day_type,omitempty"`
}

// BulkCreateTimesheetResult represents the result of bulk timesheet creation
type BulkCreateTimesheetResult struct {
	CreatedTimesheets []*domain.Timesheet
	DeletedTimesheets []*domain.Timesheet
	FailedEntries     []BulkCreateFailure
}

// BulkCreateFailure represents a failed timesheet creation in bulk operation
type BulkCreateFailure struct {
	Index   int
	Error   string
	Request BulkCreateTimesheetEntry
}

// BulkCreateTimesheets handles bulk timesheet upsert (create or update) with complete business logic
func (s *TimesheetDomainService) BulkCreateTimesheets(ctx context.Context, requests []BulkCreateTimesheetEntry, createdBy uint, userRole string) (*BulkCreateTimesheetResult, error) {
	logger := observability.GetLogger()
	logger.Info("TimesheetDomainService: BulkCreateTimesheets started",
		"total_requests", len(requests),
		"created_by", createdBy,
		"user_role", userRole)

	// Log request details for debugging
	for i, req := range requests {
		dayTypeStr := "nil"
		if req.DayType != nil {
			dayTypeStr = *req.DayType
		}
		logger.Info("TimesheetDomainService: Processing entry",
			"index", i,
			"project_id", req.ProjectID,
			"employee_id", req.EmployeeID,
			"date", req.Date,
			"hours_worked", req.HoursWorked,
			"hour_type", req.HourType,
			"day_type", dayTypeStr)
	}

	// Validate for duplicates within the request batch
	seen := make(map[string][]int) // key -> list of indices
	for i, req := range requests {
		// Create unique key for timesheet: projectId-employeeId-date-hourType-dayType
		dayType := ""
		if req.DayType != nil {
			dayType = *req.DayType
		}
		key := fmt.Sprintf("%d-%d-%s-%s-%s", req.ProjectID, req.EmployeeID, req.Date, req.HourType, dayType)
		seen[key] = append(seen[key], i)
	}

	var succeededTimesheets []*domain.Timesheet
	var deletedTimesheets []*domain.Timesheet
	var failedEntries []BulkCreateFailure

	// Check for duplicates within the request and mark them as failed
	for _, indices := range seen {
		if len(indices) > 1 {
			// Mark all duplicates after the first one as failed
			for _, idx := range indices[1:] {
				req := requests[idx]
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   idx,
					Error:   fmt.Sprintf("Mục nhập trùng lặp trong yêu cầu: Nhân viên ID %d, Dự án ID %d, Ngày %s, Loại giờ '%s' xuất hiện nhiều lần trong yêu cầu này", req.EmployeeID, req.ProjectID, req.Date, req.HourType),
					Request: req,
				})
			}
		}
	}

	// Prefetch validation data for better performance
	// 1. Collect unique employee IDs and employee/date combos
	employeeIDSet := make(map[uint]bool)
	var combos []domain.EmployeeDateCombo
	dailyTotals := make(map[string]float64) // key: projectId-employeeId-date -> totalHours

	for _, req := range requests {
		employeeIDSet[req.EmployeeID] = true

		date, err := time.ParseInLocation("2006-01-02", req.Date, time.Local)
		if err == nil {
			combos = append(combos, domain.EmployeeDateCombo{
				EmployeeID: req.EmployeeID,
				ProjectID:  req.ProjectID,
				Date:       date,
			})
		}

		key := fmt.Sprintf("%d-%d-%s", req.ProjectID, req.EmployeeID, req.Date)
		dailyTotals[key] += req.HoursWorked
	}

	// 2. Batch fetch all employees for error messages
	var employeeIDList []int64
	for id := range employeeIDSet {
		employeeIDList = append(employeeIDList, int64(id))
	}

	employeeCache := make(map[uint]string)
	if len(employeeIDList) > 0 {
		employees, err := s.employeeRepo.GetByIDs(ctx, employeeIDList)
		if err == nil {
			for _, emp := range employees {
				if emp != nil {
					employeeCache[emp.ID] = emp.Fullname
				}
			}
		}
	}

	// 3. Batch prefetch all unique (projectID, employeeID) assignments to avoid per-entry DB calls
	assignmentCache := make(map[string]*domain.ProjectEmployee) // key: "projectID-employeeID"
	{
		projectIDSet := make(map[uint]bool)
		for _, req := range requests {
			projectIDSet[req.ProjectID] = true
		}
		var projectIDList []uint
		for id := range projectIDSet {
			projectIDList = append(projectIDList, id)
		}
		var employeeIDListUint []uint
		for id := range employeeIDSet {
			employeeIDListUint = append(employeeIDListUint, id)
		}
		if len(projectIDList) > 0 && len(employeeIDListUint) > 0 {
			assignments, err := s.projectEmployeeRepo.GetActiveAssignmentsByProjectsAndEmployees(ctx, projectIDList, employeeIDListUint)
			if err == nil {
				for _, a := range assignments {
					key := fmt.Sprintf("%d-%d", a.ProjectID, a.EmployeeID)
					// Keep the most recently created assignment per pair (repo returns ordered by created_at DESC)
					if _, exists := assignmentCache[key]; !exists {
						assignmentCache[key] = a
					}
				}
			}
		}
	}

	// 4. Batch fetch all existing timesheets in a single query for upsert detection
	existingTimesheetsMap := make(map[string][]*domain.Timesheet) // key: projectId-employeeId-date
	existingByPaytypeMap := make(map[string]*domain.Timesheet)    // key: projectId-employeeId-date-paytype
	if len(combos) > 0 {
		existingTimesheets, err := s.timesheetRepo.GetByEmployeeDateCombos(ctx, combos)
		if err == nil {
			for _, ts := range existingTimesheets {
				key := fmt.Sprintf("%d-%d-%s", ts.ProjectID, ts.EmployeeID, ts.Date.Format("2006-01-02"))
				existingTimesheetsMap[key] = append(existingTimesheetsMap[key], ts)

				// Build paytype map for exact matching
				paytypeKey := fmt.Sprintf("%d-%d-%s-%s", ts.ProjectID, ts.EmployeeID, ts.Date.Format("2006-01-02"), ts.PayType)
				existingByPaytypeMap[paytypeKey] = ts
			}
		}
	}

	// Validate each daily total doesn't exceed 24 hours when combined with existing entries
	// For upsert, we need to subtract the old hours if we're updating
	for key, batchTotal := range dailyTotals {
		parts := strings.Split(key, "-")
		if len(parts) < 3 {
			continue
		}

		projectID, _ := strconv.Atoi(parts[0])
		employeeID, _ := strconv.Atoi(parts[1])
		dateStr := strings.Join(parts[2:], "-")

		// Get existing entries from prefetched map
		existingEntries := existingTimesheetsMap[key]

		// Calculate existing total hours, excluding entries that will be updated or deleted
		var existingTotal float64
		for _, ts := range existingEntries {
			// Check if this existing entry will be updated or deleted by any request
			willBeModified := false
			for i, req := range requests {
				// Skip failed entries
				skipDueToFailure := false
				for _, failedItem := range failedEntries {
					if failedItem.Index == i {
						skipDueToFailure = true
						break
					}
				}
				if skipDueToFailure {
					continue
				}

				// Build paytype for this request to check for match
				dayType := ""
				if req.DayType != nil {
					dayType = *req.DayType
				}

				date, err := time.ParseInLocation("2006-01-02", req.Date, time.Local)
				if err == nil {
					assignKey := fmt.Sprintf("%d-%d", req.ProjectID, req.EmployeeID)
					var payType string
					if assignment, ok := assignmentCache[assignKey]; ok {
						payType, err = s.paytypeConstructionService.ConstructPaytypeFromAssignment(assignment, date, req.HourType, dayType)
					} else {
						payType, err = s.paytypeConstructionService.ConstructPaytype(ctx, req.ProjectID, req.EmployeeID, date, req.HourType, dayType)
					}
					if err == nil && ts.ProjectID == req.ProjectID && ts.EmployeeID == req.EmployeeID &&
						ts.Date.Format("2006-01-02") == req.Date && ts.PayType == payType {
						// Entry will be updated or deleted (hoursWorked=0 means delete)
						willBeModified = true
						break
					}
				}
			}

			if !willBeModified {
				existingTotal += ts.HoursWorked
			}
		}

		// Check if batch + existing exceeds 24 hours
		totalHours := batchTotal + existingTotal
		if totalHours > 24 {
			// Get employee name from cache
			employeeName := "Unknown"
			if name, ok := employeeCache[uint(employeeID)]; ok {
				employeeName = name
			}

			// Mark all entries for this date as failed
			for i, req := range requests {
				if req.ProjectID == uint(projectID) && req.EmployeeID == uint(employeeID) && req.Date == dateStr {
					failedEntries = append(failedEntries, BulkCreateFailure{
						Index: i,
						Error: fmt.Sprintf("%s: Tổng giờ làm việc ngày %s vượt quá 24 giờ. Hiện tại: %.1f giờ",
							employeeName, dateStr, totalHours),
						Request: req,
					})
				}
			}
		}
	}

	for i, req := range requests {
		// Skip if this entry was already marked as failed
		skipDueToFailure := false
		for _, failedItem := range failedEntries {
			if failedItem.Index == i {
				skipDueToFailure = true
				break
			}
		}
		if skipDueToFailure {
			continue
		}

		// Parse date string to time.Time
		date, err := time.ParseInLocation("2006-01-02", req.Date, time.Local)
		if err != nil {
			failedEntries = append(failedEntries, BulkCreateFailure{
				Index:   i,
				Error:   "Định dạng ngày không hợp lệ",
				Request: req,
			})
			continue
		}

		// 1. Construct paytype using domain service
		dayType := ""
		if req.DayType != nil {
			dayType = *req.DayType
		}

		logger := observability.GetLogger()
		logger.Info("TimesheetDomainService: Constructing paytype",
			"index", i,
			"project_id", req.ProjectID,
			"employee_id", req.EmployeeID,
			"hour_type", req.HourType,
			"day_type", dayType)

		assignKey := fmt.Sprintf("%d-%d", req.ProjectID, req.EmployeeID)
		var payType string
		if assignment, ok := assignmentCache[assignKey]; ok {
			payType, err = s.paytypeConstructionService.ConstructPaytypeFromAssignment(assignment, date, req.HourType, dayType)
		} else {
			payType, err = s.paytypeConstructionService.ConstructPaytype(ctx, req.ProjectID, req.EmployeeID, date, req.HourType, dayType)
		}
		if err != nil {
			logger.Error("TimesheetDomainService: Paytype construction failed",
				"index", i,
				"project_id", req.ProjectID,
				"employee_id", req.EmployeeID,
				"hour_type", req.HourType,
				"day_type", dayType,
				"error", err.Error(),
				"error_type", fmt.Sprintf("%T", err))
			failedEntries = append(failedEntries, BulkCreateFailure{
				Index:   i,
				Error:   err.Error(),
				Request: req,
			})
			continue
		}

		logger.Info("TimesheetDomainService: Paytype constructed successfully",
			"index", i,
			"paytype", payType)

		// 2a. Zero-rate check — reject if the payrate bucket is 0 (project doesn't pay for this combination)
		if req.HoursWorked > 0 {
			if zeroRateErr := s.validationService.ValidateZeroRate(ctx, req.ProjectID, date, payType, req.EmployeeID); zeroRateErr != nil {
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   zeroRateErr.Message,
					Request: req,
				})
				continue
			}
		}

		// 2b. Check if timesheet exists (upsert logic)
		paytypeKey := fmt.Sprintf("%d-%d-%s-%s", req.ProjectID, req.EmployeeID, req.Date, payType)
		existingTimesheet, exists := existingByPaytypeMap[paytypeKey]

		if exists {
			// DELETE path: Check if hoursWorked=0 (delete intent)
			if req.HoursWorked == 0 {
				// Validate deletion permissions
				if existingTimesheet.IsPaid() {
					detailedError := s.validationService.GenerateDetailedPaidTimesheetError(ctx, existingTimesheet)
					failedEntries = append(failedEntries, BulkCreateFailure{
						Index:   i,
						Error:   detailedError,
						Request: req,
					})
					continue
				}

				// Partners cannot delete approved timesheets
				if existingTimesheet.IsApproved() && userRole == "partner" {
					failedEntries = append(failedEntries, BulkCreateFailure{
						Index:   i,
						Error:   "Không thể xóa bảng chấm công đã được phê duyệt",
						Request: req,
					})
					continue
				}

				// Perform deletion
				if err := s.timesheetRepo.Delete(ctx, existingTimesheet.ID); err != nil {
					failedEntries = append(failedEntries, BulkCreateFailure{
						Index:   i,
						Error:   fmt.Sprintf("Lỗi xóa: %s", err.Error()),
						Request: req,
					})
					continue
				}

				deletedTimesheets = append(deletedTimesheets, existingTimesheet)
				continue
			}

			// UPDATE path: Validate update permissions
			if existingTimesheet.IsPaid() {
				detailedError := s.validationService.GenerateDetailedPaidTimesheetError(ctx, existingTimesheet)
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   detailedError,
					Request: req,
				})
				continue
			}

			// Partners cannot edit approved timesheets
			if existingTimesheet.IsApproved() && userRole == "partner" {
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   "Không thể chỉnh sửa bảng chấm công đã được phê duyệt",
					Request: req,
				})
				continue
			}

			// Update the existing timesheet
			existingTimesheet.HoursWorked = req.HoursWorked
			existingTimesheet.PayType = payType

			// Validate updated fields
			if err := existingTimesheet.ValidateHoursWorked(); err != nil {
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   err.Error(),
					Request: req,
				})
				continue
			}

			if err := existingTimesheet.ValidatePayType(); err != nil {
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   err.Error(),
					Request: req,
				})
				continue
			}

			// Validate daytype consistency
			if err := s.validationService.ValidateDaytypeConsistency(ctx, existingTimesheet); err != nil {
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   err.Error(),
					Request: req,
				})
				continue
			}

			// Recalculate amount
			amount, err := s.calculationService.CalculateTimesheetAmount(ctx, existingTimesheet)
			if err != nil {
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   fmt.Sprintf("Lỗi tính toán số tiền: %s", err.Error()),
					Request: req,
				})
				continue
			}
			existingTimesheet.Amount = amount

			// Update in repository
			if err := s.timesheetRepo.Update(ctx, existingTimesheet); err != nil {
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   fmt.Sprintf("Lỗi cập nhật: %s", err.Error()),
					Request: req,
				})
				continue
			}

			succeededTimesheets = append(succeededTimesheets, existingTimesheet)
		} else {
			// Silently skip if hoursWorked=0 and entry doesn't exist (no-op)
			if req.HoursWorked == 0 {
				continue
			}

			// CREATE path: Standard creation logic
			timesheet := &domain.Timesheet{
				ProjectID:   req.ProjectID,
				EmployeeID:  req.EmployeeID,
				Date:        date,
				HoursWorked: req.HoursWorked,
				PayType:     payType,
			}

			// Validate timesheet using domain service
			logger := observability.GetLogger()
			logger.Info("TimesheetDomainService: Validating timesheet",
				"index", i,
				"project_id", timesheet.ProjectID,
				"employee_id", timesheet.EmployeeID,
				"date", timesheet.Date.Format("2006-01-02"),
				"hours_worked", timesheet.HoursWorked,
				"pay_type", timesheet.PayType)

			if err := s.validationService.ValidateTimesheet(ctx, timesheet); err != nil {
				logger.Error("TimesheetDomainService: Timesheet validation failed",
					"index", i,
					"project_id", timesheet.ProjectID,
					"employee_id", timesheet.EmployeeID,
					"date", timesheet.Date.Format("2006-01-02"),
					"hours_worked", timesheet.HoursWorked,
					"pay_type", timesheet.PayType,
					"error", err.Error(),
					"error_type", fmt.Sprintf("%T", err))
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   err.Error(),
					Request: req,
				})
				continue
			}

			logger.Info("TimesheetDomainService: Timesheet validation passed",
				"index", i)

			// Calculate amount using domain service
			amount, err := s.calculationService.CalculateTimesheetAmount(ctx, timesheet)
			if err != nil {
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   err.Error(),
					Request: req,
				})
				continue
			}
			timesheet.Amount = amount

			// Set audit fields and status based on role BEFORE calling repository
			timesheet.CreatedBy = createdBy
			if err := s.SetInitialTimesheetStatus(ctx, timesheet, createdBy, userRole); err != nil {
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   err.Error(),
					Request: req,
				})
				continue
			}

			// Create timesheet using repository (status is already set)
			if err := s.timesheetRepo.Create(ctx, timesheet); err != nil {
				failedEntries = append(failedEntries, BulkCreateFailure{
					Index:   i,
					Error:   err.Error(),
					Request: req,
				})
				continue
			}

			succeededTimesheets = append(succeededTimesheets, timesheet)
		}
	}

	logger.Info("TimesheetDomainService: BulkCreateTimesheets completed",
		"created_count", len(succeededTimesheets),
		"deleted_count", len(deletedTimesheets),
		"failed_count", len(failedEntries))

	return &BulkCreateTimesheetResult{
		CreatedTimesheets: succeededTimesheets,
		DeletedTimesheets: deletedTimesheets,
		FailedEntries:     failedEntries,
	}, nil
}
