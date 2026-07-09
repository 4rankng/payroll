package bootstrap

import "github.com/gin-gonic/gin"

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

	// Employee self-check-in advance endpoints — DEDICATED PATH (separate from the
	// admin-upload /me/advance-payment flow above). 70% advanceable cap, calendar-
	// month salary period, day-10 request window.
	checkInMe := protected.Group("/me/check-in-advance")
	{
		checkInMe.GET("", container.Handlers.AdvancePayment.GetMyCheckInAdvanceInfo)
		checkInMe.POST("/request", container.Handlers.AdvancePayment.CreateCheckInAdvanceRequest)
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
		advancePayments.POST("/:id/retry-disbursement", container.Handlers.AdvancePayment.RetryDisbursement)
		advancePayments.GET("/:id/disbursement-status", container.Handlers.AdvancePayment.GetDisbursementStatus)
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

	// Manual wallet-settlement trigger. Runs the same EOD consolidation task as
	// the daily cron on demand, settling stranded completed wallet payments into
	// ledger records. Admin-only.
	walletSettlement := protected.Group("/admin/wallet-settlement")
	walletSettlement.Use(container.Middleware.Authorization.Authorize())
	{
		walletSettlement.POST("/run", container.Handlers.AdvancePayment.RunWalletSettlement)
	}
}
