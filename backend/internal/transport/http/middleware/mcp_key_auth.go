package middleware

import (
	"strings"

	"api-server/internal/app/services/mcpapikey"
	"api-server/internal/app/services/opsapikey"
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
	opsKeys *opsapikey.Service
}

// NewMCPKeyAuthMiddleware constructs the middleware. service may be nil in
// tests — every request is then refused. opsKeys (may be nil) is consulted
// only to recognize and alert on pk_ops_* keys probing this surface.
func NewMCPKeyAuthMiddleware(service *mcpapikey.Service, opsKeys *opsapikey.Service) *MCPKeyAuthMiddleware {
	return &MCPKeyAuthMiddleware{service: service, opsKeys: opsKeys}
}

// Authenticate validates the X-API-Key header and admits valid keys. An
// absent, unknown, revoked, or orphaned key is 401 with no distinguishing
// detail. A pk_ops_* key is a valid-but-wrong-scope credential on this
// channel: 403 (never 200) plus an audit alert — the scope-probing signal.
func (m *MCPKeyAuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("X-API-Key")
		if strings.HasPrefix(raw, opsapikey.KeyPrefix) {
			if m.opsKeys != nil {
				m.opsKeys.LogScopeProbe(c.Request.Context(), safeKeyPrefix(raw), c.Request.Method,
					c.Request.URL.Path, c.ClientIP(), c.Request.UserAgent())
			}
			response.ForbiddenWithCode(c, constants.MsgOpsKeyWrongSurfaceVN, "ops_wrong_surface")
			c.Abort()
			return
		}
		if m.service == nil {
			response.Unauthorized(c, constants.MsgMCPKeyInvalidVN)
			c.Abort()
			return
		}
		if _, _, err := m.service.Authenticate(c.Request.Context(), raw); err != nil {
			response.Unauthorized(c, constants.MsgMCPKeyInvalidVN)
			c.Abort()
			return
		}
		c.Next()
	}
}
