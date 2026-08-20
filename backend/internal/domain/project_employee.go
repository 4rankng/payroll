package domain

import (
	"context"
	"strings"
	"time"

	"api-server/internal/pkg/clock"
	"gorm.io/gorm"
)

// PaymentSchedule represents the payment schedule type
type PaymentSchedule string

const (
	PaymentScheduleWeekly   PaymentSchedule = "weekly"
	PaymentScheduleMonthly  PaymentSchedule = "monthly"
	PaymentScheduleFlexible PaymentSchedule = "flexible"
)

// ProjectEmployee represents the assignment between a project and an employee
type ProjectEmployee struct {
	ID           uint       `json:"id" gorm:"primarykey;type:bigint unsigned"`
	ProjectID    uint       `json:"project_id" gorm:"not null;type:bigint unsigned"`
	EmployeeID   uint       `json:"employee_id" gorm:"not null;type:bigint unsigned"`
	EmployeeName string     `json:"employee_name" gorm:"not null;type:varchar(255)"`
	EmployeeCCCD string     `json:"employee_cccd" gorm:"not null;type:varchar(255);comment:'Citizen ID - Can cong cong dan (12 digits)'"`
	EmployeeCode string     `json:"employee_code" gorm:"type:varchar(255);comment:'Factory-assigned code, optional'"`
	Position     string     `json:"position" gorm:"type:varchar(100);not null;default:'phổ thông';comment:'Employee position for payrate calculation'"`
	StartDate    time.Time  `json:"start_date" gorm:"type:date;not null"`
	LastDate     *time.Time `json:"last_date" gorm:"type:date;comment:'NULL means currently active'"`

	// Payment Schedule Fields
	PaymentSchedule        string     `json:"payment_schedule" gorm:"type:varchar(10);not null;default:'weekly';comment:'Current payment schedule: weekly or monthly'"`
	PendingPaymentSchedule *string    `json:"pending_payment_schedule,omitempty" gorm:"type:varchar(10);comment:'Pending payment schedule change'"`
	ScheduleEffectiveFrom  *time.Time `json:"schedule_effective_from,omitempty" gorm:"type:date;comment:'Date when pending schedule change becomes effective'"`

	CheckInEnabled bool `json:"check_in_enabled" gorm:"column:check_in_enabled;type:tinyint(1);not null;default:0"`

	// Deferred check-in activation: enabling check-in takes effect on day 1 of
	// the next month. NULL pending fields = no pending change.
	PendingCheckInEnabled *bool      `json:"pending_check_in_enabled,omitempty" gorm:"type:tinyint(1);comment:'Pending check-in enable awaiting activation'"`
	CheckInEffectiveFrom  *time.Time `json:"check_in_effective_from,omitempty" gorm:"type:date;comment:'Date when the pending check-in enable becomes effective'"`

	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedBy uint           `json:"created_by" gorm:"not null;type:bigint unsigned"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// Relationships
	Project  Project  `json:"project" gorm:"foreignKey:ProjectID;references:ID"`
	Employee Employee `json:"employee" gorm:"foreignKey:EmployeeID;references:ID"`
	Creator  User     `json:"creator" gorm:"foreignKey:CreatedBy;references:ID"`
}

// ProjectEmployeeRepository defines the interface for project employee persistence operations
type ProjectEmployeeRepository interface {
	Create(ctx context.Context, assignment *ProjectEmployee) error
	GetByID(ctx context.Context, id uint) (*ProjectEmployee, error)
	GetByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*ProjectEmployee, error)
	GetActiveAssignmentByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*ProjectEmployee, error)
	GetActiveAssignmentsByProjectsAndEmployees(ctx context.Context, projectIDs []uint, employeeIDs []uint) ([]*ProjectEmployee, error)
	Update(ctx context.Context, assignment *ProjectEmployee) error
	// UpdatePosition updates only the position column, avoiding full-row Save() that
	// could overwrite concurrent changes to other fields (e.g., CheckInEnabled).
	UpdatePosition(ctx context.Context, id uint, position string) error
	// UpdatePositionIfCurrent performs a compare-and-swap update so imports never
	// overwrite a concurrent assignment edit made after their initial read.
	UpdatePositionIfCurrent(ctx context.Context, id uint, currentPosition, newPosition string) error
	Delete(ctx context.Context, id uint) error
	DeleteByProjectID(ctx context.Context, projectID uint) error
	DeleteAssignmentsByEmployeeID(ctx context.Context, employeeID uint) error
	HardDeleteAssignmentsByEmployeeID(ctx context.Context, employeeID uint) error
	HasActiveFlexiblePaymentScheduleByEmployeeID(ctx context.Context, employeeID uint) (bool, error)
	List(ctx context.Context, filters ProjectEmployeeFilters) ([]*ProjectEmployee, error)
	Count(ctx context.Context, filters ProjectEmployeeFilters) (int64, error)
	GetByProject(ctx context.Context, projectID uint) ([]*ProjectEmployee, error)
	GetByEmployee(ctx context.Context, employeeID uint) ([]*ProjectEmployee, error)
	GetActiveAssignments(ctx context.Context, projectID uint) ([]*ProjectEmployee, error)
	EndAssignment(ctx context.Context, id uint, endDate time.Time) error
	BulkUpdateLastDate(ctx context.Context, assignmentIDs []uint, lastDate time.Time) error
	GetCurrentProjectForEmployee(ctx context.Context, employeeID uint) (*Project, error)
	GetCurrentProjectsForEmployees(ctx context.Context, employeeIDs []uint) (map[uint]*Project, error)
	GetActiveProjectsForEmployee(ctx context.Context, employeeID uint) ([]*Project, error)
	GetActiveEmployeesForProjectWithCreatorFilter(ctx context.Context, projectID uint, createdBy *uint) ([]*Employee, error)
	GetOverlappingAssignments(ctx context.Context, employeeID uint, startDate, endDate *time.Time) ([]*ProjectEmployee, error)
	CountWorkingEmployees(ctx context.Context) (int, error)

	// Payment Schedule methods
	GetEmployeesByPaymentSchedule(ctx context.Context, schedule PaymentSchedule) ([]*ProjectEmployee, error)
	CountActiveEmployeesByPaymentSchedule(ctx context.Context, schedule PaymentSchedule) (int, error)
	GetEmployeesWithPendingScheduleChanges(ctx context.Context, effectiveDate time.Time) ([]*ProjectEmployee, error)
	ApplyScheduleChanges(ctx context.Context, employeeIDs []uint) error

	// Deferred check-in activation methods
	GetEmployeesWithPendingCheckInEnable(ctx context.Context, effectiveDate time.Time) ([]*ProjectEmployee, error)

	// HasAccessViaProject checks if user can access employee through project assignments
	HasAccessViaProject(ctx context.Context, employeeID, userID uint) (bool, error)
}

// ProjectEmployeeFilters represents filtering options for project employee queries
type ProjectEmployeeFilters struct {
	ProjectID       *uint
	EmployeeID      *uint
	CreatedBy       *uint
	FromDate        *time.Time
	ToDate          *time.Time
	ActiveOnly      bool // Show only active assignments (LastDate is null)
	Status          string
	Search          string           // Free-text search across employee_name, employee_cccd, employee_code
	PaymentSchedule *PaymentSchedule // Filter by payment schedule (weekly/monthly)
	CheckInEnabled  *bool            // Filter by check-in enabled status
	Limit           int
	Offset          int
	SortBy          string
	SortOrder       string
}

// GetSearch implements common.SearchFilter so the persistence layer can route
// free-text search through common.FilterBuilder.ApplyVietnameseSearch (the
// shared chokepoint used by project_repository and others), keeping accent/
// case handling consistent across every search surface.
func (f ProjectEmployeeFilters) GetSearch() string {
	return strings.TrimSpace(f.Search)
}

// GetSearchFields implements common.SearchFilter — the denormalized employee
// columns on project_employees that the free-text search scans.
func (f ProjectEmployeeFilters) GetSearchFields() []string {
	return []string{"employee_name", "employee_cccd", "employee_code"}
}

// ProjectEmployeeWithDetails represents project employee with full project and employee details
type ProjectEmployeeWithDetails struct {
	ID           uint       `json:"id"`
	ProjectID    uint       `json:"project_id"`
	EmployeeID   uint       `json:"employee_id"`
	EmployeeCode string     `json:"employee_code"`
	Position     string     `json:"position"`
	StartDate    time.Time  `json:"start_date"`
	LastDate     *time.Time `json:"last_date"`
	CreatedBy    uint       `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Payment Schedule
	PaymentSchedule        string     `json:"payment_schedule"`
	PendingPaymentSchedule *string    `json:"pending_payment_schedule,omitempty"`
	ScheduleEffectiveFrom  *time.Time `json:"schedule_effective_from,omitempty"`

	CheckInEnabled bool `json:"check_in_enabled"`

	// Deferred check-in activation
	PendingCheckInEnabled *bool      `json:"pending_check_in_enabled,omitempty"`
	CheckInEffectiveFrom  *time.Time `json:"check_in_effective_from,omitempty"`

	// Project details
	ProjectName   string        `json:"project_name"`
	ProjectCode   string        `json:"project_code"`
	ProjectStatus ProjectStatus `json:"project_status"`
	ClientName    string        `json:"client_name"`

	// Employee details
	EmployeeFullname string `json:"employee_fullname"`
	EmployeeCCCD     string `json:"employee_cccd"`
	EmployeeEmail    string `json:"employee_email"`
}

