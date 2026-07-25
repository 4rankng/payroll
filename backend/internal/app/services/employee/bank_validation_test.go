package employee

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain"
	infrastructure "api-server/internal/domain/ports/infrastructure"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- test doubles ------------------------------------------------------------

// verifierProvider is a minimal DisbursementProvider that also implements
// AccountVerifier. It lets us drive BankAccountValidator.Validate without
// hitting the real OnePay sandbox.
type verifierProvider struct {
	name      string
	checkFunc func(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error)
}

func (p *verifierProvider) Name() string { return p.name }
func (p *verifierProvider) InitiateTransfer(context.Context, infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	return nil, nil
}
func (p *verifierProvider) VerifyAndParseWebhook(context.Context, map[string]any) (*infrastructure.WebhookEvent, error) {
	return nil, nil
}
func (p *verifierProvider) CheckAccount(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
	if p.checkFunc == nil {
		return &infrastructure.AccountCheckResult{Valid: true}, nil
	}
	return p.checkFunc(ctx, req)
}

// Compile-time assertions that verifierProvider satisfies both interfaces.
var (
	_ infrastructure.DisbursementProvider = (*verifierProvider)(nil)
	_ infrastructure.AccountVerifier      = (*verifierProvider)(nil)
)

// noVerifierProvider implements only DisbursementProvider (no AccountVerifier),
// exercising the graceful-degradation path.
type noVerifierProvider struct{ name string }

func (p *noVerifierProvider) Name() string { return p.name }
func (p *noVerifierProvider) InitiateTransfer(context.Context, infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	return nil, nil
}
func (p *noVerifierProvider) VerifyAndParseWebhook(context.Context, map[string]any) (*infrastructure.WebhookEvent, error) {
	return nil, nil
}

var _ infrastructure.DisbursementProvider = (*noVerifierProvider)(nil)

// stubBankRepo overrides only GetByID (the method Validate uses).
type stubBankRepo struct {
	domain.BankRepository
	byID map[uint]*domain.Bank
	err  error
}

func (s *stubBankRepo) GetByID(_ context.Context, id uint) (*domain.Bank, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.byID[id], nil
}

// stubCache is a minimal in-memory CacheServiceUseCase for dedup tests.
// Embeds the interface so only Get/Set are exercised; the rest no-op.
// stubCache is a minimal in-memory CacheServiceUseCase for dedup tests.
// Goroutine-safe (sync.Mutex) so it can back concurrent dedup tests under
// -race. Embeds the interface so only Get/Set are exercised.
type stubCache struct {
	domain.CacheServiceUseCase
	mu         sync.Mutex
	store      map[string]string // raw JSON payloads
	getErr     error
	setErr     error
	setCallCnt int
}

func newStubCache() *stubCache {
	return &stubCache{store: map[string]string{}}
}

func (s *stubCache) Get(_ context.Context, key string, dest interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getErr != nil {
		return s.getErr
	}
	raw, ok := s.store[key]
	if !ok {
		return fmt.Errorf("not found")
	}
	return json.Unmarshal([]byte(raw), dest)
}

func (s *stubCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setCallCnt++
	if s.setErr != nil {
		return s.setErr
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.store[key] = string(payload)
	return nil
}

// setCallCount exposes the write counter under the lock for assertions.
func (s *stubCache) setCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setCallCnt
}

// resetSetCallCount zeroes the write counter under the lock.
func (s *stubCache) resetSetCallCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setCallCnt = 0
}

// seed stores a ValidationResult under the dedup key for the given bank
// info, mirroring what Validate's writeCachedResult would produce. Routes
// through Set so the JSON encoding matches the read path exactly.
func (s *stubCache) seed(swiftCode, accountNo, accountName string, r ValidationResult) {
	_ = s.Set(context.Background(), cacheKey(swiftCode, accountNo, accountName), r, 0)
}

// hasEntry reports whether a cached verdict exists for the bank info.
func (s *stubCache) hasEntry(swiftCode, accountNo, accountName string) bool {
	_, ok := s.store[cacheKey(swiftCode, accountNo, accountName)]
	return ok
}

