package bootstrap

import (
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
)

// setupIntegrationResetRoutes mounts the API-key-authenticated vendor
// integration endpoints (TingHire): employee lookup, password reset with OTP,
// and self check-in management. These are mounted on v1 (inheriting
// APIRateLimit + APIMetrics) but deliberately NOT on the protected group — no
// JWT, no Casbin.
//
// The scope gate binds the surface to the vendor key (timesheet_read): the
// vendor integration credential drives both the read-only timesheet API and
// these employee-lifecycle operations. The retired chatbot scope originally
// guarded this surface; the chatbot consumer is gone and the vendor key is
// the remaining caller.
func setupIntegrationResetRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil {
		return
	}
	g := v1.Group("/integration")
	g.Use(container.Middleware.APIKeyAuth.Authenticate())
	g.Use(container.Middleware.APIKeyAuth.RequireAPIKeyScope(domain.APIKeyScopeTimesheetRead))
	g.Use(container.Middleware.IntegrationRateLimit)
	if container.Handlers.IntegrationReset != nil {
		g.POST("/password-reset/otp", container.Handlers.IntegrationReset.RequestOTP)
		g.POST("/password-reset/verify", container.Handlers.IntegrationReset.VerifyOTP)
		g.POST("/password-reset/reset", container.Handlers.IntegrationReset.ResetPassword)
	}
	if container.Handlers.IntegrationLookup != nil {
		g.POST("/employee/lookup", container.Handlers.IntegrationLookup.LookupEmployee)
	}
	if container.Handlers.IntegrationSelfCheckin != nil {
		g.POST("/self-checkin/otp", container.Handlers.IntegrationSelfCheckin.RequestOTP)
		g.POST("/self-checkin/verify", container.Handlers.IntegrationSelfCheckin.VerifyOTP)
		g.POST("/self-checkin/update", container.Handlers.IntegrationSelfCheckin.Update)
		g.POST("/self-checkin/status", container.Handlers.IntegrationSelfCheckin.Status)
	}
}
