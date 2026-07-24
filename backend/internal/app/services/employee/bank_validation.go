package employee

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain"
	infrastructure "api-server/internal/domain/ports/infrastructure"
	"api-server/internal/pkg/clock"
)

// bankValidationTimeout is the per-call deadline for OnePay account
// verification. We deliberately keep this short: validation runs inline
// on every employee create/update (including inside bulk-import asynq
// jobs), so a hung provider must not stall the whole pipeline. On expiry
// we fail open (BankAccountStatusUnverified) rather than blocking the
// write.
const bankValidationTimeout = 5 * time.Second

// accountTypeBankAccount is the AccountCheckRequest.AccountType value
// for a bank-account (as opposed to card) lookup. Matches the literal
// used by manual_disbursement_handler.CheckAccount ("0").
const accountTypeBankAccount = "0"

// BankAccountValidator is the single DRY chokepoint that turns a set of
// employee bank fields into a persisted validation outcome by calling
// the active disbursement provider's AccountVerifier (OnePay today).
//
// It is intentionally side-effect free: it returns a ValidationResult
// that callers apply to the domain entity. This lets the four distinct
// write paths (manual create, manual update, XLSX import, BCC import via
// UpdateBankInfo) share identical semantics without re-implementing the
// provider-resolution / error-mapping logic.
type BankAccountValidator struct {
	registry *disbursement.Registry
	bankRepo domain.BankRepository
	logger   *slog.Logger
}

// NewBankAccountValidator constructs a validator. registry may be nil in
// environments without a configured provider; Validate degrades to
// Status=valid in that case.
func NewBankAccountValidator(registry *disbursement.Registry, bankRepo domain.BankRepository, logger *slog.Logger) *BankAccountValidator {
	if logger == nil {
		logger = slog.Default()
	}
	return &BankAccountValidator{
		registry: registry,
		bankRepo: bankRepo,
		logger:   logger.With("component", "BankAccountValidator"),
	}
}

// ValidationResult is the outcome of validating one account.
//
//   - Status == BankAccountStatusValid: account confirmed good (or
//     validation was skipped because there is nothing to validate).
//   - Status == BankAccountStatusInvalid: OnePay confirmed the account
//     is bad or the holder name mismatches; Reason holds a Vietnamese
//     human-readable string for the UI.
//   - Status == BankAccountStatusUnverified: OnePay was unreachable or
//     timed out; Reason is nil. The employee is NOT surfaced in the
//     warning list (fail-open).
type ValidationResult struct {
	Status string
	Reason *string
}

// okValidation is the zero-cost "nothing to check" result returned when
// banking info is incomplete or no provider is configured.
var okValidation = ValidationResult{Status: domain.BankAccountStatusValid}

