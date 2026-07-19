package payroll

import (
	"api-server/internal/constants"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/timeutil"
	"api-server/internal/pkg/utils"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"sort"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/asset"
	"api-server/internal/app/services/config"
	"api-server/internal/app/services/payroll/bulktransfer"
	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/app/services/payroll/pdf"
	"api-server/internal/app/services/reporting"
	"api-server/internal/domain"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// PayrollService provides bulk transfer functionality
type bankTransferHistoryFileRepository interface {
	ListWeeklyForWorkMonth(ctx context.Context, monthStart, monthEnd time.Time) ([]*domain.BulkTransferFile, error)
}

type PayrollService struct {
	bulkTransferService     *bulktransfer.Service
	excelService            *excel.Service
	timesheetRepo           domain.TimesheetRepository
	employeeRepo            domain.EmployeeRepository
	projectRepo             domain.ProjectRepository
	bankTransferHistoryRepo bankTransferHistoryFileRepository
	transactionCodeRepo     domain.TransactionCodeRepository
	simulationService       *SettlementSimulationService
}

// SetSimulationService wires the settlement-simulation service. Called from
// bootstrap after both PayrollService and PayrollReportByProjectService are
// constructed (avoids bloating the NewPayrollService constructor signature).
func (s *PayrollService) SetSimulationService(sim *SettlementSimulationService) {
	s.simulationService = sim
}

// NewPayrollService creates a new PayrollService with only bulk transfer functionality
func NewPayrollService(
	db interface{},
	timesheetRepo domain.TimesheetRepository,
	employeeRepo domain.EmployeeRepository,
	employeeUserRepo domain.EmployeeUserRepository,
	projectRepo domain.ProjectRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	userRepo domain.UserRepository,
	ledgerService bulktransfer.LedgerService,
	transactionService bulktransfer.TransactionService,
	assetService *asset.AssetService,
	excelConverterSvc ExcelConverter,
	settingsConfigService *config.SettingsConfigService,
	bulkTransferFileRepo domain.BulkTransferFileRepository,
	transactionCodeRepo domain.TransactionCodeRepository,
	pdfService *pdf.Service,
	notifier bulktransfer.Notifier,
	eventBus domain.EventBus,
	asynqClient bulktransfer.BulkTransferEnqueuer,
) *PayrollService {
	// Create Excel converter adapter
	excelConverterAdapter := &ExcelConverterAdapter{service: excelConverterSvc}

	// Create Excel service with settings config
	excelService := excel.NewService(settingsConfigService)

	// Convert db interface to *gorm.DB
	gormDB, ok := db.(*gorm.DB)
	if !ok {
		panic("db must be *gorm.DB")
	}

	// Create bulk transfer service config
	bulkTransferConfig := &bulktransfer.Config{
		TimesheetRepo:       timesheetRepo,
		EmployeeRepo:        employeeRepo,
		EmployeeUserRepo:    employeeUserRepo,
		ProjectRepo:         projectRepo,
		ProjectEmployeeRepo: projectEmployeeRepo,
		UserRepo:            userRepo,
		FileRepo:            bulkTransferFileRepo,
		TransactionCodeRepo: transactionCodeRepo,
		LedgerService:       ledgerService,
		TransactionService:  transactionService,
		AssetService:        assetService,
		AssetRepository:     assetService.AssetRepo,
		ExcelConverter:      excelConverterAdapter,
		ExcelService:        excelService,
		SettingsConfig:      settingsConfigService,
		PDFService:          pdfService,
		Notifier:            notifier,
		DB:                  gormDB,
		EventBus:            eventBus,
		AsynqClient:         asynqClient,
	}

	// Create bulk transfer service with config (18 params → 1)
	bulkTransferService := bulktransfer.NewService(bulkTransferConfig)
	bankTransferHistoryRepo, _ := bulkTransferFileRepo.(bankTransferHistoryFileRepository)

	return &PayrollService{
		bulkTransferService:     bulkTransferService,
		excelService:            excelService,
		timesheetRepo:           timesheetRepo,
		employeeRepo:            employeeRepo,
		projectRepo:             projectRepo,
		bankTransferHistoryRepo: bankTransferHistoryRepo,
		transactionCodeRepo:     transactionCodeRepo,
	}
}

// ExcelConverterAdapter adapts the main ExcelConverterService for payroll use
type ExcelConverterAdapter struct {
	service ExcelConverter
}

func (a *ExcelConverterAdapter) ProcessExcelFile(fileHeader *multipart.FileHeader) (*bulktransfer.ExcelFile, error) {
	sourceFile, err := a.service.ProcessExcelFile(fileHeader)
	if err != nil {
		return nil, err
	}

	// Convert from services.ExcelFile to bulktransfer.ExcelFile
	return &bulktransfer.ExcelFile{
		Data:     sourceFile.Data,
		FileName: sourceFile.FileName,
		IsXLS:    sourceFile.IsXLS,
	}, nil
}

func (a *ExcelConverterAdapter) GetSheetNames(excelFile *bulktransfer.ExcelFile) []string {
	// Just delegate to the excel service directly using the underlying excelize.File
	if excelizeFile, ok := excelFile.Data.(*excelize.File); ok {
		return excelizeFile.GetSheetList()
	}
	return []string{}
}

func (a *ExcelConverterAdapter) GetRowsFromSheet(excelFile *bulktransfer.ExcelFile, sheetName string) ([][]string, error) {
	// Just delegate to the excel service directly using the underlying excelize.File
	if excelizeFile, ok := excelFile.Data.(*excelize.File); ok {
		rows, err := excelizeFile.GetRows(sheetName)
		return rows, err
	}
	return nil, fmt.Errorf("invalid excel file data type")
}

// ExcelConverter interface for Excel operations
type ExcelConverter interface {
	ProcessExcelFile(fileHeader *multipart.FileHeader) (*reporting.ExcelFile, error)
}

// GetBulkTransferTemplate returns the bulk transfer template
func (s *PayrollService) GetBulkTransferTemplate(ctx context.Context) ([]byte, error) {
	return s.excelService.GetBulkTransferTemplate()
}

// ExportBulkTransfer exports bulk transfer data
func (s *PayrollService) ExportBulkTransfer(ctx context.Context, req *dto.ExportBulkTransferRequest) (*dto.ExportBulkTransferResponse, error) {
	return s.bulkTransferService.ExportBulkTransfer(ctx, req)
}

// SimulateSettlement projects N future exports starting from a start_date and
// returns a coverage verdict with remainders + reconciliation. Mirrors the
// real "Xuất sao kê" selection via PayrollReportByProjectService.
func (s *PayrollService) SimulateSettlement(ctx context.Context, req *dto.SimulateSettlementRequest) (*dto.SimulationResult, error) {
	if s.simulationService == nil {
		return nil, fmt.Errorf("simulation service not configured")
	}
	return s.simulationService.Simulate(ctx, req)
}

// ProcessBulkTransferResult processes bulk transfer results
func (s *PayrollService) ProcessBulkTransferResult(ctx context.Context, fileHeader *multipart.FileHeader, processedBy uint) (*dto.BulkTransferResultResponse, error) {
	return s.bulkTransferService.ProcessBulkTransferResult(ctx, fileHeader, processedBy)
}

// GetBulkTransferUploadHistories retrieves all bulk transfer upload histories with pagination
func (s *PayrollService) GetBulkTransferUploadHistories(ctx context.Context, req *dto.ListBulkTransferHistoriesRequest) (*dto.ListBulkTransferHistoriesResponse, error) {
	return s.bulkTransferService.GetBulkTransferUploadHistories(ctx, req)
}

// GetBulkTransferUploadHistoryByID retrieves a single bulk transfer upload history by ID
func (s *PayrollService) GetBulkTransferUploadHistoryByID(ctx context.Context, id uint) (*dto.BulkTransferHistoryDetail, error) {
	return s.bulkTransferService.GetBulkTransferUploadHistoryByID(ctx, id)
}

// GetPayrollHistories retrieves payment histories with access control
func (s *PayrollService) GetPayrollHistories(ctx context.Context, req *dto.ListPayrollHistoriesRequest, userID uint, userRole string) (*dto.ListPayrollHistoriesResponse, error) {
	// Build filters
	filters := domain.PaymentHistoryFilters{
		ProjectIDs:  req.ProjectID,
		EmployeeIDs: req.EmployeeID,
		Position:    req.Position,
		Search:      req.Search,
		SortBy:      req.SortBy,
		SortOrder:   req.SortOrder,
		Limit:       req.PageSize,
		Offset:      (req.Page - 1) * req.PageSize,
	}

	// Apply date filters
	loc, _ := time.LoadLocation("Local")
	if req.FromDate != "" {
		fromDate, err := time.ParseInLocation("2006-01-02", req.FromDate, loc)
		if err != nil {
			return nil, domain.NewValidationError(constants.MsgInvalidFromDateFormatVN)
		}
		filters.FromDate = &fromDate
	}

	if req.ToDate != "" {
		toDate, err := time.ParseInLocation("2006-01-02", req.ToDate, loc)
		if err != nil {
			return nil, domain.NewValidationError(constants.MsgInvalidToDateFormatVN)
		}
		filters.ToDate = &toDate
	}

	// Apply access control based on user role
	// Partners can only see employees they created OR assigned to projects
	if userRole != "admin" {
		filters.EmployeeCreatedBy = &userID
		filters.EmployeeAssignedByPartner = &userID
	}

	// Fetch payment histories from repository
	histories, totalCount, err := s.timesheetRepo.GetPaymentHistories(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Transform domain models to DTOs
	items := make([]dto.PayrollHistoryItem, 0, len(histories))
	for _, h := range histories {
		items = append(items, dto.PayrollHistoryItem{
			EmployeeID:      h.EmployeeID,
			EmployeeName:    h.EmployeeName,
			EmployeeCCCD:    h.EmployeeCCCD,
			ProjectID:       h.ProjectID,
			ProjectName:     h.ProjectName,
			Position:        h.Position,
			TotalPaidAmount: h.TotalPaidAmount,
			PaidDate:        h.PaidAt.Format(time.RFC3339),
		})
	}

	// Calculate pagination
	totalPages := int(totalCount) / req.PageSize
	if int(totalCount)%req.PageSize > 0 {
		totalPages++
	}

	return &dto.ListPayrollHistoriesResponse{
		Data: items,
		Pagination: dto.PaginationResponse{
			Page:         req.Page,
			PageSize:     req.PageSize,
			TotalPages:   totalPages,
			TotalRecords: totalCount,
		},
	}, nil
}

// GetBankTransferHistories returns completed bank postings grouped by employee and weekly cycle.
func (s *PayrollService) GetBankTransferHistories(ctx context.Context, req *dto.ListBankTransferHistoriesRequest, userID uint, userRole string) (*dto.ListBankTransferHistoriesResponse, error) {
	workMonth := clock.Now()
	if req.Month != "" {
		parsed, err := time.ParseInLocation("2006-01", req.Month, clock.DefaultLocation)
		if err != nil {
			return nil, domain.NewValidationError("Tháng không hợp lệ, định dạng đúng là YYYY-MM")
		}
		workMonth = parsed
	}
	monthStart := time.Date(workMonth.Year(), workMonth.Month(), 1, 0, 0, 0, 0, clock.DefaultLocation)
	monthEnd := time.Date(workMonth.Year(), workMonth.Month(), 28, 23, 59, 59, 0, clock.DefaultLocation)

	allowedProjects := make(map[uint]struct{})
	if userRole != "admin" {
		projects, err := s.projectRepo.List(ctx, domain.ProjectFilters{AccessibleBy: &userID, Limit: -1})
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			allowedProjects[project.ID] = struct{}{}
		}
	}
	requestedProjects := make(map[uint]struct{}, len(req.ProjectID))
	for _, id := range req.ProjectID {
		requestedProjects[id] = struct{}{}
	}
	requestedEmployees := make(map[uint]struct{}, len(req.EmployeeID))
	for _, id := range req.EmployeeID {
		requestedEmployees[id] = struct{}{}
	}

	if s.bankTransferHistoryRepo == nil {
		return nil, domain.NewInternalError("Không thể tải lịch sử chuyển khoản", fmt.Errorf("bank transfer history repository is not configured"))
	}
	files, err := s.bankTransferHistoryRepo.ListWeeklyForWorkMonth(ctx, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}

	type aggregate struct {
		item       dto.BankTransferHistoryItem
		projectSet map[uint]struct{}
		refSet     map[string]struct{}
	}
	aggregates := make(map[string]*aggregate)
	employeeIDs := make(map[uint]struct{})
	projectIDs := make(map[uint]struct{})
	parsedByFile := make(map[uint][]dto.BulkTransferFileData, len(files))
	transactionCodes := make([]string, 0)
	for _, file := range files {
		entries, parseErr := dto.ParseBulkTransferFileData(file.Data)
		if parseErr != nil {
			continue
		}
		parsedByFile[file.ID] = entries
		for _, entry := range entries {
			if entry.TransactionCode != "" {
				transactionCodes = append(transactionCodes, entry.TransactionCode)
			}
		}
	}
	codeRows, err := s.transactionCodeRepo.FindByCodes(ctx, transactionCodes)
	if err != nil {
		return nil, err
	}
	codeData := make(map[string]domain.TransactionCodeData, len(codeRows))
	timesheetIDSet := make(map[uint]struct{})
	for _, row := range codeRows {
		var data domain.TransactionCodeData
		if json.Unmarshal(row.Data, &data) != nil || data.WeeklyPay == nil {
			continue
		}
		codeData[row.Code] = data
		for _, id := range data.WeeklyPay.TimesheetIDs {
			timesheetIDSet[id] = struct{}{}
		}
	}
	timesheetIDs := make([]uint, 0, len(timesheetIDSet))
	for id := range timesheetIDSet {
		timesheetIDs = append(timesheetIDs, id)
	}
	// Only the timesheet date is needed downstream (in resolveWeeklyHistoryCycle),
	// so use the lightweight date-only projection instead of GetByIDs, which would
	// load full rows + relationship preloads for thousands of timesheets.
	timesheetDates := map[uint]time.Time{}
	if len(timesheetIDs) > 0 {
		timesheetDates, err = s.timesheetRepo.GetTimesheetDatesByIDs(ctx, timesheetIDs)
		if err != nil {
			return nil, err
		}
	}

	for _, file := range files {
		entries := parsedByFile[file.ID]
		for _, entry := range entries {
			ref := strings.TrimSpace(entry.BankTxnRef)
			if entry.TransferStatus != "completed" || ref == "" || entry.Amount <= 0 {
				continue
			}
			entryEmployeeID := entry.EmployeeID
			entryProjectID := entry.ProjectID
			weeklyData := codeData[entry.TransactionCode].WeeklyPay
			if weeklyData != nil {
				if entryEmployeeID == 0 {
					entryEmployeeID = weeklyData.EmployeeID
				}
				if entryProjectID == 0 {
					entryProjectID = weeklyData.ProjectID
				}
			}
			if entryEmployeeID == 0 || entryProjectID == 0 {
				continue
			}
			cycle, fromDate, toDate, ok := resolveWeeklyHistoryCycle(file, weeklyData, timesheetDates, monthStart)
			if !ok || (req.Cycle != 0 && req.Cycle != cycle) {
				continue
			}
			if userRole != "admin" {
				if _, ok := allowedProjects[entryProjectID]; !ok {
					continue
				}
			}
			if len(requestedProjects) > 0 {
				if _, ok := requestedProjects[entryProjectID]; !ok {
					continue
				}
			}
			if len(requestedEmployees) > 0 {
				if _, ok := requestedEmployees[entryEmployeeID]; !ok {
					continue
				}
			}

			key := fmt.Sprintf("%s:%d:%d", monthStart.Format("2006-01"), cycle, entryEmployeeID)
			agg := aggregates[key]
			if agg == nil {
				payDate := clock.PayDate(cycle, monthStart.Year(), monthStart.Month())
				agg = &aggregate{
					item: dto.BankTransferHistoryItem{
						EmployeeID:  entryEmployeeID,
						WorkMonth:   monthStart.Format("2006-01"),
						Cycle:       cycle,
						FromDate:    fromDate.Format("2006-01-02"),
						ToDate:      toDate.Format("2006-01-02"),
						PaymentDate: payDate.Format("2006-01-02"),
						Transfers:   []dto.BankTransferHistoryTransfer{},
					},
					projectSet: make(map[uint]struct{}),
					refSet:     make(map[string]struct{}),
				}
				aggregates[key] = agg
			}
			transferKey := strings.ToUpper(ref)
			if _, duplicate := agg.refSet[transferKey]; duplicate {
				continue
			}
			agg.refSet[transferKey] = struct{}{}
			agg.projectSet[entryProjectID] = struct{}{}
			agg.item.TotalAmount += entry.Amount
			paidAt := ""
			if entry.UploadedAt != nil {
				paidAt = *entry.UploadedAt
			} else if file.UploadedAt != nil {
				paidAt = file.UploadedAt.In(clock.DefaultLocation).Format(time.RFC3339)
			}
			agg.item.Transfers = append(agg.item.Transfers, dto.BankTransferHistoryTransfer{
				TransferCode:  entry.TransactionCode,
				BankReference: ref,
				Amount:        entry.Amount,
				PaidAt:        paidAt,
			})
			employeeIDs[entryEmployeeID] = struct{}{}
			projectIDs[entryProjectID] = struct{}{}
		}
	}
	if len(aggregates) == 0 {
		return emptyBankTransferHistory(req), nil
	}

	employeeIDList := make([]int64, 0, len(employeeIDs))
	for id := range employeeIDs {
		employeeIDList = append(employeeIDList, int64(id))
	}
	employees, err := s.employeeRepo.GetByIDs(ctx, employeeIDList)
	if err != nil {
		return nil, err
	}
	employeeMap := make(map[uint]*domain.Employee, len(employees))
	for _, employee := range employees {
		employeeMap[employee.ID] = employee
	}
	projectIDList := make([]uint, 0, len(projectIDs))
	for id := range projectIDs {
		projectIDList = append(projectIDList, id)
	}
	projects, err := s.projectRepo.GetByIDs(ctx, projectIDList)
	if err != nil {
		return nil, err
	}

	search := strings.TrimSpace(req.Search)
	nameNormalizer := utils.NewVietnameseNormalizer()
	nameSearch := nameNormalizer.Normalize(search)
	referenceSearch := strings.ToLower(search)
	items := make([]dto.BankTransferHistoryItem, 0, len(aggregates))
	for _, agg := range aggregates {
		if employee := employeeMap[agg.item.EmployeeID]; employee != nil {
			agg.item.EmployeeName = employee.Fullname
			agg.item.EmployeeCCCD = employee.CCCD
		}
		if search != "" && !strings.Contains(nameNormalizer.Normalize(agg.item.EmployeeName), nameSearch) && !historyTransfersContain(agg.item.Transfers, referenceSearch) {
			continue
		}
		for id := range agg.projectSet {
			agg.item.ProjectIDs = append(agg.item.ProjectIDs, id)
			if project := projects[id]; project != nil {
				agg.item.ProjectNames = append(agg.item.ProjectNames, project.Name)
			}
		}
		sort.Slice(agg.item.Transfers, func(i, j int) bool { return agg.item.Transfers[i].BankReference < agg.item.Transfers[j].BankReference })
		sort.Slice(agg.item.ProjectIDs, func(i, j int) bool { return agg.item.ProjectIDs[i] < agg.item.ProjectIDs[j] })
		sort.Strings(agg.item.ProjectNames)
		items = append(items, agg.item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Cycle != items[j].Cycle {
			return items[i].Cycle > items[j].Cycle
		}
		return items[i].EmployeeName < items[j].EmployeeName
	})

	total := len(items)
	start := (req.Page - 1) * req.PageSize
	if start > total {
		start = total
	}
	end := start + req.PageSize
	if end > total {
		end = total
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + req.PageSize - 1) / req.PageSize
	}
	return &dto.ListBankTransferHistoriesResponse{
		Data:       items[start:end],
		Pagination: dto.PaginationResponse{Page: req.Page, PageSize: req.PageSize, TotalPages: totalPages, TotalRecords: int64(total)},
	}, nil
}

