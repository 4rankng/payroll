package bootstrap

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func setupHealthRoutes(router *gin.Engine, container *Container) {
	api := router.Group("/api")
	{
		api.GET("/healthz", container.Handlers.Health.HealthCheck)
		api.GET("/ready", container.Handlers.Health.ReadinessCheck)
	}
}

func setupLegacyCacheRoutes(router *gin.Engine, container *Container) {
	if container == nil {
		return
	}
	if container.Handlers == nil || container.Handlers.Cache == nil {
		return
	}

	cache := router.Group("/v1/cache")
	if container.Middleware != nil && container.Middleware.Auth != nil {
		cache.Use(container.Middleware.Auth.Authenticate())
	}
	cache.DELETE("", container.Handlers.Cache.DeleteAllCache)
}

func setupCacheRoutes(v1 *gin.RouterGroup, container *Container) {
	cache := v1.Group("/cache")
	cache.Use(container.Middleware.Auth.Authenticate())
	{
		cache.DELETE("", container.Handlers.Cache.DeleteAllCache)
	}
}

func setupMobileRoutes(v1 *gin.RouterGroup, container *Container) {
	mobile := v1.Group("/mobile")
	mobile.Use(container.Middleware.Auth.Authenticate())
	if container.Middleware.Authorization != nil {
		mobile.Use(container.Middleware.Authorization.Authorize())
	}

	attendance := mobile.Group("/attendance")
	{
		attendance.GET("/today", container.Handlers.Attendance.GetToday)
		attendance.POST("/check-in", container.Handlers.Attendance.CheckIn)
		attendance.POST("/check-out", container.Handlers.Attendance.CheckOut)
		attendance.POST("/cancel-current", container.Handlers.Attendance.CancelCurrent)
		attendance.POST("/attempt-log", container.Handlers.Attendance.LogDeviceAttempt)
		attendance.GET("/history", container.Handlers.Attendance.List)
	}
}

func setupLedgerRoutes(protected *gin.RouterGroup, container *Container) {
	ledger := protected.Group("/ledger")
	{
		// Entry management
		ledger.POST("/entries", container.Handlers.Ledger.CreateEntries)
		ledger.GET("/entries", container.Handlers.Ledger.ListEntries)
		ledger.GET("/entries/:id", container.Handlers.Ledger.GetEntry)
		ledger.POST("/entries/:id/reverse", container.Handlers.Ledger.ReverseEntry)

		// Balance queries
		ledger.GET("/balance", container.Handlers.Ledger.GetBalance)
		ledger.POST("/balance/recalculate", container.Handlers.Ledger.RecalculateBalances)

		// Cash flow analysis
		ledger.GET("/cash-flow", container.Handlers.Ledger.GetCashFlowSummary)

		// Summary endpoint
		ledger.GET("/summary", container.Handlers.Ledger.GetLedgerSummary)
		// Account metadata endpoint
		ledger.GET("/accounts/metadata", container.Handlers.Ledger.GetAccountsMetadata)
		// Export endpoint
		ledger.GET("/export", container.Handlers.Ledger.ExportEntries)
		// OnePay fee report import
		ledger.POST("/onepay-fee-reports", container.Handlers.Ledger.UploadOnePayFeeReport)
	}
}

func setupTransactionRoutes(protected *gin.RouterGroup, container *Container) {
	transactions := protected.Group("/transactions")
	{
		// Transaction management
		transactions.POST("", container.Handlers.Transaction.CreateTransaction)
		transactions.GET("", container.Handlers.Transaction.ListTransactions)
		transactions.GET("/metadata", container.Handlers.Transaction.GetTransactionMetadata)
		transactions.GET("/export", container.Handlers.Transaction.ExportTransactions)
		transactions.GET("/:id", container.Handlers.Transaction.GetTransaction)
		transactions.PUT("/:id", container.Handlers.Transaction.UpdateTransactionEvidence)
		transactions.POST("/:id/settle", container.Handlers.Transaction.SettleTransaction)
		transactions.POST("/:id/reverse", container.Handlers.Transaction.ReverseTransaction)
	}
}

func setupNotificationRoutes(protected *gin.RouterGroup, container *Container) {
	// Admin-only endpoint for sending custom notifications
	protected.POST("/notification", container.Handlers.Notification.CreateCustomNotification)

	notifications := protected.Group("/notifications")
	{
		notifications.GET("", container.Handlers.Notification.GetNotifications)
		notifications.GET("/unread", container.Handlers.Notification.GetUnreadNotifications)
		notifications.GET("/unread/count", container.Handlers.Notification.GetUnreadCount)
		notifications.PUT("/:id/read", container.Handlers.Notification.MarkAsRead)
		notifications.PUT("/read-all", container.Handlers.Notification.MarkAllAsRead)
		notifications.POST("/no-default-password", container.Handlers.Notification.NotifyDefaultPasswordUsers)
	}
}

func setupPushRoutes(v1 *gin.RouterGroup, protected *gin.RouterGroup, container *Container) {
	// Public: VAPID key is needed before subscription and is not sensitive
	v1.GET("/push/vapid-key", container.Handlers.Push.GetVAPIDKey)

	push := protected.Group("/push")
	{
		push.POST("/subscribe", container.Handlers.Push.Subscribe)
		push.POST("/unsubscribe", container.Handlers.Push.Unsubscribe)
	}
}

