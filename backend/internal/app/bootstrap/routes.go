package bootstrap

import (
	"net/http"

	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/middleware"

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

func setupHealthRoutes(router *gin.Engine, container *Container) {
	api := router.Group("/api")
	{
		api.GET("/healthz", container.Handlers.Health.HealthCheck)
		api.GET("/ready", container.Handlers.Health.ReadinessCheck)
	}
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
	admin.GET("/:id", container.Handlers.AdminAttendance.Get)
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

// setupDisbursementWebhookRoutes registers per-provider IPN endpoints.
// Routes are unauthenticated — providers post here without JWTs and
// authenticity is enforced via the provider's signature verification.
//
// Each provider gets a static path (e.g. /webhooks/disbursement/9pay)
// rather than a generic /:provider catch-all. This is intentional: it lets
// us attach per-provider middleware (e.g. IP whitelist for OnePay) and
// keeps the surface area bounded — adding a new provider requires editing
// this function, not silently adding a new route via the registry.
//
// OnePay publishes a stable set of egress IPs we can filter on, so it
// gets the IP-whitelist middleware. 9pay does not, so the 9pay route is
// gated only by signature verification; add a symmetric AllowedIPs field
// on NinepayConfig if/when that changes.
func setupDisbursementWebhookRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.DisbursementWebhook == nil {
		return
	}
	cfg := container.Config
	if cfg == nil {
		return
	}
	hasProvider := cfg.Disbursement.Ninepay.Enabled || cfg.Disbursement.Onepay.Enabled
	if !hasProvider {
		return
	}

	webhooks := v1.Group("/webhooks/disbursement")

	if cfg.Disbursement.Ninepay.Enabled {
		webhooks.PUT("/9pay", container.Handlers.DisbursementWebhook.ReceiveFrom("9pay"))
	}

	if cfg.Disbursement.Onepay.Enabled {
		var ipOpts []middleware.Option
		if cfg.Disbursement.Onepay.AllowedIPsUseRemoteAddr {
			ipOpts = append(ipOpts, middleware.UseRemoteAddr())
		}
		webhooks.PUT(
			"/1pay",
			middleware.IPWhitelist(cfg.Disbursement.Onepay.AllowedIPs, ipOpts...),
			container.Handlers.DisbursementWebhook.ReceiveFrom("1pay"),
		)
		observability.GetLogger().Info("OnePay webhook IP whitelist enabled",
			"allowed_ips", len(cfg.Disbursement.Onepay.AllowedIPs),
			"use_remote_addr", cfg.Disbursement.Onepay.AllowedIPsUseRemoteAddr,
		)
	}
}

// setupManualDisbursementRoutes mounts the admin "Chuyển tiền" CRUD
// endpoints. All routes are always mounted. The handlers resolve the
// active provider from the registry and return a clear error when no
// provider is available.
func setupManualDisbursementRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.ManualDisbursement == nil {
		return
	}
	md := v1.Group("/admin/manual-disbursement")
	md.Use(container.Middleware.Auth.Authenticate())
	if container.Middleware.Authorization != nil {
		md.Use(container.Middleware.Authorization.Authorize())
	}
	{
		md.GET("/banks", container.Handlers.ManualDisbursement.Banks)
		md.POST("", container.Handlers.ManualDisbursement.Initiate)
		md.POST("/check-account", container.Handlers.ManualDisbursement.CheckAccount)
		md.GET("/balance", container.Handlers.Wallet.GetBalance)
		if container.Handlers.ReconciliationExport != nil {
			md.GET("/reconciliation/download", container.Handlers.ReconciliationExport.Download)
		}

		md.GET("", container.Handlers.ManualDisbursement.List)
		md.GET("/:txn_id", container.Handlers.ManualDisbursement.Status)
	}
}

func setupDisbursementSettingsRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.DisbursementSettings == nil {
		return
	}
	s := v1.Group("/admin/settings/disbursement")
	s.Use(container.Middleware.Auth.Authenticate())
	if container.Middleware.Authorization != nil {
		s.Use(container.Middleware.Authorization.Authorize())
	}
	s.GET("", container.Handlers.DisbursementSettings.Get)
}

