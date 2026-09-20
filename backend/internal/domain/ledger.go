package domain

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/constants"
	"api-server/internal/pkg/clock"
	"gorm.io/gorm"
)

// LedgerAccount represents different account types for ledger entries
type LedgerAccount = string

// LedgerEntry represents a ledger entry for double-entry bookkeeping
type LedgerEntry struct {
	ID            uint          `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Date          time.Time     `json:"date" gorm:"type:date;not null"`
	Account       LedgerAccount `json:"account" gorm:"type:varchar(255);not null"`
	Party         string        `json:"party" gorm:"type:varchar(255);not null;comment:'Who you paid or received from'"`
	Debit         int64         `json:"debit" gorm:"type:bigint;not null;default:0;comment:'Money out (VND)'"`
	Credit        int64         `json:"credit" gorm:"type:bigint;not null;default:0;comment:'Money in (VND)'"`
	Balance       int64         `json:"balance" gorm:"type:bigint;not null;default:0;comment:'Running balance (VND)'"`
	AssetID       *uint         `json:"asset_id" gorm:"type:bigint unsigned;comment:'Reference to asset table for evidence files'"`
	TransactionID *uint         `json:"transaction_id" gorm:"type:bigint unsigned;comment:'Reference to transaction table for user-facing transactions'"`
	// ReversalOfEntryID links a reversal (mirror) to the entry it offsets. It is
	// nil for originals, which is how "has this entry been reversed" is answered
	// without guessing from signs and timestamps.
	ReversalOfEntryID *uint          `json:"reversal_of_entry_id" gorm:"type:bigint unsigned;index;comment:'Entry this row reverses, NULL for originals'"`
	ReversalReason    string         `json:"reversal_reason,omitempty" gorm:"type:varchar(255);comment:'Operator-supplied reason for the reversal'"`
	SettlementID      *uint          `json:"settlement_id" gorm:"type:bigint unsigned;index;comment:'Reference to settlement table for payment clearing entries'"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedBy         uint           `json:"created_by" gorm:"not null"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`

	// Relationships
	Asset      *Asset      `json:"asset,omitempty" gorm:"foreignKey:AssetID;references:ID"`
	Settlement *Settlement `json:"settlement,omitempty" gorm:"foreignKey:SettlementID;references:ID"`
	Creator    User        `json:"creator" gorm:"foreignKey:CreatedBy;references:ID"`
}

// LedgerEntryRepository defines the interface for ledger entry persistence operations
type LedgerEntryRepository interface {
	Create(ctx context.Context, entry *LedgerEntry) error
	GetByID(ctx context.Context, id uint) (*LedgerEntry, error)
	GetByAssetID(ctx context.Context, assetID uint) ([]*LedgerEntry, error)
	GetByTransactionID(ctx context.Context, txnID uint) ([]*LedgerEntry, error)
	// ListReversalGroup returns the balanced block an entry belongs to, which is
	// what a reversal has to mirror to keep the ledger's debits and credits equal.
	// Entries carrying a transaction id group by it; manual blocks (written by the
	// /ledger/entries endpoint in one insert) carry none and group by the write
	// timestamp and author instead.
	ListReversalGroup(ctx context.Context, entry *LedgerEntry) ([]*LedgerEntry, error)
	// HasReversal reports whether the entry already has a live mirror. Reversing
	// twice would double the offset, so callers refuse instead.
	HasReversal(ctx context.Context, entryID uint) (bool, error)
	GetBySettlementID(ctx context.Context, settlementID uint) ([]*LedgerEntry, error)
	List(ctx context.Context, filters LedgerFilters) ([]*LedgerEntry, error)
	Count(ctx context.Context, filters LedgerFilters) (int64, error)
	GetByAccount(ctx context.Context, account LedgerAccount) ([]*LedgerEntry, error)
	GetByDateRange(ctx context.Context, start, end time.Time) ([]*LedgerEntry, error)
	GetBalance(ctx context.Context) (int64, error)
	GetBalanceByAccount(ctx context.Context, account LedgerAccount) (int64, error)
	// GetAccountTotalInRange returns SUM(debit) - SUM(credit) for the given
	// account over the inclusive date range [from, to]. int64 VND. Used by the
	// settlement simulation's reconciliation against the ledger receivable.
	GetAccountTotalInRange(ctx context.Context, account LedgerAccount, from, to time.Time) (int64, error)
	// GetAccountTotalForTransactions returns SUM(debit) - SUM(credit) for the
	// given account, limited to the supplied transaction IDs.
	GetAccountTotalForTransactions(ctx context.Context, account LedgerAccount, transactionIDs []uint) (int64, error)
	CreateTransaction(ctx context.Context, entries []*LedgerEntry) error
	GetCashFlowSummary(ctx context.Context, start, end time.Time) (*CashFlowSummary, error)
	GetTotalByAccountType(ctx context.Context, accountType string, startDate, endDate time.Time) (int64, error)
	GetMonthlyFinancialsBatch(ctx context.Context, startDate, endDate time.Time) (map[string]MonthlyFinancials, error)
	SearchLedgerEntries(ctx context.Context, search string, limit int) ([]*LedgerEntry, error)
	RecalculateAllBalances(ctx context.Context) error
	GetLedgerSummary(ctx context.Context, fromDate, toDate time.Time) (*LedgerSummaryData, error)
	CheckDuplicate(ctx context.Context, entry *LedgerEntry) (*LedgerEntry, error)
	BatchCheckDuplicates(ctx context.Context, entries []*LedgerEntry) (map[int]*LedgerEntry, error)
	GetFinancialChartData(ctx context.Context, period string, startDate, endDate time.Time) ([]FinancialChartDataPoint, error)
	// Deletion helpers (soft delete)
	DeleteByTransactionID(ctx context.Context, txnID uint) error
}

