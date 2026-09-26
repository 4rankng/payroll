package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/cache"
	"api-server/internal/infra/zalo"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/password"
)

// --- fakes ------------------------------------------------------------------

type fakeSession struct {
	uid      uint
	codeHash string
}

type fakeStore struct {
	sessions map[string]fakeSession
	verified map[string]uint
	seq      int
}

func newFakeStore() *fakeStore {
	return &fakeStore{sessions: map[string]fakeSession{}, verified: map[string]uint{}}
}

func (f *fakeStore) Create(_ context.Context, uid uint, codeHash string) (string, error) {
	f.seq++
	id := fmt.Sprintf("sess%d", f.seq)
	f.sessions[id] = fakeSession{uid: uid, codeHash: codeHash}
	return id, nil
}

func (f *fakeStore) Consume(_ context.Context, sessionID, codeHash string) (uint, error) {
	s, ok := f.sessions[sessionID]
	if !ok {
		return 0, cache.ErrZaloResetSessionNotFound
	}
	if s.codeHash != codeHash {
		return 0, cache.ErrZaloResetInvalidCode
	}
	delete(f.sessions, sessionID)
	return s.uid, nil
}

func (f *fakeStore) CreateVerified(_ context.Context, uid uint) (string, error) {
	f.seq++
	token := fmt.Sprintf("tok%d", f.seq)
	f.verified[token] = uid
	return token, nil
}

func (f *fakeStore) ConsumeVerified(_ context.Context, token string) (uint, error) {
	uid, ok := f.verified[token]
	if !ok {
		return 0, cache.ErrZaloResetVerifiedNotFound
	}
	delete(f.verified, token)
	return uid, nil
}

func (f *fakeStore) TTL() time.Duration { return 10 * time.Minute }

type fakeSender struct {
	result   zalo.SendResult
	err      error
	calls    int
	lastData map[string]string
}

func (f *fakeSender) Send(_ context.Context, _ string, _ string, _ string, data map[string]string) (zalo.SendResult, error) {
	f.calls++
	f.lastData = data
	return f.result, f.err
}

type fakeUserRepo struct {
	byMobile      map[string]*domain.User
	byID          map[uint]*domain.User
	hashed        map[uint]string
	invalidBefore map[uint]time.Time
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byMobile: map[string]*domain.User{}, byID: map[uint]*domain.User{}, hashed: map[uint]string{}, invalidBefore: map[uint]time.Time{}}
}

func (r *fakeUserRepo) GetByMobile(_ context.Context, mobile string) (*domain.User, error) {
	if u, ok := r.byMobile[mobile]; ok {
		return u, nil
	}
	return nil, domain.NewNotFoundError("user not found")
}

func (r *fakeUserRepo) GetByID(_ context.Context, id uint) (*domain.User, error) {
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, domain.NewNotFoundError("user not found")
}

func (r *fakeUserRepo) UpdatePasswordAndInvalidateSessions(_ context.Context, userID uint, hashed string, invalidBefore time.Time) error {
	r.hashed[userID] = hashed
	r.invalidBefore[userID] = invalidBefore
	return nil
}

type fakeEmployeeRepo struct {
	byMobile map[string][]*domain.Employee
}

func (r *fakeEmployeeRepo) ListByMobile(_ context.Context, mobile string) ([]*domain.Employee, error) {
	return r.byMobile[mobile], nil
}

func (r *fakeEmployeeRepo) GetByUserID(_ context.Context, userID uint) (*domain.Employee, error) {
	for _, list := range r.byMobile {
		for _, e := range list {
			if e.UserID != nil && *e.UserID == userID {
				return e, nil
			}
		}
	}
	return nil, domain.NewNotFoundError("employee not found")
}

type fakePassword struct{}

func (fakePassword) ValidatePassword(p string) error {
	if p == "weak" {
		return errors.New("mật khẩu không hợp lệ: weak")
	}
	return nil
}

func (fakePassword) HashNewPassword(p string) (string, error) { return "hash:" + p, nil }

type fakeEnabled struct{ on bool }

