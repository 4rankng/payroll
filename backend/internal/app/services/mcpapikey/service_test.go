package mcpapikey

import (
	"context"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/services/apikey"
	"api-server/internal/domain"

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
	return NewService(repo, users, nil)
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