// Validate resolves the bank's SWIFT code, calls the active provider's
// AccountVerifier (with a 5s timeout), and maps the outcome. It never
// returns an error: any provider failure is translated into
// BankAccountStatusUnverified so that the calling write path can
// proceed (decision: allow + flag, fail-open on provider outage).
//
// Callers MUST treat a nil receiver as "validation disabled" and skip
// the call (see applyBankAccountValidation helper).
func (v *BankAccountValidator) Validate(ctx context.Context, bankID *uint, accountNo, accountName string) ValidationResult {
	// Nothing to validate: the "missing banking info" case is handled by
	// the warning-list SQL on its own. Here we leave the default "valid"
	// status so the row is not double-flagged.
	if bankID == nil || accountNo == "" || accountName == "" {
		return okValidation
	}

	// No registry / no registered provider → degrade to valid. This keeps
	// dev sandboxes (9Pay without AccountVerifier) and provider-less
	// environments working.
	if v == nil || v.registry == nil || len(v.registry.Names()) == 0 {
		return okValidation
	}

	bank, err := v.bankRepo.GetByID(ctx, *bankID)
	if err != nil {
		// Bank row missing is unexpected (the service layer already
		// validates the FK before calling us). Fail-open rather than
		// blocking the write.
		v.logger.Warn("bank lookup failed during account validation; marking unverified",
			"bank_id", *bankID, "error", err)
		return ValidationResult{Status: domain.BankAccountStatusUnverified}
	}
	if bank.SwiftCode == "" {
		reason := "Không có mã SWIFT của ngân hàng, không thể kiểm tra"
		return ValidationResult{Status: domain.BankAccountStatusInvalid, Reason: &reason}
	}

	provider, err := v.registry.Active(ctx)
	if err != nil {
		v.logger.Warn("no active disbursement provider; skipping account validation",
			"error", err)
		return okValidation
	}
	verifier, ok := provider.(infrastructure.AccountVerifier)
	if !ok {
		// Active provider doesn't implement account verification
		// (e.g. 9Pay). Degrade to valid.
		return okValidation
	}

	// Hard 5s timeout so a hung OnePay cannot stall a bulk import.
	checkCtx, cancel := context.WithTimeout(ctx, bankValidationTimeout)
	defer cancel()

	res, err := verifier.CheckAccount(checkCtx, infrastructure.AccountCheckRequest{
		RequestID:   fmt.Sprintf("empval%d", time.Now().UnixNano()),
		SwiftCode:   bank.SwiftCode,
		BankCode:    bank.BankCode,
		AccountNo:   accountNo,
		AccountName: accountName,
		AccountType: accountTypeBankAccount,
	})
	if err != nil {
		// Network / 5xx / timeout → fail-open. Do NOT mark invalid: we
		// don't know the account is bad, only that we couldn't check.
		v.logger.Warn("account validation call failed; marking unverified",
			"bank_id", *bankID, "account_no", accountNo, "error", err)
		return ValidationResult{Status: domain.BankAccountStatusUnverified}
	}

	if res.Valid {
		return ValidationResult{Status: domain.BankAccountStatusValid}
	}

	return ValidationResult{Status: domain.BankAccountStatusInvalid, Reason: translateInvalidReason(res)}
}

// translateInvalidReason maps a provider AccountCheckResult to a
// Vietnamese human-readable reason string for the warning list UI.
func translateInvalidReason(res *infrastructure.AccountCheckResult) *string {
	switch res.RawErrorCode {
	case "name_mismatch":
		msg := "Tên chủ tài khoản không khớp với ngân hàng"
		if res.AccountName != "" {
			msg = fmt.Sprintf("%s (ngân hàng ghi: %s)", msg, res.AccountName)
		}
		return &msg
	default:
		msg := "Số tài khoản không hợp lệ"
		if res.RawMessage != "" {
			msg = fmt.Sprintf("%s: %s", msg, res.RawMessage)
		}
		return &msg
	}
}

// applyBankAccountValidation runs validation (if enabled) and writes the
// three outcome fields onto the employee entity. It is the shared helper
// used by every write touchpoint, so the field-setting stays identical
// across manual create / update and the import paths.
//
// A nil validator means the feature is disabled (e.g. in tests); the
// entity is left untouched and retains its default "valid" status.
func applyBankAccountValidation(ctx context.Context, v *BankAccountValidator, e *domain.Employee) {
	if v == nil {
		return
	}
	r := v.Validate(ctx, e.BankID, e.BankAccountNumber, e.BankAccountName)
	e.BankAccountStatus = r.Status
	e.BankAccountInvalidReason = r.Reason
	now := clock.Now()
	e.BankAccountValidatedAt = &now
}

// bankFieldsChanged reports whether any of the three banking fields
// differ between the original (pre-update) and incoming employee. Used
// by UpdateEmployee to decide whether a fresh OnePay call is warranted.
func bankFieldsChanged(original, updated *domain.Employee) bool {
	if original == nil {
		return true
	}
	oBankID, uBankID := original.BankID, updated.BankID
	if (oBankID == nil) != (uBankID == nil) {
		return true
	}
	if oBankID != nil && uBankID != nil && *oBankID != *uBankID {
		return true
	}
	return original.BankAccountNumber != updated.BankAccountNumber ||
		original.BankAccountName != updated.BankAccountName
}

// rowBankFieldsPresent reports whether an import row carries any banking
// data. Used to decide whether the XLSX-import update path needs to
// re-run OnePay validation.
func rowBankFieldsPresent(row dto.EmployeeImportRow) bool {
	return row.BankAccount != "" || row.BankAccountName != "" || row.BankName != ""
}
