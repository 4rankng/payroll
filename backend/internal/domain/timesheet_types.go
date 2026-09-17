package domain

import (
	"context"
	"time"
)

// TimesheetRepository defines the interface for timesheet persistence operations
type TimesheetRepository interface {
	Create(ctx context.Context, timesheet *Timesheet) error
	GetByID(ctx context.Context, id uint) (*Timesheet, error)
	GetByIDs(ctx context.Context, ids []uint) ([]*Timesheet, error)
	GetByIDsWithoutRelations(ctx context.Context, ids []uint) ([]*Timesheet, error)
	// GetTimesheetDatesByIDs returns id → date for the given IDs without loading
	// full rows or any relationships. Callers that only need the date (e.g. the
	// bank-transfer-histories cycle resolution) use this to avoid SELECT * and
	// relationship preloads on thousands of rows.
	GetTimesheetDatesByIDs(ctx context.Context, ids []uint) (map[uint]time.Time, error)
	Update(ctx context.Context, timesheet *Timesheet) error
	Delete(ctx context.Context, id uint) error
	HardDelete(ctx context.Context, id uint) error
	DeleteByProjectID(ctx context.Context, projectID uint) error
	List(ctx context.Context, filters TimesheetFilters) ([]*Timesheet, error)
	Count(ctx context.Context, filters TimesheetFilters) (int64, error)
	GetByProjectAndEmployee(ctx context.Context, projectID, employeeID uint, fromDate, toDate time.Time) ([]*Timesheet, error)
	GetByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) ([]*Timesheet, error)
	GetByEmployee(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*Timesheet, error)
	BulkCreate(ctx context.Context, timesheets []*Timesheet) error
	BulkUpdate(ctx context.Context, timesheets []*Timesheet) error
	Approve(ctx context.Context, id uint, approvedBy uint) error
	BulkApprove(ctx context.Context, ids []uint, approvedBy uint) error
	BulkReject(ctx context.Context, ids []uint, rejectionReason string) error
	Reject(ctx context.Context, id uint, rejectionReason string) error
	Reset(ctx context.Context, id uint) error
	// ResetAllApproved resets every approved, unpaid timesheet back to pending
	// approval (admin mass un-approval). Paid timesheets are never touched.
	// Returns the number of timesheets reset.
	ResetAllApproved(ctx context.Context) (int64, error)
	GetSummaryByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) (*TimesheetSummary, error)
	GetSummaryStats(ctx context.Context, filters TimesheetFilters) (*TimesheetSummaryStats, error)
	GetByProjectEmployeeDatePaytype(ctx context.Context, projectID, employeeID uint, date time.Time, paytype string) (*Timesheet, error)
	GetByProjectEmployeeDateHourType(ctx context.Context, projectID, employeeID uint, date time.Time, hourType string) (*Timesheet, error)
	GetPaidTimesheetsInDateRange(ctx context.Context, startDate, endDate time.Time, timesheets *[]*Timesheet) error
	CountDistinctEmployeesPaidInPeriod(ctx context.Context, startDate, endDate time.Time) (int, error)
	GetByProjectEmployeeDate(ctx context.Context, projectID, employeeID uint, date time.Time) ([]*Timesheet, error)
	GetPendingApprovalCount(ctx context.Context, startDate, endDate time.Time) (int, error)
	GetApprovedSalaryTotal(ctx context.Context, startDate, endDate time.Time) (int64, error)
	GetByEmployeeAndPeriod(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*Timesheet, error)
	BulkUpdatePaymentStatus(ctx context.Context, updates []PaymentStatusUpdate) error
	CountDistinctWorkingDays(ctx context.Context, employeeID, projectID uint, fromDate, toDate time.Time) (int, error)
	GetProjectsForEmployeeOnDate(ctx context.Context, employeeID uint, date time.Time) ([]uint, error)
	// Bulk query method for validation optimization
	GetByEmployeeDateCombos(ctx context.Context, combos []EmployeeDateCombo) ([]*Timesheet, error)

	// New methods for assignment conflict resolution
	GetLatestTimesheetDate(ctx context.Context, projectID, employeeID uint) (*time.Time, error)
	CountTimesheets(ctx context.Context, projectID, employeeID uint) (int64, error)
	CountTimesheetsByEmployeeID(ctx context.Context, employeeID uint) (int64, error)
	// GetLatestTimesheetDateByEmployeeID returns the employee's most recent
	// timesheet date across all projects, or nil when they have none. Used to
	// suggest assignment start dates that continue coverage without day overlap.
	GetLatestTimesheetDateByEmployeeID(ctx context.Context, employeeID uint) (*time.Time, error)
	// HasProtectedTimesheetsByEmployeeID reports whether the employee has a
	// financial or approved/rejected-payroll record that must remain auditable.
	// A bank-result worker can settle exported rows after an Admin rejection.
	HasProtectedTimesheetsByEmployeeID(ctx context.Context, employeeID uint) (bool, error)
	HardDeleteOperationalByEmployeeID(ctx context.Context, employeeID uint) error
	HasTimesheetsAfterDate(ctx context.Context, projectID, employeeID uint, date time.Time) (bool, error)
	HasTimesheetsForAssignment(ctx context.Context, projectID, employeeID uint) (bool, error)
	HasNonEditableTimesheetsAfterDate(ctx context.Context, projectID, employeeID uint, date time.Time) (bool, error)
	HasNonEditableTimesheetsInRange(ctx context.Context, projectID, employeeID uint, from, to *time.Time) (bool, error)
	SetForcePayroll(ctx context.Context, id uint, flag bool) error

	// Payment history methods
	GetPaymentHistories(ctx context.Context, filters PaymentHistoryFilters) ([]*PaymentHistory, int64, error)

	// Dashboard summary methods
	GetTotalPaidSalary(ctx context.Context) (int64, error)
	GetPendingSalaryForMonth(ctx context.Context, startDate, endDate time.Time) (int64, error)
	GetPendingSalaryForMonthBySchedule(ctx context.Context, startDate, endDate time.Time, schedule PaymentSchedule) (int64, error)
	GetPaidSalaryForMonth(ctx context.Context, startDate, endDate time.Time) (int64, error)
	CountEmployeesPaidInLast30Days(ctx context.Context) (int, error)
	GetFirstPaidTimesheetDate(ctx context.Context) (*time.Time, error)
	GetPaidSalaryByEmployee(ctx context.Context, startDate, endDate time.Time) (map[uint]int64, error)

	// Bulk transfer result processing methods
	BatchUpdatePaymentStatusToFailed(ctx context.Context, tx interface{}, timesheetIDs []uint) error

	// Revenue tracking methods
	BulkUpdateRevenuePaid(ctx context.Context, timesheetIDs []uint) error
	BulkUpdateRevenueReceivable(ctx context.Context, updates map[uint]int64) error
	BulkUpdateTransactionID(ctx context.Context, transactionID uint, timesheetIDs []uint) error
	GetByTransactionID(ctx context.Context, transactionID uint) ([]*Timesheet, error)

	// Optimized aggregation methods for employee summaries
	GetEmployeeTimesheetSummaryAggregated(ctx context.Context, employeeID uint) (*TimesheetSummaryAggregated, error)
	GetEmployeeCurrentWeekHours(ctx context.Context, employeeID uint) (float64, error)
	GetEmployeeMonthlyPayrollSummary(ctx context.Context, employeeID uint) (*PayrollSummaryAggregated, error)

	// Entry table method for specialized reporting
	GetEntryTableData(ctx context.Context, projectID uint, employeeIDs []uint, fromDate, toDate time.Time) ([]*Timesheet, error)

	// Grouped query methods for partner view with server-side employee pagination
	ListGroupedByEmployee(ctx context.Context, filters TimesheetFilters) ([]EmployeeGroupResult, []*Timesheet, int64, error)
	CountDistinctEmployees(ctx context.Context, filters TimesheetFilters) (int64, error)

	// Salary period change validation
	CountUnsettledPaidTimesheets(ctx context.Context, projectID uint) (int64, error)

	// Cash-readiness forecast: point-in-time facts used to reconstruct approved,
	// pending, and not-yet-created target-Ky exposure. Scoped identically to
	// GetSummaryStats; payment status is deliberately not part of this query.
	GetAccrualCohort(ctx context.Context, filters TimesheetFilters) ([]TimesheetAccrualDailyRow, error)
}

