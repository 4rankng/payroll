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
		// The normalised scope — never the raw column — so a legacy row with an
		// empty scope resolves to the chatbot default instead of matching nothing.
		c.Set(constants.CtxAPIKeyScope, string(domain.NormalizeAPIKeyScope(key.Scope)))
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
// for the surface being called — a chatbot key cannot read the vendor
// timesheets, and a vendor key cannot drive the chatbot.
//
// It must always be mounted AFTER Authenticate() so the 401 semantics stay
// unchanged: an absent/invalid key is 401 from Authenticate, and only a key
// that is genuinely valid but under-scoped reaches this guard as 403. Running
// it before authentication would also let a caller distinguish "key exists but
// lacks permission" from "no such key" without holding any valid credential.
//
// A missing scope in context fails closed (403) rather than defaulting to a
// match: if Authenticate() was ever skipped by a routing mistake, the guarded
// route must not become unprotected.
func (m *APIKeyAuthMiddleware) RequireAPIKeyScope(required domain.APIKeyScope) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.Get(constants.CtxAPIKeyScope)
		if !ok {
			response.Forbidden(c, constants.MsgAPIKeyScopeDeniedVN)
			c.Abort()
			return
		}
		granted, _ := raw.(string)
		if domain.NormalizeAPIKeyScope(domain.APIKeyScope(granted)) != required {
			response.Forbidden(c, constants.MsgAPIKeyScopeDeniedVN)
			c.Abort()
			return
		}
		c.Next()
	}
}
