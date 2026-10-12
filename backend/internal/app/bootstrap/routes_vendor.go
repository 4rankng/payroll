package bootstrap

import (
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
)

// setupVendorTimesheetRoutes mounts the read-only vendor surface. These are
// mounted on v1 (inheriting APIRateLimit + APIMetrics) but deliberately NOT on
// the protected group — no JWT, no Casbin. Every route requires an API key
// whose scope is timesheet_read.
//
// Only GET verbs are registered here. That is the structural half of the
// read-only guarantee: there is no POST/PUT/PATCH/DELETE handler under
// /vendor/* for the router to dispatch to, so no write path can reach this
// surface. The other half is that every existing write route in the system
// lives on the JWT-protected group, which an API key cannot satisfy.
func setupVendorTimesheetRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.VendorTimesheet == nil {
		return
	}
	g := v1.Group("/vendor")
	g.Use(container.Middleware.APIKeyAuth.Authenticate())
	g.Use(container.Middleware.APIKeyAuth.RequireAPIKeyScope(domain.APIKeyScopeTimesheetRead))
	// Reuse the per-API-key integration limiter: it already buckets by
	// CtxAPIKeyID, which is exactly the right granularity for a machine caller.
	g.Use(container.Middleware.IntegrationRateLimit)

	g.GET("/projects", container.Handlers.VendorTimesheet.ListProjects)
	g.GET("/employees", container.Handlers.VendorTimesheet.ListEmployees)
	g.GET("/timesheets", container.Handlers.VendorTimesheet.ListTimesheets)
}
