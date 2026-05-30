package ledger

import (
	"fmt"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// BuildTransactionLedgerEntries produces the initial double-entry pair when a Transaction is created.
// Account direction is derived from the transaction type via Transaction.GetLedgerAccounts.
// For an already-settled transaction (status=settled at creation time), the credit party uses
// the bank constant; otherwise it uses the txn's own party.
func BuildTransactionLedgerEntries(txn *domain.Transaction) ([]*domain.LedgerEntry, error) {
	debitAccount, creditAccount := txn.GetLedgerAccounts()

	creditParty := txn.Party
	if txn.Status == domain.TransactionStatusSettled {
		creditParty = constants.LedgerPartyBank
	}

	now := clock.Now()

	debitEntry := &domain.LedgerEntry{
		Date:          now,
		Account:       debitAccount,
		Party:         txn.Party,
		Debit:         txn.Amount,
		Credit:        0,
		AssetID:       txn.AssetID,
		TransactionID: &txn.ID,
		CreatedBy:     txn.CreatedBy,
	}

	creditEntry := &domain.LedgerEntry{
		Date:          now,
		Account:       creditAccount,
		Party:         creditParty,
		Debit:         0,
		Credit:        txn.Amount,
		AssetID:       txn.AssetID,
		TransactionID: &txn.ID,
		CreatedBy:     txn.CreatedBy,
	}

	if err := debitEntry.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid transaction debit entry: %w", err)
	}
	if err := creditEntry.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid transaction credit entry: %w", err)
	}

	return []*domain.LedgerEntry{debitEntry, creditEntry}, nil
}

// BuildSettlementLedgerEntries produces the double-entry pair for settling a transaction.
// Pure function: no DB I/O, no side effects. Callers are responsible for persisting the entries
// (typically inside the same DB transaction as the Settlement record, to keep them atomic).
//
// Account direction depends on transaction type:
//   - WriteOff:        Debit Expense,    Credit Receivable  (reduce receivable, recognize loss)
//   - Expense:         Debit Payable,    Credit Cash        (pay off liability with cash)
//   - Revenue:         Debit Cash,       Credit Receivable  (collect what client owed)
//   - Capital:         Debit Cash,       Credit Receivable  (owner contributed promised funds)
//   - LoanRepayment:   Debit Payable,    Credit Cash        (pay down loan principal)
func BuildSettlementLedgerEntries(txn *domain.Transaction, settlement *domain.Settlement) ([]*domain.LedgerEntry, error) {
	var debitAccount, creditAccount domain.LedgerAccount

	switch {
	case txn.IsWriteOff():
		debitAccount = domain.AccountExpense
		creditAccount = domain.AccountReceivable
	case txn.IsExpense():
		debitAccount = domain.AccountPayable
		creditAccount = domain.AccountCash
	case txn.IsRevenue():
		debitAccount = domain.AccountCash
		creditAccount = domain.AccountReceivable
	case txn.IsCapital():
		debitAccount = domain.AccountCash
		creditAccount = domain.AccountReceivable
	case txn.TransactionType == domain.TransactionTypeLoanRepayment:
		debitAccount = domain.AccountPayable
		creditAccount = domain.AccountCash
	default:
		return nil, domain.NewValidationError("loại giao dịch không hợp lệ cho thanh toán")
	}

	debitEntry := &domain.LedgerEntry{
		Date:          settlement.SettlementDate,
		Account:       debitAccount,
		Party:         txn.Party,
		Debit:         settlement.Amount,
		Credit:        0,
		AssetID:       settlement.ProofAssetID,
		TransactionID: &txn.ID,
		SettlementID:  &settlement.ID,
		CreatedBy:     settlement.CreatedBy,
	}

	creditEntry := &domain.LedgerEntry{
		Date:          settlement.SettlementDate,
		Account:       creditAccount,
		Party:         constants.LedgerPartyBank,
		Debit:         0,
		Credit:        settlement.Amount,
		AssetID:       settlement.ProofAssetID,
		TransactionID: &txn.ID,
		SettlementID:  &settlement.ID,
		CreatedBy:     settlement.CreatedBy,
	}

	if err := debitEntry.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid settlement debit entry: %w", err)
	}
	if err := creditEntry.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid settlement credit entry: %w", err)
	}

	return []*domain.LedgerEntry{debitEntry, creditEntry}, nil
}
