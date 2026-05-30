package bulktransfer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"

	"github.com/google/uuid"

	asynqlib "github.com/hibiken/asynq"
)

// AutoBulkTransferService is the provider-agnostic interface for initiating
// bulk payroll transfers. The concrete implementation resolves the active
// disbursement provider via the registry at runtime.
type AutoBulkTransferService interface {
	InitiateBulkTransfer(ctx context.Context, req *dto.ExportBulkTransferRequest) (*AutoBulkTransferResponse, error)
	GetBatchStatus(ctx context.Context, batchID string) (*AutoBulkTransferStatusResponse, error)
	EstimateFee(ctx context.Context, req *dto.ExportBulkTransferRequest) (*dto.EstimateFeeResponse, error)
}

// AsynqClient enqueues background tasks.
type AsynqClient = *asynqlib.Client

const (
	// TaskNinePayBulkExecute is the Asynq task type for executing a 9Pay bulk transfer batch.
	TaskNinePayBulkExecute = "bulk_transfer:ninepay_execute"
)

// NinePayExecutePayload is the wire format for bulk_transfer:ninepay_execute tasks.
type NinePayExecutePayload struct {
	BatchFileID uint   `json:"batch_file_id"`
	BatchID     string `json:"batch_id"`
}

// DisbursementFeeProvider resolves the per-transfer disbursement fee.
type DisbursementFeeProvider interface {
	GetDisbursementFeeVND(ctx context.Context, provider string) int64
}

// NinePayBulkTransferService orchestrates bulk payroll transfers via 9Pay API.
type NinePayBulkTransferService struct {
	exportService       *ExportService
	fileRepo            BulkTransferFileRepository
	transactionCodeRepo TransactionCodeRepository
	walletPaymentRepo   domaintx.WalletPaymentRepository
	asynqClient         AsynqClient
	feeProvider         DisbursementFeeProvider
	providerResolver    bulkProviderResolver
	eventBus            domain.EventBus
	logger              *slog.Logger
}

// bulkProviderResolver resolves the active disbursement provider name.
type bulkProviderResolver interface {
	ActiveProviderName(ctx context.Context) (string, error)
}

// SetAsynqClient sets the async client after construction.
func (s *NinePayBulkTransferService) SetAsynqClient(client AsynqClient) {
	s.asynqClient = client
}

// SetFeeProvider sets the disbursement fee provider after construction.
func (s *NinePayBulkTransferService) SetFeeProvider(fp DisbursementFeeProvider) {
	s.feeProvider = fp
}

// SetProviderResolver injects the registry for dynamic provider resolution.
func (s *NinePayBulkTransferService) SetProviderResolver(r bulkProviderResolver) {
	s.providerResolver = r
}

func (s *NinePayBulkTransferService) SetWalletPaymentRepo(r domaintx.WalletPaymentRepository) {
	s.walletPaymentRepo = r
}
func NewNinePayBulkTransferService(
	exportService *ExportService,
	fileRepo BulkTransferFileRepository,
	transactionCodeRepo TransactionCodeRepository,
	asynqClient AsynqClient,
	eventBus domain.EventBus,
	logger *slog.Logger,
) *NinePayBulkTransferService {
	if logger == nil {
		logger = slog.Default()
	}
	return &NinePayBulkTransferService{
		exportService:       exportService,
		fileRepo:            fileRepo,
		transactionCodeRepo: transactionCodeRepo,
		asynqClient:         asynqClient,
		eventBus:            eventBus,
		logger:              logger.With("component", "NinePayBulkTransferService"),
	}
}

// AutoBulkTransferResponse is the API response for initiating a 9Pay bulk transfer.
type AutoBulkTransferResponse struct {
	BatchID    string `json:"batch_id"`
	FileID     uint   `json:"file_id"`
	TotalCount int    `json:"total_count"`
	Status     string `json:"status"`
}

