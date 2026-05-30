package dto

// ExportTimesheetTemplateRequest represents the request to export timesheet template
type ExportTimesheetTemplateRequest struct {
	FromDate  string `json:"fromDate" binding:"required"`
	ToDate    string `json:"toDate" binding:"required"`
	ProjectID uint   `json:"projectId" binding:"required"`
}