func (e fakeEnabled) IsEnabled(context.Context) (bool, error) { return e.on, nil }

// --- fixtures ---------------------------------------------------------------

func ptr(s string) *string { return &s }

var fixtureNow = time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)

func newFixture(t *testing.T, sender zalo.Sender, enabled EnabledChecker) (*Service, *fakeStore, *fakeUserRepo) {
	t.Helper()
	store := newFakeStore()
	users := newFakeUserRepo()
	emp := &fakeEmployeeRepo{byMobile: map[string][]*domain.Employee{}}

	u := &domain.User{
		ID:       1,
		Username: "hoanvv",
		Fullname: "Vũ Văn Hoan",
		Role:     domain.RolePartner,
		Mobile:   ptr("0987654321"),
	}
	users.byMobile["0987654321"] = u
	users.byID[1] = u

	svc := NewService(store, users, emp, fakePassword{}, sender, "12345", enabled, nil, clock.NewFake(fixtureNow), nil)
	return svc, store, users
}

func codeOf(t *testing.T, sender *fakeSender) string {
	t.Helper()
	if sender.lastData == nil {
		t.Fatal("sender captured no data")
	}
	code := sender.lastData["otp"]
	if len(code) != 6 {
		t.Fatalf("captured otp = %q, want 6 digits", code)
	}
	return code
}

// --- tests ------------------------------------------------------------------

func TestRequestOTPHappyPathAndVerifyReset(t *testing.T) {
	sender := &fakeSender{}
	svc, store, users := newFixture(t, sender, nil)
	ctx := context.Background()

	res, err := svc.RequestOTP(ctx, "0987654321")
	if err != nil {
		t.Fatalf("RequestOTP: %v", err)
	}
	if !res.Found || !res.OTPSent {
		t.Fatalf("found=%v otp_sent=%v, want true/true", res.Found, res.OTPSent)
	}
	if res.SessionID == "" || res.EmployeeName != "Vũ Văn Hoan" {
		t.Fatalf("session=%q name=%q", res.SessionID, res.EmployeeName)
	}
	if res.ExpiresIn != 600 {
		t.Fatalf("expires_in = %d, want 600", res.ExpiresIn)
	}

	code := codeOf(t, sender)

	verify, err := svc.VerifyOTP(ctx, res.SessionID, code)
	if err != nil {
		t.Fatalf("VerifyOTP: %v", err)
	}
	if verify.ResetToken == "" || verify.ExpiresIn != 300 {
		t.Fatalf("verify = %+v", verify)
	}
	if len(store.sessions) != 0 {
		t.Fatalf("session not consumed")
	}

	reset, err := svc.ResetPassword(ctx, verify.ResetToken, "", 9, "chatbot")
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if reset.Username != "hoanvv" || len(reset.NewPassword) != 12 {
		t.Fatalf("reset = %+v", reset)
	}
	if users.hashed[1] != "hash:"+reset.NewPassword {
		t.Fatalf("stored hash = %q", users.hashed[1])
	}
	if !users.invalidBefore[1].Equal(fixtureNow) {
		t.Fatalf("invalid_before = %v, want %v", users.invalidBefore[1], fixtureNow)
	}
}

func TestRequestOTPDeliveryFailureReportsCode(t *testing.T) {
	sender := &fakeSender{result: zalo.SendResult{ErrorCode: -118, ErrorMsg: "phone not linked"}}
	svc, store, _ := newFixture(t, sender, nil)

	res, err := svc.RequestOTP(context.Background(), "0987654321")
	if err != nil {
		t.Fatalf("RequestOTP: %v", err)
	}
	if !res.Found || res.OTPSent {
		t.Fatalf("found=%v otp_sent=%v, want true/false", res.Found, res.OTPSent)
	}
	if res.FailureReason != FailureDeliveryFailed || res.DeliveryErrorCode != -118 {
		t.Fatalf("failure=%q code=%d", res.FailureReason, res.DeliveryErrorCode)
	}
	if res.SessionID != "" {
		t.Fatalf("session_id = %q, want empty", res.SessionID)
	}
	if len(store.sessions) != 0 {
		t.Fatalf("a session was created for an undelivered code")
	}
}

