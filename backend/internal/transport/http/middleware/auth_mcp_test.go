package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-server/internal/app/services/mcpapikey"
	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeMCPKeyRepo is an in-memory domain.MCPAPIKeyRepository.
type fakeMCPKeyRepo struct {
	byHash map[string]*domain.MCPAPIKey
	nextID uint
}

func newFakeMCPKeyRepo() *fakeMCPKeyRepo {
	return &fakeMCPKeyRepo{byHash: map[string]*domain.MCPAPIKey{}}
}

func (r *fakeMCPKeyRepo) Create(_ context.Context, key *domain.MCPAPIKey) error {
	r.nextID++
	key.ID = r.nextID
	clone := *key
	r.byHash[key.KeyHash] = &clone
	return nil
}

func (r *fakeMCPKeyRepo) GetByHash(_ context.Context, hash string) (*domain.MCPAPIKey, error) {
	if key, ok := r.byHash[hash]; ok {
		clone := *key
		return &clone, nil
	}
	return nil, domain.NewNotFoundError("mcp api key not found")
}

func (r *fakeMCPKeyRepo) List(_ context.Context) ([]*domain.MCPAPIKey, error) { return nil, nil }

func (r *fakeMCPKeyRepo) Revoke(_ context.Context, id uint, at time.Time) error {
	for _, key := range r.byHash {
		if key.ID == id {
			key.RevokedAt = &at
			return nil
		}
	}
	return domain.NewNotFoundError("mcp api key not found")
}

func (r *fakeMCPKeyRepo) UpdateLastUsedAt(_ context.Context, _ uint, _ time.Time) error { return nil }

// fakeMCPUsers implements only GetByID; other UserRepository methods are never
// reached.
type fakeMCPUsers struct {
	domain.UserRepository
	byID map[uint]*domain.User
}

func (f *fakeMCPUsers) GetByID(_ context.Context, id uint) (*domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domain.NewNotFoundError("user not found")
}

// seedMCPKey creates one key of the given scope and returns its plaintext.
func seedMCPKey(t *testing.T, repo *fakeMCPKeyRepo, users *fakeMCPUsers, scope domain.MCPAPIKeyScope, creatorID uint) string {
	t.Helper()
	svc := mcpapikey.NewService(repo, users, nil)
	_, plaintext, err := svc.Create(context.Background(), "agent", scope, creatorID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return plaintext
}

// runAuthGate wires AuthMiddleware.Authenticate() around a trivial handler and
// returns the recorder so a test can assert status codes and context values.
func runAuthGate(t *testing.T, svc *mcpapikey.Service, method, path, apiKey string) (*httptest.ResponseRecorder, any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	m := NewAuthMiddleware(nil, svc)

	r := gin.New()
	r.Use(m.Authenticate())
	var gotUser any
	r.Any("/*path", func(c *gin.Context) {
		gotUser, _ = c.Get(constants.CtxUserID)
		c.String(http.StatusOK, "reached handler")
	})

	req := httptest.NewRequest(method, path, nil)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec, gotUser
}

func mcpAuthFixture(t *testing.T) (*fakeMCPKeyRepo, *fakeMCPUsers) {
	t.Helper()
	users := &fakeMCPUsers{byID: map[uint]*domain.User{
		7: {ID: 7, Username: "admin", Fullname: "Quản trị", Role: domain.RoleAdmin},
	}}
	return newFakeMCPKeyRepo(), users
}

func TestMCPKeyReadOnlyGETReachesHandlerWithCreatorIdentity(t *testing.T) {
	repo, users := mcpAuthFixture(t)
	svc := mcpapikey.NewService(repo, users, nil)
	plaintext := seedMCPKey(t, repo, users, domain.MCPAPIKeyScopeReadOnly, 7)

	rec, gotUser := runAuthGate(t, svc, http.MethodGet, "/api/v1/employees", plaintext)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "reached handler", rec.Body.String())
	assert.Equal(t, uint(7), gotUser, "handler must see the creating admin's id")
}

func TestMCPKeyReadOnlyRejectsMutation(t *testing.T) {
	repo, users := mcpAuthFixture(t)
	svc := mcpapikey.NewService(repo, users, nil)
	plaintext := seedMCPKey(t, repo, users, domain.MCPAPIKeyScopeReadOnly, 7)

	rec, _ := runAuthGate(t, svc, http.MethodPost, "/api/v1/timesheets/preview", plaintext)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "chỉ đọc")
	assert.NotContains(t, rec.Body.String(), "reached handler")
}

