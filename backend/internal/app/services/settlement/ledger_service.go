package settlement

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"api-server/internal/app/accounting"
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	infraports "api-server/internal/domain/ports/infrastructure"
	"api-server/internal/infra/observability"
)

type LedgerService struct {
	logger          *slog.Logger
	LedgerRepo      domain.LedgerEntryRepository
	AccountingRules *domain.AccountingRules
	events          domain.EventBus
	cache           infraports.CachePort
}

func NewLedgerService(ledgerRepo domain.LedgerEntryRepository, events domain.EventBus, cache infraports.CachePort) *LedgerService {
	return &LedgerService{
		logger:          observability.GetLogger(),
		LedgerRepo:      ledgerRepo,
		AccountingRules: domain.NewAccountingRules(),
		events:          events,
		cache:           cache,
	}
}

func (s *LedgerService) CreateEntry(ctx context.Context, entry *domain.LedgerEntry, createdBy uint) (*domain.LedgerEntry, error) {
	// Validate the entry
	if err := entry.IsValid(); err != nil {
		return nil, err
	}
	entry.CreatedBy = createdBy

	// Check for duplicates
	existingEntry, err := s.LedgerRepo.CheckDuplicate(ctx, entry)
	if err != nil && !domain.IsNotFoundError(err) {
		return nil, fmt.Errorf("failed to check duplicate: %w", err)
	}
	if existingEntry != nil {
		// Found duplicate entry
		if entry.AssetID != nil {
			return nil, domain.NewConflictError(constants.MsgLedgerDuplicateAssetVN)
		}
		return nil, domain.NewConflictError(constants.MsgLedgerDuplicateEntryVN)
	}

	if err := s.LedgerRepo.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to create ledger entry: %w", err)
	}

	// Publish LedgerEntryCreated event
	event := domain.NewLedgerEntryCreatedEvent(ctx, entry)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish LedgerEntryCreated event", "entryID", entry.ID, "error", err)
	}

	return entry, nil
}

func (s *LedgerService) CreateEntries(ctx context.Context, entries []*domain.LedgerEntry, createdBy uint) ([]*domain.LedgerEntry, error) {
	if len(entries) == 0 {
		return nil, domain.NewValidationError(constants.MsgAtLeastOneEntryRequiredVN)
	}

	// Validate each entry
	for i, entry := range entries {
		if err := entry.IsValid(); err != nil {
			return nil, fmt.Errorf("validation failed for entry %d: %w", i+1, err)
		}
		entry.CreatedBy = createdBy
	}

	// Batch duplicate check — O(1) queries instead of O(N)
	duplicates, err := s.LedgerRepo.BatchCheckDuplicates(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to batch check duplicates: %w", err)
	}
	if len(duplicates) > 0 {
		// Report the first duplicate found
		for i, dup := range duplicates {
			entry := entries[i]
			_ = dup
			if entry.AssetID != nil {
				return nil, domain.NewConflictError(constants.MsgLedgerDuplicateAssetVN)
			}
			return nil, domain.NewConflictError(constants.MsgLedgerDuplicateEntryVN)
		}
	}

	if err := s.LedgerRepo.CreateTransaction(ctx, entries); err != nil {
		return nil, fmt.Errorf("failed to create ledger entries: %w", err)
	}

	// Publish LedgerEntryCreated events for batch creation
	for _, entry := range entries {
		event := domain.NewLedgerEntryCreatedEvent(ctx, entry)
		if err := s.events.Publish(ctx, event); err != nil {
			s.logger.Warn("Failed to publish LedgerEntryCreated event", "entryID", entry.ID, "error", err)
		}
	}

	return entries, nil
}

func (s *LedgerService) GetEntry(ctx context.Context, id uint) (*domain.LedgerEntry, error) {
	return s.LedgerRepo.GetByID(ctx, id)
}

func (s *LedgerService) ListEntries(ctx context.Context, filters domain.LedgerFilters) ([]*domain.LedgerEntry, error) {
	return s.LedgerRepo.List(ctx, filters)
}

func (s *LedgerService) CountEntries(ctx context.Context, filters domain.LedgerFilters) (int64, error) {
	return s.LedgerRepo.Count(ctx, filters)
}