// TimesheetAccrualDailyRow is a row-level, read-only forecast fact. CreatedAt
// and ApprovedAt allow leakage-free point-in-time reconstruction; employee and
// project identifiers support headcount normalization and subgroup fallback.
// ApprovedDate is retained for source compatibility with the v1 forecast tests
// and is populated alongside ApprovedAt by persistence.
type TimesheetAccrualDailyRow struct {
	WorkDate     time.Time       `gorm:"column:work_date" json:"work_date"`
	CreatedAt    time.Time       `gorm:"column:created_at" json:"created_at"`
	ApprovedAt   *time.Time      `gorm:"column:approved_at" json:"approved_at"`
	ApprovedDate time.Time       `gorm:"column:approved_date" json:"approved_date"`
	Status       TimesheetStatus `gorm:"column:timesheet_status" json:"timesheet_status"`
	EmployeeID   uint            `gorm:"column:employee_id" json:"employee_id"`
	ProjectID    uint            `gorm:"column:project_id" json:"project_id"`
	Amount       int64           `gorm:"column:amount" json:"amount"`
}

// Narrow interfaces for interface segregation — each consumer depends only on what it needs.
// The concrete TimesheetRepository struct implicitly satisfies all narrow interfaces (Go structural typing).

// TimesheetReader provides read-only timesheet queries.
type TimesheetReader interface {
	GetByID(ctx context.Context, id uint) (*Timesheet, error)
	GetByIDs(ctx context.Context, ids []uint) ([]*Timesheet, error)
	GetByIDsWithoutRelations(ctx context.Context, ids []uint) ([]*Timesheet, error)
	GetTimesheetDatesByIDs(ctx context.Context, ids []uint) (map[uint]time.Time, error)
	List(ctx context.Context, filters TimesheetFilters) ([]*Timesheet, error)
	Count(ctx context.Context, filters TimesheetFilters) (int64, error)
	GetByProjectAndEmployee(ctx context.Context, projectID, employeeID uint, fromDate, toDate time.Time) ([]*Timesheet, error)
	GetByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) ([]*Timesheet, error)
	GetByEmployee(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*Timesheet, error)
	GetByEmployeeAndPeriod(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*Timesheet, error)
	GetByProjectEmployeeDate(ctx context.Context, projectID, employeeID uint, date time.Time) ([]*Timesheet, error)
	GetByTransactionID(ctx context.Context, transactionID uint) ([]*Timesheet, error)
}

