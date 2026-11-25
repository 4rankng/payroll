package auth

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// The captcha preflight (GET /auth/captcha/required) and the login gate must
// resolve the login identifier identically: a preflight that says "not
// required" while Login still enforces the gate leaves the browser submitting
// without a proof, and the user sees a captcha rejection with no widget to
// solve.

func newCaptchaGateAuthService(users *userRepoStub, employees *employeeRepoStub, captcha *CaptchaService) *AuthService {
	return &AuthService{
		userService:    &user.UserService{UserRepo: users},
		employeeRepo:   employees,
		captchaService: captcha,
		logger:         slog.Default(),
	}
}

// Preflight must resolve an Employee-CCCD identifier exactly like Login does
// (employee.cccd → user_id), otherwise the widget never mounts for employees
// who sign in with their ID number.
func TestCaptchaRequiredForUsername_ResolvesEmployeeCCCD(t *testing.T) {
	captcha, _ := newTestCaptchaService(t, nil)
	cccd := "012345678912"
	userID := uint(7)
	account := &domain.User{ID: userID, Username: "nv001", OTPFailedAttempts: 3}

	employees := &employeeRepoStub{
		byMobile: func(context.Context, string) ([]*domain.Employee, error) {
			return nil, domain.NewNotFoundError("employee not found")
		},
		byCCCD: func(_ context.Context, c string) (*domain.Employee, error) {
			if c != cccd {
				return nil, domain.NewNotFoundError("employee not found")
			}
			return &domain.Employee{CCCD: cccd, UserID: &userID}, nil
		},
	}
	users := notFoundUsersRepo()
	users.byID = func(_ context.Context, id uint) (*domain.User, error) {
		if id == userID {
			return account, nil
		}
		return nil, domain.NewNotFoundError("user not found")
	}

	svc := newCaptchaGateAuthService(users, employees, captcha)
	require.True(t, svc.CaptchaRequiredForUsername(context.Background(), cccd),
		"preflight must answer required=true when the login gate would fire for the same identifier")

	// Same resolution, but the account is below the threshold: preflight and
	// gate must both stay quiet.
	account.OTPFailedAttempts = 2
	require.False(t, svc.CaptchaRequiredForUsername(context.Background(), cccd))
}

// A required captcha with NO payload means the client never mounted a widget:
// the error must say a captcha is required (so the frontend can render one),
// and the failure counter must keep climbing.
func TestLogin_RequiredCaptchaWithoutPayloadNamesCaptcha(t *testing.T) {
	captcha, _ := newTestCaptchaService(t, nil)
	account := &domain.User{ID: 3, Username: "nv001", OTPFailedAttempts: 3}

	users := notFoundUsersRepo()
	users.byUsername = func(_ context.Context, username string) (*domain.User, error) {
		if username == "nv001" {
			return account, nil
		}
		return nil, domain.NewNotFoundError("user not found")
	}
	var incrementedTo []int
	users.updateOTPLockout = func(_ context.Context, _ uint, failedAttempts int, _ *time.Time) error {
		incrementedTo = append(incrementedTo, failedAttempts)
		return nil
	}

	svc := newCaptchaGateAuthService(users, &employeeRepoStub{}, captcha)
	_, err := svc.Login(context.Background(), dto.LoginRequest{Username: "nv001", Password: "whatever"}, "", "")
	require.Error(t, err)
	domainErr, ok := err.(*domain.DomainError)
	require.True(t, ok, "login must return a domain error")
	require.Equal(t, constants.MsgCaptchaRequiredVN, domainErr.Message)
	require.Equal(t, []int{4}, incrementedTo, "the failure counter must climb past the threshold")
}

// A present-but-invalid payload keeps the invalid-captcha copy.
func TestLogin_RequiredCaptchaInvalidPayloadNamesCaptcha(t *testing.T) {
	captcha, _ := newTestCaptchaService(t, nil)
	account := &domain.User{ID: 3, Username: "nv001", OTPFailedAttempts: 3}

	users := notFoundUsersRepo()
	users.byUsername = func(_ context.Context, username string) (*domain.User, error) {
		if username == "nv001" {
			return account, nil
		}
		return nil, domain.NewNotFoundError("user not found")
	}
	users.updateOTPLockout = func(context.Context, uint, int, *time.Time) error { return nil }

	svc := newCaptchaGateAuthService(users, &employeeRepoStub{}, captcha)
	_, err := svc.Login(context.Background(), dto.LoginRequest{
		Username:       "nv001",
		Password:       "whatever",
		CaptchaPayload: "bm90LWEtcmVhbC1wYXlsb2Fk",
	}, "", "")
	require.Error(t, err)
	domainErr, ok := err.(*domain.DomainError)
	require.True(t, ok, "login must return a domain error")
	require.Equal(t, constants.MsgCaptchaFailedVN, domainErr.Message)
}