// AutoBulkTransferStatusResponse is the API response for batch status polling.
type AutoBulkTransferStatusResponse struct {
	BatchID     string                       `json:"batch_id"`
	TotalCount  int                          `json:"total_count"`
	Completed   int                          `json:"completed"`
	Failed      int                          `json:"failed"`
	Processing  int                          `json:"processing"`
	Status      string                       `json:"status"`
	FailedItems []dto.BulkTransferFailedItem `json:"failed_items,omitempty"`
}

// InitiateBulkTransfer gathers approved timesheets, creates transaction codes,
// saves a BulkTransferFile record (source='ninepay'), and enqueues background execution.
func (s *NinePayBulkTransferService) InitiateBulkTransfer(ctx context.Context, req *dto.ExportBulkTransferRequest) (*AutoBulkTransferResponse, error) {
	isMonthly := strings.TrimSpace(req.ForMonth) != ""
	cycle := string(domain.PaymentScheduleWeekly)
	if isMonthly {
		cycle = string(domain.PaymentScheduleMonthly)
	}

	// Resolve date ranges (same logic as ExportService.Export)
	fromDate, toDate, monthStart, err := s.resolveDateRange(req, isMonthly)
	if err != nil {
		return nil, err
	}

	// Gather timesheets using ExportService internals (same package access)
	periodCache := make(map[uint]projectPeriod)

	filters := domain.TimesheetFilters{
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
		PaymentStatus:   []domain.PaymentStatus{domain.PaymentStatusPending, domain.PaymentStatusFailed},
	}
	if !isMonthly {
		filters.FromDate = &fromDate
		filters.ToDate = &toDate
	}
	if len(req.ProjectIDs) > 0 {
		filters.ProjectIDs = req.ProjectIDs
	}

	allTimesheets, err := s.exportService.listTimesheetsForCycle(ctx, filters, req, isMonthly, monthStart, periodCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get timesheets: %w", err)
	}

	// Include force-payroll timesheets
	trueVal := true
	forcedFilters := domain.TimesheetFilters{
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
		PaymentStatus:   []domain.PaymentStatus{domain.PaymentStatusPending, domain.PaymentStatusFailed},
		ForcePayroll:    &trueVal,
	}
	if len(req.ProjectIDs) > 0 {
		forcedFilters.ProjectIDs = req.ProjectIDs
	}
	forcedAll, err := s.exportService.listTimesheetsForCycle(ctx, forcedFilters, req, isMonthly, monthStart, periodCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get forced timesheets: %w", err)
	}

	// Merge and deduplicate
	filteredTimesheets := make([]*domain.Timesheet, 0, len(allTimesheets)+len(forcedAll))
	seen := make(map[uint]bool)
	for _, t := range allTimesheets {
		if !seen[t.ID] {
			filteredTimesheets = append(filteredTimesheets, t)
			seen[t.ID] = true
		}
	}
	for _, t := range forcedAll {
		if !seen[t.ID] {
			filteredTimesheets = append(filteredTimesheets, t)
			seen[t.ID] = true
		}
	}

	if len(filteredTimesheets) == 0 {
		return nil, domain.NewValidationError("Không tìm thấy bảng chấm công đủ điều kiện để chuyển tiền")
	}
	// Aggregate by employee-project
	aggregatedData, err := s.exportService.aggregateTimesheetData(ctx, filteredTimesheets, cycle)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate timesheet data: %w", err)
	}
	if aggregatedData == nil || len(aggregatedData.EmployeeProjectAmounts) == 0 {
		return nil, domain.NewValidationError("Không tìm thấy bảng chấm công đủ điều kiện để chuyển tiền")
	}

	// Validate and filter (skip rows missing bank info)
	validationResult := s.exportService.excelService.ValidateAndFilterBulkTransferData(aggregatedData)
	if validationResult == nil || validationResult.ValidData == nil || len(validationResult.ValidData.EmployeeProjectAmounts) == 0 {
		return nil, domain.NewValidationError("Không có hàng hợp lệ sau khi lọc (thiếu thông tin ngân hàng)")
	}
	validData := validationResult.ValidData

	// Resolve active provider name for batch ID prefix and source tag
	providerName := "auto"
	if s.providerResolver != nil {
		if pName, err := s.providerResolver.ActiveProviderName(ctx); err == nil && pName != "" {
			providerName = pName
		}
	}

	// Generate batch ID
	batchUUID := uuid.New()
	uuidStr := strings.ReplaceAll(batchUUID.String(), "-", "")
	batchID := fmt.Sprintf("%s_%s_%s", providerName, cycle, uuidStr)

	// Build file data rows
	fileDataArray, totalAmount := s.buildFileData(validData)

	if len(fileDataArray) == 0 {
		return nil, domain.NewValidationError("Không có hàng chuyển tiền để xử lý")
	}

	// Create BulkTransferFile (source='ninepay')
	fileDataJSON, _ := json.Marshal(fileDataArray)
	btf := &domain.BulkTransferFile{
		Filename:          batchID,
		Cycle:             &cycle,
		CreatedBy:         req.CreatedBy,
		TransactionsCount: len(fileDataArray),
		CompletedCount:    0,
		FailedCount:       0,
		TransferAmount:    totalAmount,
		Data:              string(fileDataJSON),
		Source:            providerName,
	}
	if err := s.fileRepo.Create(ctx, btf); err != nil {
		return nil, fmt.Errorf("failed to create bulk transfer file: %w", err)
	}

	// Create transaction codes with FileID set
	transactionCodes := buildTransactionCodes(validData, cycle, btf.ID)
	if len(transactionCodes) > 0 {
		if err := s.transactionCodeRepo.CreateBatch(ctx, transactionCodes); err != nil {
			return nil, fmt.Errorf("failed to create transaction codes: %w", err)
		}
	}

	// Enqueue background execution task
	payload, _ := json.Marshal(NinePayExecutePayload{
		BatchFileID: btf.ID,
		BatchID:     batchID,
	})
	task := asynqlib.NewTask(TaskNinePayBulkExecute, payload)
	if _, err := s.asynqClient.Enqueue(task); err != nil {
		return nil, fmt.Errorf("failed to enqueue bulk transfer task: %w", err)
	}

	s.logger.Info("9Pay bulk transfer initiated",
		"batch_id", batchID,
		"total_count", len(fileDataArray),
		"total_amount", totalAmount)

	return &AutoBulkTransferResponse{
		BatchID:    batchID,
		FileID:     btf.ID,
		TotalCount: len(fileDataArray),
		Status:     "processing",
	}, nil
}