func (s *LedgerService) GetEntriesByAccount(ctx context.Context, account domain.LedgerAccount) ([]*domain.LedgerEntry, error) {
	return s.LedgerRepo.GetByAccount(ctx, account)
}

func (s *LedgerService) GetEntriesByDateRange(ctx context.Context, start, end time.Time) ([]*domain.LedgerEntry, error) {
	return s.LedgerRepo.GetByDateRange(ctx, start, end)
}

func (s *LedgerService) GetBalance(ctx context.Context) (int64, error) {
	return s.LedgerRepo.GetBalance(ctx)
}

func (s *LedgerService) GetAccountBalance(ctx context.Context, account domain.LedgerAccount) (int64, error) {
	// Validate account type
	if !domain.IsValidAccountType(account) {
		return 0, domain.NewValidationError(fmt.Sprintf("loại tài khoản không hợp lệ: %s", account))
	}

	return s.LedgerRepo.GetBalanceByAccount(ctx, account)
}

// GetAccountTotalInRange returns SUM(debit − credit) for the given account over
// the inclusive date range [from, to]. Passthrough to the repository; used by
// the settlement simulation's reconciliation against the receivable account.
func (s *LedgerService) GetAccountTotalInRange(ctx context.Context, account domain.LedgerAccount, from, to time.Time) (int64, error) {
	if !domain.IsValidAccountType(account) {
		return 0, domain.NewValidationError(fmt.Sprintf("loại tài khoản không hợp lệ: %s", account))
	}
	return s.LedgerRepo.GetAccountTotalInRange(ctx, account, from, to)
}

func (s *LedgerService) GetCashFlowSummary(ctx context.Context, start, end time.Time) (*domain.CashFlowSummary, error) {
	// Generate cache key: dashboard:cash_flow_summary:{start}:{end}
	cacheKey := fmt.Sprintf("dashboard:cash_flow_summary:%s:%s",
		start.Format("2006-01-02"), end.Format("2006-01-02"))

	// Try to get from cache
	var summary domain.CashFlowSummary
	err := s.cache.Get(ctx, cacheKey, &summary)
	if err == nil {
		// Cache hit
		return &summary, nil
	}

	// Cache miss - fetch from database
	result, err := s.LedgerRepo.GetCashFlowSummary(ctx, start, end)
	if err != nil {
		return nil, err
	}

	// Cache the result for 5 minutes
	_ = s.cache.Set(ctx, cacheKey, result, constants.LedgerSummaryCacheTTL)

	return result, nil
}

func (s *LedgerService) RecalculateAllBalances(ctx context.Context, adminUserID uint) error {
	if err := s.LedgerRepo.RecalculateAllBalances(ctx); err != nil {
		return fmt.Errorf("failed to recalculate ledger balances: %w", err)
	}

	return nil
}

func (s *LedgerService) ReverseEntry(ctx context.Context, entryID uint, reason string, createdBy uint) (*domain.LedgerEntry, error) {
	// Get the original entry
	originalEntry, err := s.LedgerRepo.GetByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to find original entry: %w", err)
	}

	reversalEntry := &domain.LedgerEntry{
		Date:      originalEntry.Date,
		Account:   originalEntry.Account,
		Party:     originalEntry.Party,
		Debit:     originalEntry.Credit, // Swap amounts
		Credit:    originalEntry.Debit,  // Swap amounts
		CreatedBy: createdBy,
	}

	// Validate the reversal entry
	if err := reversalEntry.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid reversal entry: %w", err)
	}

	// Create the reversal entry
	if err := s.LedgerRepo.Create(ctx, reversalEntry); err != nil {
		return nil, fmt.Errorf("failed to create reversal entry: %w", err)
	}

	return reversalEntry, nil
}

// CreateBalancedTransaction creates a double-entry transaction with automatic balancing
func (s *LedgerService) CreateBalancedTransaction(ctx context.Context, group *domain.TransactionGroup) error {
	// Validate the transaction group follows double-entry principles
	if err := s.AccountingRules.ValidateTransactionGroup(group); err != nil {
		return fmt.Errorf("transaction validation failed: %w", err)
	}

	// Create the transaction using the repository's validated method
	if err := s.LedgerRepo.CreateTransaction(ctx, group.Entries); err != nil {
		return fmt.Errorf("failed to create balanced transaction: %w", err)
	}

	// Publish LedgerEntryCreated events for balanced transaction
	for _, entry := range group.Entries {
		event := domain.NewLedgerEntryCreatedEvent(ctx, entry)
		if err := s.events.Publish(ctx, event); err != nil {
			s.logger.Warn("Failed to publish LedgerEntryCreated event", "entryID", entry.ID, "error", err)
		}
	}

	return nil
}

