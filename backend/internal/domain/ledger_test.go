package domain

import (
	"testing"

	"api-server/internal/pkg/clock"

	"github.com/stretchr/testify/assert"
)

func TestLedger_ValidateDate(t *testing.T) {
	l := &LedgerEntry{
		Date: clock.Now(),
	}

	err := l.ValidateDate()
	assert.NoError(t, err)

	// Test date in the future
	l.Date = clock.Now().AddDate(0, 0, 1) // tomorrow
	err = l.ValidateDate()
	assert.Error(t, err)
}

func TestIsValidAccountType(t *testing.T) {
	assert.True(t, IsValidAccountType(AccountCash))
	assert.True(t, IsValidAccountType(AccountReceivable))
	assert.True(t, IsValidAccountType(AccountPayable))
	assert.True(t, IsValidAccountType(AccountExpense))
	assert.True(t, IsValidAccountType(AccountRevenue))

	// Test invalid account type
	assert.False(t, IsValidAccountType("invalid"))
}

func TestLedger_ValidateAccount(t *testing.T) {
	l := &LedgerEntry{}

	// Test valid account
	l.Account = AccountCash
	err := l.ValidateAccount()
	assert.NoError(t, err)

	// Test empty account
	l.Account = ""
	err = l.ValidateAccount()
	assert.Error(t, err)

	// Test invalid account format
	l.Account = "invalid_account"
	err = l.ValidateAccount()
	assert.Error(t, err)
}

func TestLedger_ValidateAmounts(t *testing.T) {
	l := &LedgerEntry{}

	// Test valid amounts
	l.Debit = 100
	l.Credit = 0
	err := l.ValidateAmounts()
	assert.NoError(t, err)

	// Test negative amounts
	l.Debit = -100
	l.Credit = 0
	err = l.ValidateAmounts()
	assert.Error(t, err)

	// Test both debit and credit non-zero
	l.Debit = 100
	l.Credit = 50
	err = l.ValidateAmounts()
	assert.Error(t, err)

	// Test both debit and credit zero
	l.Debit = 0
	l.Credit = 0
	err = l.ValidateAmounts()
	assert.Error(t, err)
}

func TestLedger_IsValid(t *testing.T) {
	l := &LedgerEntry{
		Date:    clock.Now().AddDate(0, 0, -1), // yesterday
		Account: AccountCash,
		Debit:   100,
		Credit:  0,
		Party:   "Test Party",
	}

	// Test valid ledger entry
	err := l.IsValid()
	assert.NoError(t, err)

	// Test invalid ledger entry
	l.Date = clock.Now().AddDate(0, 0, 1) // tomorrow
	err = l.IsValid()
	assert.Error(t, err)
}

func TestLedger_IsDebit(t *testing.T) {
	l := &LedgerEntry{
		Debit:  100.0,
		Credit: 0.0,
	}

	assert.True(t, l.IsDebit())
	assert.False(t, l.IsCredit())

	l.Debit = 0.0
	l.Credit = 100.0
	assert.False(t, l.IsDebit())
	assert.True(t, l.IsCredit())
}

func TestLedger_GetAmount(t *testing.T) {
	l := &LedgerEntry{
		Debit:  100,
		Credit: 0,
	}

	amount := l.GetAmount()
	assert.Equal(t, int64(100), amount)

	l.Debit = 0
	l.Credit = 150
	amount = l.GetAmount()
	assert.Equal(t, int64(150), amount)
}

func TestLedger_GetSignedAmount(t *testing.T) {
	l := &LedgerEntry{
		Debit:  100,
		Credit: 0,
	}

	amount := l.GetSignedAmount()
	assert.Equal(t, int64(-100), amount) // Debit is negative

	l.Debit = 0
	l.Credit = 150
	amount = l.GetSignedAmount()
	assert.Equal(t, int64(150), amount) // Credit is positive
}
