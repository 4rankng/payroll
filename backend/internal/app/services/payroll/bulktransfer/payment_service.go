package bulktransfer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	pkgConstants "api-server/internal/pkg/constants"
)

// PaymentUpdate represents a payment update with metadata
type PaymentUpdate struct {
	Update      domain.PaymentStatusUpdate
	Detail      *dto.BulkTransferResultItemDetail
	PaidAmount  int64
	IsFailed    bool
	TimesheetID uint
}

// BulkPaymentService handles payment status updates for bulk transfers
type BulkPaymentService struct {
	timesheetRepo       TimesheetRepository
	employeeRepo        EmployeeRepository
	projectRepo         ProjectRepository
	projectEmployeeRepo ProjectEmployeeRepository
	settingsConfig      SettingsConfigService
	rowParser           *RowParser

	// Caches to reduce DB calls
	employeeCache map[uint]*domain.Employee
	projectCache  map[uint]*domain.Project
}

// NewBulkPaymentService creates a new BulkPaymentService instance
func NewBulkPaymentService(
	timesheetRepo TimesheetRepository,
	employeeRepo EmployeeRepository,
	projectRepo ProjectRepository,
	projectEmployeeRepo ProjectEmployeeRepository,
	settingsConfig SettingsConfigService,
	rowParser *RowParser,
) *BulkPaymentService {
	return &BulkPaymentService{
		timesheetRepo:       timesheetRepo,
		employeeRepo:        employeeRepo,
		projectRepo:         projectRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		settingsConfig:      settingsConfig,
		rowParser:           rowParser,
		employeeCache:       make(map[uint]*domain.Employee),
		projectCache:        make(map[uint]*domain.Project),
	}
}

// BuildPaymentUpdates creates payment updates for mapped rows
func (s *BulkPaymentService) BuildPaymentUpdates(
	ctx context.Context,
	mappedRows []*MappedRow,
	filename string,
	now time.Time,
) ([]PaymentUpdate, error) {
	var updates []PaymentUpdate

	for _, mapped := range mappedRows {
		if len(mapped.Timesheets) == 0 {
			continue
		}

		// Determine payment status
		isFailed := mapped.Detail.TransferStatus == TransferStatusFailed
		var paymentStatus domain.PaymentStatus
		if isFailed {
			paymentStatus = domain.PaymentStatusFailed
			mapped.Detail.PaymentStatus = PaymentStatusFailed
		} else {
			paymentStatus = domain.PaymentStatusPaid
			mapped.Detail.PaymentStatus = PaymentStatusPaid
		}

		// Get payment percentage
		paymentSchedule := s.detectPaymentScheduleFromFilename(filename)
		percentage := s.settingsConfig.GetPaymentPercentageForSchedule(ctx, paymentSchedule)

		// Build updates for timesheets
		timesheetUpdates := s.buildTimesheetUpdates(
			mapped.Timesheets,
			paymentStatus,
			mapped.Detail.EmployeeAccountNumber,
			percentage,
			now,
			isFailed,
		)

		// Validate total amount
		if !isFailed {
			s.validateTotalAmount(timesheetUpdates, mapped.Detail.Amount)
		}

		// Add to result
		for _, update := range timesheetUpdates {
			updates = append(updates, PaymentUpdate{
				Update:      update,
				Detail:      &mapped.Detail,
				PaidAmount:  s.extractPaidAmount(update),
				IsFailed:    isFailed,
				TimesheetID: update.TimesheetID,
			})
		}

		mapped.Detail.TimesheetsUpdated = len(timesheetUpdates)
	}

	return updates, nil
}

// buildTimesheetUpdates creates payment status updates for timesheets
func (s *BulkPaymentService) buildTimesheetUpdates(
	timesheets []*domain.Timesheet,
	status domain.PaymentStatus,
	accountNumber string,
	percentage float64,
	now time.Time,
	isFailed bool,
) []domain.PaymentStatusUpdate {
	var updates []domain.PaymentStatusUpdate
	reference := BulkTransferReferencePrefix + accountNumber

	for _, ts := range timesheets {
		var paymentDate *time.Time
		var paidAmount *int64

		if status == domain.PaymentStatusPaid {
			paymentDate = &now
			amount := s.calculatePaidAmount(ts.Amount, percentage)
			paidAmount = &amount
		}

		updates = append(updates, domain.PaymentStatusUpdate{
			TimesheetID:      ts.ID,
			PaymentStatus:    status,
			PaymentReference: &reference,
			PaymentDate:      paymentDate,
			PaidAmount:       paidAmount,
		})
	}

	return updates
}

// calculatePaidAmount calculates the paid amount based on percentage
func (s *BulkPaymentService) calculatePaidAmount(amount int64, percentage float64) int64 {
	return int64(float64(amount) * percentage)
}

