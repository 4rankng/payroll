package zaloreset

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/zalo"
)

// --- fakes (implement the service's narrow interfaces) ---------------------

type fakeUserRepo struct {
	mu         sync.Mutex
	byMobile   map[string]*domain.User
	byID       map[uint]*domain.User
	pwdUpdates []pwdUpdate
}

type fakeEmployeeRepo struct {
	byMobile map[string][]*domain.Employee
}

// ListByMobile mirrors the real repository: it returns every employee sharing
// the number (duplicates are allowed by the schema), and no error when none
// matches.
func (r *fakeEmployeeRepo) ListByMobile(_ context.Context, mobile string) ([]*domain.Employee, error) {
	return r.byMobile[mobile], nil
}

type pwdUpdate struct {
	userID uint
	hashed string
}

func (r *fakeUserRepo) GetByMobile(_ context.Context, mobile string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byMobile[mobile]
	if !ok {
		return nil, domain.NewNotFoundError("user not found")
	}
	return u, nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id uint) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[id]
	if !ok {
		return nil, domain.NewNotFoundError("user not found")
	}
	return u, nil
}

func (r *fakeUserRepo) UpdatePasswordAndInvalidateSessions(_ context.Context, userID uint, hashed string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pwdUpdates = append(r.pwdUpdates, pwdUpdate{userID, hashed})
	return nil
}

type fakeStore struct {
	mu       sync.Mutex
	sessions map[string]storeEntry
	dummies  int
	creates  int
}

type storeEntry struct {
	userID   uint
	codeHash string
}

func newFakeStore() *fakeStore { return &fakeStore{sessions: map[string]storeEntry{}} }

func (s *fakeStore) Create(_ context.Context, userID uint, codeHash string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sid := "sid-" + randStr(8)
	s.sessions[sid] = storeEntry{userID: userID, codeHash: codeHash}
	s.creates++
	return sid, nil
}

func (s *fakeStore) CreateDummy(_ context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sid := "dummy-" + randStr(8)
	s.sessions[sid] = storeEntry{userID: 0, codeHash: "dummyhash"}
	s.dummies++
	return sid, nil
}

func (s *fakeStore) Consume(_ context.Context, sessionID, codeHash string) (uint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.sessions[sessionID]
	if !ok {
		return 0, errSessionNotFound
	}
	if e.codeHash != codeHash {
		return 0, errInvalidCode
	}
	delete(s.sessions, sessionID)
	if e.userID == 0 {
		return 0, errSessionNotFound
	}
	return e.userID, nil
}

type fakeSender struct {
	mu        sync.Mutex
	lastData  map[string]string
	lastPhone string
	sends     int
	err       error
}

func (f *fakeSender) Send(_ context.Context, phone, _, _ string, data map[string]string) (zalo.SendResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sends++
	cp := map[string]string{}
	for k, v := range data {
		cp[k] = v
	}
	f.lastData = cp
	f.lastPhone = phone
	if f.err != nil {
		return zalo.SendResult{}, f.err
	}
	return zalo.SendResult{MsgID: "fake-msg", ErrorCode: 0}, nil
}

func (f *fakeSender) snapshot() (map[string]string, string, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]string{}
	for k, v := range f.lastData {
		out[k] = v
	}
	return out, f.lastPhone, f.sends
}

type alwaysEnabled struct{ on bool }

func (a alwaysEnabled) IsEnabled(_ context.Context) (bool, error) { return a.on, nil }

type fakePasswordSvc struct {
	validateErr error
	hashed      string
	hashErr     error
}

func (f fakePasswordSvc) ValidatePassword(_ string) error { return f.validateErr }
func (f fakePasswordSvc) HashNewPassword(_ string) (string, error) {
	return f.hashed, f.hashErr
}

type fakeEventBus struct {
	mu        sync.Mutex
	published int
}

func (b *fakeEventBus) Publish(_ context.Context, _ ...domain.DomainEvent) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published++
	return nil
}
func (b *fakeEventBus) Subscribe(_ string, _ domain.EventHandler) {}
func (b *fakeEventBus) SubscribeAll(_ domain.EventHandler)        {}

// fakeClock implements clock.Clock with a fixed time for deterministic tests.
type fakeClock struct{ t time.Time }

func (f fakeClock) Now() time.Time        { return f.t }
func (f fakeClock) NowUTC() time.Time     { return f.t.UTC() }
func (f fakeClock) TodayStart() time.Time { return f.t }
func (f fakeClock) TodayEnd() time.Time   { return f.t }
func (f fakeClock) UnixNow() int64        { return f.t.Unix() }