func setupCacheRoutes(v1 *gin.RouterGroup, container *Container) {
	cache := v1.Group("/cache")
	cache.Use(container.Middleware.Auth.Authenticate())
	{
		cache.DELETE("", container.Handlers.Cache.DeleteAllCache)
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

func setupAuthRoutes(v1 *gin.RouterGroup, container *Container) {
	auth := v1.Group("/auth")
	{
		auth.POST("/login", container.Middleware.LoginRateLimit, container.Handlers.Auth.Login)
		auth.POST("/google", container.Handlers.Auth.GoogleLogin)
		auth.POST("/logout", container.Middleware.Auth.Authenticate(), container.Handlers.Auth.Logout)

		auth.GET("/me", container.Middleware.Auth.Authenticate(), container.Handlers.Auth.GetProfile)
		auth.PUT("/me", container.Middleware.Auth.Authenticate(), container.Handlers.Auth.UpdateProfile)
		auth.POST("/change-password", container.Middleware.Auth.Authenticate(), container.Handlers.Auth.ChangePassword)
	}
}

func setupUserRoutes(v1 *gin.RouterGroup, container *Container) {
	users := v1.Group("/users")
	users.Use(container.Middleware.Auth.Authenticate())
	users.Use(container.Middleware.Authorization.Authorize())
	{
		users.POST("", container.Handlers.User.CreateUser)
		users.GET("", container.Handlers.User.ListUsers)
		users.GET("/summary", container.Handlers.User.GetUserSummary)
		users.GET("/:id", container.Handlers.User.GetUser)
		users.GET("/:id/activities", container.Handlers.User.GetUserActivities)
		users.PUT("/:id", container.Handlers.User.UpdateUser)
		users.DELETE("/:id", container.Handlers.User.DeleteUser)
		users.POST("/:id/reset-password", container.Handlers.User.ResetUserPassword)
		users.POST("/reset-first-time-login-password", container.Handlers.User.ResetFirstTimeLoginPasswords)
		users.GET("/reset-first-time-login-password", container.Handlers.User.GetPasswordResetJobStatus)
	}
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
	setupDBExportRoutes(protected, container)
	setupCronRoutes(protected, container)
	setupAdvPartnerRoutes(protected, container)
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
		attendance.GET("/history", container.Handlers.Attendance.List)
	}
}

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
	}
}

func setupProjectRoutes(protected *gin.RouterGroup, container *Container) {
	projects := protected.Group("/projects")
	{
		projects.POST("", container.Handlers.Project.CreateProject)
		projects.GET("", container.Handlers.Project.ListProjects)
		projects.GET("/summary", container.Handlers.Project.GetProjectSummary)
		projects.GET("/partner-summary", container.Handlers.Project.GetPartnerProjectSummary)
		projects.POST("/activate", container.Handlers.Project.ActivateProjects) // Admin-only endpoint for manual project activation
		projects.GET("/:id", container.Handlers.Project.GetProject)
		projects.PUT("/:id", container.Handlers.Project.UpdateProject)
		projects.DELETE("/:id", container.Handlers.Project.DeleteProject)

		projects.POST("/:id/employees", container.Handlers.ProjectEmployee.AssignEmployee)
		projects.PUT("/:id/employees", container.Handlers.ProjectEmployee.UpdateProjectEmployee)
		projects.POST("/:id/employees/remove", container.Handlers.ProjectEmployee.RemoveEmployeesFromProject)
		projects.GET("/:id/employees", container.Handlers.ProjectEmployee.ListProjectEmployees)
		projects.PATCH("/:id/employees/:employeeId/checkin-enabled", container.Handlers.ProjectEmployee.ToggleCheckInEnabled)
		projects.PATCH("/:id/employees/checkin-enabled/bulk", container.Handlers.ProjectEmployee.BulkToggleCheckInEnabled)

		projects.GET("/:id/payrate", container.Handlers.Project.GetCurrentProjectPayrate)
		projects.POST("/:id/payrate", container.Handlers.Project.CreateProjectPayrate)
		projects.GET("/:id/timesheets", container.Handlers.Project.ListProjectTimesheets)
		projects.GET("/:id/timesheet-entry-table", container.Handlers.Project.GetTimesheetEntryTable)
		projects.POST("/:id/timesheets/approve", container.Handlers.Timesheet.BulkApproveByProject)

		// Project sharing endpoints
		projects.GET("/:id/users", container.Handlers.Project.ListProjectUsers)
		projects.POST("/:id/users", container.Handlers.Project.GrantProjectAccess)
		projects.DELETE("/:id/users/:userId", container.Handlers.Project.RevokeProjectAccess)
	}
}

