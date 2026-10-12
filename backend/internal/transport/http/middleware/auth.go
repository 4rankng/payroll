package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	auditservice "api-server/internal/app/services/audit"
	authservice "api-server/internal/app/services/auth"
	"api-server/internal/app/services/mcpapikey"
	"api-server/internal/app/services/opsapikey"
	"api-server/internal/config"
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	authService *authservice.AuthService
	mcpKeys     *mcpapikey.Service
	opsKeys     *opsapikey.Service
	opsCfg      config.OpsMCPConfig
	opsLimiter  *opsRateLimiter
}

// NewAuthMiddleware constructs the middleware. mcpKeys and opsKeys may be
// nil in tests — each machine-key branch then fails closed with 401.
func NewAuthMiddleware(authService *authservice.AuthService, mcpKeys *mcpapikey.Service, opsKeys *opsapikey.Service, opsCfg config.OpsMCPConfig) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		mcpKeys:     mcpKeys,
		opsKeys:     opsKeys,
		opsCfg:      opsCfg,
		opsLimiter:  newOpsRateLimiter(opsCfg.RatePerMin),
	}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Machine credential first: an MCP API key (minted in the admin
		// settings) authenticates as its creating admin. The credential types
		// are mutually exclusive per request — X-API-Key present means the
		// caller is a machine agent, not a browser session. The pk_ops_*
		// namespace routes to the ops channel (agent_ops role); every other
		// key is a chat-integration credential.
		if raw := c.GetHeader("X-API-Key"); raw != "" {
			if strings.HasPrefix(raw, opsapikey.KeyPrefix) {
				m.authenticateOpsKey(c)
			} else {
				m.authenticateMCPKey(c)
			}
			return
		}

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

// authenticateMCPKey resolves a machine request carrying an MCP API key. On
// success it publishes the creating admin's identity — the key executes with
// the creator's rights, resolved at request time — and continues the chain.
// On failure it aborts with 401 (bad key) or 403 (valid key, forbidden use).
func (m *AuthMiddleware) authenticateMCPKey(c *gin.Context) {
	if m.mcpKeys == nil {
		response.Unauthorized(c, constants.MsgMCPKeyInvalidVN)
		c.Abort()
		return
	}

	key, user, err := m.mcpKeys.Authenticate(c.Request.Context(), c.GetHeader("X-API-Key"))
	if err != nil {
		// Every failure mode (unknown, revoked, orphaned creator) is
		// indistinguishable on purpose.
		response.Unauthorized(c, constants.MsgMCPKeyInvalidVN)
		c.Abort()
		return
	}

	// Path deny-list: authentication and credential management must never be
	// reachable with a machine key, however broad its scope. This includes
	// the ops key management surface — an integration key has no business
	// touching the ops channel (wrong-scope isolation).
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/api/v1/auth") ||
		strings.HasPrefix(path, "/api/v1/admin/api-keys") ||
		strings.HasPrefix(path, "/api/v1/admin/mcp-api-keys") ||
		strings.HasPrefix(path, "/api/v1/admin/ops-api-keys") {
		response.Forbidden(c, constants.MsgMCPKeyPathDeniedVN)
		c.Abort()
		return
	}

	// Breadth gate: a READ ONLY key may only use read verbs.
	if key.IsReadOnly() && c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		response.Forbidden(c, constants.MsgMCPKeyReadOnlyVN)
		c.Abort()
		return
	}

	// Publish identity as the creating admin: the key is the machine's
	// credential, the creator is the authorization subject. otp_verified is
	// set true because the key was minted in an OTP-verified admin session —
	// the key itself is the machine's second factor.
	c.Set(constants.CtxUserID, user.ID)
	c.Set("username", user.Username)
	c.Set(constants.CtxUserRole, string(user.Role))
	c.Set("otp_verified", true)
	c.Set("tenant_id", user.ID)
	c.Set(constants.CtxMCPKeyID, key.ID)

	ctx := auditservice.WithAuditContext(c.Request.Context(), user.ID, user.Fullname)
	c.Request = c.Request.WithContext(ctx)

	// Fire-and-forget last-used stamp (never blocks the response). The key
	// itself must never be logged — only its id.
	keyID := key.ID
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Default().Warn("mcp key auth: touch last_used panicked", "panic", r, "mcp_key_id", keyID)
			}
		}()
		touchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.mcpKeys.TouchLastUsed(touchCtx, keyID); err != nil {
			slog.Default().Warn("mcp key auth: touch last_used failed", "error", err, "mcp_key_id", keyID)
		}
	}()

	c.Next()
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