// --- helpers ----------------------------------------------------------------

func randStr(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	out := make([]byte, n)
	for i := range out {
		out[i] = chars[i%len(chars)]
	}
	return string(out)
}

var (
	errSessionNotFound = errors.New("session not found")
	errInvalidCode     = errors.New("invalid code")
)

// --- test factory -----------------------------------------------------------

func newTestService(t *testing.T, enabled bool) (*Service, *fakeUserRepo, *fakeStore, *fakeSender, *fakeEventBus) {
	t.Helper()
	repo := &fakeUserRepo{byMobile: map[string]*domain.User{}, byID: map[uint]*domain.User{}}
	store := newFakeStore()
	sender := &fakeSender{}
	bus := &fakeEventBus{}
	svc := &Service{
		store:        store,
		userRepo:     repo,
		employeeRepo: &fakeEmployeeRepo{byMobile: map[string][]*domain.Employee{}},
		userService:  fakePasswordSvc{hashed: "hashed-pwd"},
		zalo:         sender,
		templateID:   "619684",
		codeTTL:      10 * time.Minute,
		enabled:      alwaysEnabled{on: enabled},
		eventBus:     bus,
		clk:          fakeClock{t: time.Now()},
		logger:       slog.Default(),
	}
	return svc, repo, store, sender, bus
}

// addEmployee registers an employee holding `mobile` and the account it links
// to. Calling it twice with the same number reproduces the duplicate numbers
// the schema allows.
func addEmployee(svc *Service, repo *fakeUserRepo, id uint, mobile, fullname string) {
	u := &domain.User{ID: id, Username: "u" + mobile, Fullname: fullname, Role: domain.RoleEmployee}
	repo.byID[id] = u
	employees := svc.employeeRepo.(*fakeEmployeeRepo)
	employees.byMobile[mobile] = append(employees.byMobile[mobile], &domain.Employee{UserID: &u.ID, Mobile: mobile})
}

// --- RequestReset tests -----------------------------------------------------

func TestRequestReset_KnownEmployee_SendsZNSWithOTP(t *testing.T) {
	svc, repo, _, sender, _ := newTestService(t, true)
	addEmployee(svc, repo, 5, "0987654321", "Nguyễn Văn A")

	// We need to capture the code that was generated. The store received the
	// code hash; the sender received the plaintext code. Both happen in the
	// request goroutine path — but the send is async. Since the fake sender is
	// synchronous (no goroutine), we can observe it after RequestReset returns.
	// However our service dispatches in a goroutine; wait briefly.
	sid, err := svc.RequestReset(context.Background(), "0987654321")
	if err != nil {
		t.Fatalf("RequestReset err: %v", err)
	}
	if !strings.HasPrefix(sid, "sid-") {
		t.Errorf("expected real session id, got %q (dummy path taken?)", sid)
	}
	// Wait for the async send goroutine.
	waitFor(t, func() bool { _, _, n := sender.snapshot(); return n == 1 })

	data, phone, _ := sender.snapshot()
	if phone != "0987654321" {
		t.Errorf("send phone = %q", phone)
	}
	if data["otp"] == "" || len(data["otp"]) != 6 {
		t.Errorf("otp missing/not 6 digits: %q", data["otp"])
	}
	if len(data) != 1 {
		t.Errorf("template data = %#v, want only otp", data)
	}
}

func TestRequestReset_UnknownMobile_DummyAndNoSend(t *testing.T) {
	svc, _, store, sender, _ := newTestService(t, true)
	sid, err := svc.RequestReset(context.Background(), "0000000000")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.HasPrefix(sid, "dummy-") {
		t.Errorf("expected dummy session id, got %q", sid)
	}
	if store.dummies != 1 {
		t.Errorf("dummies = %d, want 1", store.dummies)
	}
	_, _, sends := sender.snapshot()
	if sends != 0 {
		t.Errorf("sends = %d, want 0 (no ZNS for unknown mobile)", sends)
	}
}