func setupEmployeeRoutes(protected *gin.RouterGroup, container *Container) {
	employees := protected.Group("/employees")
	{
		// Admin-only: Initialize user accounts for employees
		employees.POST("/init-users", container.Handlers.EmployeeProfile.InitializeEmployeeUsers)

		// Employee import endpoints
		employees.POST("/import", container.Handlers.EmployeeImport.PostImportEmployees)
		employees.GET("/import/:id/status", container.Handlers.EmployeeImport.GetImportStatus)

		employees.POST("", container.Handlers.Employee.CreateEmployee)
		employees.GET("", container.Handlers.Employee.ListEmployees)
		employees.GET("/export", container.Handlers.Employee.ExportEmployees)
		employees.GET("/summary", container.Handlers.Employee.GetEmployeesSummary)
		employees.GET("/unassigned", container.Handlers.Employee.GetUnassignedEmployees)
		employees.GET("/missing-bank-details", container.Handlers.Employee.GetEmployeesMissingBankDetails)
		employees.GET("/cccd/:cccd", container.Handlers.Employee.GetEmployeeByCCCD)
		employees.GET("/:id", container.Handlers.Employee.GetEmployee)
		employees.GET("/:id/export", container.Handlers.Employee.ExportEmployeeDetail)
		employees.PUT("/:id", container.Handlers.Employee.UpdateEmployee)
		employees.DELETE("/:id", container.Handlers.Employee.DeleteEmployee)
		employees.PUT("/:id/change-password", container.Handlers.Employee.ChangeEmployeePassword)

		employees.PUT("/:id/projects", container.Handlers.Employee.UpdateEmployeeProject)

		employees.GET("/:id/users", container.Handlers.EmployeeUsers.ListEmployeeUsers)
		employees.POST("/:id/users", container.Handlers.EmployeeUsers.GrantEmployeeAccess)
		employees.DELETE("/:id/users/:userId", container.Handlers.EmployeeUsers.RevokeEmployeeAccess)

		employees.GET("/:id/summary", container.Handlers.Employee.GetEmployeeSummary)
		employees.GET("/:id/payroll", container.Handlers.Employee.GetEmployeePayroll)
		employees.GET("/:id/timesheet", container.Handlers.Employee.GetEmployeeTimesheet)
		employees.GET("/:id/timesheets/summary", container.Handlers.Employee.GetEmployeeTimesheetSummary)
		employees.GET("/:id/current-projects", container.Handlers.Employee.GetEmployeeCurrentProjects)
	}
}

func setupProjectEmployeeRoutes(protected *gin.RouterGroup, container *Container) {
	projectEmployees := protected.Group("/project-employees")
	{
		// Payment schedule management
		projectEmployees.POST("/:id/payment-schedule", container.Handlers.ProjectEmployee.RequestPaymentScheduleChange)
		projectEmployees.DELETE("/:id/payment-schedule", container.Handlers.ProjectEmployee.CancelPaymentScheduleChange)
		projectEmployees.GET("/pending-schedule-changes", container.Handlers.ProjectEmployee.GetPendingScheduleChanges)
	}
}

func setupBankRoutes(protected *gin.RouterGroup, container *Container) {
	banks := protected.Group("/banks")
	{
		banks.POST("", container.Handlers.Bank.CreateBank)
		banks.GET("", container.Handlers.Bank.ListBanks)
		banks.GET("/:id", container.Handlers.Bank.GetBank)
		banks.PUT("/:id", container.Handlers.Bank.UpdateBank)
		banks.DELETE("/:id", container.Handlers.Bank.DeleteBank)
	}
}

