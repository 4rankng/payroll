package bulktransfer

import (
	"api-server/internal/constants"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/app/services/payroll/pdf"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"gorm.io/gorm"
)

// Service handles bulk transfer operations for payroll
type Service struct {
	// Core repositories
	timesheetRepo       TimesheetRepository
	employeeRepo        EmployeeRepository
	employeeUserRepo    EmployeeUserRepository
	projectRepo         ProjectRepository
	projectEmployeeRepo ProjectEmployeeRepository
	userRepo            UserRepository
	fileRepo            BulkTransferFileRepository

	// External services
	ledgerService      LedgerService
	transactionService TransactionService
	assetService       AssetService
	excelConverter     ExcelConverter
	excelService       *excel.Service
	settingsConfig     SettingsConfigService
	pdfService         *pdf.Service
	notifier           Notifier

	// Module services
	periodCalculator    *PeriodCalculator
	rowParser           *RowParser
	exportService       *ExportService
	notificationService *NotificationService
	fileHistoryService  *FileHistoryService
	resultProcessor     *ResultProcessor
	simulationService   *SimulationService
	assetRepo           domain.AssetRepository

	// Bank result parsers (Strategy Pattern)
	resultParsers []BankResultParser
}

// TimesheetRepository interface for timesheet operations
type TimesheetRepository interface {
	List(ctx context.Context, filters domain.TimesheetFilters) ([]*domain.Timesheet, error)
	GetByIDs(ctx context.Context, ids []uint) ([]*domain.Timesheet, error)
	GetByProjectAndEmployee(ctx context.Context, projectID, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error)
	GetByEmployeeAndPeriod(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error)
	CountDistinctWorkingDays(ctx context.Context, employeeID, projectID uint, startDate, endDate time.Time) (int, error)
	BulkUpdatePaymentStatus(ctx context.Context, updates []domain.PaymentStatusUpdate) error
	BulkUpdateRevenueReceivable(ctx context.Context, updates map[uint]int64) error
	BulkUpdateTransactionID(ctx context.Context, transactionID uint, timesheetIDs []uint) error
	Update(ctx context.Context, timesheet *domain.Timesheet) error
}

// EmployeeRepository interface for employee operations
type EmployeeRepository interface {
	GetByID(ctx context.Context, id uint) (*domain.Employee, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*domain.Employee, error)
	GetByBankAccountNumber(ctx context.Context, accountNumber string) (*domain.Employee, error)
}

// ProjectRepository interface for project operations
type ProjectRepository interface {
	GetByID(ctx context.Context, id uint) (*domain.Project, error)
	List(ctx context.Context, filters domain.ProjectFilters) ([]*domain.Project, error)
}

// ProjectEmployeeRepository interface for project-employee operations
type ProjectEmployeeRepository interface {
	GetActiveAssignmentByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error)
	GetActiveAssignmentsByProjectsAndEmployees(ctx context.Context, projectIDs []uint, employeeIDs []uint) ([]*domain.ProjectEmployee, error)
}

// LedgerService interface for ledger operations
type LedgerService interface {
	CreateEntries(ctx context.Context, entries []*domain.LedgerEntry, createdBy uint) ([]*domain.LedgerEntry, error)
	ListEntries(ctx context.Context, filters domain.LedgerFilters) ([]*domain.LedgerEntry, error)
	GetEntriesByAssetID(ctx context.Context, assetID uint) ([]*domain.LedgerEntry, error)
	DeleteEntriesByTransactionID(ctx context.Context, transactionID uint) error
	// GetAccountTotalInRange returns SUM(debit − credit) for an account over
	// [from, to]. Used by the settlement simulation's reconciliation step.
	GetAccountTotalInRange(ctx context.Context, account domain.LedgerAccount, from, to time.Time) (int64, error)
}

// AssetService interface for asset operations
type AssetService interface {
	UploadAsset(ctx context.Context, file multipart.File, header *multipart.FileHeader, req domain.AssetUploadRequest, uploadedBy uint) (*domain.Asset, error)
	GetAsset(ctx context.Context, id uint) (*domain.Asset, error)
	CheckFileExists(filePath string) (bool, string, error)
}

