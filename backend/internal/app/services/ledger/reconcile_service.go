package ledger

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"

	"gorm.io/gorm"
)

// ReconcileService handles account-level rounding adjustments
type ReconcileService struct {
	logger           *slog.Logger
	db               *gorm.DB
	ledgerRepo       domain.LedgerEntryRepository
	notificationRepo domain.NotificationRepository
}

// NewReconcileService creates a new reconcile service instance
func NewReconcileService(
	db *gorm.DB,
	ledgerRepo domain.LedgerEntryRepository,
	notificationRepo domain.NotificationRepository,
) *ReconcileService {
	return &ReconcileService{
		logger:           observability.GetLogger().With("component", "ReconcileService"),
		db:               db,
		ledgerRepo:       ledgerRepo,
		notificationRepo: notificationRepo,
	}
}

// ReconcileRequest represents a request to reconcile receivable account
type ReconcileRequest struct {
	// No fields needed - reconciliation is automatic
}

// ReconcileResponse represents the result of a reconciliation operation
type ReconcileResponse struct {
	Success               bool   `json:"success"`
	Message               string `json:"message"`
	WasFixed              bool   `json:"was_fixed"`
	ImbalanceAmount       int64  `json:"imbalance_amount"`
	AdjustmentEntryID     *uint  `json:"adjustment_entry_id,omitempty"`
	NewReceivableBalance  int64  `json:"new_receivable_balance"`
	WriteOffCreated       bool   `json:"write_off_created,omitempty"`
	WriteOffTransactionID *uint  `json:"write_off_transaction_id,omitempty"`
	WriteOffAmount        int64  `json:"write_off_amount,omitempty"`
}

// ReconcileReceivableAccount fixes receivable account imbalances within rounding tolerance
func (s *ReconcileService) ReconcileReceivableAccount(ctx context.Context, req *ReconcileRequest, userID uint) (*ReconcileResponse, error) {
	// Automatic reconciliation - no confirmation needed

	// First, check for write-off conditions
	writeOffTxn, err := s.CheckAndCreateWriteOff(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to check/create write-off transaction", "error", err)
		// Continue with reconciliation even if write-off fails
	}

	// Get current receivable balance (after potential write-off) using lightweight direct query
	receivableBalance, err := s.getReceivableBalanceLightweight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable balance: %w", err)
	}

	// Calculate imbalance (positive = more debits than credits)
	imbalance := receivableBalance

	// Check if imbalance is within tolerance
	if imbalance == 0 {
		response := &ReconcileResponse{
			Success:              true,
			Message:              "Receivable account is already balanced",
			WasFixed:             false,
			ImbalanceAmount:      0,
			NewReceivableBalance: 0,
		}

		// Include write-off info if created
		if writeOffTxn != nil {
			response.WriteOffCreated = true
			transactionID := writeOffTxn.ID
			response.WriteOffTransactionID = &transactionID
			response.WriteOffAmount = writeOffTxn.Amount
			response.Message = "Receivable account balanced with write-off transaction created"
		}

		return response, nil
	}

	// Only handle negative balances (business owes clients)
	// Positive balances (clients owe business) are normal and require no action
	if imbalance > 0 {
		// Positive receivable balance is normal business - no action needed
		return &ReconcileResponse{
			Success:              true,
			Message:              "Receivable account balance is normal (clients owe money)",
			WasFixed:             false,
			ImbalanceAmount:      imbalance,
			NewReceivableBalance: imbalance,
		}, nil
	}

	// At this point, imbalance is negative (business owes clients)
	absImbalance := -imbalance // Make it positive for comparison

	// Only handle small negative balances (rounding errors)
	// Large negative balances are also normal business - no action needed
	if absImbalance > constants.RoundingTolerance {
		// Large negative balance is normal business - no action needed
		return &ReconcileResponse{
			Success:              true,
			Message:              "Receivable account balance is normal (business owes clients)",
			WasFixed:             false,
			ImbalanceAmount:      imbalance,
			NewReceivableBalance: imbalance,
		}, nil
	}

	// Small negative balance - proceed with automatic adjustment (rounding errors)
	var adjustmentEntryID *uint

	// Create adjustment in database transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction context
		txCtx := &domain.TransactionContext{
			TX:              tx,
			IsTransactional: true,
		}
		newCtx := domain.WithTransactionContext(ctx, txCtx)

		// Create balancing ledger entries
		entries, err := s.createBalancingEntries(newCtx, imbalance, userID)
		if err != nil {
			return fmt.Errorf("failed to create balancing entries: %w", err)
		}

		// Create entries in database
		if err := s.ledgerRepo.CreateTransaction(newCtx, entries); err != nil {
			return fmt.Errorf("failed to create adjustment transaction: %w", err)
		}

		// Get the ID of the adjustment entry for response
		// For surplus: use receivable entry, for shortage: use cash entry
		var targetAccount string
		if imbalance > 0 {
			targetAccount = domain.AccountReceivable
		} else {
			targetAccount = domain.AccountCash
		}

		for _, entry := range entries {
			if entry.Account == targetAccount {
				adjustmentEntryID = &entry.ID
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to reconcile receivable account: %w", err)
	}

	// Get new balance after adjustment using lightweight direct query
	newBalance, err := s.getReceivableBalanceLightweight(ctx)
	if err != nil {
		s.logger.Warn("Failed to get new receivable balance", "error", err)
		newBalance = 0
	}

	s.logger.Info("Successfully reconciled receivable account",
		"original_imbalance", utils.FormatVND(imbalance),
		"new_balance", utils.FormatVND(newBalance),
		"adjustment_entry_id", adjustmentEntryID,
		"created_by", userID)

	response := &ReconcileResponse{
		Success:              true,
		Message:              "Receivable account reconciled successfully",
		WasFixed:             true,
		ImbalanceAmount:      imbalance,
		AdjustmentEntryID:    adjustmentEntryID,
		NewReceivableBalance: newBalance,
	}

	// Include write-off info if created (need to check since it was done at the beginning)
	if writeOffTxn != nil {
		response.WriteOffCreated = true
		transactionID := writeOffTxn.ID
		response.WriteOffTransactionID = &transactionID
		response.WriteOffAmount = writeOffTxn.Amount
		if response.Message == "Receivable account reconciled successfully" {
			response.Message = "Receivable account reconciled with write-off transaction created"
		}
	}

	return response, nil
}

