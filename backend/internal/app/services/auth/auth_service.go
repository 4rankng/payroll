package auth

import (
	"api-server/internal/pkg/clock"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/audit"
	"api-server/internal/app/services/otp"
	"api-server/internal/app/services/user"
	"api-server/internal/config"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/cache"
	auditctx "api-server/internal/pkg/context"
	"api-server/internal/pkg/ipgeo"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/api/idtoken"
)

type AuthService struct {
	userService          *user.UserService
	employeeRepo         domain.EmployeeRepository
	blacklistedTokenRepo domain.BlacklistedTokenRepository
	eventBus             domain.EventBus
	jwtSecret            string
	accessTTL            time.Duration
	logger               *slog.Logger
	otpService           *otp.OTPService
	otpConfig            config.OTPConfig
	googleClientID       string
	nonceStore           *cache.NonceStore
	captchaService       *CaptchaService
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Role     string `json:"role"`
	// OTPVerified marks that the bearer completed the email-OTP second factor.
	// Defaults false; set true ONLY on tokens minted after a successful
	// /auth/login/verify (or on non-gated paths — employee, or admin/partner
	// when OTP_ENABLE=false). Enforced by the Authorize middleware (RT-C1/C2).
	OTPVerified bool `json:"otp_verified,omitempty"`
	jwt.RegisteredClaims
}

func NewAuthService(userService *user.UserService, employeeRepo domain.EmployeeRepository, blacklistedTokenRepo domain.BlacklistedTokenRepository, eventBus domain.EventBus, jwtSecret string, accessTTL time.Duration, otpService *otp.OTPService, otpCfg config.OTPConfig, googleClientID string, nonceStore *cache.NonceStore, captchaService *CaptchaService, logger *slog.Logger) *AuthService {
	return &AuthService{
		userService:          userService,
		employeeRepo:         employeeRepo,
		blacklistedTokenRepo: blacklistedTokenRepo,
		eventBus:             eventBus,
		jwtSecret:            jwtSecret,
		accessTTL:            accessTTL,
		otpService:           otpService,
		otpConfig:            otpCfg,
		googleClientID:       googleClientID,
		nonceStore:           nonceStore,
		captchaService:       captchaService,
		logger:               logger,
	}
}

// requiresOTP reports whether the email-OTP second factor applies to this user.
// Gating conditions: feature flag on AND role is admin or partner. Employees
// and adv_partner are never gated in v1 (low-value accounts, see plan).
func (s *AuthService) requiresOTP(user *domain.User) bool {
	if !s.otpConfig.Enabled || s.otpService == nil {
		return false
	}
	return user.IsAdmin() || user.IsPartner()
}

// CaptchaRequiredForUsername checks whether a CAPTCHA is needed for the next
// login attempt on this account (based on consecutive failures). Used by the
// frontend to decide whether to show the CAPTCHA widget before submitting.
func (s *AuthService) CaptchaRequiredForUsername(ctx context.Context, username string) bool {
	if s.captchaService == nil || !s.captchaService.Enabled() {
		return false
	}
	user, err := s.userService.UserRepo.GetByUsername(ctx, username)
	if err != nil || user == nil {
		return false
	}
	return s.captchaService.RequiredForFailures(user.OTPFailedAttempts)
}

// GenerateCaptcha creates a new image CAPTCHA challenge.
func (s *AuthService) GenerateCaptcha(ctx context.Context) (id, imageBase64 string, err error) {
	if s.captchaService == nil {
		return "", "", domain.NewInternalError("captcha not configured", nil)
	}
	return s.captchaService.Generate(ctx)
}

// mapOTPError translates an OTPService error into the domain error a handler
// will surface as an HTTP response. Missing-email and locked-account become
// Unauthorized with a clear Vietnamese message; other errors pass through as
// internal errors so they 500 rather than fail open.
func (s *AuthService) mapOTPError(err error) error {
	switch {
	case errors.Is(err, otp.ErrOTPRequiredMissingEmail):
		return domain.NewUnauthorizedError(constants.MsgOTPAccountMissingEmailVN)
	case errors.Is(err, otp.ErrOTPLocked):
		return domain.NewUnauthorizedError(constants.MsgOTPAccountLockedVN)
	default:
		return domain.NewInternalError(constants.MsgOTPStartFailedVN, err)
	}
}