// validateTotalAmount validates that the sum of paid amounts matches the file amount
func (s *BulkPaymentService) validateTotalAmount(
	updates []domain.PaymentStatusUpdate,
	fileAmount string,
) {
	if fileAmount == "" {
		return
	}

	var totalPaid int64
	for _, update := range updates {
		if update.PaidAmount != nil {
			totalPaid += *update.PaidAmount
		}
	}

	expectedAmount, err := s.rowParser.ParseAmountFromDetail(fileAmount)
	if err != nil {
		return
	}

	// Allow small rounding differences (up to 1 VND per timesheet)
	tolerance := int64(len(updates))
	diff := totalPaid - expectedAmount

	if diff < -tolerance || diff > tolerance {
		observability.GetLogger().Warn("sum of paid amounts differs from file amount",
			"totalPaid", totalPaid, "expectedAmount", expectedAmount, "diff", diff)
	}
}

// extractPaidAmount extracts the paid amount from an update
func (s *BulkPaymentService) extractPaidAmount(update domain.PaymentStatusUpdate) int64 {
	if update.PaidAmount != nil {
		return *update.PaidAmount
	}
	return 0
}

// detectPaymentScheduleFromFilename determines payment schedule from filename
func (s *BulkPaymentService) detectPaymentScheduleFromFilename(filename string) string {
	lowerFilename := strings.ToLower(filename)

	// Check for monthly pattern
	if strings.Contains(lowerFilename, pkgConstants.CycleMonthly) {
		return string(domain.PaymentScheduleMonthly)
	}

	// Check for weekly pattern
	if strings.Contains(lowerFilename, pkgConstants.CycleWeekly) {
		return string(domain.PaymentScheduleWeekly)
	}

	// Default to weekly
	return string(domain.PaymentScheduleWeekly)
}

// ApplyUpdates applies payment updates to the database
func (s *BulkPaymentService) ApplyUpdates(
	ctx context.Context,
	updates []PaymentUpdate,
) error {
	if len(updates) == 0 {
		return nil
	}

	// Extract domain updates
	domainUpdates := make([]domain.PaymentStatusUpdate, len(updates))
	for i, update := range updates {
		domainUpdates[i] = update.Update
	}

	return s.timesheetRepo.BulkUpdatePaymentStatus(ctx, domainUpdates)
}

// PrefetchCache prefetches employees and projects to reduce DB calls
func (s *BulkPaymentService) PrefetchCache(
	ctx context.Context,
	mappedRows []*MappedRow,
) error {
	// Collect unique IDs
	employeeIDs := make(map[uint]bool)
	projectIDs := make(map[uint]bool)

	for _, row := range mappedRows {
		if row.Employee != nil {
			employeeIDs[row.Employee.ID] = true
		}
		if row.ProjectID != 0 {
			projectIDs[row.ProjectID] = true
		}
	}

	// Prefetch employees
	if len(employeeIDs) > 0 {
		ids := make([]int64, 0, len(employeeIDs))
		for id := range employeeIDs {
			ids = append(ids, int64(id))
		}

		employees, err := s.employeeRepo.GetByIDs(ctx, ids)
		if err != nil {
			return fmt.Errorf("failed to prefetch employees: %w", err)
		}

		for _, emp := range employees {
			s.employeeCache[emp.ID] = emp
		}
	}

	// Prefetch projects (one by one as there's no bulk method)
	for projectID := range projectIDs {
		project, err := s.projectRepo.GetByID(ctx, projectID)
		if err != nil {
			return fmt.Errorf("failed to prefetch project %d: %w", projectID, err)
		}
		s.projectCache[projectID] = project
	}

	return nil
}

// GetEmployee retrieves an employee from cache or database
func (s *BulkPaymentService) GetEmployee(ctx context.Context, id uint) (*domain.Employee, error) {
	if emp, exists := s.employeeCache[id]; exists {
		return emp, nil
	}

	emp, err := s.employeeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	s.employeeCache[id] = emp
	return emp, nil
}

// GetProject retrieves a project from cache or database
func (s *BulkPaymentService) GetProject(ctx context.Context, id uint) (*domain.Project, error) {
	if proj, exists := s.projectCache[id]; exists {
		return proj, nil
	}

	proj, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	s.projectCache[id] = proj
	return proj, nil
}

// CalculateTotalTransferAmount calculates the total transfer amount (excluding failed)
func (s *BulkPaymentService) CalculateTotalTransferAmount(updates []PaymentUpdate) float64 {
	var total float64
	for _, update := range updates {
		if !update.IsFailed {
			total += float64(update.PaidAmount)
		}
	}
	return total
}
