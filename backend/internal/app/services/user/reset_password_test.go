package user

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeResetUserRepo implements domain.UserRepository, recording which write
// path ResetUserPassword took. It embeds the interface so only the methods this
// test needs have real behaviour; any other call panics, which is itself the
// assertion — a full-row Save would land on Update.
type fakeResetUserRepo struct {
	domain.UserRepository

	mu sync.Mutex

	user *domain.User

	pwdUpdates    int
	invalidBefore []time.Time
	fullRowSaves  int
	lastHash      string
}

func (r *fakeResetUserRepo) GetByID(_ context.Context, id uint) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil || r.user.ID != id {
		return nil, domain.NewNotFoundError("user not found")
	}
	// Return a copy so a full-row Save cannot be observed via shared state.
	c := *r.user
	return &c, nil
}

// UpdatePasswordAndInvalidateSessions is the column-scoped path we expect.
func (r *fakeResetUserRepo) UpdatePasswordAndInvalidateSessions(_ context.Context, userID uint, hashed string, invalidBefore time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.user == nil || r.user.ID != userID {
		return domain.NewNotFoundError("user not found")
	}
	r.pwdUpdates++
	r.lastHash = hashed
	r.invalidBefore = append(r.invalidBefore, invalidBefore)
	r.user.Password = hashed
	r.user.TokensInvalidBefore = &invalidBefore
	return nil
}

// Update is the full-row Save path we must NOT take.
func (r *fakeResetUserRepo) Update(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fullRowSaves++
	if r.user == nil {
		return domain.NewNotFoundError("user not found")
	}
	r.user = user
	return nil
}

// Get is reached by auditservice.GetActorFullName when resolving the actor.
func (r *fakeResetUserRepo) Get(ctx context.Context, id uint) (*domain.User, error) {
	return r.GetByID(ctx, id)
}

type fakeResetEventBus struct {
	domain.EventBus
	mu     sync.Mutex
	events []domain.DomainEvent
}

func (b *fakeResetEventBus) Publish(_ context.Context, events ...domain.DomainEvent) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, events...)
	return nil
}

func newResetUserService(repo *fakeResetUserRepo, bus domain.EventBus) *UserService {
	svc := NewUserService(repo, nil, bus, "test-secret", "test-salt", nil)
	svc.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	return svc
}

func validResetFixture() *domain.User {
	// A sparse profile on purpose: an empty fullname/short username must not
	// block a credential reset, which is exactly what UserRepo.Update's
	// IsValid() gate used to do.
	return &domain.User{
		ID:       40,
		Username: "ab",
		Fullname: "",
		Role:     domain.RoleEmployee,
	}
}

// TestResetUserPassword_UsesColumnScopedUpdate pins the primary defect:
// ResetUserPassword must not perform a full-row Save. A full-row Save both
// clobbers concurrent profile edits and leaves tokens_invalid_before untouched,
// so sessions issued under the old password keep working — the reset looks
// successful while the previous credential stays live.
func TestResetUserPassword_UsesColumnScopedUpdate(t *testing.T) {
	repo := &fakeResetUserRepo{user: validResetFixture()}
	bus := &fakeResetEventBus{}
	svc := newResetUserService(repo, bus)

	require.NoError(t, svc.ResetUserPassword(context.Background(), 40, "N3wPassword!"))

	assert.Equal(t, 0, repo.fullRowSaves,
		"ResetUserPassword must not use the full-row Save path (UserRepo.Update)")
	assert.Equal(t, 1, repo.pwdUpdates,
		"ResetUserPassword must persist via UpdatePasswordAndInvalidateSessions")
	assert.NotEmpty(t, repo.lastHash)
	assert.NotEqual(t, "N3wPassword!", repo.lastHash, "password must be stored hashed, never in clear")
}

// TestResetUserPassword_InvalidatesExistingSessions covers the user-visible
// half of the same defect: after a reset, tokens minted under the old password
// must be rejected.
func TestResetUserPassword_InvalidatesExistingSessions(t *testing.T) {
	repo := &fakeResetUserRepo{user: validResetFixture()}
	svc := newResetUserService(repo, &fakeResetEventBus{})

	require.NoError(t, svc.ResetUserPassword(context.Background(), 40, "N3wPassword!"))

	require.Len(t, repo.invalidBefore, 1, "tokens_invalid_before must be advanced")
	assert.False(t, repo.invalidBefore[0].IsZero(),
		"tokens_invalid_before must carry the reset instant, not a zero time")
}

// TestResetUserPassword_RejectsWeakPassword keeps the existing policy gate.
func TestResetUserPassword_RejectsWeakPassword(t *testing.T) {
	repo := &fakeResetUserRepo{user: validResetFixture()}
	svc := newResetUserService(repo, &fakeResetEventBus{})

	err := svc.ResetUserPassword(context.Background(), 40, "weak")

	require.Error(t, err)
	assert.Equal(t, 0, repo.pwdUpdates, "a rejected password must not be written")
}
