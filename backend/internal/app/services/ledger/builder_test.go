package ledger

import (
	"testing"

	"api-server/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestBuildTransactionLedgerEntries_SplitsScheduledLoanPayment(t *testing.T) {
	txn := &domain.Transaction{
		ID:                  1,
		TransactionType:     domain.TransactionTypeLoanRepayment,
		Amount:              503_125_000,
		LoanPrincipalAmount: 500_000_000,
		LoanInterestAmount:  3_125_000,
		Party:               "Nguyễn Đức Dũng",
		Status:              domain.TransactionStatusSettled,
		CreatedBy:           1,
	}

	entries, err := BuildTransactionLedgerEntries(txn)

	require.NoError(t, err)
	require.Len(t, entries, 3)
	require.Equal(t, domain.AccountLoan, entries[0].Account)
	require.Equal(t, int64(500_000_000), entries[0].Debit)
	require.Equal(t, domain.AccountExpense, entries[1].Account)
	require.Equal(t, int64(3_125_000), entries[1].Debit)
	require.Equal(t, domain.AccountCash, entries[2].Account)
	require.Equal(t, int64(503_125_000), entries[2].Credit)
}
