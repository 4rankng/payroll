package bulktransfer

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	pkgConstants "api-server/internal/pkg/constants"
	"api-server/internal/pkg/timeutil"

	"github.com/google/uuid"
)

// ExportService handles bulk transfer export operations
type ExportService struct {
	timesheetRepo       TimesheetRepository
	employeeRepo        EmployeeRepository
	projectRepo         ProjectRepository
	projectEmployeeRepo ProjectEmployeeRepository
	fileRepo            BulkTransferFileRepository
	transactionCodeRepo TransactionCodeRepository
	excelService        *excel.Service
	periodCalculator    *PeriodCalculator
	checksumCalculator  *ChecksumCalculator
	eventBus            domain.EventBus
}

// NewExportService creates a new export service instance
func NewExportService(
	timesheetRepo TimesheetRepository,
	employeeRepo EmployeeRepository,
	projectRepo ProjectRepository,
	projectEmployeeRepo ProjectEmployeeRepository,
	fileRepo BulkTransferFileRepository,
	transactionCodeRepo TransactionCodeRepository,
	excelService *excel.Service,
	periodCalculator *PeriodCalculator,
	eventBus domain.EventBus,
) *ExportService {
	return &ExportService{
		timesheetRepo:       timesheetRepo,
		employeeRepo:        employeeRepo,
		projectRepo:         projectRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		fileRepo:            fileRepo,
		transactionCodeRepo: transactionCodeRepo,
		excelService:        excelService,
		periodCalculator:    periodCalculator,
		checksumCalculator:  NewChecksumCalculator(),
		eventBus:            eventBus,
	}
}

// Export generates an Excel file with bulk transfer data
func (es *ExportService) Export(ctx context.Context, req *dto.ExportBulkTransferRequest) (*dto.ExportBulkTransferResponse, error) {
	isMonthly := strings.TrimSpace(req.ForMonth) != ""
	var (
		fromDate   time.Time
		toDate     time.Time
		monthStart time.Time
		err        error
	)

	periodCache := make(map[uint]projectPeriod)
	cycle := string(domain.PaymentScheduleWeekly)

	if isMonthly {
		cycle = string(domain.PaymentScheduleMonthly)
		monthStart, err = es.periodCalculator.ResolveMonthlyRange(req)
		if err != nil {
			return nil, err
		}
		// For display purposes, use standard month range
		fromDate = monthStart
		toDate = endOfMonth(monthStart)
	} else {
		fromDate, toDate, err = es.periodCalculator.ResolveWeeklyRange(req)
		if err != nil {
			return nil, err
		}
	}

	fromDateStr := fromDate.Format(timeutil.DateFormat)
	toDateStr := toDate.Format(timeutil.DateFormat)

	// Base filters: eligible timesheets per existing rule
	filters := domain.TimesheetFilters{
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
		PaymentStatus: []domain.PaymentStatus{
			domain.PaymentStatusPending,
			domain.PaymentStatusFailed,
		},
	}
	// For weekly exports, include date filters; for monthly, dates are handled per-project
	if !isMonthly {
		filters.FromDate = &fromDate
		filters.ToDate = &toDate
	}
	if len(req.ProjectIDs) > 0 {
		filters.ProjectIDs = req.ProjectIDs
	}

	// Get all timesheets matching the date and status criteria
	allTimesheets, err := es.listTimesheetsForCycle(ctx, filters, req, isMonthly, monthStart, periodCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get timesheets: %w", err)
	}

	// Filter by employee IDs if provided in request
	baseFiltered := es.filterTimesheetsByRequest(allTimesheets, req)

	// Also fetch admin-marked timesheets that should always be included until paid
	trueVal := true
	forcedFilters := domain.TimesheetFilters{
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
		PaymentStatus: []domain.PaymentStatus{
			domain.PaymentStatusPending,
			domain.PaymentStatusFailed,
		},
		ForcePayroll: &trueVal,
	}
	if len(req.ProjectIDs) > 0 {
		forcedFilters.ProjectIDs = req.ProjectIDs
	}

	forcedAll, err := es.listTimesheetsForCycle(ctx, forcedFilters, req, isMonthly, monthStart, periodCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get forced timesheets: %w", err)
	}
	forced := es.filterTimesheetsByRequest(forcedAll, req)

	// Union forced set with baseFiltered, de-duplicated
	filteredTimesheets := make([]*domain.Timesheet, 0, len(baseFiltered)+len(forced))
	seen := make(map[uint]bool)
	for _, t := range baseFiltered {
		if !seen[t.ID] {
			filteredTimesheets = append(filteredTimesheets, t)
			seen[t.ID] = true
		}
	}
	for _, t := range forced {
		if !seen[t.ID] {
			filteredTimesheets = append(filteredTimesheets, t)
			seen[t.ID] = true
		}
	}

	// Aggregate data by employee-project combination with payment schedule filter
	aggregatedData, err := es.aggregateTimesheetData(ctx, filteredTimesheets, cycle)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate timesheet data: %w", err)
	}

	// Validate and filter the data before saving
	validationResult := es.excelService.ValidateAndFilterBulkTransferData(aggregatedData)

	// Save bulk transfer file record to database
	filename, err := es.saveBulkTransferFile(ctx, req, validationResult.ValidData, cycle, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to save bulk transfer file: %w", err)
	}

	// Generate ZIP file with bulk transfer data
	response, err := es.excelService.GenerateBulkTransferExcel(ctx, aggregatedData, req, fromDateStr, toDateStr, cycle)
	if err != nil {
		return nil, err
	}

	response.FromDate = fromDateStr
	response.ToDate = toDateStr
	response.Cycle = cycle
	response.Filename = filename

	// Audit emit: top-level "BulkTransferFileExported". Best-effort — a publish
	// failure must not fail the export itself, since the file is already
	// generated and the user is downloading it.
	es.publishExportAudit(ctx, req, filename, cycle, fromDateStr, toDateStr, validationResult.ValidData)

	return response, nil
}