// createBalancingEntries creates ledger entries to balance the receivable account
func (s *ReconcileService) createBalancingEntries(ctx context.Context, imbalance int64, userID uint) ([]*domain.LedgerEntry, error) {
	now := clock.Now()
	entries := make([]*domain.LedgerEntry, 0, 2)

	if imbalance > 0 {
		// Surplus: treat as income - increase cash and revenue
		// Debit Cash (increase cash)
		entries = append(entries, &domain.LedgerEntry{
			Date:      now,
			Account:   domain.AccountCash,
			Party:     "Hệ thống",
			Debit:     imbalance,
			Credit:    0,
			CreatedBy: userID,
		})

		// Debit Revenue (treat surplus as income)
		entries = append(entries, &domain.LedgerEntry{
			Date:      now,
			Account:   domain.AccountRevenue,
			Party:     "Hệ thống",
			Debit:     imbalance,
			Credit:    0,
			CreatedBy: userID,
		})

	} else {
		// Shortage: use our own cash to settle - reduce cash and receivable
		absImbalance := -imbalance

		// Credit Cash (reduce cash - we pay the difference)
		entries = append(entries, &domain.LedgerEntry{
			Date:      now,
			Account:   domain.AccountCash,
			Party:     "Hệ thống",
			Debit:     0,
			Credit:    absImbalance,
			CreatedBy: userID,
		})

		// Credit Receivable (reduce what client owes - writing off shortage)
		entries = append(entries, &domain.LedgerEntry{
			Date:      now,
			Account:   domain.AccountReceivable,
			Party:     "Hệ thống",
			Debit:     0,
			Credit:    absImbalance,
			CreatedBy: userID,
		})
	}

	// Validate all entries
	for _, entry := range entries {
		if err := entry.IsValid(); err != nil {
			return nil, fmt.Errorf("invalid balancing entry: %w", err)
		}
	}

	return entries, nil
}

