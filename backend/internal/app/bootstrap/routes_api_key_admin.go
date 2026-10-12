package bootstrap

import (
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
// wildcard covers the routes and no other role holds a rule for them.
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
}
