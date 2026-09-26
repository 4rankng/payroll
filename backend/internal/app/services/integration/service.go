// Package integration hosts the machine-facing operations the external chatbot
// calls with an API key: a three-step password reset (send OTP → verify OTP →
// reset password) and an employee detail lookup for identity double-check.
//
// Unlike the self-service flow (internal/app/services/zaloreset), which hides
// every outcome behind a dummy session for enumeration resistance and
// dispatches asynchronously, this API reports explicitly whether the account
// was found and whether the OTP was delivered — an authenticated machine caller
// needs those facts to guide the employee.
package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/app/services/identity"
	"api-server/internal/app/services/otp"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/cache"
	"api-server/internal/infra/zalo"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/phone"
)

// Failure-reason values reported by RequestOTP.
const (
	FailureAccountNotFound = "account_not_found"
	FailureZaloDisabled    = "zalo_disabled"
	FailureDeliveryFailed  = "delivery_failed"
)

// EnabledChecker is implemented by zaloconnect.Service; nil means always-on.
type EnabledChecker interface {
	IsEnabled(ctx context.Context) (bool, error)
}

// ResetStore is the subset of *cache.ZaloResetStore the service needs:
// OTP sessions plus post-verification single-use tokens.
type ResetStore interface {
	Create(ctx context.Context, userID uint, codeHashHex string) (string, error)
	Consume(ctx context.Context, sessionID, codeHashHex string) (uint, error)
	CreateVerified(ctx context.Context, userID uint) (string, error)
	ConsumeVerified(ctx context.Context, token string) (uint, error)
	TTL() time.Duration
}

// userRepo is the subset of domain.UserRepository the service needs.
type userRepo interface {
	GetByMobile(ctx context.Context, mobile string) (*domain.User, error)
	GetByID(ctx context.Context, id uint) (*domain.User, error)
	UpdatePasswordAndInvalidateSessions(ctx context.Context, userID uint, hashedPassword string, invalidBefore time.Time) error
}

// employeeRepo is the subset of domain.EmployeeRepository the service needs.
// Employee mobile numbers live in the employees table, so account resolution
// and detail lookup both consult it; GetByUserID supplies the authoritative
// display name for a resolved account.
type employeeRepo interface {
	ListByMobile(ctx context.Context, mobile string) ([]*domain.Employee, error)
	GetByUserID(ctx context.Context, userID uint) (*domain.Employee, error)
}

// passwordService is the subset of *user.UserService the service needs.
type passwordService interface {
	ValidatePassword(password string) error
	HashNewPassword(password string) (string, error)
}

// Service orchestrates the chatbot integration operations.
type Service struct {
	store        ResetStore
	userRepo     userRepo
	employeeRepo employeeRepo
	userService  passwordService
	zalo         zalo.Sender
	templateID   string
	enabled      EnabledChecker
	eventBus     domain.EventBus
	clk          clock.Clock
	logger       *slog.Logger
}

// NewService constructs the service. enabled may be nil (always-on, dev).
func NewService(
	store ResetStore,
	userRepo userRepo,
	employeeRepo employeeRepo,
	userService passwordService,
	zaloSender zalo.Sender,
	templateID string,
	enabled EnabledChecker,
	eventBus domain.EventBus,
	clk clock.Clock,
	logger *slog.Logger,
) *Service {
	if clk == nil {
		clk = clock.New()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		store:        store,
		userRepo:     userRepo,
		employeeRepo: employeeRepo,
		userService:  userService,
		zalo:         zaloSender,
		templateID:   templateID,
		enabled:      enabled,
		eventBus:     eventBus,
		clk:          clk,
		logger:       logger,
	}
}

// RequestResult is the explicit outcome of a send-OTP request.
type RequestResult struct {
	Found             bool
	OTPSent           bool
	SessionID         string
	ExpiresIn         int
	EmployeeName      string
	FailureReason     string // "" when OTPSent; else a Failure* constant
	DeliveryErrorCode int    // Zalo business code when FailureDeliveryFailed
}

// VerifyResult is the outcome of a successful OTP verification.
type VerifyResult struct {
	ResetToken string
	ExpiresIn  int
}

