package seed

import (
	"context"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

// settingRow is one default row for the settings table.
type settingRow struct {
	key       string
	value     string
	valueType string
}

// bootFlags are the settings rows the server ensures at every boot
// (bootstrap.NewContainer → EnsureSettings). These flags are read by clients
// before an admin ever saves the owning panel, and GET /settings/key/{key}
// answers 404 for a missing row — the exact shape that fed the
// high-error-rate alert for employee.mobile_required (85× 404 in 24h). Every
// seeded default equals the documented missing-row behavior, so creating the
// row never changes what clients see, and FirstOrCreate never overwrites an
// admin-saved value.
var bootFlags = []settingRow{
	// Missing row = OFF (docs/api.md). Seeded so the Add-Employee sheet's
	// first GET reads 200 "false" instead of 404.
	{key: "employee.mobile_required", value: "false", valueType: "boolean"},
	// Seeded so GET /settings/key/transfer_bank_visible stops 404ing for
	// environments where the admin has never saved the sao kê bank panel
	// (the settings form creates the row on first save; its missing-row
	// default is visible=true, matching this value).
	{key: "transfer_bank_visible", value: "true", valueType: "string"},
	// Same rationale for the FlexPay (OnePay) beneficiary block.
	{key: "flexpay_transfer_bank_visible", value: "true", valueType: "string"},
}

// devDefaults are placeholder values created only by GenerateTestData in
// dev/test environments. They encode business defaults (source account, tax
// rate, alert thresholds), so boot must never write them into a production
// database — EnsureSettings intentionally covers bootFlags only.
var devDefaults = []settingRow{
	{key: "SourceAccountName", value: "Công ty TNHH Quản lý Nhân sự Việt Nam", valueType: "string"},
	{key: "SourceAccountNumber", value: "1234567890123", valueType: "string"},
	{key: "SourceBankName", value: "Ngân hàng TMCP Ngoại thương Việt Nam", valueType: "string"},
	{key: "SourceBankBranch", value: "Chi nhánh Hà Nội", valueType: "string"},
	{key: "DefaultCurrency", value: "₫", valueType: "string"},
	{key: "TaxRate", value: "10", valueType: "number"},
	{key: "CompanyName", value: "Công ty TNHH Quản lý Nhân sự Việt Nam", valueType: "string"},
	{key: "CompanyAddress", value: "123 đường Nguyễn Trãi, Quận Thanh Xuân, Hà Nội", valueType: "string"},
	{key: "CompanyPhone", value: "024-1234-5678", valueType: "string"},
	{key: "CompanyEmail", value: "info@payroll-company.vn", valueType: "string"},
	{key: "AutoApproveTimesheets", value: "false", valueType: "boolean"},
	{key: "MaxHoursPerDay", value: "12", valueType: "number"},
	{key: "MinHoursPerDay", value: "1", valueType: "number"},
	{key: "bulk_transfer_workbook_limit_vnd", value: "400000000", valueType: "number"},
	{key: "self_check_in_advance_percentage", value: "70", valueType: "number"},
	{key: "self_check_in_advance_hold_hours", value: "24", valueType: "number"},
	{key: "wallet_balance_alert_threshold_vnd", value: "50000000", valueType: "number"},
}

// EnsureSettings creates any missing boot-flag settings row. Idempotent:
// existing rows (including admin-saved values) are left untouched. Called at
// boot; callers treat a failure as best-effort and must not block startup.
func EnsureSettings(ctx context.Context, db *gorm.DB) error {
	return NewSeeder(db).upsertSettings(ctx, db, bootFlags)
}

func (s *Seeder) seedSettings(ctx context.Context, db *gorm.DB) error {
	return s.upsertSettings(ctx, db, append(append([]settingRow{}, bootFlags...), devDefaults...))
}

func (s *Seeder) upsertSettings(ctx context.Context, db *gorm.DB, rows []settingRow) error {
	for _, setting := range rows {
		if err := db.Where("`key` = ?", setting.key).FirstOrCreate(&domain.Settings{
			Key:       setting.key,
			Value:     &setting.value,
			ValueType: domain.SettingsValueType(setting.valueType),
		}).Error; err != nil {
			return fmt.Errorf("create setting %s: %w", setting.key, err)
		}
	}

	return nil
}