func setupAssetRoutes(protected *gin.RouterGroup, container *Container) {
	assets := protected.Group("/assets")
	{
		assets.POST("/upload", container.Handlers.Asset.UploadAsset)
		assets.GET("", container.Handlers.Asset.ListAssets)
		assets.GET("/:id/download", container.Handlers.Asset.DownloadAsset)
		assets.GET("/:id", container.Handlers.Asset.GetAsset)
	}
}

func setupLoanRoutes(protected *gin.RouterGroup, container *Container) {
	// Lenders
	lenders := protected.Group("/lenders")
	{
		lenders.POST("", container.Handlers.Lender.CreateLender)
		lenders.GET("", container.Handlers.Lender.ListLenders)
		lenders.GET("/:id", container.Handlers.Lender.GetLender)
		lenders.PUT("/:id", container.Handlers.Lender.UpdateLender)
		lenders.DELETE("/:id", container.Handlers.Lender.DeleteLender)
	}

	// Loans
	loans := protected.Group("/loans")
	{
		loans.POST("", container.Handlers.Loan.CreateLoan)
		loans.GET("", container.Handlers.Loan.ListLoans)
		loans.GET("/:id", container.Handlers.Loan.GetLoan)
		loans.POST("/:id/disburse", container.Handlers.Loan.DisburseLoan)
		loans.POST("/:id/repay", container.Handlers.Loan.RepayPrincipal)
		loans.GET("/:id/schedule", container.Handlers.Loan.GetSchedule)
		loans.POST("/:id/repay-schedule", container.Handlers.Loan.ProcessScheduledPayment) // New endpoint for scheduled payments
		loans.PATCH("/:id", container.Handlers.Loan.UpdateLoan)
		loans.DELETE("/:id", container.Handlers.Loan.DeleteLoan)
	}
}

func setupEmailRoutes(protected *gin.RouterGroup, container *Container) {
	emails := protected.Group("/email")
	{
		emails.GET("/senders", container.Handlers.Email.GetAvailableSenders)
		emails.GET("/history", container.Handlers.Email.GetEmailHistory)
		emails.POST("/history/:id/settle", container.Handlers.Settlement.SettleFromNotification)
		emails.POST("/history/:id/upload-settlement", container.Handlers.Settlement.UploadSettlement)
		emails.POST("/send", container.Handlers.Email.SendGenericEmail)
	}
}

func setupDBRoutes(protected *gin.RouterGroup, container *Container) {
	// Removed migration routes as they are no longer needed
}

func setupMetricsRoutes(protected *gin.RouterGroup, container *Container) {
	metrics := protected.Group("/metrics")
	{
		metrics.GET("/api", container.Handlers.Metric.GetAPISummary)
		metrics.GET("/event-bus", container.Handlers.Metric.GetEventBusMetrics)
		metrics.GET("/cache", container.Handlers.Metric.GetCacheMetrics)
		metrics.GET("/errors/by-user", container.Handlers.Metric.GetErrorBreakdownByUser)
		metrics.GET("/latency/trend", container.Handlers.Metric.GetLatencyTrend)
		metrics.GET("/latency/slowest", container.Handlers.Metric.GetSlowestEndpoints)
		metrics.GET("/errors/recent", container.Handlers.Metric.GetRecentErrors)
		metrics.GET("/errors/count", container.Handlers.Metric.GetErrorCount)
		metrics.GET("/latency/endpoint-trend", container.Handlers.Metric.GetEndpointLatencyTrend)
		metrics.GET("/top-endpoints", container.Handlers.Metric.GetTopEndpoints)
		metrics.GET("/failed-logins", container.Handlers.Metric.GetFailedLogins)
		metrics.GET("/failed-logins/:identifier", container.Handlers.Metric.GetFailedLoginsByIdentifier)
		metrics.GET("/browser-platform-stats", container.Handlers.Metric.GetBrowserPlatformStats)
		metrics.GET("/browser-platform-stats/users", container.Handlers.Metric.GetBrowserPlatformUsers)
		metrics.GET("/os-stats", container.Handlers.Metric.GetOSStats)
		metrics.GET("/os-stats/users", container.Handlers.Metric.GetOSUsers)
		metrics.GET("/browser-stats", container.Handlers.Metric.GetBrowserStats)
		metrics.GET("/browser-stats/users", container.Handlers.Metric.GetBrowserUsers)
	}
}

func setupAuditRoutes(protected *gin.RouterGroup, container *Container) {
	audit := protected.Group("/audit")
	{
		audit.GET("/logs", container.Handlers.Audit.ListAuditLogs)
		audit.GET("/logs/:id", container.Handlers.Audit.GetAuditLog)
	}
}

func setupAdvPartnerRoutes(protected *gin.RouterGroup, container *Container) {
	advPartner := protected.Group("/adv-partner")
	{
		advPartner.PUT("/users/:id", container.Handlers.AdvPartnerUser.UpdateAdvPartnerUser)
	}
}

func setupTestRoutes(v1 *gin.RouterGroup) {
	v1.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
}