func fixedWeeklyCycle(fromDate, toDate time.Time) (int, bool) {
	starts := []int{0, 1, 8, 15, 22}
	ends := []int{0, 7, 14, 21, 28}
	if fromDate.Year() != toDate.Year() || fromDate.Month() != toDate.Month() {
		return 0, false
	}
	for cycle := 1; cycle <= 4; cycle++ {
		if fromDate.Day() == starts[cycle] && toDate.Day() == ends[cycle] {
			return cycle, true
		}
	}
	return 0, false
}

func resolveWeeklyHistoryCycle(file *domain.BulkTransferFile, weeklyData *domain.CyclePayData, timesheetDates map[uint]time.Time, workMonth time.Time) (int, time.Time, time.Time, bool) {
	if file.FromDate != nil && file.ToDate != nil {
		if cycle, ok := fixedWeeklyCycle(*file.FromDate, *file.ToDate); ok && file.FromDate.Year() == workMonth.Year() && file.FromDate.Month() == workMonth.Month() {
			return cycle, file.FromDate.In(clock.DefaultLocation), file.ToDate.In(clock.DefaultLocation), true
		}
	}
	if weeklyData == nil {
		return 0, time.Time{}, time.Time{}, false
	}
	for _, id := range weeklyData.TimesheetIDs {
		date, ok := timesheetDates[id]
		if !ok || date.Year() != workMonth.Year() || date.Month() != workMonth.Month() || date.Day() > 28 {
			continue
		}
		cycle := clock.KyFromWorkDay(date.Day())
		fromDate := time.Date(workMonth.Year(), workMonth.Month(), clock.WorkStartDay(cycle), 0, 0, 0, 0, clock.DefaultLocation)
		return cycle, fromDate, fromDate.AddDate(0, 0, 6), true
	}
	return 0, time.Time{}, time.Time{}, false
}

