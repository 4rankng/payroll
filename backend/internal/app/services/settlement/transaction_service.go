package settlement

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"api-server/internal/app/services/audit"
	"api-server/internal/app/services/ledger"
	"api-server/internal/constants"
	"api-server/internal/domain"
	infraports "api-server/internal/domain/ports/infrastructure"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionService struct {
	logger          *slog.Logger
	db              *gorm.DB
	TransactionRepo domain.TransactionRepository
	SettlementRepo  domain.SettlementRepository
	LedgerRepo      domain.LedgerEntryRepository
	LoanRepo        domain.LoanRepository
	UserRepo        domain.UserRepository
	TimesheetRepo   domain.TimesheetRepository
	cache           infraports.CachePort
	events          domain.EventBus
	auditBuilder    *audit.MessageBuilder
}

func NewTransactionService(
	db *gorm.DB,
	transactionRepo domain.TransactionRepository,
	settlementRepo domain.SettlementRepository,
	ledgerRepo domain.LedgerEntryRepository,
	loanRepo domain.LoanRepository,
	userRepo domain.UserRepository,
	timesheetRepo domain.TimesheetRepository,
	cache infraports.CachePort,
	events domain.EventBus,
) *TransactionService {
	return &TransactionService{
		logger:          observability.GetLogger(),
		db:              db,
		TransactionRepo: transactionRepo,
		SettlementRepo:  settlementRepo,
		LedgerRepo:      ledgerRepo,
		LoanRepo:        loanRepo,
		UserRepo:        userRepo,
		TimesheetRepo:   timesheetRepo,
		cache:           cache,
		events:          events,
		auditBuilder:    audit.NewMessageBuilder(userRepo),
	}
}

// CreateTransaction creates a user transaction and automatically generates double-entry ledger records.
// Returns the persisted transaction and the ledger entries (committed atomically with it).
func (s *TransactionService) CreateTransaction(ctx context.Context, txn *domain.Transaction) (*domain.Transaction, []*domain.LedgerEntry, error) {
	if err := txn.Validate(); err != nil {
		return nil, nil, err
	}

	var finalTxn *domain.Transaction
	var finalEntries []*domain.LedgerEntry

	existingTxCtx, ok := domain.GetTransactionFromContext(ctx)
	insideExistingTransaction := ok && existingTxCtx != nil && existingTxCtx.IsTransactional
	if insideExistingTransaction {
		entries, err := s.createTransactionInContextWithEntries(ctx, txn)
		if err != nil {
			return nil, nil, err
		}
		finalTxn = txn
		finalEntries = entries
	} else {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			txCtx := &domain.TransactionContext{TX: tx, IsTransactional: true}
			newCtx := domain.WithTransactionContext(ctx, txCtx)
			entries, err := s.createTransactionInContextWithEntries(newCtx, txn)
			if err != nil {
				return err
			}
			finalTxn = txn
			finalEntries = entries
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	entityName := encodeTransactionEntityName(txn.Amount, string(txn.TransactionType))
	auditMessage := s.auditBuilder.BuildMessageWithUser(ctx, txn.CreatedBy, domain.AuditActionCreate, domain.EntityTypeTransaction, entityName)
	event := domain.NewTransactionCreatedEventWithAudit(ctx, txn, auditMessage)
	publish := func() {
		if pubErr := s.events.Publish(ctx, event); pubErr != nil {
			s.logger.Warn("Failed to publish TransactionCreated event", "transactionID", txn.ID, "error", pubErr)
		}
		s.invalidateTransactionCache(ctx)
	}
	if insideExistingTransaction {
		domain.RegisterAfterCommit(ctx, publish)
	} else {
		publish()
	}

	return finalTxn, finalEntries, nil
}

// GetTransactionForUpdate locks a transaction inside the caller's database
// transaction. Wallet-bulk reversal reconciliation uses this to adjust only
// the receivable share of the employee whose transfer was returned.
func (s *TransactionService) GetTransactionForUpdate(ctx context.Context, id uint) (*domain.Transaction, error) {
	return s.TransactionRepo.GetByIDForUpdate(ctx, id)
}

// IncrementTransactionAmount atomically adjusts a transaction amount and
// invalidates transaction caches only after the surrounding transaction commits.
func (s *TransactionService) IncrementTransactionAmount(ctx context.Context, id uint, delta int64) error {
	if err := s.TransactionRepo.IncrementAmount(ctx, id, delta); err != nil {
		return err
	}
	domain.RegisterAfterCommit(ctx, func() { s.invalidateTransactionCache(ctx) })
	return nil
}

// createTransactionInContextWithEntries performs the actual transaction creation within a DB
// transaction context. Creates the transaction record AND its initial double-entry ledger pair
// atomically. Returns the created ledger entries.
func (s *TransactionService) createTransactionInContextWithEntries(ctx context.Context, txn *domain.Transaction) ([]*domain.LedgerEntry, error) {
	txnCode := uuid.NewString()
	txn.TransactionCode = &txnCode

	if err := s.TransactionRepo.Create(ctx, txn); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Inline ledger entry creation (was previously routed through LedgerWorker via outbox).
	entries, err := ledger.BuildTransactionLedgerEntries(txn)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction ledger entries: %w", err)
	}
	if err := s.LedgerRepo.CreateTransaction(ctx, entries); err != nil {
		return nil, fmt.Errorf("failed to create transaction ledger entries: %w", err)
	}

	return entries, nil
}