// ValidateProjectID validates the project ID
func (pe *ProjectEmployee) ValidateProjectID() error {
	if pe.ProjectID == 0 {
		return NewValidationError("Mã dự án là bắt buộc")
	}
	return nil
}

// ValidateEmployeeID validates the employee ID
func (pe *ProjectEmployee) ValidateEmployeeID() error {
	if pe.EmployeeID == 0 {
		return NewValidationError("Mã nhân viên là bắt buộc")
	}
	return nil
}

// ValidateDates validates start and end dates
func (pe *ProjectEmployee) ValidateDates() error {
	if pe.StartDate.IsZero() {
		return NewValidationError("Ngày bắt đầu là bắt buộc")
	}

	if pe.LastDate != nil {
		if pe.LastDate.Before(pe.StartDate) {
			return NewValidationError("Ngày kết thúc phải sau ngày bắt đầu")
		}
	}

	return nil
}

// ValidatePosition validates the position field
func (pe *ProjectEmployee) ValidatePosition() error {
	if pe.Position == "" {
		return NewValidationError("Vị trí làm việc là bắt buộc")
	}
	// Position can be any string value, no specific validation needed
	return nil
}

// ValidatePaymentSchedule validates the payment schedule field
func (pe *ProjectEmployee) ValidatePaymentSchedule() error {
	if pe.PaymentSchedule == "" {
		return NewValidationError("Chu kỳ thanh toán là bắt buộc")
	}
	if pe.PaymentSchedule != string(PaymentScheduleWeekly) && pe.PaymentSchedule != string(PaymentScheduleMonthly) && pe.PaymentSchedule != string(PaymentScheduleFlexible) {
		return NewValidationError("Chu kỳ thanh toán phải là '" + string(PaymentScheduleWeekly) + "', '" + string(PaymentScheduleMonthly) + "' hoặc '" + string(PaymentScheduleFlexible) + "'")
	}
	return nil
}