// ExcelConverter interface for Excel operations
type ExcelConverter interface {
	ProcessExcelFile(fileHeader *multipart.FileHeader) (*ExcelFile, error)
	GetSheetNames(excelFile *ExcelFile) []string
	GetRowsFromSheet(excelFile *ExcelFile, sheetName string) ([][]string, error)
}

// ExcelFile represents a processed Excel file
type ExcelFile struct {
	Data     ExcelData
	FileName string
	IsXLS    bool
}

// ExcelData interface for Excel file data
type ExcelData interface {
	Close() error
}

// TransactionService interface for transaction operations
type TransactionService interface {
	CreateTransaction(ctx context.Context, txn *domain.Transaction) (*domain.Transaction, []*domain.LedgerEntry, error)
	DeleteTransaction(ctx context.Context, id uint) error
}

// TransactionRepository interface for transaction operations
type TransactionRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.Transaction, error)
	Delete(ctx context.Context, id int64) error
}

// LedgerRepository interface for ledger operations
type LedgerRepository interface {
	DeleteByTransactionID(ctx context.Context, transactionID int64) error
}

// SettingsConfigService interface for settings configuration operations
type SettingsConfigService interface {
	GetWeeklyPaymentPercentage(ctx context.Context) float64
	GetMonthlyPaymentPercentage(ctx context.Context) float64
	GetAdvanceCashFeePercentage(ctx context.Context) float64
	GetPartnerCompany(ctx context.Context) string
	GetPaymentPercentageForSchedule(ctx context.Context, schedule string) float64
}

// UserRepository interface for user operations (role checking)
type UserRepository interface {
	GetByID(ctx context.Context, id uint) (*domain.User, error)
}

// EmployeeUserRepository interface for employee-user assignments
type EmployeeUserRepository interface {
	GetEmployeeUsers(ctx context.Context, employeeID uint) ([]*domain.EmployeeUser, error)
}

// Notifier defines minimal notification capability to avoid tight coupling
type Notifier interface {
	CreateNotification(ctx context.Context, userID uint, notificationType domain.NotificationType, title, message string) error
}

// BulkTransferFileRepository interface for bulk transfer file operations
type BulkTransferFileRepository interface {
	Create(ctx context.Context, file *domain.BulkTransferFile) error
	GetByID(ctx context.Context, id uint) (*domain.BulkTransferFile, error)
	GetByAssetID(ctx context.Context, assetID uint) (*domain.BulkTransferFile, error)
	GetByFilename(ctx context.Context, filename string) (*domain.BulkTransferFile, error)
	FindDataByTransactionCodes(ctx context.Context, transactionCodes []string) ([]domain.BulkTransferFileDataEntry, string, error)
	UpdateWithLock(ctx context.Context, id uint, updates map[string]interface{}) error
	ListWithFilters(ctx context.Context, cycle string, fromDate, toDate *time.Time, limit, offset int) ([]*domain.BulkTransferFile, int64, error)
	ListResultUploads(ctx context.Context, fromDate, toDate *time.Time, limit, offset int, sortOrder string) ([]*domain.BulkTransferFile, int64, error)
	UpdateCounts(ctx context.Context, id uint, completedCount, failedCount int) error
}

// TransactionCodeRepository interface for transaction code operations
type TransactionCodeRepository interface {
	Create(ctx context.Context, tc *domain.TransactionCode) error
	GetByCode(ctx context.Context, code string) (*domain.TransactionCode, error)
	CreateBatch(ctx context.Context, tcs []*domain.TransactionCode) error
	UpdateFileIDByCodes(ctx context.Context, codes []string, fileID uint) error
}