// DeleteTransaction soft deletes a transaction and logs audit
func (s *TransactionService) DeleteTransaction(ctx context.Context, id uint) error {
	// Get transaction before deletion for event
	txn, err := s.TransactionRepo.GetByID(ctx, id)
	if err != nil && !domain.IsNotFoundError(err) {
		s.logger.Error("Failed to fetch transaction before delete", "transactionID", id, "error", err)
	}

	if err := s.TransactionRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	// Publish TransactionDeleted event (async audit logging via event handler)
	if txn != nil {
		event := domain.NewTransactionDeletedEvent(ctx, txn)
		if err := s.events.Publish(ctx, event); err != nil {
			s.logger.Warn("Failed to publish TransactionDeleted event", "transactionID", id, "error", err)
		}
	}

	// Invalidate cache
	s.invalidateTransactionCache(ctx)
	return nil
}

// CreateSettlement creates a settlement for a transaction (supports partial settlements)
func (s *TransactionService) CreateSettlement(
	ctx context.Context,
	txnID uint,
	settlement *domain.Settlement,
) (*domain.Transaction, *domain.Settlement, []*domain.LedgerEntry, error) {
	var finalTxn *domain.Transaction
	var finalSettlement *domain.Settlement
	var finalLedgerEntries []*domain.LedgerEntry

	// Wrap all operations in a database transaction for atomicity
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction context for repositories
		txCtx := &domain.TransactionContext{
			TX:              tx,
			IsTransactional: true,
		}
		newCtx := domain.WithTransactionContext(ctx, txCtx)

		// 1. Get transaction and validate
		txn, err := s.TransactionRepo.GetByID(newCtx, txnID)
		if err != nil {
			return err
		}

		if txn.IsFullySettled() {
			return domain.NewValidationError(constants.MsgTransactionFullySettledVN)
		}

		// 2. Validate settlement amount
		if err := txn.CanSettle(settlement.Amount); err != nil {
			return err
		}

		// 3. Validate settlement
		settlement.TransactionID = txnID
		if err := settlement.Validate(); err != nil {
			return err
		}

		// 4. Generate settlement UUID for idempotency
		settlementUUID := uuid.NewString()
		settlement.SettlementUUID = &settlementUUID

		// 5. Create settlement record
		if err := s.SettlementRepo.Create(newCtx, settlement); err != nil {
			return fmt.Errorf("failed to create settlement: %w", err)
		}

		// 6. Update in-memory settled_amount instead of reloading transaction
		// Calculate new total from existing settlements + newly created settlement
		var currentSettled int64
		if txn.Settlements != nil {
			for _, s := range txn.Settlements {
				currentSettled += s.Amount
			}
		}
		txn.SettledAmount = currentSettled + settlement.Amount
		txn.NormalizeSettledAmount(txn.SettledAmount > 0)

		// 7. Update transaction status based on recalculated settled_amount
		txn.UpdateStatus()
		if err := s.TransactionRepo.Update(newCtx, txn); err != nil {
			return fmt.Errorf("failed to update transaction: %w", err)
		}

		// 8. If this settlement is for a loan interest transaction, update the loan's total interest paid
		if txn.LoanID != nil && txn.TransactionType == domain.TransactionTypeExpense {
			loan, err := s.LoanRepo.GetByID(newCtx, *txn.LoanID)
			if err != nil {
				s.logger.Error("Failed to get loan for interest update", "loanID", *txn.LoanID, "error", err)
				return fmt.Errorf("failed to get loan for interest update: %w", err)
			}
			if loan == nil {
				return fmt.Errorf("loan %d not found", *txn.LoanID)
			}

			loan.TotalInterestPaid += settlement.Amount
			if err := s.LoanRepo.Update(newCtx, loan); err != nil {
				s.logger.Error("Failed to update loan interest paid", "loanID", loan.ID, "amount", utils.FormatVND(settlement.Amount), "error", err)
				return fmt.Errorf("failed to update loan interest paid: %w", err)
			}
			s.logger.Info("Updated loan interest paid", "loanID", loan.ID, "added", utils.FormatVND(settlement.Amount), "total", utils.FormatVND(int64(loan.TotalInterestPaid)))
		}

		// 9. Create settlement's double-entry ledger pair INLINE in the same DB transaction.
		// Settlement + transaction status + ledger entries are now one atomic commit.
		ledgerEntries, err := ledger.BuildSettlementLedgerEntries(txn, settlement)
		if err != nil {
			s.logger.Error("Failed to build settlement ledger entries",
				"settlementID", settlement.ID, "transactionID", txn.ID, "error", err)
			return fmt.Errorf("failed to build settlement ledger entries: %w", err)
		}
		if err := s.LedgerRepo.CreateTransaction(newCtx, ledgerEntries); err != nil {
			s.logger.Error("Failed to persist settlement ledger entries",
				"settlementID", settlement.ID, "transactionID", txn.ID, "error", err)
			return fmt.Errorf("failed to create settlement ledger entries: %w", err)
		}
		s.logger.Info("Settlement ledger entries created inline",
			"settlementID", settlement.ID, "transactionID", txn.ID, "entries", len(ledgerEntries))

		// Build audit message — still needed for the TransactionSettled event below.
		txnCode := ""
		if txn.TransactionCode != nil {
			txnCode = *txn.TransactionCode
		} else {
			txnCode = fmt.Sprintf("#%d", txn.ID)
		}
		settlementEntityName := fmt.Sprintf("tất toán %s cho giao dịch %s", utils.FormatVND(settlement.Amount), txnCode)
		_ = s.auditBuilder.BuildMessageWithUser(newCtx, settlement.CreatedBy, domain.AuditActionCreate, domain.EntityTypeTransaction, settlementEntityName)

		// 10. If the transaction is now fully settled AND it's a revenue transaction, mark all
		// its timesheets revenue_paid=1 inline (was previously: emit TransactionSettled to outbox
		// → SettlementEventHandler.handleTransactionSettled does the same bulk update).
		if txn.Status == domain.TransactionStatusSettled && txn.TransactionType == domain.TransactionTypeRevenue {
			var timesheets []*domain.Timesheet
			if err := tx.WithContext(newCtx).
				Where("transaction_id = ?", txn.ID).
				Select("id").
				Find(&timesheets).Error; err != nil {
				return fmt.Errorf("failed to load timesheets for settled txn %d: %w", txn.ID, err)
			}
			if len(timesheets) > 0 {
				ids := make([]uint, 0, len(timesheets))
				for _, t := range timesheets {
					ids = append(ids, t.ID)
				}
				if err := s.TimesheetRepo.BulkUpdateRevenuePaid(newCtx, ids); err != nil {
					return fmt.Errorf("failed to mark timesheets as revenue paid for txn %d: %w", txn.ID, err)
				}
				s.logger.Info("Marked all timesheets revenue_paid for fully-settled txn",
					"transactionID", txn.ID, "timesheet_count", len(ids))
			}
		}

		// Store results to return after transaction commits
		finalTxn = txn
		finalSettlement = settlement
		finalLedgerEntries = ledgerEntries

		return nil
	})

	if err != nil {
		return nil, nil, nil, err
	}

	settledEvent := domain.NewTransactionSettledEvent(ctx, finalTxn, finalSettlement)
	if pubErr := s.events.Publish(ctx, settledEvent); pubErr != nil {
		s.logger.Warn("Failed to publish TransactionSettled audit event",
			"transactionID", finalTxn.ID, "settlementID", finalSettlement.ID, "error", pubErr)
	}

	// Invalidate cache
	s.invalidateTransactionCache(ctx)

	var settlementUUID string
	if finalSettlement.SettlementUUID != nil {
		settlementUUID = *finalSettlement.SettlementUUID
	}
	s.logger.Info("Settlement created successfully with inline ledger entries",
		"transactionID", txnID,
		"settlementID", finalSettlement.ID,
		"settlementUUID", settlementUUID,
		"amount", utils.FormatVND(finalSettlement.Amount),
		"remaining", utils.FormatVND(finalTxn.GetRemainingAmount()),
		"ledger_entries", len(finalLedgerEntries))

	return finalTxn, finalSettlement, finalLedgerEntries, nil
}