func emptyBankTransferHistory(req *dto.ListBankTransferHistoriesRequest) *dto.ListBankTransferHistoriesResponse {
	return &dto.ListBankTransferHistoriesResponse{
		Data:       []dto.BankTransferHistoryItem{},
		Pagination: dto.PaginationResponse{Page: req.Page, PageSize: req.PageSize, TotalPages: 0, TotalRecords: 0},
	}
}

func historyTransfersContain(transfers []dto.BankTransferHistoryTransfer, search string) bool {
	for _, transfer := range transfers {
		if strings.Contains(strings.ToLower(transfer.TransferCode), search) ||
			strings.Contains(strings.ToLower(transfer.BankReference), search) {
			return true
		}
	}
	return false
}

// ExportPayrollHistories exports payroll histories to Excel with multiple sheets per project
func (s *PayrollService) ExportPayrollHistories(ctx context.Context, req *dto.ExportPayrollHistoriesRequest, userID uint, userRole string) ([]byte, string, error) {
	// Parse dates
	loc, _ := time.LoadLocation("Local")
	fromDate, err := time.ParseInLocation("2006-01-02", req.FromDate, loc)
	if err != nil {
		return nil, "", domain.NewValidationError(constants.MsgInvalidFromDateFormatVN)
	}

	toDate, err := time.ParseInLocation("2006-01-02", req.ToDate, loc)
	if err != nil {
		return nil, "", domain.NewValidationError(constants.MsgInvalidToDateFormatVN)
	}

	// Include entire end day
	toDate = timeutil.EndOfDay(toDate)

	// Build filters
	filters := domain.PaymentHistoryFilters{
		FromDate:  &fromDate,
		ToDate:    &toDate,
		SortBy:    "project_name,employee_name,paid_at",
		SortOrder: "asc",
		Limit:     -1, // Fetch all for export
		Offset:    0,
	}

	// Apply role-based access control
	if userRole != "admin" {
		partnerID := userID
		filters.EmployeeAssignedByPartner = &partnerID
	}

	// Fetch data
	histories, _, err := s.timesheetRepo.GetPaymentHistories(ctx, filters)
	if err != nil {
		return nil, "", err
	}

	// Generate Excel
	excelData, err := s.excelService.ExportPayrollHistoriesToExcel(ctx, histories, req.FromDate, req.ToDate)
	if err != nil {
		return nil, "", err
	}

	// Generate filename
	filename := fmt.Sprintf("lich-su-luong_%s_%s.xlsx", req.FromDate, req.ToDate)

	return excelData, filename, nil
}

