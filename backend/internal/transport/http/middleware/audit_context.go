package middleware

import (
	"context"

	auditctx "api-server/internal/pkg/context"

	"github.com/gin-gonic/gin"
)

// AuditContext middleware adds IP address and user agent to the request context
// for audit logging purposes
func AuditContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract client IP address
		ipAddress := c.ClientIP()

		// Extract user agent
		userAgent := c.GetHeader("User-Agent")

		// Add audit information to the context
		ctx := auditctx.WithAuditInfo(c.Request.Context(), ipAddress, userAgent)

		// Also set standard context keys for the audit plugin
		ctx = context.WithValue(ctx, auditctx.StandardIPKey, ipAddress)
		ctx = context.WithValue(ctx, auditctx.StandardUAKey, userAgent)

		// Replace the request context with the enriched one
		c.Request = c.Request.WithContext(ctx)

		// Continue to the next handler
		c.Next()
	}
}