// TimesheetWriter provides create/update/delete operations.
type TimesheetWriter interface {
	Create(ctx context.Context, timesheet *Timesheet) error
	Update(ctx context.Context, timesheet *Timesheet) error
	Delete(ctx context.Context, id uint) error
	HardDelete(ctx context.Context, id uint) error
	DeleteByProjectID(ctx context.Context, projectID uint) error
	BulkCreate(ctx context.Context, timesheets []*Timesheet) error
	BulkUpdate(ctx context.Context, timesheets []*Timesheet) error
}

// TimesheetApprover handles approval workflow operations.
type TimesheetApprover interface {
	Approve(ctx context.Context, id uint, approvedBy uint) error
	BulkApprove(ctx context.Context, ids []uint, approvedBy uint) error
	BulkReject(ctx context.Context, ids []uint, rejectionReason string) error
	Reject(ctx context.Context, id uint, rejectionReason string) error
	Reset(ctx context.Context, id uint) error
}

// TimesheetUnpaidRejector performs the filtered Admin rejection command
// without expanding every timesheet consumer's repository contract.
type TimesheetUnpaidRejector interface {
	RejectUnpaidByProjectDateRange(ctx context.Context, projectID uint, fromDate, toDate time.Time, rejectionReason string, rejectedBy uint) (int64, error)
}

// TimesheetEditRequestResetter atomically resets an approved timesheet only
// while it still owns the pending edit-request link being approved.
type TimesheetEditRequestResetter interface {
	ResetForApprovedEditRequest(ctx context.Context, timesheetID, requestID uint) error
}

