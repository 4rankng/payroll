package middleware

import (
	"strings"

	auditservice "api-server/internal/app/services/audit"
	authservice "api-server/internal/app/services/auth"
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	authService *authservice.AuthService
}

func NewAuthMiddleware(authService *authservice.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "Xin đăng nhập để tiếp tục")
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "Invalid authorization format")
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			response.Unauthorized(c, "Invalid token")
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(constants.CtxUserID, claims.UserID)
		c.Set("username", claims.Username)
		c.Set(constants.CtxUserRole, claims.Role)
		c.Set("token_jti", claims.ID)
		// otp_verified: propagated from the JWT claim so Authorize() can gate
		// privileged routes on completion of the email-OTP second factor (RT-C1).
		c.Set("otp_verified", claims.OTPVerified)
		// Set tenant_id for tenant isolation middleware (using user_id as tenant identifier)
		c.Set("tenant_id", claims.UserID)

		// Add user information to audit context using shared helper
		ctx := auditservice.WithAuditContext(c.Request.Context(), claims.UserID, claims.Fullname)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			token := parts[1]
			if claims, err := m.authService.ValidateToken(c.Request.Context(), token); err == nil {
				c.Set(constants.CtxUserID, claims.UserID)
				c.Set("username", claims.Username)
				c.Set(constants.CtxUserRole, claims.Role)
				c.Set("token_jti", claims.ID)
				c.Set("otp_verified", claims.OTPVerified)
				// Set tenant_id for tenant isolation middleware (using user_id as tenant identifier)
				c.Set("tenant_id", claims.UserID)

				// Add user information to audit context using shared helper
				ctx := auditservice.WithAuditContext(c.Request.Context(), claims.UserID, claims.Fullname)
				c.Request = c.Request.WithContext(ctx)
			}
		}

		c.Next()
	}
}
