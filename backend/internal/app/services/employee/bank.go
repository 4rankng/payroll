package employee

import (
	"context"
	"log/slog"

	bankpkg "api-server/internal/pkg/bank"
)

// ResolveBankID maps a raw bank name (from Excel/import) to a database bank ID.
// Returns nil if the bank cannot be resolved (non-fatal).
func (s *EmployeeService) ResolveBankID(ctx context.Context, bankName string) *uint {
	if bankName == "" {
		return nil
	}
	mappedName := bankpkg.MapName(bankName)
	if mappedName == "" {
		return nil
	}
	banks, err := s.BankRepo.SearchByBranchName(ctx, mappedName, 1)
	if err != nil {
		slog.Warn("bank search failed", "mapped_name", mappedName, "original", bankName, "error", err)
		return nil
	}
	if len(banks) == 0 {
		slog.Warn("bank not found", "mapped_name", mappedName, "original", bankName)
		return nil
	}
	return &banks[0].ID
}

// UpdateBankInfo updates bank-related fields for an employee (targeted update).
// Uses column-level update to avoid full Save overwriting concurrent changes.
func (s *EmployeeService) UpdateBankInfo(ctx context.Context, employeeID uint, bankUpdates map[string]any) error {
	return s.EmployeeRepo.UpdateColumns(ctx, employeeID, bankUpdates)
}

// UpdateMobile sets the employee's mobile number (targeted update to avoid
// full Save overwriting concurrent changes).
func (s *EmployeeService) UpdateMobile(ctx context.Context, employeeID uint, mobile string) error {
	if mobile == "" {
		return nil
	}
	return s.EmployeeRepo.UpdateColumns(ctx, employeeID, map[string]any{"mobile": mobile})
}
