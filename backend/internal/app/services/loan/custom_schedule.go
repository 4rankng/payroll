package loan

import (
	"api-server/internal/constants"
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
)

// CreateCustomScheduleLoan creates a new loan with custom repayment schedule
func (s *LoanService) CreateCustomScheduleLoan(ctx context.Context, loan *domain.Loan, schedules []domain.LoanRepaymentSchedule, createdBy uint) (*domain.Loan, error) {
	// Calculate derived fields from schedules
	if len(schedules) > 0 {
		// Start date should be the earliest schedule date
		loan.StartDate = schedules[0].DueDate
		// End date should be the latest schedule date
		loan.EndDate = schedules[len(schedules)-1].DueDate

		// Calculate term months from number of schedules
		loan.TermMonths = len(schedules)

		// Calculate total repayment amount
		var totalRepayment int64
		for _, schedule := range schedules {
			totalRepayment += schedule.Amount
		}

		// Calculate total interest (difference between total repayment and principal)
		totalInterest := totalRepayment - loan.PrincipalAmount

		// Derive effective annual interest rate in basis points
		if loan.TermMonths > 0 && loan.PrincipalAmount > 0 {
			loan.InterestRateBps = int((totalInterest * 12 * 10000) / (loan.PrincipalAmount * int64(loan.TermMonths)))
			if loan.InterestRateBps < 0 {
				loan.InterestRateBps = 0
			}
		} else {
			loan.InterestRateBps = 0
		}
	}
	// Validate loan after derived fields are set
	if err := loan.Validate(); err != nil {
		return nil, err
	}

	// Verify lender exists
	_, err := s.LenderRepo.GetByID(ctx, loan.LenderID)
	if err != nil {
		return nil, domain.NewNotFoundError(constants.MsgLenderNotFoundVN2)
	}

	// Verify schedules are valid
	if len(schedules) == 0 {
		return nil, domain.NewValidationError(constants.MsgMinOneScheduleRequiredVN)
	}
	if err := domain.AllocateRepaymentScheduleComponents(loan.PrincipalAmount, schedules); err != nil {
		return nil, err
	}

	// Verify schedule dates are in order
	for i := 0; i < len(schedules)-1; i++ {
		if schedules[i].DueDate.After(schedules[i+1].DueDate) {
			return nil, domain.NewValidationError(constants.MsgScheduleDueDatesMustAscendVN)
		}
	}

	// Retry logic to handle race conditions in loan code generation
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Wrap sequence generation and loan creation in a transaction for atomicity
		err := s.TxManager.WithTransaction(ctx, func(txCtx context.Context) error {
			// Generate loan code within transaction
			sequence, err := s.LoanRepo.GetNextSequenceForYear(txCtx, clock.Now().Year())
			if err != nil {
				return fmt.Errorf("failed to generate loan code: %w", err)
			}
			loan.LoanCode = domain.GenerateLoanCode(clock.Now(), sequence)

			// Set initial values
			loan.OutstandingPrincipal = 0 // Will be set when loan is disbursed
			loan.TotalInterestPaid = 0
			loan.Status = domain.LoanStatusActive
			loan.CreatedBy = createdBy

			// Create loan within transaction
			if err := s.LoanRepo.Create(txCtx, loan); err != nil {
				return err
			}

			// Create repayment schedules
			for i := range schedules {
				schedules[i].LoanID = loan.ID
				schedules[i].Period = i + 1
				schedules[i].Status = domain.ScheduleStatusPending
				if err := s.RepaymentScheduleRepo.Create(txCtx, &schedules[i]); err != nil {
					return fmt.Errorf("failed to create repayment schedule: %w", err)
				}
			}

			return nil
		})

		// Success - break out of retry loop
		if err == nil {
			break
		}

		lastErr = err

		// Check if this is a duplicate key error
		isDuplicate := s.isDuplicateKeyError(err)

		// If it's a duplicate and we have retries left, wait and retry
		if isDuplicate && attempt < maxRetries-1 {
			s.logger.Warn("Duplicate loan code detected, retrying",
				"attempt", attempt+1,
				"maxRetries", maxRetries,
				"error", err)

			// Exponential backoff: 50ms, 100ms, 200ms
			backoff := time.Duration(50*(1<<uint(attempt))) * time.Millisecond
			time.Sleep(backoff)
			continue
		}

		// If it's not a duplicate or we're out of retries, return error
		if isDuplicate {
			return nil, domain.NewValidationError(constants.MsgFailedToGenerateUniqueLoanCodeVN)
		}

		return nil, fmt.Errorf("failed to create loan: %w", err)
	}

	// If we exhausted all retries
	if lastErr != nil {
		return nil, fmt.Errorf("failed to create loan after %d attempts: %w", maxRetries, lastErr)
	}

	// Publish event
	event := domain.NewLoanCreatedEvent(ctx, loan)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish LoanCreated event", "loanID", loan.ID, "error", err)
	}

	// Invalidate cache
	s.invalidateLoanCache(ctx)

	s.logger.Info("Custom schedule loan created successfully", "loanID", loan.ID, "loanCode", loan.LoanCode)

	// Reload with relationships
	return s.LoanRepo.GetByID(ctx, loan.ID)
}

