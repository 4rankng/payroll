package seed

import (
	"context"
	"fmt"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

func (s *Seeder) seedBanks(ctx context.Context, db *gorm.DB) error {
	for _, bank := range VietnameseBanks {
		bankEntity := &domain.Bank{
			BranchName: bank.Name,
		}

		if err := db.Where("branch_name = ?", bank.Name).FirstOrCreate(bankEntity).Error; err != nil {
			return fmt.Errorf("create bank %s: %w", bank.Name, err)
		}
	}
	return nil
}