// GetBatchStatus returns the current progress of a 9Pay bulk transfer batch.
// When the batch is still "processing", queries wallet_payments for real-time
// counts instead of relying on cached file-level counts (which can be stale
// due to async IPN delivery).
func (s *NinePayBulkTransferService) GetBatchStatus(ctx context.Context, batchID string) (*AutoBulkTransferStatusResponse, error) {
	btf, err := s.fileRepo.GetByFilename(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("batch not found: %w", err)
	}
	if btf.Source != "ninepay" && btf.Source != "1pay" && btf.Source != "auto" {
		return nil, fmt.Errorf("not an auto bulk transfer batch (source=%s)", btf.Source)
	}

	completed := btf.CompletedCount
	failed := btf.FailedCount
	status := "completed"
	if completed+failed < btf.TransactionsCount {
		status = "processing"
		// Live query wallet_payments for accurate counts (IPNs may have
		// completed since the worker wrote the cached counts).
		if s.walletPaymentRepo != nil {
			payments, queryErr := s.walletPaymentRepo.ListByBatchID(ctx, batchID)
			if queryErr != nil {
				s.logger.Warn("GetBatchStatus: failed to list wallet payments", "error", queryErr)
			} else {
				completed, failed = 0, 0
				for _, p := range payments {
					if !p.IsTerminal() {
						continue
					}
					if p.Status == domaintx.StateCompleted {
						completed++
					} else {
						failed++
					}
				}
				if completed+failed >= btf.TransactionsCount {
					status = "completed"
				}
			}
		}
	}

	// Extract failed rows from file data for detailed failure reporting.
	var failedItems []dto.BulkTransferFailedItem
	fileData, parseErr := dto.ParseBulkTransferFileData(btf.Data)
	if parseErr != nil {
		s.logger.Warn("GetBatchStatus: failed to parse file data", "error", parseErr)
	}
	for _, row := range fileData {
		if row.TransferStatus == ResultStatusFailed {
			failedItems = append(failedItems, dto.BulkTransferFailedItem{
				TimesheetIDs: row.TimesheetIDs,
				EmployeeID:   row.EmployeeID,
				Amount:       row.Amount,
				Reason:       row.ErrorMessage,
				ErrorCode:    row.ErrorMessage,
			})
		}
	}

	return &AutoBulkTransferStatusResponse{
		BatchID:     btf.Filename,
		TotalCount:  btf.TransactionsCount,
		Completed:   completed,
		Failed:      failed,
		Processing:  btf.TransactionsCount - completed - failed,
		Status:      status,
		FailedItems: failedItems,
	}, nil
}