// getCreditParty returns the appropriate party name for credit entry
func (s *TransactionService) getCreditParty(txn *domain.Transaction) string {
	if txn.Status == domain.TransactionStatusSettled {
		return constants.LedgerPartyBank
	}
	return txn.Party
}

// GetTransactionSettlements retrieves all settlements for a transaction
func (s *TransactionService) GetTransactionSettlements(ctx context.Context, txnID uint) ([]*domain.Settlement, error) {
	return s.SettlementRepo.GetByTransactionID(ctx, txnID)
}

// GetTransaction retrieves a transaction by ID with caching
func (s *TransactionService) GetTransaction(ctx context.Context, id uint) (*domain.Transaction, error) {
	// Generate cache key: transactions:detail:{id}
	cacheKey := fmt.Sprintf("transactions:detail:%d", id)

	// Try to get from cache first
	var cachedTxn domain.Transaction
	if err := s.cache.Get(ctx, cacheKey, &cachedTxn); err == nil {
		s.logger.Info("Transaction retrieved from cache", "transactionID", id)
		return &cachedTxn, nil
	}

	// Get from database
	txn, err := s.TransactionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Store in cache
	if cacheErr := s.cache.Set(ctx, cacheKey, txn, constants.TransactionDetailCacheTTL); cacheErr != nil {
		s.logger.Error("Failed to cache transaction", "transactionID", id, "error", cacheErr)
	}

	return txn, nil
}

