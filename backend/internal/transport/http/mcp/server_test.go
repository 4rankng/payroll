package mcptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/services/mcpapikey"
	"api-server/internal/domain"
	"api-server/internal/transport/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- fakes -----------------------------------------------------------------

type fakeKeyRepo struct {
	byHash map[string]*domain.MCPAPIKey
	nextID uint
}

func (r *fakeKeyRepo) Create(_ context.Context, key *domain.MCPAPIKey) error {
	r.nextID++
	key.ID = r.nextID
	clone := *key
	r.byHash[key.KeyHash] = &clone
	return nil
}

func (r *fakeKeyRepo) GetByHash(_ context.Context, hash string) (*domain.MCPAPIKey, error) {
	if key, ok := r.byHash[hash]; ok {
		clone := *key
		return &clone, nil
	}
	return nil, domain.NewNotFoundError("mcp api key not found")
}

func (r *fakeKeyRepo) List(_ context.Context) ([]*domain.MCPAPIKey, error) { return nil, nil }

func (r *fakeKeyRepo) Revoke(_ context.Context, _ uint, _ time.Time) error {
	return domain.NewNotFoundError("mcp api key not found")
}

func (r *fakeKeyRepo) UpdateLastUsedAt(_ context.Context, _ uint, _ time.Time) error { return nil }

type fakeUsers struct {
	domain.UserRepository
	byID map[uint]*domain.User
}

func (f *fakeUsers) GetByID(_ context.Context, id uint) (*domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domain.NewNotFoundError("user not found")
}

// newTestEngine builds the real Auth MCP key branch plus stub routes and the
// MCP endpoint itself — the whole path a production request takes.
func newTestEngine() (*gin.Engine, *mcpapikey.Service) {
	gin.SetMode(gin.TestMode)
	repo := &fakeKeyRepo{byHash: map[string]*domain.MCPAPIKey{}}
	users := &fakeUsers{byID: map[uint]*domain.User{7: {ID: 7, Username: "admin", Role: domain.RoleAdmin}}}
	keys := mcpapikey.NewService(repo, users, nil)

	engine := gin.New()
	auth := middleware.NewAuthMiddleware(nil, keys)
	v1 := engine.Group("/api/v1")
	// Stub REST surface behind the real auth chain (mirrors the protected
	// group in production).
	protected := v1.Group("")
	protected.Use(auth.Authenticate())
	protected.GET("/echo", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"ok": true}})
	})
	protected.POST("/echo", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"ok": true}})
	})
	// The MCP endpoint itself carries only the handshake gate — exactly like
	// production, where it is not mounted on the protected group.
	g := v1.Group("/mcp")
	g.Use(middleware.NewMCPKeyAuthMiddleware(keys).Authenticate())
	g.Any("", gin.WrapH(NewHandler(engine, keys, nil)))
	return engine, keys
}

func mint(t *testing.T, keys *mcpapikey.Service, scope domain.MCPAPIKeyScope) string {
	t.Helper()
	_, plaintext, err := keys.Create(context.Background(), "agent", scope, 7)
	require.NoError(t, err)
	return plaintext
}

// --- raw JSON-RPC over Streamable HTTP -------------------------------------

type rpcResponse struct {
	ID     any             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

// rpcCall POSTs one JSON-RPC message with the caller's key and decodes the
// response. Streamable HTTP may answer with SSE or plain JSON depending on
// negotiation; both are handled.
func rpcCall(t *testing.T, engine *gin.Engine, sessionID, apiKey string, payload map[string]any) (*httptest.ResponseRecorder, *rpcResponse) {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("X-API-Key", apiKey)
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if payload["id"] == nil {
		return rec, nil
	}
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	raw := rec.Body.String()
	if strings.HasPrefix(rec.Header().Get("Content-Type"), "text/event-stream") {
		raw = ""
		for _, line := range strings.Split(rec.Body.String(), "\n") {
			if strings.HasPrefix(line, "data: ") {
				raw = strings.TrimPrefix(line, "data: ")
			}
		}
	}
	require.NotEmpty(t, raw, "no JSON-RPC message in response: %s", rec.Body.String())
	var resp rpcResponse
	require.NoError(t, json.Unmarshal([]byte(raw), &resp), raw)
	return rec, &resp
}

// startSession runs initialize + notifications/initialized and returns the
// session id.
func startSession(t *testing.T, engine *gin.Engine, apiKey string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
		},
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("X-API-Key", apiKey)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// The server name must surface in the initialize result.
	require.Contains(t, rec.Body.String(), `"payroll"`)

	sessionID := rec.Header().Get("Mcp-Session-Id")
	notify, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	nreq := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(string(notify)))
	nreq.Header.Set("Content-Type", "application/json")
	nreq.Header.Set("Accept", "application/json, text/event-stream")
	nreq.Header.Set("X-API-Key", apiKey)
	if sessionID != "" {
		nreq.Header.Set("Mcp-Session-Id", sessionID)
	}
	nrec := httptest.NewRecorder()
	engine.ServeHTTP(nrec, nreq)
	return sessionID
}