// ResetResult is the outcome of a successful password reset.
type ResetResult struct {
	Username     string
	NewPassword  string
	EmployeeName string
}

// RequestOTP resolves the phone to an account and synchronously dispatches a
// Zalo ZNS OTP, reporting explicitly whether the account existed and whether
// delivery succeeded. Phone normalization happens before the enabled check so a
// disabled OA never masks a malformed number.
func (s *Service) RequestOTP(ctx context.Context, rawPhone string) (*RequestResult, error) {
	recipient, err := phone.NormalizeVietnameseMobile(rawPhone)
	if err != nil {
		return nil, domain.NewValidationError(constants.MsgIntegrationPhoneInvalidVN)
	}

	// Hot-check the admin toggle after the phone check.
	if s.enabled != nil {
		if on, _ := s.enabled.IsEnabled(ctx); !on {
			return &RequestResult{Found: true, FailureReason: FailureZaloDisabled}, nil
		}
	}

	u, emp, err := s.resolveAccount(ctx, rawPhone)
	if err != nil {
		// Not found, or the number is carried by several employees (conflict).
		return &RequestResult{Found: false, FailureReason: FailureAccountNotFound}, nil
	}

	code, err := otp.GenerateCode()
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgIntegrationOTPSendFailedVN, err)
	}

	trackingID := fmt.Sprintf("integ_pwreset_%d_%d", u.ID, s.clk.Now().UnixNano())
	res, sendErr := s.zalo.Send(ctx, recipient, s.templateID, trackingID, map[string]string{"otp": code})
	if sendErr != nil {
		s.logger.Error("integration reset: zns transport error", "error", sendErr, "user_id", u.ID)
		return &RequestResult{Found: true, FailureReason: FailureDeliveryFailed}, nil
	}
	if res.ErrorCode != 0 {
		s.logger.Info("integration reset: zns business error",
			"user_id", u.ID, "error_code", res.ErrorCode, "error_msg", res.ErrorMsg)
		return &RequestResult{
			Found:             true,
			FailureReason:     FailureDeliveryFailed,
			DeliveryErrorCode: res.ErrorCode,
		}, nil
	}

	// Deliverable code → create the session so the employee can verify it.
	sessionID, err := s.store.Create(ctx, u.ID, hashHex(otp.HashCode(code)))
	if err != nil {
		s.logger.Error("integration reset: store create failed", "error", err, "user_id", u.ID)
		return nil, domain.NewInternalError(constants.MsgIntegrationOTPSendFailedVN, err)
	}

	return &RequestResult{
		Found:         true,
		OTPSent:       true,
		SessionID:     sessionID,
		ExpiresIn:     int(s.store.TTL().Seconds()),
		EmployeeName:  displayName(u, emp),
		FailureReason: "",
	}, nil
}

// VerifyOTP consumes the OTP session (single use) and, on success, mints a
// single-use reset token bound to the account. A wrong code leaves the session
// intact for retry within its TTL.
func (s *Service) VerifyOTP(ctx context.Context, sessionID, code string) (*VerifyResult, error) {
	userID, err := s.store.Consume(ctx, sessionID, hashHex(otp.HashCode(code)))
	if err != nil {
		return nil, s.mapConsumeError(err)
	}

	token, err := s.store.CreateVerified(ctx, userID)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgZaloResetStoreDownVN, err)
	}

	return &VerifyResult{
		ResetToken: token,
		ExpiresIn:  int(cache.DefaultZaloResetVerifiedTTL.Seconds()),
	}, nil
}

