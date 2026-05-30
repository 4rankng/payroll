package domain

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// GeofenceGate represents a named entry/exit point for geofence validation
type GeofenceGate struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

// ProjectStatus represents project status enum
type ProjectStatus string

const (
	ProjectStatusDraft     ProjectStatus = "draft"
	ProjectStatusRunning   ProjectStatus = "active" // Fixed: was "running", should be "active" to match DB schema
	ProjectStatusPaused    ProjectStatus = "paused"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusCancelled ProjectStatus = "cancelled"
)

// Project represents a client project in the domain
type Project struct {
	ID                   uint           `json:"id" gorm:"primarykey;type:bigint unsigned"`
	ClientName           string         `json:"client_name" gorm:"type:varchar(255);comment:'e.g. Nha may san xuat hoa my pham VERICO'"`
	Name                 string         `json:"name" gorm:"type:varchar(255);not null;comment:'e.g. San xuat xa phong'"`
	Code                 string         `json:"code" gorm:"type:varchar(255);uniqueIndex;comment:'Initials of client_name and YYMM of created_at date or manually input by users'"`
	Description          string         `json:"description" gorm:"type:text;comment:'Rich text format supported'"`
	StartDate            *time.Time     `json:"start_date" gorm:"type:date"`
	EndDate              *time.Time     `json:"end_date" gorm:"type:date"`
	SalaryPeriodFrom     int            `json:"salary_period_from,omitempty" gorm:"column:salary_period_from;type:int;comment:'Day of previous month payroll period starts (0 or NULL = 1st)'"`                               // 0-28
	SalaryPeriodTo       int            `json:"salary_period_to,omitempty" gorm:"column:salary_period_to;type:int;comment:'Day of current month payroll period ends (0 or NULL = last day)'"`                                 // 0-28
	OffDays              int            `json:"off_days" gorm:"column:off_days;type:tinyint unsigned;not null;default:0;comment:'Bitmask of weekly off days: bit0=Sun,bit1=Mon,...,bit6=Sat. Default 0 = no fixed off days'"` // bitmask
	TotalPayoutVND       float64        `json:"total_payout_vnd" gorm:"type:decimal(15,2);not null;default:0;comment:'Total paid to employees to date'"`
	PendingPayableVND    float64        `json:"pending_payable_vnd" gorm:"type:decimal(15,2);not null;default:0;comment:'Pending payment to employees to date'"`
	PendingReceivableVND float64        `json:"pending_receivable_vnd" gorm:"type:decimal(15,2);not null;default:0;comment:'Pending payment from the partner (staffing company) to date'"`
	TotalReceivedVND     float64        `json:"total_received_vnd" gorm:"type:decimal(15,2);not null;default:0;comment:'Total received from the partner to date'"`
	ProjectStatus        ProjectStatus  `json:"project_status" gorm:"column:project_status;type:enum('draft','active','paused','completed','cancelled');not null;default:'draft'"`
	IsFlexible           bool           `json:"is_flexible" gorm:"column:is_flexible;type:tinyint(1);not null;default:0;comment:'Whether this project uses flexible check-in schedules'"`
	GeofenceGates        []GeofenceGate `json:"geofence_gates" gorm:"type:json;serializer:json"`
	GeofenceRadiusMeters uint           `json:"geofence_radius_meters" gorm:"column:geofence_radius_meters;type:int unsigned;not null;default:100"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedBy            uint           `json:"created_by" gorm:"not null;type:bigint unsigned"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`

	// Relationships
	Creator User `json:"creator" gorm:"foreignKey:CreatedBy;references:ID"`
}

// NormalizeSalaryPeriodDay treats nil or zero values as the default period boundary (first/last day)
func NormalizeSalaryPeriodDay(value *int) int {
	if value == nil || *value == 0 {
		return 0
	}
	return *value
}

// GetAuditEntityType implements Auditable interface
func (p Project) GetAuditEntityType() string {
	return "project"
}

// GetAuditEntityID implements Auditable interface
func (p Project) GetAuditEntityID() uint {
	return p.ID
}

// ProjectRepository defines the interface for project persistence operations
type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id uint) (*Project, error)
	GetByIDs(ctx context.Context, ids []uint) (map[uint]*Project, error)
	GetByIDWithEmployeeCount(ctx context.Context, id uint) (*ProjectWithEmployeeCount, error)
	GetByCode(ctx context.Context, code string) (*Project, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters ProjectFilters) ([]*Project, error)
	ListWithEmployeeCount(ctx context.Context, filters ProjectFilters) ([]*ProjectWithEmployeeCount, error)
	Count(ctx context.Context, filters ProjectFilters) (int64, error)
	GetSummary(ctx context.Context) (*ProjectSummary, error)
	GetActiveCountAtDate(ctx context.Context, date time.Time) (int, error)
	GetStatusCounts(ctx context.Context) (map[string]int, error)
	GetStatusCountsForCreator(ctx context.Context, createdBy uint) (map[string]int, error)
	SearchProjects(ctx context.Context, search string, limit int) ([]*Project, error)
	GetPendingActivatedProjects(ctx context.Context, date time.Time) ([]*Project, error)
	GetPendingCompletedProjects(ctx context.Context, date time.Time) ([]*Project, error)
	CodeExistsIncludingDeleted(ctx context.Context, code string) bool
}

// ProjectFilters represents filtering options for project queries
type ProjectFilters struct {
	ProjectStatus           []ProjectStatus
	CreatedBy               *uint
	AccessibleBy            *uint // New field for filtering accessible projects (owned or shared)
	SkipAccessibilityFilter bool  // Skip accessibility filter for separate query approach
	FromDate                *time.Time
	ToDate                  *time.Time
	Search                  string
	Limit                   int
	Offset                  int
	SortBy                  string
	SortOrder               string
}

// GetStatuses implements common.StatusFilter interface
func (pf *ProjectFilters) GetStatuses() []string {
	if len(pf.ProjectStatus) == 0 {
		return nil
	}
	statuses := make([]string, len(pf.ProjectStatus))
	for i, status := range pf.ProjectStatus {
		statuses[i] = string(status)
	}
	return statuses
}

// GetStatusField implements common.StatusFilter interface
func (pf *ProjectFilters) GetStatusField() string {
	return "project_status"
}

// GetFromDate implements common.DateRangeFilter interface
func (pf *ProjectFilters) GetFromDate() *time.Time {
	return pf.FromDate
}

// GetToDate implements common.DateRangeFilter interface
func (pf *ProjectFilters) GetToDate() *time.Time {
	return pf.ToDate
}

// GetDateField implements common.DateRangeFilter interface
func (pf *ProjectFilters) GetDateField() string {
	return "created_at"
}

// GetCreatedBy implements common.CreatorFilter interface
func (pf *ProjectFilters) GetCreatedBy() *uint {
	return pf.CreatedBy
}

// GetCreatorField implements common.CreatorFilter interface
func (pf *ProjectFilters) GetCreatorField() string {
	return "created_by"
}

// GetSearch implements common.SearchFilter interface
func (pf *ProjectFilters) GetSearch() string {
	return pf.Search
}

// GetSearchFields implements common.SearchFilter interface
func (pf *ProjectFilters) GetSearchFields() []string {
	return []string{"name", "code", "client_name", "description"}
}

// ProjectSummary represents aggregated project metrics
type ProjectSummary struct {
	TotalActiveProjects       int64   `json:"total_active_projects"`
	TotalReceivedVND          float64 `json:"total_received_vnd"`
	TotalPayoutVND            float64 `json:"total_payout_vnd"`
	TotalPendingPayableVND    float64 `json:"total_pending_payable_vnd"`
	TotalPendingReceivableVND float64 `json:"total_pending_receivable_vnd"`
}

// PartnerProjectSummary represents project statistics for partner users
type PartnerProjectSummary struct {
	TotalProjects     int `json:"total_projects"`
	ActiveProjects    int `json:"active_projects"`
	CompletedProjects int `json:"completed_projects"`
	TotalEmployees    int `json:"total_employees"`
}

// ValidateName validates the project name
func (p *Project) ValidateName() error {
	if p.Name == "" {
		return NewValidationError("project name is required")
	}
	return nil
}

// ValidateDates validates start and end dates
func (p *Project) ValidateDates() error {
	if p.StartDate != nil && p.EndDate != nil {
		if p.EndDate.Before(*p.StartDate) {
			return NewValidationError("end date must be after start date")
		}
	}
	return nil
}

// ValidateStatus validates the project status
func (p *Project) ValidateStatus() error {
	validStatuses := map[ProjectStatus]bool{
		ProjectStatusDraft:     true,
		ProjectStatusRunning:   true,
		ProjectStatusPaused:    true,
		ProjectStatusCompleted: true,
		ProjectStatusCancelled: true,
	}

	if !validStatuses[p.ProjectStatus] {
		return NewValidationError("invalid project status")
	}
	return nil
}

// IsValid validates the entire project entity
func (p *Project) IsValid() error {
	if err := p.ValidateName(); err != nil {
		return err
	}
	if err := p.ValidateDates(); err != nil {
		return err
	}
	if err := p.ValidateStatus(); err != nil {
		return err
	}
	return nil
}

// ValidateGeofenceGates validates geofence gate configuration
func (p *Project) ValidateGeofenceGates() error {
	if p.GeofenceRadiusMeters < 10 || p.GeofenceRadiusMeters > 1000 {
		return NewValidationError("Bán kính geofence phải từ 10 đến 1000 mét")
	}
	if len(p.GeofenceGates) > 20 {
		return NewValidationError("Tối đa 20 cổng check-in mỗi dự án")
	}
	for i, gate := range p.GeofenceGates {
		if gate.Name == "" {
			return NewValidationError(fmt.Sprintf("Tên cổng %d là bắt buộc", i+1))
		}
		if len(gate.Name) > 50 {
			return NewValidationError(fmt.Sprintf("Tên cổng %d không được vượt quá 50 ký tự", i+1))
		}
		if gate.Lat < -90 || gate.Lat > 90 {
			return NewValidationError(fmt.Sprintf("Vĩ độ cổng %d không hợp lệ (-90 đến 90)", i+1))
		}
		if gate.Lng < -180 || gate.Lng > 180 {
			return NewValidationError(fmt.Sprintf("Kinh độ cổng %d không hợp lệ (-180 đến 180)", i+1))
		}
	}
	return nil
}

// IsRunning returns true if project status is running
func (p *Project) IsRunning() bool {
	return p.ProjectStatus == ProjectStatusRunning
}

// IsDraft returns true if project status is draft
func (p *Project) IsDraft() bool {
	return p.ProjectStatus == ProjectStatusDraft
}

// IsCompleted returns true if project status is completed
func (p *Project) IsCompleted() bool {
	return p.ProjectStatus == ProjectStatusCompleted
}

// IsCancelled returns true if project status is cancelled
func (p *Project) IsCancelled() bool {
	return p.ProjectStatus == ProjectStatusCancelled
}

// IsPaused returns true if project status is paused
func (p *Project) IsPaused() bool {
	return p.ProjectStatus == ProjectStatusPaused
}

// CanTransitionTo checks if project status transition is allowed
func (p *Project) CanTransitionTo(newStatus ProjectStatus) bool {
	currentStatus := p.ProjectStatus

	// Define allowed transitions
	allowedTransitions := map[ProjectStatus][]ProjectStatus{
		ProjectStatusDraft: {
			ProjectStatusRunning,
			ProjectStatusPaused,
			ProjectStatusCancelled,
		},
		ProjectStatusRunning: {
			ProjectStatusPaused,
			ProjectStatusCompleted,
			ProjectStatusCancelled,
		},
		ProjectStatusPaused: {
			ProjectStatusRunning,
			ProjectStatusCompleted,
			ProjectStatusCancelled,
		},
		ProjectStatusCompleted: {}, // No transitions allowed from completed
		ProjectStatusCancelled: {}, // No transitions allowed from cancelled
	}

	allowedTargets, exists := allowedTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowed := range allowedTargets {
		if allowed == newStatus {
			return true
		}
	}

	return false
}

// GetTotalExpenses calculates total expenses from payout + pending payable
func (p *Project) GetTotalExpenses() float64 {
	return p.TotalPayoutVND + p.PendingPayableVND
}

// GetProfitMargin calculates profit margin percentage based on total revenue
func (p *Project) GetProfitMargin() float64 {
	totalRevenue := p.TotalReceivedVND + p.PendingReceivableVND
	if totalRevenue == 0 {
		return 0
	}

	totalExpenses := p.GetTotalExpenses()
	profit := totalRevenue - totalExpenses

	return (profit / totalRevenue) * 100
}

// Additional types for repository operations

// ProjectWithEmployeeCount represents project with employee count
type ProjectWithEmployeeCount struct {
	Project
	CreatorFullname            string `json:"creator_fullname"`
	EmployeeCount              int    `json:"employee_count"`
	WeeklySalaryEmployeeCount  int    `json:"weekly_salary_employee_count"`
	MonthlySalaryEmployeeCount int    `json:"monthly_salary_employee_count"`
}

// ProjectFinancialSummary represents financial summary for projects
type ProjectFinancialSummary struct {
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	TotalProjects   int       `json:"total_projects"`
	TotalRevenue    float64   `json:"total_revenue"`
	TotalExpenses   float64   `json:"total_expenses"`
	NetProfit       float64   `json:"net_profit"`
	PendingRevenue  float64   `json:"pending_revenue"`
	PendingExpenses float64   `json:"pending_expenses"`
}