// TimesheetPaymentUpdater handles payment status transitions.
type TimesheetPaymentUpdater interface {
	BulkUpdatePaymentStatus(ctx context.Context, updates []PaymentStatusUpdate) error
	BatchUpdatePaymentStatusToFailed(ctx context.Context, tx interface{}, timesheetIDs []uint) error
	GetPaidTimesheetsInDateRange(ctx context.Context, startDate, endDate time.Time, timesheets *[]*Timesheet) error
	GetPaymentHistories(ctx context.Context, filters PaymentHistoryFilters) ([]*PaymentHistory, int64, error)
}

// TimesheetValidator provides conflict detection for timesheet creation.
type TimesheetValidator interface {
	GetByProjectEmployeeDatePaytype(ctx context.Context, projectID, employeeID uint, date time.Time, paytype string) (*Timesheet, error)
	GetByProjectEmployeeDateHourType(ctx context.Context, projectID, employeeID uint, date time.Time, hourType string) (*Timesheet, error)
	GetByEmployeeDateCombos(ctx context.Context, combos []EmployeeDateCombo) ([]*Timesheet, error)
	CountDistinctWorkingDays(ctx context.Context, employeeID, projectID uint, fromDate, toDate time.Time) (int, error)
	GetProjectsForEmployeeOnDate(ctx context.Context, employeeID uint, date time.Time) ([]uint, error)
	CountUnsettledPaidTimesheets(ctx context.Context, projectID uint) (int64, error)
}

// TimesheetAssignmentChecker detects assignment conflicts.
type TimesheetAssignmentChecker interface {
	GetLatestTimesheetDate(ctx context.Context, projectID, employeeID uint) (*time.Time, error)
	CountTimesheets(ctx context.Context, projectID, employeeID uint) (int64, error)
	CountTimesheetsByEmployeeID(ctx context.Context, employeeID uint) (int64, error)
	HasTimesheetsAfterDate(ctx context.Context, projectID, employeeID uint, date time.Time) (bool, error)
	HasTimesheetsForAssignment(ctx context.Context, projectID, employeeID uint) (bool, error)
	HasNonEditableTimesheetsAfterDate(ctx context.Context, projectID, employeeID uint, date time.Time) (bool, error)
	HasNonEditableTimesheetsInRange(ctx context.Context, projectID, employeeID uint, from, to *time.Time) (bool, error)
}

// TimesheetDashboardReader provides dashboard summary queries.
type TimesheetDashboardReader interface {
	GetTotalPaidSalary(ctx context.Context) (int64, error)
	GetPendingSalaryForMonth(ctx context.Context, startDate, endDate time.Time) (int64, error)
	GetPendingSalaryForMonthBySchedule(ctx context.Context, startDate, endDate time.Time, schedule PaymentSchedule) (int64, error)
	GetPaidSalaryForMonth(ctx context.Context, startDate, endDate time.Time) (int64, error)
	CountEmployeesPaidInLast30Days(ctx context.Context) (int, error)
	GetFirstPaidTimesheetDate(ctx context.Context) (*time.Time, error)
	GetPaidSalaryByEmployee(ctx context.Context, startDate, endDate time.Time) (map[uint]int64, error)
	GetPendingApprovalCount(ctx context.Context, startDate, endDate time.Time) (int, error)
	GetApprovedSalaryTotal(ctx context.Context, startDate, endDate time.Time) (int64, error)
	CountDistinctEmployeesPaidInPeriod(ctx context.Context, startDate, endDate time.Time) (int, error)
}

// TimesheetRevenueUpdater handles revenue tracking updates.
type TimesheetRevenueUpdater interface {
	BulkUpdateRevenuePaid(ctx context.Context, timesheetIDs []uint) error
	BulkUpdateRevenueReceivable(ctx context.Context, updates map[uint]int64) error
	BulkUpdateTransactionID(ctx context.Context, transactionID uint, timesheetIDs []uint) error
}