// NewService creates a new bulk transfer service with all module services
// Uses Config struct to reduce constructor parameters from 18 to 1
func NewService(cfg *Config) *Service {
	// Initialize bank result parsers (Strategy Pattern)
	resultParsers := []BankResultParser{
		NewMBankParser(),
	}

	// Initialize utility modules
	periodCalculator := NewPeriodCalculator(cfg.ProjectRepo)
	rowParser := NewRowParser()

	// Initialize service modules
	notificationService := NewNotificationService(
		cfg.EmployeeRepo,
		cfg.EmployeeUserRepo,
		cfg.UserRepo,
		cfg.Notifier,
		rowParser,
	)

	fileHistoryService := NewFileHistoryService(cfg.FileRepo)

	exportService := NewExportService(
		cfg.TimesheetRepo,
		cfg.EmployeeRepo,
		cfg.ProjectRepo,
		cfg.ProjectEmployeeRepo,
		cfg.FileRepo,
		cfg.TransactionCodeRepo,
		cfg.ExcelService,
		periodCalculator,
		cfg.EventBus,
	)

	resultProcessor := NewResultProcessor(
		cfg.DB,
		cfg.FileRepo,
		cfg.TransactionCodeRepo,
		cfg.EmployeeRepo,
		cfg.AssetService,
		cfg.AssetRepository,
		cfg.ExcelConverter,
		cfg.EventBus,
		cfg.AsynqClient,
	)

	svc := &Service{
		timesheetRepo:       cfg.TimesheetRepo,
		employeeRepo:        cfg.EmployeeRepo,
		employeeUserRepo:    cfg.EmployeeUserRepo,
		projectRepo:         cfg.ProjectRepo,
		projectEmployeeRepo: cfg.ProjectEmployeeRepo,
		userRepo:            cfg.UserRepo,
		fileRepo:            cfg.FileRepo,
		ledgerService:       cfg.LedgerService,
		transactionService:  cfg.TransactionService,
		assetService:        cfg.AssetService,
		excelConverter:      cfg.ExcelConverter,
		excelService:        cfg.ExcelService,
		settingsConfig:      cfg.SettingsConfig,
		pdfService:          cfg.PDFService,
		notifier:            cfg.Notifier,
		periodCalculator:    periodCalculator,
		rowParser:           rowParser,
		exportService:       exportService,
		notificationService: notificationService,
		fileHistoryService:  fileHistoryService,
		resultProcessor:     resultProcessor,
		resultParsers:       resultParsers,
		assetRepo:           cfg.AssetRepository,
		simulationService:   NewSimulationService(exportService.Planner(), cfg.LedgerService, clock.New()),
	}

	return svc
}

// ExportBulkTransfer delegates to the export service
func (s *Service) ExportBulkTransfer(ctx context.Context, req *dto.ExportBulkTransferRequest) (*dto.ExportBulkTransferResponse, error) {
	return s.exportService.Export(ctx, req)
}

// SimulateSettlement projects the current + next N−1 payroll cycles read-only
// and returns a full-pool coverage verdict. Delegates to SimulationService.
func (s *Service) SimulateSettlement(ctx context.Context, req *dto.SimulateSettlementRequest) (*dto.SimulationResult, error) {
	return s.simulationService.Simulate(ctx, req)
}

// ProcessBulkTransferResult delegates to the result processor
// Uses the new checksum-based processing with automatic format detection
func (s *Service) ProcessBulkTransferResult(
	ctx context.Context,
	fileHeader *multipart.FileHeader,
	processedBy uint,
) (*dto.BulkTransferResultResponse, error) {
	return s.resultProcessor.ProcessBulkTransferResultWithChecksum(ctx, fileHeader, processedBy)
}

// GetBulkTransferUploadHistories delegates to the file history service (v2)
func (s *Service) GetBulkTransferUploadHistories(
	ctx context.Context,
	req *dto.ListBulkTransferHistoriesRequest,
) (*dto.ListBulkTransferHistoriesResponse, error) {
	return s.fileHistoryService.GetBulkTransferUploadHistories(ctx, req, s.periodCalculator)
}

// GetBulkTransferUploadHistoryByID retrieves a single bulk transfer history detail from bulk_transfer_files
func (s *Service) GetBulkTransferUploadHistoryByID(
	ctx context.Context,
	id uint,
) (*dto.BulkTransferHistoryDetail, error) {
	file, err := s.fileRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError(constants.MsgBulkTransferHistoryNotFoundVN)
		}
		return nil, err
	}

	if file.AssetID == nil && file.Source != "ninepay" {
		return nil, domain.NewNotFoundError(constants.MsgBulkTransferHistoryNotFoundVN)
	}

	var detailItems []dto.BulkTransferHistoryDetailItem
	if file.Data != "" && file.Data != "null" {
		if isLegacyDataFormat(file.Data) {
			detailItems, err = parseLegacyData(file.Data)
		} else {
			var fileData []dto.BulkTransferFileData
			fileData, detailItems, err = parseCurrentDataDetailed(file.Data, file)
			if err == nil && len(fileData) > 0 {
				s.enrichEmployeeCCCD(ctx, fileData, detailItems)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse transfer data: %w", err)
		}
	}

	return &dto.BulkTransferHistoryDetail{
		Data:         detailItems,
		TotalTxn:     file.TransactionsCount,
		CompletedTxn: file.CompletedCount,
		FailedTxn:    file.FailedCount,
	}, nil
}