// DeleteEntriesByTransactionID removes all ledger entries tied to a transaction
func (s *LedgerService) DeleteEntriesByTransactionID(ctx context.Context, transactionID uint) error {
	return s.LedgerRepo.DeleteByTransactionID(ctx, transactionID)
}

// CreateSalaryPayment creates a balanced salary payment transaction
func (s *LedgerService) CreateSalaryPayment(ctx context.Context, date time.Time, employeeName string, amount float64, createdBy uint) error {
	amountMoney := accounting.NewMoney(amount)

	group, err := s.AccountingRules.CreateBalancedSalaryTransaction(
		date.Format("2006-01-02"),
		employeeName,
		amountMoney,
		createdBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create salary transaction: %w", err)
	}

	// Set date for all entries
	for _, entry := range group.Entries {
		entry.Date = date
	}

	return s.CreateBalancedTransaction(ctx, group)
}

// CreateRevenuePayment creates a balanced revenue payment transaction
func (s *LedgerService) CreateRevenuePayment(ctx context.Context, date time.Time, clientName string, amount float64, createdBy uint) error {
	amountMoney := accounting.NewMoney(amount)

	group, err := s.AccountingRules.CreateBalancedRevenueTransaction(
		date.Format("2006-01-02"),
		clientName,
		amountMoney,
		createdBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create revenue transaction: %w", err)
	}

	// Set date for all entries
	for _, entry := range group.Entries {
		entry.Date = date
	}

	return s.CreateBalancedTransaction(ctx, group)
}

// CreateReversalEntry creates a reversal entry using accounting rules
func (s *LedgerService) CreateReversalEntry(ctx context.Context, entryID uint, reason string, createdBy uint) (*domain.LedgerEntry, error) {
	// Get the original entry
	originalEntry, err := s.LedgerRepo.GetByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to find original entry: %w", err)
	}

	// Create reversal using accounting rules
	reversalEntry, err := s.AccountingRules.CreateReversalTransaction(originalEntry, reason, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create reversal transaction: %w", err)
	}

	// Create the reversal entry
	if err := s.LedgerRepo.Create(ctx, reversalEntry); err != nil {
		return nil, fmt.Errorf("failed to create reversal entry: %w", err)
	}

	// Publish LedgerEntryCreated event for reversal
	event := domain.NewLedgerEntryCreatedEvent(ctx, reversalEntry)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish LedgerEntryCreated event", "entryID", reversalEntry.ID, "error", err)
	}

	return reversalEntry, nil
}

// GetBalanceWithMoney returns the current balance as a Money type for precise calculations
func (s *LedgerService) GetBalanceWithMoney(ctx context.Context) (*accounting.Money, error) {
	balance, err := s.LedgerRepo.GetBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return accounting.NewMoney(float64(balance)), nil
}

// ValidateTransactionGroup validates a group of entries follows double-entry rules
func (s *LedgerService) ValidateTransactionGroup(group *domain.TransactionGroup) error {
	return s.AccountingRules.ValidateTransactionGroup(group)
}

// CreatePartnerPayment creates a balanced partner payment transaction
func (s *LedgerService) CreatePartnerPayment(ctx context.Context, date time.Time, partnerName string, amount float64, createdBy uint) error {
	debitEntry := &domain.LedgerEntry{
		Date:      date,
		Account:   "expense",
		Party:     partnerName,
		Debit:     int64(math.Round(amount)),
		Credit:    0,
		CreatedBy: createdBy,
	}

	creditEntry := &domain.LedgerEntry{
		Date:      date,
		Account:   "capital",
		Party:     constants.LedgerPartyBank,
		Debit:     0,
		Credit:    int64(math.Round(amount)),
		CreatedBy: createdBy,
	}

	group := &domain.TransactionGroup{
		Entries: []*domain.LedgerEntry{debitEntry, creditEntry},
	}

	return s.CreateBalancedTransaction(ctx, group)
}

