package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-server/internal/app/services/opsapikey"
	"api-server/internal/config"
	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeOpsKeyRepo is an in-memory domain.OpsAPIKeyRepository.
type fakeOpsKeyRepo struct {
	byHash map[string]*domain.OpsAPIKey
	byID   map[uint]*domain.OpsAPIKey
	nextID uint
}

func newFakeOpsKeyRepo() *fakeOpsKeyRepo {
	return &fakeOpsKeyRepo{byHash: map[string]*domain.OpsAPIKey{}, byID: map[uint]*domain.OpsAPIKey{}}
}

func (r *fakeOpsKeyRepo) Create(_ context.Context, key *domain.OpsAPIKey) error {
	r.nextID++
	key.ID = r.nextID
	clone := *key
	r.byHash[key.KeyHash] = &clone
	r.byID[key.ID] = &clone
	return nil
}

func (r *fakeOpsKeyRepo) GetByHash(_ context.Context, hash string) (*domain.OpsAPIKey, error) {
	if key, ok := r.byHash[hash]; ok {
		clone := *key
		return &clone, nil
	}
	return nil, domain.NewNotFoundError("ops api key not found")
}

func (r *fakeOpsKeyRepo) GetByID(_ context.Context, id uint) (*domain.OpsAPIKey, error) {
	if key, ok := r.byID[id]; ok {
		clone := *key
		return &clone, nil
	}
	return nil, domain.NewNotFoundError("ops api key not found")
}

func (r *fakeOpsKeyRepo) List(_ context.Context) ([]*domain.OpsAPIKey, error) { return nil, nil }

func (r *fakeOpsKeyRepo) Revoke(_ context.Context, id uint, at time.Time) error {
	for _, key := range r.byHash {
		if key.ID == id {
			key.RevokedAt = &at
			return nil
		}
	}
	return domain.NewNotFoundError("ops api key not found")
}

func (r *fakeOpsKeyRepo) UpdateLastUsedAt(_ context.Context, _ uint, _ time.Time) error { return nil }

func (r *fakeOpsKeyRepo) UpdateScope(_ context.Context, id uint, scope domain.OpsAPIKeyScope, window *time.Time) error {
	for _, key := range r.byHash {
		if key.ID == id {
			key.Scope = scope
			key.WriteWindowExpiresAt = window
		}
	}
	for _, key := range r.byID {
		if key.ID == id {
			key.Scope = scope
			key.WriteWindowExpiresAt = window
		}
	}
	return nil
}

func (r *fakeOpsKeyRepo) RevertExpiredWriteWindows(_ context.Context, now time.Time) (int64, error) {
	var n int64
	for _, key := range r.byID {
		if key.Scope == domain.OpsAPIKeyScopeReadWrite &&
			key.WriteWindowExpiresAt != nil && now.After(*key.WriteWindowExpiresAt) {
			key.Scope = domain.OpsAPIKeyScopeReadOnly
			key.WriteWindowExpiresAt = nil
			n++
		}
	}
	return n, nil
}

// fakeOpsUsers implements only GetByID.
type fakeOpsUsers struct {
	domain.UserRepository
	byID map[uint]*domain.User
}

func (f *fakeOpsUsers) GetByID(_ context.Context, id uint) (*domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domain.NewNotFoundError("user not found")
}

// seedOpsKey creates one key of the given scope and returns its plaintext.
func seedOpsKey(t *testing.T, repo *fakeOpsKeyRepo, users *fakeOpsUsers, scope domain.OpsAPIKeyScope, creatorID uint) string {
	t.Helper()
	svc := opsapikey.NewService(repo, users, nil, nil)
	_, plaintext, err := svc.Create(context.Background(), "hermes", scope, creatorID, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return plaintext
}

func opsAuthFixture(t *testing.T) (*fakeOpsKeyRepo, *fakeOpsUsers, *opsapikey.Service) {
	t.Helper()
	users := &fakeOpsUsers{byID: map[uint]*domain.User{
		7: {ID: 7, Username: "admin", Fullname: "Quản trị", Role: domain.RoleAdmin},
	}}
	repo := newFakeOpsKeyRepo()
	return repo, users, opsapikey.NewService(repo, users, nil, nil)
}

// runOpsAuthGate wires AuthMiddleware.Authenticate() around a trivial handler
// and returns the recorder plus the values the handler saw in context.
func runOpsAuthGate(t *testing.T, svc *opsapikey.Service, opsCfg config.OpsMCPConfig, method, path, apiKey string, body io.Reader) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	m := NewAuthMiddleware(nil, nil, svc, opsCfg)

	r := gin.New()
	r.Use(m.Authenticate())
	got := map[string]any{}
	r.Any("/*path", func(c *gin.Context) {
		for _, k := range []string{constants.CtxUserID, constants.CtxUserRole, constants.CtxOpsKeyID, constants.CtxOpsKeyAgent, constants.CtxOpsKeyScope} {
			if v, ok := c.Get(k); ok {
				got[k] = v
			}
		}
		c.String(http.StatusOK, "reached handler")
	})

	req := httptest.NewRequest(method, path, body)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec, got
}

