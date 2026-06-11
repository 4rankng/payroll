package loan

import (
	"api-server/internal/constants"
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/domain"
	infraports "api-server/internal/domain/ports/infrastructure"
	serviceports "api-server/internal/domain/ports/services"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"
)

type LoanService struct {
	logger                *slog.Logger
	LoanRepo              domain.LoanRepository
	LenderRepo            domain.LenderRepository
	LedgerRepo            domain.LedgerEntryRepository
	TransactionRepo       domain.TransactionRepository
	transaction           serviceports.TransactionPort
	TxManager             domain.TransactionManager
	cache                 infraports.CachePort
	events                domain.EventBus
	notification          infraports.NotificationPort
	RepaymentScheduleRepo domain.LoanRepaymentScheduleRepository
}

func NewLoanService(
	loanRepo domain.LoanRepository,
	lenderRepo domain.LenderRepository,
	ledgerRepo domain.LedgerEntryRepository,
	transactionRepo domain.TransactionRepository,
	transaction serviceports.TransactionPort,
	txManager domain.TransactionManager,
	cache infraports.CachePort,
	events domain.EventBus,
	notification infraports.NotificationPort,
	repaymentScheduleRepo domain.LoanRepaymentScheduleRepository,
) *LoanService {
	return &LoanService{
		logger:                observability.GetLogger(),
		LoanRepo:              loanRepo,
		LenderRepo:            lenderRepo,
		LedgerRepo:            ledgerRepo,
		TransactionRepo:       transactionRepo,
		transaction:           transaction,
		TxManager:             txManager,
		cache:                 cache,
		events:                events,
		notification:          notification,
		RepaymentScheduleRepo: repaymentScheduleRepo,
	}
}

// isDuplicateKeyError checks if the error is a duplicate key constraint violation
func (s *LoanService) isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// Check for MySQL duplicate entry errors
	return len(errMsg) > 0 && (
	// MySQL error patterns
	len(errMsg) >= 9 && errMsg[:9] == "Duplicate" ||
		len(errMsg) >= 9 && errMsg[:9] == "duplicate" ||
		checkSubstring(errMsg, "Duplicate entry") ||
		checkSubstring(errMsg, "duplicate key") ||
		checkSubstring(errMsg, "unique constraint") ||
		checkSubstring(errMsg, "Error 1062"))
}

