package employee

import (
	"context"
	"errors"
	"testing"

	"api-server/internal/domain"
)

type uniqueUsernameUserRepo struct {
	domain.UserRepository
	prefixResult []string
	prefixErr    error
}

func (r *uniqueUsernameUserRepo) GetUsernamesByPrefix(context.Context, string) ([]string, error) {
	return r.prefixResult, r.prefixErr
}

// TestEnsureUniqueUsername_PropagatesPrefixQueryError verifies a failed prefix
// lookup aborts account creation instead of silently reusing the base username,
// which would collide at INSERT time.
func TestEnsureUniqueUsername_PropagatesPrefixQueryError(t *testing.T) {
	service := &EmployeeService{
		UserRepo: &uniqueUsernameUserRepo{prefixErr: errors.New("db down")},
	}

	_, err := service.ensureUniqueUsername(context.Background(), "quyenlv")
	if err == nil {
		t.Fatal("expected error when prefix lookup fails, got nil")
	}
}

// TestEnsureUniqueUsername_SuffixesWhenBaseTaken verifies the numeric-suffix
// fallback still applies when committed usernames conflict.
func TestEnsureUniqueUsername_SuffixesWhenBaseTaken(t *testing.T) {
	service := &EmployeeService{
		UserRepo: &uniqueUsernameUserRepo{prefixResult: []string{"quyenlv", "quyenlv2"}},
	}

	got, err := service.ensureUniqueUsername(context.Background(), "quyenlv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "quyenlv3" {
		t.Fatalf("expected quyenlv3, got %q", got)
	}
}

// TestEnsureUniqueUsername_BaseAvailable verifies the base username is kept
// when no usernames share the prefix.
func TestEnsureUniqueUsername_BaseAvailable(t *testing.T) {
	service := &EmployeeService{
		UserRepo: &uniqueUsernameUserRepo{prefixResult: nil},
	}

	got, err := service.ensureUniqueUsername(context.Background(), "quyenlv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "quyenlv" {
		t.Fatalf("expected quyenlv, got %q", got)
	}
}
