package domain

import (
	"api-server/internal/app/accounting"
	"api-server/internal/constants"
	"fmt"
	"math"
	"time"
)

// TransactionGroup represents a group of ledger entries that should balance
type TransactionGroup struct {
	Entries     []*LedgerEntry
	Description string
	Reference   string
}

// BalancedTransaction represents a double-entry transaction with guaranteed balance
type BalancedTransaction struct {
	DebitEntry  *LedgerEntry
	CreditEntry *LedgerEntry
	Amount      *accounting.Money
}

// AccountingRules contains validation rules for double-entry bookkeeping
type AccountingRules struct{}

// NewAccountingRules creates a new instance of accounting rules validator
func NewAccountingRules() *AccountingRules {
	return &AccountingRules{}
}

// ValidateTransactionGroup validates that a group of entries follows double-entry rules
func (r *AccountingRules) ValidateTransactionGroup(group *TransactionGroup) error {
	if group == nil {
		return NewValidationError("nhóm giao dịch không thể là nil")
	}

	if len(group.Entries) == 0 {
		return NewValidationError("nhóm giao dịch không thể trống")
	}

	totalDebits := accounting.NewMoney(0)
	totalCredits := accounting.NewMoney(0)

	for i, entry := range group.Entries {
		if entry == nil {
			return NewValidationError(fmt.Sprintf("mục tại chỉ số %d không thể là nil", i))
		}

		// Validate individual entry
		if err := entry.IsValid(); err != nil {
			return fmt.Errorf("invalid entry at index %d: %w", i, err)
		}

		// Add to running totals
		if entry.Debit > 0 {
			totalDebits = totalDebits.Add(accounting.NewMoney(float64(entry.Debit)))
		}
		if entry.Credit > 0 {
			totalCredits = totalCredits.Add(accounting.NewMoney(float64(entry.Credit)))
		}
	}

	// Verify debits equal credits
	if !totalDebits.Equal(totalCredits) {
		return NewValidationError(fmt.Sprintf("giao dịch không cân bằng: nợ=%s, có=%s",
			totalDebits.String(), totalCredits.String()))
	}

	return nil
}

// ValidateBalancedTransaction validates a simple debit/credit pair
func (r *AccountingRules) ValidateBalancedTransaction(transaction *BalancedTransaction) error {
	if transaction.DebitEntry == nil {
		return NewValidationError("mục nợ không thể là nil")
	}

	if transaction.CreditEntry == nil {
		return NewValidationError("mục có không thể là nil")
	}

	if transaction.Amount == nil || transaction.Amount.IsZero() {
		return NewValidationError("số tiền giao dịch phải dương")
	}

	// Validate individual entries
	if err := transaction.DebitEntry.IsValid(); err != nil {
		return fmt.Errorf("invalid debit entry: %w", err)
	}

	if err := transaction.CreditEntry.IsValid(); err != nil {
		return fmt.Errorf("invalid credit entry: %w", err)
	}

	// Verify amounts match
	debitAmount := accounting.NewMoney(float64(transaction.DebitEntry.Debit))
	creditAmount := accounting.NewMoney(float64(transaction.CreditEntry.Credit))

	if !debitAmount.Equal(transaction.Amount) {
		return NewValidationError("số tiền nợ không khớp với số tiền giao dịch")
	}

	if !creditAmount.Equal(transaction.Amount) {
		return NewValidationError("số tiền có không khớp với số tiền giao dịch")
	}

	if !debitAmount.Equal(creditAmount) {
		return NewValidationError("số tiền nợ và có phải bằng nhau")
	}

	return nil
}

// ValidateAccountBalance validates account balance rules based on account type
func (r *AccountingRules) ValidateAccountBalance(account LedgerAccount, balance *accounting.Money) error {
	if balance == nil {
		return NewValidationError("số dư không thể là nil")
	}

	// All 5 account types can have any balance
	// Validation is performed at the entry level, not at the balance level
	return nil
}