// VerifyLoginOTP completes the two-step login: validates the submitted code
// against the pending session, and on success issues the real 14-day JWT with
// otp_verified=true. On failure the per-account failed-attempt counter is
// bumped (RT-H1) and the account may be locked (Phase 6 backoff). The IP/UA
// must match the session binding (RT-M5).
func (s *AuthService) VerifyLoginOTP(ctx context.Context, sessionID, code, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	user, err := s.otpService.VerifyLogin(ctx, sessionID, code, ipAddress, userAgent)
	if err != nil {
		return nil, s.mapVerifyError(err)
	}

	// Issue the real token, now marked as OTP-verified.
	accessToken, err := s.generateAccessToken(user, true)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgFailedToGenerateTokenVN, err)
	}

	// Stamp last-login + success audit (mirrors the password-login path).
	go func(userID uint) {
		bgCtx := context.Background()
		if err := s.userService.UserRepo.UpdateLastLogin(bgCtx, userID, clock.Now()); err != nil {
			s.logger.Error("Failed to update last login after OTP verify", "error", err, "user_id", userID)
		}
	}(user.ID)

	userResponse := dto.ToUserResponse(user)
	return &dto.LoginResponse{
		User:        &userResponse,
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.accessTTL.Seconds()),
	}, nil
}

// mapVerifyError maps OTP verify failures to client-facing domain errors.
// Session-not-found / binding-mismatch / invalid-code all surface as 401 with
// a generic "invalid code or session" message to avoid leaking which failed.
func (s *AuthService) mapVerifyError(err error) error {
	switch {
	case errors.Is(err, otp.ErrOTPLocked), errors.Is(err, otp.ErrUserRevokedOrDisabled):
		return domain.NewUnauthorizedError(constants.MsgOTPAccountLockedVN)
	case errors.Is(err, otp.ErrSessionNotFound),
		errors.Is(err, otp.ErrSessionBindingMismatch),
		errors.Is(err, otp.ErrInvalidCode):
		return domain.NewUnauthorizedError(constants.MsgOTPInvalidCodeOrSessionVN)
	default:
		return domain.NewInternalError(constants.MsgOTPVerifyFailedVN, err)
	}
}

// ResendOTPCode re-issues an OTP code for a pending session (email didn't
// arrive). Respects the resend cooldown and per-account lockout. Returns the
// session id (unchanged) and the remaining TTL in seconds.
func (s *AuthService) ResendOTPCode(ctx context.Context, sessionID, ipAddress, userAgent string) (string, int64, error) {
	sessionID, err := s.otpService.ResendCode(ctx, sessionID, ipAddress, userAgent)
	if err != nil {
		return "", 0, s.mapResendError(err)
	}
	return sessionID, int64(s.otpConfig.CodeTTL.Seconds()), nil
}

// mapResendError maps resend failures to client-facing domain errors.
func (s *AuthService) mapResendError(err error) error {
	switch {
	case errors.Is(err, otp.ErrOTPLocked):
		return domain.NewUnauthorizedError(constants.MsgOTPAccountLockedVN)
	case errors.Is(err, otp.ErrSessionNotFound), errors.Is(err, otp.ErrSessionBindingMismatch):
		return domain.NewUnauthorizedError(constants.MsgOTPSessionIDMissingVN)
	case errors.Is(err, otp.ErrResendCooldown):
		return domain.NewValidationError(constants.MsgOTPResendTooSoonVN)
	default:
		return domain.NewInternalError(constants.MsgOTPStartFailedVN, err)
	}
}

// setupAuditContext creates a context with audit information
func (s *AuthService) setupAuditContext(ctx context.Context, userID uint, ipAddress, userAgent string) context.Context {
	ctx = auditctx.WithUserID(ctx, userID)
	// Note: We could also add IP and user agent to context if needed for future use
	return ctx
}