func TestRequestReset_DuplicateEmployeeMobile_DummyAndNoSend(t *testing.T) {
	// One number on two employee records cannot be tied to a single account, so
	// the reset must fail closed: dummy session, no ZNS dispatch at all.
	svc, repo, store, sender, _ := newTestService(t, true)
	addEmployee(svc, repo, 41, "0374990377", "Nguyễn Văn A")
	addEmployee(svc, repo, 42, "0374990377", "Nguyễn Văn B")

	sid, err := svc.RequestReset(context.Background(), "0374990377")
	if err != nil {
		t.Fatalf("RequestReset err: %v", err)
	}
	if !strings.HasPrefix(sid, "dummy-") {
		t.Errorf("ambiguous number should take the dummy path, got %q", sid)
	}
	if store.creates != 0 {
		t.Errorf("store creates = %d, want 0 (no real session for an ambiguous number)", store.creates)
	}
	_, _, sends := sender.snapshot()
	if sends != 0 {
		t.Errorf("sends = %d, want 0 (no OTP for an ambiguous number)", sends)
	}
}

func TestRequestReset_CountryCodeInput_ResolvesEmployee(t *testing.T) {
	// The number is typed with the country code; the employee record stores the
	// domestic form, and the dispatched ZNS must use that canonical form.
	svc, repo, _, sender, _ := newTestService(t, true)
	addEmployee(svc, repo, 5, "0987654321", "Nguyễn Văn A")

	sid, err := svc.RequestReset(context.Background(), "+84 987 654 321")
	if err != nil {
		t.Fatalf("RequestReset err: %v", err)
	}
	if strings.HasPrefix(sid, "dummy-") {
		t.Errorf("country-code input for a known employee took the dummy path, got %q", sid)
	}
	waitFor(t, func() bool { _, _, n := sender.snapshot(); return n == 1 })
	_, recipient, _ := sender.snapshot()
	if recipient != "0987654321" {
		t.Errorf("ZNS recipient = %q, want the canonical form 0987654321", recipient)
	}
}

func TestRequestReset_AdminRole_GetsZNS(t *testing.T) {
	// All roles (admin/partner/employee) can use Zalo reset — the role gate
	// was removed so the feature serves every user with a mobile on file.
	svc, repo, store, sender, _ := newTestService(t, true)
	adminMobile := "0912345678"
	u := &domain.User{ID: 9, Username: "admin1", Fullname: "Admin", Role: domain.RoleAdmin, Mobile: &adminMobile}
	repo.byMobile[adminMobile] = u
	repo.byID[9] = u

	sid, _ := svc.RequestReset(context.Background(), adminMobile)
	if strings.HasPrefix(sid, "dummy-") {
		t.Errorf("admin mobile should NOT take dummy path (all roles eligible), got %q", sid)
	}
	if store.dummies != 0 {
		t.Errorf("dummies = %d, want 0", store.dummies)
	}
	waitFor(t, func() bool { _, _, n := sender.snapshot(); return n == 1 })
	_, _, sends := sender.snapshot()
	if sends != 1 {
		t.Errorf("sends = %d, want 1 (admin gets ZNS too)", sends)
	}
}

func TestRequestReset_DisabledToggle_DummyAndNoSend(t *testing.T) {
	svc, repo, _, sender, _ := newTestService(t, false) // disabled
	addEmployee(svc, repo, 5, "0987654321", "Worker")

	sid, _ := svc.RequestReset(context.Background(), "0987654321")
	if !strings.HasPrefix(sid, "dummy-") {
		t.Errorf("disabled toggle should take dummy path, got %q", sid)
	}
	_, _, sends := sender.snapshot()
	if sends != 0 {
		t.Errorf("sends = %d, want 0 (disabled → no dispatch)", sends)
	}
}

// --- ConfirmReset tests -----------------------------------------------------

func TestConfirmReset_Happy(t *testing.T) {
	svc, repo, store, _, bus := newTestService(t, true)
	addEmployee(svc, repo, 7, "0987654321", "Worker")

	// Manually plant a session with a known code so we can confirm it.
	sid, _ := store.Create(context.Background(), 7, hashHexFromString("123456"))
	_ = sid

	err := svc.ConfirmReset(context.Background(), sid, "123456", "newStrongPwd1!")
	if err != nil {
		t.Fatalf("ConfirmReset err: %v", err)
	}
	repo.mu.Lock()
	if len(repo.pwdUpdates) != 1 || repo.pwdUpdates[0].userID != 7 {
		t.Errorf("pwdUpdates = %+v, want one for uid 7", repo.pwdUpdates)
	}
	if repo.pwdUpdates[0].hashed != "hashed-pwd" {
		t.Errorf("hashed = %q", repo.pwdUpdates[0].hashed)
	}
	bus.mu.Lock()
	if bus.published != 1 {
		t.Errorf("published events = %d, want 1", bus.published)
	}
	repo.mu.Unlock()
	bus.mu.Unlock()
}

