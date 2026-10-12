package opsapikey

import (
	"context"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/services/apikey"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// fakeRepo is an in-memory domain.OpsAPIKeyRepository.
type fakeRepo struct {
	byHash map[string]*domain.OpsAPIKey
	byID   map[uint]*domain.OpsAPIKey
	nextID uint
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byHash: map[string]*domain.OpsAPIKey{}, byID: map[uint]*domain.OpsAPIKey{}}
}

func (r *fakeRepo) Create(_ context.Context, key *domain.OpsAPIKey) error {
	r.nextID++
	key.ID = r.nextID
	clone := *key
	r.byHash[key.KeyHash] = &clone
	r.byID[key.ID] = &clone
	return nil
}

func (r *fakeRepo) GetByHash(_ context.Context, hash string) (*domain.OpsAPIKey, error) {
	if key, ok := r.byHash[hash]; ok {
		clone := *key
		return &clone, nil
	}
	return nil, domain.NewNotFoundError("ops api key not found")
}

func (r *fakeRepo) GetByID(_ context.Context, id uint) (*domain.OpsAPIKey, error) {
	if key, ok := r.byID[id]; ok {
		clone := *key
		return &clone, nil
	}
	return nil, domain.NewNotFoundError("ops api key not found")
}

func (r *fakeRepo) List(_ context.Context) ([]*domain.OpsAPIKey, error) { return nil, nil }

func (r *fakeRepo) Revoke(_ context.Context, id uint, at time.Time) error {
	key, ok := r.byID[id]
	if !ok {
		return domain.NewNotFoundError("ops api key not found")
	}
	key.RevokedAt = &at
	if hashed, ok := r.byHash[key.KeyHash]; ok {
		hashed.RevokedAt = &at
	}
	return nil
}

func (r *fakeRepo) UpdateLastUsedAt(_ context.Context, id uint, at time.Time) error {
	if key, ok := r.byID[id]; ok {
		key.LastUsedAt = &at
	}
	return nil
}

func (r *fakeRepo) UpdateScope(_ context.Context, id uint, scope domain.OpsAPIKeyScope, window *time.Time) error {
	key, ok := r.byID[id]
	if !ok || key.RevokedAt != nil {
		return domain.NewNotFoundError("ops api key not found")
	}
	key.Scope = scope
	key.WriteWindowExpiresAt = window
	if hashed, ok := r.byHash[key.KeyHash]; ok {
		hashed.Scope = scope
		hashed.WriteWindowExpiresAt = window
	}
	return nil
}

func (r *fakeRepo) RevertExpiredWriteWindows(_ context.Context, now time.Time) (int64, error) {
	var n int64
	for _, key := range r.byID {
		if key.Scope == domain.OpsAPIKeyScopeReadWrite &&
			key.WriteWindowExpiresAt != nil && now.After(*key.WriteWindowExpiresAt) {
			key.Scope = domain.OpsAPIKeyScopeReadOnly
			key.WriteWindowExpiresAt = nil
			if hashed, ok := r.byHash[key.KeyHash]; ok {
				hashed.Scope = domain.OpsAPIKeyScopeReadOnly
				hashed.WriteWindowExpiresAt = nil
			}
			n++
		}
	}
	return n, nil
}

// fakeUsers implements only GetByID; other UserRepository methods are never
// reached by the service.
type fakeUsers struct {
	domain.UserRepository
	byID map[uint]*domain.User
}

func (f *fakeUsers) GetByID(_ context.Context, id uint) (*domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domain.NewNotFoundError("user not found")
}

func newService(repo domain.OpsAPIKeyRepository, users domain.UserRepository, clk clock.Clock) *Service {
	return NewService(repo, users, nil, clk)
}

func activeUser(id uint) *domain.User {
	return &domain.User{ID: id, Username: "admin", Role: domain.RoleAdmin}
}