// BulkTransferSvc returns the internal bulk transfer service for reuse.
func (s *PayrollService) BulkTransferSvc() *bulktransfer.Service { return s.bulkTransferService }

// MarkExternallyPaid marks the given timesheets as paid externally (outside the app).
// Only timesheets with status=approved and payment_status=pending or payment_status=failed are updated.
func (s *PayrollService) MarkExternallyPaid(ctx context.Context, req *dto.MarkExternallyPaidRequest, userID uint) (*dto.MarkExternallyPaidResponse, error) {
	timesheets, err := s.timesheetRepo.GetByIDs(ctx, req.TimesheetIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timesheets: %w", err)
	}

	now := clock.Now()
	ref := req.Reference
	var updates []domain.PaymentStatusUpdate
	for _, ts := range timesheets {
		if ts.Status != domain.TimesheetStatusApproved {
			continue
		}
		if ts.PaymentStatus != domain.PaymentStatusPending && ts.PaymentStatus != domain.PaymentStatusFailed {
			continue
		}
		updates = append(updates, domain.PaymentStatusUpdate{
			TimesheetID:      ts.ID,
			PaymentStatus:    domain.PaymentStatusPaid,
			PaymentReference: &ref,
			PaidAt:           &now,
		})
	}

	if len(updates) == 0 {
		return &dto.MarkExternallyPaidResponse{MarkedCount: 0}, nil
	}

	if err := s.timesheetRepo.BulkUpdatePaymentStatus(ctx, updates); err != nil {
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

	return &dto.MarkExternallyPaidResponse{MarkedCount: len(updates)}, nil
}
