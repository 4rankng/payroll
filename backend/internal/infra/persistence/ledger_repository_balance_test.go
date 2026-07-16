package persistence

import (
	"testing"

	"api-server/internal/domain"
)

func TestCalculateNetWorthFromAccountTotals(t *testing.T) {
	rows := []accountAgg{
		{Account: domain.AccountCash, TotalDebit: 100, TotalCredit: 20},
		{Account: domain.AccountReceivable, TotalDebit: 40, TotalCredit: 0},
		{Account: domain.AccountPayable, TotalDebit: 5, TotalCredit: 30},
		{Account: domain.AccountLoan, TotalDebit: 0, TotalCredit: 20},
		{Account: domain.AccountRevenue, TotalDebit: 0, TotalCredit: 500},
	}

	if got, want := calculateNetWorthFromAccountTotals(rows), int64(75); got != want {
		t.Fatalf("calculateNetWorthFromAccountTotals() = %d, want %d", got, want)
	}
}