// ListTransactions lists transactions with filters and caching
func (s *TransactionService) ListTransactions(ctx context.Context, filters domain.TransactionFilters) ([]*domain.Transaction, error) {
	// Build cache key with strings.Builder to avoid repeated allocations
	var b strings.Builder
	b.WriteString("transactions:list:limit:")
	b.WriteString(strconv.Itoa(filters.Limit))
	b.WriteString(":offset:")
	b.WriteString(strconv.Itoa(filters.Offset))
	b.WriteString(":sortBy:")
	b.WriteString(filters.SortBy)
	b.WriteString(":sortOrder:")
	b.WriteString(filters.SortOrder)
	if filters.TransactionType != nil {
		b.WriteString(":type:")
		b.WriteString(*filters.TransactionType)
	}
	if filters.Status != nil {
		b.WriteString(":status:")
		b.WriteString(*filters.Status)
	}
	if filters.Party != nil {
		b.WriteString(":party:")
		b.WriteString(*filters.Party)
	}
	if filters.CreatedBy != nil {
		b.WriteString(":createdBy:")
		b.WriteString(strconv.FormatUint(uint64(*filters.CreatedBy), 10))
	}
	if filters.FromDate != nil {
		b.WriteString(":fromDate:")
		b.WriteString(filters.FromDate.Format("2006-01-02"))
	}
	if filters.ToDate != nil {
		b.WriteString(":toDate:")
		b.WriteString(filters.ToDate.Format("2006-01-02"))
	}
	cacheKey := b.String()

	var cachedTransactions []*domain.Transaction
	if err := s.cache.Get(ctx, cacheKey, &cachedTransactions); err == nil {
		s.logger.Info("Transactions retrieved from cache", "cacheKey", cacheKey, "count", len(cachedTransactions))
		return cachedTransactions, nil
	}

	transactions, err := s.TransactionRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	if cacheErr := s.cache.Set(ctx, cacheKey, transactions, constants.TransactionListCacheTTL); cacheErr != nil {
		s.logger.Error("Failed to cache transaction list", "error", cacheErr)
	}

	return transactions, nil
}