// LedgerFilters represents filtering options for ledger queries
type LedgerFilters struct {
	Account   []LedgerAccount
	CreatedBy *uint
	Party     *string
	FromDate  *time.Time
	ToDate    *time.Time
	Search    string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// CashFlowSummary represents cash flow summary information
type CashFlowSummary struct {
	StartDate      time.Time                        `json:"start_date"`
	EndDate        time.Time                        `json:"end_date"`
	TotalInflow    int64                            `json:"total_inflow"`
	TotalOutflow   int64                            `json:"total_outflow"`
	NetCashFlow    int64                            `json:"net_cash_flow"`
	ByAccount      map[LedgerAccount]AccountSummary `json:"by_account"`
	OpeningBalance int64                            `json:"opening_balance"`
	ClosingBalance int64                            `json:"closing_balance"`
}

// AccountSummary represents summary for a specific account
type AccountSummary struct {
	Account     LedgerAccount `json:"account"`
	TotalDebit  int64         `json:"total_debit"`
	TotalCredit int64         `json:"total_credit"`
	NetAmount   int64         `json:"net_amount"`
	EntryCount  int           `json:"entry_count"`
}

// LedgerEntryWithDetails represents ledger entry with additional details
type LedgerEntryWithDetails struct {
	LedgerEntry
	CreatorName string `json:"creator_name"`
}

// LedgerSummaryData holds the raw summary data from repository
type LedgerSummaryData struct {
	OpeningBalance     int64
	ClosingBalance     int64
	AccountSummaries   map[string]AccountSummaryData
	OwnerContributions []OwnerContributionData
}

// AccountSummaryData holds account-specific summary information
type AccountSummaryData struct {
	TotalDebit  int64
	TotalCredit int64
	NetAmount   int64
}

// OwnerContributionData represents capital contribution by an owner
type OwnerContributionData struct {
	Owner             string
	TotalContribution int64
	Percentage        float64
}

// MonthlyFinancials represents aggregated revenue and expenses for a single month
type MonthlyFinancials struct {
	Month    string
	Revenue  int64
	Expenses int64
}

// FinancialChartDataPoint represents aggregated financial data for a time period
type FinancialChartDataPoint struct {
	Date            string
	CashVND         int64
	ReceivableVND   int64
	PayableVND      int64
	RevenueVND      int64
	ExpensesVND     int64
	ProfitVND       int64
	ActiveEmployees int
}

// ValidateDate validates the ledger entry date
func (le *LedgerEntry) ValidateDate() error {
	if le.Date.IsZero() {
		return NewValidationError("ngày là bắt buộc")
	}

	// Date cannot be in the future
	// Compare date-only in local timezone to avoid midnight boundary issues
	now := clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	entryDate := time.Date(le.Date.Year(), le.Date.Month(), le.Date.Day(), 0, 0, 0, 0, now.Location())

	if entryDate.After(today) {
		return NewValidationError("ngày không thể ở tương lai")
	}

	return nil
}

// IsValidAccountType checks if the account type is valid
func IsValidAccountType(account LedgerAccount) bool {
	validAccounts := GetValidLedgerAccounts()
	for _, validAccount := range validAccounts {
		if account == validAccount {
			return true
		}
	}
	return false
}

// ValidateAccount validates the account type
func (le *LedgerEntry) ValidateAccount() error {
	if le.Account == "" {
		return NewValidationError("tài khoản là bắt buộc")
	}

	if !IsValidAccountType(le.Account) {
		return NewValidationError(fmt.Sprintf("loại tài khoản không hợp lệ: %s", le.Account))
	}

	return nil
}

// ValidateParty validates the party field
func (le *LedgerEntry) ValidateParty() error {
	if le.Party == "" {
		return NewValidationError("quản lý là bắt buộc")
	}
	return nil
}

// ValidateAmounts validates debit and credit amounts
func (le *LedgerEntry) ValidateAmounts() error {
	if le.Debit < 0 {
		return NewValidationError("số tiền nợ không thể âm")
	}

	if le.Credit < 0 {
		return NewValidationError("số tiền có không thể âm")
	}

	// Exactly one of debit or credit should be non-zero
	if le.Debit > 0 && le.Credit > 0 {
		return NewValidationError("mục không thể có cả số tiền nợ và có")
	}

	if le.Debit == 0 && le.Credit == 0 {
		return NewValidationError("mục phải có số tiền nợ hoặc có")
	}

	return nil
}

// IsValid validates the entire ledger entry
func (le *LedgerEntry) IsValid() error {
	if err := le.ValidateDate(); err != nil {
		return err
	}
	if err := le.ValidateAccount(); err != nil {
		return err
	}
	if err := le.ValidateParty(); err != nil {
		return err
	}
	if err := le.ValidateAmounts(); err != nil {
		return err
	}
	return nil
}

// IsDebit returns true if this entry is a debit
func (le *LedgerEntry) IsDebit() bool {
	return le.Debit > 0
}

// IsCredit returns true if this entry is a credit
func (le *LedgerEntry) IsCredit() bool {
	return le.Credit > 0
}

// GetAmount returns the absolute amount of the entry
func (le *LedgerEntry) GetAmount() int64 {
	if le.IsDebit() {
		return le.Debit
	}
	return le.Credit
}

// GetSignedAmount returns the amount with appropriate sign (negative for debit, positive for credit)
func (le *LedgerEntry) GetSignedAmount() int64 {
	if le.IsDebit() {
		return -le.Debit
	}
	return le.Credit
}

// SetBalance sets the running balance
func (le *LedgerEntry) SetBalance(balance int64) {
	le.Balance = balance
}

// ValidateBalance validates the running balance calculation
func (le *LedgerEntry) ValidateBalance(expectedBalance int64) error {
	if le.Balance != expectedBalance {
		return NewValidationError(fmt.Sprintf("số dư không khớp: mong muốn %d, nhận được %d", expectedBalance, le.Balance))
	}
	return nil
}

// CreateDebitEntry creates a new debit entry
func CreateDebitEntry(date time.Time, account LedgerAccount, party string, amount int64, createdBy uint) *LedgerEntry {
	return &LedgerEntry{
		Date:      date,
		Account:   account,
		Party:     party,
		Debit:     amount,
		Credit:    0,
		CreatedBy: createdBy,
	}
}

// CreateCreditEntry creates a new credit entry
func CreateCreditEntry(date time.Time, account LedgerAccount, party string, amount int64, createdBy uint) *LedgerEntry {
	return &LedgerEntry{
		Date:      date,
		Account:   account,
		Party:     party,
		Debit:     0,
		Credit:    amount,
		CreatedBy: createdBy,
	}
}

// CreateRevenueEntries creates ledger entries for revenue receipt
func CreateRevenueEntries(date time.Time, clientName string, amount int64, createdBy uint) []*LedgerEntry {
	entries := []*LedgerEntry{
		// Debit: Cash (increase in asset)
		{
			Date:      date,
			Account:   AccountCash,
			Party:     constants.LedgerPartyBank,
			Debit:     amount,
			Credit:    0,
			CreatedBy: createdBy,
		},
		// Credit: Receivable (offset accounts receivable)
		{
			Date:      date,
			Account:   AccountReceivable,
			Party:     clientName,
			Debit:     0,
			Credit:    amount,
			CreatedBy: createdBy,
		},
	}

	return entries
}

// Standard account types - simplified to essential accounts
const (
	AccountCash       = "cash"
	AccountReceivable = "receivable"
	AccountPayable    = "payable"
	AccountExpense    = "expense"
	AccountRevenue    = "revenue"
	AccountEquity     = "equity"
	AccountLoan       = "loan" // Liability account for borrowed funds
)

// Account categories for classification
type AccountCategory string

const (
	CategoryAsset     AccountCategory = "asset"
	CategoryLiability AccountCategory = "liability"
	CategoryEquity    AccountCategory = "equity"
	CategoryRevenue   AccountCategory = "revenue"
	CategoryExpense   AccountCategory = "expense"
)

// GetAccountCategory returns the category for an account
func GetAccountCategory(account string) AccountCategory {
	switch account {
	case AccountCash, AccountReceivable:
		return CategoryAsset
	case AccountPayable, AccountLoan:
		return CategoryLiability
	case AccountExpense:
		return CategoryExpense
	case AccountRevenue:
		return CategoryRevenue
	case AccountEquity:
		return CategoryEquity
	default:
		return CategoryAsset // default for unknown accounts
	}
}

// IsDebitAccount returns true if account normally has debit balance
func IsDebitAccount(account string) bool {
	category := GetAccountCategory(account)
	return category == CategoryAsset || category == CategoryExpense
}

// IsCreditAccount returns true if account normally has credit balance
func IsCreditAccount(account string) bool {
	category := GetAccountCategory(account)
	return category == CategoryEquity || category == CategoryRevenue || category == CategoryLiability
}

// GetValidLedgerAccounts returns all valid ledger account types
func GetValidLedgerAccounts() []LedgerAccount {
	return []LedgerAccount{
		AccountCash,
		AccountReceivable,
		AccountPayable,
		AccountExpense,
		AccountRevenue,
		AccountEquity,
		AccountLoan,
	}
}

// GetAccountDisplayName returns the Vietnamese display name for an account type
func GetAccountDisplayName(account LedgerAccount) string {
	displayNames := map[LedgerAccount]string{
		AccountCash:       "Tiền mặt",
		AccountReceivable: "Phải thu",
		AccountPayable:    "Phải trả",
		AccountExpense:    "Chi phí",
		AccountRevenue:    "Doanh thu",
		AccountEquity:     "Vốn chủ sở hữu",
		AccountLoan:       "Khoản vay",
	}

	if name, exists := displayNames[account]; exists {
		return name
	}

	return account
}

// AccountMetadata represents metadata for an account type
type AccountMetadata struct {
	Value      LedgerAccount   `json:"value"`
	Label      string          `json:"label"`
	Category   AccountCategory `json:"category"`
	NormalSide string          `json:"normal_side"`
}

// GetAccountsMetadata returns metadata for all valid account types
func GetAccountsMetadata() []AccountMetadata {
	validAccounts := GetValidLedgerAccounts()
	metadata := make([]AccountMetadata, 0, len(validAccounts))

	for _, account := range validAccounts {
		normalSide := "credit"
		// Assets normally have debit balance
		if IsDebitAccount(string(account)) {
			normalSide = "debit"
		}
		// Special case for expense account per API specification
		if account == AccountExpense {
			normalSide = "credit"
		}

		metadata = append(metadata, AccountMetadata{
			Value:      account,
			Label:      GetAccountDisplayName(account),
			Category:   GetAccountCategory(string(account)),
			NormalSide: normalSide,
		})
	}

	return metadata
}

// GetUserBalance calculates the intuitive balance for an account from internal debit/credit
// Returns positive values according to user's mental model:
// - Cash: positive = more money in hand (Asset: Debit increases)
// - Expense: positive = more spending (Expense: Debit increases)
// - Revenue: positive = more earnings (Revenue: Credit increases)
// - Payable: positive = owe more to vendors (Liability: Credit increases)
// - Receivable: positive = clients owe more (Asset: Debit increases)
// - Equity: positive = more owner's capital (Equity: Credit increases)
//
// Examples:
// - Cash: If debit=100, credit=30, balance=70 (you have 70 in cash)
// - Expense: If debit=50, credit=0, balance=50 (you spent 50)
// - Revenue: If debit=0, credit=80, balance=80 (you earned 80)
// - Equity: If debit=0, credit=100, balance=100 (owner invested 100)
func GetUserBalance(account LedgerAccount, debitTotal, creditTotal int64) int64 {
	switch account {
	case AccountCash, AccountReceivable:
		// Assets: Debit increases balance (normal debit account)
		// Positive means more assets
		return debitTotal - creditTotal
	case AccountExpense:
		// Expense: Debit increases spending (normal debit account)
		// Positive means more spending
		return debitTotal - creditTotal
	case AccountRevenue:
		// Revenue: Credit increases earnings (normal credit account)
		// Positive means more income
		return creditTotal - debitTotal
	case AccountPayable:
		// Payable: Credit increases liability (normal credit account)
		// Positive means owe more to vendors
		return creditTotal - debitTotal
	case AccountEquity:
		// Equity: Credit increases capital (normal credit account)
		// Positive means more owner's investment
		return creditTotal - debitTotal
	case AccountLoan:
		// Loan: Credit increases liability (normal credit account)
		// Positive means more borrowed (owe more to lenders)
		return creditTotal - debitTotal
	default:
		return debitTotal - creditTotal
	}
}

// GetUserBalanceFromEntry calculates intuitive balance change from a single entry
func GetUserBalanceFromEntry(entry *LedgerEntry) int64 {
	return GetUserBalance(entry.Account, entry.Debit, entry.Credit)
}

// ComputeBalanceDelta calculates how a single ledger entry affects the operational liquidity balance.
// This is the SINGLE SOURCE OF TRUTH for balance calculation logic.
//
// Business Rule: balance tracks operational liquidity = Cash - Payable - Loan + Receivable
//
// Account semantics:
// - Assets (Cash, Receivable): credit - debit increases balance
// - Liabilities (Payable, Loan): credit - debit decreases balance (we owe more)
// - Revenue, Expense, Equity: do not affect the balance field at all
//
// The balance field represents working capital / net liquidity, not full P&L.
func ComputeBalanceDelta(account LedgerAccount, debit, credit int64) int64 {
	entryAmount := credit - debit

	switch account {
	case AccountCash, AccountReceivable:
		// Assets: add to balance
		return entryAmount

	case AccountPayable, AccountLoan:
		// Liabilities: subtract from balance
		return -entryAmount

	case AccountRevenue, AccountExpense, AccountEquity:
		// Do not affect liquidity balance
		return 0

	default:
		// Safe default: don't touch balance if we don't know the type
		return 0
	}
}

// CalculateNextBalance calculates the next running balance given the previous balance and current entry.
// This is a pure calculation function for efficient running balance computation.
//
// Usage:
//
//	previousBalance := int64(1000000)
//	entry := &LedgerEntry{Account: AccountCash, Debit: 0, Credit: 500000}
//	newBalance := CalculateNextBalance(previousBalance, entry)
//	// newBalance = 1500000
func CalculateNextBalance(previousBalance int64, entry *LedgerEntry) int64 {
	delta := ComputeBalanceDelta(entry.Account, entry.Debit, entry.Credit)
	return previousBalance + delta
}

// CalculateRunningBalances updates the Balance field for a slice of ledger entries in-place.
// Entries must be pre-sorted in chronological order (by date ASC, id ASC).
// This function mutates the Balance field of each entry.
//
// Usage:
//
//	entries := []*LedgerEntry{entry1, entry2, entry3}
//	startingBalance := int64(1000000)
//	CalculateRunningBalances(startingBalance, entries)
//	// entries[0].Balance, entries[1].Balance, entries[2].Balance are now set
func CalculateRunningBalances(startingBalance int64, entries []*LedgerEntry) {
	runningBalance := startingBalance
	for _, entry := range entries {
		runningBalance = CalculateNextBalance(runningBalance, entry)
		entry.Balance = runningBalance
	}
}
