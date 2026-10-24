package middleware

import (
	"context"
	"log/slog"
	"time"

	"api-server/internal/app/services/apikey"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// APIKeyAuthMiddleware authenticates machine callers via the X-API-Key header.
// It is mutually exclusive with JWT auth: integration routes carry no user
// identity, so this middleware sets api_key_id/api_key_name (and tenant_id for
// the tenant-scoped machinery) but never user_id.
type APIKeyAuthMiddleware struct {
	service *apikey.Service
	logger  *slog.Logger
}

// NewAPIKeyAuthMiddleware constructs the middleware. logger may be nil.
func NewAPIKeyAuthMiddleware(service *apikey.Service, logger *slog.Logger) *APIKeyAuthMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &APIKeyAuthMiddleware{service: service, logger: logger}
}

// Authenticate validates the X-API-Key header and publishes the resolved key
// identity into the request context.
func (m *APIKeyAuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		plaintext := c.GetHeader("X-API-Key")
		if plaintext == "" {
			response.Unauthorized(c, constants.MsgAPIKeyMissingVN)
			c.Abort()
			return
		}

		key, err := m.service.Authenticate(c.Request.Context(), plaintext)
		if err != nil {
			response.Unauthorized(c, constants.MsgAPIKeyInvalidVN)
			c.Abort()
			return
		}

		c.Set(constants.CtxAPIKeyID, key.ID)
		c.Set(constants.CtxAPIKeyName, key.Name)
		// The raw stored scope: an empty or unknown value must stay as-is so
		// every scope comparison fails closed instead of matching a default.
		c.Set(constants.CtxAPIKeyScope, string(key.Scope))
		// tenant_id is used by tenant-scoped middleware/utilities; a machine
		// key has no user, so its creator is the closest stable tenant anchor.
		c.Set("tenant_id", key.CreatedBy)

		// Fire-and-forget last-used stamp (never blocks the response).
		keyID := key.ID
		go func() {
			defer func() {
				if r := recover(); r != nil {
					m.logger.Warn("api key auth: touch last_used panicked", "panic", r, "api_key_id", keyID)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := m.service.TouchLastUsed(ctx, keyID); err != nil {
				m.logger.Warn("api key auth: touch last_used failed", "error", err, "api_key_id", keyID)
			}
		}()

		c.Next()
	}
}

// RequireAPIKeyScope refuses a request whose authenticated key was not minted
// for the surface being called — only a key bound to exactly that scope may
// pass.
//
// It must always be mounted AFTER Authenticate() so the 401 semantics stay
// unchanged: an absent/invalid key is 401 from Authenticate, and only a key
// that is genuinely valid but under-scoped reaches this guard as 403. Running
// it before authentication would also let a caller distinguish "key exists but
// lacks permission" from "no such key" without holding any valid credential.
//
// A missing or unrecognised scope fails closed (403) rather than defaulting to
// a match: if Authenticate() was ever skipped by a routing mistake, the guarded
// route must not become unprotected.
func (m *APIKeyAuthMiddleware) RequireAPIKeyScope(required ...domain.APIKeyScope) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.Get(constants.CtxAPIKeyScope)
		if !ok {
			response.Forbidden(c, constants.MsgAPIKeyScopeDeniedVN)
			c.Abort()
			return
		}
		granted, _ := raw.(string)
		grantedScope := domain.APIKeyScope(granted)
		for _, req := range required {
			if grantedScope == req {
				c.Next()
				return
			}
			// TingHire surface accepts the legacy chatbot scope: keys minted
			// before the scope was normalized to 'tinghire' keep working.
			// timesheet_read is deliberately NOT accepted here — it belongs to
			// the read-only vendor surface and must not drive integration
			// (employee lookup, OTP reset, self check-in) endpoints.
			if req == domain.APIKeyScopeTingHire && grantedScope == domain.APIKeyScopeChatbot {
				c.Next()
				return
			}
		}
		response.Forbidden(c, constants.MsgAPIKeyScopeDeniedVN)
		c.Abort()
	}
}
