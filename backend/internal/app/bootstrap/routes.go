package bootstrap

import (
	"api-server/internal/infra/observability"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, container *Container) {
	// Expose metrics endpoint for Prometheus; requires admin authentication and authorization
	if container != nil {
		// Prometheus metrics endpoint with authentication and authorization
		protectedMetrics := router.Group("/")
		protectedMetrics.Use(container.Middleware.Auth.Authenticate())
		protectedMetrics.Use(container.Middleware.Authorization.Authorize())
		protectedMetrics.GET("/metrics", func(c *gin.Context) {
			handler := observability.MetricsHandler()
			handler.ServeHTTP(c.Writer, c.Request)
		})
	}
	setupLegacyCacheRoutes(router, container)
	setupHealthRoutes(router, container)
	setupAPIRoutes(router, container)
}

func setupAPIRoutes(router *gin.Engine, container *Container) {
	api := router.Group("/api")
	v1 := api.Group("/v1")

	v1.Use(container.Middleware.APIRateLimit)
	v1.Use(container.Middleware.APIMetrics)

	setupAuthRoutes(v1, container)
	setupUserRoutes(v1, container)
	setupProtectedRoutes(v1, container)
	setupMobileRoutes(v1, container)
	setupCacheRoutes(v1, container)
	setupDisbursementWebhookRoutes(v1, container)
	setupManualDisbursementRoutes(v1, container)
	setupDisbursementSettingsRoutes(v1, container)
	setupProviderTransactionsAdminRoutes(v1, container)
	setupAdminAttendanceRoutes(v1, container)
	setupClockRoutes(v1, container)
	setupWalletRoutes(v1, container)
	setupTestRoutes(v1)
}

func setupProtectedRoutes(v1 *gin.RouterGroup, container *Container) {
	protected := v1.Group("/")
	protected.Use(container.Middleware.Auth.Authenticate())
	protected.Use(container.Middleware.Authorization.Authorize())

	// Employee self-service endpoints
	protected.GET("/me", container.Handlers.EmployeeProfile.GetMyProfile)
	protected.PUT("/me", container.Handlers.EmployeeProfile.UpdateMyProfile)
	protected.PUT("/me/password", container.Handlers.EmployeeProfile.UpdateMyPassword)
	protected.GET("/me/timesheet", container.Handlers.EmployeeProfile.GetMyTimesheets)
	protected.GET("/me/summary", container.Handlers.EmployeeProfile.GetMySummary)

	setupDashboardRoutes(protected, container)
	setupProjectRoutes(protected, container)
	setupEmployeeRoutes(protected, container)
	setupProjectEmployeeRoutes(protected, container)
	setupBankRoutes(protected, container)
	setupTimesheetRoutes(protected, container)
	setupPayrollRoutes(protected, container)
	setupPayrateRoutes(protected, container)
	setupLedgerRoutes(protected, container)
	setupTransactionRoutes(protected, container)
	setupNotificationRoutes(protected, container)
	setupPushRoutes(v1, protected, container)
	setupAssetRoutes(protected, container)
	setupLoanRoutes(protected, container)
	setupEmailRoutes(protected, container)
	setupDBRoutes(protected, container)
	setupMetricsRoutes(protected, container)
	setupAdvancePaymentRoutes(protected, container)
	setupAuditRoutes(protected, container)
	setupSettingsRoutes(v1, container) // Admin-only routes, use v1 directly
	setupZaloAdminRoutes(v1, container)
	setupDBExportRoutes(protected, container)
	setupCronRoutes(protected, container)
	setupAdvPartnerRoutes(protected, container)
}
