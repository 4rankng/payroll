package auth

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/user"
	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type rehashUserRepo struct {
	domain.UserRepository
	user       *domain.User
	rehashed   int
	storedHash string
}

func (r *rehashUserRepo) GetByUsername(context.Context, string) (*domain.User, error) {
	return r.user, nil
}

func (r *rehashUserRepo) GetByID(context.Context, uint) (*domain.User, error) {
	return r.user, nil
}

func (r *rehashUserRepo) UpdatePasswordAndInvalidateSessions(_ context.Context, _ uint, hashed string, _ time.Time) error {
	r.rehashed++
	r.storedHash = hashed
	return nil
}

func (r *rehashUserRepo) UpdateLastLogin(context.Context, uint, time.Time) error { return nil }

func (r *rehashUserRepo) UpdateOTPLockout(context.Context, uint, int, *time.Time) error { return nil }

// Login must heal legacy bcrypt hashes transparently: the correct password
// verifies through the bcrypt fallback in VerifyPasswordHash, then the stored
// hash is upgraded to Argon2id without failing the login (accounts nv478/nv548
// could never log in before this).
func TestLogin_RehashesLegacyBcryptPassword(t *testing.T) {
	plain := "Password123!"
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	require.NoError(t, err)

	repo := &rehashUserRepo{
		user: &domain.User{
			ID:       1417,
			Username: "nv478",
			Fullname: "NV 478",
			Role:     domain.RoleEmployee,
			Password: string(bcryptHash),
		},
	}

	svc := &AuthService{
		userService: user.NewUserService(repo, nil, nil, "hash-secret", "hash-salt", nil),
		logger:      slog.Default(),
		jwtSecret:   "jwt-secret",
		accessTTL:   15 * time.Minute,
	}

	res, err := svc.Login(context.Background(), dto.LoginRequest{Username: "nv478", Password: plain}, "", "")
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotEmpty(t, res.AccessToken)

	require.Equal(t, 1, repo.rehashed, "a legacy bcrypt hash must be rehashed on successful login")
	require.True(t, strings.HasPrefix(repo.storedHash, "$argon2id$"), "stored hash must be Argon2id, got %q", repo.storedHash)
	require.True(t, svc.userService.VerifyPasswordHash(plain, repo.storedHash), "the rehashed hash must still verify with the same password")
}

// A wrong password on a legacy account must fail exactly as before and must
// NOT trigger a rehash.
func TestLogin_WrongPasswordOnBcryptAccountDoesNotRehash(t *testing.T) {
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	require.NoError(t, err)

	repo := &rehashUserRepo{
		user: &domain.User{
			ID:       1418,
			Username: "nv548",
			Role:     domain.RoleEmployee,
			Password: string(bcryptHash),
		},
	}

	svc := &AuthService{
		userService: user.NewUserService(repo, nil, nil, "hash-secret", "hash-salt", nil),
		logger:      slog.Default(),
		jwtSecret:   "jwt-secret",
		accessTTL:   15 * time.Minute,
	}

	_, err = svc.Login(context.Background(), dto.LoginRequest{Username: "nv548", Password: "wrong"}, "", "")
	require.Error(t, err)
	require.Equal(t, 0, repo.rehashed, "a failed login must never rehash")
}

// Already-Argon2id accounts are untouched: no extra write per login.
func TestLogin_Argon2idAccountIsNotRehashed(t *testing.T) {
	repo := &rehashUserRepo{}

	svc := &AuthService{
		userService: user.NewUserService(repo, nil, nil, "hash-secret", "hash-salt", nil),
		logger:      slog.Default(),
		jwtSecret:   "jwt-secret",
		accessTTL:   15 * time.Minute,
	}

	hashed, err := svc.userService.HashNewPassword("Password123!")
	require.NoError(t, err)

	repo.user = &domain.User{
		ID:       1,
		Username: "admin",
		Role:     domain.RoleEmployee,
		Password: hashed,
	}

	_, err = svc.Login(context.Background(), dto.LoginRequest{Username: "admin", Password: "Password123!"}, "", "")
	require.NoError(t, err)
	require.Equal(t, 0, repo.rehashed, "an Argon2id account must not be rehashed")
}