// ValidateScheduleChange validates a pending payment schedule change
func (pe *ProjectEmployee) ValidateScheduleChange(newSchedule PaymentSchedule, effectiveDate time.Time) error {
	// Cannot change to the same schedule
	if string(newSchedule) == pe.PaymentSchedule {
		return NewValidationError("Chu kỳ thanh toán mới phải khác với chu kỳ hiện tại")
	}

	// Validate the new schedule value
	if newSchedule != PaymentScheduleWeekly && newSchedule != PaymentScheduleMonthly && newSchedule != PaymentScheduleFlexible {
		return NewValidationError("Chu kỳ thanh toán phải là '" + string(PaymentScheduleWeekly) + "', '" + string(PaymentScheduleMonthly) + "' hoặc '" + string(PaymentScheduleFlexible) + "'")
	}

	// Effective date must be first day of a month
	if effectiveDate.Day() != 1 {
		return NewValidationError("Ngày hiệu lực phải là ngày đầu tiên của tháng")
	}

	// Effective date must be in the future
	now := clock.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if !effectiveDate.After(startOfToday) {
		return NewValidationError("Ngày hiệu lực phải là ngày trong tương lai")
	}

	// Must be at least the first day of next month
	firstDayOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	if effectiveDate.Before(firstDayOfNextMonth) {
		return NewValidationError("Chỉ có thể thay đổi chu kỳ thanh toán từ tháng sau trở đi")
	}

	return nil
}

// IsValid validates the entire project employee assignment
func (pe *ProjectEmployee) IsValid() error {
	if err := pe.ValidateProjectID(); err != nil {
		return err
	}
	if err := pe.ValidateEmployeeID(); err != nil {
		return err
	}
	if err := pe.ValidatePosition(); err != nil {
		return err
	}
	if err := pe.ValidateDates(); err != nil {
		return err
	}
	if err := pe.ValidatePaymentSchedule(); err != nil {
		return err
	}
	return nil
}