// checkSubstring checks if a string contains a substring
func checkSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// DisburseLoan disburses a loan and creates ledger entries
func (s *LoanService) DisburseLoan(ctx context.Context, loanID uint, disbursementDate time.Time, reference string, createdBy uint) (*domain.Loan, []uint, error) {
	// Get loan
	loan, err := s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, nil, err
	}

	// Check if can disburse
	if err := loan.CanDisburse(); err != nil {
		return nil, nil, err
	}

	var ledgerEntryIDs []uint
	var createdTransactionID *uint

	// Execute in database transaction
	err = s.TxManager.WithTransaction(ctx, func(txCtx context.Context) error {

		// Create a user-facing transaction record for disbursement (settled)
		txn := &domain.Transaction{
			Description:     fmt.Sprintf("Tiền vay nợ từ %s", loan.Lender.Name),
			TransactionType: domain.TransactionTypeLoanDisbursement,
			Amount:          loan.PrincipalAmount,
			Party:           loan.Lender.Name,
			Status:          domain.TransactionStatusSettled,
			LoanID:          &loan.ID,
			CreatedBy:       createdBy,
		}
		if err := s.TransactionRepo.Create(txCtx, txn); err != nil {
			return fmt.Errorf("failed to create disbursement transaction: %w", err)
		}
		createdTransactionID = &txn.ID
		// Create ledger entries: Dr Cash, Cr Loan (liability)

		ledgerEntries := []*domain.LedgerEntry{
			// Debit: Cash (increase asset)
			{
				Date:          disbursementDate,
				Account:       domain.AccountCash,
				Party:         loan.Lender.Name,
				Debit:         loan.PrincipalAmount,
				Credit:        0,
				CreatedBy:     createdBy,
				TransactionID: createdTransactionID,
			},
			// Credit: Loan (increase liability)
			{
				Date:          disbursementDate,
				Account:       domain.AccountLoan,
				Party:         loan.Lender.Name,
				Debit:         0,
				Credit:        loan.PrincipalAmount,
				CreatedBy:     createdBy,
				TransactionID: createdTransactionID,
			},
		}

		// Create ledger entries
		if err := s.LedgerRepo.CreateTransaction(txCtx, ledgerEntries); err != nil {
			return fmt.Errorf("failed to create ledger entries: %w", err)
		}

		// Collect ledger entry IDs
		for _, entry := range ledgerEntries {
			ledgerEntryIDs = append(ledgerEntryIDs, entry.ID)
		}

		// Update loan
		now := clock.Now()
		loan.DisbursedAt = &now
		loan.OutstandingPrincipal = loan.PrincipalAmount

		if err := s.LoanRepo.Update(txCtx, loan); err != nil {
			return fmt.Errorf("failed to update loan: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	// Publish event
	event := domain.NewLoanDisbursedEvent(ctx, loan)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish LoanDisbursed event", "loanID", loan.ID, "error", err)
	}

	// Invalidate cache
	s.invalidateLoanCache(ctx)

	s.logger.Info("Loan disbursed successfully", "loanID", loanID, "amount", loan.PrincipalAmount)

	// Reload loan
	loan, err = s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, ledgerEntryIDs, err
	}

	return loan, ledgerEntryIDs, nil
}

// RepayPrincipal makes a principal repayment
func (s *LoanService) RepayPrincipal(ctx context.Context, loanID uint, amount int64, paymentDate time.Time, reference string, notes *string, createdBy uint) (*domain.Loan, []uint, error) {
	// Get loan
	loan, err := s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, nil, err
	}

	// Check if can repay
	if err := loan.CanRepay(amount); err != nil {
		return nil, nil, err
	}

	var ledgerEntryIDs []uint
	var createdTransactionID *uint

	// Execute in database transaction
	err = s.TxManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create a user-facing transaction record for principal repayment (settled)
		desc := reference
		if notes != nil && *notes != "" {
			desc = fmt.Sprintf("%s - %s", desc, *notes)
		}
		txn := &domain.Transaction{
			Description:     desc,
			TransactionType: domain.TransactionTypeLoanRepayment,
			Amount:          amount,
			Party:           loan.Lender.Name,
			Status:          domain.TransactionStatusSettled,
			LoanID:          &loan.ID,
			CreatedBy:       createdBy,
		}
		if err := s.TransactionRepo.Create(txCtx, txn); err != nil {
			return fmt.Errorf("failed to create principal repayment transaction: %w", err)
		}
		createdTransactionID = &txn.ID
		// Create ledger entries: Dr Loan (liability), Cr Cash

		ledgerEntries := []*domain.LedgerEntry{
			// Debit: Loan (decrease liability)
			{
				Date:          paymentDate,
				Account:       domain.AccountLoan,
				Party:         loan.Lender.Name,
				Debit:         amount,
				Credit:        0,
				CreatedBy:     createdBy,
				TransactionID: createdTransactionID,
			},
			// Credit: Cash (decrease asset)
			{
				Date:          paymentDate,
				Account:       domain.AccountCash,
				Party:         loan.Lender.Name,
				Debit:         0,
				Credit:        amount,
				CreatedBy:     createdBy,
				TransactionID: createdTransactionID,
			},
		}

		// Create ledger entries
		if err := s.LedgerRepo.CreateTransaction(txCtx, ledgerEntries); err != nil {
			return fmt.Errorf("failed to create ledger entries: %w", err)
		}

		txnID := txn.ID

		// Record repayment in schedule table to keep timeline consistent
		if err := s.recordPrincipalRepaymentSchedule(txCtx, loan, amount, paymentDate, reference, &txnID); err != nil {
			return fmt.Errorf("failed to record principal repayment schedule: %w", err)
		}

		// Collect ledger entry IDs
		for _, entry := range ledgerEntries {
			ledgerEntryIDs = append(ledgerEntryIDs, entry.ID)
		}

		// Update loan
		loan.OutstandingPrincipal -= amount

		// If fully repaid, close the loan
		if loan.OutstandingPrincipal == 0 {
			loan.Status = domain.LoanStatusClosed
		}

		if err := s.LoanRepo.Update(txCtx, loan); err != nil {
			return fmt.Errorf("failed to update loan: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	// Publish event
	event := domain.NewLoanUpdatedEvent(ctx, loan, nil)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish LoanUpdated event", "loanID", loan.ID, "error", err)
	}

	// Invalidate cache
	s.invalidateLoanCache(ctx)

	s.logger.Info("Loan repayment successful", "loanID", loanID, "amount", amount, "outstanding", loan.OutstandingPrincipal)

	// Reload loan
	loan, err = s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, ledgerEntryIDs, err
	}

	return loan, ledgerEntryIDs, nil
}

