package bootstrap

import "github.com/gin-gonic/gin"

// setupAPIKeyAdminRoutes mounts the admin-only API key management endpoints.
// Casbin grants `admin, /api/*, *, allow`, and no other role holds a rule for
// /api/v1/admin/*, so the generic Authorize() middleware is sufficient.
func setupAPIKeyAdminRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.APIKey == nil {
		return
	}
	g := v1.Group("/admin/api-keys")
	g.Use(container.Middleware.Auth.Authenticate())
	g.Use(container.Middleware.Authorization.Authorize())
	g.POST("", container.Handlers.APIKey.Create)
	g.GET("", container.Handlers.APIKey.List)
	g.DELETE("/:id", container.Handlers.APIKey.Revoke)
}

// setupIntegrationResetRoutes mounts the API-key-authenticated chatbot
// endpoints. These are mounted on v1 (inheriting APIRateLimit + APIMetrics) but
// deliberately NOT on the protected group — no JWT, no Casbin.
func setupIntegrationResetRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil {
		return
	}
	g := v1.Group("/integration")
	g.Use(container.Middleware.APIKeyAuth.Authenticate())
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
	}
	if container.Handlers.IntegrationZaloToken != nil {
		g.GET("/zalo/token", container.Handlers.IntegrationZaloToken.GetToken)
	}
}