// publishExportAudit emits the audit event for a successful bulk-transfer file
// export. Failures are logged but do not propagate.
func (es *ExportService) publishExportAudit(
	ctx context.Context,
	req *dto.ExportBulkTransferRequest,
	filename, cycle, fromDateStr, toDateStr string,
	validData *excel.BulkTransferData,
) {
	if es.eventBus == nil {
		return
	}

	var totalAmount int64
	var txnCount int
	if validData != nil {
		txnCount = len(validData.EmployeeProjectAmounts)
		for _, amount := range validData.EmployeeProjectAmounts {
			totalAmount += amount
		}
	}

	forMonth := strings.TrimSpace(req.ForMonth)
	fromForEvent := fromDateStr
	toForEvent := toDateStr
	if forMonth != "" {
		fromForEvent = ""
		toForEvent = ""
	}

	event := domain.NewBulkTransferFileExportedEvent(
		ctx,
		req.CreatedBy,
		filename,
		cycle,
		fromForEvent,
		toForEvent,
		forMonth,
		txnCount,
		totalAmount,
	)
	if err := es.eventBus.Publish(ctx, event); err != nil {
		observability.GetLogger().Warn("Failed to publish BulkTransferFileExported audit event",
			"filename", filename,
			"error", err)
	}
}

// filterTimesheetsByRequest filters timesheets by employee IDs if specified in request
func (es *ExportService) filterTimesheetsByRequest(allTimesheets []*domain.Timesheet, req *dto.ExportBulkTransferRequest) []*domain.Timesheet {
	if len(req.EmployeeIDs) == 0 {
		return allTimesheets
	}

	// Build a set for O(1) lookup instead of O(N) slices.Contains per iteration
	employeeSet := make(map[uint]struct{}, len(req.EmployeeIDs))
	for _, id := range req.EmployeeIDs {
		employeeSet[id] = struct{}{}
	}

	filtered := make([]*domain.Timesheet, 0, len(allTimesheets))
	for _, ts := range allTimesheets {
		if _, ok := employeeSet[ts.EmployeeID]; ok {
			filtered = append(filtered, ts)
		}
	}
	return filtered
}