// TimesheetReportingReader provides reporting and aggregation queries.
type TimesheetReportingReader interface {
	GetSummaryByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) (*TimesheetSummary, error)
	GetSummaryStats(ctx context.Context, filters TimesheetFilters) (*TimesheetSummaryStats, error)
	GetEmployeeTimesheetSummaryAggregated(ctx context.Context, employeeID uint) (*TimesheetSummaryAggregated, error)
	GetEmployeeCurrentWeekHours(ctx context.Context, employeeID uint) (float64, error)
	GetEmployeeMonthlyPayrollSummary(ctx context.Context, employeeID uint) (*PayrollSummaryAggregated, error)
	ListGroupedByEmployee(ctx context.Context, filters TimesheetFilters) ([]EmployeeGroupResult, []*Timesheet, int64, error)
	CountDistinctEmployees(ctx context.Context, filters TimesheetFilters) (int64, error)
	GetEntryTableData(ctx context.Context, projectID uint, employeeIDs []uint, fromDate, toDate time.Time) ([]*Timesheet, error)
}

// EmployeeGroupResult represents the result of employee grouping query
type EmployeeGroupResult struct {
	EmployeeID   uint    `json:"employee_id"`
	EmployeeName string  `json:"employee_name"`
	EmployeeCode string  `json:"employee_code"`
	TotalHours   float64 `json:"total_hours"`
	TotalAmount  float64 `json:"total_amount"`
	EntryCount   int     `json:"entry_count"`
}

// TimesheetFilters represents filtering options for timesheet queries
type TimesheetFilters struct {
	ProjectIDs                []uint // Filter by multiple project IDs (empty = all projects)
	EmployeeID                *uint
	EmployeeIDs               []uint // Filter by multiple employee IDs (empty = all)
	PayrateID                 *uint
	TimesheetStatus           []TimesheetStatus
	PaymentStatus             []PaymentStatus
	PayType                   []string
	CreatedBy                 *uint
	ApprovedBy                *uint
	EmployeeCreatedBy         *uint      // Filter by who created the employee
	EmployeeAssignedByPartner *uint      // Filter by who assigned the employee to project (for partner access control)
	Date                      *time.Time // Exact date match
	FromDate                  *time.Time // Date range start
	ToDate                    *time.Time // Date range end
	PendingApproval           bool
	Limit                     int
	Offset                    int
	SortBy                    string
	SortOrder                 string
	ForcePayroll              *bool  // filter by admin inclusion flag
	AllowedEdit               *bool  // filter by allowed_edit flag
	RequestEditID             *uint  // filter by specific edit request
	HasRequestEdit            *bool  // filter by presence of pending edit request
	SkipRelations             bool   // skip loading relationships for performance
	Search                    string // search by employee fullname or code
}

// NewPendingPaymentTimesheetFilters returns the canonical cohort that is ready
// for payroll: approved timesheets that have not been paid or whose payment
// attempt failed and can be retried.
func NewPendingPaymentTimesheetFilters() TimesheetFilters {
	return TimesheetFilters{
		TimesheetStatus: []TimesheetStatus{TimesheetStatusApproved},
		PaymentStatus: []PaymentStatus{
			PaymentStatusPending,
			PaymentStatusFailed,
		},
	}
}

// NewOperationalPaymentBacklogTimesheetFilters returns all accrued salary that
// still needs operational processing. Pending-approval rows belong in the
// dashboard backlog, but remain excluded from the payroll-ready cohort above.
func NewOperationalPaymentBacklogTimesheetFilters() TimesheetFilters {
	return TimesheetFilters{
		TimesheetStatus: []TimesheetStatus{
			TimesheetStatusPendingApproval,
			TimesheetStatusApproved,
		},
		PaymentStatus: []PaymentStatus{
			PaymentStatusPending,
			PaymentStatusFailed,
		},
	}
}

// GetStatuses implements common.StatusFilter interface for TimesheetStatus
func (tf *TimesheetFilters) GetStatuses() []string {
	if len(tf.TimesheetStatus) == 0 {
		return nil
	}
	statuses := make([]string, len(tf.TimesheetStatus))
	for i, status := range tf.TimesheetStatus {
		statuses[i] = string(status)
	}
	return statuses
}

// GetStatusField implements common.StatusFilter interface
func (tf *TimesheetFilters) GetStatusField() string {
	return "timesheet_status"
}

// GetFromDate implements common.DateRangeFilter interface
func (tf *TimesheetFilters) GetFromDate() *time.Time {
	return tf.FromDate
}

