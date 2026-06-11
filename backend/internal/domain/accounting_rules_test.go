package domain

import (
	"api-server/internal/app/accounting"
	"api-server/internal/pkg/clock"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccountingRules(t *testing.T) {
	rules := NewAccountingRules()
	assert.NotNil(t, rules)
}

func TestAccountingRules_ValidateTransactionGroup(t *testing.T) {
	rules := NewAccountingRules()
	// Use clock.Now() to match the validator which uses clock.Now() for future-date checks.
	now := clock.Now()

	tests := []struct {
		name    string
		group   *TransactionGroup
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid balanced transaction group",
			group: &TransactionGroup{
				Entries: []*LedgerEntry{
					{
						Date:      now,
						Account:   AccountExpense,
						Party:     "Test Party",
						Debit:     1000,
						Credit:    0,
						CreatedBy: 1,
					},
					{
						Date:      now,
						Account:   AccountCash,
						Party:     "Bank",
						Debit:     0,
						Credit:    1000,
						CreatedBy: 1,
					},
				},
				Reference: "TEST-001",
			},
			wantErr: false,
		},
		{
			name:    "Nil transaction group",
			group:   nil,
			wantErr: true,
			errMsg:  "nhóm giao dịch không thể là nil",
		},
		{
			name: "Empty entries",
			group: &TransactionGroup{
				Entries: []*LedgerEntry{},
			},
			wantErr: true,
			errMsg:  "nhóm giao dịch không thể trống",
		},
		{
			name: "Nil entry in group",
			group: &TransactionGroup{
				Entries: []*LedgerEntry{
					nil,
				},
			},
			wantErr: true,
			errMsg:  "mục tại chỉ số 0 không thể là nil",
		},
		{
			name: "Unbalanced transaction - debits > credits",
			group: &TransactionGroup{
				Entries: []*LedgerEntry{
					{
						Date:      now,
						Account:   AccountExpense,
						Party:     "Test Party",
						Debit:     2000,
						Credit:    0,
						CreatedBy: 1,
					},
					{
						Date:      now,
						Account:   AccountCash,
						Party:     "Bank",
						Debit:     0,
						Credit:    1000,
						CreatedBy: 1,
					},
				},
			},
			wantErr: true,
			errMsg:  "giao dịch không cân bằng",
		},
		{
			name: "Unbalanced transaction - credits > debits",
			group: &TransactionGroup{
				Entries: []*LedgerEntry{
					{
						Date:      now,
						Account:   AccountExpense,
						Party:     "Test Party",
						Debit:     1000,
						Credit:    0,
						CreatedBy: 1,
					},
					{
						Date:      now,
						Account:   AccountCash,
						Party:     "Bank",
						Debit:     0,
						Credit:    2000,
						CreatedBy: 1,
					},
				},
			},
			wantErr: true,
			errMsg:  "giao dịch không cân bằng",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rules.ValidateTransactionGroup(tt.group)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAccountingRules_ValidateBalancedTransaction(t *testing.T) {
	rules := NewAccountingRules()
	// Use clock.Now() to match the validator which uses clock.Now() for future-date checks.
	now := clock.Now()
	amount := accounting.NewMoney(1000.0)

	tests := []struct {
		name        string
		transaction *BalancedTransaction
		wantErr     bool
		errMsg      string
	}{
		{
			name: "Valid balanced transaction",
			transaction: &BalancedTransaction{
				DebitEntry: &LedgerEntry{
					Date:      now,
					Account:   AccountExpense,
					Party:     "Test Party",
					Debit:     1000,
					Credit:    0,
					CreatedBy: 1,
				},
				CreditEntry: &LedgerEntry{
					Date:      now,
					Account:   AccountCash,
					Party:     "Bank",
					Debit:     0,
					Credit:    1000,
					CreatedBy: 1,
				},
				Amount: amount,
			},
			wantErr: false,
		},
		{
			name: "Nil debit entry",
			transaction: &BalancedTransaction{
				DebitEntry: nil,
				CreditEntry: &LedgerEntry{
					Date:      now,
					Account:   AccountCash,
					Party:     "Bank",
					Debit:     0,
					Credit:    1000,
					CreatedBy: 1,
				},
				Amount: amount,
			},
			wantErr: true,
			errMsg:  "mục nợ không thể là nil",
		},
		{
			name: "Nil credit entry",
			transaction: &BalancedTransaction{
				DebitEntry: &LedgerEntry{
					Date:      now,
					Account:   AccountExpense,
					Party:     "Test Party",
					Debit:     1000,
					Credit:    0,
					CreatedBy: 1,
				},
				CreditEntry: nil,
				Amount:      amount,
			},
			wantErr: true,
			errMsg:  "mục có không thể là nil",
		},
		{
			name: "Zero amount",
			transaction: &BalancedTransaction{
				DebitEntry: &LedgerEntry{
					Date:      now,
					Account:   AccountExpense,
					Party:     "Test Party",
					Debit:     0,
					Credit:    0,
					CreatedBy: 1,
				},
				CreditEntry: &LedgerEntry{
					Date:      now,
					Account:   AccountCash,
					Party:     "Bank",
					Debit:     0,
					Credit:    0,
					CreatedBy: 1,
				},
				Amount: accounting.NewMoney(0),
			},
			wantErr: true,
			errMsg:  "số tiền giao dịch phải dương",
		},
		{
			name: "Mismatched debit amount",
			transaction: &BalancedTransaction{
				DebitEntry: &LedgerEntry{
					Date:      now,
					Account:   AccountExpense,
					Party:     "Test Party",
					Debit:     500,
					Credit:    0,
					CreatedBy: 1,
				},
				CreditEntry: &LedgerEntry{
					Date:      now,
					Account:   AccountCash,
					Party:     "Bank",
					Debit:     0,
					Credit:    1000,
					CreatedBy: 1,
				},
				Amount: amount,
			},
			wantErr: true,
			errMsg:  "số tiền nợ không khớp với số tiền giao dịch",
		},
		{
			name: "Mismatched credit amount",
			transaction: &BalancedTransaction{
				DebitEntry: &LedgerEntry{
					Date:      now,
					Account:   AccountExpense,
					Party:     "Test Party",
					Debit:     1000,
					Credit:    0,
					CreatedBy: 1,
				},
				CreditEntry: &LedgerEntry{
					Date:      now,
					Account:   AccountCash,
					Party:     "Bank",
					Debit:     0,
					Credit:    500,
					CreatedBy: 1,
				},
				Amount: amount,
			},
			wantErr: true,
			errMsg:  "số tiền có không khớp với số tiền giao dịch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rules.ValidateBalancedTransaction(tt.transaction)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAccountingRules_ValidateAccountBalance(t *testing.T) {
	rules := NewAccountingRules()

	tests := []struct {
		name    string
		account LedgerAccount
		balance *accounting.Money
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid positive balance",
			account: AccountCash,
			balance: accounting.NewMoney(1000.0),
			wantErr: false,
		},
		{
			name:    "Valid zero balance",
			account: AccountRevenue,
			balance: accounting.NewMoney(0),
			wantErr: false,
		},
		{
			name:    "Valid negative balance",
			account: AccountExpense,
			balance: accounting.NewMoney(-500.0),
			wantErr: false,
		},
		{
			name:    "Nil balance",
			account: AccountCash,
			balance: nil,
			wantErr: true,
			errMsg:  "số dư không thể là nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rules.ValidateAccountBalance(tt.account, tt.balance)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAccountingRules_CreateBalancedSalaryTransaction(t *testing.T) {
	rules := NewAccountingRules()

	tests := []struct {
		name         string
		date         string
		employeeName string
		amount       *accounting.Money
		createdBy    uint
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "Valid salary transaction",
			date:         "2024-01-15",
			employeeName: "John Doe",
			amount:       accounting.NewMoney(5000000.0),
			createdBy:    1,
			wantErr:      false,
		},
		{
			name:         "Invalid date format",
			date:         "15-01-2024",
			employeeName: "John Doe",
			amount:       accounting.NewMoney(5000000.0),
			createdBy:    1,
			wantErr:      true,
			errMsg:       "invalid date format",
		},
		{
			name:         "Zero amount",
			date:         "2024-01-15",
			employeeName: "John Doe",
			amount:       accounting.NewMoney(0),
			createdBy:    1,
			wantErr:      true,
			errMsg:       "invalid salary amount",
		},
		{
			name:         "Negative amount",
			date:         "2024-01-15",
			employeeName: "John Doe",
			amount:       accounting.NewMoney(-5000000.0),
			createdBy:    1,
			wantErr:      true,
			errMsg:       "invalid salary amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, err := rules.CreateBalancedSalaryTransaction(tt.date, tt.employeeName, tt.amount, tt.createdBy)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, group)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, group)
				assert.Len(t, group.Entries, 2)
				// Verify debit entry
				assert.Equal(t, AccountExpense, group.Entries[0].Account)
				assert.Greater(t, group.Entries[0].Debit, int64(0))
				// Verify credit entry
				assert.Equal(t, AccountCash, group.Entries[1].Account)
				assert.Greater(t, group.Entries[1].Credit, int64(0))
			}
		})
	}
}

func TestAccountingRules_CreateBalancedRevenueTransaction(t *testing.T) {
	rules := NewAccountingRules()

	tests := []struct {
		name       string
		date       string
		clientName string
		amount     *accounting.Money
		createdBy  uint
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "Valid revenue transaction",
			date:       "2024-01-15",
			clientName: "ABC Company",
			amount:     accounting.NewMoney(10000000.0),
			createdBy:  1,
			wantErr:    false,
		},
		{
			name:       "Invalid date format",
			date:       "2024/01/15",
			clientName: "ABC Company",
			amount:     accounting.NewMoney(10000000.0),
			createdBy:  1,
			wantErr:    true,
			errMsg:     "invalid date format",
		},
		{
			name:       "Zero revenue amount",
			date:       "2024-01-15",
			clientName: "ABC Company",
			amount:     accounting.NewMoney(0),
			createdBy:  1,
			wantErr:    true,
			errMsg:     "invalid revenue amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, err := rules.CreateBalancedRevenueTransaction(tt.date, tt.clientName, tt.amount, tt.createdBy)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, group)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, group)
				assert.Len(t, group.Entries, 2)
			}
		})
	}
}
