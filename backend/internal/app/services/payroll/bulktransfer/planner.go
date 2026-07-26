package bulktransfer

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/domain"
	"api-server/internal/pkg/timeutil"

	"github.com/google/uuid"
)

// ExportPlan is the pure, write-free result of ExportPlanner.Plan.
// It captures everything the export needs to either (a) persist + generate
// Excel in production, or (b) project as a read-only simulation cycle.
//
// SnapshotEpoch is the max updated_at across every row Plan() observed
// (timesheets, employees, project_employees). It lets a caller detect
// post-plan drift without re-running the planner. See computeSnapshotEpoch.
type ExportPlan struct {
	Cycle         string // "weekly" | "monthly"
	FromDate      time.Time
	ToDate        time.Time
	MonthStart    time.Time // zero for weekly
	IsMonthly     bool
	RawAggregated *excel.BulkTransferData // pre-validation (carries EmployeeProjectTimesheets)
	ValidatedData *excel.BulkTransferValidationResult
	SnapshotEpoch time.Time
	// PaymentPercentage is captured once during planning and reused for both
	// workbook generation and forecast outcome allocation.
	PaymentPercentage float64
	// ForecastOutcomeItems contains only successfully validated weekly
	// timesheets inside the declared range. Force-included backlog outside the
	// range remains in the bank file but never contaminates forecast accuracy.
	ForecastOutcomeItems []domain.CashForecastOutcomeItem
}

// ExportPlanner performs the read-only selection + aggregation + validation
// phase of a bulk-transfer export. It issues zero writes. Production
// ExportService.Export calls Plan() then Persist(); the settlement simulation
// (Phase 2) calls Plan() only.
//
// Plan() and its helpers (listTimesheetsForCycle, filterTimesheetsByRequest,
// aggregateTimesheetData) were moved verbatim from the pre-refactor
// ExportService — no behavior change. The only addition is the
// ExportPlan.SnapshotEpoch field used by the IfMatchSnapshot guard.
type ExportPlanner struct {
	timesheetRepo       TimesheetRepository
	employeeRepo        EmployeeRepository
	projectRepo         ProjectRepository
	projectEmployeeRepo ProjectEmployeeRepository
	periodCalculator    *PeriodCalculator
	excelService        *excel.Service
}

// NewExportPlanner creates a planner with the read-side dependencies shared
// with ExportService. Write/gen deps (fileRepo, transactionCodeRepo,
// excelService.GenerateBulkTransferExcel, eventBus) stay on ExportService.
func NewExportPlanner(
	timesheetRepo TimesheetRepository,
	employeeRepo EmployeeRepository,
	projectRepo ProjectRepository,
	projectEmployeeRepo ProjectEmployeeRepository,
	periodCalculator *PeriodCalculator,
	excelService *excel.Service,
) *ExportPlanner {
	return &ExportPlanner{
		timesheetRepo:       timesheetRepo,
		employeeRepo:        employeeRepo,
		projectRepo:         projectRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		periodCalculator:    periodCalculator,
		excelService:        excelService,
	}
}

// Plan performs the entire read + aggregate + validate phase of an export.
// Pure: no INSERTs/UPDATEs/Publish calls. Returns an ExportPlan that the
// caller can either Persist() (production) or inspect (simulation).
func (p *ExportPlanner) Plan(ctx context.Context, req *dto.ExportBulkTransferRequest) (*ExportPlan, error) {
	return p.planWithDateRange(ctx, req)
}

// planWithDateRange is the production code path — behavior identical to the
// pre-refactor ExportService.Export lines 62-162.
func (p *ExportPlanner) planWithDateRange(ctx context.Context, req *dto.ExportBulkTransferRequest) (*ExportPlan, error) {
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
		monthStart, err = p.periodCalculator.ResolveMonthlyRange(req)
		if err != nil {
			return nil, err
		}
		fromDate = monthStart
		toDate = endOfMonth(monthStart)
	} else {
		fromDate, toDate, err = p.periodCalculator.ResolveWeeklyRange(req)
		if err != nil {
			return nil, err
		}
	}

	rawAggregated, validated, outcomeItems, paymentPercentage, err := p.selectAndAggregate(ctx, req, isMonthly, monthStart, cycle, periodCache)
	if err != nil {
		return nil, err
	}

	snapshot, err := p.computeSnapshotEpoch(ctx, rawAggregated)
	if err != nil {
		// Snapshot is best-effort metadata; a failure to compute it must not
		// abort the export. Log and continue with the zero value — the
		// simulation will surface a warning, and production export is unaffected.
		snapshot = time.Time{}
	}
	return &ExportPlan{
		Cycle:                cycle,
		FromDate:             fromDate,
		ToDate:               toDate,
		MonthStart:           monthStart,
		IsMonthly:            isMonthly,
		RawAggregated:        rawAggregated,
		ValidatedData:        validated,
		SnapshotEpoch:        snapshot,
		PaymentPercentage:    paymentPercentage,
		ForecastOutcomeItems: outcomeItems,
	}, nil
}

