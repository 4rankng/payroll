// Package passwordreset implements the self-service email password-reset flow
// for users with an email on file. A user requests a reset link from the login
// page; the service generates a single-use 256-bit token, stores its SHA-256
// hash in Redis (30-min TTL), and emails a branded magic link. The user clicks
// through to a "new password" form whose confirm call atomically consumes the
// token and updates the password + invalidates all existing sessions in one
// DB transaction. See plans/260724-2100-password-reset-email/.
package passwordreset

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/cache"
	"api-server/internal/pkg/clock"
)

// Service orchestrates the self-service password-reset flow. It owns token
// creation/consumption (via PasswordResetTokenStore), password validation +
// hashing (via *user.UserService), and async email dispatch.
type Service struct {
	tokenStore  *cache.PasswordResetTokenStore
	userRepo    domain.UserRepository
	userService *user.UserService
	emailSender domain.EmailDeliveryPort // Red Team C4: reuse otpEmailSender from init.go
	eventBus    domain.EventBus
	fromEmail   string
	resetURL    string
	clk         clock.Clock
	logger      *slog.Logger
}

// NewService constructs the service. emailSender should be the existing
// otpEmailSender variable from bootstrap/services/init.go (it resolves Sandbox
// in dev vs Resend in prod/OTP-enabled) — do NOT re-instantiate a provider.
func NewService(
	tokenStore *cache.PasswordResetTokenStore,
	userRepo domain.UserRepository,
	userService *user.UserService,
	emailSender domain.EmailDeliveryPort,
	eventBus domain.EventBus,
	fromEmail, resetURL string,
	clk clock.Clock,
	logger *slog.Logger,
) *Service {
	if clk == nil {
		clk = clock.New()
	}
	return &Service{
		tokenStore:  tokenStore,
		userRepo:    userRepo,
		userService: userService,
		emailSender: emailSender,
		eventBus:    eventBus,
		fromEmail:   fromEmail,
		resetURL:    resetURL,
		clk:         clk,
		logger:      logger,
	}
}

// RequestReset looks up the user by email and, if found, creates a token and
// emails a magic link asynchronously. It ALWAYS returns nil so the handler can
// return a generic 200 regardless of email existence (anti-enumeration).
//
// Red Team H2 (timing equalization): on the not-found path it performs a dummy
// tokenStore.Create so both paths have identical Redis-RTT + hashing profiles,
// defeating timing oracles.
func (s *Service) RequestReset(ctx context.Context, email string) error {
	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Not found OR transient DB error — treat both as "act like not found"
		// to preserve anti-enumeration. Equalize timing with a dummy Create.
		if _, derr := s.tokenStore.Create(ctx, 0); derr != nil {
			s.logger.Debug("password reset: dummy create failed (ignored)", "error", derr)
		}
		return nil
	}

	token, err := s.tokenStore.Create(ctx, u.ID)
	if err != nil {
		// Don't leak via 500 — the handler still returns the generic 200.
		s.logger.Error("password reset: create token failed", "error", err, "user_id", u.ID)
		return nil
	}

	recEmail, recName := "", u.Fullname
	if u.Email != nil {
		recEmail = *u.Email
	}
	fromEmail := s.fromEmail
	resetURL := s.resetURL
	uid := u.ID
	// Red Team H4: recover panics so a Send failure can't crash the process.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("password reset email goroutine panicked", "panic", r, "user_id", uid)
			}
		}()
		bg, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		msg, err := BuildResetEmailMessage(token, recEmail, recName, fromEmail, resetURL)
		if err != nil {
			s.logger.Error("password reset: build email failed", "error", err, "user_id", uid)
			return
		}
		if _, err := s.emailSender.Send(bg, msg); err != nil {
			s.logger.Error("password reset: send email failed", "error", err, "user_id", uid)
		}
	}()

	return nil
}

// ConfirmReset validates the token, applies password-strength rules, and in a
// single DB transaction (Red Team C1) updates the password + invalidates all
// existing sessions. The token is consumed (atomic GETDEL) before the DB write;
// if the DB transaction fails the token is gone but the account is unchanged,
// and the user requests a new link — the least-bad partial-failure outcome.
func (s *Service) ConfirmReset(ctx context.Context, token string, newPassword string) error {
	userID, err := s.tokenStore.Consume(ctx, token)
	if err != nil {
		return s.mapConsumeError(err)
	}

	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		// Token pointed at a deleted user; treat as invalid.
		return domain.NewUnauthorizedError(constants.MsgPasswordResetTokenInvalidVN)
	}

	if err := s.userService.ValidatePassword(newPassword); err != nil {
		return domain.NewValidationError(err.Error())
	}

	hashed, err := s.userService.HashNewPassword(newPassword)
	if err != nil {
		return domain.NewInternalError(constants.MsgFailedToHashNewPasswordVN, err)
	}

	// Red Team C1: atomic transaction — password + tokens_invalid_before commit
	// together or not at all. Column-scoped updates (not full-row Save) avoid
	// lost-update on concurrent profile edits.
	if err := s.userRepo.UpdatePasswordAndInvalidateSessions(ctx, u.ID, hashed, s.clk.Now()); err != nil {
		return err
	}

	// Red Team Security-5: the user IS the actor in a self-service reset.
	// Matches NewUserLogoutEvent's zero-actor fix (event_factory_user.go:136).
	// actorFullName empty — the actor is the user themselves; audit relies on
	// actorUserID = u.ID for attribution.
	event := domain.NewPasswordChangedEvent(ctx, u.ID, u.Username, "email_reset", u.ID, "")
	if s.eventBus != nil {
		if err := s.eventBus.Publish(ctx, event); err != nil {
			s.logger.Warn("password reset: failed to publish PasswordChangedEvent", "error", err, "user_id", u.ID)
		}
	}

	s.logger.Info("password reset via email completed", "user_id", u.ID, "username", u.Username)
	return nil
}

// mapConsumeError translates token-store errors into client-facing domain
// errors. Red Team H5: a Redis outage (ErrPasswordResetStoreUnavailable)
// becomes a 500 "try again", NOT a 401 "invalid token" — the user isn't lied
// to that their valid link is expired.
func (s *Service) mapConsumeError(err error) error {
	switch {
	case errors.Is(err, cache.ErrPasswordResetStoreUnavailable):
		return domain.NewInternalError(constants.MsgPasswordResetStoreUnavailableVN, err)
	case errors.Is(err, cache.ErrPasswordResetTokenNotFound):
		return domain.NewUnauthorizedError(constants.MsgPasswordResetTokenInvalidVN)
	default:
		return domain.NewInternalError(constants.MsgPasswordResetStoreUnavailableVN, err)
	}
}
