package domain

import (
	"context"
	"time"
)

// SelfCheckInAdvanceablePercent is the percent of earned salary (from check-in/out)
// that a self-check-in employee may take as an advance. Centralized so accumulation
// (repo), request validation, and UI display all agree.
// For the self-check-in flow: max_adv_amount = floor(salary * percent / 100).
const SelfCheckInAdvanceablePercent uint64 = 70

// QuotaCreditHoldDuration is how long a self-check-out earning stays pending
// before it is banked into the advance-payment quota pool. Check-out enqueues a
// deferred credit task at checkOutTime + QuotaCreditHoldDuration; admin manual
// approvals credit immediately and bypass this hold. Centralized alongside the
// other self-check-in advance-policy knobs.
const QuotaCreditHoldDuration = 24 * time.Hour

// AdvancePayment represents monthly advance payment limits from Flexible Payroll Template uploads
type AdvancePayment struct {
	ID           uint   `json:"id" gorm:"primaryKey;type:bigint unsigned"`
	ProjectID    uint   `json:"project_id" gorm:"not null;type:bigint unsigned;index"`
	EmployeeID   uint   `json:"employee_id" gorm:"not null;type:bigint unsigned;index"`
	ForMonth     string `json:"for_month" gorm:"size:7;not null;index"` // YYYY-MM
	UploadDate   string `json:"upload_date" gorm:"size:10;not null"`    // YYYY-MM-DD
	MaxAdvAmount uint64 `json:"max_adv_amount" gorm:"type:bigint unsigned;not null"`
	// Salary is the total earned wages from check-in/out (100%) for the self-check-in flow.
	// MaxAdvAmount = floor(Salary * SelfCheckInAdvanceablePercent / 100). Unused by the
	// admin-upload (BCC) flow, which sets MaxAdvAmount directly.
	Salary             uint64    `json:"salary" gorm:"type:bigint unsigned;not null;default:0"`
	LastAppliedAssetID *uint     `json:"last_applied_asset_id,omitempty" gorm:"type:bigint unsigned"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	// Relationships
	Project  Project  `json:"project" gorm:"foreignKey:ProjectID;references:ID"`
	Employee Employee `json:"employee" gorm:"foreignKey:EmployeeID;references:ID"`
}

// TableName returns the table name for AdvancePayment
func (AdvancePayment) TableName() string {
	return "advance_payments"
}

// ValidateForMonth validates the for_month field
func (ap *AdvancePayment) ValidateForMonth() error {
	if ap.ForMonth == "" {
		return NewValidationError("Tháng là bắt buộc")
	}
	if len(ap.ForMonth) != 7 {
		return NewValidationError("Định dạng tháng không hợp lệ (YYYY-MM)")
	}
	return nil
}

// IsValid validates the entire AdvancePayment entity
func (ap *AdvancePayment) IsValid() error {
	return ap.ValidateForMonth()
}

// AdvancePaymentRepository defines the interface for advance payment persistence operations
type AdvancePaymentRepository interface {
	Create(ctx context.Context, ap *AdvancePayment) error
	Upsert(ctx context.Context, ap *AdvancePayment) error
	GetByID(ctx context.Context, id uint64) (*AdvancePayment, error)
	GetByEmployeeAndMonth(ctx context.Context, employeeID uint64, forMonth string) ([]*AdvancePayment, error)
	GetMonthsByEmployee(ctx context.Context, employeeID uint64) ([]string, error)
	SumMaxAdvByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (uint64, error)
	SumSalaryAndMaxAdvByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (salary, maxAdv uint64, err error)
	// SumPendingEarningsByEmployeeMonth returns the total earning_amount held in
	// the 24h credit window for the employee's month: checked-out attendances
	// whose earning has not yet been banked into the quota pool
	// (quota_credited_at IS NULL, earning_amount > 0), scoped by check-out month.
	// Displayed separately from Salary (already-credited) on the self-check-in
	// advance screen so the worker can see money is coming; it is NOT part of the
	// 70% advanceable cap.
	SumPendingEarningsByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (uint64, error)
	BatchCreate(ctx context.Context, aps []*AdvancePayment) error
	BatchUpsert(ctx context.Context, aps []*AdvancePayment) error
	Update(ctx context.Context, ap *AdvancePayment) error
	AccumulateSalary(ctx context.Context, id uint64, earning int64) error
	ZeroOutQuota(ctx context.Context, projectID, employeeID uint, currentMonth string) error
	BatchZeroOutQuota(ctx context.Context, projectID uint, employeeIDs []uint, currentMonth string) error
	GetLatestForMonth(ctx context.Context) (string, error)
	GetEmployeeAdvanceStats(ctx context.Context, filters EmployeeAdvanceStatsFilters) ([]*EmployeeAdvanceStats, int64, error)
	GetAvailableMonths(ctx context.Context) ([]*AvailableMonth, error)
	GetEmployeeByID(ctx context.Context, employeeID uint64) (*Employee, error)
	// GetEmployeesByIDs batch-fetches employees in one query. Used by the sao-ke
	// export to avoid an N+1 of GetEmployeeByID per (employee, project) pair.
	GetEmployeesByIDs(ctx context.Context, employeeIDs []uint64) (map[uint64]*Employee, error)
	HasDataForMonth(ctx context.Context, forMonth string) (bool, error)
	// GetQuotaAnomalies returns quota rows that violate the named invariant.
	// anomalyType is one of: "drift" (max_adv != floor(salary*70/100)),
	// "missing" (earning>0 attendance but no advance_payments row), or
	// "stale" (salary>0 but assignment check_in_enabled=false).
	GetQuotaAnomalies(ctx context.Context, forMonth, anomalyType string) ([]QuotaAnomaly, error)
	// CountQuotaAnomalies returns the count of rows violating the named invariant.
	CountQuotaAnomalies(ctx context.Context, forMonth, anomalyType string) (int, error)
	// SumSalaryAndMaxAdvForMonth returns the total salary and max_adv_amount across
	// all advance_payments rows for the given month (the throughput tile B4).
	SumSalaryAndMaxAdvForMonth(ctx context.Context, forMonth string) (salary, maxAdv uint64, err error)
}

// QuotaAnomaly represents a quota row that violates an expected invariant.
type QuotaAnomaly struct {
	EmployeeID   uint   `json:"employee_id" gorm:"column:employee_id"`
	ProjectID    uint   `json:"project_id" gorm:"column:project_id"`
	ForMonth     string `json:"for_month" gorm:"column:for_month"`
	Salary       int64  `json:"salary" gorm:"column:salary"`
	MaxAdvAmount int64  `json:"max_adv_amount" gorm:"column:max_adv_amount"`
	Reason       string `json:"reason" gorm:"column:reason"`
}

// AvailableMonth represents a month with flex pay data
type AvailableMonth struct {
	ForMonth      string `json:"for_month"`
	EmployeeCount int    `json:"employee_count"`
}

// AdvancePaymentSummary represents summary data for an employee's advance payments
type AdvancePaymentSummary struct {
	TotalMaxAdvance     uint64 `json:"total_max_advance"`
	CompletedAmount     uint64 `json:"completed_amount"`
	PendingAmount       uint64 `json:"pending_amount"`
	RemainingAmount     uint64 `json:"remaining_amount"`
	CurrentMonth        string `json:"current_month"`
	CanRequest          bool   `json:"can_request"`
	HasFlexibleSchedule bool   `json:"has_flexible_schedule"`
}

// EmployeeAdvanceInfo represents advance payment info for an employee (used by employee endpoints)
type AdvancePaymentQuota struct {
	ForMonth         string `json:"for_month"`
	MaxAdvanceAmount uint64 `json:"max_advance_amount"`
	CompletedAmount  uint64 `json:"completed_amount"`
	PendingAmount    uint64 `json:"pending_amount"`
	RemainingAmount  uint64 `json:"remaining_amount"`
}

type EmployeeAdvanceInfo struct {
	TotalMaxAdvance     uint64                `json:"total_max_advance"`
	CompletedAmount     uint64                `json:"completed_amount"`
	PendingAmount       uint64                `json:"pending_amount"`
	RemainingAmount     uint64                `json:"remaining_amount"`
	CurrentMonth        string                `json:"current_month"`
	CanRequest          bool                  `json:"can_request"`
	CanRequestTitle     string                `json:"can_request_title,omitempty"`
	CanRequestReason    string                `json:"can_request_reason,omitempty"`
	HasFlexibleSchedule bool                  `json:"has_flexible_schedule"`
	Quotas              []AdvancePaymentQuota `json:"quotas"`
}

// EmployeeAdvanceStats represents an employee with their advance payment statistics
type EmployeeAdvanceStats struct {
	EmployeeID             uint       `json:"employee_id" gorm:"column:employee_id"`
	Fullname               string     `json:"fullname" gorm:"column:fullname"`
	CCCD                   string     `json:"cccd" gorm:"column:cccd"`
	Username               string     `json:"username" gorm:"column:username"`
	Email                  *string    `json:"email" gorm:"column:email"`
	Mobile                 string     `json:"mobile" gorm:"column:mobile"`
	BankID                 *uint      `json:"bank_id" gorm:"column:bank_id"`
	BankName               *string    `json:"bank_name" gorm:"column:bank_name"`
	BankAccountNumber      string     `json:"bank_account_number" gorm:"column:bank_account_number"`
	BankAccountName        string     `json:"bank_account_name" gorm:"column:bank_account_name"`
	ProjectID              uint       `json:"project_id" gorm:"column:project_id"`
	ProjectName            string     `json:"project_name" gorm:"column:project_name"`
	ProjectCode            string     `json:"project_code" gorm:"column:project_code"`
	ForMonth               string     `json:"for_month" gorm:"column:for_month"`
	MaxAdvanceAmount       uint64     `json:"max_advance_amount" gorm:"column:max_advance_amount"`
	UtilizedAmount         uint64     `json:"utilized_amount" gorm:"column:utilized_amount"`
	TotalFeeGenerated      uint64     `json:"total_fee_generated" gorm:"column:total_fee_generated"`
	PendingAmount          uint64     `json:"pending_amount" gorm:"column:pending_amount"`
	CompletedRequestsCount int        `json:"completed_requests_count" gorm:"column:completed_requests_count"`
	PendingRequestsCount   int        `json:"pending_requests_count" gorm:"column:pending_requests_count"`
	CreatedAt              *time.Time `json:"created_at" gorm:"column:created_at"`
	ProjectEmployeeID      uint       `json:"project_employee_id" gorm:"column:project_employee_id"`
	CheckInEnabled         bool       `json:"check_in_enabled" gorm:"column:check_in_enabled"`
}

// EmployeeAdvanceStatsFilters represents filters for querying employee advance stats
type EmployeeAdvanceStatsFilters struct {
	ForMonth  *string
	Search    string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}
