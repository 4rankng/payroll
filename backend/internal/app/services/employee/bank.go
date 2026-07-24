package employee

import (
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/domain"
	bankpkg "api-server/internal/pkg/bank"
	"api-server/internal/pkg/clock"
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
//
// As the single chokepoint for BCC-import bank updates, it also re-runs
// OnePay account validation and folds the outcome into the same column
// update so it is persisted atomically.
func (s *EmployeeService) UpdateBankInfo(ctx context.Context, employeeID uint, bankUpdates map[string]any) error {
	// Fold validation outcome into the column update if (a) the validator
	// is configured and (b) the caller is actually changing bank fields.
	if s.bankAccountValidator != nil && hasBankColumns(bankUpdates) {
		s.augmentBankUpdateWithValidation(ctx, employeeID, bankUpdates)
	}
	return s.EmployeeRepo.UpdateColumns(ctx, employeeID, bankUpdates)
}

// hasBankColumns reports whether the update map touches any of the three
// banking columns. Validation is only relevant when at least one is set.
func hasBankColumns(m map[string]any) bool {
	for _, k := range []string{"bank_id", "bank_account_number", "bank_account_name"} {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

// augmentBankUpdateWithValidation reads the current employee, applies the
// incoming bank columns on top to derive the effective values, runs the
// validator, and writes status/reason/validated_at into the map.
//
// Errors reading the employee are non-fatal: we simply skip validation
// (the bank update itself still proceeds).
func (s *EmployeeService) augmentBankUpdateWithValidation(ctx context.Context, employeeID uint, bankUpdates map[string]any) {
	current, err := s.EmployeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		slog.Warn("UpdateBankInfo: could not load employee for validation; skipping",
			"employee_id", employeeID, "error", err)
		return
	}

	// Derive effective bankID: incoming column if present, else current.
	var effectiveBankID *uint
	switch v := bankUpdates["bank_id"].(type) {
	case nil:
		effectiveBankID = current.BankID
	case uint:
		effectiveBankID = &v
	case *uint:
		effectiveBankID = v
	case int64:
		u := uint(v)
		effectiveBankID = &u
	default:
		// Unknown type (e.g. a future caller passing a JSON-decoded
		// float64). Fail-open to unverified rather than silently
		// validating against the OLD bank — a stale verdict is worse
		// than no verdict.
		slog.Warn("UpdateBankInfo: unexpected bank_id type in update map; marking unverified",
			"employee_id", employeeID, "type", fmt.Sprintf("%T", bankUpdates["bank_id"]))
		bankUpdates["bank_account_status"] = domain.BankAccountStatusUnverified
		bankUpdates["bank_account_invalid_reason"] = nil
		bankUpdates["bank_account_validated_at"] = clock.Now()
		return
	}

	// Derive effective account number / name: incoming column if present,
	// else current.
	effectiveAcctNo := current.BankAccountNumber
	if v, ok := bankUpdates["bank_account_number"]; ok {
		if s, ok := v.(string); ok {
			effectiveAcctNo = s
		}
	}
	effectiveAcctName := current.BankAccountName
	if v, ok := bankUpdates["bank_account_name"]; ok {
		if s, ok := v.(string); ok {
			effectiveAcctName = s
		}
	}

	r := s.bankAccountValidator.Validate(ctx, effectiveBankID, effectiveAcctNo, effectiveAcctName)
	bankUpdates["bank_account_status"] = r.Status
	if r.Reason != nil {
		bankUpdates["bank_account_invalid_reason"] = *r.Reason
	} else {
		// Clear any prior reason when status is valid/unverified.
		bankUpdates["bank_account_invalid_reason"] = nil
	}
	now := clock.Now()
	bankUpdates["bank_account_validated_at"] = now
}

// UpdateMobile sets the employee's mobile number (targeted update to avoid
// full Save overwriting concurrent changes).
func (s *EmployeeService) UpdateMobile(ctx context.Context, employeeID uint, mobile string) error {
	if mobile == "" {
		return nil
	}
	return s.EmployeeRepo.UpdateColumns(ctx, employeeID, map[string]any{"mobile": mobile})
}
