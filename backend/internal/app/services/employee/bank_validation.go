package employee

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strings"
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

// bankValidationCacheTTL bounds how long a OnePay verdict is reused to
// deduplicate calls for the same bank info within and across import
// jobs. 15 min covers within-chunk duplicates (rows processed seconds
// apart) and the re-upload race window (user re-uploads while a prior
// job is still running). Note: very large imports (>~2000 fully-distinct
// rows at 2 TPS) exceed this window, so cross-row duplicates processed
// >15 min apart will miss the cache and re-call OnePay — acceptable,
// since dedup is an optimization and the verdict is still correct.
const bankValidationCacheTTL = 15 * time.Minute

// bankValidationCacheKeyPrefix namespaces the dedup keys in Redis so
// they don't collide with other consumers of the shared cache.
const bankValidationCacheKeyPrefix = "bankval"

const bankAccountInvalidErrorCode = "BANK_ACCOUNT_INVALID"

// BankAccountValidator is the single DRY chokepoint that turns a set of
// employee bank fields into a persisted validation outcome by calling
// the active disbursement provider's AccountVerifier (OnePay today).
//
// It is intentionally side-effect free: it returns a ValidationResult
// that callers apply to the domain entity. This lets the four distinct
// write paths (manual create, manual update, XLSX import, BCC import via
// UpdateBankInfo) share identical semantics without re-implementing the
// provider-resolution / error-mapping logic.
//
// The optional cache deduplicates OnePay calls by bank info (swift code
// + account number + account holder name): when the same bank info is
// checked multiple times within bankValidationCacheTTL — across rows in
// a single import, or across overlapping import jobs — the cached
// verdict is reused and OnePay is not called again. A nil cache disables
// dedup (used in tests).
type BankAccountValidator struct {
	registry *disbursement.Registry
	bankRepo domain.BankRepository
	cache    domain.CacheServiceUseCase // nil = dedup disabled
	logger   *slog.Logger
}

// NewBankAccountValidator constructs a validator. registry may be nil in
// environments without a configured provider; Validate degrades to
// Status=valid in that case. cache may be nil to disable dedup.
func NewBankAccountValidator(registry *disbursement.Registry, bankRepo domain.BankRepository, cache domain.CacheServiceUseCase, logger *slog.Logger) *BankAccountValidator {
	if logger == nil {
		logger = slog.Default()
	}
	return &BankAccountValidator{
		registry: registry,
		bankRepo: bankRepo,
		cache:    cache,
		logger:   logger.With("component", "BankAccountValidator"),
	}
}

// cacheKey returns a deterministic Redis key from the three fields that
// uniquely define a OnePay account check: swift code, account number,
// and account holder name. Same inputs MUST return the same verdict, so
// they key the cache together. SHA-256 keeps the key compact and avoids
// leaking raw account numbers in the Redis keyspace.
func cacheKey(swiftCode, accountNo, accountName string) string {
	h := sha256.Sum256([]byte(strings.Join([]string{swiftCode, accountNo, accountName}, "|")))
	return fmt.Sprintf("%s:%x", bankValidationCacheKeyPrefix, h[:16])
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

	// Dedup: within the TTL window, reuse the cached verdict for the same
	// bank info. This collapses both within-import duplicates (same person
	// on multiple rows) and across-job duplicates (user re-uploads the same
	// Excel before the first job finishes) into a single OnePay call. Cache
	// failures (Redis down, decode error) fall through to OnePay — dedup is
	// an optimization, never a blocker.
	key := cacheKey(bank.SwiftCode, accountNo, accountName)
	if v.cache != nil {
		if cached, ok := readCachedResult(ctx, v.cache, key); ok {
			return cached
		}
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

	// Map the provider outcome to a ValidationResult (single point so the
	// cache-write below covers every branch).
	var r ValidationResult
	switch {
	case err != nil:
		// Network / 5xx / timeout → fail-open. Do NOT mark invalid: we
		// don't know the account is bad, only that we couldn't check.
		v.logger.Warn("account validation call failed; marking unverified",
			"bank_id", *bankID, "account_no", accountNo, "error", err)
		r = ValidationResult{Status: domain.BankAccountStatusUnverified}
	case res.Valid:
		r = ValidationResult{Status: domain.BankAccountStatusValid}
	default:
		r = ValidationResult{Status: domain.BankAccountStatusInvalid, Reason: translateInvalidReason(res)}
	}

	// Best-effort cache of the OnePay outcome so the next check of the
	// same bank info (within TTL) skips OnePay entirely. Errors are
	// ignored: a flaky Redis must not break the employee write.
	//
	// Only cache provider *answers* (valid / invalid). Do NOT cache
	// `unverified` (outage/timeout): a transient OnePay blip cached for 15
	// min would leave accounts silently unvalidated for up to 15 min after
	// OnePay recovers — defeating the fail-open design. The 2 TPS
	// accountLimiter already bounds how hard we hammer a down provider on
	// immediate retries, so skipping the unverified cache costs nothing in
	// outage protection.
	if v.cache != nil && r.Status != domain.BankAccountStatusUnverified {
		writeCachedResult(ctx, v.cache, key, r)
	}
	return r
}

// readCachedResult fetches a ValidationResult from the cache. Returns
// (zero, false) on miss or any error (decode failure, Redis down) so the
// caller falls through to a fresh OnePay call. The cache abstraction
// (CacheServiceUseCase.Get) JSON-unmarshals the stored payload directly
// into dest, so we hand it a pointer to the struct.
func readCachedResult(ctx context.Context, c domain.CacheServiceUseCase, key string) (ValidationResult, bool) {
	var r ValidationResult
	if err := c.Get(ctx, key, &r); err != nil {
		return ValidationResult{}, false
	}
	// Defensive: an empty Status would corrupt the entity. Treat as miss.
	if r.Status == "" {
		return ValidationResult{}, false
	}
	return r, true
}

// writeCachedResult stores a ValidationResult under key with the standard
// TTL. Errors are swallowed — dedup is best-effort. The cache abstraction
// (CacheServiceUseCase.Set) JSON-marshals the value itself, so we hand it
// the struct directly (double-encoding would corrupt readCachedResult).
func writeCachedResult(ctx context.Context, c domain.CacheServiceUseCase, key string, r ValidationResult) {
	_ = c.Set(ctx, key, r, bankValidationCacheTTL)
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

// validateManualBankAccount rejects a confirmed-invalid account before a
// single create or update reaches persistence. Provider outages remain
// fail-open as "unverified"; batch imports keep their separate allow-and-flag
// behavior through applyBankAccountValidation.
func validateManualBankAccount(ctx context.Context, v *BankAccountValidator, e *domain.Employee) error {
	applyBankAccountValidation(ctx, v, e)
	if e.BankAccountStatus != domain.BankAccountStatusInvalid {
		return nil
	}

	reason := "Tài khoản ngân hàng cần kiểm tra"
	if e.BankAccountInvalidReason != nil && *e.BankAccountInvalidReason != "" {
		reason = *e.BankAccountInvalidReason
	}

	return domain.NewValidationErrorWithCode(bankAccountInvalidErrorCode, reason).
		WithContext("attempted_account_number", e.BankAccountNumber)
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