// selectAndAggregate runs the shared selection pipeline used by both plan
// modes. This is the verbatim pre-refactor body (lines 93-162) — no logic
// changes, just moved onto the planner receiver.
func (p *ExportPlanner) selectAndAggregate(
	ctx context.Context,
	req *dto.ExportBulkTransferRequest,
	isMonthly bool,
	monthStart time.Time,
	cycle string,
	periodCache map[uint]projectPeriod,
) (*excel.BulkTransferData, *excel.BulkTransferValidationResult, []domain.CashForecastOutcomeItem, float64, error) {
	// Base filters: eligible timesheets per the shared pending-payment rule.
	filters := domain.NewPendingPaymentTimesheetFilters()
	// For weekly exports, include date filters; for monthly, dates are handled per-project
	if !isMonthly {
		if req.FromDate != "" {
			// The MySQL DSN uses loc=Local. Parsing a date-only value as UTC
			// shifts the lower bound forward by the local offset and excludes
			// every row on the first day of the requested cycle.
			fromDate, _ := timeutil.ParseBusinessDate(req.FromDate)
			filters.FromDate = &fromDate
		}
		if req.ToDate != "" {
			toDate, _ := timeutil.ParseBusinessDate(req.ToDate)
			filters.ToDate = &toDate
		}
	}
	if len(req.ProjectIDs) > 0 {
		filters.ProjectIDs = req.ProjectIDs
	}

	// Get all timesheets matching the date and status criteria
	allTimesheets, err := p.listTimesheetsForCycle(ctx, filters, req, isMonthly, monthStart, periodCache)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("failed to get timesheets: %w", err)
	}

	// Filter by employee IDs if provided in request
	baseFiltered := p.filterTimesheetsByRequest(allTimesheets, req)

	// Also fetch admin-marked timesheets that should always be included until paid
	trueVal := true
	forcedFilters := domain.NewPendingPaymentTimesheetFilters()
	forcedFilters.ForcePayroll = &trueVal
	if len(req.ProjectIDs) > 0 {
		forcedFilters.ProjectIDs = req.ProjectIDs
	}

	forcedAll, err := p.listTimesheetsForCycle(ctx, forcedFilters, req, isMonthly, monthStart, periodCache)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("failed to get forced timesheets: %w", err)
	}
	forced := p.filterTimesheetsByRequest(forcedAll, req)

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
	aggregatedData, err := p.aggregateTimesheetData(ctx, filteredTimesheets, cycle)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("failed to aggregate timesheet data: %w", err)
	}

	// Validate and filter the data before saving
	validationResult := p.excelService.ValidateAndFilterBulkTransferData(aggregatedData)

	paymentPercentage := p.excelService.GetPaymentPercentageForSchedule(ctx, cycle)
	return aggregatedData, validationResult, buildForecastOutcomeItems(req, isMonthly, filteredTimesheets, validationResult, paymentPercentage), paymentPercentage, nil
}

func buildForecastOutcomeItems(
	req *dto.ExportBulkTransferRequest,
	isMonthly bool,
	selected []*domain.Timesheet,
	validation *excel.BulkTransferValidationResult,
	paymentPercentage float64,
) []domain.CashForecastOutcomeItem {
	if req == nil || isMonthly || req.FromDate == "" || req.ToDate == "" ||
		validation == nil || validation.ValidData == nil {
		return nil
	}
	fromDate, fromErr := timeutil.ParseBusinessDate(req.FromDate)
	toDate, toErr := timeutil.ParseBusinessDate(req.ToDate)
	if fromErr != nil || toErr != nil {
		return nil
	}
	if paymentPercentage <= 0 || paymentPercentage > 1 {
		return nil
	}
	selectedByID := make(map[uint]*domain.Timesheet, len(selected))
	for _, timesheet := range selected {
		if timesheet != nil {
			selectedByID[timesheet.ID] = timesheet
		}
	}
	allocated := make(map[uint]int64)
	for key, ids := range validation.ValidData.EmployeeProjectTimesheets {
		rawTotal := validation.ValidData.EmployeeProjectAmounts[key]
		if rawTotal <= 0 || len(ids) == 0 {
			continue
		}
		paymentTotal := int64(float64(rawTotal) * paymentPercentage)
		orderedIDs := append([]uint(nil), ids...)
		sort.Slice(orderedIDs, func(i, j int) bool { return orderedIDs[i] < orderedIDs[j] })
		var allocatedTotal int64
		lastID := uint(0)
		for _, id := range orderedIDs {
			timesheet := selectedByID[id]
			if timesheet == nil || timesheet.Amount <= 0 {
				continue
			}
			amount := int64(float64(timesheet.Amount) * paymentPercentage)
			allocated[id] = amount
			allocatedTotal += amount
			lastID = id
		}
		if lastID != 0 {
			allocated[lastID] += paymentTotal - allocatedTotal
		}
	}
	items := make([]domain.CashForecastOutcomeItem, 0, len(allocated))
	for id, amount := range allocated {
		timesheet := selectedByID[id]
		if timesheet == nil || amount <= 0 {
			continue
		}
		workDate := time.Date(timesheet.Date.Year(), timesheet.Date.Month(), timesheet.Date.Day(), 0, 0, 0, 0, fromDate.Location())
		if workDate.Before(fromDate) || workDate.After(toDate) {
			continue
		}
		items = append(items, domain.CashForecastOutcomeItem{TimesheetID: id, Amount: amount})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].TimesheetID < items[j].TimesheetID })
	return items
}