// IsCurrentlyAssigned returns true if employee is currently assigned (no end date)
func (pe *ProjectEmployee) IsCurrentlyAssigned() bool {
	return pe.LastDate == nil
}

// HasEnded returns true if assignment has ended
func (pe *ProjectEmployee) HasEnded() bool {
	return pe.LastDate != nil
}

// GetAssignmentDuration calculates the duration of assignment in days
func (pe *ProjectEmployee) GetAssignmentDuration() int {
	endDate := clock.Now()
	if pe.LastDate != nil {
		endDate = *pe.LastDate
	}

	duration := endDate.Sub(pe.StartDate)
	return int(duration.Hours() / 24)
}

// CanEndAssignment checks if the assignment can be ended
func (pe *ProjectEmployee) CanEndAssignment() bool {
	return pe.IsCurrentlyAssigned()
}

// EndAssignment sets the end date for the assignment
func (pe *ProjectEmployee) EndAssignment(endDate time.Time) error {
	if !pe.CanEndAssignment() {
		return NewValidationError("Không thể kết thúc phân công này")
	}

	if endDate.Before(pe.StartDate) {
		return NewValidationError("Ngày kết thúc phải sau ngày bắt đầu")
	}

	pe.LastDate = &endDate

	return nil
}

// CanCreateTimesheet checks if timesheet can be created for this assignment on given date
func (pe *ProjectEmployee) CanCreateTimesheet(date time.Time) bool {
	// Date must be on or after start date
	if date.Before(pe.StartDate) {
		return false
	}

	// If assignment has ended, date must be before or on end date
	if pe.LastDate != nil && date.After(*pe.LastDate) {
		return false
	}

	return true
}

// IsAssignedOnDate checks if employee was assigned to project on specific date
func (pe *ProjectEmployee) IsAssignedOnDate(date time.Time) bool {
	// Date must be on or after start date
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	startDateOnly := time.Date(pe.StartDate.Year(), pe.StartDate.Month(), pe.StartDate.Day(), 0, 0, 0, 0, pe.StartDate.Location())

	if dateOnly.Before(startDateOnly) {
		return false
	}

	// If assignment has ended, date must be on or before end date
	if pe.LastDate != nil {
		endDateOnly := time.Date(pe.LastDate.Year(), pe.LastDate.Month(), pe.LastDate.Day(), 0, 0, 0, 0, pe.LastDate.Location())
		if dateOnly.After(endDateOnly) {
			return false
		}
	}

	return true
}

// Payment Schedule Management Methods

// HasPendingScheduleChange returns true if there's a pending payment schedule change
func (pe *ProjectEmployee) HasPendingScheduleChange() bool {
	return pe.PendingPaymentSchedule != nil && pe.ScheduleEffectiveFrom != nil
}

// CanChangeSchedule checks if the payment schedule can be changed
func (pe *ProjectEmployee) CanChangeSchedule() bool {
	// Only active employees can change their payment schedule
	return pe.IsCurrentlyAssigned()
}

// RequestScheduleChange schedules a payment schedule change
func (pe *ProjectEmployee) RequestScheduleChange(newSchedule PaymentSchedule, effectiveDate time.Time) error {
	if !pe.CanChangeSchedule() {
		return NewValidationError("Chỉ có thể thay đổi chu kỳ thanh toán cho nhân viên đang làm việc")
	}

	if err := pe.ValidateScheduleChange(newSchedule, effectiveDate); err != nil {
		return err
	}

	// Check if there's already a pending change
	if pe.HasPendingScheduleChange() {
		return NewValidationError("Đã có thay đổi chu kỳ thanh toán đang chờ xử lý")
	}

	scheduleStr := string(newSchedule)
	pe.PendingPaymentSchedule = &scheduleStr
	pe.ScheduleEffectiveFrom = &effectiveDate

	return nil
}

// ApplyScheduleChangeImmediately switches the payment schedule without waiting period
func (pe *ProjectEmployee) ApplyScheduleChangeImmediately(newSchedule PaymentSchedule) error {
	if !pe.CanChangeSchedule() {
		return NewValidationError("Chỉ có thể thay đổi chu kỳ thanh toán cho nhân viên đang làm việc")
	}

	if newSchedule != PaymentScheduleWeekly && newSchedule != PaymentScheduleMonthly && newSchedule != PaymentScheduleFlexible {
		return NewValidationError("Chu kỳ thanh toán phải là '" + string(PaymentScheduleWeekly) + "', '" + string(PaymentScheduleMonthly) + "' hoặc '" + string(PaymentScheduleFlexible) + "'")
	}

	if string(newSchedule) == pe.PaymentSchedule {
		return NewValidationError("Chu kỳ thanh toán mới phải khác với chu kỳ hiện tại")
	}

	if pe.HasPendingScheduleChange() {
		return NewValidationError("Đã có thay đổi chu kỳ thanh toán đang chờ xử lý")
	}

	pe.PaymentSchedule = string(newSchedule)
	pe.PendingPaymentSchedule = nil
	pe.ScheduleEffectiveFrom = nil

	return nil
}