func (s *NinePayBulkTransferService) resolveDateRange(req *dto.ExportBulkTransferRequest, isMonthly bool) (fromDate, toDate time.Time, monthStart time.Time, err error) {

	if isMonthly {
		monthStart, err = s.exportService.periodCalculator.ResolveMonthlyRange(req)
		if err != nil {
			return
		}
		fromDate = monthStart
		toDate = endOfMonth(monthStart)
		return
	}
	fromDate, toDate, err = s.exportService.periodCalculator.ResolveWeeklyRange(req)
	return
}

// EstimateFee calculates the total 9Pay disbursement fee for a potential bulk transfer
// without actually initiating it. Reuses the same timesheet-gathering logic as InitiateBulkTransfer.
func (s *NinePayBulkTransferService) EstimateFee(ctx context.Context, req *dto.ExportBulkTransferRequest) (*dto.EstimateFeeResponse, error) {
	isMonthly := strings.TrimSpace(req.ForMonth) != ""
	cycle := string(domain.PaymentScheduleWeekly)
	if isMonthly {
		cycle = string(domain.PaymentScheduleMonthly)
	}

	fromDate, toDate, monthStart, err := s.resolveDateRange(req, isMonthly)
	if err != nil {
		return nil, err
	}

	periodCache := make(map[uint]projectPeriod)
	filters := domain.TimesheetFilters{
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
		PaymentStatus:   []domain.PaymentStatus{domain.PaymentStatusPending, domain.PaymentStatusFailed},
	}
	if !isMonthly {
		filters.FromDate = &fromDate
		filters.ToDate = &toDate
	}
	if len(req.ProjectIDs) > 0 {
		filters.ProjectIDs = req.ProjectIDs
	}

	allTimesheets, err := s.exportService.listTimesheetsForCycle(ctx, filters, req, isMonthly, monthStart, periodCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get timesheets: %w", err)
	}

	trueVal := true
	forcedFilters := domain.TimesheetFilters{
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
		PaymentStatus:   []domain.PaymentStatus{domain.PaymentStatusPending, domain.PaymentStatusFailed},
		ForcePayroll:    &trueVal,
	}
	if len(req.ProjectIDs) > 0 {
		forcedFilters.ProjectIDs = req.ProjectIDs
	}
	forcedAll, err := s.exportService.listTimesheetsForCycle(ctx, forcedFilters, req, isMonthly, monthStart, periodCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get forced timesheets: %w", err)
	}

	filteredTimesheets := make([]*domain.Timesheet, 0, len(allTimesheets)+len(forcedAll))
	seen := make(map[uint]bool)
	for _, t := range allTimesheets {
		if !seen[t.ID] {
			filteredTimesheets = append(filteredTimesheets, t)
			seen[t.ID] = true
		}
	}
	for _, t := range forcedAll {
		if !seen[t.ID] {
			filteredTimesheets = append(filteredTimesheets, t)
			seen[t.ID] = true
		}
	}

	aggregatedData, err := s.exportService.aggregateTimesheetData(ctx, filteredTimesheets, cycle)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate timesheet data: %w", err)
	}
	if aggregatedData == nil || len(aggregatedData.EmployeeProjectAmounts) == 0 {
		return &dto.EstimateFeeResponse{TotalFee: 0, TransferCount: 0, FeePerTransfer: 0}, nil
	}

	validationResult := s.exportService.excelService.ValidateAndFilterBulkTransferData(aggregatedData)
	if validationResult == nil || validationResult.ValidData == nil || len(validationResult.ValidData.EmployeeProjectAmounts) == 0 {
		return &dto.EstimateFeeResponse{TotalFee: 0, TransferCount: 0, FeePerTransfer: 0}, nil
	}
	transferCount := len(validationResult.ValidData.EmployeeProjectAmounts)

	var feePerTransfer int64
	if s.feeProvider != nil {
		if s.providerResolver != nil {
			if pName, err := s.providerResolver.ActiveProviderName(ctx); err == nil {
				feePerTransfer = s.feeProvider.GetDisbursementFeeVND(ctx, pName)
			}
		}
	}

	return &dto.EstimateFeeResponse{
		TotalFee:       int64(transferCount) * feePerTransfer,
		TransferCount:  transferCount,
		FeePerTransfer: feePerTransfer,
	}, nil
}

