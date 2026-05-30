package seed

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type Seeder struct {
	db *gorm.DB
}

func NewSeeder(db *gorm.DB) *Seeder {
	return &Seeder{db: db}
}

// GenerateTestData generates comprehensive test data for the application
func (s *Seeder) GenerateTestData(ctx context.Context) error {
	fmt.Println("Generating realistic test data...")

	return s.db.Transaction(func(tx *gorm.DB) error {
		fmt.Println("  - Creating banks...")
		if err := s.seedBanks(ctx, tx); err != nil {
			return fmt.Errorf("seed banks: %w", err)
		}

		fmt.Println("  - Creating partner users...")
		if err := s.seedPartnerUsers(ctx, tx); err != nil {
			return fmt.Errorf("seed partner users: %w", err)
		}

		fmt.Println("  - Creating projects...")
		if err := s.seedProjects(ctx, tx); err != nil {
			return fmt.Errorf("seed projects: %w", err)
		}

		fmt.Println("  - Creating employees...")
		if err := s.seedEmployees(ctx, tx); err != nil {
			return fmt.Errorf("seed employees: %w", err)
		}

		fmt.Println("  - Assigning employees to projects...")
		if err := s.seedProjectEmployees(ctx, tx); err != nil {
			return fmt.Errorf("seed project employees: %w", err)
		}

		fmt.Println("  - Creating payrates...")
		if err := s.seedPayrates(ctx, tx); err != nil {
			return fmt.Errorf("seed payrates: %w", err)
		}

		fmt.Println("  - Creating timesheets...")
		if err := s.seedTimesheets(ctx, tx); err != nil {
			return fmt.Errorf("seed timesheets: %w", err)
		}

		fmt.Println("  - Creating ledger entries...")
		if err := s.seedLedgerEntries(ctx, tx); err != nil {
			return fmt.Errorf("seed ledger entries: %w", err)
		}

		fmt.Println("  - Creating system settings...")
		if err := s.seedSettings(ctx, tx); err != nil {
			return fmt.Errorf("seed settings: %w", err)
		}

		fmt.Println("  - Creating notifications...")
		if err := s.seedNotifications(ctx, tx); err != nil {
			return fmt.Errorf("seed notifications: %w", err)
		}

		fmt.Println("  - Creating sample assets...")
		if err := s.seedAssets(ctx, tx); err != nil {
			return fmt.Errorf("seed assets: %w", err)
		}

		return nil
	})
}