// ApplyPendingSchedule applies the pending payment schedule change if the effective date has arrived
func (pe *ProjectEmployee) ApplyPendingSchedule() bool {
	if !pe.HasPendingScheduleChange() {
		return false
	}

	now := clock.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Check if the effective date has arrived or passed
	if !pe.ScheduleEffectiveFrom.After(startOfToday) {
		pe.PaymentSchedule = *pe.PendingPaymentSchedule
		pe.PendingPaymentSchedule = nil
		pe.ScheduleEffectiveFrom = nil
		return true
	}

	return false
}

// CancelPendingScheduleChange cancels a pending payment schedule change
func (pe *ProjectEmployee) CancelPendingScheduleChange() error {
	if !pe.HasPendingScheduleChange() {
		return NewValidationError("Không có thay đổi chu kỳ thanh toán nào đang chờ xử lý")
	}

	pe.PendingPaymentSchedule = nil
	pe.ScheduleEffectiveFrom = nil

	return nil
}

// Deferred check-in activation methods (mirror the schedule-pending pattern).

// HasPendingCheckInEnable returns true if a check-in enable is awaiting activation
func (pe *ProjectEmployee) HasPendingCheckInEnable() bool {
	return pe.PendingCheckInEnabled != nil && *pe.PendingCheckInEnabled && pe.CheckInEffectiveFrom != nil
}

// RequestCheckInEnable records a deferred check-in enable: activation happens
// on day 1 of the month AFTER the effective date's request time (strict next
// month — enabling on the 1st still defers to the following month).
func (pe *ProjectEmployee) RequestCheckInEnable(effectiveDate time.Time) error {
	if !pe.IsCurrentlyAssigned() {
		return NewValidationError("Chỉ có thể bật chấm công cho nhân viên đang làm việc")
	}

	if pe.CheckInEnabled {
		return NewValidationError("Nhân viên đã được bật chấm công")
	}

	if pe.HasPendingCheckInEnable() {
		return NewValidationError("Đã có yêu cầu bật chấm công đang chờ kích hoạt")
	}

	enabled := true
	pe.PendingCheckInEnabled = &enabled
	pe.CheckInEffectiveFrom = &effectiveDate

	return nil
}

// ApplyPendingCheckIn activates the pending check-in enable if the effective
// date has arrived. Returns true when the row changed.
func (pe *ProjectEmployee) ApplyPendingCheckIn() bool {
	if !pe.HasPendingCheckInEnable() {
		return false
	}

	now := clock.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if !pe.CheckInEffectiveFrom.After(startOfToday) {
		pe.CheckInEnabled = true
		pe.PendingCheckInEnabled = nil
		pe.CheckInEffectiveFrom = nil
		return true
	}

	return false
}

// CancelPendingCheckInEnable cancels a pending check-in enable
func (pe *ProjectEmployee) CancelPendingCheckInEnable() error {
	if !pe.HasPendingCheckInEnable() {
		return NewValidationError("Không có yêu cầu bật chấm công nào đang chờ kích hoạt")
	}

	pe.PendingCheckInEnabled = nil
	pe.CheckInEffectiveFrom = nil

	return nil
}

// IsWeekly returns true if employee is on weekly payment schedule
func (pe *ProjectEmployee) IsWeekly() bool {
	return pe.PaymentSchedule == string(PaymentScheduleWeekly)
}

// IsMonthly returns true if employee is on monthly payment schedule
func (pe *ProjectEmployee) IsMonthly() bool {
	return pe.PaymentSchedule == string(PaymentScheduleMonthly)
}

// Additional types for repository operations

// ProjectAssignmentSummary represents assignment summary for a project
type ProjectAssignmentSummary struct {
	ProjectID         uint  `json:"project_id"`
	TotalAssignments  int64 `json:"total_assignments"`
	ActiveAssignments int64 `json:"active_assignments"`
	EndedAssignments  int64 `json:"ended_assignments"`
}
