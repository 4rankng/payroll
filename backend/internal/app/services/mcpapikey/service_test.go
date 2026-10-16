package mcpapikey

import (
	"context"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/services/apikey"
	"api-server/internal/domain"
	asynqinfra "api-server/internal/infra/asynq"

	"gorm.io/gorm"
)

// fakeRepo is an in-memory domain.MCPAPIKeyRepository.
type fakeRepo struct {
	byHash map[string]*domain.MCPAPIKey
	byID   map[uint]*domain.MCPAPIKey
	nextID uint
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byHash: map[string]*domain.MCPAPIKey{}, byID: map[uint]*domain.MCPAPIKey{}}
}

func (r *fakeRepo) Create(_ context.Context, key *domain.MCPAPIKey) error {
	r.nextID++
	key.ID = r.nextID
	clone := *key
	r.byHash[key.KeyHash] = &clone
	r.byID[key.ID] = &clone
	return nil
}

func (r *fakeRepo) GetByHash(_ context.Context, hash string) (*domain.MCPAPIKey, error) {
	if key, ok := r.byHash[hash]; ok {
		clone := *key
		return &clone, nil
	}
	return nil, domain.NewNotFoundError("mcp api key not found")
}

func (r *fakeRepo) GetByID(_ context.Context, id uint) (*domain.MCPAPIKey, error) {
	if key, ok := r.byID[id]; ok {
		clone := *key
		return &clone, nil
	}
	return nil, domain.NewNotFoundError("mcp api key not found")
}

func (r *fakeRepo) List(_ context.Context) ([]*domain.MCPAPIKey, error) { return nil, nil }