// CountTransactions counts transactions with filters
func (s *TransactionService) CountTransactions(ctx context.Context, filters domain.TransactionFilters) (int64, error) {
	return s.TransactionRepo.Count(ctx, filters)
}

// GetTransactionsByParty retrieves transactions for a specific party
func (s *TransactionService) GetTransactionsByParty(ctx context.Context, party string) ([]*domain.Transaction, error) {
	return s.TransactionRepo.GetByParty(ctx, party)
}

// ReverseTransaction reverses a transaction by creating opposite ledger entries
func (s *TransactionService) ReverseTransaction(
	ctx context.Context,
	txnID uint,
	reason string,
	userID uint,
) (*domain.Transaction, *domain.Transaction, []*domain.LedgerEntry, error) {
	// 1. Get original transaction
	originalTxn, err := s.TransactionRepo.GetByID(ctx, txnID)
	if err != nil {
		return nil, nil, nil, err
	}

	// 2. Validate that transaction can be reversed
	if err := originalTxn.CanReverse(); err != nil {
		return nil, nil, nil, err
	}

	// 3. Build reversal description
	reversalDescription := fmt.Sprintf("Đảo ngược: %s", originalTxn.Description)
	if reason != "" {
		reversalDescription = fmt.Sprintf("%s - %s", reversalDescription, reason)
	}

	// 4. Create reversal transaction with same attributes
	txnCode := uuid.NewString()
	reversalTxn := &domain.Transaction{
		Description:     reversalDescription,
		TransactionCode: &txnCode,
		TransactionType: originalTxn.TransactionType,
		Amount:          originalTxn.Amount,
		Party:           originalTxn.Party,
		Status:          domain.TransactionStatusSettled, // Reversals are always settled
		AssetID:         originalTxn.AssetID,
		CreatedBy:       userID,
	}

	// Validate reversal transaction
	if err := reversalTxn.Validate(); err != nil {
		return nil, nil, nil, err
	}

	// 5. Create reversal transaction record
	if err := s.TransactionRepo.Create(ctx, reversalTxn); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create reversal transaction: %w", err)
	}

	// 6. Generate opposite ledger entries
	ledgerEntries, err := s.generateReversalLedgerEntries(ctx, originalTxn, reversalTxn)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate reversal ledger entries: %w", err)
	}

	// 7. Create ledger entries
	if err := s.LedgerRepo.CreateTransaction(ctx, ledgerEntries); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create reversal ledger entries: %w", err)
	}

	// 8. Update original transaction to mark it as reversed
	originalTxn.ReversedTransactionID = &reversalTxn.ID
	if err := s.TransactionRepo.Update(ctx, originalTxn); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to update original transaction: %w", err)
	}

	// 9. Publish TransactionCreated event for reversal (async audit logging via event handler)
	// Build audit message with user name for reversal transaction
	entityName := encodeTransactionEntityName(reversalTxn.Amount, string(reversalTxn.TransactionType))
	auditMessage := s.auditBuilder.BuildMessageWithUser(ctx, reversalTxn.CreatedBy, domain.AuditActionCreate, domain.EntityTypeTransaction, entityName)

	event := domain.NewTransactionCreatedEventWithAudit(ctx, reversalTxn, auditMessage)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish TransactionCreated event for reversal",
			"originalID", originalTxn.ID,
			"reversalID", reversalTxn.ID,
			"error", err)
	}

	// 10. Invalidate cache
	s.invalidateTransactionCache(ctx)

	s.logger.Info("Transaction reversed successfully",
		"originalID", originalTxn.ID,
		"reversalID", reversalTxn.ID,
		"amount", reversalTxn.Amount)

	return originalTxn, reversalTxn, ledgerEntries, nil
}

