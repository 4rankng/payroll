package payroll

import (
	"api-server/internal/constants"
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"mime/multipart"
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
type PayrollService struct {
	bulkTransferService *bulktransfer.Service
	excelService        *excel.Service
	timesheetRepo       domain.TimesheetRepository
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

	return &PayrollService{
		bulkTransferService: bulkTransferService,
		excelService:        excelService,
		timesheetRepo:       timesheetRepo,
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
	toDate = toDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

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