func TestMCPKeyFullPathDeniedForCredentialManagement(t *testing.T) {
	repo, users := mcpAuthFixture(t)
	svc := mcpapikey.NewService(repo, users, nil)
	plaintext := seedMCPKey(t, repo, users, domain.MCPAPIKeyScopeFull, 7)

	for _, path := range []string{"/api/v1/admin/mcp-api-keys", "/api/v1/admin/api-keys", "/api/v1/auth/login"} {
		rec, _ := runAuthGate(t, svc, http.MethodPost, path, plaintext)
		assert.Equal(t, http.StatusForbidden, rec.Code, path)
		assert.Contains(t, rec.Body.String(), "không được truy cập", path)
	}
}

func TestMCPKeyUnknownKeyIs401(t *testing.T) {
	repo, users := mcpAuthFixture(t)
	svc := mcpapikey.NewService(repo, users, nil)

	rec, _ := runAuthGate(t, svc, http.MethodGet, "/api/v1/employees", "mcp_unknown")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.NotContains(t, rec.Body.String(), "reached handler")
}

func TestMCPKeyRevokedKeyIs401(t *testing.T) {
	repo, users := mcpAuthFixture(t)
	svc := mcpapikey.NewService(repo, users, nil)
	plaintext := seedMCPKey(t, repo, users, domain.MCPAPIKeyScopeFull, 7)

	require.NoError(t, svc.Revoke(context.Background(), 1))

	rec, _ := runAuthGate(t, svc, http.MethodGet, "/api/v1/employees", plaintext)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMCPKeyFullScopeAllowsMutation(t *testing.T) {
	repo, users := mcpAuthFixture(t)
	svc := mcpapikey.NewService(repo, users, nil)
	plaintext := seedMCPKey(t, repo, users, domain.MCPAPIKeyScopeFull, 7)

	rec, _ := runAuthGate(t, svc, http.MethodPost, "/api/v1/timesheets/preview", plaintext)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "reached handler", rec.Body.String())
}

// TestJWTPathUnchanged pins that the machine-key branch does not disturb the
// session flow: a request without X-API-Key still takes the JWT path (and with
// no Authorization header it is still the familiar 401).
func TestJWTPathUnchanged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewAuthMiddleware(nil, mcpapikey.NewService(newFakeMCPKeyRepo(), &fakeMCPUsers{byID: map[uint]*domain.User{}}, nil))

	r := gin.New()
	r.Use(m.Authenticate())
	r.GET("/api/v1/employees", func(c *gin.Context) { c.String(http.StatusOK, "reached handler") })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Xin đăng nhập")
}

// TestMCPKeyNilServiceFailsClosed: with no service wired (tests, partial
// deployments) a machine-key request is refused rather than trusted.
func TestMCPKeyNilServiceFailsClosed(t *testing.T) {
	rec, _ := runAuthGate(t, nil, http.MethodGet, "/api/v1/employees", "mcp_any")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestMCPKeyMissingCreatorFailsClosed: deleting/deactivating the creating
// admin disables the key.
func TestMCPKeyMissingCreatorFailsClosed(t *testing.T) {
	repo := newFakeMCPKeyRepo()
	users := &fakeMCPUsers{byID: map[uint]*domain.User{}}
	svc := mcpapikey.NewService(repo, users, nil)
	plaintext := seedMCPKey(t, repo, users, domain.MCPAPIKeyScopeReadOnly, 7)

	rec, _ := runAuthGate(t, svc, http.MethodGet, "/api/v1/employees", plaintext)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestMCPKeySoftDeletedCreatorFailsClosed covers the gorm soft-delete shape.
func TestMCPKeySoftDeletedCreatorFailsClosed(t *testing.T) {
	repo := newFakeMCPKeyRepo()
	deleted := &domain.User{ID: 7, Username: "admin", Role: domain.RoleAdmin,
		DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}}
	users := &fakeMCPUsers{byID: map[uint]*domain.User{7: deleted}}
	svc := mcpapikey.NewService(repo, users, nil)
	plaintext := seedMCPKey(t, repo, users, domain.MCPAPIKeyScopeReadOnly, 7)

	rec, _ := runAuthGate(t, svc, http.MethodGet, "/api/v1/employees", plaintext)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