func newTestValidator(p infrastructure.DisbursementProvider, bank *domain.Bank, bankErr error) *BankAccountValidator {
	return newTestValidatorWithCache(p, bank, bankErr, nil)
}

func newTestValidatorWithCache(p infrastructure.DisbursementProvider, bank *domain.Bank, bankErr error, cache domain.CacheServiceUseCase) *BankAccountValidator {
	reg := disbursement.NewRegistry()
	if p != nil {
		reg.Register(p)
	}
	repo := &stubBankRepo{}
	if bank != nil {
		repo.byID = map[uint]*domain.Bank{bank.ID: bank}
	}
	repo.err = bankErr
	return NewBankAccountValidator(reg, repo, cache, nil)
}

func uintPtr(u uint) *uint { return &u }

// --- Validate ---------------------------------------------------------------

func TestValidate_NoBankingInfo_SkipsValidation(t *testing.T) {
	v := newTestValidator(nil, nil, nil) // no provider registered
	id := uint(1)
	r := v.Validate(context.Background(), nil, "", "")
	assert.Equal(t, domain.BankAccountStatusValid, r.Status)
	assert.Nil(t, r.Reason)

	// Partial info also short-circuits (no account number).
	r = v.Validate(context.Background(), &id, "", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusValid, r.Status)
}

func TestValidate_NoProvider_DegradesToValid(t *testing.T) {
	// Empty registry → degrade to valid, no OnePay call.
	v := newTestValidator(nil, nil, nil)
	id := uint(1)
	r := v.Validate(context.Background(), &id, "123456789", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusValid, r.Status)
}

func TestValidate_BankMissingSwiftCode_Invalid(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: ""}
	v := newTestValidator(&verifierProvider{name: "onepay"}, bank, nil)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusInvalid, r.Status)
	if r.Reason == nil {
		t.Fatal("expected a reason")
	}
	assert.Contains(t, *r.Reason, "SWIFT")
}

func TestValidate_BankLookupError_Unverified(t *testing.T) {
	id := uint(1)
	v := newTestValidator(&verifierProvider{name: "onepay"}, nil, errors.New("db down"))

	r := v.Validate(context.Background(), &id, "123456789", "Nguyen Van A")
	// Fail-open: bank lookup error → unverified, NOT invalid.
	assert.Equal(t, domain.BankAccountStatusUnverified, r.Status)
	assert.Nil(t, r.Reason)
}

func TestValidate_ProviderSaysValid(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX", BankCode: "970436"}
	p := &verifierProvider{name: "onepay"} // default CheckAccount returns Valid:true
	v := newTestValidator(p, bank, nil)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusValid, r.Status)
	assert.Nil(t, r.Reason)
}

func TestValidate_NameMismatch_InvalidWithVietnameseReason(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	p := &verifierProvider{
		name: "onepay",
		checkFunc: func(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
			return &infrastructure.AccountCheckResult{
				Valid:        false,
				AccountName:  "Tran Thi B",
				RawErrorCode: "name_mismatch",
				RawMessage:   "holder name differs",
			}, nil
		},
	}
	v := newTestValidator(p, bank, nil)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusInvalid, r.Status)
	if r.Reason == nil {
		t.Fatal("expected a reason")
	}
	assert.Contains(t, *r.Reason, "không khớp")
	assert.Contains(t, *r.Reason, "Tran Thi B") // bank-confirmed name surfaced
}

func TestValidate_AccountNotFound_InvalidGenericReason(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	p := &verifierProvider{
		name: "onepay",
		checkFunc: func(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
			return &infrastructure.AccountCheckResult{
				Valid:        false,
				RawErrorCode: "account_not_found",
				RawMessage:   "no such account",
			}, nil
		},
	}
	v := newTestValidator(p, bank, nil)

	r := v.Validate(context.Background(), &bank.ID, "000000000", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusInvalid, r.Status)
	if r.Reason == nil {
		t.Fatal("expected a reason")
	}
	assert.Equal(t, "Số tài khoản không hợp lệ", *r.Reason)
	assert.NotContains(t, *r.Reason, "no such account")
}