// generateReversalLedgerEntries generates opposite ledger entries to cancel out original transaction
func (s *TransactionService) generateReversalLedgerEntries(ctx context.Context, originalTxn *domain.Transaction, reversalTxn *domain.Transaction) ([]*domain.LedgerEntry, error) {
	// Get the accounts that were used in the original transaction
	debitAccount, creditAccount := originalTxn.GetLedgerAccounts()

	// For reversal, we swap debit and credit
	// Original: Debit Expense, Credit Cash
	// Reversal: Debit Cash, Credit Expense

	// Create debit entry (opposite of original credit)
	debitEntry := &domain.LedgerEntry{
		Date:          clock.Now(),
		Account:       creditAccount, // Swap: use original credit account
		Party:         s.getCreditParty(originalTxn),
		Debit:         reversalTxn.Amount,
		Credit:        0,
		AssetID:       reversalTxn.AssetID,
		TransactionID: &reversalTxn.ID,
		CreatedBy:     reversalTxn.CreatedBy,
	}

	// Create credit entry (opposite of original debit)
	creditEntry := &domain.LedgerEntry{
		Date:          clock.Now(),
		Account:       debitAccount, // Swap: use original debit account
		Party:         originalTxn.Party,
		Debit:         0,
		Credit:        reversalTxn.Amount,
		AssetID:       reversalTxn.AssetID,
		TransactionID: &reversalTxn.ID,
		CreatedBy:     reversalTxn.CreatedBy,
	}

	// Validate entries
	if err := debitEntry.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid reversal debit entry: %w", err)
	}

	if err := creditEntry.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid reversal credit entry: %w", err)
	}

	return []*domain.LedgerEntry{debitEntry, creditEntry}, nil
}

// invalidateTransactionCache invalidates all transaction-related cache entries
func (s *TransactionService) invalidateTransactionCache(ctx context.Context) {
	// Clear all transaction list cache entries
	if err := s.cache.InvalidatePattern(ctx, "transactions:list:*"); err != nil {
		s.logger.Error("Failed to clear transaction list cache", "error", err)
	}

	// Clear all transaction detail cache entries
	if err := s.cache.InvalidatePattern(ctx, "transactions:detail:*"); err != nil {
		s.logger.Error("Failed to clear transaction detail cache", "error", err)
	}

	// Clear pending transactions cache
	if err := s.cache.InvalidatePattern(ctx, "transactions:pending:*"); err != nil {
		s.logger.Error("Failed to clear pending transactions cache", "error", err)
	}
}

// UpdateTransactionEvidence updates only the evidence fields (URL, AssetID) for a transaction
// It does not change any financial fields or generate any ledger entries
func (s *TransactionService) UpdateTransactionEvidence(ctx context.Context, id uint, url *string, assetID *uint) (*domain.Transaction, error) {
	// Load existing transaction
	txn, err := s.TransactionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	originalTxn := *txn

	// Apply updates only to evidence fields
	if url != nil {
		txn.URL = *url
	}
	if assetID != nil {
		txn.AssetID = assetID
	}

	if err := s.TransactionRepo.Update(ctx, txn); err != nil {
		return nil, fmt.Errorf("failed to update transaction evidence: %w", err)
	}

	// Publish TransactionUpdated event (async audit logging via event handler)
	event := domain.NewTransactionUpdatedEvent(ctx, txn, &originalTxn)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish TransactionUpdated event", "transactionID", txn.ID, "error", err)
	}

	// Invalidate caches for transactions
	s.invalidateTransactionCache(ctx)

	return txn, nil
}

// encodeTransactionEntityName encodes transaction entity information for audit messages
// so that the final sentence reads naturally in Vietnamese, e.g.:
// "Frank Nguyen tạo doanh thu 45.000 ₫".
func encodeTransactionEntityName(amount int64, txType string) string {
	// Format amount with thousand separators (dots) and space before ₫
	amountStr := utils.FormatVND(amount)

	// Return lowercase Vietnamese transaction type + formatted amount
	switch txType {
	case string(domain.TransactionTypeRevenue):
		return fmt.Sprintf("doanh thu %s", amountStr)
	case string(domain.TransactionTypeExpense):
		return fmt.Sprintf("chi phí %s", amountStr)
	case string(domain.TransactionTypeCapital):
		return fmt.Sprintf("vốn %s", amountStr)
	case string(domain.TransactionTypeLoanDisbursement):
		return fmt.Sprintf("giải ngân khoản vay %s", amountStr)
	case string(domain.TransactionTypeLoanRepayment):
		return fmt.Sprintf("trả nợ gốc %s", amountStr)
	default:
		// Fallback: just return the amount for unknown types
		return amountStr
	}
}