func setupTimesheetRoutes(protected *gin.RouterGroup, container *Container) {
	timesheets := protected.Group("/timesheets")
	{
		timesheets.GET("/summary", container.Handlers.Timesheet.GetSummary)
		timesheets.POST("", container.Handlers.Timesheet.BulkCreateTimesheets)
		timesheets.POST("/preview", container.Handlers.Timesheet.PreviewTimesheets)
		timesheets.GET("", container.Handlers.Timesheet.ListTimesheets)
		timesheets.GET("/grouped", container.Handlers.Timesheet.ListGroupedTimesheets)
		timesheets.GET("/export", container.Handlers.Timesheet.ExportTimesheets)
		timesheets.POST("/export-entries-template", container.Handlers.Timesheet.ExportTimesheetTemplate)
		timesheets.POST("/upload-entries-excel", container.Handlers.Timesheet.UploadTimesheetEntries)

		timesheets.POST("/payroll/report/send-email", container.Handlers.Email.SendPayrollReportEmail)
		timesheets.GET("/payroll/report", container.Handlers.Timesheet.PayrollReportExport)
		timesheets.POST("/payroll/upload-settlement-result", container.Handlers.Timesheet.UploadSettlementResult)
		timesheets.GET("/projects/:id", container.Handlers.Timesheet.GetTimesheetsByProjectAndDate)
		timesheets.POST("/bulk-approve", container.Handlers.Timesheet.BulkApprove)
		timesheets.POST("/bulk-reject", container.Handlers.Timesheet.BulkReject)
		timesheets.POST("/bulk-reset", container.Handlers.Timesheet.BulkReset)
		timesheets.POST("/approve-all", container.Handlers.Timesheet.ApproveAllTimesheets)

		// Edit request routes
		timesheets.GET("/edit-requests", container.Handlers.TimesheetEditRequest.ListEditRequests)
		timesheets.GET("/edit-requests/:id", container.Handlers.TimesheetEditRequest.GetEditRequest)
		timesheets.PUT("/edit-requests/:id/approve", container.Handlers.TimesheetEditRequest.ApproveEditRequest)
		timesheets.PUT("/edit-requests/:id/reject", container.Handlers.TimesheetEditRequest.RejectEditRequest)

		// Partner BCC import endpoints
		partnerImport := timesheets.Group("/partner-import")
		{
			partnerImport.POST("", container.Handlers.BCCImport.UploadBCC)
			partnerImport.GET("", container.Handlers.BCCImport.ListPartnerImports)
			partnerImport.GET("/:id", container.Handlers.BCCImport.GetPartnerImport)
			partnerImport.GET("/:id/download", container.Handlers.BCCImport.DownloadPartnerImport)
		}

		timesheets.GET("/:id", container.Handlers.Timesheet.GetTimesheet)
		timesheets.PUT("/:id", container.Handlers.Timesheet.UpdateTimesheet)
		timesheets.DELETE("/:id", container.Handlers.Timesheet.DeleteTimesheet)
		timesheets.PUT("/:id/approve", container.Handlers.Timesheet.ApproveTimesheet)
		timesheets.PUT("/:id/reject", container.Handlers.Timesheet.RejectTimesheet)
		timesheets.POST("/:id/add-to-payroll", container.Handlers.Timesheet.AddToPayroll)
		timesheets.POST("/:id/request-edit", container.Handlers.TimesheetEditRequest.CreateEditRequest)
		timesheets.POST("/:id/request-edit-cancel", container.Handlers.TimesheetEditRequest.CancelEditRequest)
	}
}

func setupPayrollRoutes(protected *gin.RouterGroup, container *Container) {
	payrolls := protected.Group("/payrolls")
	{
		payrolls.GET("/bulk-transfer-template", container.Handlers.Payroll.GetPayrollTemplate)
		payrolls.POST("/export-bulk-transfer", container.Handlers.Payroll.ExportBulkTransfer)
		payrolls.POST("/bulk-transfer-external-mark", container.Handlers.Payroll.MarkExternallyPaid)
		payrolls.GET("/auto-bulk-transfer/config", container.Handlers.Payroll.GetAutoBulkTransferConfig)
		payrolls.POST("/auto-bulk-transfer/estimate-fee", container.Handlers.Payroll.EstimateFee)
		payrolls.POST("/auto-bulk-transfer", container.Handlers.Payroll.InitiateAutoBulkTransfer)
		payrolls.GET("/auto-bulk-transfer/:batch_id/status", container.Handlers.Payroll.GetAutoBulkTransferStatus)
		payrolls.POST("/bulk-transfer-result", container.Handlers.Payroll.ImportBulkTransferResult)
		payrolls.GET("/pending-uploads", container.Handlers.Payroll.GetPendingUploads)
		payrolls.GET("/bulk-transfer-upload-histories", container.Handlers.Payroll.GetBulkTransferUploadHistories)
		payrolls.GET("/bulk-transfer-upload-histories/:id", container.Handlers.Payroll.GetBulkTransferUploadHistoryByID)
		payrolls.GET("/bulk-transfer-upload-histories/:id/download", container.Handlers.Payroll.DownloadBulkTransferHistoryFile)
		payrolls.GET("/histories", container.Handlers.Payroll.GetPayrollHistories)
		payrolls.POST("/histories/export", container.Handlers.Payroll.ExportPayrollHistories)
	}
}

