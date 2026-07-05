package handlers

import (
	"strings"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/auth"
	"api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *auth.AuthService
	userService *user.UserService
}

func NewAuthHandler(authService *auth.AuthService, userService *user.UserService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
	}
}

// @Summary User login
// @Description Authenticate user with username and password
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body dto.LoginRequest true "Login credentials"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	loginResponse, err := h.authService.Login(c.Request.Context(), req, ipAddress, userAgent)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, loginResponse, "Login successful")
}

// @Summary Generate login CAPTCHA
// @Description Generate a digit-image CAPTCHA for brute-force protection (required after N failed logins)
// @Tags auth
// @Produce json
// @Success 200 {object} response.SuccessResponse
// @Router /auth/captcha [get]
func (h *AuthHandler) GetCaptcha(c *gin.Context) {
	id, image, err := h.authService.GenerateCaptcha(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Không thể tạo mã captcha")
		return
	}
	response.Success(c, gin.H{
		"captcha_id": id,
		"image":      image,
	}, "OK")
}

// @Summary Verify OTP login code
// @Description Complete the two-step login by submitting the emailed 6-digit code
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.VerifyOTPRequest true "OTP session id + code"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/login/verify [post]
func (h *AuthHandler) VerifyLoginOTP(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if !helpers.BindJSON(c, &req) {
		return
	}
	if req.OTPSessionID == "" {
		response.BadRequest(c, constants.MsgOTPSessionIDRequiredVN)
		return
	}
	if req.Code == "" {
		response.BadRequest(c, constants.MsgOTPCodeRequiredVN)
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	loginResponse, err := h.authService.VerifyLoginOTP(c.Request.Context(), req.OTPSessionID, req.Code, ipAddress, userAgent)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, loginResponse, "Login successful")
}

// @Summary Resend OTP code
// @Description Re-issue the emailed OTP code for a pending login session
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.ResendOTPRequest true "OTP session id"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /auth/login/resend [post]
func (h *AuthHandler) ResendOTPCode(c *gin.Context) {
	var req dto.ResendOTPRequest
	if !helpers.BindJSON(c, &req) {
		return
	}
	if req.OTPSessionID == "" {
		response.BadRequest(c, constants.MsgOTPSessionIDRequiredVN)
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	sessionID, expiresIn, err := h.authService.ResendOTPCode(c.Request.Context(), req.OTPSessionID, ipAddress, userAgent)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, gin.H{
		"otp_required":  true,
		"otp_session_id": sessionID,
		"expires_in":    expiresIn,
	}, "Mã mới đã được gửi")
}
// @Description Get the profile of the currently authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} dto.UserResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /me [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, constants.MsgUserNotAuthenticatedVN)
		return
	}

	// Get complete user information from database
	user, err := h.userService.GetUser(c.Request.Context(), userID.(uint))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Convert to UserResponse DTO
	userResponse := dto.ToUserResponse(user)
	response.Success(c, userResponse, "User profile retrieved successfully")
}

// @Summary User logout
// @Description Logout user and blacklist the current token
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c, constants.MsgAuthorizationHeaderRequiredVN)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		response.Unauthorized(c, constants.MsgInvalidAuthorizationFormatVN)
		return
	}

	token := parts[1]
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Logout and blacklist token
	err := h.authService.Logout(c.Request.Context(), token, ipAddress, userAgent)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, "Logout successful")
}

// @Summary Change password
// @Description Change user password (requires current password)
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.ChangePasswordRequest true "Password change data"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, constants.MsgUserNotAuthenticatedVN)
		return
	}

	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c, constants.MsgAuthorizationHeaderRequiredVN)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		response.Unauthorized(c, constants.MsgInvalidAuthorizationFormatVN)
		return
	}

	token := parts[1]

	var req dto.ChangePasswordRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	err := h.authService.ChangePasswordAndBlacklistToken(c.Request.Context(), userID.(uint), req.CurrentPassword, req.NewPassword, token, ipAddress, userAgent)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, "Mật khẩu đã được thay đổi thành công")
}

// @Summary Update current user profile
// @Description Update the current user's profile including email, fullname, and optionally password
// @Tags auth
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.UpdateProfileRequest true "Profile update data"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /me [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, constants.MsgUserNotAuthenticatedVN)
		return
	}

	var req dto.UpdateProfileRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	updatedUser, err := h.authService.UpdateProfile(c.Request.Context(), userID.(uint), req, ipAddress, userAgent)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, updatedUser, "Profile updated successfully")
}

// @Summary Google OAuth login
// @Description Authenticate user with Google ID token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body dto.GoogleLoginRequest true "Google credentials"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /auth/google [post]
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var req dto.GoogleLoginRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	loginResponse, err := h.authService.LoginWithGoogle(c.Request.Context(), req, ipAddress, userAgent)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, loginResponse, "Login successful")
}
