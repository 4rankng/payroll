package bootstrap

import "github.com/gin-gonic/gin"

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
