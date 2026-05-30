package seed

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence"

	"gorm.io/gorm"
)

func (s *Seeder) seedLedgerEntries(ctx context.Context, db *gorm.DB) error {
	ledgerRepo := persistence.NewLedgerEntryRepository(&persistence.Database{DB: db})

	var adminUser domain.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		return fmt.Errorf("find admin user: %w", err)
	}

	// Create simple seed data with only 3 accounts: cash, receivable, payable
	startDate := clock.Now().AddDate(-1, 0, 0)

	// Initial cash entry
	initialCashEntry := &domain.LedgerEntry{
		Date:      startDate,
		Account:   domain.AccountCash,
		Party:     "Bank",
		Debit:     5000000000,
		Credit:    0,
		Balance:   5000000000,
		CreatedBy: adminUser.ID,
	}

	if err := ledgerRepo.Create(ctx, initialCashEntry); err != nil {
		return fmt.Errorf("create initial cash entry: %w", err)
	}

	// Sample receivable entry
	receivableEntry := &domain.LedgerEntry{
		Date:      startDate.AddDate(0, 1, 0),
		Account:   domain.AccountReceivable,
		Party:     "Client ABC",
		Debit:     100000000,
		Credit:    0,
		Balance:   0,
		CreatedBy: adminUser.ID,
	}

	if err := ledgerRepo.Create(ctx, receivableEntry); err != nil {
		return fmt.Errorf("create receivable entry: %w", err)
	}

	// Sample payable entry
	payableEntry := &domain.LedgerEntry{
		Date:      startDate.AddDate(0, 1, 5),
		Account:   domain.AccountPayable,
		Party:     "Supplier XYZ",
		Debit:     0,
		Credit:    50000000,
		Balance:   0,
		CreatedBy: adminUser.ID,
	}

	if err := ledgerRepo.Create(ctx, payableEntry); err != nil {
		return fmt.Errorf("create payable entry: %w", err)
	}

	return nil
}
