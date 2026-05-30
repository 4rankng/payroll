package seed

import (
	"context"

	"api-server/internal/domain"
	"gorm.io/gorm"
)

// seedAccounts creates the standard chart of accounts
func (s *Seeder) seedAccounts(ctx context.Context, tx *gorm.DB) error {
	accounts := []domain.Account{
		// Assets
		{Code: "1000", Name: "Cash", Type: domain.CategoryAsset},
		{Code: "1100", Name: "Accounts Receivable", Type: domain.CategoryAsset},

		// Liabilities
		{Code: "2000", Name: "Accounts Payable", Type: domain.CategoryLiability},
		{Code: "2100", Name: "Loans Payable", Type: domain.CategoryLiability},

		// Equity
		{Code: "5000", Name: "Owner's Equity", Type: domain.CategoryEquity},

		// Revenue
		{Code: "4000", Name: "Service Revenue", Type: domain.CategoryRevenue},

		// Expenses
		{Code: "3000", Name: "Operating Expenses", Type: domain.CategoryExpense},
	}

	for _, account := range accounts {
		var existing domain.Account
		err := tx.Where("code = ?", account.Code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := tx.Create(&account).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		// Account exists, skip
	}

	return nil
}
