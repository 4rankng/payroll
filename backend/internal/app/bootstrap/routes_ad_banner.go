package bootstrap

import "github.com/gin-gonic/gin"

func setupAdBannerRoutes(protected *gin.RouterGroup, container *Container) {
	// Employee self-service: resolve the single campaign addressing the
	// authenticated employee and record CTA taps. GET is already covered by
	// the employee /api/v1/me/* GET Casbin wildcard; the click POST needs its
	// own policy line.
	me := protected.Group("/me/ad-banner")
	{
		me.GET("", container.Handlers.AdBanner.GetMyAdBanner)
		me.POST("/:id/click", container.Middleware.AdBannerClickRateLimit, container.Handlers.AdBanner.RecordClick)
	}

	// Admin campaign CRUD. Casbin grants admin the /api/* wildcard; the
	// explicit Authorize() keeps partner out (no allow rule exists for them),
	// same pattern as the fee-schedule groups.
	ads := protected.Group("/ad-banners")
	ads.Use(container.Middleware.Authorization.Authorize())
	{
		ads.GET("", container.Handlers.AdBanner.List)
		ads.POST("", container.Handlers.AdBanner.Create)
		ads.PUT("/:id", container.Handlers.AdBanner.Update)
		ads.DELETE("/:id", container.Handlers.AdBanner.Delete)
	}
}
