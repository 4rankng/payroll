package apikey

import (
	"context"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

type fakeRepo struct {
	byHash map[string]*domain.APIKey
	nextID uint
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byHash: map[string]*domain.APIKey{}}
}

func (r *fakeRepo) Create(_ context.Context, key *domain.APIKey) error {
	r.nextID++
	key.ID = r.nextID
	key.CreatedAt = time.Now()
	r.byHash[key.KeyHash] = key
	return nil
}

func (r *fakeRepo) GetByHash(_ context.Context, hash string) (*domain.APIKey, error) {
	if k, ok := r.byHash[hash]; ok {
		return k, nil
	}
	return nil, domain.NewNotFoundError("api key not found")
}

func (r *fakeRepo) List(_ context.Context) ([]*domain.APIKey, error) {
	out := make([]*domain.APIKey, 0, len(r.byHash))
	for _, k := range r.byHash {
		out = append(out, k)
	}
	return out, nil
}

func (r *fakeRepo) Revoke(_ context.Context, id uint, revokedAt time.Time) error {
	for _, k := range r.byHash {
		if k.ID == id && k.RevokedAt == nil {
			t := revokedAt
			k.RevokedAt = &t
			return nil
		}
	}
	return domain.NewNotFoundError("api key not found")
}

func (r *fakeRepo) UpdateLastUsedAt(_ context.Context, id uint, at time.Time) error {
	for _, k := range r.byHash {
		if k.ID == id {
			t := at
			k.LastUsedAt = &t
			return nil
		}
	}
	return domain.NewNotFoundError("api key not found")
}

func newService(repo *fakeRepo) *Service {
	return NewService(repo, clock.NewFake(time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)))
}

func TestCreateReturnsKeyAndStoresHashAndPrefix(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)

	key, plaintext, err := svc.Create(context.Background(), "Zalo Chatbot", domain.APIKeyScopeChatbot, 7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.HasPrefix(plaintext, "ttk_") {
		t.Fatalf("plaintext %q missing ttk_ prefix", plaintext)
	}
	if len(plaintext) != 47 {
		t.Fatalf("plaintext length = %d, want 47", len(plaintext))
	}
	if key.KeyPrefix != plaintext[:12] {
		t.Fatalf("key_prefix = %q, want %q", key.KeyPrefix, plaintext[:12])
	}
	if key.KeyHash != HashKey(plaintext) {
		t.Fatalf("stored hash does not match sha256(plaintext)")
	}
	if key.CreatedBy != 7 {
		t.Fatalf("created_by = %d, want 7", key.CreatedBy)
	}
}

func TestAuthenticateRoundTripAndFailures(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)
	ctx := context.Background()

	created, plaintext, err := svc.Create(ctx, "chatbot", domain.APIKeyScopeChatbot, 1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := svc.Authenticate(ctx, plaintext)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("authenticated id = %d, want %d", got.ID, created.ID)
	}

	if _, err := svc.Authenticate(ctx, "ttk_unknown"); err != ErrInvalidAPIKey {
		t.Fatalf("unknown key err = %v, want ErrInvalidAPIKey", err)
	}
	if _, err := svc.Authenticate(ctx, ""); err != ErrInvalidAPIKey {
		t.Fatalf("empty key err = %v, want ErrInvalidAPIKey", err)
	}

	if err := svc.Revoke(ctx, created.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := svc.Authenticate(ctx, plaintext); err != ErrInvalidAPIKey {
		t.Fatalf("revoked key err = %v, want ErrInvalidAPIKey", err)
	}
}

func TestCreateRejectsEmptyName(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)

	_, _, err := svc.Create(context.Background(), "   ", domain.APIKeyScopeChatbot, 1)
	if !domain.IsValidationError(err) {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestTouchLastUsedStamps(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)
	ctx := context.Background()

	created, _, err := svc.Create(ctx, "chatbot", domain.APIKeyScopeChatbot, 1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.TouchLastUsed(ctx, created.ID); err != nil {
		t.Fatalf("TouchLastUsed: %v", err)
	}
	if created.LastUsedAt == nil {
		t.Fatalf("last_used_at not stamped")
	}
}

func TestCreateStoresRequestedScope(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)
	ctx := context.Background()

	for _, scope := range domain.ValidAPIKeyScopes {
		key, plaintext, err := svc.Create(ctx, "vendor "+string(scope), scope, 1)
		if err != nil {
			t.Fatalf("Create(%q): %v", scope, err)
		}
		if key.Scope != scope {
			t.Fatalf("stored scope = %q, want %q", key.Scope, scope)
		}

		// The scope must survive the round-trip through Authenticate, since
		// that is what the middleware reads to gate routes.
		got, err := svc.Authenticate(ctx, plaintext)
		if err != nil {
			t.Fatalf("Authenticate(%q): %v", scope, err)
		}
		if !got.HasScope(scope) {
			t.Fatalf("authenticated key scope = %q, want %q", got.Scope, scope)
		}
	}
}

func TestCreateRejectsUnknownScope(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)

	_, _, err := svc.Create(context.Background(), "typo", domain.APIKeyScope("time_sheet_read"), 1)
	if !domain.IsValidationError(err) {
		t.Fatalf("err = %v, want validation error", err)
	}
}

// TestCreateDefaultsEmptyScopeToChatbot keeps older admin clients — which send
// no scope at all — working exactly as before the column existed.
func TestCreateDefaultsEmptyScopeToChatbot(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)

	key, _, err := svc.Create(context.Background(), "legacy", domain.APIKeyScope(""), 1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if key.Scope != domain.APIKeyScopeChatbot {
		t.Fatalf("scope = %q, want %q", key.Scope, domain.APIKeyScopeChatbot)
	}
}

// TestEmptyStoredScopeNeverMatchesVendorSurface is the fail-closed guard: a row
// written before the migration (or by a direct insert) must not silently become
// a vendor credential, and must not act as a wildcard.
func TestEmptyStoredScopeNeverMatchesVendorSurface(t *testing.T) {
	legacy := &domain.APIKey{
		ID:        99,
		Name:      "legacy",
		Scope:     domain.APIKeyScope(""),
		KeyPrefix: "ttk_legacy",
		KeyHash:   HashKey("legacy-plaintext"),
	}

	if legacy.HasScope(domain.APIKeyScopeTimesheetRead) {
		t.Fatal("empty scope must not match timesheet_read")
	}
	if !legacy.HasScope(domain.APIKeyScopeChatbot) {
		t.Fatal("empty scope must resolve to chatbot")
	}

	// An unrecognised non-empty scope matches nothing at all.
	garbage := &domain.APIKey{Scope: domain.APIKeyScope("nonsense")}
	if garbage.HasScope(domain.APIKeyScopeChatbot) || garbage.HasScope(domain.APIKeyScopeTimesheetRead) {
		t.Fatal("unknown scope must not match any surface")
	}
}

// TestScopesAreNotInterchangeable is the cross-tenant guard: a chatbot key
// must never satisfy the vendor gate, and vice versa.
func TestScopesAreNotInterchangeable(t *testing.T) {
	chatbot := &domain.APIKey{Scope: domain.APIKeyScopeChatbot}
	vendor := &domain.APIKey{Scope: domain.APIKeyScopeTimesheetRead}

	if chatbot.HasScope(domain.APIKeyScopeTimesheetRead) {
		t.Fatal("chatbot key must not pass a timesheet_read gate")
	}
	if vendor.HasScope(domain.APIKeyScopeChatbot) {
		t.Fatal("timesheet_read key must not pass a chatbot gate")
	}
}