func TestValidateManualBankAccount_ConfirmedInvalidReturnsTypedError(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	p := &verifierProvider{
		name: "onepay",
		checkFunc: func(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
			return &infrastructure.AccountCheckResult{
				Valid:        false,
				RawErrorCode: "account_not_found",
				RawMessage:   "Invalid account info",
			}, nil
		},
	}
	v := newTestValidator(p, bank, nil)
	employee := &domain.Employee{
		BankID:            &bank.ID,
		BankAccountNumber: "000000000",
		BankAccountName:   "Nguyen Van A",
	}

	err := validateManualBankAccount(context.Background(), v, employee)

	var domainErr *domain.DomainError
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, bankAccountInvalidErrorCode, domainErr.Code)
	assert.Equal(t, "Số tài khoản không hợp lệ", domainErr.Message)
	assert.Equal(t, domain.BankAccountStatusInvalid, employee.BankAccountStatus)
}

func TestValidate_ProviderError_UnverifiedFailOpen(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	p := &verifierProvider{
		name: "onepay",
		checkFunc: func(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
			return nil, errors.New("onepay 503")
		},
	}
	v := newTestValidator(p, bank, nil)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	// 5xx/timeout → fail-open, NOT invalid.
	assert.Equal(t, domain.BankAccountStatusUnverified, r.Status)
	assert.Nil(t, r.Reason)
}

// TestValidate_RateLimiterBlocked_UnverifiedFailOpen proves the validator
// fails open when the queuedProvider's accountLimiter blocks past the 5s
// validation budget. We simulate this by having the provider return
// context.DeadlineExceeded — exactly what limiter.Wait returns when it
// can't obtain a token within the ctx deadline.
func TestValidate_RateLimiterBlocked_UnverifiedFailOpen(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	p := &verifierProvider{
		name: "onepay",
		checkFunc: func(ctx context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
			// Mimics queuedProvider.CheckAccount returning the
			// rate.Limiter.Wait(ctx) error when no token is available.
			return nil, context.DeadlineExceeded
		},
	}
	v := newTestValidator(p, bank, nil)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	// Rate-limiter blocked → fail-open, NOT invalid. The employee is
	// persisted and NOT shown in the warning list (unverified excluded).
	assert.Equal(t, domain.BankAccountStatusUnverified, r.Status)
	assert.Nil(t, r.Reason)
}

func TestValidate_ProviderWithoutAccountVerifier_DegradesToValid(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	reg := disbursement.NewRegistry(&noVerifierProvider{name: "9pay-noop"})
	repo := &stubBankRepo{byID: map[uint]*domain.Bank{1: bank}}
	v := NewBankAccountValidator(reg, repo, nil, nil)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusValid, r.Status)
}

// --- helpers ----------------------------------------------------------------

func TestApplyBankAccountValidation_NilValidator_Noop(t *testing.T) {
	e := &domain.Employee{BankID: uintPtr(1), BankAccountNumber: "x"}
	applyBankAccountValidation(context.Background(), nil, e)
	// Untouched: default zero-value status.
	assert.Empty(t, e.BankAccountStatus)
	assert.Nil(t, e.BankAccountInvalidReason)
	assert.Nil(t, e.BankAccountValidatedAt)
}

func TestBankFieldsChanged(t *testing.T) {
	id1 := uint(1)
	id2 := uint(2)
	cases := []struct {
		name     string
		original *domain.Employee
		updated  *domain.Employee
		want     bool
	}{
		{"nil original", nil, &domain.Employee{}, true},
		{"identical", &domain.Employee{BankID: &id1, BankAccountNumber: "x", BankAccountName: "y"},
			&domain.Employee{BankID: &id1, BankAccountNumber: "x", BankAccountName: "y"}, false},
		{"number changed", &domain.Employee{BankID: &id1, BankAccountNumber: "x"},
			&domain.Employee{BankID: &id1, BankAccountNumber: "z"}, true},
		{"bankID nil vs set", &domain.Employee{BankID: nil}, &domain.Employee{BankID: &id1}, true},
		{"bankID value changed", &domain.Employee{BankID: &id1}, &domain.Employee{BankID: &id2}, true},
		{"name changed", &domain.Employee{BankAccountName: "a"}, &domain.Employee{BankAccountName: "b"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, bankFieldsChanged(c.original, c.updated))
		})
	}
}

