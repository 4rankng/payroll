package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthCookieName is the HttpOnly cookie that carries the session JWT
// (PAYR-37 item 4: the token moved out of localStorage so client JS can no
// longer read it — XSS can no longer exfiltrate a live credential).
const AuthCookieName = "payroll_token"

// SetAuthCookie writes the session JWT as an HttpOnly cookie.
//
// SameSite=Lax: the cookie rides same-site requests and top-level GET
// navigations, never cross-site POSTs — this is what contains the CSRF risk
// of replacing the Bearer header with a browser-managed credential.
// Secure: on in production (HTTPS terminated at nginx); off in dev so
// localhost:5173 → localhost:8080 keeps working.
func SetAuthCookie(c *gin.Context, token string, maxAge time.Duration, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AuthCookieName, token, int(maxAge.Seconds()), "/", "", secure, true)
}

// ClearAuthCookie expires the session cookie (logout, migration cleanup).
func ClearAuthCookie(c *gin.Context, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AuthCookieName, "", -1, "/", "", secure, true)
}

// extractSessionToken resolves the session JWT for a request. The
// Authorization: Bearer header wins when present — legacy localStorage
// clients (cached pre-deploy SPAs, integration tests) keep authenticating
// through the migration window; otherwise the HttpOnly cookie is used.
func extractSessionToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" && parts[1] != "" {
			return parts[1]
		}
		return ""
	}
	token, err := c.Cookie(AuthCookieName)
	if err != nil || token == "" {
		return ""
	}
	return token
}

// authenticatedViaBearerHeader reports whether the request carried the token
// in the Authorization header rather than the cookie.
func authenticatedViaBearerHeader(c *gin.Context) bool {
	return strings.HasPrefix(c.GetHeader("Authorization"), "Bearer ")
}
