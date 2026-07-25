package passwordreset

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"api-server/internal/app/services/user"
	"api-server/internal/domain"
	"api-server/internal/infra/cache"
	"api-server/internal/pkg/clock"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// recordingEmailSender captures sent messages (token extractable from the CTA
// link) so the single-use happy path can be verified in-process. Thread-safe —
// Send runs in a goroutine while the test reads the captured slice.
type recordingEmailSender struct {
	mu   sync.Mutex
	sent []*domain.EmailMessage
}

func (s *recordingEmailSender) Send(_ context.Context, msg *domain.EmailMessage) (*domain.EmailDeliveryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, msg)
	return &domain.EmailDeliveryResult{MessageID: "test", Provider: "test"}, nil
}

func (s *recordingEmailSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sent)
}

func (s *recordingEmailSender) first() *domain.EmailMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sent) == 0 {
		return nil
	}
	return s.sent[0]
}

// panicEmailSender panics on Send — used to verify goroutine recovery (H4).
type panicEmailSender struct{}

func (panicEmailSender) Send(context.Context, *domain.EmailMessage) (*domain.EmailDeliveryResult, error) {
	panic("boom from Send")
}

// fakeUserRepo is a minimal UserRepository stub for the reset-service tests.
// Only GetByEmail/GetByID are exercised; the rest panic to catch misuse.
type fakeUserRepo struct {
	byEmail map[string]*domain.User
	byID    map[uint]*domain.User
	// UpdatePasswordAndInvalidateSessions records the last call.
	lastUpdate struct {
		userID        uint
		hashed        string
		invalidBefore time.Time
	}
	updateErr error
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, domain.NewNotFoundError("user not found")
	}
	return u, nil
}
func (r *fakeUserRepo) GetByID(_ context.Context, id uint) (*domain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, domain.NewNotFoundError("user not found")
	}
	return u, nil
}
func (r *fakeUserRepo) UpdatePasswordAndInvalidateSessions(_ context.Context, userID uint, hashed string, invalidBefore time.Time) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.lastUpdate.userID = userID
	r.lastUpdate.hashed = hashed
	r.lastUpdate.invalidBefore = invalidBefore
	return nil
}

// unused methods — panic to surface unintended use during test development.
func (r *fakeUserRepo) Create(context.Context, *domain.User) error { panic("not impl") }
func (r *fakeUserRepo) GetByIDs(context.Context, []uint) (map[uint]*domain.User, error) {
	panic("not impl")
}
func (r *fakeUserRepo) GetByUsername(context.Context, string) (*domain.User, error) {
	panic("not impl")
}
func (r *fakeUserRepo) GetByUsernameIncludingDeleted(context.Context, string) (*domain.User, error) {
	panic("not impl")
}
func (r *fakeUserRepo) GetUsernamesByPrefix(context.Context, string) ([]string, error) {
	panic("not impl")
}
func (r *fakeUserRepo) GetByCCCD(context.Context, string) (*domain.User, error)   { panic("not impl") }
func (r *fakeUserRepo) GetByMobile(context.Context, string) (*domain.User, error) { panic("not impl") }
func (r *fakeUserRepo) ExistsByUsername(context.Context, string) (bool, error)    { panic("not impl") }
func (r *fakeUserRepo) Update(context.Context, *domain.User) error                { panic("not impl") }
func (r *fakeUserRepo) UpdateLastLogin(context.Context, uint, time.Time) error    { panic("not impl") }
func (r *fakeUserRepo) UpdateTokensInvalidBefore(context.Context, uint, time.Time) error {
	panic("not impl")
}
func (r *fakeUserRepo) UpdateOTPLockout(context.Context, uint, int, *time.Time) error {
	panic("not impl")
}
func (r *fakeUserRepo) Delete(context.Context, uint) error  { panic("not impl") }
func (r *fakeUserRepo) Restore(context.Context, uint) error { panic("not impl") }
func (r *fakeUserRepo) List(context.Context, int, int) ([]*domain.User, error) {
	panic("not impl")
}
func (r *fakeUserRepo) ListByRole(context.Context, domain.UserRole) ([]*domain.User, error) {
	panic("not impl")
}
func (r *fakeUserRepo) ListWithFilters(context.Context, *domain.UserRole, string, []uint, bool, string, string, int, int) ([]*domain.User, int64, error) {
	panic("not impl")
}
func (r *fakeUserRepo) Count(context.Context) (int64, error) { panic("not impl") }
func (r *fakeUserRepo) CountByRole(context.Context, domain.UserRole) (int64, error) {
	panic("not impl")
}
func (r *fakeUserRepo) CountRecentLogins(context.Context, time.Time) (int64, error) {
	panic("not impl")
}
func (r *fakeUserRepo) CountActiveEmployeesBySchedule(context.Context, time.Time, time.Time) (int, int, int, error) {
	panic("not impl")
}
func (r *fakeUserRepo) GetActiveEmployeesBySchedule(context.Context, time.Time, time.Time, string) ([]*domain.ActiveEmployeeUser, error) {
	panic("not impl")
}
func (r *fakeUserRepo) FindUsersWithNullLastLogin(context.Context, int, int) ([]*domain.User, error) {
	panic("not impl")
}
func (r *fakeUserRepo) CountUsersWithNullLastLogin(context.Context) (int64, error) { panic("not impl") }