// GetToDate implements common.DateRangeFilter interface
func (tf *TimesheetFilters) GetToDate() *time.Time {
	return tf.ToDate
}

// GetDateField implements common.DateRangeFilter interface
func (tf *TimesheetFilters) GetDateField() string {
	return "date"
}

// UsesCalendarDates identifies timesheets.date as an SQL DATE column. The
// persistence filter builder binds its range as timezone-free YYYY-MM-DD
// values, preventing driver/location conversion from dropping boundary days.
func (tf *TimesheetFilters) UsesCalendarDates() bool {
	return true
}

// GetCreatedBy implements common.CreatorFilter interface
func (tf *TimesheetFilters) GetCreatedBy() *uint {
	return tf.CreatedBy
}

// GetCreatorField implements common.CreatorFilter interface
func (tf *TimesheetFilters) GetCreatorField() string {
	return "created_by"
}

// GetSearch implements common.SearchFilter interface
func (tf *TimesheetFilters) GetSearch() string {
	return tf.Search
}

// GetSearchFields implements common.SearchFilter interface
func (tf *TimesheetFilters) GetSearchFields() []string {
	return []string{"e.fullname", "e.code"}
}

// TimesheetSummary represents aggregated timesheet metrics
type TimesheetSummary struct {
	TotalHours      float64            `json:"total_hours"`
	TotalAmount     int64              `json:"total_amount"`
	HoursByPayType  map[string]float64 `json:"hours_by_pay_type"`
	AmountByPayType map[string]int64   `json:"amount_by_pay_type"`
	EmployeeCount   int                `json:"employee_count"`
	EntryCount      int                `json:"entry_count"`
}

// TimesheetSummaryStats represents summary statistics for timesheets
type TimesheetSummaryStats struct {
	TotalEntries         int       `json:"totalEntries"`
	TotalEmployees       int       `json:"totalEmployees"`
	PendingApproval      int       `json:"pendingApproval"`
	PendingEmployees     int       `json:"pendingEmployees"`
	PendingPaymentAmount int64     `json:"pendingPaymentAmount"`
	ApprovedEntries      int       `json:"approvedEntries"`
	PaidEntries          int       `json:"paidEntries"`
	PaidAmount           int64     `json:"paidAmount"`
	PaidEmployees        int       `json:"paidEmployees"`
	RejectedEntries      int       `json:"rejectedEntries"`
	LastUpdated          time.Time `json:"lastUpdated"`
}

// PaymentHistory represents aggregated payment history for an employee
type PaymentHistory struct {
	EmployeeID      uint
	EmployeeName    string
	EmployeeCCCD    string
	ProjectID       uint
	ProjectName     string
	Position        string
	TotalPaidAmount int64
	PaidAt          time.Time
}

// PaymentHistoryFilters represents filtering options for payment history queries
type PaymentHistoryFilters struct {
	EmployeeCreatedBy         *uint      // Filter by who created the employee (for partner access control)
	EmployeeAssignedByPartner *uint      // Filter by who assigned the employee to project (for partner access control)
	ProjectIDs                []uint     // Filter by project IDs
	EmployeeIDs               []uint     // Filter by employee IDs
	Position                  string     // Filter by position
	Search                    string     // Search across employee fullname and CCCD
	FromDate                  *time.Time // Payment date range start
	ToDate                    *time.Time // Payment date range end
	SortBy                    string     // Field to sort by
	SortOrder                 string     // Sort direction
	Limit                     int        // Pagination limit
	Offset                    int        // Pagination offset
}

// BulkOperationResult represents the result of a bulk operation
type BulkOperationResult struct {
	Approved          int              `json:"approved,omitempty"`
	Rejected          int              `json:"rejected,omitempty"`
	Skipped           int              `json:"skipped,omitempty"`
	Failed            int              `json:"failed,omitempty"`
	NotificationsSent int              `json:"notifications_sent,omitempty"`
	Results           []BulkItemResult `json:"results"`
}