// ProcessLoanPayment processes a payment for a loan according to its type
func (s *LoanService) ProcessLoanPayment(ctx context.Context, loanID uint, scheduleID uint, paymentDate time.Time, reference string, notes *string, createdBy uint) (*domain.Loan, []uint, error) {
	// Get loan
	loan, err := s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, nil, err
	}

	// Get the schedule that is being paid
	schedule, err := s.RepaymentScheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return nil, nil, err
	}

	// Verify the schedule belongs to the loan
	if schedule.LoanID != loanID {
		return nil, nil, domain.NewValidationError(constants.MsgScheduleNotBelongToLoanVN)
	}

	strategy := domain.NewCustomScheduleStrategy(s.RepaymentScheduleRepo)

	var ledgerEntryIDs []uint
	var createdTransactionID *uint

	// Execute in database transaction
	err = s.TxManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create a user-facing transaction record for the scheduled payment (settled)
		desc := reference
		if notes != nil && *notes != "" {
			desc = fmt.Sprintf("%s - %s", desc, *notes)
		}
		txn := &domain.Transaction{
			Description:     desc,
			TransactionType: domain.TransactionTypeLoanRepayment,
			Amount:          schedule.Amount,
			Party:           loan.Lender.Name,
			Status:          domain.TransactionStatusSettled,
			LoanID:          &loan.ID,
			CreatedBy:       createdBy,
		}
		if err := s.TransactionRepo.Create(txCtx, txn); err != nil {
			return fmt.Errorf("failed to create scheduled repayment transaction: %w", err)
		}
		createdTransactionID = &txn.ID

		// Create ledger entries: Dr Loan (liability), Cr Cash

		ledgerEntries := []*domain.LedgerEntry{
			// Debit: Loan (decrease liability)
			{
				Date:          paymentDate,
				Account:       domain.AccountLoan,
				Party:         loan.Lender.Name,
				Debit:         schedule.Amount,
				Credit:        0,
				CreatedBy:     createdBy,
				TransactionID: createdTransactionID,
			},
			// Credit: Cash (decrease asset)
			{
				Date:          paymentDate,
				Account:       domain.AccountCash,
				Party:         loan.Lender.Name,
				Debit:         0,
				Credit:        schedule.Amount,
				CreatedBy:     createdBy,
				TransactionID: createdTransactionID,
			},
		}

		// Create ledger entries
		if err := s.LedgerRepo.CreateTransaction(txCtx, ledgerEntries); err != nil {
			return fmt.Errorf("failed to create ledger entries: %w", err)
		}

		// Collect ledger entry IDs
		for _, entry := range ledgerEntries {
			ledgerEntryIDs = append(ledgerEntryIDs, entry.ID)
		}

		// Process payment using loan strategy
		if err := strategy.ProcessPayment(loan, schedule.Amount, scheduleID); err != nil {
			return err
		}

		// Update loan with changes made by strategy
		if err := s.LoanRepo.Update(txCtx, loan); err != nil {
			return fmt.Errorf("failed to update loan after processing payment: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	// Publish event
	event := domain.NewLoanUpdatedEvent(ctx, loan, nil)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish LoanUpdated event", "loanID", loan.ID, "error", err)
	}

	// Invalidate cache
	s.invalidateLoanCache(ctx)

	s.logger.Info("Loan payment processed successfully", "loanID", loanID, "scheduleID", scheduleID, "amount", schedule.Amount, "outstanding", loan.OutstandingPrincipal)

	// Reload loan
	loan, err = s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, ledgerEntryIDs, err
	}

	return loan, ledgerEntryIDs, nil
}