func opsCfgOn() config.OpsMCPConfig { return config.OpsMCPConfig{Disabled: false, RatePerMin: 1000} }

// TestOpsKeyReadOnlyGETRunsAsCreatingAdmin: the key is admin-equivalent — a
// read reaches the handler with the creator's id and role, plus the ops-key
// audit context.
func TestOpsKeyReadOnlyGETRunsAsCreatingAdmin(t *testing.T) {
	repo, users, svc := opsAuthFixture(t)
	plaintext := seedOpsKey(t, repo, users, domain.OpsAPIKeyScopeReadOnly, 7)

	rec, got := runOpsAuthGate(t, svc, opsCfgOn(), http.MethodGet, "/api/v1/employees", plaintext, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "reached handler", rec.Body.String())
	assert.Equal(t, uint(7), got[constants.CtxUserID])
	assert.Equal(t, "admin", got[constants.CtxUserRole], "ops key executes as its creating admin")
	assert.Equal(t, uint(1), got[constants.CtxOpsKeyID])
	assert.Equal(t, "hermes", got[constants.CtxOpsKeyAgent])
	assert.Equal(t, "read_only", got[constants.CtxOpsKeyScope])
}

func TestOpsKeyReadOnlyRejectsMutation(t *testing.T) {
	repo, users, svc := opsAuthFixture(t)
	plaintext := seedOpsKey(t, repo, users, domain.OpsAPIKeyScopeReadOnly, 7)

	rec, _ := runOpsAuthGate(t, svc, opsCfgOn(), http.MethodPost, "/api/v1/timesheets/preview", plaintext, nil)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "chỉ đọc")
	assert.Contains(t, rec.Body.String(), "ops_read_only")
	assert.NotContains(t, rec.Body.String(), "reached handler")
}

// TestOpsKeyReadWriteAllowsMutation: read_write is the full admin breadth —
// mutations reach the handler (Casbin admin policy applies downstream).
func TestOpsKeyReadWriteAllowsMutation(t *testing.T) {
	repo, users, svc := opsAuthFixture(t)
	plaintext := seedOpsKey(t, repo, users, domain.OpsAPIKeyScopeReadWrite, 7)

	rec, _ := runOpsAuthGate(t, svc, opsCfgOn(), http.MethodPost, "/api/v1/timesheets/preview", plaintext, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "reached handler", rec.Body.String())
}

