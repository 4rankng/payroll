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
	if s.bankAccountValidator == nil || !hasBankColumns(bankUpdates) {
		return s.EmployeeRepo.UpdateColumns(ctx, employeeID, bankUpdates)
	}

	// Validate against a snapshot without holding a database lock across the
	// external provider call.
	current, err := s.EmployeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		slog.Error("UpdateBankInfo: could not load employee for validation",
			"employee_id", employeeID, "error", err)
		return fmt.Errorf("không thể xác minh thông tin ngân hàng: %w", err)
	}
	if err := s.augmentBankUpdateWithValidation(ctx, current, bankUpdates); err != nil {
		return err
	}

	// BCC jobs already carry a workbook-wide transaction in ctx. Start a small,
	// independent transaction for the financial tuple so the row lock is held
	// only for the compare-and-write window, not for the rest of the workbook.
	writeCtx := context.WithValue(ctx, domain.TransactionContextKey{}, nil)
	write := func(txCtx context.Context) error {
		locked, err := s.EmployeeRepo.GetByIDForUpdate(txCtx, employeeID)
		if err != nil {
			return fmt.Errorf("không thể kiểm tra lại thông tin ngân hàng: %w", err)
		}
		if !sameBankAccountTuple(current, locked) {
			return domain.NewConflictError(
				"Thông tin ngân hàng vừa được thay đổi. Vui lòng thử lại.",
			)
		}
		return s.EmployeeRepo.UpdateColumns(txCtx, employeeID, bankUpdates)
	}
	if s.TransactionManager == nil {
		// Unit-test and narrowly wired service fallback. Production always
		// supplies a transaction manager.
		return write(writeCtx)
	}
	return s.TransactionManager.WithTransaction(writeCtx, write)
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

// augmentBankUpdateWithValidation applies incoming bank columns on top of a
// previously read snapshot, runs the validator, and writes the verdict into
// the same update map.
func (s *EmployeeService) augmentBankUpdateWithValidation(
	ctx context.Context,
	current *domain.Employee,
	bankUpdates map[string]any,
) error {
	// Derive effective bankID: incoming column if present, else current.
	var effectiveBankID *uint
	incomingBankID, hasIncomingBankID := bankUpdates["bank_id"]
	if !hasIncomingBankID {
		effectiveBankID = current.BankID
	} else {
		switch v := incomingBankID.(type) {
		case nil:
			effectiveBankID = nil
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
				"employee_id", current.ID, "type", fmt.Sprintf("%T", incomingBankID))
			bankUpdates["bank_account_status"] = domain.BankAccountStatusUnverified
			bankUpdates["bank_account_invalid_reason"] = nil
			bankUpdates["bank_account_validated_at"] = clock.Now()
			return nil
		}
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
	return nil
}

func sameBankAccountTuple(expected, actual *domain.Employee) bool {
	if expected == nil || actual == nil {
		return false
	}
	if (expected.BankID == nil) != (actual.BankID == nil) {
		return false
	}
	if expected.BankID != nil && *expected.BankID != *actual.BankID {
		return false
	}
	return expected.BankAccountNumber == actual.BankAccountNumber &&
		expected.BankAccountName == actual.BankAccountName
}

// UpdateMobile sets the employee's mobile number (targeted update to avoid
// full Save overwriting concurrent changes).
func (s *EmployeeService) UpdateMobile(ctx context.Context, employeeID uint, mobile string) error {
	if mobile == "" {
		return nil
	}
	return s.EmployeeRepo.UpdateColumns(ctx, employeeID, map[string]any{"mobile": mobile})
}