// ResetPassword consumes a verified reset token (single use) and sets the new
// password, invalidating every existing session. An empty newPassword makes the
// server generate a policy-compliant password and return it.
func (s *Service) ResetPassword(ctx context.Context, resetToken, newPassword string, actorID uint, actorName string) (*ResetResult, error) {
	userID, err := s.store.ConsumeVerified(ctx, resetToken)
	if err != nil {
		return nil, s.mapVerifiedError(err)
	}

	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		// Token pointed at a deleted account — treat as an invalid token.
		return nil, domain.NewUnauthorizedError(constants.MsgIntegrationResetTokenInvalidVN)
	}

	final := newPassword
	if final == "" {
		generated, genErr := GeneratePassword()
		if genErr != nil {
			return nil, domain.NewInternalError(constants.MsgFailedToHashNewPasswordVN, genErr)
		}
		final = generated
	}
	if err := s.userService.ValidatePassword(final); err != nil {
		return nil, domain.NewValidationError(err.Error())
	}

	hashed, err := s.userService.HashNewPassword(final)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgFailedToHashNewPasswordVN, err)
	}

	if err := s.userRepo.UpdatePasswordAndInvalidateSessions(ctx, u.ID, hashed, s.clk.Now()); err != nil {
		return nil, err
	}

	event := domain.NewPasswordChangedEvent(ctx, u.ID, u.Username, "integration_api", actorID, actorName)
	if s.eventBus != nil {
		if err := s.eventBus.Publish(ctx, event); err != nil {
			s.logger.Warn("integration reset: publish PasswordChangedEvent failed", "error", err, "user_id", u.ID)
		}
	}

	s.logger.Info("integration reset: password changed", "user_id", u.ID, "username", u.Username, "actor_api_key", actorID)

	// Prefer the employee record's name so reset agrees with lookup/otp.
	name := u.Fullname
	if emp, empErr := s.employeeRepo.GetByUserID(ctx, u.ID); empErr == nil && emp != nil && emp.Fullname != "" {
		name = emp.Fullname
	}

	return &ResetResult{Username: u.Username, NewPassword: final, EmployeeName: name}, nil
}

// resolveAccount maps a phone to exactly one account via the shared identity
// resolver (same lookup order as login and the self-service reset). When the
// number is carried by a single employee record linked to that account, the
// employee row is returned too — it is the authoritative source for that
// person's name and CCCD, so every endpoint reports the same identity.
func (s *Service) resolveAccount(ctx context.Context, raw string) (*domain.User, *domain.Employee, error) {
	u, err := identity.New(s.userRepo, s.employeeRepo).ResolveUserByMobile(ctx, raw)
	if err != nil {
		return nil, nil, err
	}
	if matches := identity.EmployeesByMobile(ctx, s.employeeRepo, raw); len(matches) == 1 {
		if emp := matches[0]; emp.UserID != nil && *emp.UserID == u.ID {
			return u, emp, nil
		}
	}
	return u, nil, nil
}

// displayName picks the authoritative display name: the employee record's when
// present, otherwise the account's. Shared so RequestOTP and LookupEmployee
// never disagree on the same person's name.
func displayName(u *domain.User, emp *domain.Employee) string {
	if emp != nil && emp.Fullname != "" {
		return emp.Fullname
	}
	return u.Fullname
}

// mapConsumeError translates OTP-session store errors into client-facing
// domain errors: a Redis outage is a 500; a missing/expired/consumed session or
// a wrong code is the same 401 (so a wrong code never reveals a valid session).
func (s *Service) mapConsumeError(err error) error {
	switch {
	case errors.Is(err, cache.ErrZaloResetStoreUnavailable):
		return domain.NewInternalError(constants.MsgZaloResetStoreDownVN, err)
	case errors.Is(err, cache.ErrZaloResetSessionNotFound), errors.Is(err, cache.ErrZaloResetInvalidCode):
		return domain.NewUnauthorizedError(constants.MsgZaloResetCodeInvalidVN)
	default:
		return domain.NewInternalError(constants.MsgZaloResetStoreDownVN, err)
	}
}

// mapVerifiedError translates verified-token store errors into client-facing
// domain errors.
func (s *Service) mapVerifiedError(err error) error {
	switch {
	case errors.Is(err, cache.ErrZaloResetVerifiedNotFound):
		return domain.NewUnauthorizedError(constants.MsgIntegrationResetTokenInvalidVN)
	case errors.Is(err, cache.ErrZaloResetStoreUnavailable):
		return domain.NewInternalError(constants.MsgZaloResetStoreDownVN, err)
	default:
		return domain.NewInternalError(constants.MsgZaloResetStoreDownVN, err)
	}
}

// hashHex converts the otp.HashCode byte slice to its hex string form for
// storage + comparison.
func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