func TestCreateDefaultsToReadOnlyPkOpsNamespace(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	svc := newService(repo, users, nil)

	key, plaintext, err := svc.Create(context.Background(), "hermes", "", 7, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.HasPrefix(plaintext, "pk_ops_") {
		t.Fatalf("plaintext must live in the pk_ops_ namespace, got %q", plaintext[:10])
	}
	if key.Scope != domain.OpsAPIKeyScopeReadOnly {
		t.Fatalf("default scope must be read_only, got %q", key.Scope)
	}
	if key.KeyPrefix != plaintext[:12] {
		t.Fatalf("stored prefix must be the first 12 plaintext chars")
	}
	if key.KeyHash != apikey.HashKey(plaintext) {
		t.Fatalf("stored hash must be sha256 of the plaintext")
	}
	if key.WriteWindowExpiresAt != nil {
		t.Fatalf("read_only key must carry no write window")
	}
}

func TestCreateReadWriteOpensWriteWindow(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	clk := clock.NewFake(time.Date(2026, 10, 12, 9, 0, 0, 0, time.Local))
	svc := newService(repo, users, clk)

	hours := 4
	key, _, err := svc.Create(context.Background(), "hermes", domain.OpsAPIKeyScopeReadWrite, 7, &hours)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	want := clk.Now().Add(4 * time.Hour)
	if key.WriteWindowExpiresAt == nil || !key.WriteWindowExpiresAt.Equal(want) {
		t.Fatalf("write window must expire 4h from now, got %v want %v", key.WriteWindowExpiresAt, want)
	}
}

func TestAuthenticateResolvesCreatorAndRejectsBadKeys(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	svc := newService(repo, users, nil)
	ctx := context.Background()
	_, plaintext, _ := svc.Create(ctx, "hermes", "", 7, nil)

	if _, _, err := svc.Authenticate(ctx, plaintext); err != nil {
		t.Fatalf("valid key must authenticate: %v", err)
	}
	if _, _, err := svc.Authenticate(ctx, "pk_ops_unknown"); err == nil {
		t.Fatalf("unknown key must fail")
	}
	// Deleting/deactivating the creator disables the key.
	if _, _, err := svc.Authenticate(ctx, plaintext); err != nil {
		t.Fatalf("setup: %v", err)
	}
	emptyUsers := &fakeUsers{byID: map[uint]*domain.User{}}
	svc2 := NewService(repo, emptyUsers, nil, nil)
	if _, _, err := svc2.Authenticate(ctx, plaintext); err == nil {
		t.Fatalf("orphaned key must fail closed")
	}
	// Revocation disables the key.
	if err := svc.Revoke(ctx, 1, 7); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, _, err := svc.Authenticate(ctx, plaintext); err == nil {
		t.Fatalf("revoked key must fail closed")
	}
}

// TestEffectiveScopeWriteWindowLapse is the stale-scope contract: a
// read_write grant whose window has lapsed reads as read_only, without any
// background job — evaluation is per request.
func TestEffectiveScopeWriteWindowLapse(t *testing.T) {
	before := time.Date(2026, 10, 12, 9, 0, 0, 0, time.Local)
	clk := clock.NewFake(before)
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	svc := newService(repo, users, clk)
	_, plaintext, _ := svc.Create(context.Background(), "hermes", domain.OpsAPIKeyScopeReadWrite, 7, nil)

	key, _, err := svc.Authenticate(context.Background(), plaintext)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if got := key.EffectiveScope(clk.Now()); got != domain.OpsAPIKeyScopeReadWrite {
		t.Fatalf("inside the window the scope must be read_write, got %q", got)
	}

	clk.Advance(9 * time.Hour) // default window is 8h
	key, _, err = svc.Authenticate(context.Background(), plaintext)
	if err != nil {
		t.Fatalf("Authenticate after lapse: %v", err)
	}
	if got := key.EffectiveScope(clk.Now()); got != domain.OpsAPIKeyScopeReadOnly {
		t.Fatalf("after the window lapses the effective scope must be read_only, got %q", got)
	}
	if !key.WriteWindowExpired(clk.Now()) {
		t.Fatalf("WriteWindowExpired must report the lapse")
	}
}

// TestSetScopeAppliesImmediately is the stale-scope contract for the admin
// toggle: flipping to read_only applies to the very next authenticate call.
func TestSetScopeAppliesImmediately(t *testing.T) {
	clk := clock.NewFake(time.Date(2026, 10, 12, 9, 0, 0, 0, time.Local))
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	svc := newService(repo, users, clk)
	ctx := context.Background()
	hours := 2
	_, plaintext, _ := svc.Create(ctx, "hermes", domain.OpsAPIKeyScopeReadWrite, 7, &hours)

	if _, err := svc.SetScope(ctx, 1, domain.OpsAPIKeyScopeReadOnly, nil, 7); err != nil {
		t.Fatalf("SetScope: %v", err)
	}
	key, _, err := svc.Authenticate(ctx, plaintext)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if key.Scope != domain.OpsAPIKeyScopeReadOnly || key.WriteWindowExpiresAt != nil {
		t.Fatalf("toggle to read_only must clear the window, got scope=%q window=%v", key.Scope, key.WriteWindowExpiresAt)
	}

	// Toggle back re-opens the window from NOW.
	if _, err := svc.SetScope(ctx, 1, domain.OpsAPIKeyScopeReadWrite, nil, 7); err != nil {
		t.Fatalf("SetScope: %v", err)
	}
	key, _ = svc.GetByID(ctx, 1)
	want := clk.Now().Add(DefaultWriteWindowHours * time.Hour)
	if key.WriteWindowExpiresAt == nil || !key.WriteWindowExpiresAt.Equal(want) {
		t.Fatalf("renewed window must run DefaultWriteWindowHours from now, got %v want %v", key.WriteWindowExpiresAt, want)
	}
}

func TestSetScopeUnknownKeyIsNotFound(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	svc := newService(repo, users, nil)

	if _, err := svc.SetScope(context.Background(), 99, domain.OpsAPIKeyScopeReadWrite, nil, 7); err == nil {
		t.Fatalf("unknown key must be not-found")
	}
}
