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
	"api-server/internal/pkg/clock"
	pkgConstants "api-server/internal/pkg/constants"
	"api-server/internal/pkg/timeutil"

	"github.com/google/uuid"
)

// ExportService handles bulk transfer export operations.
//
// Post-Phase-1 refactor: Export() is now a thin orchestrator. The read +
// aggregate + validate phase lives in ExportPlanner.Plan() (see planner.go);
// the write phase (transaction_codes batch + audit event) lives in
// ExportService.Persist(). Production calls Plan() → Excel gen → Persist().
// The settlement simulation (Phase 2) calls Plan() only, so it cannot drift
// from production selection logic.
type ExportService struct {
	planner             *ExportPlanner
	fileRepo            BulkTransferFileRepository
	transactionCodeRepo TransactionCodeRepository
	excelService        *excel.Service
	checksumCalculator  *ChecksumCalculator
	eventBus            domain.EventBus
}

// NewExportService creates a new export service instance. The planner is
// constructed internally from the same read-side deps; callers (DI container)
// see the same constructor signature as before the refactor.
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
	planner := NewExportPlanner(
		timesheetRepo,
		employeeRepo,
		projectRepo,
		projectEmployeeRepo,
		periodCalculator,
		excelService,
	)
	return &ExportService{
		planner:             planner,
		fileRepo:            fileRepo,
		transactionCodeRepo: transactionCodeRepo,
		excelService:        excelService,
		checksumCalculator:  NewChecksumCalculator(),
		eventBus:            eventBus,
	}
}

// Planner exposes the read-only planner for the simulation service (Phase 2)
// to reuse without duplicating selection logic.
func (es *ExportService) Planner() *ExportPlanner { return es.planner }

// Export generates an Excel file with bulk transfer data.
//
// Behavior is byte-for-byte identical to the pre-refactor implementation —
// verified by the existing flow_manual_bulk_transfer integration test. The
// only structural change is that the read phase is delegated to Plan() and
// the write phase to Persist(), so the simulation can call Plan() alone.
func (es *ExportService) Export(ctx context.Context, req *dto.ExportBulkTransferRequest) (*dto.ExportBulkTransferResponse, error) {
	plan, err := es.planner.Plan(ctx, req)
	if err != nil {
		return nil, err
	}

	// Stale-snapshot guard (opt-in via IfMatchSnapshot — typically from a prior
	// /payrolls/simulate-settlement response). Rejects with HTTP 409 when any
	// relevant row has moved since the simulation, so an export cannot silently
	// proceed using stale results (acceptance criterion #6). Absent = backward
	// compatible.
	if req.IfMatchSnapshot != nil && !req.IfMatchSnapshot.IsZero() && plan.SnapshotEpoch.After(*req.IfMatchSnapshot) {
		return nil, domain.NewConflictErrorWithCode(
			"STALE_SIMULATION",
			"Dữ liệu đã thay đổi kể từ lần mô phỏng gần nhất — vui lòng chạy lại mô phỏng đối soát trước khi xuất.",
		)
	}

	// Re-read bank details at the file-generation boundary. Planning can take
	// long enough for a bank account to be confirmed invalid after the initial
	// snapshot; the manual Chuyển lô file must use the latest eligibility state.
	if err := es.planner.RefreshEmployeeBankDetails(ctx, plan.RawAggregated); err != nil {
		return nil, err
	}
	plan.ValidatedData = es.excelService.ValidateAndFilterBulkTransferData(plan.RawAggregated)
	plan.ForecastOutcomeItems = buildForecastOutcomeItems(req, plan.IsMonthly, plan.SelectedTimesheets, plan.ValidatedData, plan.PaymentPercentage)

	// Generate and validate every workbook before the write phase. A configured
	// threshold can reject an individual row, and that failure must not leave
	// transaction codes behind for a file the Admin never received.
	fromDateStr, toDateStr := formatDateRange(plan)
	workbookLimit := es.excelService.GetBulkTransferWorkbookLimit(ctx)
	response, err := es.excelService.GenerateBulkTransferExcelWithPaymentPercentage(
		plan.RawAggregated,
		req,
		fromDateStr,
		toDateStr,
		plan.Cycle,
		plan.PaymentPercentage,
		workbookLimit,
	)
	if err != nil {
		return nil, err
	}

	filename, err := es.Persist(ctx, req, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to save bulk transfer file: %w", err)
	}

	response.FromDate = fromDateStr
	response.ToDate = toDateStr
	response.Cycle = plan.Cycle
	response.Filename = filename

	// Only a successfully generated bank file is an export outcome. Publishing
	// before generation would let a failed download calibrate cash forecasts.
	txnCount, totalAmount := exportAuditTotals(plan)
	es.publishExportAudit(ctx, req, filename, plan.Cycle, plan, txnCount, totalAmount)
	return response, nil
}

