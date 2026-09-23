package seed

import (
	"context"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

func (s *Seeder) seedSettings(ctx context.Context, db *gorm.DB) error {
	settings := []struct {
		key       string
		value     string
		valueType string
	}{
		{"SourceAccountName", "Công ty TNHH Quản lý Nhân sự Việt Nam", "string"},
		{"SourceAccountNumber", "1234567890123", "string"},
		{"SourceBankName", "Ngân hàng TMCP Ngoại thương Việt Nam", "string"},
		{"SourceBankBranch", "Chi nhánh Hà Nội", "string"},
		{"DefaultCurrency", "₫", "string"},
		{"TaxRate", "10", "number"},
		{"CompanyName", "Công ty TNHH Quản lý Nhân sự Việt Nam", "string"},
		{"CompanyAddress", "123 đường Nguyễn Trãi, Quận Thanh Xuân, Hà Nội", "string"},
		{"CompanyPhone", "024-1234-5678", "string"},
		{"CompanyEmail", "info@payroll-company.vn", "string"},
		{"AutoApproveTimesheets", "false", "boolean"},
		{"MaxHoursPerDay", "12", "number"},
		{"MinHoursPerDay", "1", "number"},
		{"bulk_transfer_workbook_limit_vnd", "400000000", "number"},
		{"self_check_in_advance_percentage", "70", "number"},
		{"self_check_in_advance_hold_hours", "24", "number"},
		// Seeded so GET /settings/key/transfer_bank_visible stops 404ing for
		// environments where the admin has never saved the sao kê bank panel
		// (the settings form creates the row on first save).
		{"transfer_bank_visible", "true", "string"},
		// Same rationale for the FlexPay (OnePay) beneficiary block.
		{"flexpay_transfer_bank_visible", "true", "string"},
	}

	for _, setting := range settings {
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