// enrichEmployeeCCCD fills CCCD by looking up employee IDs.
func (s *Service) enrichEmployeeCCCD(ctx context.Context, fileData []dto.BulkTransferFileData, items []dto.BulkTransferHistoryDetailItem) {
	idSet := make(map[int64]bool)
	for _, d := range fileData {
		if d.EmployeeID != 0 {
			idSet[int64(d.EmployeeID)] = true
		}
	}
	if len(idSet) == 0 {
		return
	}

	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	employees, err := s.employeeRepo.GetByIDs(ctx, ids)
	if err != nil {
		return
	}

	cccdByID := make(map[uint]string)
	for _, emp := range employees {
		cccdByID[emp.ID] = emp.CCCD
	}

	for i, d := range fileData {
		if i < len(items) && items[i].EmployeeCCCD == "" {
			items[i].EmployeeCCCD = cccdByID[d.EmployeeID]
		}
	}
}

// isLegacyDataFormat detects old BulkTransferResultItem format (has "row" but not "stt" keys).
func isLegacyDataFormat(data string) bool {
	return strings.Contains(data, `"row"`) && !strings.Contains(data, `"stt"`)
}

// parseLegacyData handles the old BulkTransferResultItem JSON format stored by earlier import flows.
func parseLegacyData(data string) ([]dto.BulkTransferHistoryDetailItem, error) {
	var items []dto.BulkTransferResultItem
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		return nil, err
	}
	result := make([]dto.BulkTransferHistoryDetailItem, len(items))
	for i, item := range items {
		paymentStatus := item.PaymentStatus
		if paymentStatus == "completed" {
			paymentStatus = PaymentStatusPaid
		}
		result[i] = dto.BulkTransferHistoryDetailItem{
			Row:                   item.Row,
			EmployeeBank:          item.EmployeeBank,
			EmployeeAccountNumber: item.EmployeeAccountNumber,
			EmployeeName:          item.EmployeeName,
			EmployeeCCCD:          item.EmployeeCCCD,
			Amount:                item.Amount,
			PaymentStatus:         paymentStatus,
			PaidAt:                item.PaidAt,
		}
	}
	return result, nil
}

// parseCurrentDataDetailed parses BulkTransferFileData format and returns both raw data (for enrichment)
// and detail items. Empty TransferStatus is treated as "paid" (successful items from 9Pay).
func parseCurrentDataDetailed(data string, file *domain.BulkTransferFile) ([]dto.BulkTransferFileData, []dto.BulkTransferHistoryDetailItem, error) {
	var fileData []dto.BulkTransferFileData
	if err := json.Unmarshal([]byte(data), &fileData); err != nil {
		return nil, nil, err
	}

	result := make([]dto.BulkTransferHistoryDetailItem, len(fileData))
	var paidAtStr *string
	if file.Source == "ninepay" && !file.CreatedAt.IsZero() {
		s := file.CreatedAt.Format(time.RFC3339)
		paidAtStr = &s
	}

	for i, d := range fileData {
		paymentStatus := d.TransferStatus
		if paymentStatus == "completed" || paymentStatus == "" {
			paymentStatus = PaymentStatusPaid
		}
		paidAt := d.UploadedAt
		if paidAt == nil {
			paidAt = paidAtStr
		}
		result[i] = dto.BulkTransferHistoryDetailItem{
			Row:                   d.STT,
			EmployeeBank:          d.BankName,
			EmployeeBankCode:      d.BankCode,
			EmployeeAccountNumber: d.AccountNumber,
			EmployeeName:          d.AccountName,
			Amount:                strconv.FormatInt(d.Amount, 10),
			PaymentStatus:         paymentStatus,
			PaidAt:                paidAt,
		}
	}
	return fileData, result, nil
}

// ExportService returns the internal export service for reuse by NinePayBulkTransferService.
func (s *Service) ExportService() *ExportService { return s.exportService }

// FileRepo returns the internal file repository for reuse.
func (s *Service) FileRepo() BulkTransferFileRepository { return s.fileRepo }
