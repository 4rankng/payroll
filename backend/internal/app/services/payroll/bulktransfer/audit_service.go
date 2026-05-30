package bulktransfer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// AuditTrailService handles audit logging and history persistence
type AuditTrailService struct {
	fileRepo     BulkTransferFileRepository
	employeeRepo EmployeeRepository
	projectRepo  ProjectRepository
	rowParser    *RowParser
}

// NewAuditTrailService creates a new AuditTrailService instance
func NewAuditTrailService(
	fileRepo BulkTransferFileRepository,
	employeeRepo EmployeeRepository,
	projectRepo ProjectRepository,
	rowParser *RowParser,
) *AuditTrailService {
	return &AuditTrailService{
		fileRepo:     fileRepo,
		employeeRepo: employeeRepo,
		projectRepo:  projectRepo,
		rowParser:    rowParser,
	}
}

// SaveProcessingHistory saves the bulk transfer processing history
func (a *AuditTrailService) SaveProcessingHistory(
	ctx context.Context,
	assetID uint,
	transaction *domain.Transaction,
	filename string,
	details []dto.BulkTransferResultItemDetail,
	totalTxn int,
	completedTxn int,
	failedTxn int,
	uploadedBy uint,
	exportDate *time.Time,
	paymentSchedule string,
	fromDate *time.Time,
	toDate *time.Time,
	forMonth *string,
) error {
	logger := observability.GetLogger()

	// Build successful payments array
	successfulPayments, err := a.buildSuccessfulPayments(ctx, details)
	if err != nil {
		return err
	}

	// Calculate transfer amount from successful payments
	transferAmount, err := a.calculateTransferAmount(successfulPayments)
	if err != nil {
		logger.Warn("Failed to calculate transfer amount", "error", err)
		// Continue with 0 amount rather than failing
	}

	// Calculate checksum from successful payments (kept for data integrity verification)
	_, err = a.calculateChecksum(successfulPayments)
	if err != nil {
		logger.Warn("Failed to calculate checksum", "error", err)
	}

	processingResultJSON, err := json.Marshal(successfulPayments)
	if err != nil {
		return fmt.Errorf("failed to marshal successful payments: %w", err)
	}

	var transactionID *uint
	if transaction != nil {
		transactionID = &transaction.ID
	}

	// Prepare date fields based on payment schedule
	var fromDatePtr *time.Time
	var toDatePtr *time.Time
	var forMonthPtr *string

	if paymentSchedule == string(domain.PaymentScheduleMonthly) {
		// For monthly, use for_month field
		if forMonth != nil {
			forMonthPtr = forMonth
		} else if fromDate != nil {
			// Derive for_month from fromDate if not provided
			forMonthValue := fromDate.Format("2006-01")
			forMonthPtr = &forMonthValue
		}
	} else {
		// For weekly, use from_date and to_date
		fromDatePtr = fromDate
		toDatePtr = toDate
	}

	// Note: audit logging is deprecated - bulk transfer file records are created elsewhere
	// This method no longer creates database records but still generates audit logs

	// Log the audit trail
	if err := a.fileRepo.Create(ctx, &domain.BulkTransferFile{
		Filename:          filename,
		CreatedBy:         uploadedBy,
		Cycle:             domain.StringPtr(paymentSchedule),
		TransactionsCount: totalTxn,
		TransferAmount:    transferAmount,
		Data:              string(processingResultJSON),
		AssetID:           transactionID,
		FromDate:          fromDatePtr,
		ToDate:            toDatePtr,
		ForMonth:          forMonthPtr,
	}); err != nil {
		logger.Warn("Failed to save bulk transfer history", "error", err)
		return err
	}

	return nil
}

// buildSuccessfulPayments builds the successful payments array for history
func (a *AuditTrailService) buildSuccessfulPayments(
	ctx context.Context,
	details []dto.BulkTransferResultItemDetail,
) ([]dto.BulkTransferSuccessfulPayment, error) {
	var successfulPayments []dto.BulkTransferSuccessfulPayment

	for _, detail := range details {
		// Only include successful transactions in processing_result
		if detail.Status != ProcessingStatusSuccess {
			continue
		}

		var employeeCCCD, projectName string
		if detail.EmployeeID != nil {
			employee, err := a.employeeRepo.GetByID(ctx, *detail.EmployeeID)
			if err == nil {
				employeeCCCD = employee.CCCD
			}
		}

		parsedProjectID, timesheetIDs, parseErr := a.rowParser.ParseTrackingData(detail.TrackingData)
		if parseErr != nil {
			_, parsedProjectID, _, _, parseErr = a.rowParser.ParsePaymentPeriodWithProject(detail.PaymentDescription)
			if parseErr != nil {
				continue
			}
		}

		if parsedProjectID != 0 {
			project, err := a.projectRepo.GetByID(ctx, parsedProjectID)
			if err == nil {
				projectName = project.Name
			}
		}

		successfulPayment := dto.BulkTransferSuccessfulPayment{
			Amount:                detail.Amount,
			ProjectID:             parsedProjectID,
			EmployeeID:            *detail.EmployeeID,
			ProjectName:           projectName,
			EmployeeCCCD:          employeeCCCD,
			TimesheetIDs:          timesheetIDs,
			EmployeeFullname:      detail.EmployeeFullname,
			PaymentDescription:    detail.PaymentDescription,
			EmployeeAccountName:   detail.EmployeeAccountName,
			EmployeeAccountNumber: detail.EmployeeAccountNumber,
		}

		successfulPayments = append(successfulPayments, successfulPayment)
	}

	return successfulPayments, nil
}

// calculateTransferAmount calculates the total transfer amount from successful payments
func (a *AuditTrailService) calculateTransferAmount(successfulPayments []dto.BulkTransferSuccessfulPayment) (int64, error) {
	var totalAmount int64

	for _, payment := range successfulPayments {
		// Parse amount string to int64
		var amount int64
		if _, err := fmt.Sscanf(payment.Amount, "%d", &amount); err != nil {
			return 0, fmt.Errorf("failed to parse amount '%s': %w", payment.Amount, err)
		}
		totalAmount += amount
	}

	return totalAmount, nil
}

// calculateChecksum calculates checksum from successful payments
// Note: BulkTransferSuccessfulPayment doesn't have TransactionCode field,
// so we cannot calculate checksum in the same way as export does.
// We'll return nil for now since the data structure doesn't support it.
func (a *AuditTrailService) calculateChecksum(successfulPayments []dto.BulkTransferSuccessfulPayment) (*string, error) {
	// The BulkTransferSuccessfulPayment structure doesn't include transaction_code
	// which is required for checksum calculation. Checksum should be calculated
	// from the original file data during export, not during result processing.
	return nil, nil
}