func (r *fakeRepo) Revoke(_ context.Context, id uint, at time.Time) error {
	key, ok := r.byID[id]
	if !ok {
		return domain.NewNotFoundError("mcp api key not found")
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

func (r *fakeRepo) UpdateScope(_ context.Context, id uint, scope domain.MCPAPIKeyScope, at time.Time) error {
	key, ok := r.byID[id]
	if !ok || key.RevokedAt != nil {
		return domain.NewNotFoundError("mcp api key not found")
	}
	key.Scope = scope
	key.ScopeUpdatedAt = &at
	if hashed, ok := r.byHash[key.KeyHash]; ok {
		hashed.Scope = scope
		hashed.ScopeUpdatedAt = &at
	}
	return nil
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

func newService(repo domain.MCPAPIKeyRepository, users domain.UserRepository) *Service {
	return NewService(repo, users, nil, nil)
}

func activeUser(id uint) *domain.User {
	return &domain.User{ID: id, Username: "admin", Role: domain.RoleAdmin}
}

func TestCreateMintsScopeAndStoresHash(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	svc := newService(repo, users)
	ctx := context.Background()

	for _, scope := range []domain.MCPAPIKeyScope{domain.MCPAPIKeyScopeReadOnly, domain.MCPAPIKeyScopeFull} {
		key, plaintext, err := svc.Create(ctx, "agent "+string(scope), scope, 7)
		if err != nil {
			t.Fatalf("Create(%q): %v", scope, err)
		}
		if key.Scope != scope {
			t.Fatalf("stored scope = %q, want %q", key.Scope, scope)
		}
		if !strings.HasPrefix(plaintext, "mcp_") {
			t.Fatalf("plaintext %q missing mcp_ prefix", plaintext)
		}
		if key.KeyPrefix != plaintext[:12] {
			t.Fatalf("key_prefix = %q, want %q", key.KeyPrefix, plaintext[:12])
		}
		if key.KeyHash != apikey.HashKey(plaintext) {
			t.Fatal("stored hash does not match sha256(plaintext)")
		}
		if key.CreatedBy != 7 {
			t.Fatalf("created_by = %d, want 7", key.CreatedBy)
		}
		if (key.Scope == domain.MCPAPIKeyScopeReadOnly) != key.IsReadOnly() {
			t.Fatalf("IsReadOnly disagrees with scope %q", key.Scope)
		}
	}
}

func TestCreateRejectsUnknownScope(t *testing.T) {
	svc := newService(newFakeRepo(), &fakeUsers{byID: map[uint]*domain.User{}})

	_, _, err := svc.Create(context.Background(), "typo", domain.MCPAPIKeyScope("read-only"), 1)
	if !domain.IsValidationError(err) {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestAuthenticateReturnsCreatorUser(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	svc := newService(repo, users)
	ctx := context.Background()

	created, plaintext, err := svc.Create(ctx, "agent", domain.MCPAPIKeyScopeReadOnly, 7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	key, user, err := svc.Authenticate(ctx, plaintext)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if key.ID != created.ID {
		t.Fatalf("authenticated id = %d, want %d", key.ID, created.ID)
	}
	if user == nil || user.ID != 7 {
		t.Fatalf("authenticated user = %+v, want the creating admin", user)
	}
}

func TestAuthenticateRejectsRevokedKey(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo, &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}})
	ctx := context.Background()

	_, plaintext, err := svc.Create(ctx, "agent", domain.MCPAPIKeyScopeFull, 7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Revoke(ctx, 1); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	if _, _, err := svc.Authenticate(ctx, plaintext); err != ErrInvalidMCPAPIKey {
		t.Fatalf("err = %v, want ErrInvalidMCPAPIKey", err)
	}
}

func TestAuthenticateRejectsUnknownKey(t *testing.T) {
	svc := newService(newFakeRepo(), &fakeUsers{byID: map[uint]*domain.User{}})

	if _, _, err := svc.Authenticate(context.Background(), "mcp_unknown"); err != ErrInvalidMCPAPIKey {
		t.Fatalf("err = %v, want ErrInvalidMCPAPIKey", err)
	}
	if _, _, err := svc.Authenticate(context.Background(), ""); err != ErrInvalidMCPAPIKey {
		t.Fatalf("err = %v, want ErrInvalidMCPAPIKey", err)
	}
}

func TestAuthenticateRejectsMissingOrInactiveCreator(t *testing.T) {
	ctx := context.Background()

	// Deleted (missing) creator.
	repo := newFakeRepo()
	svc := newService(repo, &fakeUsers{byID: map[uint]*domain.User{}})
	_, plaintext, err := svc.Create(ctx, "orphan", domain.MCPAPIKeyScopeReadOnly, 42)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := svc.Authenticate(ctx, plaintext); err != ErrInvalidMCPAPIKey {
		t.Fatalf("missing creator: err = %v, want ErrInvalidMCPAPIKey", err)
	}

	// Soft-deleted creator is inactive.
	deleted := activeUser(7)
	deleted.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	repo2 := newFakeRepo()
	svc2 := newService(repo2, &fakeUsers{byID: map[uint]*domain.User{7: deleted}})
	_, plaintext2, err := svc2.Create(ctx, "inactive", domain.MCPAPIKeyScopeFull, 7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := svc2.Authenticate(ctx, plaintext2); err != ErrInvalidMCPAPIKey {
		t.Fatalf("inactive creator: err = %v, want ErrInvalidMCPAPIKey", err)
	}
}

// fakeAudit captures audit:log:write payloads in memory.
type fakeAudit struct {
	payloads []asynqinfra.AuditLogWritePayload
}

func (f *fakeAudit) EnqueueAuditLogWrite(p asynqinfra.AuditLogWritePayload) error {
	f.payloads = append(f.payloads, p)
	return nil
}

func TestSetScopeTogglesScopeOnSameKey(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	audit := &fakeAudit{}
	svc := NewService(repo, users, audit, nil)
	ctx := context.Background()

	created, plaintext, err := svc.Create(ctx, "agent", domain.MCPAPIKeyScopeReadOnly, 7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// read_only -> full: same key id, no revoke/recreate.
	updated, err := svc.SetScope(ctx, created.ID, domain.MCPAPIKeyScopeFull, 9)
	if err != nil {
		t.Fatalf("SetScope(full): %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("scope toggle must keep the same key id: got %d, want %d", updated.ID, created.ID)
	}
	if updated.Scope != domain.MCPAPIKeyScopeFull {
		t.Fatalf("scope = %q, want full", updated.Scope)
	}
	if updated.ScopeUpdatedAt == nil {
		t.Fatal("scope_updated_at must be stamped on toggle")
	}

	// The very next authentication sees the new scope (no cache).
	key, _, err := svc.Authenticate(ctx, plaintext)
	if err != nil {
		t.Fatalf("Authenticate after toggle: %v", err)
	}
	if key.IsReadOnly() {
		t.Fatal("key must behave as full right after the toggle")
	}

	// full -> read_only: back on the same key.
	updated, err = svc.SetScope(ctx, created.ID, domain.MCPAPIKeyScopeReadOnly, 9)
	if err != nil {
		t.Fatalf("SetScope(read_only): %v", err)
	}
	if updated.Scope != domain.MCPAPIKeyScopeReadOnly || !updated.IsReadOnly() {
		t.Fatalf("scope = %q, want read_only", updated.Scope)
	}

	if len(audit.payloads) != 2 {
		t.Fatalf("audit rows = %d, want 2", len(audit.payloads))
	}
	first := audit.payloads[0]
	if first.Action != ActionKeyScopeChange || first.EntityType != "mcp_api_key" {
		t.Fatalf("audit row mislabeled: %+v", first)
	}
	if first.UserID != 9 {
		t.Fatalf("audit actor = %d, want 9", first.UserID)
	}
	if got := first.Message; !strings.Contains(got, "old_scope=read_only") || !strings.Contains(got, "new_scope=full") {
		t.Fatalf("audit message must carry old/new scope, got %q", got)
	}
}

func TestSetScopeRejectsValuesOutsideTheClosedEnum(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	svc := NewService(repo, users, nil, nil)
	ctx := context.Background()

	created, _, err := svc.Create(ctx, "agent", domain.MCPAPIKeyScopeReadOnly, 7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// "revoked" (and any other string) must never be settable as a scope —
	// revocation stays on its own endpoint.
	for _, bogus := range []domain.MCPAPIKeyScope{"revoked", "read-write", ""} {
		if _, err := svc.SetScope(ctx, created.ID, bogus, 9); !domain.IsValidationError(err) {
			t.Fatalf("SetScope(%q): err = %v, want validation error", bogus, err)
		}
	}
	if key, _ := repo.byID[created.ID]; key.Scope != domain.MCPAPIKeyScopeReadOnly {
		t.Fatalf("scope must be untouched after rejected toggles, got %q", key.Scope)
	}
}

func TestSetScopeRejectsUnknownAndRevokedKeys(t *testing.T) {
	repo := newFakeRepo()
	users := &fakeUsers{byID: map[uint]*domain.User{7: activeUser(7)}}
	svc := NewService(repo, users, nil, nil)
	ctx := context.Background()

	if _, err := svc.SetScope(ctx, 404, domain.MCPAPIKeyScopeFull, 9); !domain.IsNotFoundError(err) {
		t.Fatalf("unknown key: err = %v, want not-found", err)
	}

	created, _, err := svc.Create(ctx, "agent", domain.MCPAPIKeyScopeReadOnly, 7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Revoke(ctx, created.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := svc.SetScope(ctx, created.ID, domain.MCPAPIKeyScopeFull, 9); !domain.IsNotFoundError(err) {
		t.Fatalf("revoked key: err = %v, want not-found", err)
	}
}
