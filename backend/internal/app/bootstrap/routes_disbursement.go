package bootstrap

import (
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/middleware"

	"github.com/gin-gonic/gin"
)

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
