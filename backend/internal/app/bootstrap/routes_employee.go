package bootstrap

import "github.com/gin-gonic/gin"

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