// CreateTransactionsForDueSchedules generates pending transactions for schedules that are due
func (s *LoanService) CreateTransactionsForDueSchedules(ctx context.Context, asOfDate time.Time, systemUserID uint) (int, error) {
	if s.transaction == nil {
		return 0, fmt.Errorf("transaction service is not initialized")
	}

	type notificationPayload struct {
		loanCode      string
		period        int
		dueDate       time.Time
		amount        int64
		transactionID uint
	}

	dueSchedules, err := s.RepaymentScheduleRepo.GetDueSchedules(ctx, asOfDate)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch due repayment schedules: %w", err)
	}

	if len(dueSchedules) == 0 {
		return 0, nil
	}

	successCount := 0
	var notifications []notificationPayload

	for _, sched := range dueSchedules {
		var notif *notificationPayload
		err := s.TxManager.WithTransaction(ctx, func(txCtx context.Context) error {
			currentSchedule, err := s.RepaymentScheduleRepo.GetByID(txCtx, sched.ID)
			if err != nil {
				return err
			}

			// Skip if processed concurrently
			if currentSchedule.Status != domain.ScheduleStatusPending || currentSchedule.TransactionID != nil {
				return nil
			}

			loan, err := s.LoanRepo.GetByID(txCtx, currentSchedule.LoanID)
			if err != nil {
				return err
			}

			txn := &domain.Transaction{
				TransactionType: domain.TransactionTypeLoanRepayment,
				Amount:          currentSchedule.Amount,
				Party:           loan.Lender.Name,
				Status:          domain.TransactionStatusPending,
				LoanID:          &loan.ID,
				CreatedBy:       systemUserID,
			}

			if err := s.transaction.Create(txCtx, txn); err != nil {
				return fmt.Errorf("failed to create pending loan repayment transaction: %w", err)
			}
			createdTxn := txn

			currentSchedule.TransactionID = &createdTxn.ID
			if err := s.RepaymentScheduleRepo.Update(txCtx, currentSchedule); err != nil {
				return fmt.Errorf("failed to update loan repayment schedule: %w", err)
			}

			notif = &notificationPayload{
				loanCode:      loan.LoanCode,
				period:        currentSchedule.Period,
				dueDate:       currentSchedule.DueDate,
				amount:        currentSchedule.Amount,
				transactionID: createdTxn.ID,
			}

			return nil
		})

		if err != nil {
			s.logger.Error("Failed to create transaction for due loan schedule",
				"schedule_id", sched.ID,
				"loan_id", sched.LoanID,
				"error", err)
			continue
		}

		if notif != nil {
			notifications = append(notifications, *notif)
		}

		successCount++
	}

	if successCount > 0 {
		s.invalidateLoanCache(ctx)
	}

	if len(notifications) > 0 && s.notification != nil {
		for _, payload := range notifications {
			title := fmt.Sprintf("Khoản vay %s đến hạn trả", payload.loanCode)
			message := fmt.Sprintf(
				"Kỳ #%d của khoản vay %s đến hạn ngày %s với số tiền %s. Giao dịch #%d đã được tạo và chờ thanh toán.",
				payload.period,
				payload.loanCode,
				payload.dueDate.Format("02/01/2006"),
				utils.FormatVND(payload.amount),
				payload.transactionID,
			)

			// Note: NotificationPort doesn't have NotifyUsersByRole method
			// This would need to be adapted based on the actual port interface
			s.logger.Info("Loan repayment notification", "title", title, "message", message)
		}
	}

	return successCount, nil
}

// GetLoanSchedule generates the payment schedule for a loan
func (s *LoanService) GetLoanSchedule(ctx context.Context, loanID uint) ([]domain.ScheduleItem, error) {
	// Get loan
	loan, err := s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, err
	}

	strategy := domain.NewCustomScheduleStrategy(s.RepaymentScheduleRepo)
	return strategy.GenerateSchedule(loan)
}

// GetLoan retrieves a loan by ID
func (s *LoanService) GetLoan(ctx context.Context, id uint) (*domain.Loan, error) {
	return s.LoanRepo.GetByID(ctx, id)
}

// ListLoans retrieves a paginated list of loans
func (s *LoanService) ListLoans(ctx context.Context, filters domain.LoanFilters) ([]*domain.Loan, int64, error) {
	loans, total, err := s.LoanRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list loans: %w", err)
	}

	return loans, total, nil
}

// UpdateLoan updates loan metadata (description, payment day)
func (s *LoanService) UpdateLoan(ctx context.Context, id uint, updates map[string]interface{}) (*domain.Loan, error) {
	// Get existing loan
	loan, err := s.LoanRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates (only non-financial fields)
	if description, ok := updates["description"]; ok {
		if description == nil {
			loan.Description = nil
		} else {
			descStr := description.(string)
			loan.Description = &descStr
		}
	}

	if paymentDay, ok := updates["payment_day_of_month"]; ok && paymentDay != nil {
		loan.PaymentDayOfMonth = paymentDay.(uint8)
	}

	// Update loan
	if err := s.LoanRepo.Update(ctx, loan); err != nil {
		return nil, fmt.Errorf("failed to update loan: %w", err)
	}

	// Publish event
	event := domain.NewLoanUpdatedEvent(ctx, loan, nil)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish LoanUpdated event", "loanID", loan.ID, "error", err)
	}

	// Invalidate cache
	s.invalidateLoanCache(ctx)

	s.logger.Info("Loan updated successfully", "loanID", loan.ID)

	return loan, nil
}

