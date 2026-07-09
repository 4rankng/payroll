// Package otp implements the email-OTP login second factor for admin/partner
// accounts. A successful password check at /auth/login does NOT issue a JWT for
// gated roles; instead OTPService.StartLogin generates a 6-digit code, emails
// it, and stores a SHA hash in Redis under an opaque session id. The client
// then POSTs the code to /auth/login/verify, which (on success) issues the
// real 14-day JWT. See plans/260704-1400-otp-2fa-admin-partner/.
package otp

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/infra/cache"
	"api-server/internal/pkg/clock"
)

// Errors returned to the service layer. Callers map these to HTTP responses.
var (
	// ErrOTPRequiredMissingEmail: a gated role has no email on file — login is
	// refused with a clear message (owner decision: hard block, not silent).
	ErrOTPRequiredMissingEmail = errors.New("otp: account has no email; cannot enable OTP")
	// ErrOTPLocked: the account is in the post-failed-attempts lockout window.
	ErrOTPLocked = errors.New("otp: account temporarily locked")
	// ErrSessionNotFound: the pending session id is absent/expired/superseded.
	ErrSessionNotFound = errors.New("otp: session not found or expired")
	// ErrSessionBindingMismatch: IP/UA of the verify request differs from login.
	ErrSessionBindingMismatch = errors.New("otp: session binding mismatch")
	// ErrInvalidCode: the submitted code did not match.
	ErrInvalidCode = errors.New("otp: invalid code")
	// ErrUserRevokedOrDisabled: account was revoked/disabled mid-session.
	ErrUserRevokedOrDisabled = errors.New("otp: account revoked or disabled during session")
)

// EmailSender is the narrow port OTPService needs to dispatch the code email.
// Implemented by the existing EmailDeliveryPort (ResendProvider via Resend).
type EmailSender interface {
	Send(ctx context.Context, msg *domain.EmailMessage) (*domain.EmailDeliveryResult, error)
}

// OTPService orchestrates the email-OTP login flow. It is the single owner of
// code generation, hashing, Redis pending-session lifecycle, and per-account
// brute-force state (via UserRepository).
type OTPService struct {
	pendingStore *cache.OTPPendingStore
	userRepo     domain.UserRepository
	emailSender  EmailSender
	fromEmail    string
	cfg          config.OTPConfig
	clock        clock.Clock
	logger       *slog.Logger
}

// NewOTPService constructs the service. ttl is the pending-session TTL
// (defaults to cfg.CodeTTL; cache package has its own fallback).
func NewOTPService(
	pendingStore *cache.OTPPendingStore,
	userRepo domain.UserRepository,
	emailSender EmailSender,
	fromEmail string,
	cfg config.OTPConfig,
	clk clock.Clock,
	logger *slog.Logger,
) *OTPService {
	if clk == nil {
		clk = clock.New()
	}
	return &OTPService{
		pendingStore: pendingStore,
		userRepo:     userRepo,
		emailSender:  emailSender,
		fromEmail:    fromEmail,
		cfg:          cfg,
		clock:        clk,
		logger:       logger,
	}
}

// StartLogin runs the OTP gate after a successful password check: it generates a
// code, emails it, and stores a pending session. Returns the opaque session id
// to hand to the client. The caller (AuthService.Login) MUST NOT issue a JWT
// when this path is taken.
//
// Pre-checks (callAuthService.Login is responsible for the role/flag gate; this
// method focuses on email-presence + lockout + dispatch):
//   - ErrOTPRequiredMissingEmail: user.Email is nil/empty.
//   - ErrOTPLocked: user.OTPLockedUntil is in the future.
func (s *OTPService) StartLogin(ctx context.Context, user *domain.User, ip, ua string) (string, error) {
	if user.Email == nil || *user.Email == "" {
		return "", ErrOTPRequiredMissingEmail
	}
	if user.OTPLockedUntil != nil && user.OTPLockedUntil.After(s.clock.Now()) {
		return "", ErrOTPLocked
	}

	code, err := GenerateCode()
	if err != nil {
		return "", err
	}
	codeHash := HashCode(code)

	sessionID, err := s.pendingStore.CreateSession(ctx, user.ID, codeHash, ip, ua)
	if err != nil {
		return "", err
	}

	// Dispatch the email asynchronously so the login response returns immediately.
	// A send failure is logged but does NOT fail StartLogin — the user can hit
	// /auth/login/resend (Phase 3) if the email never arrives.
	recEmail, recName := *user.Email, user.Fullname
	go func() {
		// Use a fresh background context; the request ctx may be cancelled as
		// soon as we return the response.
		bg, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		msg, err := BuildOTPEmailMessage(code, recEmail, recName, s.fromEmail)
		if err != nil {
			s.logger.Error("otp: build email failed", "error", err, "user_id", user.ID)
			return
		}
		if _, err := s.emailSender.Send(bg, msg); err != nil {
			s.logger.Error("otp: send email failed", "error", err, "user_id", user.ID)
			return
		}
		s.logger.Info("otp: code emailed", "user_id", user.ID)
	}()

	return sessionID, nil
}

