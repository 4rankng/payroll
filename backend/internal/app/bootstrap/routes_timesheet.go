package bootstrap

import "github.com/gin-gonic/gin"

func setupTimesheetRoutes(protected *gin.RouterGroup, container *Container) {
	timesheets := protected.Group("/timesheets")
	{
		timesheets.GET("/summary", container.Handlers.Timesheet.GetSummary)
		timesheets.GET("/cash-readiness", container.Handlers.Timesheet.GetCashReadiness)
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
		timesheets.POST("/reject-unpaid", container.Handlers.Timesheet.RejectUnpaidTimesheets)
		timesheets.POST("/bulk-reset", container.Handlers.Timesheet.BulkReset)
		timesheets.POST("/approve-all", container.Handlers.Timesheet.ApproveAllTimesheets)
		timesheets.POST("/reset-all", container.Handlers.Timesheet.ResetAllTimesheets)

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
