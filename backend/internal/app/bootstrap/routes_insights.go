package bootstrap

import (
	"github.com/gin-gonic/gin"
)

// setupInsightsRoutes mounts the daily ops digest. Admin-only: the wildcard
// admin grant covers it; explicit deny rows keep every other role out.
func setupInsightsRoutes(protected *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.Insights == nil {
		return
	}
	g := protected.Group("/insights")
	{
		g.GET("/digest", container.Handlers.Insights.GetDigest)
		g.POST("/:id/dismiss", container.Handlers.Insights.Dismiss)
	}
}