// callTool runs one tools/call and returns the tool text + IsError.
func callTool(t *testing.T, engine *gin.Engine, sessionID, apiKey, name string, args map[string]any) (string, bool) {
	t.Helper()
	_, resp := rpcCall(t, engine, sessionID, apiKey, map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call",
		"params": map[string]any{"name": name, "arguments": args},
	})
	require.NotNil(t, resp)
	require.Empty(t, resp.Error, string(resp.Error))

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	require.NoError(t, json.Unmarshal(resp.Result, &result), string(resp.Result))
	require.NotEmpty(t, result.Content)
	return result.Content[0].Text, result.IsError
}

// --- tests -----------------------------------------------------------------

func TestPayrollEndpointsListsEchoAndExcludesGuarded(t *testing.T) {
	engine, keys := newTestEngine()
	apiKey := mint(t, keys, domain.MCPAPIKeyScopeReadOnly)
	sessionID := startSession(t, engine, apiKey)

	text, isError := callTool(t, engine, sessionID, apiKey, "payroll_endpoints", map[string]any{})

	assert.False(t, isError)
	assert.Contains(t, text, "GET /api/v1/echo")
	assert.Contains(t, text, "[read]")
	assert.NotContains(t, text, "/api/v1/mcp")
	assert.NotContains(t, text, "/api/v1/auth")
}

func TestPayrollAPIGetEcho(t *testing.T) {
	engine, keys := newTestEngine()
	apiKey := mint(t, keys, domain.MCPAPIKeyScopeReadOnly)
	sessionID := startSession(t, engine, apiKey)

	text, isError := callTool(t, engine, sessionID, apiKey, "payroll_api", map[string]any{
		"method": "GET", "path": "/api/v1/echo",
	})

	assert.False(t, isError)
	assert.True(t, strings.HasPrefix(text, "HTTP 200"), text)
	assert.Contains(t, text, `{"ok":true}`)
}

func TestPayrollAPIReadOnlyRejectsMutation(t *testing.T) {
	engine, keys := newTestEngine()
	apiKey := mint(t, keys, domain.MCPAPIKeyScopeReadOnly)
	sessionID := startSession(t, engine, apiKey)

	text, isError := callTool(t, engine, sessionID, apiKey, "payroll_api", map[string]any{
		"method": "POST", "path": "/api/v1/echo", "body": map[string]any{},
	})

	assert.True(t, isError)
	assert.Contains(t, text, "chỉ đọc")
}

func TestPayrollAPIGuardedPathRejected(t *testing.T) {
	engine, keys := newTestEngine()
	apiKey := mint(t, keys, domain.MCPAPIKeyScopeFull)
	sessionID := startSession(t, engine, apiKey)

	text, isError := callTool(t, engine, sessionID, apiKey, "payroll_api", map[string]any{
		"method": "POST", "path": "/api/v1/admin/api-keys", "body": map[string]any{},
	})

	assert.True(t, isError)
	assert.Contains(t, text, "không được phép truy cập qua MCP")
}

func TestPayrollAPIFullScopeAllowsMutation(t *testing.T) {
	engine, keys := newTestEngine()
	apiKey := mint(t, keys, domain.MCPAPIKeyScopeFull)
	sessionID := startSession(t, engine, apiKey)

	text, isError := callTool(t, engine, sessionID, apiKey, "payroll_api", map[string]any{
		"method": "POST", "path": "/api/v1/echo", "body": map[string]any{"x": 1},
	})

	assert.False(t, isError)
	assert.True(t, strings.HasPrefix(text, "HTTP 200"), text)
}

func TestMCPHandshakeRefusesInvalidKey(t *testing.T) {
	engine, _ := newTestEngine()

	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
			"clientInfo": map[string]any{"name": "test", "version": "1.0"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("X-API-Key", "mcp_bogus")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDoLoopbackDeniesGuardedPrefixes(t *testing.T) {
	engine, _ := newTestEngine()
	for _, path := range []string{
		"/api/v1/auth/login",
		"/api/v1/admin/api-keys",
		"/api/v1/admin/mcp-api-keys",
		"/api/v1/webhooks/x",
		"/api/v1/mcp",
		"/api/v1/vendor/projects",
		"/elsewhere",
	} {
		_, _, err := doLoopback(engine, "mcp_x", http.MethodGet, path, "", nil)
		assert.ErrorContains(t, err, "không được phép truy cập qua MCP", path)
	}
}
