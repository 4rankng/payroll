package zaloreset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/app/services/otp"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/cache"
	"api-server/internal/infra/zalo"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/phone"
)

// ErrFeatureDisabled is returned by confirm-style operations when the Zalo
// reset service was never wired (the feature is off at the bootstrap level).
// RequestReset handles this gracefully (returns a dummy session); confirm
// surfaces it as a 503 so the client knows to retry later.
var ErrFeatureDisabled = errors.New("zalo reset: feature not enabled")

// EnabledChecker is implemented by the zaloconnect service (Phase 4). It lets
// RequestReset hot-check the admin toggle on every call without depending on
// env or config. Returns false if the settings row is missing or disabled.
// A nil checker means "always enabled" (used in dev/tests).
type EnabledChecker interface {
	IsEnabled(ctx context.Context) (bool, error)
}

// ResetStore is the narrow subset of *cache.ZaloResetStore the service needs.
// Defining it here (rather than depending on the concrete type) lets the
// service be unit-tested with a fake store.
type ResetStore interface {
	Create(ctx context.Context, userID uint, codeHashHex string) (string, error)
	CreateDummy(ctx context.Context) (string, error)
	Consume(ctx context.Context, sessionID, codeHashHex string) (uint, error)
}

// userRepo is the narrow subset of domain.UserRepository the service needs.
type userRepo interface {
	GetByMobile(ctx context.Context, mobile string) (*domain.User, error)
	GetByID(ctx context.Context, id uint) (*domain.User, error)
	UpdatePasswordAndInvalidateSessions(ctx context.Context, userID uint, hashedPassword string, invalidBefore time.Time) error
}

// employeeRepo is the narrow subset of domain.EmployeeRepository the service
// needs. Employees' mobile numbers live in the employees table (not users), so
// the reset lookup must also check here. The linked user account (if any) is
// reached via Employee.UserID.
type employeeRepo interface {
	GetByMobile(ctx context.Context, mobile string) (*domain.Employee, error)
}

// passwordService is the narrow subset of *user.UserService the service needs.
type passwordService interface {
	ValidatePassword(password string) error
	HashNewPassword(password string) (string, error)
}

// Service orchestrates the self-service Zalo-OTP password-reset flow.
type Service struct {
	store          ResetStore
	userRepo       userRepo
	employeeRepo   employeeRepo // may be nil — when nil, only users.mobile is checked
	userService    passwordService
	zalo           zalo.Sender
	templateID     string
	codeTTL        time.Duration // drives the reset-session lifetime
	enabled        EnabledChecker
	resendCooldown time.Duration
	eventBus       domain.EventBus
	clk            clock.Clock
	logger         *slog.Logger
}

// NewService constructs the service. codeTTL drives the reset-session lifetime.
// enabled may be nil (always-on, dev).
// resendCooldown defaults to 60s when zero.
func NewService(
	store ResetStore,
	userRepo userRepo,
	employeeRepo employeeRepo,
	userService passwordService,
	zaloSender zalo.Sender,
	templateID string,
	codeTTL time.Duration,
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
	if codeTTL <= 0 {
		codeTTL = 10 * time.Minute
	}
	return &Service{
		store:          store,
		userRepo:       userRepo,
		employeeRepo:   employeeRepo,
		userService:    userService,
		zalo:           zaloSender,
		templateID:     templateID,
		codeTTL:        codeTTL,
		enabled:        enabled,
		resendCooldown: 60 * time.Second,
		eventBus:       eventBus,
		clk:            clk,
		logger:         logger,
	}
}

// RequestReset resolves admin/partner numbers from users.mobile and employee
// numbers from employees.mobile, then dispatches a ZNS code. It ALWAYS
// returns (sessionID, nil) and dispatches a structurally identical session id
// for known and unknown mobiles (anti-enumeration).
//
// The returned sessionID is what the client submits to /confirm. For not-found
// or disabled-toggle paths it is a dummy id whose Consume always fails — the
// client cannot tell the difference.
func (s *Service) RequestReset(ctx context.Context, mobile string) (string, error) {
	// Hot-check the admin toggle. If off, behave exactly like a not-found:
	// dummy session, no ZNS dispatch. Anti-enumeration preserved.
	if s.enabled != nil {
		if on, _ := s.enabled.IsEnabled(ctx); !on {
			return s.dummy(ctx)
		}
	}

	normalizedMobile, err := phone.NormalizeVietnameseMobile(mobile)
	if err != nil {
		return s.dummy(ctx)
	}

	u, err := s.resolveUserByMobile(ctx, normalizedMobile, mobile)
	if err != nil {
		// Not found in either table — dummy path (anti-enumeration).
		return s.dummy(ctx)
	}

	code, err := otp.GenerateCode()
	if err != nil {
		// Code-gen failure is a transient infra error; degrade to dummy so we
		// don't leak a 500 that distinguishes this mobile from others.
		s.logger.Error("zalo reset: generate code failed", "error", err, "user_id", u.ID)
		return s.dummy(ctx)
	}
	codeHash := hashHex(otp.HashCode(code))

	sessionID, err := s.store.Create(ctx, u.ID, codeHash)
	if err != nil {
		s.logger.Error("zalo reset: store create failed", "error", err, "user_id", u.ID)
		return s.dummy(ctx)
	}

	// Async dispatch — never blocks the response, never crashes the process.
	recipMobile := normalizedMobile
	tpl := s.templateID
	uid := u.ID
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("zalo reset: send goroutine panicked", "panic", r, "user_id", uid)
			}
		}()
		bg, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		trackingID := fmt.Sprintf("pwreset_%d_%d", uid, s.clk.Now().UnixNano())
		res, sendErr := s.zalo.Send(bg, recipMobile, tpl, trackingID, map[string]string{
			"otp": code,
		})
		if sendErr != nil {
			s.logger.Error("zalo reset: send failed (transport)", "error", sendErr, "user_id", uid)
			return
		}
		if res.ErrorCode != 0 {
			// Business errors (-118 no Zalo account, -115 quota, -127 sandbox) are
			// expected and not crash-worthy. Log at Info so ops can see delivery
			// issues without paging on every -118.
			s.logger.Info("zalo reset: zns business error",
				"user_id", uid, "error_code", res.ErrorCode, "error_msg", res.ErrorMsg)
			return
		}
		s.logger.Info("zalo reset: code sent", "user_id", uid, "msg_id", res.MsgID)
	}()

	return sessionID, nil
}