func TestRowBankFieldsPresent(t *testing.T) {
	empty := dto.EmployeeImportRow{}
	assert.False(t, rowBankFieldsPresent(empty))

	withAcct := dto.EmployeeImportRow{BankAccount: "123"}
	assert.True(t, rowBankFieldsPresent(withAcct))

	withName := dto.EmployeeImportRow{BankAccountName: "Nguyen Van A"}
	assert.True(t, rowBankFieldsPresent(withName))

	withBank := dto.EmployeeImportRow{BankName: "VCB"}
	assert.True(t, rowBankFieldsPresent(withBank))
}

// --- UpdateBankInfo (BCC chokepoint) ----------------------------------------

// stubEmployeeRepo captures UpdateColumns calls and serves GetByID from an
// in-memory map. Embeds the interface so we only override what we use.
type stubEmployeeRepo struct {
	domain.EmployeeRepository
	byID        map[uint]*domain.Employee
	getErr      error
	updatedID   uint
	cols        map[string]any
	updateErr   error
	updateCalls int
}

func (s *stubEmployeeRepo) GetByID(_ context.Context, id uint) (*domain.Employee, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.byID[id], nil
}

func (s *stubEmployeeRepo) UpdateColumns(_ context.Context, id uint, cols map[string]any) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updatedID = id
	s.cols = cols
	return nil
}

func (s *stubEmployeeRepo) Update(_ context.Context, _ *domain.Employee) error {
	s.updateCalls++
	return s.updateErr
}

type directTransactionManager struct{}

