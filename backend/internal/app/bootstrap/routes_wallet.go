package bootstrap

import (
	"github.com/gin-gonic/gin"
)

func setupWalletRoutes(v1 *gin.RouterGroup, container *Container) {
	if container == nil || container.Handlers == nil || container.Handlers.Wallet == nil {
		return
	}
	w := v1.Group("/wallet")
	w.Use(container.Middleware.Auth.Authenticate())
	if container.Middleware.Authorization != nil {
		w.Use(container.Middleware.Authorization.Authorize())
	}
	// Note: the 10MB upload cap is enforced in the handler via
	// http.MaxBytesReader BEFORE c.FormFile reads the body. Gin's default
	// MaxMultipartMemory (32MB) is above our limit, so requests that exceed
	// 10MB are rejected by MaxBytesReader before they spill to disk.
	{
		w.GET("/balance", container.Handlers.Wallet.GetBalance)
		w.GET("/demand-forecast", container.Handlers.Wallet.GetDemandForecast)
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

	// Bulk-transfer sub-group: /wallet/bulk-transfer/*
	// Inherits Authenticate + Authorize middleware from the parent group.
	// Casbin policy in configs/casbin_policy.csv denies partner access.
	if container.Handlers.WalletBulkTransfer != nil {
		bt := w.Group("/bulk-transfer")
		{
			bt.POST("/upload", container.Handlers.WalletBulkTransfer.UploadBulkTransfer)
			bt.GET("/batches", container.Handlers.WalletBulkTransfer.ListBatches)
			bt.GET("/batches/:id", container.Handlers.WalletBulkTransfer.GetBatch)
			bt.GET("/batches/:id/kq", container.Handlers.WalletBulkTransfer.DownloadKQ)
		}
	}
}
