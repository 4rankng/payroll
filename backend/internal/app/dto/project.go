package dto

import (
	"time"
)

// GeofenceGateRequest represents a geofence gate in API requests
type GeofenceGateRequest struct {
	Name string  `json:"name" binding:"required"`
	Lat  float64 `json:"lat" binding:"required"`
	Lng  float64 `json:"lng" binding:"required"`
}

// CreateProjectRequest represents the request to create a new project
type CreateProjectRequest struct {
	ClientName  string  `json:"client_name" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Code        string  `json:"code,omitempty"`
	Description string  `json:"description,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
	Status      *string `json:"status,omitempty" binding:"omitempty,oneof=draft active"`
	// Salary period configuration (monthly exports)
	SalaryPeriodFrom *int `json:"salary_period_from,omitempty"`
	SalaryPeriodTo   *int `json:"salary_period_to,omitempty"`
	// Off days configuration (bitmask: bit0=Sun, bit1=Mon, ..., bit6=Sat)
	OffDays *int `json:"off_days,omitempty"`
	// Flexible project configuration
	IsFlexible           *bool                 `json:"is_flexible,omitempty"`
	GeofenceGates        []GeofenceGateRequest `json:"geofence_gates,omitempty"`
	GeofenceRadiusMeters *uint                 `json:"geofence_radius_meters,omitempty"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	ClientName       *string `json:"client_name,omitempty"`
	Name             *string `json:"name,omitempty"`
	Code             *string `json:"code,omitempty"`
	Description      *string `json:"description,omitempty"`
	StartDate        *string `json:"start_date,omitempty"`
	EndDate          *string `json:"end_date,omitempty"`
	Status           *string `json:"status,omitempty" binding:"omitempty,oneof=draft active completed cancelled inactive"`
	SalaryPeriodFrom *int    `json:"salary_period_from,omitempty"`
	SalaryPeriodTo   *int    `json:"salary_period_to,omitempty"`
	// Off days configuration (bitmask: bit0=Sun, bit1=Mon, ..., bit6=Sat)
	OffDays *int `json:"off_days,omitempty"`
	// Flexible project configuration
	IsFlexible           *bool                 `json:"is_flexible,omitempty"`
	GeofenceGates        []GeofenceGateRequest `json:"geofence_gates,omitempty"`
	GeofenceRadiusMeters *uint                 `json:"geofence_radius_meters,omitempty"`
}
type ProjectResponse struct {
	ID                         uint                  `json:"id"`
	ClientName                 string                `json:"client_name"`
	Name                       string                `json:"name"`
	Code                       string                `json:"code"`
	Description                string                `json:"description"`
	StartDate                  *string               `json:"start_date"`
	EndDate                    *string               `json:"end_date"`
	SalaryPeriodFrom           int                   `json:"salary_period_from"`
	SalaryPeriodTo             int                   `json:"salary_period_to"`
	OffDays                    int                   `json:"off_days"`
	EmployeeCount              int                   `json:"employee_count"`
	WeeklySalaryEmployeeCount  int                   `json:"weekly_salary_employee_count"`
	MonthlySalaryEmployeeCount int                   `json:"monthly_salary_employee_count"`
	Status                     string                `json:"status"`
	IsFlexible                 bool                  `json:"is_flexible"`
	GeofenceGates              []GeofenceGateRequest `json:"geofence_gates"`
	GeofenceRadiusMeters       uint                  `json:"geofence_radius_meters"`
	CreatedBy                  uint                  `json:"created_by"`
	CreatedAt                  time.Time             `json:"created_at"`
	UpdatedAt                  time.Time             `json:"updated_at"`
}

// ListProjectsResponse represents the response for listing projects
type ListProjectsResponse struct {
	Projects []ProjectResponse `json:"projects"`
	Total    int64             `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// ProjectSummaryResponse represents aggregated project metrics
type ProjectSummaryResponse struct {
	TotalActiveProjects       int64   `json:"total_active_projects"`
	TotalReceivedVND          float64 `json:"total_received_vnd"`
	TotalPayoutVND            float64 `json:"total_payout_vnd"`
	TotalPendingPayableVND    float64 `json:"total_pending_payable_vnd"`
	TotalPendingReceivableVND float64 `json:"total_pending_receivable_vnd"`
}

// ProjectEmployeeAssignments represents employee assignment summary for a project
type ProjectEmployeeAssignments struct {
	TotalEmployees  int                     `json:"total_employees"`
	Positions       []ProjectPositionCount  `json:"positions"`
	RecentEmployees []ProjectRecentEmployee `json:"recent_employees,omitempty"`
}

// ProjectPositionCount represents position counts in a project
type ProjectPositionCount struct {
	Position string `json:"position"`
	Count    int    `json:"count"`
}

// ProjectRecentEmployee represents recent employee information
type ProjectRecentEmployee struct {
	EmployeeID uint   `json:"employee_id"`
	FullName   string `json:"fullname"`
	Position   string `json:"position"`
	StartDate  string `json:"start_date"`
}

// ProjectCurrentPayrate represents current active payrate for a project
type ProjectCurrentPayrate struct {
	ID            uint                   `json:"id"`
	FromDate      string                 `json:"fromDate"`
	ToDate        *string                `json:"toDate"`
	Configuration map[string]interface{} `json:"configuration"`
	Status        string                 `json:"status"`
}

// ProjectTimesheetSummary represents timesheet summary for a project
type ProjectTimesheetSummary struct {
	TotalTimesheets   int     `json:"total_timesheets"`
	PendingApproval   int     `json:"pending_approval"`
	ApprovedCount     int     `json:"approved_count"`
	TotalHoursWorked  float64 `json:"total_hours_worked"`
	TotalAmountVND    int64   `json:"total_amount_vnd"`
	CurrentMonthHours float64 `json:"current_month_hours"`
	LastEntryDate     *string `json:"last_entry_date,omitempty"`
}

// ProjectFinancialSummary represents enhanced financial summary
type ProjectFinancialSummary struct {
	TotalPayoutVND       float64 `json:"total_payout_vnd"`
	PendingPayableVND    float64 `json:"pending_payable_vnd"`
	PendingReceivableVND float64 `json:"pending_receivable_vnd"`
	TotalReceivedVND     float64 `json:"total_received_vnd"`
	TotalRevenueVND      float64 `json:"total_revenue_vnd"`
	TotalExpensesVND     float64 `json:"total_expenses_vnd"`
	NetProfitVND         float64 `json:"net_profit_vnd"`
	ProfitMarginPercent  float64 `json:"profit_margin_percent"`
}

// ProjectRecentActivity represents recent project activities
type ProjectRecentActivity struct {
	Type         string    `json:"type"`
	Description  string    `json:"description"`
	Date         time.Time `json:"date"`
	EmployeeName *string   `json:"employee_name,omitempty"`
}

// ProjectDetailedResponse represents comprehensive project information
type ProjectDetailedResponse struct {
	// Basic project information
	ID                   uint                  `json:"id"`
	ClientName           string                `json:"client_name"`
	Name                 string                `json:"name"`
	Code                 string                `json:"code"`
	Description          string                `json:"description"`
	StartDate            *string               `json:"start_date"`
	EndDate              *string               `json:"end_date"`
	SalaryPeriodFrom     int                   `json:"salary_period_from"`
	SalaryPeriodTo       int                   `json:"salary_period_to"`
	OffDays              int                   `json:"off_days"`
	Status               string                `json:"status"`
	IsFlexible           bool                  `json:"is_flexible"`
	GeofenceGates        []GeofenceGateRequest `json:"geofence_gates"`
	GeofenceRadiusMeters uint                  `json:"geofence_radius_meters"`
	CreatedBy            uint                  `json:"created_by"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`

	// Comprehensive project details
	EmployeeAssignments *ProjectEmployeeAssignments `json:"employee_assignments,omitempty"`
	CurrentPayrate      *ProjectCurrentPayrate      `json:"current_payrate,omitempty"`
	TimesheetSummary    *ProjectTimesheetSummary    `json:"timesheet_summary,omitempty"`
	FinancialSummary    *ProjectFinancialSummary    `json:"financial_summary,omitempty"`
	RecentActivity      []ProjectRecentActivity     `json:"recent_activity,omitempty"`
}

// ProjectActivationResponse represents the response for manual project activation
type ProjectActivationResponse struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	UserID    uint      `json:"user_id"`
}

// Common error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Project sharing DTOs

// GrantProjectAccessRequest represents request to grant project access to a user
type GrantProjectAccessRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

// ProjectUserResponse represents a user with project access
type ProjectUserResponse struct {
	ID           uint      `json:"id"`
	UserID       uint      `json:"user_id"`
	UserFullname string    `json:"user_fullname"`
	UserEmail    string    `json:"user_email"`
	GrantedBy    uint      `json:"granted_by"`
	GrantedAt    time.Time `json:"granted_at"`
}
