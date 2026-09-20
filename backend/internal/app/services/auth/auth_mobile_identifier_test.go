package auth

import (
	"context"
	"log/slog"
	"testing"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/user"
	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// The test doubles embed the domain repository interfaces and implement only the
// methods these paths use: a method that is unexpectedly reached fails loudly on
// a nil function instead of returning a zero value, and the doubles keep
// compiling when the interfaces grow.
type userRepoStub struct {
	domain.UserRepository
	byUsername  func(ctx context.Context, username string) (*domain.User, error)
	byMobile    func(ctx context.Context, mobile string) (*domain.User, error)
	byID        func(ctx context.Context, id uint) (*domain.User, error)
	queriedForM []string
}

func (s *userRepoStub) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return s.byUsername(ctx, username)
}

func (s *userRepoStub) GetByMobile(ctx context.Context, mobile string) (*domain.User, error) {
	s.queriedForM = append(s.queriedForM, mobile)
	return s.byMobile(ctx, mobile)
}

func (s *userRepoStub) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	return s.byID(ctx, id)
}

type employeeRepoStub struct {
	domain.EmployeeRepository
	byMobile    func(ctx context.Context, mobile string) ([]*domain.Employee, error)
	byCCCD      func(ctx context.Context, cccd string) (*domain.Employee, error)
	queriedForM []string
}

func (s *employeeRepoStub) ListByMobile(ctx context.Context, mobile string) ([]*domain.Employee, error) {
	s.queriedForM = append(s.queriedForM, mobile)
	return s.byMobile(ctx, mobile)
}

func (s *employeeRepoStub) GetByCCCD(ctx context.Context, cccd string) (*domain.Employee, error) {
	return s.byCCCD(ctx, cccd)
}

func newMobileAuthService(users *userRepoStub, employees *employeeRepoStub) *AuthService {
	return &AuthService{
		userService:  &user.UserService{UserRepo: users},
		employeeRepo: employees,
		logger:       slog.Default(),
	}
}

func notFoundUsersRepo() *userRepoStub {
	return &userRepoStub{
		byUsername: func(context.Context, string) (*domain.User, error) {
			return nil, domain.NewNotFoundError("user not found")
		},
		byMobile: func(context.Context, string) (*domain.User, error) {
			return nil, domain.NewNotFoundError("user not found")
		},
		byID: func(context.Context, uint) (*domain.User, error) {
			return nil, domain.NewNotFoundError("user not found")
		},
	}
}

// Login resolves a phone identifier from users.mobile (admin/partner/adv_partner)
// or employees.mobile, and never hands an ambiguous number to one of its matches.
func TestFindUserByMobileIdentifier_ResolvesBothPhoneStores(t *testing.T) {
	t.Run("admin from users.mobile", func(t *testing.T) {
		mobile := "0912345678"
		admin := &domain.User{ID: 3, Username: "admin1", Role: domain.RoleAdmin, Mobile: &mobile}
		users := notFoundUsersRepo()
		users.byMobile = func(_ context.Context, mobile string) (*domain.User, error) {
			if mobile == "0912345678" {
				return admin, nil
			}
			return nil, domain.NewNotFoundError("user not found")
		}
		svc := newMobileAuthService(users, &employeeRepoStub{})

		got, err := svc.findUserByMobileIdentifier(context.Background(), "+84 912 345 678")
		require.NoError(t, err)
		require.Equal(t, uint(3), got.ID)
		require.Equal(t, []string{"0912345678"}, users.queriedForM, "the country-code form must be normalized before lookup")
	})

	t.Run("employee from employees.mobile", func(t *testing.T) {
		employeeUserID := uint(7)
		employeeUser := &domain.User{ID: employeeUserID, Username: "emp1", Role: domain.RoleEmployee}
		users := notFoundUsersRepo()
		users.byID = func(_ context.Context, id uint) (*domain.User, error) {
			require.Equal(t, employeeUserID, id)
			return employeeUser, nil
		}
		employees := &employeeRepoStub{byMobile: func(_ context.Context, mobile string) ([]*domain.Employee, error) {
			if mobile == "0987654321" {
				return []*domain.Employee{{ID: 12, UserID: &employeeUserID, Mobile: mobile}}, nil
			}
			return nil, nil
		}}
		svc := newMobileAuthService(users, employees)

		got, err := svc.findUserByMobileIdentifier(context.Background(), "0987654321")
		require.NoError(t, err)
		require.Equal(t, employeeUserID, got.ID)
		require.Equal(t, []string{"0987654321"}, employees.queriedForM)
	})

	t.Run("number shared by two employees is a conflict", func(t *testing.T) {
		firstID, secondID := uint(41), uint(42)
		users := notFoundUsersRepo()
		users.byID = func(context.Context, uint) (*domain.User, error) {
			t.Error("an ambiguous number must not be resolved to any account")
			return nil, nil
		}
		employees := &employeeRepoStub{byMobile: func(_ context.Context, mobile string) ([]*domain.Employee, error) {
			return []*domain.Employee{
				{ID: firstID, UserID: &firstID, Mobile: mobile},
				{ID: secondID, UserID: &secondID, Mobile: mobile},
			}, nil
		}}
		svc := newMobileAuthService(users, employees)

		got, err := svc.findUserByMobileIdentifier(context.Background(), "0374990377")
		require.Nil(t, got)
		require.True(t, domain.IsConflictError(err), "want conflict error, got %v", err)
	})

	t.Run("unknown number is not found", func(t *testing.T) {
		employees := &employeeRepoStub{byMobile: func(context.Context, string) ([]*domain.Employee, error) {
			return nil, nil
		}}
		svc := newMobileAuthService(notFoundUsersRepo(), employees)

		got, err := svc.findUserByMobileIdentifier(context.Background(), "0900000000")
		require.Nil(t, got)
		require.True(t, domain.IsNotFoundError(err), "want not-found, got %v", err)
	})
}

// The whole Login flow must reject a number carried by two employees: no token,
// and nothing that would tell an attacker the number exists.
func TestLogin_AmbiguousEmployeeMobile_NoToken(t *testing.T) {
	firstID, secondID := uint(41), uint(42)
	employees := &employeeRepoStub{
		byMobile: func(_ context.Context, mobile string) ([]*domain.Employee, error) {
			return []*domain.Employee{
				{ID: firstID, UserID: &firstID, Mobile: mobile},
				{ID: secondID, UserID: &secondID, Mobile: mobile},
			}, nil
		},
		byCCCD: func(context.Context, string) (*domain.Employee, error) {
			return nil, domain.NewNotFoundError("employee not found")
		},
	}
	svc := newMobileAuthService(notFoundUsersRepo(), employees)

	resp, err := svc.Login(context.Background(), dto.LoginRequest{Username: "0374990377", Password: "whatever"}, "", "")

	require.Nil(t, resp, "no response (and therefore no token) for an ambiguous number")
	require.True(t, domain.IsUnauthorizedError(err), "want unauthorized, got %v", err)
}
