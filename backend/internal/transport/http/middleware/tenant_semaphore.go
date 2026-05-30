package middleware

import (
	"net/http"
	"sync"
	"time"

	"api-server/internal/infra/observability"

	"github.com/gin-gonic/gin"
)

// TenantSemaphoreMiddleware limits concurrent requests per tenant (by user_id or tenant_id set in context)
// It uses a map of semaphores keyed by tenant identifier.
type TenantSemaphoreMiddleware struct {
	mu         sync.Mutex
	semaphores map[string]chan struct{}
	limit      int
}

func NewTenantSemaphoreMiddleware(limit int) *TenantSemaphoreMiddleware {
	return &TenantSemaphoreMiddleware{
		semaphores: make(map[string]chan struct{}),
		limit:      limit,
	}
}

func (t *TenantSemaphoreMiddleware) getSemaphore(key string) chan struct{} {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.semaphores[key]
	if !ok {
		s = make(chan struct{}, t.limit)
		t.semaphores[key] = s
	}
	return s
}

func (t *TenantSemaphoreMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Expect middleware earlier to set "tenant_id" in context; fallback to IP if not set
		tenant := c.GetString("tenant_id")
		if tenant == "" {
			tenant = c.ClientIP()
		}

		sem := t.getSemaphore(tenant)

		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
			return
		case <-time.After(5 * time.Second):
			// Record metric for tenant throttle
			observability.IncrementTenantThrottle(tenant)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"status": "error", "message": "Too many concurrent requests for tenant"})
			return
		}
	}
}
