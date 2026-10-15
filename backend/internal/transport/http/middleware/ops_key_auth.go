package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	auditservice "api-server/internal/app/services/audit"
	"api-server/internal/app/services/opsapikey"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// HeaderOpsMCPTool names the MCP tool issuing the REST call (caller
// metadata only — recorded in the audit row, never trusted for authority).
const HeaderOpsMCPTool = "X-MCP-Tool"

// opsDeniedPathPrefixes are surfaces a pk_ops_* key must never reach, however
// broad its scope: authentication, credential management (including its own
// key console — scope toggles are human/OTP actions), inbound webhooks, the
// vendor surface, and /api/v1/mcp — the chat-integration surface, a different
// credential namespace by design. Every hit is a wrong-scope probe: 403 plus
// an audit alert row.
var opsDeniedPathPrefixes = []string{
	"/api/v1/auth",
	"/api/v1/admin/api-keys",
	"/api/v1/admin/mcp-api-keys",
	"/api/v1/admin/ops-api-keys",
	"/api/v1/webhooks",
	"/api/v1/vendor",
	"/api/v1/mcp",
}

// maxHashedBodyBytes caps how much of a request body feeds the audit payload
// hash. Tool payloads are small; the cap is a guard, not a functional limit.
const maxHashedBodyBytes = 1 << 20

// opsBucket is one fixed-window counter.
type opsBucket struct {
	windowStart time.Time
	count       int
}

// opsRateLimiter is the per-key ops-tier rate limit: a dedicated in-process
// pool with its own budget, deliberately sharing nothing with the per-IP /
// per-account limiters or the chat-integration channel.
type opsRateLimiter struct {
	mu      sync.Mutex
	limit   int
	buckets map[uint]*opsBucket
}

func newOpsRateLimiter(limit int) *opsRateLimiter {
	return &opsRateLimiter{limit: limit, buckets: map[uint]*opsBucket{}}
}

// Allow reports whether keyID still has budget in the current minute window.
func (l *opsRateLimiter) Allow(keyID uint, now time.Time) bool {
	if l == nil || l.limit <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	// Opportunistic sweep so the map cannot grow unbounded.
	if len(l.buckets) > 10_000 {
		for id, b := range l.buckets {
			if now.Sub(b.windowStart) >= time.Minute {
				delete(l.buckets, id)
			}
		}
	}
	b := l.buckets[keyID]
	if b == nil || now.Sub(b.windowStart) >= time.Minute {
		b = &opsBucket{windowStart: now}
		l.buckets[keyID] = b
	}
	b.count++
	return b.count <= l.limit
}

// hashOpsRequestBody computes "sha256:<16 hex>" of the request body for the
// audit row, restoring the body for downstream handlers. The raw body is
// never persisted — only its digest.
func hashOpsRequestBody(c *gin.Context) string {
	if c.Request == nil || c.Request.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxHashedBodyBytes))
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	if err != nil || len(raw) == 0 {
		return ""
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])[:16]
}

// authenticateOpsKey resolves a machine request carrying a pk_ops_* key.
// The key is ADMIN-EQUIVALENT: it executes as its creating admin (role
// resolved at request time; a deactivated creator disables the key), so the
// app's own RBAC and business rules apply exactly as for an admin session.
// The breadth limiter is the key's scope — read_only permits GET/HEAD only,
// read_write permits every verb — enforced per request, never cached.
//
// Enforcement order is deliberate: kill switch → credential → wrong-surface
// deny (with audit alert) → per-key rate limit → scope-at-time verb gate.
func (m *AuthMiddleware) authenticateOpsKey(c *gin.Context) {
	if m.opsKeys == nil {
		response.Unauthorized(c, constants.MsgOpsKeyInvalidVN)
		c.Abort()
		return
	}

	// Kill switch: fail closed within one request cycle, structured error.
	if m.opsCfg.Disabled {
		response.ServiceUnavailableWithCode(c, constants.MsgOpsMCPDisabledVN, "ops_mcp_disabled")
		c.Abort()
		return
	}

	raw := c.GetHeader("X-API-Key")
	key, user, err := m.opsKeys.Authenticate(c.Request.Context(), raw)
	if err != nil {
		response.Unauthorized(c, constants.MsgOpsKeyInvalidVN)
		c.Abort()
		return
	}

	// Wrong-surface deny: 403 (never 200) plus an audit alert — a valid key
	// probing another channel is a signal worth investigating.
	path := c.Request.URL.Path
	for _, prefix := range opsDeniedPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			m.opsKeys.LogScopeProbe(c.Request.Context(), safeKeyPrefix(raw), c.Request.Method, path,
				c.ClientIP(), c.Request.UserAgent())
			response.ForbiddenWithCode(c, constants.MsgOpsKeyWrongSurfaceVN, "ops_wrong_surface")
			c.Abort()
			return
		}
	}

	now := clock.Now()
	if !m.opsLimiter.Allow(key.ID, now) {
		response.TooManyRequests(c, constants.MsgOpsRateLimitVN, 60)
		c.Abort()
		return
	}

	// Scope-at-time: an admin flip to read_only or a lapsed write window
	// applies to THIS request — scope is never cached at authentication.
	scope := key.EffectiveScope(now)
	if key.WriteWindowExpired(now) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			m.opsKeys.RevertExpiredWriteWindows(ctx)
		}()
	}
	if scope == domain.OpsAPIKeyScopeReadOnly && c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		response.ForbiddenWithCode(c, constants.MsgOpsKeyReadOnlyVN, "ops_read_only")
		c.Abort()
		return
	}

	// Identity: the key is the machine's credential; the creating admin is
	// the authorization subject (same model as the mcp channel). otp_verified
	// is true because the key was minted in an OTP-verified admin session.
	c.Set(constants.CtxUserID, user.ID)
	c.Set("username", user.Username)
	c.Set(constants.CtxUserRole, string(user.Role))
	c.Set("otp_verified", true)
	c.Set("tenant_id", user.ID)
	c.Set(constants.CtxOpsKeyID, key.ID)
	c.Set(constants.CtxOpsKeyAgent, key.Name)
	c.Set(constants.CtxOpsKeyScope, string(scope))

	// Audit context attributes the agent behind the admin identity.
	ctx := auditservice.WithAuditContext(c.Request.Context(), user.ID,
		fmt.Sprintf("%s (agent:%s)", user.Fullname, key.Name))
	c.Request = c.Request.WithContext(ctx)

	// Exactly one audit row per authenticated call: key id, agent identity,
	// scope-at-time, tool tag, payload digest.
	m.opsKeys.LogCall(ctx, opsapikey.CallAudit{
		KeyID:       key.ID,
		AgentName:   key.Name,
		CreatorID:   user.ID,
		ScopeAtTime: scope,
		Method:      c.Request.Method,
		Path:        path,
		Tool:        c.GetHeader(HeaderOpsMCPTool),
		PayloadHash: hashOpsRequestBody(c),
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	// Fire-and-forget last-used stamp (never blocks the response). Only the
	// key id is referenced — never key material.
	keyID := key.ID
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Default().Warn("ops key auth: touch last_used panicked", "panic", r, "ops_key_id", keyID)
			}
		}()
		touchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.opsKeys.TouchLastUsed(touchCtx, keyID); err != nil {
			slog.Default().Warn("ops key auth: touch last_used failed", "error", err, "ops_key_id", keyID)
		}
	}()

	c.Next()
}

// safeKeyPrefix returns the display-safe prefix of a presented key (never
// enough material to reconstruct the credential).
func safeKeyPrefix(raw string) string {
	if len(raw) <= 12 {
		return raw
	}
	return raw[:12]
}