// listTimesheetsForCycle retrieves timesheets with cycle-specific logic
func (es *ExportService) listTimesheetsForCycle(ctx context.Context, baseFilters domain.TimesheetFilters, req *dto.ExportBulkTransferRequest, isMonthly bool, monthStart time.Time, periodCache map[uint]projectPeriod) ([]*domain.Timesheet, error) {
	// For weekly exports, use the provided date range directly
	if !isMonthly {
		return es.timesheetRepo.List(ctx, baseFilters)
	}

	// For monthly exports, we need to query per-project based on each project's salary period
	var projectIDs []uint
	if len(req.ProjectIDs) > 0 {
		projectIDs = req.ProjectIDs
	} else {
		// Get all active projects
		projects, err := es.projectRepo.List(ctx, domain.ProjectFilters{})
		if err != nil {
			return nil, fmt.Errorf("failed to get projects: %w", err)
		}
		projectIDs = make([]uint, 0, len(projects))
		for _, project := range projects {
			projectIDs = append(projectIDs, project.ID)
		}
	}

	combined := make([]*domain.Timesheet, 0)
	seenTimesheets := make(map[uint]struct{})
	handledProjects := make(map[uint]struct{})

	for _, projectID := range projectIDs {
		if _, handled := handledProjects[projectID]; handled {
			continue
		}
		handledProjects[projectID] = struct{}{}

		period, err := es.periodCalculator.GetProjectPeriod(ctx, projectID, monthStart, periodCache)
		if err != nil {
			return nil, err
		}

		projectFilters := baseFilters
		start := period.start
		end := period.end
		projectFilters.ProjectIDs = []uint{projectID}
		projectFilters.FromDate = &start
		projectFilters.ToDate = &end

		timesheets, err := es.timesheetRepo.List(ctx, projectFilters)
		if err != nil {
			return nil, err
		}

		for _, timesheet := range timesheets {
			if _, exists := seenTimesheets[timesheet.ID]; exists {
				continue
			}
			combined = append(combined, timesheet)
			seenTimesheets[timesheet.ID] = struct{}{}
		}
	}

	return combined, nil
}

// aggregateTimesheetData groups timesheets by employee-project combination and fetches related data
func (es *ExportService) aggregateTimesheetData(ctx context.Context, timesheets []*domain.Timesheet, cycle string) (*excel.BulkTransferData, error) {
	employeeProjectAmounts := make(map[excel.EmployeeProjectKey]int64)
	employeeProjectTimesheets := make(map[excel.EmployeeProjectKey][]uint)
	employeeData := make(map[uint]excel.Employee)
	projectData := make(map[uint]excel.Project)
	transactionCodes := make(map[excel.EmployeeProjectKey]string)
	assignmentCache := make(map[excel.EmployeeProjectKey]*domain.ProjectEmployee)

	var paymentScheduleFilter *string
	if cycle != "" {
		paymentScheduleFilter = &cycle
	}

	for _, timesheet := range timesheets {
		key := excel.EmployeeProjectKey{
			EmployeeID: timesheet.EmployeeID,
			ProjectID:  timesheet.ProjectID,
		}

		// If payment schedule filter is provided, check employee's payment schedule
		if paymentScheduleFilter != nil && *paymentScheduleFilter != "" {
			assignment, ok := assignmentCache[key]
			if !ok {
				var err error
				assignment, err = es.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, timesheet.ProjectID, timesheet.EmployeeID)
				if err != nil {
					// Skip if assignment not found
					continue
				}
				assignmentCache[key] = assignment
			}

			if assignment == nil || assignment.PaymentSchedule != *paymentScheduleFilter {
				continue
			}
		}

		employeeProjectAmounts[key] += timesheet.Amount
		employeeProjectTimesheets[key] = append(employeeProjectTimesheets[key], timesheet.ID)

		// Generate transaction code for this key if not already generated.
		// Use first 8 chars of UUID — shorter but still unique (DB enforces
		// uniqueness). Format is alphanumeric only (no hyphen): this code
		// flows through to 9pay as request_id, and 9pay rejects hyphens
		// with error 318.
		if _, exists := transactionCodes[key]; !exists {
			txUUID := uuid.New()
			txShort := strings.ReplaceAll(txUUID.String(), "-", "")[:8]
			if paymentScheduleFilter != nil && *paymentScheduleFilter == string(domain.PaymentScheduleFlexible) {
				transactionCodes[key] = fmt.Sprintf("tt%s", txShort)
			} else {
				transactionCodes[key] = fmt.Sprintf("VFIC%s", txShort)
			}
		}

		// Get employee data if not already fetched
		if _, exists := employeeData[timesheet.EmployeeID]; !exists {
			employee, err := es.employeeRepo.GetByID(ctx, timesheet.EmployeeID)
			if err != nil {
				return nil, fmt.Errorf("failed to get employee %d: %w", timesheet.EmployeeID, err)
			}

			// Convert to excel.Employee
			excelEmployee := excel.Employee{
				ID:                employee.ID,
				Fullname:          employee.FormattedFullname(),
				BankAccountNumber: employee.BankAccountNumber,
				BankAccountName:   employee.BankAccountName,
			}
			if employee.Bank != nil {
				excelEmployee.Bank = &excel.Bank{
					ID:         employee.Bank.ID,
					BranchName: employee.Bank.BranchName,
					BankCode:   employee.Bank.BankCode,
				}
			}
			employeeData[timesheet.EmployeeID] = excelEmployee
		}

		// Get project data if not already fetched
		if _, exists := projectData[timesheet.ProjectID]; !exists {
			project, err := es.projectRepo.GetByID(ctx, timesheet.ProjectID)
			if err != nil {
				return nil, fmt.Errorf("failed to get project %d: %w", timesheet.ProjectID, err)
			}
			projectData[timesheet.ProjectID] = excel.Project{
				ID:   project.ID,
				Name: project.Name,
			}
		}
	}

	return &excel.BulkTransferData{
		EmployeeProjectAmounts:    employeeProjectAmounts,
		EmployeeProjectTimesheets: employeeProjectTimesheets,
		EmployeeData:              employeeData,
		ProjectData:               projectData,
		TransactionCodes:          transactionCodes,
	}, nil
}