// CheckAndCreateWriteOff creates a write-off transaction if receivable balance is < 1000 VND
func (s *ReconcileService) CheckAndCreateWriteOff(ctx context.Context, userID uint) (*domain.Transaction, error) {
	// Get current receivable balance using lightweight direct query
	receivableBalance, err := s.getReceivableBalanceLightweight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable balance: %w", err)
	}

	s.logger.Info("Checking receivable balance for write-off",
		"balance", utils.FormatVND(receivableBalance),
		"threshold", utils.FormatVND(constants.RoundingTolerance))

	// Only create write-off if balance is positive and less than threshold
	if receivableBalance <= 0 || receivableBalance >= constants.RoundingTolerance {
		return nil, nil // No write-off needed
	}

	// Create write-off transaction
	writeOffTxn := &domain.Transaction{
		Description:     fmt.Sprintf("Xóa nợ công nợ nhỏ %s", utils.FormatVND(receivableBalance)),
		TransactionType: domain.TransactionTypeWriteOff,
		Amount:          receivableBalance,
		Party:           "Hệ thống",
		Status:          domain.TransactionStatusSettled, // Write-offs are always settled
		CreatedBy:       userID,
	}

	// Create transaction and ledger entries using separate direct DB operations
	// Create transaction directly without using repository transaction context
	if err := s.db.WithContext(ctx).Create(writeOffTxn).Error; err != nil {
		return nil, fmt.Errorf("failed to create write-off transaction: %w", err)
	}

	// Generate ledger entries based on transaction type
	debitAccount, creditAccount := writeOffTxn.GetLedgerAccounts()
	now := clock.Now()

	// Create ledger entries directly
	transactionID := writeOffTxn.ID
	entries := []*domain.LedgerEntry{
		{
			TransactionID: &transactionID,
			Date:          now,
			Account:       debitAccount,
			Party:         writeOffTxn.Party,
			Debit:         writeOffTxn.Amount,
			Credit:        0,
			CreatedBy:     userID,
		},
		{
			TransactionID: &transactionID,
			Date:          now,
			Account:       creditAccount,
			Party:         writeOffTxn.Party,
			Debit:         0,
			Credit:        writeOffTxn.Amount,
			CreatedBy:     userID,
		},
	}

	// Create ledger entries directly without using the complex repository method
	if err := s.db.WithContext(ctx).Create(entries).Error; err != nil {
		return nil, fmt.Errorf("failed to create ledger entries for write-off: %w", err)
	}

	createdTxn := writeOffTxn

	s.logger.Info("Write-off transaction created successfully",
		"transactionID", createdTxn.ID,
		"amount", utils.FormatVND(receivableBalance),
		"createdBy", userID)

	return createdTxn, nil
}

// getReceivableBalanceLightweight calculates receivable balance using minimal SQL query
// This bypasses GORM complexities and potential locking issues
func (s *ReconcileService) getReceivableBalanceLightweight(ctx context.Context) (int64, error) {
	type BalanceResult struct {
		TotalDebit  int64 `gorm:"column:total_debit"`
		TotalCredit int64 `gorm:"column:total_credit"`
	}

	var result BalanceResult

	err := s.db.WithContext(ctx).
		Table("ledger_entries").
		Select("CAST(COALESCE(SUM(CASE WHEN debit > 0 THEN debit ELSE 0 END), 0) AS SIGNED) as total_debit, CAST(COALESCE(SUM(CASE WHEN credit > 0 THEN credit ELSE 0 END), 0) AS SIGNED) as total_credit").
		Where("account = ? AND deleted_at IS NULL", domain.AccountReceivable).
		Scan(&result).Error

	if err != nil {
		return 0, fmt.Errorf("lightweight balance query failed: %w", err)
	}

	// Calculate net balance: debit - credit for receivable accounts
	return result.TotalDebit - result.TotalCredit, nil
}