// BulkItemResult represents the result of a single item in bulk operation
type BulkItemResult struct {
	ID              uint   `json:"id"`
	Status          string `json:"status,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	Error           string `json:"error,omitempty"`
}

// PaymentStatusUpdate represents an update to payment status for a timesheet
type PaymentStatusUpdate struct {
	TimesheetID      uint          `json:"timesheet_id"`
	PaymentStatus    PaymentStatus `json:"payment_status"`
	PaymentReference *string       `json:"payment_reference,omitempty"`
	PaymentDate      *time.Time    `json:"payment_date,omitempty"`
	PaidAmount       *int64        `json:"paid_amount,omitempty"`
	PaidAt           *time.Time    `json:"paid_at,omitempty"`
}

// EmployeeTimesheetSummary represents timesheet summary for an employee
type EmployeeTimesheetSummary struct {
	EmployeeID         uint               `json:"employeeId"`
	EmployeeName       string             `json:"employeeName"`
	TotalHours         map[string]float64 `json:"totalHours"`
	TotalAmount        int64              `json:"totalAmount"`
	AverageHoursPerDay float64            `json:"averageHoursPerDay"`
	WorkingDays        int                `json:"workingDays"`
	LastEntryDate      *string            `json:"lastEntryDate"`
	CurrentWeekHours   float64            `json:"currentWeekHours"`
	PendingEntries     int                `json:"pendingEntries"`
	ApprovedEntries    int                `json:"approvedEntries"`
	RejectedEntries    int                `json:"rejectedEntries"`
	TotalEntries       int                `json:"totalEntries"`
}

// TimesheetSummaryAggregated represents aggregated timesheet summary data from database
type TimesheetSummaryAggregated struct {
	TotalEntries       int        `json:"totalEntries"`
	TotalHours         float64    `json:"totalHours"`
	TotalAmount        int64      `json:"totalAmount"`
	AverageHoursPerDay float64    `json:"averageHoursPerDay"`
	WorkingDays        int        `json:"workingDays"`
	LastEntryDate      *time.Time `json:"lastEntryDate"`
	PendingEntries     int        `json:"pendingEntries"`
	ApprovedEntries    int        `json:"approvedEntries"`
	RejectedEntries    int        `json:"rejectedEntries"`
}

// PayrollSummaryAggregated represents aggregated payroll summary data from database
type PayrollSummaryAggregated struct {
	TotalPayments     int        `json:"totalPayments"`
	TotalPaidAmount   int64      `json:"totalPaidAmount"`
	LastPaymentDate   *time.Time `json:"lastPaymentDate"`
	AvgWeeklyEarnings int64      `json:"avgWeeklyEarnings"`
}

// TimesheetWithDetails represents timesheet with full project and employee details
type TimesheetWithDetails struct {
	ID               uint            `json:"id"`
	ProjectID        uint            `json:"project_id"`
	EmployeeID       uint            `json:"employee_id"`
	PayrateID        uint            `json:"payrate_id"`
	Date             time.Time       `json:"date"`
	HoursWorked      float64         `json:"hours_worked"`
	PayType          string          `json:"paytype"`
	PayRate          int64           `json:"payrate"`
	Amount           int64           `json:"amount"`
	TimesheetStatus  TimesheetStatus `json:"timesheet_status"`
	PaymentStatus    PaymentStatus   `json:"payment_status"`
	PaymentReference *string         `json:"payment_reference"`
	PaymentDate      *time.Time      `json:"payment_date"`
	CreatedBy        uint            `json:"created_by"`
	ApprovedBy       *uint           `json:"approved_by"`
	ApprovedAt       *time.Time      `json:"approved_at"`
	RejectionReason  string          `json:"rejection_reason"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`

	// Project details
	ProjectName   string        `json:"project_name"`
	ProjectCode   string        `json:"project_code"`
	ProjectStatus ProjectStatus `json:"project_status"`
	ClientName    string        `json:"client_name"`

	// Employee details
	EmployeeFullname string `json:"employee_fullname"`
	EmployeeCCCD     string `json:"employee_cccd"`
	EmployeeCode     string `json:"employee_code"`

	// User details
	CreatedUserFullname  string `json:"created_user_fullname"`
	ApprovedUserFullname string `json:"approved_user_fullname"`

	// Payroll details
	PayrollBatchCode *string `json:"payroll_batch_code"`
}
