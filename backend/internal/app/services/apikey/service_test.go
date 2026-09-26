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

	key, plaintext, err := svc.Create(context.Background(), "Zalo Chatbot", 7)
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

	created, plaintext, err := svc.Create(ctx, "chatbot", 1)
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

	_, _, err := svc.Create(context.Background(), "   ", 1)
	if !domain.IsValidationError(err) {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestTouchLastUsedStamps(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)
	ctx := context.Background()

	created, _, err := svc.Create(ctx, "chatbot", 1)
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