// Persist performs the write phase of an export: generates a filename and
// creates the transaction_codes batch. The successful-export event is emitted
// by Export only after file generation succeeds.
//
// Note (red-team Finding 1): this does NOT create a bulk_transfer_files row.
// That row is written elsewhere (audit_service.go, ninepay_service.go,
// result_processor.go) when a file asset is actually materialized. Persist's
// write surface is exactly: transactionCodeRepo.CreateBatch + eventBus.Publish.
func (es *ExportService) Persist(ctx context.Context, req *dto.ExportBulkTransferRequest, plan *ExportPlan) (string, error) {
	filename := generateFilename(plan.Cycle)

	if plan.ValidatedData == nil || plan.ValidatedData.ValidData == nil {
		return filename, nil
	}

	data := plan.ValidatedData.ValidData

	// Convert data to BulkTransferFileData array + prepare transaction codes
	var fileDataArray []dto.BulkTransferFileData
	var totalAmount int64
	var transactionCodesToCreate []*domain.TransactionCode
	stt := 1

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

		// Prepare transaction code data.
		// Populate FromDate/ToDate/CycleNum so the bank-transfer-history view
		// can resolve the cycle WITHOUT fetching timesheet dates (Phase A).
		fromDate := plan.FromDate
		toDate := plan.ToDate
		cycleNum := 1
		if plan.Cycle != string(domain.PaymentScheduleMonthly) {
			cycleNum = clock.KyFromWorkDay(plan.FromDate.Day())
		}
		var tcData domain.TransactionCodeData
		if plan.Cycle == string(domain.PaymentScheduleMonthly) {
			tcData = domain.TransactionCodeData{
				MonthlyPay: &domain.CyclePayData{
					TimesheetIDs: timesheetIDs,
					EmployeeID:   employee.ID,
					ProjectID:    project.ID,
					Amount:       amount,
					FromDate:     &fromDate,
					ToDate:       &toDate,
					CycleNum:     cycleNum,
				},
			}
		} else {
			tcData = domain.TransactionCodeData{
				WeeklyPay: &domain.CyclePayData{
					TimesheetIDs: timesheetIDs,
					EmployeeID:   employee.ID,
					ProjectID:    project.ID,
					Amount:       amount,
					FromDate:     &fromDate,
					ToDate:       &toDate,
					CycleNum:     cycleNum,
				},
			}
		}
		tcDataBytes, _ := json.Marshal(tcData)
		transactionCodesToCreate = append(transactionCodesToCreate, &domain.TransactionCode{
			Code: transactionCode,
			Data: tcDataBytes,
		})
	}

	// Sort by STT to ensure consistent ordering (pre-refactor behavior).
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

func exportAuditTotals(plan *ExportPlan) (int, int64) {
	if plan == nil || plan.ValidatedData == nil || plan.ValidatedData.ValidData == nil {
		return 0, 0
	}
	data := plan.ValidatedData.ValidData
	var total int64
	for _, amount := range data.EmployeeProjectAmounts {
		total += amount
	}
	return len(data.EmployeeProjectAmounts), total
}

// publishExportAudit emits the audit event for a successful bulk-transfer file
// export. Failures are logged but do not propagate. Body-identical to the
// pre-refactor helper; signature changed to take the ExportPlan so it can
// derive date strings without re-parsing the request.
func (es *ExportService) publishExportAudit(
	ctx context.Context,
	req *dto.ExportBulkTransferRequest,
	filename, cycle string,
	plan *ExportPlan,
	txnCount int,
	totalAmount int64,
) {
	if es.eventBus == nil {
		return
	}

	forMonth := strings.TrimSpace(req.ForMonth)
	fromForEvent, toForEvent := formatDateRangeStrings(plan)
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
		len(req.ProjectIDs) == 0 && len(req.EmployeeIDs) == 0,
		plan.ForecastOutcomeItems,
	)
	if err := es.eventBus.Publish(ctx, event); err != nil {
		observability.GetLogger().Warn("Failed to publish BulkTransferFileExported audit event",
			"filename", filename,
			"error", err)
	}
}

// generateFilename produces the MBank-style filename {prefix}_{cycle}_{uuid}.
// Extracted from the pre-refactor saveBulkTransferFile.
func generateFilename(cycle string) string {
	fileUUID := uuid.New()
	uuidStr := strings.ReplaceAll(fileUUID.String(), "-", "")
	return fmt.Sprintf("%s_%s_%s", pkgConstants.MBankPrefix, cycle, uuidStr)
}

// formatDateRange returns the (from, to) display strings for a plan, or zero
// strings for full-pool mode (Phase 2) where there is no date window.
func formatDateRange(plan *ExportPlan) (string, string) {
	if plan == nil || plan.FromDate.IsZero() {
		return "", ""
	}
	return plan.FromDate.Format(timeutil.DateFormat), plan.ToDate.Format(timeutil.DateFormat)
}

// formatDateRangeStrings is a thin alias used by publishExportAudit for
// readability alongside the forMonth branch.
func formatDateRangeStrings(plan *ExportPlan) (string, string) {
	return formatDateRange(plan)
}

// `time` is referenced indirectly via formatDateRange's *ExportPlan (which
// carries time.Time fields) and the IsZero check on plan.FromDate, not via a
// direct type reference in this file. This anchor keeps the import valid.
var _ = time.Time{}