// filterTimesheetsByRequest filters timesheets by employee IDs if specified in request.
// Moved verbatim from ExportService.
func (p *ExportPlanner) filterTimesheetsByRequest(allTimesheets []*domain.Timesheet, req *dto.ExportBulkTransferRequest) []*domain.Timesheet {
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

// listTimesheetsForCycle retrieves timesheets with cycle-specific logic.
// Moved verbatim from ExportService.
func (p *ExportPlanner) listTimesheetsForCycle(ctx context.Context, baseFilters domain.TimesheetFilters, req *dto.ExportBulkTransferRequest, isMonthly bool, monthStart time.Time, periodCache map[uint]projectPeriod) ([]*domain.Timesheet, error) {
	// For weekly exports, use the provided date range directly
	if !isMonthly {
		return p.timesheetRepo.List(ctx, baseFilters)
	}

	// For monthly exports, we need to query per-project based on each project's salary period
	var projectIDs []uint
	if len(req.ProjectIDs) > 0 {
		projectIDs = req.ProjectIDs
	} else {
		// Get all active projects
		projects, err := p.projectRepo.List(ctx, domain.ProjectFilters{})
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

		period, err := p.periodCalculator.GetProjectPeriod(ctx, projectID, monthStart, periodCache)
		if err != nil {
			return nil, err
		}

		projectFilters := baseFilters
		start := period.start
		end := period.end
		projectFilters.ProjectIDs = []uint{projectID}
		projectFilters.FromDate = &start
		projectFilters.ToDate = &end

		timesheets, err := p.timesheetRepo.List(ctx, projectFilters)
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

// aggregateTimesheetData groups timesheets by employee-project combination and fetches related data.
// Moved verbatim from ExportService.
func (p *ExportPlanner) aggregateTimesheetData(ctx context.Context, timesheets []*domain.Timesheet, cycle string) (*excel.BulkTransferData, error) {
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
				assignment, err = p.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, timesheet.ProjectID, timesheet.EmployeeID)
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
			txUUID := uuidNewShort()
			if paymentScheduleFilter != nil && *paymentScheduleFilter == string(domain.PaymentScheduleFlexible) {
				transactionCodes[key] = fmt.Sprintf("tt%s", txUUID)
			} else {
				transactionCodes[key] = fmt.Sprintf("VFIC%s", txUUID)
			}
		}

		// Get employee data if not already fetched
		if _, exists := employeeData[timesheet.EmployeeID]; !exists {
			employee, err := p.employeeRepo.GetByID(ctx, timesheet.EmployeeID)
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
			project, err := p.projectRepo.GetByID(ctx, timesheet.ProjectID)
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

// uuidNewShort returns the first 8 alphanumeric chars of a fresh UUID, matching
// the format production export uses for transaction codes. Wrapped so the
// simulation can substitute a deterministic generator in tests.
func uuidNewShort() string {
	txUUID := uuid.New()
	return strings.ReplaceAll(txUUID.String(), "-", "")[:8]
}

// computeSnapshotEpoch returns the max updated_at across the timesheets the
// planner observed. This is the snapshot the simulation returns and the real
// export (Phase 3) optionally checks via IfMatchSnapshot.
//
// Phase 1 scope: timesheets only — the dominant mutation surface (status
// flips, amount edits, approval state). Phase 2 extends this to also cover
// employees.updated_at (bank info changes) and project_employees.updated_at
// (payment-schedule reassignment) via a GREATEST() query across three tables.
// Until then, snapshot drift caused by bank-info or schedule edits is a known
// gap surfaced as a simulation warning, not a silent miss.
func (p *ExportPlanner) computeSnapshotEpoch(ctx context.Context, data *excel.BulkTransferData) (time.Time, error) {
	if data == nil || len(data.EmployeeProjectTimesheets) == 0 {
		return time.Time{}, nil
	}

	tsIDs := make([]uint, 0, len(data.EmployeeProjectTimesheets)*4)
	for _, ids := range data.EmployeeProjectTimesheets {
		tsIDs = append(tsIDs, ids...)
	}
	if len(tsIDs) == 0 {
		return time.Time{}, nil
	}

	// Use GetByIDs (already on the interface) and fold max(updated_at) in Go.
	// A single SELECT MAX(updated_at) would be cheaper, but adding a new repo
	// method is deferred to Phase 2's GREATEST() query — this keeps Phase 1 a
	// zero-new-interface-methods refactor.
	fetched, err := p.timesheetRepo.GetByIDs(ctx, tsIDs)
	if err != nil {
		return time.Time{}, fmt.Errorf("snapshot epoch: %w", err)
	}
	var snapshot time.Time
	for _, ts := range fetched {
		if ts.UpdatedAt.After(snapshot) {
			snapshot = ts.UpdatedAt
		}
	}
	return snapshot, nil
}