func setupPayrateRoutes(protected *gin.RouterGroup, container *Container) {
	payrates := protected.Group("/payrates")
	{
		payrates.POST("", container.Handlers.Payrate.CreatePayrate)
		payrates.POST("/validate", container.Handlers.Payrate.ValidatePayrate)
		payrates.GET("", container.Handlers.Payrate.ListPayrates)
		payrates.GET("/:id", container.Handlers.Payrate.GetPayrate)
		payrates.PUT("/:id", container.Handlers.Payrate.UpdatePayrate)
		payrates.DELETE("/:id", container.Handlers.Payrate.DeletePayrate)
		payrates.POST("/:id/validate", container.Handlers.Payrate.ValidatePayrate)
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
	}
}

func setupEmailRoutes(protected *gin.RouterGroup, container *Container) {
	emails := protected.Group("/email")
	{
		emails.GET("/history", container.Handlers.Email.GetEmailHistory)
		emails.POST("/history/:id/settle", container.Handlers.Settlement.SettleFromNotification)
		emails.POST("/history/:id/upload-settlement", container.Handlers.Settlement.UploadSettlement)
		emails.POST("/send", container.Handlers.Email.SendGenericEmail)
	}
}

func setupDBRoutes(protected *gin.RouterGroup, container *Container) {
	// Removed migration routes as they are no longer needed
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

func setupAdvancePaymentRoutes(protected *gin.RouterGroup, container *Container) {
	// Employee endpoints - for their own advance payments
	me := protected.Group("/me/advance-payment")
	{
		me.GET("", container.Handlers.AdvancePayment.GetMyAdvancePaymentInfo)
		me.POST("/request", container.Handlers.AdvancePayment.CreateAdvancePaymentRequest)
		me.POST("/request/:id/cancel", container.Handlers.AdvancePayment.CancelMyAdvancePaymentRequest)
		me.GET("/history", container.Handlers.AdvancePayment.GetMyAdvancePaymentHistory)
		me.POST("/calculate-fee", container.Handlers.AdvancePayment.CalculateFeePreview)
	}

	// Admin endpoints - for managing advance payments
	advancePayments := protected.Group("/advance-payments")
	{
		advancePayments.GET("", container.Handlers.AdvancePayment.ListAdvancePayments)
		advancePayments.GET("/available-months", container.Handlers.AdvancePayment.GetAvailableMonths)
		advancePayments.GET("/employees", container.Handlers.AdvancePayment.GetEmployees)
		advancePayments.GET("/employees/export", container.Handlers.AdvancePayment.ExportFlexPayEmployees)
		advancePayments.GET("/summary", container.Handlers.AdvancePayment.GetAdvancePaymentSummary)
		advancePayments.GET("/export", container.Handlers.AdvancePayment.ExportAdvancePayments)
		advancePayments.POST("/:id/cancel", container.Handlers.AdvancePayment.CancelAdvancePaymentRequest)
		advancePayments.POST("/upload-result", container.Handlers.AdvancePayment.UploadAdvancePaymentResult)
		advancePayments.GET("/transfer-histories/:id/download", container.Handlers.AdvancePayment.DownloadTransferHistoryFile)
		advancePayments.POST("/import", container.Handlers.AdvancePayment.ImportFlexPayFile)
		advancePayments.GET("/import/:id", container.Handlers.AdvancePayment.GetImportJobStatus)
		advancePayments.GET("/files", container.Handlers.AdvancePayment.ListUploadedFiles)
		advancePayments.GET("/files/:id/download", container.Handlers.AdvancePayment.DownloadUploadedFile)
		advancePayments.GET("/reconciliation/export", container.Handlers.AdvancePayment.ExportReconciliation)
		advancePayments.POST("/reconciliation/send-email", container.Handlers.AdvancePayment.SendReconciliationEmail)
		advancePayments.POST("/reconciliation/settle", container.Handlers.AdvancePayment.UploadReconciliationSettlement)
		advancePayments.POST("/import-employee-list", container.Handlers.AdvancePayment.ImportFlexibleEmployeeList)
	}

	// Admin fee schedule CRUD. Casbin policy already grants admin role wildcard
	// /api/* access; the explicit Authorize() middleware prevents partner role
	// from reaching these endpoints (no allow rule exists for them).
	feeSchedule := protected.Group("/admin/advance-payment-fees")
	feeSchedule.Use(container.Middleware.Authorization.Authorize())
	{
		feeSchedule.GET("", container.Handlers.AdvancePaymentFee.List)
		feeSchedule.POST("", container.Handlers.AdvancePaymentFee.Create)
		feeSchedule.PATCH("/:id", container.Handlers.AdvancePaymentFee.Update)
		feeSchedule.DELETE("/:id", container.Handlers.AdvancePaymentFee.Delete)
	}

	// Admin disbursement-fee schedule CRUD. Same admin gating as the
	// advance-payment fee endpoints above.
	disbursementFee := protected.Group("/admin/disbursement-fees")
	disbursementFee.Use(container.Middleware.Authorization.Authorize())
	{
		disbursementFee.GET("", container.Handlers.DisbursementFee.List)
		disbursementFee.POST("", container.Handlers.DisbursementFee.Create)
		disbursementFee.PATCH("/:id", container.Handlers.DisbursementFee.Update)
		disbursementFee.DELETE("/:id", container.Handlers.DisbursementFee.Delete)
	}
}

func setupTestRoutes(v1 *gin.RouterGroup) {
	v1.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
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

func setupPushRoutes(v1 *gin.RouterGroup, protected *gin.RouterGroup, container *Container) {
	// Public: VAPID key is needed before subscription and is not sensitive
	v1.GET("/push/vapid-key", container.Handlers.Push.GetVAPIDKey)

	push := protected.Group("/push")
	{
		push.POST("/subscribe", container.Handlers.Push.Subscribe)
		push.POST("/unsubscribe", container.Handlers.Push.Unsubscribe)
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

func setupAuditRoutes(protected *gin.RouterGroup, container *Container) {
	audit := protected.Group("/audit")
	{
		audit.GET("/logs", container.Handlers.Audit.ListAuditLogs)
		audit.GET("/logs/:id", container.Handlers.Audit.GetAuditLog)
	}
}

func setupWalletRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.Wallet == nil {
		return
	}
	w := v1.Group("/wallet")
	w.Use(container.Middleware.Auth.Authenticate())
	if container.Middleware.Authorization != nil {
		w.Use(container.Middleware.Authorization.Authorize())
	}
	{
		w.GET("/balance", container.Handlers.Wallet.GetBalance)
		w.POST("/balance/sync", container.Handlers.Wallet.SyncBalance)
		w.POST("/balance/adjust", container.Handlers.Wallet.AdjustBalance)
		w.GET("/transactions", container.Handlers.Wallet.GetTransactions)

		w.GET("/topups", container.Handlers.Wallet.ListTopups)
		w.POST("/topups", container.Handlers.Wallet.CreateTopup)
		w.GET("/topups/:id", container.Handlers.Wallet.GetTopupByID)

		w.GET("/payments", container.Handlers.Wallet.ListPayments)
		w.GET("/payments/:id", container.Handlers.Wallet.GetPaymentByID)
		w.POST("/payments/:id/resolve", container.Handlers.Wallet.ResolvePayment)

		w.POST("/reconcile/upload", container.Handlers.Wallet.UploadReconciliation)
		w.POST("/reconcile/auto", container.Handlers.Wallet.AutoReconcile)
		w.GET("/reconcile/jobs/:id", container.Handlers.Wallet.GetReconciliationJobStatus)
		w.GET("/reconcile/export", container.Handlers.Wallet.ExportReconciliationReport)
	}
}

func setupAdvPartnerRoutes(protected *gin.RouterGroup, container *Container) {
	advPartner := protected.Group("/adv-partner")
	{
		advPartner.PUT("/users/:id", container.Handlers.AdvPartnerUser.UpdateAdvPartnerUser)
	}
}
