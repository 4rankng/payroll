package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutOverride raises the request deadline for one path prefix — used by
// long-bound endpoints (chat turns) that enforce their own tighter deadline.
type TimeoutOverride struct {
	Prefix  string
	Timeout time.Duration
}

// RequestTimeout returns a middleware that sets a timeout on the request context
func RequestTimeout(timeout time.Duration, overrides ...TimeoutOverride) gin.HandlerFunc {
	return func(c *gin.Context) {
		if timeout <= 0 {
			c.Next()
			return
		}
		effective := timeout
		path := c.Request.URL.Path
		for _, o := range overrides {
			if strings.HasPrefix(path, o.Prefix) {
				effective = o.Timeout
				break
			}
		}
		if effective <= 0 {
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), effective)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
