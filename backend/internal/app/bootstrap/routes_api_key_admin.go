package bootstrap

import (
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// setupAPIKeyAdminRoutes mounts the admin-only API key management endpoints.
// Casbin grants `admin, /api/*, *, allow`, and no other role holds a rule for
// /api/v1/admin/*, so the generic Authorize() middleware is sufficient.
func setupAPIKeyAdminRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.APIKey == nil {
		return
	}
	g := v1.Group("/admin/api-keys")
	g.Use(container.Middleware.Auth.Authenticate())
	g.Use(container.Middleware.Authorization.Authorize())
	g.POST("", container.Handlers.APIKey.Create)
	g.GET("", container.Handlers.APIKey.List)
	g.DELETE("/:id", container.Handlers.APIKey.Revoke)
}

// setupMCPAPIKeyAdminRoutes mounts the admin-only MCP API key management
// endpoints. Same policy as the vendor key management above: the Casbin admin
// wildcard covers the routes and no other role holds a rule for them. The
// scope toggle additionally requires an OTP-verified session (2FA re-entry) —
// privileged mutation on a machine credential, mirroring the ops-key toggle.
func setupMCPAPIKeyAdminRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.MCPAPIKey == nil {
		return
	}
	g := v1.Group("/admin/mcp-api-keys")
	g.Use(container.Middleware.Auth.Authenticate())
	g.Use(container.Middleware.Authorization.Authorize())
	g.POST("", container.Handlers.MCPAPIKey.Create)
	g.GET("", container.Handlers.MCPAPIKey.List)
	g.DELETE("/:id", container.Handlers.MCPAPIKey.Revoke)
	g.PUT("/:id/scope", requireOTPVerified(), container.Handlers.MCPAPIKey.SetScope)
}

// setupOpsAPIKeyAdminRoutes mounts the admin-only ops API key management
// endpoints (payroll-ops-mcp channel). Same policy as the MCP key management
// above: the Casbin admin wildcard covers the routes and no other role holds
// a rule for them. The scope toggle additionally requires an OTP-verified
// session (2FA re-entry) — privileged mutation on a machine credential.
func setupOpsAPIKeyAdminRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.OpsAPIKey == nil {
		return
	}
	g := v1.Group("/admin/ops-api-keys")
	g.Use(container.Middleware.Auth.Authenticate())
	g.Use(container.Middleware.Authorization.Authorize())
	g.POST("", container.Handlers.OpsAPIKey.Create)
	g.GET("", container.Handlers.OpsAPIKey.List)
	g.DELETE("/:id", container.Handlers.OpsAPIKey.Revoke)
	g.PUT("/:id/scope", requireOTPVerified(), container.Handlers.OpsAPIKey.SetScope)

	// Machine self-introspection: which tools may THIS key expose right now.
	ops := v1.Group("/ops")
	ops.Use(container.Middleware.Auth.Authenticate())
	ops.Use(container.Middleware.Authorization.Authorize())
	ops.GET("/key-info", container.Handlers.OpsAPIKey.SelfInfo)
}

// requireOTPVerified gates a route on the email-OTP second factor
// independently of the global RT-C1 flag: privileged mutations on machine
// credentials (the ops scope toggle) demand a 2FA-verified session. When
// email-OTP is disabled, login already issues otp_verified=true, so the gate
// degrades gracefully; when enabled, the admin must have completed
// /auth/login/verify for this session.
func requireOTPVerified() gin.HandlerFunc {
	return func(c *gin.Context) {
		if v, _ := c.Get("otp_verified"); v != true {
			response.Forbidden(c, constants.MsgOTPVerificationRequiredVN)
			c.Abort()
			return
		}
		c.Next()
	}
}
