package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security-related HTTP headers to prevent common attacks
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Enable XSS protection (for older browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// HSTS - force HTTPS for 2 years. Emitted when the connection to this
		// process is TLS, OR when a trusted reverse proxy (nginx) signals HTTPS
		// via X-Forwarded-Proto. The latter is the real production path: nginx
		// terminates TLS and proxies plain HTTP, so c.Request.TLS is nil even on
		// the live https:// site. Keying only on c.Request.TLS (the prior
		// behavior) silently dropped HSTS on every deployment.
		isHTTPS := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
		if isHTTPS {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}

		// Permissions-Policy (formerly Feature-Policy)
		c.Header("Permissions-Policy", "geolocation=(self), microphone=(), camera=()")

		// Content Security Policy
		// - script-src: allow 'unsafe-eval' for pdfmake and CDN sources for xlsx/pdfmake lazy loading
		// - style-src: allow Google Fonts stylesheets
		// - font-src: allow Google Fonts
		// - connect-src: allow CDN fetches for xlsx and pdfmake
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-eval' https://cdnjs.cloudflare.com https://cdn.sheetjs.com https://accounts.google.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: https:; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https://cdnjs.cloudflare.com https://cdn.sheetjs.com https://fonts.googleapis.com https://accounts.google.com https://oauth2.googleapis.com; frame-src https://accounts.google.com; frame-ancestors 'none'")

		c.Next()
	}
}