func (s *NinePayBulkTransferService) buildFileData(data *excel.BulkTransferData) ([]dto.BulkTransferFileData, int64) {
	fileDataArray := make([]dto.BulkTransferFileData, 0, len(data.EmployeeProjectAmounts))
	var totalAmount int64
	stt := 1

	for key, amount := range data.EmployeeProjectAmounts {
		employee := data.EmployeeData[key.EmployeeID]
		bankName := ""
		bankCode := ""
		if employee.Bank != nil {
			bankName = employee.Bank.BranchName
			bankCode = employee.Bank.BankCode
		}

		fileDataArray = append(fileDataArray, dto.BulkTransferFileData{
			STT:             stt,
			EmployeeID:      employee.ID,
			ProjectID:       data.ProjectData[key.ProjectID].ID,
			TimesheetIDs:    data.EmployeeProjectTimesheets[key],
			AccountNumber:   employee.BankAccountNumber,
			AccountName:     employee.BankAccountName,
			BankName:        bankName,
			BankCode:        bankCode,
			Amount:          amount,
			TransactionCode: data.TransactionCodes[key],
		})
		totalAmount += amount
		stt++
	}

	sort.Slice(fileDataArray, func(i, j int) bool {
		return fileDataArray[i].STT < fileDataArray[j].STT
	})

	return fileDataArray, totalAmount
}

// buildTransactionCodes creates TransactionCode records with FileID pre-set.
func buildTransactionCodes(data *excel.BulkTransferData, cycle string, fileID uint) []*domain.TransactionCode {
	var codes []*domain.TransactionCode

	for key := range data.EmployeeProjectAmounts {
		transactionCode := data.TransactionCodes[key]
		if transactionCode == "" {
			continue
		}

		timesheetIDs := data.EmployeeProjectTimesheets[key]
		employee := data.EmployeeData[key.EmployeeID]
		project := data.ProjectData[key.ProjectID]
		amount := data.EmployeeProjectAmounts[key]

		var tcData domain.TransactionCodeData
		payData := &domain.CyclePayData{
			TimesheetIDs: timesheetIDs,
			EmployeeID:   employee.ID,
			ProjectID:    project.ID,
			Amount:       amount,
			FileID:       &fileID,
		}

		if cycle == string(domain.PaymentScheduleMonthly) {
			tcData = domain.TransactionCodeData{MonthlyPay: payData}
		} else {
			tcData = domain.TransactionCodeData{WeeklyPay: payData}
		}

		tcDataBytes, _ := json.Marshal(tcData)
		codes = append(codes, &domain.TransactionCode{
			Code: transactionCode,
			Data: tcDataBytes,
		})
	}

	return codes
}
