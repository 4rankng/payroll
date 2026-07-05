package bootstrap

import (
	"api-server/internal/config"
	"api-server/internal/transport/http/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupMiddleware(router *gin.Engine, container *Container) {
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Security headers middleware - apply first to ensure all responses have security headers
	router.Use(middleware.SecurityHeaders())

	// Request timeout middleware
	if container != nil && container.Config != nil {
		router.Use(middleware.RequestTimeout(container.Config.RequestTimeout))
	}

	// Add audit context middleware to capture IP and User-Agent for audit logging
	router.Use(middleware.AuditContext())
	// Tenant concurrency semaphore to provide per-tenant fairness
	if container != nil && container.Middleware != nil && container.Middleware.TenantSemaphore != nil {
		router.Use(container.Middleware.TenantSemaphore.Handler())
	}

	setupCORS(router, container.Config)
}

func setupCORS(router *gin.Engine, cfg *config.Config) {
	corsConfig := cors.DefaultConfig()
	if cfg != nil && len(cfg.CORS.AllowOrigins) > 0 {
		corsConfig.AllowOrigins = cfg.CORS.AllowOrigins
	} else {
		corsConfig.AllowOrigins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"https://tingting.vip",
			"https://www.tingting.vip",
		}
	}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Requested-With"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	corsConfig.ExposeHeaders = []string{"Content-Disposition"}
	router.Use(cors.New(corsConfig))
}
