package seed

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	mathrand "math/rand/v2"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

func (s *Seeder) seedAssets(_ context.Context, db *gorm.DB) error {
	var users []domain.User
	if err := db.Find(&users).Error; err != nil {
		return fmt.Errorf("find users: %w", err)
	}

	assetData := []struct {
		filename   string
		uploadType string
	}{
		{"receipt_001.pdf", "ledger_evidence"},
		{"receipt_002.jpg", "ledger_evidence"},
		{"invoice_003.pdf", "ledger_evidence"},
		{"contract_001.pdf", "document"},
		{"project_plan_001.docx", "document"},
		{"employee_handbook.pdf", "general"},
		{"company_logo.png", "general"},
		{"bank_statement.pdf", "ledger_evidence"},
		{"salary_slip.pdf", "document"},
		{"timesheet_export.xlsx", "general"},
	}

	for _, data := range assetData {
		uploader := users[mathrand.IntN(len(users))]

		filePath := fmt.Sprintf("uploads/%s/%s", clock.Now().Format("2006/01"), data.filename)

		asset := domain.Asset{
			Filename:   data.filename,
			FilePath:   filePath,
			UploadType: data.uploadType,
			UploadedBy: uploader.ID,
			CreatedAt:  clock.Now().AddDate(0, 0, -mathrand.IntN(60)),
		}

		if err := db.Create(&asset).Error; err != nil {
			return fmt.Errorf("create asset %s: %w", data.filename, err)
		}
	}

	return nil
}
