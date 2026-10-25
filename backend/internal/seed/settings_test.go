package seed

import (
	"context"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Raw DDL: the model's MySQL enum('string',...) column type is not valid
	// sqlite. Column shapes only — FirstOrCreate behavior is type-agnostic.
	require.NoError(t, db.Exec(`CREATE TABLE settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		` + "`key`" + ` varchar(100) NOT NULL UNIQUE,
		value TEXT,
		value_type varchar(10) NOT NULL DEFAULT 'string',
		deleted_at DATETIME,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`).Error)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func settingsValue(t *testing.T, db *gorm.DB, key string) (string, bool) {
	t.Helper()
	var row domain.Settings
	err := db.Where("`key` = ?", key).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return "", false
	}
	require.NoError(t, err)
	require.NotNil(t, row.Value)
	return *row.Value, true
}

// EnsureSettings exists so a never-saved flag reads 200 with its documented
// missing-row default instead of 404 (the shape that fed the
// high-error-rate alert for employee.mobile_required).
func TestEnsureSettingsCreatesBootFlagsWithMissingRowDefaults(t *testing.T) {
	db := newSettingsTestDB(t)

	require.NoError(t, EnsureSettings(context.Background(), db))

	value, ok := settingsValue(t, db, "employee.mobile_required")
	require.True(t, ok)
	require.Equal(t, "false", value)

	value, ok = settingsValue(t, db, "transfer_bank_visible")
	require.True(t, ok)
	require.Equal(t, "true", value)

	value, ok = settingsValue(t, db, "flexpay_transfer_bank_visible")
	require.True(t, ok)
	require.Equal(t, "true", value)
}

// Boot must not write business placeholder defaults into a production
// database; those stay GenerateTestData-only.
func TestEnsureSettingsDoesNotSeedDevDefaults(t *testing.T) {
	db := newSettingsTestDB(t)

	require.NoError(t, EnsureSettings(context.Background(), db))

	_, ok := settingsValue(t, db, "TaxRate")
	require.False(t, ok)
	_, ok = settingsValue(t, db, "wallet_balance_alert_threshold_vnd")
	require.False(t, ok)
}

// FirstOrCreate semantics: an admin-saved value (e.g. mobile_required=true,
// the live prod row) must survive a reboot.
func TestEnsureSettingsPreservesExistingValue(t *testing.T) {
	db := newSettingsTestDB(t)
	required := "true"
	require.NoError(t, db.Create(&domain.Settings{
		Key:       "employee.mobile_required",
		Value:     &required,
		ValueType: domain.ValueTypeBoolean,
	}).Error)

	require.NoError(t, EnsureSettings(context.Background(), db))

	value, ok := settingsValue(t, db, "employee.mobile_required")
	require.True(t, ok)
	require.Equal(t, "true", value)
}
