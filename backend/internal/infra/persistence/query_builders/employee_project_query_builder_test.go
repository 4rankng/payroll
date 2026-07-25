package query_builders

import (
	"strings"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildMissingBankDetailsQueryIncludesEveryRequiredBankField(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	require.NoError(t, err)

	builder := NewEmployeeProjectQueryBuilder(db)
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		builder.db = tx
		return builder.BuildMissingBankDetailsQuery(domain.EmployeeFilters{}).
			Find(&[]domain.Employee{})
	})

	normalizedSQL := strings.ToLower(sql)
	assert.Contains(t, normalizedSQL, "employees.bank_id is null")
	assert.Contains(t, normalizedSQL, "coalesce(trim(employees.bank_account_number), '') = ''")
	assert.Contains(t, normalizedSQL, "coalesce(trim(employees.bank_account_name), '') = ''")
	assert.Contains(t, normalizedSQL, "employees.bank_account_status = 'invalid'")
	assert.NotContains(t, normalizedSQL, "employees.bank_account_status = 'unverified'")
}

func TestMissingBankDetailsPredicateHandlesNullAndEmptyValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE employees (
			id INTEGER PRIMARY KEY,
			bank_id INTEGER NULL,
			bank_account_number TEXT NULL,
			bank_account_name TEXT NULL,
			bank_account_status TEXT NULL
		)
	`).Error)

	rows := []struct {
		id            int
		bankID        any
		accountNumber any
		accountName   any
		status        any
	}{
		{1, 10, nil, "NGUYEN VAN A", "valid"},
		{2, 10, "123", nil, "valid"},
		{3, 10, "   ", "NGUYEN VAN A", "valid"},
		{4, 10, "123", "NGUYEN VAN A", "invalid"},
		{5, nil, "123", "NGUYEN VAN A", "valid"},
		{6, 10, "123", "NGUYEN VAN A", "unverified"},
		{7, 10, "123", "NGUYEN VAN A", "valid"},
	}
	for _, row := range rows {
		require.NoError(t, db.Exec(
			`INSERT INTO employees
				(id, bank_id, bank_account_number, bank_account_name, bank_account_status)
			 VALUES (?, ?, ?, ?, ?)`,
			row.id,
			row.bankID,
			row.accountNumber,
			row.accountName,
			row.status,
		).Error)
	}

	var ids []int
	require.NoError(t, db.Table("employees").
		Where(missingBankDetailsPredicate).
		Order("id").
		Pluck("id", &ids).Error)

	assert.Equal(t, []int{1, 2, 3, 4, 5}, ids)
}