// ProcessScheduledPayment processes a payment against a specific schedule
func (s *LoanService) ProcessScheduledPayment(ctx context.Context, loanID uint, scheduleID uint, paymentDate time.Time, reference string, notes *string, createdBy uint) (*domain.Loan, []uint, error) {
	// Load the original state for the event. The financial mutation below is
	// performed under row locks in one transaction.
	loan, err := s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, nil, err
	}
	originalLoan := *loan

	var ledgerEntryIDs []uint
	var processedSchedule domain.LoanRepaymentSchedule
	err = s.TxManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Lock the schedule first, so duplicate requests serialize before any
		// settlement or ledger entries are created.
		currentSchedule, err := s.RepaymentScheduleRepo.GetByIDForUpdate(txCtx, scheduleID)
		if err != nil {
			return err
		}
		if currentSchedule.LoanID != loanID {
			return domain.NewValidationError(constants.MsgScheduleNotBelongToLoanVN)
		}

		if currentSchedule.Status == domain.ScheduleStatusPaid {
			return domain.NewValidationError(constants.MsgScheduleAlreadyPaidVN)
		}
		if err := currentSchedule.ValidateComponents(); err != nil {
			return err
		}

		// Lock the loan too. Different installments for the same loan may be
		// paid concurrently, and each must apply to the latest aggregates.
		loanForUpdate, err := s.LoanRepo.GetByIDForUpdate(txCtx, loanID)
		if err != nil {
			return err
		}

		var transactionID *uint
		if currentSchedule.TransactionID != nil {
			txn, err := s.TransactionRepo.GetByIDForUpdate(txCtx, *currentSchedule.TransactionID)
			if err != nil {
				return fmt.Errorf("failed to fetch existing transaction: %w", err)
			}
			if txn.Status == domain.TransactionStatusPending {
				settlement := &domain.Settlement{
					TransactionID:  txn.ID,
					Amount:         currentSchedule.Amount,
					SettlementDate: paymentDate,
					CreatedBy:      createdBy,
				}
				if notes != nil && *notes != "" {
					settlement.Notes = *notes
				}
				updatedTxn, _, entries, err := s.transaction.CreateSettlement(txCtx, txn.ID, settlement)
				if err != nil {
					return fmt.Errorf("failed to settle scheduled repayment transaction: %w", err)
				}
				transactionID = &updatedTxn.ID
				for _, entry := range entries {
					ledgerEntryIDs = append(ledgerEntryIDs, entry.ID)
				}
			} else {
				return domain.NewValidationError("Giao dịch của kỳ thanh toán không còn ở trạng thái chờ")
			}
		}

		if transactionID == nil {
			desc := fmt.Sprintf("Trả nợ theo lịch %s - Kỳ %d", loanForUpdate.LoanCode, currentSchedule.Period)
			if notes != nil && *notes != "" {
				desc = fmt.Sprintf("%s - %s", desc, *notes)
			}
			txn := &domain.Transaction{
				Description:         desc,
				TransactionType:     domain.TransactionTypeLoanRepayment,
				Amount:              currentSchedule.Amount,
				LoanPrincipalAmount: currentSchedule.PrincipalAmount,
				LoanInterestAmount:  currentSchedule.InterestAmount,
				Party:               loanForUpdate.Lender.Name,
				Status:              domain.TransactionStatusSettled,
				LoanID:              &loanForUpdate.ID,
				CreatedBy:           createdBy,
			}
			createdTxn, entries, err := s.transaction.CreateTransaction(txCtx, txn)
			if err != nil {
				return fmt.Errorf("failed to create scheduled repayment transaction: %w", err)
			}
			transactionID = &createdTxn.ID
			for _, entry := range entries {
				ledgerEntryIDs = append(ledgerEntryIDs, entry.ID)
			}
		}

		// Update the schedule as paid only after its financial transaction is
		// staged in the same database transaction.
		now := clock.Now()
		currentSchedule.Status = domain.ScheduleStatusPaid
		currentSchedule.PaidAt = &now
		if reference != "" {
			currentSchedule.PaymentRef = &reference
		}
		currentSchedule.TransactionID = transactionID

		if err := s.RepaymentScheduleRepo.Update(txCtx, currentSchedule); err != nil {
			return fmt.Errorf("failed to update repayment schedule: %w", err)
		}

		if err := loanForUpdate.ApplyScheduledPayment(currentSchedule.PrincipalAmount, currentSchedule.InterestAmount); err != nil {
			return err
		}

		if err := s.LoanRepo.Update(txCtx, loanForUpdate); err != nil {
			return fmt.Errorf("failed to update loan: %w", err)
		}

		// Update outer values for logging and the domain event after commit.
		processedSchedule = *currentSchedule
		// Update outer loan variable
		*loan = *loanForUpdate

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	// Publish event
	event := domain.NewLoanUpdatedEvent(ctx, loan, &originalLoan)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish LoanUpdated event", "loanID", loanID, "error", err)
	}

	// Invalidate cache
	s.invalidateLoanCache(ctx)

	s.logger.Info("Scheduled loan payment successful", "loanID", loanID, "scheduleID", scheduleID, "amount", processedSchedule.Amount, "outstanding", loan.OutstandingPrincipal)

	// Reload loan to ensure we return the latest state
	loan, err = s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, ledgerEntryIDs, err
	}

	return loan, ledgerEntryIDs, nil
}

// GetLoanWithSchedules retrieves a loan with its repayment schedules
func (s *LoanService) GetLoanWithSchedules(ctx context.Context, loanID uint) (*domain.Loan, []*domain.LoanRepaymentSchedule, error) {
	// Get loan
	loan, err := s.LoanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, nil, err
	}

	// Get repayment schedules
	schedules, err := s.RepaymentScheduleRepo.GetByLoanID(ctx, loanID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get repayment schedules: %w", err)
	}

	return loan, schedules, nil
}

// GetSchedulesByLoanIDs batch-fetches schedules for multiple loans in one query,
// grouped by loan id. Used by list endpoints to avoid an N+1 of
// GetLoanWithSchedules per loan. Loans with no schedules are absent from the map.
func (s *LoanService) GetSchedulesByLoanIDs(ctx context.Context, loanIDs []uint) (map[uint][]*domain.LoanRepaymentSchedule, error) {
	return s.RepaymentScheduleRepo.GetByLoanIDs(ctx, loanIDs)
}