func (s *AuthService) writeFailedLoginAudit(ctx context.Context, userID uint, attemptedIdentifier, ipAddress, userAgent, reason string) {
	var location map[string]string
	if ipAddress != "" {
		geoCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if loc, err := ipgeo.Lookup(geoCtx, ipAddress); err == nil && loc != nil {
			location = map[string]string{
				"country": loc.Country,
				"city":    loc.City,
				"region":  loc.Region,
			}
		}
	}

	username := attemptedIdentifier
	var userFullName string
	if userID != 0 {
		if user, err := s.userService.UserRepo.GetByID(ctx, userID); err == nil && user != nil {
			username = user.Username
			userFullName = user.Fullname
		}
	}

	event := domain.NewUserLoginEvent(ctx, userID, username, userFullName, ipAddress, userAgent, reason, attemptedIdentifier)
	event.Location = location

	if s.eventBus != nil {
		if err := s.eventBus.Publish(ctx, event); err != nil {
			s.logger.Error("failed to publish failed login event", "error", err, "identifier", attemptedIdentifier)
		} else {
			s.logger.Info("failed login audit event published", "identifier", attemptedIdentifier, "reason", reason)
		}
	}
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	// Try to resolve user by multiple identifiers: username, CCCD, or mobile
	userRepo := s.userService.UserRepo
	var user *domain.User
	var err error

	// Try username first
	user, err = userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		// If username not found, try User CCCD (for admin/partner)
		user, err = userRepo.GetByCCCD(ctx, req.Username)
		if err != nil {
			// Try User mobile (for admin/partner)
			user, err = userRepo.GetByMobile(ctx, req.Username)
			if err != nil {
				// Try Employee CCCD (for employee)
				employee, empErr := s.employeeRepo.GetByCCCD(ctx, req.Username)
				if empErr == nil && employee.UserID != nil {
					user, err = userRepo.GetByID(ctx, *employee.UserID)
				}

				// Try Employee mobile (for employee)
				if err != nil {
					employee, empErr = s.employeeRepo.GetByMobile(ctx, req.Username)
					if empErr == nil && employee.UserID != nil {
						user, err = userRepo.GetByID(ctx, *employee.UserID)
					}
				}

				// If none found, return unauthorized
				if err != nil {
					s.writeFailedLoginAudit(ctx, 0, req.Username, ipAddress, userAgent, "thất bại: không tìm thấy tài khoản")
					return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
				}
			}
		}
	}

	// CAPTCHA gate: after `threshold` consecutive failed logins on this account,
	// require a valid captcha code before the password is even checked. Prevents
	// automated brute-force tools from hammering the password check. Reuses the
	// otp_failed_attempts counter as the consecutive-failure tracker (reset on
	// successful login below).
	if s.captchaService != nil && s.captchaService.RequiredForFailures(user.OTPFailedAttempts) {
		if !s.captchaService.Verify(ctx, req.CaptchaID, req.CaptchaCode) {
			s.writeFailedLoginAudit(ctx, user.ID, req.Username, ipAddress, userAgent, "thất bại: captcha không hợp lệ")
			// Keep the failure counter climbing so the account continues to
			// escalate toward OTP-lockout; a sustained captcha-failed flood must
			// not freeze the counter at the threshold indefinitely.
			s.userService.UserRepo.UpdateOTPLockout(ctx, user.ID, user.OTPFailedAttempts+1, nil)
			return nil, domain.NewValidationError(constants.MsgCaptchaFailedVN)
		}
	}

	if !s.userService.VerifyPasswordHash(req.Password, user.Password) {
		s.writeFailedLoginAudit(ctx, user.ID, req.Username, ipAddress, userAgent, "thất bại: sai mật khẩu")
		// Increment consecutive failure count (used by CAPTCHA threshold + brute-force tracking).
		s.userService.UserRepo.UpdateOTPLockout(ctx, user.ID, user.OTPFailedAttempts+1, nil)
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
	}

	// Reset the consecutive-failure counter on successful password verification.
	// NOTE: only the counter is cleared — an existing OTP lockout
	// (user.OTPLockedUntil in the future, set by the OTP brute-force backoff) is
	// preserved so a correct password cannot unlock an account the OTP subsystem
	// locked. The shared counter is reused for CAPTCHA-threshold tracking, which
	// is safe here: an attacker without the password can never reach this reset
	// (it requires a valid password) nor the OTP increment path (also behind the
	// password check), so they cannot deplete the counter to dodge the CAPTCHA.
	if user.OTPFailedAttempts > 0 {
		s.userService.UserRepo.UpdateOTPLockout(ctx, user.ID, 0, user.OTPLockedUntil)
	}

	// Email-OTP 2FA gate (RT-C2): when the feature is enabled and the caller is
	// an admin/partner, do NOT mint a JWT here — start the OTP second step.
	// Employees (and any non-gated role) fall straight through to token issuance.
	otpVerified := true
	if s.requiresOTP(user) {
		sessionID, otpErr := s.otpService.StartLogin(ctx, user, ipAddress, userAgent)
		if otpErr != nil {
			// Missing email (RT-M8) / locked account -> surface a clear VN message.
			return nil, s.mapOTPError(otpErr)
		}
		return &dto.LoginResponse{
			OTPRequired:  true,
			OTPSessionID: sessionID,
			ExpiresIn:    int64(s.otpConfig.CodeTTL.Seconds()),
		}, nil
	}

	// Generate access token (non-gated path: otpVerified stays true).
	accessToken, err := s.generateAccessToken(user, otpVerified)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgFailedToGenerateTokenVN, err)
	}

	// Update last login timestamp asynchronously to reduce latency
	go func(userID uint) {
		bgCtx := context.Background()
		now := clock.Now()
		if err := s.userService.UserRepo.UpdateLastLogin(bgCtx, userID, now); err != nil {
			s.logger.Error("Failed to update last login", "error", err, "user_id", userID)
		}
	}(user.ID)

	// Log successful login asynchronously to reduce latency
	go func(userID uint, username, userFullName string) {
		ctx := context.Background()

		// Geo-lookup for successful login
		var location map[string]string
		if ipAddress != "" {
			geoCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if loc, err := ipgeo.Lookup(geoCtx, ipAddress); err == nil && loc != nil {
				location = map[string]string{
					"country": loc.Country,
					"city":    loc.City,
					"region":  loc.Region,
				}
			}
		}

		// Pass the original identifier the user typed (req.Username) as attemptedIdentifier
		// so we can show whether they logged in via username, CCCD, or mobile
		event := domain.NewUserLoginEvent(ctx, userID, username, userFullName, ipAddress, userAgent, "", req.Username)
		event.Location = location
		if s.eventBus != nil {
			if err := s.eventBus.Publish(ctx, event); err != nil {
				s.logger.Error("Failed to publish UserLoginEvent for successful login", "error", err, "user_id", userID)
			}
		}
	}(user.ID, user.Username, user.Fullname)

	userResponse := dto.ToUserResponse(user)
	return &dto.LoginResponse{
		User:        &userResponse,
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// Verify the signing method to prevent algorithm confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Explicitly verify it's HS256
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("invalid algorithm: %s, expected HS256", token.Method.Alg())
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Check if token is blacklisted
		isBlacklisted, err := s.blacklistedTokenRepo.IsBlacklisted(ctx, claims.ID)
		if err != nil {
			return nil, domain.NewInternalError(constants.MsgFailedToCheckTokenBlacklistVN, err)
		}

		if isBlacklisted {
			return nil, domain.NewUnauthorizedError(constants.MsgTokenBlacklistedVN)
		}

		// Enforce per-user token invalidation: if an admin revoked all of this
		// user's sessions (RevokeUserTokens), any JWT issued before that instant
		// is rejected even though its signature/expiry are still valid. This closes
		// the gap where a stolen token would otherwise live for the full 14-day TTL.
		if user, err := s.userService.UserRepo.GetByID(ctx, claims.UserID); err == nil && user.TokensInvalidBefore != nil {
			if claims.IssuedAt == nil || claims.IssuedAt.Before(*user.TokensInvalidBefore) {
				return nil, domain.NewUnauthorizedError(constants.MsgTokenBlacklistedVN)
			}
		}

		return claims, nil
	}

	return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
}

