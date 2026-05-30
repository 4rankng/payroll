package bulktransfer

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// TransactionManager handles atomic transaction and ledger operations
type TransactionManager struct {
	ledgerService      LedgerService
	transactionService TransactionService
	settingsConfig     SettingsConfigService
}

// NewTransactionManager creates a new TransactionManager instance
func NewTransactionManager(
	ledgerService LedgerService,
	transactionService TransactionService,
	settingsConfig SettingsConfigService,
) *TransactionManager {
	return &TransactionManager{
		ledgerService:      ledgerService,
		transactionService: transactionService,
		settingsConfig:     settingsConfig,
	}
}

// CreateTransactionAndLedger creates a transaction and ledger entries atomically
func (tm *TransactionManager) CreateTransactionAndLedger(
	ctx context.Context,
	totalTransferAmount float64,
	asset *domain.Asset,
	filename string,
	createdBy uint,
) (*domain.Transaction, error) {
	logger := observability.GetLogger()

	if totalTransferAmount <= 0 {
		return nil, nil
	}

	logger.Info("Creating transaction and ledger entries for bulk transfer",
		"totalAmount", totalTransferAmount)

	// Get partner company
	partnerCompany := tm.settingsConfig.GetPartnerCompany(ctx)

	// Build ledger plan
	plan := BuildLedgerPlan(ctx, tm.settingsConfig, totalTransferAmount, partnerCompany, filename)

	// Create transaction
	transaction := &domain.Transaction{
		TransactionType: domain.TransactionTypeRevenue,
		Amount:          plan.ReceivableAmount,
		Party:           partnerCompany,
		Status:          domain.TransactionStatusPending,
		URL:             "",
		AssetID:         &asset.ID,
		CreatedBy:       createdBy,
	}

	_, _, err := tm.transactionService.CreateTransaction(ctx, transaction)
	if err != nil {
		logger.Error("Failed to create transaction record", "error", err)
		return nil, fmt.Errorf("failed to create transaction record: %w", err)
	}

	logger.Info("Successfully created transaction record for bulk transfer",
		"transactionID", transaction.ID,
		"amount", plan.ReceivableAmount)

	// Create ledger entries
	now := clock.Now()
	entries := plan.Entries(now, createdBy, partnerCompany)
	if _, err := tm.ledgerService.CreateEntries(ctx, entries, createdBy); err != nil {
		logger.Error("Failed to create ledger entries", "error", err)
		// Rollback of the created transaction is not performed here.
		// The error is returned so callers can decide how to handle partial state.
		return nil, fmt.Errorf("failed to create ledger entries: %w", err)
	}

	netRevenue := plan.ReceivableAmount - plan.CashOutAmount
	logger.Info("Successfully created ledger entries for bulk transfer",
		"cashOut", plan.CashOutAmount,
		"receivableFromClient", plan.ReceivableAmount,
		"netRevenue", netRevenue)

	return transaction, nil
}