func (s *Service) resolveUserByMobile(ctx context.Context, normalizedMobile, originalMobile string) (*domain.User, error) {
	candidates := []string{normalizedMobile}
	if originalMobile != normalizedMobile {
		candidates = append(candidates, originalMobile)
	}
	for _, candidate := range candidates {
		if accountUser, err := s.userRepo.GetByMobile(ctx, candidate); err == nil &&
			(accountUser.Role == domain.RoleAdmin || accountUser.Role == domain.RolePartner) {
			return accountUser, nil
		}
	}

	if s.employeeRepo == nil {
		return nil, domain.NewNotFoundError("user not found")
	}
	for _, candidate := range candidates {
		employee, err := s.employeeRepo.GetByMobile(ctx, candidate)
		if err != nil || employee.UserID == nil {
			continue
		}
		accountUser, err := s.userRepo.GetByID(ctx, *employee.UserID)
		if err == nil && accountUser.Role == domain.RoleEmployee {
			return accountUser, nil
		}
	}
	return nil, domain.NewNotFoundError("user not found")
}

// dummy creates a placeholder session for the not-found / disabled path and
// returns nil. The session id is structurally identical to a real one.
func (s *Service) dummy(ctx context.Context) (string, error) {
	sid, err := s.store.CreateDummy(ctx)
	if err != nil {
		// Redis is down — we can't even fake it. Return a syntactically-valid
		// random id so the client still gets a response; its confirm will 500
		// later. Anti-enumeration is best-effort under Redis outage.
		s.logger.Error("zalo reset: dummy create failed (redis down?)", "error", err)
		return sid, nil // sid is "" here; the client still proceeds
	}
	return sid, nil
}

// ConfirmReset validates the code, applies password-strength rules, and in a
// single DB transaction updates the password + invalidates all existing
// sessions. The code is consumed (atomic) before the DB write; if the DB
// transaction fails the code is gone but the account is unchanged, and the user
// requests a new code — the least-bad partial-failure outcome.
func (s *Service) ConfirmReset(ctx context.Context, sessionID, code, newPassword string) error {
	userID, err := s.store.Consume(ctx, sessionID, hashHex(otp.HashCode(code)))
	if err != nil {
		return s.mapConsumeError(err)
	}

	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		// Session pointed at a deleted user — treat as invalid.
		return domain.NewUnauthorizedError(constants.MsgZaloResetCodeInvalidVN)
	}

	if err := s.userService.ValidatePassword(newPassword); err != nil {
		return domain.NewValidationError(err.Error())
	}

	hashed, err := s.userService.HashNewPassword(newPassword)
	if err != nil {
		return domain.NewInternalError(constants.MsgFailedToHashNewPasswordVN, err)
	}

	// Atomic transaction: password + tokens_invalid_before commit together.
	if err := s.userRepo.UpdatePasswordAndInvalidateSessions(ctx, u.ID, hashed, s.clk.Now()); err != nil {
		return err
	}

	// Audit event — actor is the user themselves (matches the email reset's
	// Red Team Security-5 fix).
	event := domain.NewPasswordChangedEvent(ctx, u.ID, u.Username, "zalo_reset", u.ID, "")
	if s.eventBus != nil {
		if err := s.eventBus.Publish(ctx, event); err != nil {
			s.logger.Warn("zalo reset: publish PasswordChangedEvent failed", "error", err, "user_id", u.ID)
		}
	}

	s.logger.Info("zalo reset: password changed", "user_id", u.ID, "username", u.Username)
	return nil
}

// mapConsumeError translates store errors into client-facing domain errors.
// A Redis outage becomes a 500 "try again" (don't lie that the code is expired);
// a missing/expired/consumed session or wrong code becomes a 401 invalid.
func (s *Service) mapConsumeError(err error) error {
	switch {
	case errors.Is(err, cache.ErrZaloResetStoreUnavailable):
		return domain.NewInternalError(constants.MsgZaloResetStoreDownVN, err)
	case errors.Is(err, cache.ErrZaloResetSessionNotFound), errors.Is(err, cache.ErrZaloResetInvalidCode):
		// Not-found and wrong-code map to the same message so the client can't
		// distinguish "no such session" from "wrong digits" — the latter would
		// confirm the session id is valid, a minor enumeration aid.
		return domain.NewUnauthorizedError(constants.MsgZaloResetCodeInvalidVN)
	default:
		return domain.NewInternalError(constants.MsgZaloResetStoreDownVN, err)
	}
}

// hashHex converts the otp.HashCode byte slice to its hex string form for
// storage + Lua comparison.
func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
