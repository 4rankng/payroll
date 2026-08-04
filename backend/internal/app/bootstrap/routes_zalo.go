package bootstrap

import "github.com/gin-gonic/gin"

// setupZaloAdminRoutes mounts the admin-only Zalo connection management
// endpoints. All routes require authentication; write actions additionally
// require admin authorization. The OAuth callback is admin-authenticated too —
// it completes in the admin's browser, so the session carries through.
func setupZaloAdminRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.AdminZalo == nil {
		return
	}
	g := v1.Group("/admin/zalo")
	g.Use(container.Middleware.Auth.Authenticate())
	{
		// GET status — any authenticated user (the UI is admin-gated client-side,
		// but the data is non-sensitive: masked connection status only).
		g.GET("", container.Handlers.AdminZalo.GetStatus)

		// Write actions — admin only.
		adminOnly := g.Group("")
		adminOnly.Use(container.Middleware.Authorization.Authorize())
		adminOnly.PUT("/credentials", container.Handlers.AdminZalo.SaveCredentials)
		adminOnly.POST("/oauth/start", container.Handlers.AdminZalo.StartOAuth)
		adminOnly.POST("/oauth/callback", container.Handlers.AdminZalo.OAuthCallback)
		adminOnly.PUT("/enabled", container.Handlers.AdminZalo.SetEnabled)
		adminOnly.POST("/refresh", container.Handlers.AdminZalo.RefreshNow)
		adminOnly.POST("/test", container.Handlers.AdminZalo.TestSend)
	}
}