// UpdateInterestPaid updates the total interest paid (called after settlement)
func (s *LoanService) UpdateInterestPaid(ctx context.Context, loanID uint, amount int64) error {
	loan, err := s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return err
	}

	loan.TotalInterestPaid += amount

	if err := s.LoanRepo.Update(ctx, loan); err != nil {
		return fmt.Errorf("failed to update loan interest paid: %w", err)
	}

	s.logger.Info("Loan interest paid updated", "loanID", loanID, "amount", amount, "total", loan.TotalInterestPaid)

	return nil
}

// recordPrincipalRepaymentSchedule ensures principal repayments are reflected in the repayment schedule table
func (s *LoanService) recordPrincipalRepaymentSchedule(ctx context.Context, loan *domain.Loan, amount int64, paymentDate time.Time, reference string, transactionID *uint) error {
	schedules, err := s.RepaymentScheduleRepo.GetByLoanID(ctx, loan.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch repayment schedules: %w", err)
	}

	var pendingTarget *domain.LoanRepaymentSchedule
	nextPeriod := 1

	for _, schedule := range schedules {
		if schedule.Period >= nextPeriod {
			nextPeriod = schedule.Period + 1
		}
		if schedule.Status == domain.ScheduleStatusPending {
			pendingTarget = schedule
		}
	}

	now := clock.Now()

	if pendingTarget != nil {
		switch {
		case pendingTarget.Amount == amount:
			pendingTarget.Status = domain.ScheduleStatusPaid
			pendingTarget.PaidAt = &now
			if reference != "" {
				pendingTarget.PaymentRef = &reference
			}
			if transactionID != nil {
				pendingTarget.TransactionID = transactionID
			}
			if err := s.RepaymentScheduleRepo.Update(ctx, pendingTarget); err != nil {
				return fmt.Errorf("failed to update repayment schedule: %w", err)
			}
			return nil
		case pendingTarget.Amount > amount:
			pendingTarget.Amount -= amount
			if err := s.RepaymentScheduleRepo.Update(ctx, pendingTarget); err != nil {
				return fmt.Errorf("failed to update repayment schedule: %w", err)
			}
		default:
			pendingTarget = nil
		}
	}

	newSchedule := &domain.LoanRepaymentSchedule{
		LoanID:        loan.ID,
		Period:        nextPeriod,
		DueDate:       paymentDate,
		Amount:        amount,
		Status:        domain.ScheduleStatusPaid,
		PaidAt:        &now,
		TransactionID: transactionID,
	}

	if reference != "" {
		newSchedule.PaymentRef = &reference
	}

	if err := s.RepaymentScheduleRepo.Create(ctx, newSchedule); err != nil {
		return fmt.Errorf("failed to create repayment schedule entry: %w", err)
	}

	return nil
}

// invalidateLoanCache invalidates loan-related cache entries
func (s *LoanService) invalidateLoanCache(ctx context.Context) {
	patterns := []string{
		"loan:*",
		"loans:*",
	}

	for _, pattern := range patterns {
		if err := s.cache.InvalidatePattern(ctx, pattern); err != nil {
			s.logger.Error("Failed to invalidate loan cache", "pattern", pattern, "error", err)
		}
	}
}

// DeleteLoan deletes a loan that has not been disbursed
func (s *LoanService) DeleteLoan(ctx context.Context, id uint) error {
	// Load loan
	loan, err := s.LoanRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Business rule: cannot delete if already disbursed
	if loan.DisbursedAt != nil {
		return domain.NewValidationError(constants.MsgLoanAlreadyDisbursedVN2)
	}

	// Update loan_code to free up the original code for reuse
	// Append timestamp to make it unique
	originalCode := loan.LoanCode
	loan.LoanCode = fmt.Sprintf("%s-DELETED-%d", loan.LoanCode, clock.Now().Unix())
	if err := s.LoanRepo.Update(ctx, loan); err != nil {
		return fmt.Errorf("failed to update loan code: %w", err)
	}

	// Perform soft delete
	if err := s.LoanRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete loan: %w", err)
	}

	// Publish event
	event := domain.NewLoanDeletedEvent(ctx, loan)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish LoanDeleted event", "loanID", loan.ID, "error", err)
	}

	// Invalidate cache
	s.invalidateLoanCache(ctx)

	s.logger.Info("Loan deleted successfully", "loanID", id, "originalCode", originalCode)
	return nil
}
