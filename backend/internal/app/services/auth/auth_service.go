package auth

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/audit"
	"api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/domain"
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
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthService(userService *user.UserService, employeeRepo domain.EmployeeRepository, blacklistedTokenRepo domain.BlacklistedTokenRepository, eventBus domain.EventBus, jwtSecret string, accessTTL time.Duration, logger *slog.Logger) *AuthService {
	return &AuthService{
		userService:          userService,
		employeeRepo:         employeeRepo,
		blacklistedTokenRepo: blacklistedTokenRepo,
		eventBus:             eventBus,
		jwtSecret:            jwtSecret,
		accessTTL:            accessTTL,
		logger:               logger,
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

	if !s.userService.VerifyPasswordHash(req.Password, user.Password) {
		s.writeFailedLoginAudit(ctx, user.ID, req.Username, ipAddress, userAgent, "thất bại: sai mật khẩu")
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
	}

	// Generate access token
	accessToken, err := s.generateAccessToken(user)
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

	return &dto.LoginResponse{
		User:        dto.ToUserResponse(user),
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

		return claims, nil
	}

	return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
}

func (s *AuthService) generateAccessToken(user *domain.User) (string, error) {
	jti := uuid.New().String()
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Fullname: user.Fullname,
		Role:     string(user.Role),
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

// RevokeUserTokens blacklists all tokens for a specific user (admin function)
func (s *AuthService) RevokeUserTokens(ctx context.Context, userID uint, revokedByUserID uint, reason domain.BlacklistReason, ipAddress, userAgent string) error {
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

	// If password is provided, handle password change separately
	if req.Password != nil && *req.Password != "" {
		if err := s.userService.ValidatePassword(*req.Password); err != nil {
			return nil, err
		}
		if err := s.userService.ResetUserPassword(auditCtx, userID, *req.Password); err != nil {
			return nil, err
		}
		actorFullName := audit.GetActorFullName(ctx, s.userService.UserRepo, userID)
		passwordEvent := domain.NewPasswordChangedEvent(ctx, userID, currentUserDomain.Username, "profile_update", auditctx.GetUserIDOrZero(ctx), actorFullName)
		if s.eventBus != nil {
			if err := s.eventBus.Publish(ctx, passwordEvent); err != nil {
				s.logger.Error("Failed to publish PasswordChangedEvent", "error", err, "user_id", userID)
			}
		}
	}

	updatedResponse := dto.ToUserResponse(user)
	return &updatedResponse, nil
}

func (s *AuthService) LoginWithGoogle(ctx context.Context, req dto.GoogleLoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	if googleClientID == "" {
		return nil, domain.NewInternalError("GOOGLE_CLIENT_ID is not configured in the environment", nil)
	}

	payload, err := idtoken.Validate(ctx, req.IDToken, googleClientID)
	if err != nil {
		s.logger.Warn("Google OAuth ID token validation failed", "error", err)
		s.writeFailedLoginAudit(ctx, 0, "Google OAuth", ipAddress, userAgent, "thất bại: ID Token không hợp lệ")
		return nil, domain.NewUnauthorizedError(constants.MsgInvalidCredentialsVN)
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

	// Generate access token
	accessToken, err := s.generateAccessToken(user)
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

	return &dto.LoginResponse{
		User:        dto.ToUserResponse(user),
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.accessTTL.Seconds()),
	}, nil
}