// VerifyLogin validates the submitted code against the pending session and, on
// success, returns the user whose session it was (so the caller can mint the
// real JWT). On any failure it bumps the per-account failed-attempt counter and
// conditionally locks the account.
//
// RT-M4: this re-checks the user's current lockout/revocation state, because
// the user may have been revoked between StartLogin and VerifyLogin.
// RT-M5: validates IP/UA match the session binding.
// RT-H1: failed attempts count against the account, not the session.
func (s *OTPService) VerifyLogin(ctx context.Context, sessionID, code, ip, ua string) (*domain.User, error) {
	session, err := s.pendingStore.GetSession(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	// RT-M5: bind to the originating client.
	if session.IP != ip || session.UserAgent != ua {
		return nil, ErrSessionBindingMismatch
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	// RT-M4: mid-session revocation / disable.
	if user.OTPLockedUntil != nil && user.OTPLockedUntil.After(s.clock.Now()) {
		return nil, ErrOTPLocked
	}

	if !EqualCodeHash(code, session.CodeHash) {
		s.recordFailedAttempt(ctx, user)
		return nil, ErrInvalidCode
	}

	// Success: clear the session and reset the per-account counter.
	_ = s.pendingStore.DeleteSession(ctx, sessionID)
	if user.OTPFailedAttempts > 0 {
		if err := s.userRepo.UpdateOTPLockout(ctx, user.ID, 0, nil); err != nil {
			s.logger.Error("otp: failed to reset lockout counter after success", "error", err, "user_id", user.ID)
		}
	}
	return user, nil
}

// recordFailedAttempt bumps the per-account counter (RT-H1) and locks when the
// cap is reached. Errors are logged — a counter-update failure must not turn a
// wrong-code into a 500 that leaks whether the code was right.
func (s *OTPService) recordFailedAttempt(ctx context.Context, user *domain.User) {
	attempts := user.OTPFailedAttempts + 1
	var lockedUntil *time.Time
	if attempts >= s.cfg.MaxAttempts {
		t := s.clock.Now().Add(s.cfg.LockDuration)
		lockedUntil = &t
	}
	if err := s.userRepo.UpdateOTPLockout(ctx, user.ID, attempts, lockedUntil); err != nil {
		s.logger.Error("otp: failed to record failed attempt", "error", err, "user_id", user.ID)
	}
}

// DeleteSessionsForUser is called by RevokeUserTokens (RT-M4) so revoking a
// user also kills their in-flight OTP step.
func (s *OTPService) DeleteSessionsForUser(ctx context.Context, userID uint) error {
	return s.pendingStore.DeleteSessionsForUser(ctx, userID)
}

// Errors specific to resend.
var (
	// ErrResendCooldown: a resend was attempted within the configured cooldown.
	ErrResendCooldown = errors.New("otp: resend too soon")
)

// ResendCode re-issues a code for an existing pending session (the email didn't
// arrive, or the user wants a fresh one). It enforces:
//   - session exists & not expired (ErrSessionNotFound)
//   - IP/UA match the session binding (RT-M5: ErrSessionBindingMismatch)
//   - resend cooldown (ErrResendCooldown) to prevent inbox-flooding a victim
//   - account not locked mid-session (RT-M4)
//
// On success the session is overwritten with the new code hash (RT-H1: the old
// code no longer verifies) and a new email is dispatched. Returns the session id
// (unchanged — the client keeps the same id).
func (s *OTPService) ResendCode(ctx context.Context, sessionID, ip, ua string) (string, error) {
	session, err := s.pendingStore.GetSession(ctx, sessionID)
	if err != nil {
		return "", ErrSessionNotFound
	}
	if session.IP != ip || session.UserAgent != ua {
		return "", ErrSessionBindingMismatch
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return "", err
	}
	if user.OTPLockedUntil != nil && user.OTPLockedUntil.After(s.clock.Now()) {
		return "", ErrOTPLocked
	}

	// Cooldown check.
	now := s.clock.Now()
	if !session.LastResendAt.IsZero() && now.Sub(session.LastResendAt) < s.cfg.ResendCooldown {
		return "", ErrResendCooldown
	}

	code, err := GenerateCode()
	if err != nil {
		return "", err
	}
	session.CodeHash = HashCode(code)
	session.LastResendAt = now
	if err := s.pendingStore.SaveSession(ctx, sessionID, session); err != nil {
		return "", err
	}

	recEmail, recName := *user.Email, user.Fullname
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		msg, err := BuildOTPEmailMessage(code, recEmail, recName, s.fromEmail)
		if err != nil {
			s.logger.Error("otp: build resend email failed", "error", err, "user_id", user.ID)
			return
		}
		if _, err := s.emailSender.Send(bg, msg); err != nil {
			s.logger.Error("otp: resend email failed", "error", err, "user_id", user.ID)
			return
		}
		s.logger.Info("otp: code resent", "user_id", user.ID)
	}()

	return sessionID, nil
}