func (directTransactionManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (directTransactionManager) WithTransactionResult(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	return fn(ctx)
}

// newEmployeeServiceWithValidator builds an EmployeeService whose only wired
// dependencies are the repo + validator, enough to drive UpdateBankInfo.
func newEmployeeServiceWithValidator(repo domain.EmployeeRepository, v *BankAccountValidator) *EmployeeService {
	return &EmployeeService{
		EmployeeRepo:         repo,
		bankAccountValidator: v,
	}
}

func TestUpdateEmployee_InvalidManualBankEditDoesNotPersist(t *testing.T) {
	bank := &domain.Bank{ID: 5, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	id := uint(1)
	original := &domain.Employee{
		ID:                id,
		Fullname:          "Nguyen Van A",
		CCCD:              "123456789012",
		BankID:            &bank.ID,
		BankAccountNumber: "VALID123",
		BankAccountName:   "Nguyen Van A",
		BankAccountStatus: domain.BankAccountStatusValid,
	}
	repo := &stubEmployeeRepo{byID: map[uint]*domain.Employee{id: original}}
	provider := &verifierProvider{
		name: "onepay",
		checkFunc: func(context.Context, infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
			return &infrastructure.AccountCheckResult{
				Valid:        false,
				AccountName:  "NGUYEN VAN A",
				RawErrorCode: "name_mismatch",
			}, nil
		},
	}
	validator := newTestValidator(provider, bank, nil)
	service := &EmployeeService{
		EmployeeRepo:         repo,
		BankRepo:             &stubBankRepo{byID: map[uint]*domain.Bank{bank.ID: bank}},
		TransactionManager:   directTransactionManager{},
		bankAccountValidator: validator,
	}
	edited := &domain.Employee{
		ID:                id,
		Fullname:          original.Fullname,
		CCCD:              original.CCCD,
		BankID:            &bank.ID,
		BankAccountNumber: "INVALID456",
		BankAccountName:   "Wrong Name",
	}

	err := service.UpdateEmployee(context.Background(), edited, 99)

	var domainErr *domain.DomainError
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, bankAccountInvalidErrorCode, domainErr.Code)
	assert.Zero(t, repo.updateCalls, "invalid bank details must not reach repository persistence")
	assert.Equal(t, "VALID123", original.BankAccountNumber)
	assert.Equal(t, "Nguyen Van A", original.BankAccountName)
}

func TestUpdateBankInfo_NoBankColumns_SkipsValidation(t *testing.T) {
	repo := &stubEmployeeRepo{}
	// Validator with a provider that would fail the test if called.
	v := newTestValidator(&verifierProvider{name: "onepay", checkFunc: func(context.Context, infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
		t.Fatal("validator should not be called when no bank columns are present")
		return nil, nil
	}}, nil, nil)
	svc := newEmployeeServiceWithValidator(repo, v)

	if err := svc.UpdateBankInfo(context.Background(), 1, map[string]any{"mobile": "0900"}); err != nil {
		t.Fatalf("UpdateBankInfo: %v", err)
	}
	_, hasStatus := repo.cols["bank_account_status"]
	assert.False(t, hasStatus, "validation fields must not be injected when no bank columns changed")
}

func TestUpdateBankInfo_BankColumnsChanged_RunsValidationAndFoldsResult(t *testing.T) {
	bank := &domain.Bank{ID: 5, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	id := uint(1)
	current := &domain.Employee{ID: id, BankID: &bank.ID, BankAccountNumber: "OLD", BankAccountName: "Old Name"}
	repo := &stubEmployeeRepo{byID: map[uint]*domain.Employee{id: current}}

	var seenReq infrastructure.AccountCheckRequest
	p := &verifierProvider{
		name: "onepay",
		checkFunc: func(_ context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
			seenReq = req
			return &infrastructure.AccountCheckResult{Valid: true}, nil
		},
	}
	v := newTestValidator(p, bank, nil)
	svc := newEmployeeServiceWithValidator(repo, v)

	updates := map[string]any{
		"bank_account_number": "NEW123",
		"bank_account_name":   "New Name",
		"bank_id":             bank.ID, // uint form (matches 3 of 4 BCC callers)
	}
	if err := svc.UpdateBankInfo(context.Background(), id, updates); err != nil {
		t.Fatalf("UpdateBankInfo: %v", err)
	}

	// Validator saw the EFFECTIVE (incoming) values, not the stale current ones.
	assert.Equal(t, "NEW123", seenReq.AccountNo)
	assert.Equal(t, "New Name", seenReq.AccountName)
	assert.Equal(t, "ICBVVNVX", seenReq.SwiftCode)

	// Outcome folded into the SAME update map → atomic write.
	assert.Equal(t, domain.BankAccountStatusValid, updates["bank_account_status"])
	assert.Nil(t, updates["bank_account_invalid_reason"])
	assert.NotNil(t, updates["bank_account_validated_at"])
}

func TestUpdateBankInfo_BankIDAsPointer_Handled(t *testing.T) {
	// 1 of 4 BCC callers passes *uint via ResolveBankID.
	bank := &domain.Bank{ID: 5, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	id := uint(1)
	current := &domain.Employee{ID: id}
	repo := &stubEmployeeRepo{byID: map[uint]*domain.Employee{id: current}}
	bankIDPtr := uint(5)

	var seenSwift string
	p := &verifierProvider{
		name: "onepay",
		checkFunc: func(_ context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
			seenSwift = req.SwiftCode
			return &infrastructure.AccountCheckResult{Valid: true}, nil
		},
	}
	v := newTestValidator(p, bank, nil)
	svc := newEmployeeServiceWithValidator(repo, v)

	updates := map[string]any{
		"bank_account_number": "123",
		"bank_account_name":   "Nguyen Van A",
		"bank_id":             &bankIDPtr, // *uint form
	}
	if err := svc.UpdateBankInfo(context.Background(), id, updates); err != nil {
		t.Fatalf("UpdateBankInfo: %v", err)
	}
	assert.Equal(t, "ICBVVNVX", seenSwift, "*uint bank_id must resolve to the bank's swift code")
}

func TestUpdateBankInfo_UnexpectedBankIDType_FailsOpenUnverified(t *testing.T) {
	id := uint(1)
	current := &domain.Employee{ID: id}
	repo := &stubEmployeeRepo{byID: map[uint]*domain.Employee{id: current}}
	v := newTestValidator(&verifierProvider{name: "onepay"}, nil, nil)
	svc := newEmployeeServiceWithValidator(repo, v)

	// JSON-decoded float64 — a type the switch does not recognise.
	updates := map[string]any{
		"bank_account_number": "123",
		"bank_id":             float64(5),
	}
	if err := svc.UpdateBankInfo(context.Background(), id, updates); err != nil {
		t.Fatalf("UpdateBankInfo: %v", err)
	}
	// Fail-open: unverified, NOT a stale valid/invalid against the old bank.
	assert.Equal(t, domain.BankAccountStatusUnverified, updates["bank_account_status"])
}

func TestUpdateBankInfo_GetByIDError_SkipsValidationSilently(t *testing.T) {
	repo := &stubEmployeeRepo{getErr: errors.New("db down")}
	v := newTestValidator(&verifierProvider{name: "onepay"}, nil, nil)
	svc := newEmployeeServiceWithValidator(repo, v)

	updates := map[string]any{"bank_account_number": "123"}
	if err := svc.UpdateBankInfo(context.Background(), 1, updates); err != nil {
		t.Fatalf("UpdateBankInfo: %v", err)
	}
	// Validation skipped; original bank columns untouched.
	_, hasStatus := updates["bank_account_status"]
	assert.False(t, hasStatus)
}

func TestHasBankColumns(t *testing.T) {
	assert.False(t, hasBankColumns(map[string]any{"mobile": "x"}))
	assert.True(t, hasBankColumns(map[string]any{"bank_id": uint(1)}))
	assert.True(t, hasBankColumns(map[string]any{"bank_account_number": "x"}))
	assert.True(t, hasBankColumns(map[string]any{"bank_account_name": "x"}))
}

// --- Dedup cache ------------------------------------------------------------
//
// The next six tests cover the Redis-backed dedup layer: identical bank
// info (swift code + account number + holder name) checked within the TTL
// must reuse the cached verdict and skip the OnePay call. Different bank
// info must NOT collide. Cache failures fall through to OnePay.

func TestValidate_CacheHit_SkipsOnePayCall(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	cache := newStubCache()
	// Seed a cached "invalid" verdict for the exact bank info we'll check.
	cachedReason := "Tên chủ tài khoản không khớp"
	cache.seed("ICBVVNVX", "123456789", "Nguyen Van A", ValidationResult{
		Status: domain.BankAccountStatusInvalid, Reason: &cachedReason,
	})
	// Reset the Set counter so the assertion below measures only writes
	// performed by Validate (not the seed).
	cache.resetSetCallCount()

	// Provider's checkFunc must never run on a cache hit.
	p := &verifierProvider{name: "onepay", checkFunc: func(context.Context, infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
		t.Fatal("CheckAccount must not be called on a cache hit")
		return nil, nil
	}}
	v := newTestValidatorWithCache(p, bank, nil, cache)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusInvalid, r.Status)
	assert.NotNil(t, r.Reason)
	assert.Equal(t, cachedReason, *r.Reason)
	assert.Equal(t, 0, cache.setCallCount(), "cache hit must not trigger a re-write")
}

func TestValidate_CacheMiss_CallsOnePayAndStoresResult(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	cache := newStubCache() // empty → miss

	callCount := 0
	p := &verifierProvider{name: "onepay", checkFunc: func(_ context.Context, _ infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
		callCount++
		return &infrastructure.AccountCheckResult{Valid: true}, nil
	}}
	v := newTestValidatorWithCache(p, bank, nil, cache)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusValid, r.Status)
	assert.Equal(t, 1, callCount, "cache miss must call OnePay exactly once")
	assert.True(t, cache.hasEntry("ICBVVNVX", "123456789", "Nguyen Van A"),
		"verdict must be stored in the cache for future dedup")
}

func TestValidate_SecondCallHitsCache_NoSecondOnePayCall(t *testing.T) {
	// The end-to-end dedup story: two Validate calls with identical bank
	// info must result in exactly ONE OnePay call.
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	cache := newStubCache()

	callCount := 0
	p := &verifierProvider{name: "onepay", checkFunc: func(_ context.Context, _ infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
		callCount++
		return &infrastructure.AccountCheckResult{Valid: true}, nil
	}}
	v := newTestValidatorWithCache(p, bank, nil, cache)

	id := bank.ID
	_ = v.Validate(context.Background(), &id, "123456789", "Nguyen Van A")
	_ = v.Validate(context.Background(), &id, "123456789", "Nguyen Van A")
	_ = v.Validate(context.Background(), &id, "123456789", "Nguyen Van A")

	assert.Equal(t, 1, callCount, "three identical checks → one OnePay call (dedup)")
}

func TestValidate_DifferentAccountInfo_DifferentCacheKeys(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	cache := newStubCache()
	// Seed a verdict for (acct=1, name=X).
	cache.seed("ICBVVNVX", "1", "Nguyen Van A", ValidationResult{Status: domain.BankAccountStatusValid})

	// Different name → different key → must call OnePay.
	called := false
	p := &verifierProvider{name: "onepay", checkFunc: func(_ context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
		called = true
		assert.Equal(t, "Tran Thi B", req.AccountName, "must reach OnePay with the new name")
		return &infrastructure.AccountCheckResult{Valid: true}, nil
	}}
	v := newTestValidatorWithCache(p, bank, nil, cache)

	r := v.Validate(context.Background(), &bank.ID, "1", "Tran Thi B")
	assert.True(t, called, "different holder name must miss the cache and call OnePay")
	assert.Equal(t, domain.BankAccountStatusValid, r.Status)
}

func TestValidate_CacheGetError_FailsOpenToOnePay(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	cache := newStubCache()
	cache.getErr = errors.New("redis down")

	called := false
	p := &verifierProvider{name: "onepay", checkFunc: func(_ context.Context, _ infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
		called = true
		return &infrastructure.AccountCheckResult{Valid: true}, nil
	}}
	v := newTestValidatorWithCache(p, bank, nil, cache)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	assert.True(t, called, "Redis Get error must fall through to OnePay (fail-open dedup)")
	assert.Equal(t, domain.BankAccountStatusValid, r.Status)
}

func TestValidate_CacheSetError_IgnoredNotBlocking(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	cache := newStubCache()
	cache.setErr = errors.New("redis write failure")

	p := &verifierProvider{name: "onepay", checkFunc: func(_ context.Context, _ infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
		return &infrastructure.AccountCheckResult{Valid: true}, nil
	}}
	v := newTestValidatorWithCache(p, bank, nil, cache)

	// Must NOT error — Set failure is swallowed, the OnePay verdict still returns.
	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusValid, r.Status)
}