// generateAccessToken mints the 14-day access JWT. otpVerified is propagated
// into the claims so the Authorize middleware can gate privileged routes (RT-C1).
// Callers MUST pass otpVerified=true only when the login path completed OTP (or
// is not gated — employee, or admin/partner when OTP_ENABLE=false).
func (s *AuthService) generateAccessToken(user *domain.User, otpVerified bool) (string, error) {
	jti := uuid.New().String()
	claims := Claims{
		UserID:       user.ID,
		Username:     user.Username,
		Fullname:     user.Fullname,
		Role:         string(user.Role),
		OTPVerified:  otpVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(clock.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(clock.Now()),
			NotBefore: jwt.NewNumericDate(clock.Now()),
			Subject:   "access",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// Logout blacklists the given token
func (s *AuthService) Logout(ctx context.Context, tokenString, ipAddress, userAgent string) error {
	// Validate token first
	claims, err := s.ValidateToken(ctx, tokenString)
	if err != nil {
		return err
	}

	// Blacklist the token
	err = s.blacklistedTokenRepo.BlacklistToken(
		ctx,
		claims.ID,
		claims.UserID,
		claims.ExpiresAt.Time,
		domain.BlacklistReasonLogout,
	)
	if err != nil {
		return domain.NewInternalError(constants.MsgFailedToBlacklistTokenVN, err)
	}

	// Publish UserLogoutEvent for audit logging
	// Get user's full name for audit message
	userFullName := claims.Username
	if user, err := s.userService.UserRepo.GetByID(ctx, claims.UserID); err == nil && user != nil {
		userFullName = user.Fullname
		if userFullName == "" {
			userFullName = user.Username
		}
	}

	event := domain.NewUserLogoutEvent(ctx, claims.UserID, claims.Username, userFullName)
	if s.eventBus != nil {
		if err := s.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail the logout
			// The token is already blacklisted, which is the main goal
			s.logger.Error("Failed to publish UserLogoutEvent", "error", err, "user_id", claims.UserID)
		}
	}

	return nil
}

// GetUserFromToken extracts user information from a valid token
func (s *AuthService) GetUserFromToken(ctx context.Context, tokenString string) (*domain.User, error) {
	claims, err := s.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	return s.userService.GetUser(ctx, claims.UserID)
}

// RevokeUserTokens invalidates all currently-issued JWTs for a user (admin
// function). It sets tokens_invalid_before = now on the user record; every
// existing token whose IssuedAt is older than that timestamp is then rejected by
// ValidateToken on its next use. Tokens issued after this instant remain valid,
// so the user can log back in immediately. The audit event is still published.
func (s *AuthService) RevokeUserTokens(ctx context.Context, userID uint, revokedByUserID uint, reason domain.BlacklistReason, ipAddress, userAgent string) error {
	now := clock.Now()

	if err := s.userService.UserRepo.UpdateTokensInvalidBefore(ctx, userID, now); err != nil {
		s.logger.Error("Failed to set tokens_invalid_before for user", "error", err, "user_id", userID)
		return err
	}
	s.logger.Info("Revoked all user tokens via tokens_invalid_before", "user_id", userID, "revoked_by", revokedByUserID, "invalid_before", now)

	// RT-M4: also kill any in-flight OTP login session for this user, so a
	// revocation mid-second-step cannot be completed by an attacker who later
	// obtains the code. Best-effort: a Redis failure here must not undo the
	// successful token revocation above.
	if s.otpService != nil {
		if err := s.otpService.DeleteSessionsForUser(ctx, userID); err != nil {
			s.logger.Error("Failed to delete pending OTP sessions during revoke", "error", err, "user_id", userID)
		}
	}

	// Get user for audit logging
	user, err := s.userService.GetUser(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user for revoke tokens audit", "error", err, "user_id", userID)
	}

	// Publish UserUpdatedEvent for audit logging
	if user != nil {
		actorFullName := audit.GetActorFullName(ctx, s.userService.UserRepo, revokedByUserID)
		event := domain.NewUserUpdatedEvent(ctx, user, revokedByUserID, actorFullName, nil)
		if s.eventBus != nil {
			// We could create a more specific event for token revocation, but UserUpdatedEvent covers the audit requirement
			if err := s.eventBus.Publish(ctx, event); err != nil {
				s.logger.Error("Failed to publish UserUpdatedEvent for revoke tokens", "error", err, "user_id", userID)
				return err
			}
		}
	}

	return nil
}

// ChangePassword delegates to UserService
func (s *AuthService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword, ipAddress, userAgent string) error {
	return s.userService.ChangePassword(ctx, userID, currentPassword, newPassword, ipAddress, userAgent)
}

// ChangePasswordAndBlacklistToken changes password and blacklists the current token for security
func (s *AuthService) ChangePasswordAndBlacklistToken(ctx context.Context, userID uint, currentPassword, newPassword, tokenString, ipAddress, userAgent string) error {
	// First, change the password
	err := s.userService.ChangePassword(ctx, userID, currentPassword, newPassword, ipAddress, userAgent)
	if err != nil {
		return err
	}

	// Validate the token to get claims (JTI and expiration)
	claims, err := s.ValidateToken(ctx, tokenString)
	if err != nil {
		// Password was already changed, log the error but don't fail
		s.logger.Error("Failed to validate token for blacklisting after password change", "error", err, "user_id", userID)
		return nil
	}

	// Blacklist the current token for security reasons
	err = s.blacklistedTokenRepo.BlacklistToken(
		ctx,
		claims.ID, // JTI
		userID,
		claims.ExpiresAt.Time,
		domain.BlacklistReasonSecurity,
	)
	if err != nil {
		// Password was already changed, log the error but don't fail
		s.logger.Error("Failed to blacklist token after password change", "error", err, "user_id", userID, "jti", claims.ID)
		return nil
	}

	s.logger.Info("Token blacklisted after password change", "user_id", userID, "jti", claims.ID)
	return nil
}

// ValidatePassword delegates to UserService
func (s *AuthService) ValidatePassword(password string) error {
	return s.userService.ValidatePassword(password)
}

// GetPasswordStrength delegates to UserService
func (s *AuthService) GetPasswordStrength(password string) (int, []string) {
	return s.userService.GetPasswordStrength(password)
}

// UpdateProfile updates the current user's profile (email, fullname, and optionally password)
func (s *AuthService) UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest, ipAddress, userAgent string) (*dto.UserResponse, error) {
	// Validate email format if provided and not empty
	if req.Email != nil && *req.Email != "" {
		if err := s.userService.ValidateEmailFormat(*req.Email); err != nil {
			return nil, domain.NewValidationError(constants.MsgInvalidEmailFormatVN)
		}
	}

	// Set up audit context with user ID, IP address, and user agent
	auditCtx := s.setupAuditContext(ctx, userID, ipAddress, userAgent)

	// Get current user
	user, err := s.userService.UserRepo.GetByID(auditCtx, userID)
	if err != nil {
		return nil, err
	}

	// Keep a copy for audit logging
	currentUserDomain := *user

	// Apply email update
	if req.Email != nil {
		if *req.Email == "" {
			user.Email = nil
		} else {
			// Check if email is already taken by another user
			if existing, err := s.userService.UserRepo.GetByEmail(auditCtx, *req.Email); err == nil && existing.ID != userID {
				return nil, domain.NewConflictError(constants.MsgEmailAlreadyTakenVN)
			}
			user.Email = req.Email
		}
	}
	if req.Fullname != nil && *req.Fullname != "" {
		user.Fullname = *req.Fullname
	}

	// Apply CCCD update
	if req.CCCD != nil {
		if *req.CCCD == "" {
			user.CCCD = nil
		} else {
			// Check uniqueness: no other active user can have this CCCD
			if existing, err := s.userService.UserRepo.GetByCCCD(auditCtx, *req.CCCD); err == nil && existing.ID != userID {
				return nil, domain.NewConflictError(constants.MsgCCCDUsedByAnotherAccountVN)
			}
			cccdVal := *req.CCCD
			user.CCCD = &cccdVal
		}
	}

	// Apply Mobile update
	if req.Mobile != nil {
		if *req.Mobile == "" {
			user.Mobile = nil
		} else {
			// Check uniqueness: no other active user can have this mobile
			if existing, err := s.userService.UserRepo.GetByMobile(auditCtx, *req.Mobile); err == nil && existing.ID != userID {
				return nil, domain.NewConflictError(constants.MsgPhoneUsedByAnotherAccountVN)
			}
			mobileVal := *req.Mobile
			user.Mobile = &mobileVal
		}
	}

	// Save all changes in one operation
	if err := s.userService.UserRepo.Update(auditCtx, user); err != nil {
		return nil, err
	}

	// Build audit message
	if req.Email != nil || req.Fullname != nil || req.CCCD != nil || req.Mobile != nil {
		event := domain.NewUserUpdatedEvent(ctx, user, auditctx.GetUserIDOrZero(ctx), user.Fullname, &currentUserDomain)
		if s.eventBus != nil {
			if err := s.eventBus.Publish(ctx, event); err != nil {
				s.logger.Error("Failed to publish UserUpdatedEvent for profile update", "error", err, "user_id", userID)
			}
		}
	}

	// RT-H3: password changes via UpdateProfile are REJECTED. The dedicated
	// /auth/change-password endpoint requires the current password (re-auth);
	// this path never did, which made it a takeover-persistence primitive — a
	// momentary session theft (XSS, stolen token, unattended browser) let an
	// attacker set a password they then knew, surviving the OTP gate. Force all
	// password changes through the re-auth endpoint.
	if req.Password != nil && *req.Password != "" {
		return nil, domain.NewValidationError(constants.MsgPasswordChangeNotAllowedOnProfileVN)
	}

	updatedResponse := dto.ToUserResponse(user)
	return &updatedResponse, nil
}

func (s *AuthService) LoginWithGoogle(ctx context.Context, req dto.GoogleLoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	if s.googleClientID == "" {
		return nil, domain.NewInternalError("GOOGLE_CLIENT_ID is not configured", nil)
	}

	payload, err := idtoken.Validate(ctx, req.IDToken, s.googleClientID)
	if err != nil {
		s.logger.Warn("Google OAuth ID token validation failed", "error", err)
		s.writeFailedLoginAudit(ctx, 0, "Google OAuth", ipAddress, userAgent, "thất bại: ID Token không hợp lệ")
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
	}

	// Defense-in-depth: pin the issuer explicitly. idtoken.Validate already
	// checks the audience (== s.googleClientID) and signature; this rejects any
	// token whose issuer is not Google's accounts service.
	if payload.Issuer != "https://accounts.google.com" && payload.Issuer != "accounts.google.com" {
		s.logger.Warn("Google OAuth rejected — unexpected issuer", "issuer", payload.Issuer)
		s.writeFailedLoginAudit(ctx, 0, "Google OAuth", ipAddress, userAgent, "thất bại: issuer không hợp lệ")
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
	}

	// Replay defense: the id_token carries the nonce the client generated and
	// sent to Google. Consume it once here — a second presentation of the same
	// token (captured from a redirect URL, browser history, or an extension) is
	// rejected. A missing nonce is treated as a replay (the OIDC flow always
	// includes one).
	if s.nonceStore != nil {
		nonce, _ := payload.Claims["nonce"].(string)
		if err := s.nonceStore.Consume(ctx, nonce); err != nil {
			s.logger.Warn("Google OAuth rejected — nonce replay or missing", "nonce_present", nonce != "")
			s.writeFailedLoginAudit(ctx, 0, "Google OAuth", ipAddress, userAgent, "thất bại: nonce không hợp lệ")
			return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
		}
	}

	// email_verified gate: Google only guarantees the email belongs to the
	// account holder when email_verified=true. Check BEFORE email extraction so
	// the rejection is indistinguishable from any other invalid-credential
	// failure (avoids a user-enumeration oracle via the distinct not-linked
	// message the email-lookup below returns). A missing claim is treated as
	// "not verified" and audited distinctly from an explicit false.
	verifiedVal, hasVerifiedClaim := payload.Claims["email_verified"]
	isEmailVerified := false
	if hasVerifiedClaim {
		if b, ok := verifiedVal.(bool); ok {
			isEmailVerified = b
		}
	}
	if !isEmailVerified {
		reason := "email_verified claim absent"
		if hasVerifiedClaim {
			reason = "email_verified=false"
		}
		s.logger.Warn("Google OAuth rejected — email not verified", "has_claim", hasVerifiedClaim)
		s.writeFailedLoginAudit(ctx, 0, "Google OAuth", ipAddress, userAgent, "thất bại: "+reason)
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
	}

	emailVal, ok := payload.Claims["email"]
	if !ok {
		s.writeFailedLoginAudit(ctx, 0, "Google OAuth", ipAddress, userAgent, "thất bại: email không tồn tại trong token payload")
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
	}

	email, ok := emailVal.(string)
	if !ok || email == "" {
		s.writeFailedLoginAudit(ctx, 0, "Google OAuth", ipAddress, userAgent, "thất bại: email không hợp lệ trong token payload")
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
	}

	// Query user by email
	user, err := s.userService.UserRepo.GetByEmail(ctx, email)
	if err != nil {
		s.logger.Warn("Google OAuth user not found by email", "email", email, "error", err)
		s.writeFailedLoginAudit(ctx, 0, email, ipAddress, userAgent, "thất bại: "+constants.MsgGoogleAccountNotLinkedVN)
		return nil, domain.NewUnauthorizedError(constants.MsgGoogleAccountNotLinkedVN)
	}

	if user == nil {
		s.writeFailedLoginAudit(ctx, 0, email, ipAddress, userAgent, "thất bại: "+constants.MsgGoogleAccountNotLinkedVN)
		return nil, domain.NewUnauthorizedError(constants.MsgGoogleAccountNotLinkedVN)
	}

	// Google OAuth is treated as a complete authentication (the second factor
	// is Google's own account protection). This mirrors the common pattern on
	// most sites: a verified Google id_token grants access without an
	// additional emailed OTP. The compensating controls that make this safe:
	//   - Google's signature on the id_token is verified (idtoken.Validate)
	//   - audience is bound to this app's GOOGLE_CLIENT_ID
	//   - issuer is pinned to accounts.google.com (above)
	//   - email_verified is required (above)
	//   - the id_token's nonce is single-use (nonceStore.Consume above) so a
	//     captured token cannot be replayed
	//
	// RESIDUAL RISK (accepted by product decision, 2026-07-04): the id_token
	// does not carry the user's 2SV status, the factor used, or how the Google
	// session was established. A stolen Google session cookie (infostealer
	// malware) bypasses Google 2SV entirely and yields a valid id_token — and
	// therefore a valid 14-day admin/partner JWT here. No step-up auth on money
	// routes and no shortened JWT TTL were added (deliberate UX-over-security
	// call). Mitigation is operational, not technical: admin/partner Google
	// accounts are expected to keep 2SV on a phishing-resistant factor (security
	// key / passkey), and endpoint hygiene (no malware) is the user's duty.
	// See plans/reports/2026-07-04-google-oauth-no-otp-research.md.
	accessToken, err := s.generateAccessToken(user, true)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgFailedToGenerateTokenVN, err)
	}

	// Update last login timestamp asynchronously
	go func(userID uint) {
		bgCtx := context.Background()
		now := clock.Now()
		if err := s.userService.UserRepo.UpdateLastLogin(bgCtx, userID, now); err != nil {
			s.logger.Error("Failed to update last login", "error", err, "user_id", userID)
		}
	}(user.ID)

	// Log successful login asynchronously
	go func(userID uint, username, userFullName string) {
		ctx := context.Background()

		var location map[string]string
		if ipAddress != "" {
			geoCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if loc, err := ipgeo.Lookup(geoCtx, ipAddress); err == nil && loc != nil {
				location = map[string]string{
					"country": loc.Country,
					"city":    loc.City,
					"region":  loc.Region,
				}
			}
		}

		event := domain.NewUserLoginEvent(ctx, userID, username, userFullName, ipAddress, userAgent, "", "Google OAuth ("+email+")")
		event.Location = location
		if s.eventBus != nil {
			if err := s.eventBus.Publish(ctx, event); err != nil {
				s.logger.Error("Failed to publish UserLoginEvent for Google OAuth successful login", "error", err, "user_id", userID)
			}
		}
	}(user.ID, user.Username, user.Fullname)

	userResponse := dto.ToUserResponse(user)
	return &dto.LoginResponse{
		User:        &userResponse,
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.accessTTL.Seconds()),
	}, nil
}