// CreateBalancedSalaryTransaction creates a balanced transaction for salary payments
// Accounting: Debit Expense (increase expense) + Credit Cash (decrease cash)
func (r *AccountingRules) CreateBalancedSalaryTransaction(
	date string,
	employeeName string,
	amount *accounting.Money,
	createdBy uint,
) (*TransactionGroup, error) {

	if err := accounting.ValidateMoneyAmount(amount); err != nil {
		return nil, fmt.Errorf("invalid salary amount: %w", err)
	}

	// Parse the date string
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	debitEntry := &LedgerEntry{
		Date:      parsedDate,
		Account:   AccountExpense,
		Party:     employeeName,
		Debit:     int64(math.Round(amount.Float())),
		Credit:    0,
		CreatedBy: createdBy,
	}

	creditEntry := &LedgerEntry{
		Date:      parsedDate,
		Account:   AccountCash,
		Party:     constants.LedgerPartyBank,
		Debit:     0,
		Credit:    int64(math.Round(amount.Float())),
		CreatedBy: createdBy,
	}

	group := &TransactionGroup{
		Entries: []*LedgerEntry{debitEntry, creditEntry},
	}

	// Validate the created transaction
	if err := r.ValidateTransactionGroup(group); err != nil {
		return nil, fmt.Errorf("failed to create valid salary transaction: %w", err)
	}

	return group, nil
}

// CreateBalancedRevenueTransaction creates a balanced transaction for revenue
// Accounting: Debit Cash (increase cash) + Credit Revenue (increase revenue)
func (r *AccountingRules) CreateBalancedRevenueTransaction(
	date string,
	clientName string,
	amount *accounting.Money,
	createdBy uint,
) (*TransactionGroup, error) {

	if err := accounting.ValidateMoneyAmount(amount); err != nil {
		return nil, fmt.Errorf("invalid revenue amount: %w", err)
	}

	// Parse the date string
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	debitEntry := &LedgerEntry{
		Date:      parsedDate,
		Account:   AccountCash,
		Party:     constants.LedgerPartyBank,
		Debit:     int64(math.Round(amount.Float())),
		Credit:    0,
		CreatedBy: createdBy,
	}

	creditEntry := &LedgerEntry{
		Date:      parsedDate,
		Account:   AccountRevenue,
		Party:     clientName,
		Debit:     0,
		Credit:    int64(math.Round(amount.Float())),
		CreatedBy: createdBy,
	}

	group := &TransactionGroup{
		Entries: []*LedgerEntry{debitEntry, creditEntry},
	}

	// Validate the created transaction
	if err := r.ValidateTransactionGroup(group); err != nil {
		return nil, fmt.Errorf("failed to create valid revenue transaction: %w", err)
	}

	return group, nil
}

// CreateReversalTransaction creates a reversal transaction for an existing entry
func (r *AccountingRules) CreateReversalTransaction(
	originalEntry *LedgerEntry,
	reason string,
	createdBy uint,
) (*LedgerEntry, error) {

	if originalEntry == nil {
		return nil, NewValidationError("mục gốc không thể là nil")
	}

	if reason == "" {
		return nil, NewValidationError("lý do đảo ngược là bắt buộc")
	}

	reversalEntry := &LedgerEntry{
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

	return reversalEntry, nil
}

// CalculateRunningBalance calculates running balance with precise arithmetic
func (r *AccountingRules) CalculateRunningBalance(
	previousBalance *accounting.Money,
	entry *LedgerEntry,
) *accounting.Money {

	if previousBalance == nil {
		previousBalance = accounting.NewMoney(0)
	}

	credit := accounting.NewMoney(float64(entry.Credit))
	debit := accounting.NewMoney(float64(entry.Debit))

	// Balance = Previous Balance + Credits - Debits
	return previousBalance.Add(credit).Subtract(debit)
}
