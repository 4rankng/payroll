package bootstrap

import (
	mcptransport "api-server/internal/transport/http/mcp"

	"github.com/gin-gonic/gin"
)

// setupMCPRoutes mounts the MCP agent surface (Streamable HTTP) at
// /api/v1/mcp. The endpoint itself requires a valid MCP key (handshake gate);
// every tool call loops back through the real engine with that key, so the
// REST middleware chain stays the single authorization authority.
func setupMCPRoutes(engine *gin.Engine, v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Middleware == nil || container.Middleware.MCPKeyAuth == nil {
		return
	}
	g := v1.Group("/mcp")
	g.Use(container.Middleware.MCPKeyAuth.Authenticate())
	g.Any("", gin.WrapH(mcptransport.NewHandler(engine, container.Services.MCPAPIKey, container.Logger)))
}