func TestRequestOTPUnknownPhone(t *testing.T) {
	sender := &fakeSender{}
	svc, _, _ := newFixture(t, sender, nil)

	res, err := svc.RequestOTP(context.Background(), "0900000000")
	if err != nil {
		t.Fatalf("RequestOTP: %v", err)
	}
	if res.Found || res.FailureReason != FailureAccountNotFound {
		t.Fatalf("res = %+v", res)
	}
	if sender.calls != 0 {
		t.Fatalf("sender called for unknown phone")
	}
}

func TestRequestOTPDisabledReportsZaloDisabled(t *testing.T) {
	sender := &fakeSender{}
	svc, _, _ := newFixture(t, sender, fakeEnabled{on: false})

	res, err := svc.RequestOTP(context.Background(), "0987654321")
	if err != nil {
		t.Fatalf("RequestOTP: %v", err)
	}
	if !res.Found || res.FailureReason != FailureZaloDisabled {
		t.Fatalf("res = %+v", res)
	}
	if sender.calls != 0 {
		t.Fatalf("sender called while disabled")
	}
}

func TestRequestOTPInvalidPhone(t *testing.T) {
	sender := &fakeSender{}
	svc, _, _ := newFixture(t, sender, nil)

	_, err := svc.RequestOTP(context.Background(), "034090005032")
	if !domain.IsValidationError(err) {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestVerifyWrongCodeKeepsSession(t *testing.T) {
	sender := &fakeSender{}
	svc, store, _ := newFixture(t, sender, nil)
	ctx := context.Background()

	res, _ := svc.RequestOTP(ctx, "0987654321")
	code := codeOf(t, sender)

	if _, err := svc.VerifyOTP(ctx, res.SessionID, "000000"); !domain.IsUnauthorizedError(err) {
		t.Fatalf("wrong code err = %v, want unauthorized", err)
	}
	if len(store.sessions) != 1 {
		t.Fatalf("session consumed on wrong code")
	}

	// Correct code now succeeds.
	if _, err := svc.VerifyOTP(ctx, res.SessionID, code); err != nil {
		t.Fatalf("VerifyOTP after retry: %v", err)
	}
	// Session is single-use: a second verify fails.
	if _, err := svc.VerifyOTP(ctx, res.SessionID, code); !domain.IsUnauthorizedError(err) {
		t.Fatalf("second verify err = %v, want unauthorized", err)
	}
}

func TestResetTokenSingleUse(t *testing.T) {
	sender := &fakeSender{}
	svc, _, _ := newFixture(t, sender, nil)
	ctx := context.Background()

	res, _ := svc.RequestOTP(ctx, "0987654321")
	verify, _ := svc.VerifyOTP(ctx, res.SessionID, codeOf(t, sender))

	if _, err := svc.ResetPassword(ctx, verify.ResetToken, "Hoan@2026", 1, "k"); err != nil {
		t.Fatalf("first reset: %v", err)
	}
	if _, err := svc.ResetPassword(ctx, verify.ResetToken, "Hoan@2026", 1, "k"); !domain.IsUnauthorizedError(err) {
		t.Fatalf("second reset err = %v, want unauthorized", err)
	}
}

func TestResetPasswordRejectsWeakSupplied(t *testing.T) {
	sender := &fakeSender{}
	svc, _, _ := newFixture(t, sender, nil)
	ctx := context.Background()

	res, _ := svc.RequestOTP(ctx, "0987654321")
	verify, _ := svc.VerifyOTP(ctx, res.SessionID, codeOf(t, sender))

	if _, err := svc.ResetPassword(ctx, verify.ResetToken, "weak", 1, "k"); !domain.IsValidationError(err) {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestLookupEmployeePrefersEmployeeRecord(t *testing.T) {
	sender := &fakeSender{}
	store := newFakeStore()
	users := newFakeUserRepo()
	emp := &fakeEmployeeRepo{byMobile: map[string][]*domain.Employee{}}
	u := &domain.User{ID: 5, Username: "e5", Fullname: "Nguyễn Văn A", Role: domain.RoleEmployee}
	users.byID[5] = u
	emp.byMobile["0912345678"] = []*domain.Employee{{ID: 9, Fullname: "Nguyễn Văn A", CCCD: "012345678901", Mobile: "0912345678", UserID: &u.ID}}
	svc := NewService(store, users, emp, fakePassword{}, sender, "1", nil, nil, clock.NewFake(fixtureNow), nil)

	res, err := svc.LookupEmployee(context.Background(), "0912345678")
	if err != nil {
		t.Fatalf("LookupEmployee: %v", err)
	}
	if !res.Found || res.CCCD != "012345678901" || res.EmployeeName != "Nguyễn Văn A" || res.Mobile != "0912345678" {
		t.Fatalf("res = %+v", res)
	}
}

func TestLookupEmployeeUnknownAndInvalid(t *testing.T) {
	sender := &fakeSender{}
	svc, _, _ := newFixture(t, sender, nil)
	ctx := context.Background()

	res, err := svc.LookupEmployee(ctx, "0900000000")
	if err != nil {
		t.Fatalf("unknown: %v", err)
	}
	if res.Found {
		t.Fatalf("expected found=false")
	}

	if _, err := svc.LookupEmployee(ctx, "034090005032"); !domain.IsValidationError(err) {
		t.Fatalf("invalid phone err = %v, want validation error", err)
	}
}

func TestNameCoherenceAcrossEndpoints(t *testing.T) {
	sender := &fakeSender{}
	store := newFakeStore()
	users := newFakeUserRepo()
	emp := &fakeEmployeeRepo{byMobile: map[string][]*domain.Employee{}}
	u := &domain.User{ID: 3, Username: "e3", Fullname: "User Row Name", Role: domain.RoleEmployee}
	users.byID[3] = u
	emp.byMobile["0912345678"] = []*domain.Employee{{
		ID: 7, Fullname: "Employee Row Name", CCCD: "012345678901", Mobile: "0912345678", UserID: &u.ID,
	}}
	svc := NewService(store, users, emp, fakePassword{}, sender, "1", nil, nil, clock.NewFake(fixtureNow), nil)
	ctx := context.Background()

	lookup, err := svc.LookupEmployee(ctx, "0912345678")
	if err != nil {
		t.Fatalf("LookupEmployee: %v", err)
	}
	otp, err := svc.RequestOTP(ctx, "0912345678")
	if err != nil {
		t.Fatalf("RequestOTP: %v", err)
	}
	verify, err := svc.VerifyOTP(ctx, otp.SessionID, codeOf(t, sender))
	if err != nil {
		t.Fatalf("VerifyOTP: %v", err)
	}
	reset, err := svc.ResetPassword(ctx, verify.ResetToken, "", 1, "k")
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	for name, got := range map[string]string{
		"lookup": lookup.EmployeeName,
		"otp":    otp.EmployeeName,
		"reset":  reset.EmployeeName,
	} {
		if got != "Employee Row Name" {
			t.Fatalf("%s employee_name = %q, want %q", name, got, "Employee Row Name")
		}
	}
	if lookup.CCCD != "012345678901" || lookup.Mobile != "0912345678" {
		t.Fatalf("lookup detail = %+v", lookup)
	}
}

func TestGeneratePasswordSatisfiesPolicy(t *testing.T) {
	validator := password.NewPasswordValidator()
	for range 100 {
		p, err := GeneratePassword()
		if err != nil {
			t.Fatalf("GeneratePassword: %v", err)
		}
		if len([]rune(p)) != 12 {
			t.Fatalf("length = %d (%q)", len(p), p)
		}
		if err := validator.Validate(p); err != nil {
			t.Fatalf("generated password %q failed policy: %v", p, err)
		}
		if strings.ContainsAny(p, "Il0O1") {
			t.Fatalf("generated password %q contains an ambiguous character", p)
		}
	}
}