// TestOpsKeyStaleScopeBlocksInFlightMutations: an admin flip to read_only
// applies to the very next request — enforcement is per request, not cached
// at authentication (red-team stale-scope constraint).
func TestOpsKeyStaleScopeBlocksInFlightMutations(t *testing.T) {
	repo, users, svc := opsAuthFixture(t)
	plaintext := seedOpsKey(t, repo, users, domain.OpsAPIKeyScopeReadWrite, 7)

	rec, _ := runOpsAuthGate(t, svc, opsCfgOn(), http.MethodPost, "/api/v1/timesheets/preview", plaintext, nil)
	require.Equal(t, http.StatusOK, rec.Code, "read_write must allow mutations before the flip")

	_, err := svc.SetScope(context.Background(), 1, domain.OpsAPIKeyScopeReadOnly, nil, 7)
	require.NoError(t, err)

	rec, _ = runOpsAuthGate(t, svc, opsCfgOn(), http.MethodPost, "/api/v1/timesheets/preview", plaintext, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.NotContains(t, rec.Body.String(), "reached handler")
}

// TestOpsKeyWrongSurfaceIs403: a pk_ops_* key on the chat-integration surface
// (/api/v1/mcp) or credential management is 403 — never 200 — with the
// structured ops_wrong_surface code.
func TestOpsKeyWrongSurfaceIs403(t *testing.T) {
	repo, users, svc := opsAuthFixture(t)
	plaintext := seedOpsKey(t, repo, users, domain.OpsAPIKeyScopeReadWrite, 7)

	for _, path := range []string{"/api/v1/mcp", "/api/v1/mcp/sessions", "/api/v1/admin/ops-api-keys", "/api/v1/auth/login"} {
		rec, _ := runOpsAuthGate(t, svc, opsCfgOn(), http.MethodPost, path, plaintext, nil)
		assert.Equal(t, http.StatusForbidden, rec.Code, path)
		assert.Contains(t, rec.Body.String(), "ops_wrong_surface", path)
		assert.NotContains(t, rec.Body.String(), "reached handler", path)
	}
}

// TestOpsKeyKillSwitchFailsClosed: PAYROLL_OPS_MCP_DISABLED trips a
// structured 503 for every call, within one request cycle.
func TestOpsKeyKillSwitchFailsClosed(t *testing.T) {
	repo, users, svc := opsAuthFixture(t)
	plaintext := seedOpsKey(t, repo, users, domain.OpsAPIKeyScopeReadWrite, 7)
	cfg := config.OpsMCPConfig{Disabled: true, RatePerMin: 1000}

	rec, _ := runOpsAuthGate(t, svc, cfg, http.MethodGet, "/api/v1/employees", plaintext, nil)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "ops_mcp_disabled")
	assert.Contains(t, rec.Body.String(), "PAYROLL_OPS_MCP_DISABLED")
	assert.NotContains(t, rec.Body.String(), "reached handler")
}

func TestOpsKeyUnknownKeyIs401(t *testing.T) {
	repo, users, svc := opsAuthFixture(t)
	_ = repo
	_ = users

	rec, _ := runOpsAuthGate(t, svc, opsCfgOn(), http.MethodGet, "/api/v1/employees", "pk_ops_unknown", nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestOpsKeyRateLimitSeparateTier: the per-key ops budget is its own pool —
// exceeding it yields 429 even for read_only GETs on a valid key. One
// middleware instance (as in production) serves all requests.
func TestOpsKeyRateLimitSeparateTier(t *testing.T) {
	repo, users, svc := opsAuthFixture(t)
	plaintext := seedOpsKey(t, repo, users, domain.OpsAPIKeyScopeReadOnly, 7)
	cfg := config.OpsMCPConfig{Disabled: false, RatePerMin: 2}

	gin.SetMode(gin.TestMode)
	m := NewAuthMiddleware(nil, nil, svc, cfg)
	r := gin.New()
	r.Use(m.Authenticate())
	r.Any("/*path", func(c *gin.Context) { c.String(http.StatusOK, "reached handler") })

	do := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
		req.Header.Set("X-API-Key", plaintext)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	require.Equal(t, http.StatusOK, do().Code, "first call within budget")
	require.Equal(t, http.StatusOK, do().Code, "second call within budget")
	assert.Equal(t, http.StatusTooManyRequests, do().Code, "third call exceeds the per-key budget")
}

// TestMCPHandshakeRejectsOpsKey: the chat-integration handshake gate treats a
// pk_ops_* key as a wrong-scope probe — 403, not 401, so probing is
// distinguishable from guessing.
func TestMCPHandshakeRejectsOpsKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo, users, svc := opsAuthFixture(t)
	_ = repo
	_ = users

	m := NewMCPKeyAuthMiddleware(nil, svc)
	r := gin.New()
	r.Use(m.Authenticate())
	r.Any("/api/v1/mcp", func(c *gin.Context) { c.String(http.StatusOK, "reached mcp") })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", nil)
	req.Header.Set("X-API-Key", "pk_ops_probably_valid_shaped_key")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "ops_wrong_surface")
	assert.NotContains(t, rec.Body.String(), "reached mcp")
}

// TestOpsKeyNilServiceFailsClosed: with no service wired a pk_ops_ key is
// refused rather than trusted.
func TestOpsKeyNilServiceFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewAuthMiddleware(nil, nil, nil, opsCfgOn())
	r := gin.New()
	r.Use(m.Authenticate())
	r.Any("/*path", func(c *gin.Context) { c.String(http.StatusOK, "reached handler") })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	req.Header.Set("X-API-Key", "pk_ops_anything")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