func TestConfirmReset_WrongCode_NoPasswordChange(t *testing.T) {
	svc, repo, store, _, _ := newTestService(t, true)
	addEmployee(svc, repo, 7, "0987654321", "Worker")
	sid, _ := store.Create(context.Background(), 7, hashHexFromString("123456"))

	err := svc.ConfirmReset(context.Background(), sid, "999999", "newPwd1!")
	if err == nil {
		t.Fatal("expected error on wrong code")
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.pwdUpdates) != 0 {
		t.Errorf("wrong code should NOT change password; updates = %d", len(repo.pwdUpdates))
	}
}

func TestConfirmReset_ConsumedSession_RejectsSecondConfirm(t *testing.T) {
	svc, repo, store, _, _ := newTestService(t, true)
	addEmployee(svc, repo, 7, "0987654321", "Worker")
	sid, _ := store.Create(context.Background(), 7, hashHexFromString("123456"))

	if err := svc.ConfirmReset(context.Background(), sid, "123456", "newPwd1!"); err != nil {
		t.Fatalf("first confirm: %v", err)
	}
	// Second confirm with the same code → session consumed → not-found.
	err := svc.ConfirmReset(context.Background(), sid, "123456", "newPwd1!")
	if err == nil {
		t.Error("second confirm should fail (session consumed)")
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.pwdUpdates) != 1 {
		t.Errorf("password changed %d times, want 1", len(repo.pwdUpdates))
	}
}

func TestConfirmReset_DummySession_Rejects(t *testing.T) {
	svc, repo, store, _, _ := newTestService(t, true)
	sid, _ := store.CreateDummy(context.Background())
	err := svc.ConfirmReset(context.Background(), sid, "anycode", "newPwd1!")
	if err == nil {
		t.Error("dummy session should reject confirm")
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.pwdUpdates) != 0 {
		t.Errorf("dummy confirm changed password: %+v", repo.pwdUpdates)
	}
}

func TestConfirmReset_WeakPassword_Rejected(t *testing.T) {
	// Build a service whose password service rejects validation.
	repo := &fakeUserRepo{byMobile: map[string]*domain.User{}, byID: map[uint]*domain.User{}}
	store := newFakeStore()
	svc := &Service{
		store: store, userRepo: repo,
		userService: fakePasswordSvc{validateErr: errors.New("too weak")},
		zalo:        &fakeSender{}, templateID: "617976", codeTTL: 10 * time.Minute,
		enabled: alwaysEnabled{on: true}, clk: fakeClock{t: time.Now()},
		logger: slog.Default(),
	}
	sid, _ := store.Create(context.Background(), 7, hashHexFromString("123456"))
	err := svc.ConfirmReset(context.Background(), sid, "123456", "weak")
	if err == nil {
		t.Fatal("expected validation error")
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.pwdUpdates) != 0 {
		t.Errorf("weak password should not change it; updates=%d", len(repo.pwdUpdates))
	}
}

// --- helpers ----------------------------------------------------------------

// hashHexFromString hashes a plaintext code the same way the service does
// (sha256 of the otp.HashCode fnv hash). For tests we shortcut by hashing the
// raw string — the store fake just compares the stored hash to the supplied
// hash, so we must produce them consistently. The service computes
// hashHex(otp.HashCode(code)). To match, the test must do the same. We import
// nothing extra and instead replicate: otp.HashCode = fnv64a of code → 8 bytes;
// then hashHex = sha256 hex of those 8 bytes.
func hashHexFromString(code string) string {
	return hashHex(otpHashCode(code))
}

// otpHashCode mirrors otp.HashCode (fnv64a) without an import cycle in the
// test's helper. Keep in sync with app/services/otp/code.go HashCode.
func otpHashCode(code string) []byte {
	return fnv64a(code)
}

func fnv64a(s string) []byte {
	// inline fnv 64a
	const (
		offset64 uint64 = 14695981039346656037
		prime64  uint64 = 1099511628211
	)
	h := offset64
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	out := make([]byte, 8)
	for i := 0; i < 8; i++ {
		out[7-i] = byte(h >> (8 * i))
	}
	return out
}

// waitFor polls cond until it returns true or the timeout elapses.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition never became true within 2s")
}