// saveBulkTransferFile saves the bulk transfer export request to database
func (es *ExportService) saveBulkTransferFile(
	ctx context.Context,
	req *dto.ExportBulkTransferRequest,
	data *excel.BulkTransferData,
	cycle string,
	fromDate, toDate time.Time,
) (string, error) {
	// Generate UUID for filename
	fileUUID := uuid.New()
	uuidStr := strings.ReplaceAll(fileUUID.String(), "-", "")

	// Create filename: {bankprefix}_{cycle}_{uuid_no_hyphen}
	filename := fmt.Sprintf("%s_%s_%s", pkgConstants.MBankPrefix, cycle, uuidStr)

	// Convert data to BulkTransferFileData array
	var fileDataArray []dto.BulkTransferFileData
	var totalAmount int64
	stt := 1

	// Prepare transaction codes for batch creation (created after file)
	var transactionCodesToCreate []*domain.TransactionCode

	if data != nil {
		for key, amount := range data.EmployeeProjectAmounts {
			employee := data.EmployeeData[key.EmployeeID]
			project := data.ProjectData[key.ProjectID]
			timesheetIDs := data.EmployeeProjectTimesheets[key]
			transactionCode := data.TransactionCodes[key]

			bankName := ""
			if employee.Bank != nil {
				bankName = employee.Bank.BranchName
			}

			fileData := dto.BulkTransferFileData{
				STT:             stt,
				EmployeeID:      employee.ID,
				ProjectID:       project.ID,
				TimesheetIDs:    timesheetIDs,
				AccountNumber:   employee.BankAccountNumber,
				AccountName:     employee.BankAccountName,
				BankName:        bankName,
				Amount:          amount,
				TransactionCode: transactionCode,
			}
			fileDataArray = append(fileDataArray, fileData)
			totalAmount += amount
			stt++

			// Prepare transaction code data
			var tcData domain.TransactionCodeData
			if cycle == string(domain.PaymentScheduleMonthly) {
				tcData = domain.TransactionCodeData{
					MonthlyPay: &domain.CyclePayData{
						TimesheetIDs: timesheetIDs,
						EmployeeID:   employee.ID,
						ProjectID:    project.ID,
						Amount:       amount,
					},
				}
			} else {
				tcData = domain.TransactionCodeData{
					WeeklyPay: &domain.CyclePayData{
						TimesheetIDs: timesheetIDs,
						EmployeeID:   employee.ID,
						ProjectID:    project.ID,
						Amount:       amount,
					},
				}
			}
			tcDataBytes, _ := json.Marshal(tcData)
			transactionCodesToCreate = append(transactionCodesToCreate, &domain.TransactionCode{
				Code: transactionCode,
				Data: tcDataBytes,
			})
		}
	}

	// Sort by STT to ensure consistent ordering
	sort.Slice(fileDataArray, func(i, j int) bool {
		return fileDataArray[i].STT < fileDataArray[j].STT
	})

	// Create transaction codes (FileID will be set when result is uploaded)
	if len(transactionCodesToCreate) > 0 {
		if err := es.transactionCodeRepo.CreateBatch(ctx, transactionCodesToCreate); err != nil {
			return "", fmt.Errorf("failed to create transaction codes: %w", err)
		}
	}

	return filename, nil
}
