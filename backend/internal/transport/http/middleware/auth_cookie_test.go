package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authservice "api-server/internal/app/services/auth"
	"api-server/internal/app/services/user"
	"api-server/internal/config"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cookieTestUserRepo satisfies the UserRepository dependency ValidateToken
// touches (GetByID); any error is swallowed by tokenInvalidatedForUser, so a
// miss is enough. Other methods panic via the embedded interface if invoked —
// none are on the validation path.
type cookieTestUserRepo struct {
	domain.UserRepository
}

func (f *cookieTestUserRepo) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	return nil, errors.New("not found")
}

type cookieTestBlacklistRepo struct {
	domain.BlacklistedTokenRepository
}

func (f *cookieTestBlacklistRepo) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	return false, nil
}

func newCookieTestAuthService(t *testing.T, accessTTL time.Duration) *authservice.AuthService {
	t.Helper()
	userSvc := user.NewUserService(&cookieTestUserRepo{}, nil, nil, "h", "s", nil)
	return authservice.NewAuthService(
		userSvc, nil, &cookieTestBlacklistRepo{}, nil, "test-secret", accessTTL,
		nil, config.OTPConfig{}, "", nil, nil, nil, nil,
	)
}

func newCookieTestToken(t *testing.T, svc *authservice.AuthService, ttl time.Duration) string {
	t.Helper()
	claims := &authservice.Claims{
		UserID:      42,
		Username:    "itest-user",
		Fullname:    "Integration User",
		Role:        "admin",
		OTPVerified: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "cookie-test-jti",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   "access",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)
	return signed
}

func newCookieTestRouter(svc *authservice.AuthService, m *AuthMiddleware) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.Authenticate())
	r.GET("/api/v1/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "reached handler")
	})
	_ = svc
	return r
}

// Pin: the HttpOnly cookie alone authenticates a request — the migration's
// primary path once the SPA stops holding the token in localStorage.
func TestAuthenticateAcceptsSessionCookie(t *testing.T) {
	svc := newCookieTestAuthService(t, 14*24*time.Hour)
	token := newCookieTestToken(t, svc, time.Hour)
	m := NewAuthMiddleware(svc, nil, nil, config.OpsMCPConfig{}, false)
	r := newCookieTestRouter(svc, m)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.AddCookie(&http.Cookie{Name: AuthCookieName, Value: token})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "reached handler", rec.Body.String())
}

// Pin: Bearer header wins when both channels are present (legacy header
// takes precedence; a stale cookie can't shadow a fresh header token).
func TestAuthenticateBearerHeaderWinsOverCookie(t *testing.T) {
	svc := newCookieTestAuthService(t, 14*24*time.Hour)
	good := newCookieTestToken(t, svc, time.Hour)
	m := NewAuthMiddleware(svc, nil, nil, config.OpsMCPConfig{}, false)
	r := newCookieTestRouter(svc, m)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.AddCookie(&http.Cookie{Name: AuthCookieName, Value: "garbage-not-a-jwt"})
	req.Header.Set("Authorization", "Bearer "+good)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// Pin: missing header AND cookie → 401 (unchanged contract).
func TestAuthenticateMissingCredentialRejected(t *testing.T) {
	svc := newCookieTestAuthService(t, 14*24*time.Hour)
	m := NewAuthMiddleware(svc, nil, nil, config.OpsMCPConfig{}, false)
	r := newCookieTestRouter(svc, m)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Xin đăng nhập")
}

// Pin (migration auto-upgrade): a legacy Bearer-only request gets a session
// cookie minted for the token's remaining lifetime, so cached old-SPA
// sessions convert to cookie auth on their first post-deploy request.
func TestAuthenticateBearerAutoUpgradeSetsCookie(t *testing.T) {
	svc := newCookieTestAuthService(t, 14*24*time.Hour)
	token := newCookieTestToken(t, svc, time.Hour)
	m := NewAuthMiddleware(svc, nil, nil, config.OpsMCPConfig{}, false)
	r := newCookieTestRouter(svc, m)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1, "expected exactly one Set-Cookie")
	c := cookies[0]
	assert.Equal(t, AuthCookieName, c.Name)
	assert.Equal(t, token, c.Value)
	assert.True(t, c.HttpOnly, "session cookie must be HttpOnly")
	assert.Equal(t, http.SameSiteLaxMode, c.SameSite)
	assert.False(t, c.Secure, "Secure must follow the dev flag")
	assert.Equal(t, "/", c.Path)
	// Max-Age carries the token's REMAINING lifetime (≈1h), not the full TTL.
	assert.Greater(t, c.MaxAge, 0)
	assert.LessOrEqual(t, c.MaxAge, int(time.Hour.Seconds()))

	// Already cookied → no duplicate Set-Cookie on the same request.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req2.AddCookie(&http.Cookie{Name: AuthCookieName, Value: token})
	req2.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	assert.Empty(t, rec2.Result().Cookies(), "no Set-Cookie when the cookie is already present")
}

// Pin: cookie attribute matrix — Secure follows the production flag.
func TestSetClearAuthCookieAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	build := func(secure bool) *httptest.ResponseRecorder {
		r := gin.New()
		r.GET("/set", func(c *gin.Context) {
			SetAuthCookie(c, "tok", 7*24*time.Hour, secure)
			c.String(http.StatusOK, "ok")
		})
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/set", nil))
		return rec
	}

	devRec := build(false)
	{
		cookies := devRec.Result().Cookies()
		require.Len(t, cookies, 1)
		c := cookies[0]
		assert.Equal(t, AuthCookieName, c.Name)
		assert.True(t, c.HttpOnly)
		assert.Equal(t, http.SameSiteLaxMode, c.SameSite)
		assert.False(t, c.Secure)
		assert.Equal(t, int((7 * 24 * time.Hour).Seconds()), c.MaxAge)
	}

	prodRec := build(true)
	{
		cookies := prodRec.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.True(t, cookies[0].Secure)
	}

	// Clear expires the cookie.
	r := gin.New()
	r.POST("/clear", func(c *gin.Context) {
		ClearAuthCookie(c, false)
		c.String(http.StatusOK, "ok")
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/clear", nil))
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	c := cookies[0]
	assert.Equal(t, AuthCookieName, c.Name)
	assert.LessOrEqual(t, c.MaxAge, 0, "clear must expire the cookie")
}