func TestValidate_SwiftMissing_NotCached(t *testing.T) {
	// The swift-missing config error returns early BEFORE the cache layer,
	// so it must neither read from nor write to the cache. This matters
	// because caching it would mask an admin fix to the bank record within
	// the TTL window.
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: ""}
	cache := newStubCache()

	p := &verifierProvider{name: "onepay", checkFunc: func(context.Context, infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
		t.Fatal("CheckAccount must not run when swift code is missing")
		return nil, nil
	}}
	v := newTestValidatorWithCache(p, bank, nil, cache)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusInvalid, r.Status)
	assert.Equal(t, 0, cache.setCallCount(), "swift-missing early return must not touch the cache")
}

// TestValidate_UnverifiedNotCached locks in the design decision that a
// provider outage/timeout verdict is NOT cached. A transient OnePay blip
// cached for 15 min would leave accounts silently unvalidated long after
// recovery — defeating the fail-open design. The 2 TPS accountLimiter
// already bounds re-hammering during a real outage.
func TestValidate_UnverifiedNotCached(t *testing.T) {
	bank := &domain.Bank{ID: 1, BranchName: "VCB", SwiftCode: "ICBVVNVX"}
	cache := newStubCache()

	p := &verifierProvider{name: "onepay", checkFunc: func(_ context.Context, _ infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
		return nil, errors.New("onepay 503")
	}}
	v := newTestValidatorWithCache(p, bank, nil, cache)

	r := v.Validate(context.Background(), &bank.ID, "123456789", "Nguyen Van A")
	assert.Equal(t, domain.BankAccountStatusUnverified, r.Status)
	assert.False(t, cache.hasEntry("ICBVVNVX", "123456789", "Nguyen Van A"),
		"unverified verdict must NOT be cached so recovery re-validates immediately")
	assert.Equal(t, 0, cache.setCallCount(), "unverified must not trigger a cache write")
}
