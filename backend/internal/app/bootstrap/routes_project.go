package bootstrap

import "github.com/gin-gonic/gin"

func setupProjectRoutes(protected *gin.RouterGroup, container *Container) {
	projects := protected.Group("/projects")
	{
		projects.POST("", container.Handlers.Project.CreateProject)
		projects.GET("", container.Handlers.Project.ListProjects)
		projects.GET("/summary", container.Handlers.Project.GetProjectSummary)
		projects.GET("/partner-summary", container.Handlers.Project.GetPartnerProjectSummary)
		projects.GET("/checkin-configurable", container.Handlers.ProjectEmployee.ListCheckInConfigurableProjects)
		projects.POST("/activate", container.Handlers.Project.ActivateProjects) // Admin-only endpoint for manual project activation
		projects.GET("/:id", container.Handlers.Project.GetProject)
		projects.PUT("/:id", container.Handlers.Project.UpdateProject)
		projects.DELETE("/:id", container.Handlers.Project.DeleteProject)

		projects.POST("/:id/employees", container.Handlers.ProjectEmployee.AssignEmployee)
		projects.PUT("/:id/employees", container.Handlers.ProjectEmployee.UpdateProjectEmployee)
		projects.POST("/:id/employees/remove", container.Handlers.ProjectEmployee.RemoveEmployeesFromProject)
		projects.GET("/:id/employees", container.Handlers.ProjectEmployee.ListProjectEmployees)
		projects.GET("/:id/employees/checkin-configuration", container.Handlers.ProjectEmployee.GetCheckInConfiguration)
		projects.PATCH("/:id/employees/:employeeId/checkin-enabled", container.Handlers.ProjectEmployee.ToggleCheckInEnabled)
		projects.PATCH("/:id/employees/checkin-enabled/bulk", container.Handlers.ProjectEmployee.BulkToggleCheckInEnabled)
		projects.PATCH("/:id/employees/checkin-enabled/disable-inactive", container.Handlers.ProjectEmployee.DisableInactiveCheckInEmployees)
		projects.PATCH("/:id/employees/checkin-enabled/disable-pending", container.Handlers.ProjectEmployee.DisablePendingCheckInEmployees)
		projects.DELETE("/:id/employees/:employeeId/checkin-enabled", container.Handlers.ProjectEmployee.CancelPendingCheckInEnable)

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
