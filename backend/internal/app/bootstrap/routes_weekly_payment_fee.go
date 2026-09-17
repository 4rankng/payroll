package bootstrap

import "github.com/gin-gonic/gin"

// setupWeeklyPaymentFeeRoutes registers admin CRUD on the weekly-payment fee
// schedule (phí trả lương tuần). Casbin policy already grants admin role
// wildcard /api/* access; the explicit Authorize() middleware prevents
// partner role from reaching these endpoints (no allow rule exists for them).
func setupWeeklyPaymentFeeRoutes(protected *gin.RouterGroup, container *Container) {
	feeSchedule := protected.Group("/admin/weekly-payment-fees")
	feeSchedule.Use(container.Middleware.Authorization.Authorize())
	{
		feeSchedule.GET("", container.Handlers.WeeklyPaymentFee.List)
		feeSchedule.POST("", container.Handlers.WeeklyPaymentFee.Create)
		feeSchedule.PATCH("/:id", container.Handlers.WeeklyPaymentFee.Update)
		feeSchedule.DELETE("/:id", container.Handlers.WeeklyPaymentFee.Delete)
	}
}
