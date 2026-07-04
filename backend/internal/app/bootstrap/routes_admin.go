package bootstrap

import "github.com/gin-gonic/gin"

func setupDashboardRoutes(protected *gin.RouterGroup, container *Container) {
	dashboard := protected.Group("/dashboard")
	{
		dashboard.GET("/summary", container.Handlers.Dashboard.GetDashboardSummary)
		dashboard.GET("/financial-overview", container.Handlers.Dashboard.GetFinancialOverview)
		dashboard.GET("/financial", container.Handlers.Dashboard.GetFinancialChartData)
		dashboard.GET("/salary-distribution", container.Handlers.Dashboard.GetSalaryDistribution)
		dashboard.GET("/recent-activities", container.Handlers.Dashboard.GetRecentActivities)
		dashboard.GET("/notifications", container.Handlers.Dashboard.GetSystemNotifications)
		dashboard.GET("/new-employees", container.Handlers.Dashboard.GetNewEmployees)
		dashboard.GET("/historical", container.Handlers.Dashboard.GetHistoricalData)
		dashboard.GET("/monthly-financials", container.Handlers.Dashboard.GetMonthlyFinancials)
		dashboard.GET("/employee-activity", container.Handlers.Dashboard.GetEmployeeActivityStats)
		dashboard.GET("/top-paid-employees", container.Handlers.Dashboard.GetTopPaidEmployees)
		dashboard.GET("/partner", container.Handlers.Dashboard.GetPartnerDashboard)
		dashboard.GET("/partner/employees", container.Handlers.Dashboard.GetPartnerEmployeeList)
		dashboard.GET("/bank-usage", container.Handlers.Dashboard.GetBankUsage)
		dashboard.GET("/bank-usage/projects", container.Handlers.Dashboard.GetBankUsageAllProjects)
		dashboard.GET("/employee-activity/users", container.Handlers.Dashboard.GetActiveEmployeesBySchedule)
		dashboard.GET("/project-profitability", container.Handlers.Dashboard.GetProjectProfitability)
		dashboard.GET("/project-weekly-profit", container.Handlers.Dashboard.GetProjectWeeklyProfit)
		dashboard.GET("/check-in-health", container.Handlers.Dashboard.GetCheckInHealth)
		dashboard.GET("/quota-anomalies", container.Handlers.Dashboard.GetQuotaAnomalies)
	}
}

// setupClockRoutes mounts the admin clock manipulation endpoints.
// Only available in non-production environments. Allows integration tests
// to manipulate the server's internal clock for time-dependent scenarios.
func setupClockRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.AdminClock == nil {
		return
	}
	if container.Config.App.Env == "production" {
		return
	}
	admin := v1.Group("/admin/clock")
	admin.Use(container.Middleware.Auth.Authenticate())
	if container.Middleware.Authorization != nil {
		admin.Use(container.Middleware.Authorization.Authorize())
	}
	admin.GET("", container.Handlers.AdminClock.GetTime)
	admin.POST("/set", container.Handlers.AdminClock.SetTime)
	admin.POST("/advance", container.Handlers.AdminClock.AdvanceTime)
	admin.POST("/reset", container.Handlers.AdminClock.ResetTime)
}

func setupAdminAttendanceRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.AdminAttendance == nil {
		return
	}
	admin := v1.Group("/admin/attendances")
	admin.Use(container.Middleware.Auth.Authenticate())
	if container.Middleware.Authorization != nil {
		admin.Use(container.Middleware.Authorization.Authorize())
	}
	admin.GET("", container.Handlers.AdminAttendance.List)
	admin.GET("/failed-attempts", container.Handlers.AdminAttendance.AdminListFailedAttempts)
	admin.GET("/:id", container.Handlers.AdminAttendance.Get)
	admin.POST("/:id/approve", container.Handlers.AdminAttendance.Approve)
	admin.POST("/:id/reject", container.Handlers.AdminAttendance.Reject)
}

// setupProviderTransactionsAdminRoutes mounts the read-only stats endpoint
// for provider_transactions. Casbin-protects to admin via the standard
// Authorize() middleware. Always mounted — even when 9pay is dormant,
// the table exists and reports an empty (or historical) result set.
func setupProviderTransactionsAdminRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.ProviderTransactions == nil {
		return
	}
	stats := v1.Group("/admin/provider-transactions")
	stats.Use(container.Middleware.Auth.Authenticate())
	if container.Middleware.Authorization != nil {
		stats.Use(container.Middleware.Authorization.Authorize())
	}
	stats.GET("/stats", container.Handlers.ProviderTransactions.Stats)
}

func setupSettingsRoutes(v1 *gin.RouterGroup, container *Container) {
	settings := v1.Group("/settings")
	settings.Use(container.Middleware.Auth.Authenticate())
	{
		// GET endpoints - accessible by all authenticated users
		settings.GET("", container.Handlers.Settings.ListSettings)
		settings.GET("/:id", container.Handlers.Settings.GetSetting)
		settings.GET("/key/:key", container.Handlers.Settings.GetSettingByKey)

		// Write endpoints - admin only
		settings.POST("", container.Middleware.Authorization.Authorize(), container.Handlers.Settings.CreateSetting)
		settings.PUT("/:id", container.Middleware.Authorization.Authorize(), container.Handlers.Settings.UpdateSetting)
		settings.DELETE("/:id", container.Middleware.Authorization.Authorize(), container.Handlers.Settings.DeleteSetting)
	}
}

func setupCronRoutes(protected *gin.RouterGroup, container *Container) {
	cronJobs := protected.Group("/cron-jobs")
	{
		cronJobs.GET("", container.Handlers.Cron.GetCronJobs)
		cronJobs.PUT("/:name/toggle", container.Handlers.Cron.ToggleCronJob)
		cronJobs.POST("/:name/run", container.Handlers.Cron.RunCronJob)
	}
}

func setupDBExportRoutes(protected *gin.RouterGroup, container *Container) {
	dbExport := protected.Group("/db-export")
	{
		dbExport.POST("", container.Handlers.DBExport.StartExport)
		dbExport.GET("", container.Handlers.DBExport.ListExportJobs)
		dbExport.GET("/:jobId/status", container.Handlers.DBExport.GetExportStatus)
		dbExport.GET("/:jobId/download", container.Handlers.DBExport.DownloadExport)
	}
}