// GetLedgerSummary generates comprehensive ledger summary for specified date range
func (s *LedgerService) GetLedgerSummary(ctx context.Context, fromDate, toDate time.Time) (*dto.LedgerSummaryResponse, error) {
	// Validate date range
	if fromDate.After(toDate) {
		return nil, domain.NewValidationError(constants.MsgFromDateAfterToDateVN)
	}

	// Generate cache key: dashboard:ledger_summary:{fromDate}:{toDate}
	cacheKey := fmt.Sprintf("dashboard:ledger_summary:%s:%s",
		fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))

	// Try to get from cache
	var response dto.LedgerSummaryResponse
	err := s.cache.Get(ctx, cacheKey, &response)
	if err == nil {
		// Cache hit
		return &response, nil
	}

	// Cache miss - fetch from database
	summaryData, err := s.LedgerRepo.GetLedgerSummary(ctx, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger summary: %w", err)
	}

	// Build response with integer VND values
	result := &dto.LedgerSummaryResponse{
		Period: dto.PeriodInfo{
			From: fromDate.Format("2006-01-02"),
			To:   toDate.Format("2006-01-02"),
		},
		Totals: dto.TotalInfo{
			OpeningBalance: summaryData.OpeningBalance,
			ClosingBalance: summaryData.ClosingBalance,
			NetCashflow:    summaryData.ClosingBalance - summaryData.OpeningBalance,
		},
		ByAccount: make(map[string]dto.AccountSummary),
		ByOwners:  make([]dto.OwnerContribution, 0, len(summaryData.OwnerContributions)),
	}

	// Account names are already in the correct API format
	accountMapping := map[string]string{
		domain.AccountCash:       "cash",
		domain.AccountReceivable: "receivable",
		domain.AccountPayable:    "payable",
		domain.AccountRevenue:    "revenue",
		domain.AccountExpense:    "expense",
		domain.AccountEquity:     "equity",
		domain.AccountLoan:       "loan",
	}

	// Ensure required accounts are included in the response
	requiredAccounts := []string{"cash", "receivable", "payable", "loan", "revenue", "expense", "equity"}

	for _, apiAccount := range requiredAccounts {
		result.ByAccount[apiAccount] = dto.AccountSummary{
			Debit:     0,
			Credit:    0,
			NetAmount: 0,
		}
	}

	// Populate actual data from database with rounded values
	for domainAccount, summary := range summaryData.AccountSummaries {
		if apiAccount, exists := accountMapping[domainAccount]; exists {
			// Compute net amount per account as integer VND
			var net int64
			switch domainAccount {
			case domain.AccountCash, domain.AccountReceivable, domain.AccountExpense:
				net = summary.TotalDebit - summary.TotalCredit
			case domain.AccountRevenue, domain.AccountPayable, domain.AccountEquity, domain.AccountLoan:
				net = summary.TotalCredit - summary.TotalDebit
			default:
				net = summary.TotalDebit - summary.TotalCredit
			}
			result.ByAccount[apiAccount] = dto.AccountSummary{
				Debit:     summary.TotalDebit,
				Credit:    summary.TotalCredit,
				NetAmount: net,
			}
		}
	}

	// Map owner contributions to response
	for _, ownerData := range summaryData.OwnerContributions {
		result.ByOwners = append(result.ByOwners, dto.OwnerContribution{
			Owner:             ownerData.Owner,
			TotalContribution: ownerData.TotalContribution,
			Percentage:        math.Round(ownerData.Percentage*100) / 100, // Round to 2 decimal places
		})
	}

	// Cache the result for 5 minutes
	_ = s.cache.Set(ctx, cacheKey, result, constants.LedgerSummaryCacheTTL)

	return result, nil
}

// GetAccountsMetadata returns account types metadata for frontend
func (s *LedgerService) GetAccountsMetadata(ctx context.Context) ([]domain.AccountMetadata, error) {
	return domain.GetAccountsMetadata(), nil
}

// GetEntriesByAssetID returns ledger entries associated with a specific asset
func (s *LedgerService) GetEntriesByAssetID(ctx context.Context, assetID uint) ([]*domain.LedgerEntry, error) {
	return s.LedgerRepo.GetByAssetID(ctx, assetID)
}
