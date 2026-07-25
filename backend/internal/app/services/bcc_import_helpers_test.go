package services

import (
	"testing"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
)

// TestApplySTKBankFields covers the BCC import regression where employees
// were rejected with "bank selection is required when providing banking
// information" because the STK sheet had an account number but no bank name
// (so BankID could not be resolved). In that case the profile must still be
// created — without bank fields — so the timesheet can be imported.
func TestApplySTKBankFields(t *testing.T) {
	bankID := uint(7)

	t.Run("complete STK row populates all bank fields", func(t *testing.T) {
		emp := &domain.Employee{}
		row := excelparser.STKRow{
			BankAccount: "123456789",
			BankName:    "Vietcombank",
		}
		applySTKBankFields(emp, row, &bankID, "Nguyễn Văn A")
		assert.Equal(t, &bankID, emp.BankID)
		assert.Equal(t, "123456789", emp.BankAccountNumber)
		assert.Equal(t, "NGUYỄN VĂN A", emp.BankAccountName)
	})

	t.Run("account number without bank id leaves fields empty", func(t *testing.T) {
		// This is the exact reported bug: STK has BankAccount but no BankName,
		// ResolveBankID returns nil. The profile must be created without bank
		// info rather than being rejected by ValidateBankingInfo.
		emp := &domain.Employee{}
		row := excelparser.STKRow{
			BankAccount: "987654321",
			BankName:    "", // unresolved → nil bankID
		}
		applySTKBankFields(emp, row, nil, "Trần Thị B")
		assert.Nil(t, emp.BankID)
		assert.Empty(t, emp.BankAccountNumber)
		assert.Empty(t, emp.BankAccountName)
	})

	t.Run("bank id without account number leaves fields empty", func(t *testing.T) {
		// BankID alone is not enough — without an account number the record
		// is incomplete, so skip bank fields entirely.
		emp := &domain.Employee{}
		row := excelparser.STKRow{
			BankAccount: "",
			BankName:    "Vietcombank",
		}
		applySTKBankFields(emp, row, &bankID, "Lê Văn C")
		assert.Nil(t, emp.BankID)
		assert.Empty(t, emp.BankAccountNumber)
		assert.Empty(t, emp.BankAccountName)
	})

	t.Run("empty STK row leaves fields empty", func(t *testing.T) {
		emp := &domain.Employee{}
		applySTKBankFields(emp, excelparser.STKRow{}, nil, "Phạm Thị D")
		assert.Nil(t, emp.BankID)
		assert.Empty(t, emp.BankAccountNumber)
		assert.Empty(t, emp.BankAccountName)
	})

	t.Run("empty employee passes ValidateBankingInfo after sanitize", func(t *testing.T) {
		// Regression guard: the sanitized result must pass IsValid() so that
		// CreateEmployee no longer rejects the import.
		emp := &domain.Employee{}
		row := excelparser.STKRow{
			BankAccount: "987654321",
			BankName:    "",
		}
		applySTKBankFields(emp, row, nil, "Lò Thị Hiêm")
		assert.NoError(t, emp.ValidateBankingInfo())
	})

	t.Run("whitespace-only fullname yields empty account name", func(t *testing.T) {
		emp := &domain.Employee{}
		row := excelparser.STKRow{
			BankAccount: "111",
			BankName:    "BIDV",
		}
		applySTKBankFields(emp, row, &bankID, "   ")
		assert.Equal(t, &bankID, emp.BankID)
		assert.Empty(t, emp.BankAccountName) // TrimSpace collapses to ""
	})
}

func TestBuildSTKBankUpdates(t *testing.T) {
	oldBankID := uint(7)
	newBankID := uint(8)
	employee := &domain.Employee{
		BankID:            &oldBankID,
		BankAccountNumber: "123456789",
		BankAccountName:   "Nguyễn Văn A",
	}

	t.Run("changed account number creates a complete update", func(t *testing.T) {
		updates := buildSTKBankUpdates(
			employee,
			excelparser.STKRow{BankAccount: "987654321"},
			nil,
			"Nguyễn Văn A",
		)

		assert.Equal(t, "987654321", updates["bank_account_number"])
		assert.Equal(t, "NGUYỄN VĂN A", updates["bank_account_name"])
		assert.NotContains(t, updates, "bank_id")
	})

	t.Run("changed bank is included when it resolves", func(t *testing.T) {
		updates := buildSTKBankUpdates(
			employee,
			excelparser.STKRow{BankAccount: "123456789", BankName: "BIDV"},
			&newBankID,
			"Nguyễn Văn A",
		)

		assert.Equal(t, newBankID, updates["bank_id"])
	})

	t.Run("unchanged values do not trigger revalidation", func(t *testing.T) {
		updates := buildSTKBankUpdates(
			employee,
			excelparser.STKRow{BankAccount: "123456789"},
			&oldBankID,
			"NGUYỄN VĂN A",
		)

		assert.Nil(t, updates)
	})

	t.Run("empty account number is not authoritative", func(t *testing.T) {
		updates := buildSTKBankUpdates(
			employee,
			excelparser.STKRow{BankName: "BIDV"},
			&newBankID,
			"Nguyễn Văn A",
		)

		assert.Nil(t, updates)
	})

	t.Run("provided but unresolved bank clears the stale bank relation", func(t *testing.T) {
		updates := buildSTKBankUpdates(
			employee,
			excelparser.STKRow{
				BankAccount: "987654321",
				BankName:    "Ngân hàng không xác định",
			},
			nil,
			"Nguyễn Văn A",
		)

		assert.Contains(t, updates, "bank_id")
		assert.Nil(t, updates["bank_id"])
		assert.Equal(t, "987654321", updates["bank_account_number"])
	})
}
