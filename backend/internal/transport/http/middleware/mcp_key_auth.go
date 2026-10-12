package middleware

import (
	"api-server/internal/app/services/mcpapikey"
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// MCPKeyAuthMiddleware guards the MCP endpoint itself. It validates the
// X-API-Key header so handshakes never reach invalid keys; the per-call
// authority is re-established by the loopback through the real Auth middleware
// on every REST call the tools make.
type MCPKeyAuthMiddleware struct {
	service *mcpapikey.Service
}

// NewMCPKeyAuthMiddleware constructs the middleware. service may be nil in
// tests — every request is then refused.
func NewMCPKeyAuthMiddleware(service *mcpapikey.Service) *MCPKeyAuthMiddleware {
	return &MCPKeyAuthMiddleware{service: service}
}

// Authenticate validates the X-API-Key header and admits valid keys. An
// absent, unknown, revoked, or orphaned key is 401 with no distinguishing
// detail.
func (m *MCPKeyAuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.service == nil {
			response.Unauthorized(c, constants.MsgMCPKeyInvalidVN)
			c.Abort()
			return
		}
		if _, _, err := m.service.Authenticate(c.Request.Context(), c.GetHeader("X-API-Key")); err != nil {
			response.Unauthorized(c, constants.MsgMCPKeyInvalidVN)
			c.Abort()
			return
		}
		c.Next()
	}
}