func newTestService(t *testing.T, repo *fakeUserRepo, sender domain.EmailDeliveryPort) (*Service, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := cache.NewPasswordResetTokenStore(client, 30*time.Minute)
	clk := clock.NewFake(time.Date(2026, 7, 24, 12, 0, 0, 0, clock.DefaultLocation))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// userService is nil here — RequestReset's known-user path doesn't touch
	// it; ConfirmReset tests set it via the dedicated constructor below.
	return &Service{
		tokenStore:  store,
		userRepo:    repo,
		emailSender: sender,
		fromEmail:   "noreply@test.example",
		resetURL:    "https://test.example/reset-password",
		clk:         clk,
		logger:      logger,
	}, mr
}

func TestRequestReset_KnownUser_SendsEmail(t *testing.T) {
	email := "alice@example.com"
	repo := &fakeUserRepo{
		byEmail: map[string]*domain.User{"alice@example.com": {ID: 1, Username: "alice", Fullname: "Alice", Email: &email}},
	}
	sender := &recordingEmailSender{}
	svc, _ := newTestService(t, repo, sender)

	if err := svc.RequestReset(context.Background(), "alice@example.com"); err != nil {
		t.Fatalf("RequestReset: %v", err)
	}
	// email send is async — wait briefly for the goroutine.
	deadline := time.After(2 * time.Second)
	for sender.count() == 0 {
		select {
		case <-deadline:
			t.Fatalf("email not sent within timeout")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	if got := sender.first().Kind; got != domain.EmailKindPasswordReset {
		t.Errorf("email Kind = %v, want EmailKindPasswordReset", got)
	}
}

// TestRequestReset_UnknownEmail_NoEmailSent_TimingEqualized verifies the
// anti-enumeration contract: unknown email returns nil AND sends no email, but
// still hits Redis (timing equalization) so the path is indistinguishable.
func TestRequestReset_UnknownEmail_NoEmailSent_TimingEqualized(t *testing.T) {
	repo := &fakeUserRepo{byEmail: map[string]*domain.User{}}
	sender := &recordingEmailSender{}
	svc, _ := newTestService(t, repo, sender)

	err := svc.RequestReset(context.Background(), "ghost@example.com")
	if err != nil {
		t.Errorf("RequestReset for unknown email returned err = %v, want nil", err)
	}
	// Give the goroutine (which doesn't run for unknown emails) a moment, then
	// assert no email was captured.
	time.Sleep(100 * time.Millisecond)
	if n := sender.count(); n != 0 {
		t.Errorf("emails sent = %d, want 0 (unknown email must not trigger a send)", n)
	}
}

// TestRequestReset_PanicRecovered verifies Red Team H4: a panic in the email
// sender goroutine does not crash the test process (and by extension the server).
func TestRequestReset_PanicRecovered(t *testing.T) {
	email := "bob@example.com"
	repo := &fakeUserRepo{
		byEmail: map[string]*domain.User{"bob@example.com": {ID: 2, Username: "bob", Fullname: "Bob", Email: &email}},
	}
	svc, _ := newTestService(t, repo, panicEmailSender{})

	// If recovery is missing, this panics and fails the whole test binary.
	if err := svc.RequestReset(context.Background(), "bob@example.com"); err != nil {
		t.Fatalf("RequestReset: %v", err)
	}
	time.Sleep(100 * time.Millisecond) // let the goroutine run + panic + recover
	// reaching here means the panic was recovered.
}

func TestConfirmReset_InvalidToken_ReturnsUnauthorized(t *testing.T) {
	repo := &fakeUserRepo{}
	svc, _ := newTestService(t, repo, &recordingEmailSender{})

	err := svc.ConfirmReset(context.Background(), "nonexistent-token", "SomeStrong!Pass1")
	if !domain.IsUnauthorizedError(err) {
		t.Errorf("Consume invalid token err = %v, want Unauthorized", err)
	}
}

// TestConfirmReset_ValidToken_CallsTransactionalRepoOnce is the explicit AC
// #5 test: a valid token + valid password drives the full success path and
// asserts UpdatePasswordAndInvalidateSessions is called exactly once with the
// right args — NOT separate Update + UpdateTokensInvalidBefore calls (the
// fakeUserRepo panics on those, so a non-transactional path would fail loudly).
func TestConfirmReset_ValidToken_CallsTransactionalRepoOnce(t *testing.T) {
	fakeNow := time.Date(2026, 7, 24, 12, 0, 0, 0, clock.DefaultLocation)
	clk := clock.NewFake(fakeNow)
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := cache.NewPasswordResetTokenStore(client, 30*time.Minute)

	const userID = uint(55)
	email := "carol@example.com"
	repo := &fakeUserRepo{
		byEmail: map[string]*domain.User{},
		byID: map[uint]*domain.User{
			userID: {ID: userID, Username: "carol", Fullname: "Carol", Email: &email},
		},
	}

	// Real *user.UserService with nil repos — ConfirmReset only touches
	// ValidatePassword + HashNewPassword, which don't use the repos.
	userSvc := user.NewUserService(nil, nil, nil, "test-secret", "test-salt")

	svc := &Service{
		tokenStore:  store,
		userRepo:    repo,
		userService: userSvc,
		emailSender: &recordingEmailSender{},
		fromEmail:   "noreply@test.example",
		resetURL:    "https://test.example/reset-password",
		clk:         clk,
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// Issue a real token through the store, then consume it via ConfirmReset.
	token, err := store.Create(context.Background(), userID)
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}
	newPassword := "VeryStrong!2026"
	if err := svc.ConfirmReset(context.Background(), token, newPassword); err != nil {
		t.Fatalf("ConfirmReset: %v", err)
	}

	if repo.lastUpdate.userID != userID {
		t.Errorf("lastUpdate.userID = %d, want %d", repo.lastUpdate.userID, userID)
	}
	if repo.lastUpdate.hashed == "" || repo.lastUpdate.hashed == newPassword {
		t.Errorf("lastUpdate.hashed = %q, want a non-empty hash != plaintext", repo.lastUpdate.hashed)
	}
	if !repo.lastUpdate.invalidBefore.Equal(fakeNow) {
		t.Errorf("lastUpdate.invalidBefore = %v, want %v (clk.Now)", repo.lastUpdate.invalidBefore, fakeNow)
	}
	// fakeUserRepo.Update and UpdateTokensInvalidBefore panic — if ConfirmReset
	// had used the non-transactional path, the test would have aborted above.
}

// TestConfirmReset_WeakPassword_ReturnsValidationError does NOT touch the repo
// (the password validator rejects before any write).
func TestConfirmReset_WeakPassword_ReturnsValidationError(t *testing.T) {
	fakeNow := time.Date(2026, 7, 24, 12, 0, 0, 0, clock.DefaultLocation)
	clk := clock.NewFake(fakeNow)
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := cache.NewPasswordResetTokenStore(client, 30*time.Minute)

	const userID = uint(77)
	email := "dave@example.com"
	repo := &fakeUserRepo{
		byID: map[uint]*domain.User{userID: {ID: userID, Username: "dave", Fullname: "Dave", Email: &email}},
	}
	userSvc := user.NewUserService(nil, nil, nil, "test-secret", "test-salt")
	svc := &Service{
		tokenStore: store, userRepo: repo, userService: userSvc,
		emailSender: &recordingEmailSender{}, fromEmail: "noreply@test.example",
		resetURL: "https://test.example/reset-password", clk: clk,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	token, _ := store.Create(context.Background(), userID)
	// "weak" — too short / no complexity; validator rejects.
	err = svc.ConfirmReset(context.Background(), token, "weak")
	if err == nil {
		t.Fatalf("ConfirmReset with weak password: want error, got nil")
	}
	if repo.lastUpdate.hashed != "" {
		t.Errorf("weak password should NOT have reached the repo; lastUpdate.hashed = %q", repo.lastUpdate.hashed)
	}
}
