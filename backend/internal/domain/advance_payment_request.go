package domain

import (
	"context"
	"time"
)

// AdvancePaymentRequestStatus represents the status of an advance payment request
type AdvancePaymentRequestStatus string

const (
	AdvancePaymentStatusPending   AdvancePaymentRequestStatus = "PENDING"
	AdvancePaymentStatusApproved  AdvancePaymentRequestStatus = "APPROVED"
	AdvancePaymentStatusCancelled AdvancePaymentRequestStatus = "CANCELLED"
	AdvancePaymentStatusCompleted AdvancePaymentRequestStatus = "COMPLETED"
	AdvancePaymentStatusFailed    AdvancePaymentRequestStatus = "FAILED"
)

// AdvancePaymentRequest represents an individual advance payment request from an employee
type AdvancePaymentRequest struct {
	ID                      uint                        `json:"id" gorm:"primaryKey;type:bigint unsigned"`
	AdvPayID                uint                        `json:"adv_pay_id" gorm:"not null;type:bigint unsigned;index"`
	ProjectID               uint                        `json:"project_id" gorm:"not null;type:bigint unsigned;index"`
	EmployeeID              uint                        `json:"employee_id" gorm:"not null;type:bigint unsigned;index"`
	RequestAmount           uint64                      `json:"request_amount" gorm:"type:bigint unsigned;not null"`
	Fee                     uint64                      `json:"fee" gorm:"type:bigint unsigned;not null"`
	ProviderFee             uint64                      `json:"provider_fee" gorm:"type:bigint unsigned;not null;default:0"`
	NetAmount               uint64                      `json:"net_amount" gorm:"type:bigint unsigned;not null"`
	Status                  AdvancePaymentRequestStatus `json:"status" gorm:"size:20;not null;default:'PENDING';index"`
	PaymentReference        *string                     `json:"payment_reference" gorm:"size:255"`
	PaidAt                  *time.Time                  `json:"paid_at"`
	ReceivableSettledAt     *time.Time                  `json:"receivable_settled_at"`                                                                                             // When client paid the receivable
	SettlementTransactionID *uint                       `json:"settlement_transaction_id" gorm:"type:bigint unsigned;index;comment:'Links to transaction for tracing settlement'"` // Transaction that holds the receivable
	CreatedAt               time.Time                   `json:"created_at"`
	UpdatedAt               time.Time                   `json:"updated_at"`

	// Relationships
	AdvancePayment        *AdvancePayment `json:"advance_payment,omitempty" gorm:"foreignKey:AdvPayID"`
	Project               *Project        `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	Employee              *Employee       `json:"employee,omitempty" gorm:"foreignKey:EmployeeID"`
	SettlementTransaction *Transaction    `json:"settlement_transaction,omitempty" gorm:"foreignKey:SettlementTransactionID;references:ID"`
}

// TableName returns the table name for AdvancePaymentRequest
func (AdvancePaymentRequest) TableName() string {
	return "advance_payment_requests"
}

// ValidateRequestAmount validates the request amount
func (apr *AdvancePaymentRequest) ValidateRequestAmount() error {
	if apr.RequestAmount == 0 {
		return NewValidationError("Số tiền yêu cầu là bắt buộc")
	}
	if apr.RequestAmount < 10000 {
		return NewValidationError("Số tiền yêu cầu tối thiểu là 10,000 VND")
	}
	return nil
}

// IsValid validates the entire AdvancePaymentRequest entity
func (apr *AdvancePaymentRequest) IsValid() error {
	return apr.ValidateRequestAmount()
}

// IsPending returns true if the request is pending
func (apr *AdvancePaymentRequest) IsPending() bool {
	return apr.Status == AdvancePaymentStatusPending
}

// IsApproved returns true if the request is approved
func (apr *AdvancePaymentRequest) IsApproved() bool {
	return apr.Status == AdvancePaymentStatusApproved
}

// IsCompleted returns true if the request is completed
func (apr *AdvancePaymentRequest) IsCompleted() bool {
	return apr.Status == AdvancePaymentStatusCompleted
}

// IsFailed returns true if the request is failed
func (apr *AdvancePaymentRequest) IsFailed() bool {
	return apr.Status == AdvancePaymentStatusFailed
}

// IsCancelled returns true if the request is cancelled
func (apr *AdvancePaymentRequest) IsCancelled() bool {
	return apr.Status == AdvancePaymentStatusCancelled
}

// MarkCompleted marks the request as completed
func (apr *AdvancePaymentRequest) MarkCompleted(paymentRef string, paidAt time.Time) {
	apr.Status = AdvancePaymentStatusCompleted
	apr.PaymentReference = &paymentRef
	apr.PaidAt = &paidAt
}

// MarkFailed marks the request as failed
func (apr *AdvancePaymentRequest) MarkFailed(paymentRef string) {
	apr.Status = AdvancePaymentStatusFailed
	apr.PaymentReference = &paymentRef
}

// MarkCancelled marks the request as cancelled
func (apr *AdvancePaymentRequest) MarkCancelled() {
	apr.Status = AdvancePaymentStatusCancelled
}

// MarkReceivableSettled marks the receivable as settled by the client
func (apr *AdvancePaymentRequest) MarkReceivableSettled(settledAt time.Time) {
	apr.ReceivableSettledAt = &settledAt
}

// IsReceivableSettled returns true if the receivable has been settled
func (apr *AdvancePaymentRequest) IsReceivableSettled() bool {
	return apr.ReceivableSettledAt != nil
}

// AdvancePaymentRequestRepository defines the interface for advance payment request persistence operations
type AdvancePaymentRequestRepository interface {
	Create(ctx context.Context, req *AdvancePaymentRequest) error
	GetByID(ctx context.Context, id uint64) (*AdvancePaymentRequest, error)
	GetByEmployee(ctx context.Context, employeeID uint64, limit, offset int, fromDate, toDate *time.Time, forMonth *string) ([]*AdvancePaymentRequest, int64, error)
	GetPendingByDateRange(ctx context.Context, fromDate, toDate time.Time) ([]*AdvancePaymentRequest, error)
	GetPendingGroupedByEmployee(ctx context.Context, forMonth string) ([]*EmployeePendingRequests, error)
	SumCompletedByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (uint64, error)
	SumPendingAndCompletedByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (uint64, error)
	SumPendingByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (uint64, error)
	UpdateStatus(ctx context.Context, id uint64, status AdvancePaymentRequestStatus, paymentRef string, paidAt *time.Time) error
	BatchUpdateStatus(ctx context.Context, ids []uint64, status AdvancePaymentRequestStatus, paymentRef string, paidAt *time.Time) error
	Cancel(ctx context.Context, id uint64) error
	List(ctx context.Context, filters AdvancePaymentRequestFilters) ([]*AdvancePaymentRequest, int64, error)
	GetByIDs(ctx context.Context, ids []uint64) ([]*AdvancePaymentRequest, error)
	// GetByIDsLean is GetByIDs without preloading AdvancePayment/Project/Employee,
	// for callers that only need scalar fields (e.g. settlement aggregation).
	GetByIDsLean(ctx context.Context, ids []uint64) ([]*AdvancePaymentRequest, error)
	GetStatsSummary(ctx context.Context, fromDate, toDate time.Time, forMonth string) (*AdvancePaymentStatsSummary, error)
	// MarkReceivableSettled marks the specified requests as receivable settled
	MarkReceivableSettled(ctx context.Context, ids []uint64, settledAt time.Time) (int64, error)
	// UpdateSettlementTransactionID links requests to their settlement transaction
	UpdateSettlementTransactionID(ctx context.Context, ids []uint64, transactionID uint) error
	// CountPending returns the total number of pending advance payment requests
	CountPending(ctx context.Context) (int64, error)
	// GetPendingSummary returns aggregate counts for PENDING+APPROVED requests in a given month.
	// Returns requestCount, employeeCount, totalAmount.
	GetPendingSummary(ctx context.Context, forMonth string) (int64, int64, uint64, error)
	// CountCompletedProjectsByMonth returns the number of distinct projects
	// with at least one COMPLETED advance payment request for the given month.
	CountCompletedProjectsByMonth(ctx context.Context, forMonth string) (int64, error)
	// ClaimPendingForDisbursement atomically locks PENDING requests and sets
	// them to APPROVED within a single transaction using FOR UPDATE SKIP LOCKED.
	// Returns the claimed requests with Employee and Bank preloaded.
	ClaimPendingForDisbursement(ctx context.Context, limit int) ([]*AdvancePaymentRequest, error)
	// GetOrphanedApproved returns APPROVED requests that have no corresponding
	// wallet_payment for the given provider and were approved more than 5
	// minutes ago, indicating they were claimed but never successfully processed.
	// An empty provider string skips the provider filter (legacy fallback).
	GetOrphanedApproved(ctx context.Context, limit int, provider string) ([]*AdvancePaymentRequest, error)
	// GetTotalProviderFee returns the all-time sum of provider_fee for COMPLETED requests.
	GetTotalProviderFee(ctx context.Context) (uint64, error)
	// GetTotalFeeEarned returns the all-time sum of (fee - provider_fee) for COMPLETED requests.
	GetTotalFeeEarned(ctx context.Context) (uint64, error)
	// GetTotalPayableAmount returns the all-time net cash obligation for requests
	// that can still be disbursed (PENDING + APPROVED).
	GetTotalPayableAmount(ctx context.Context) (int64, error)
	// CreateWithBudgetCheck atomically creates a request only if the employee's
	// budget is not exceeded. Uses SELECT FOR UPDATE to prevent TOCTOU races.
	CreateWithBudgetCheck(ctx context.Context, req *AdvancePaymentRequest, employeeID uint64, forMonth string) error
	// ResetToPending atomically resets APPROVED requests back to PENDING.
	// Used by the poller when wallet balance is insufficient to process claimed requests.
	// Only resets requests that are still APPROVED (idempotent, safe for concurrent pollers).
	ResetToPending(ctx context.Context, ids []uint64) error
	// CountStuckPending returns the number of PENDING requests older than olderThan
	// (disbursement worker stalled — should trend to 0).
	CountStuckPending(ctx context.Context, olderThan time.Time) (int64, error)
	// CountStuckPendingInWindow returns the number of PENDING requests older than olderThan
	// and created within [since, until).
	CountStuckPendingInWindow(ctx context.Context, olderThan, since, until time.Time) (int64, error)
	// CountByStatusInWindow returns request counts grouped by status for rows created
	// within [since, until). Used by the health dashboard throughput tiles.
	CountByStatusInWindow(ctx context.Context, since, until time.Time) (map[AdvancePaymentRequestStatus]int64, error)
	// GetCohortByMonths returns cohort rows (one per for_month × cycle-day × status)
	// for the given for_months, scoped to flexible-schedule project assignments.
	// Used by the wallet demand-forecast chart and prediction. cycle_day is 1-indexed
	// from the period start (day 20 of for_month); the persistence layer derives it in
	// Asia/Ho_Chi_Minh so prod UTC created_at values resolve to the correct local day.
	GetCohortByMonths(ctx context.Context, forMonths []string) ([]CohortRow, error)
	// GetCycleForecastState returns the current uploaded advance capacity and the
	// gross amount already consuming it. The capacity is a ceiling for projected
	// future demand, never a substitute for demand from actual requests.
	GetCycleForecastState(ctx context.Context, forMonth string) (*AdvancePaymentCycleForecastState, error)
}

// CohortRow is one cell of the advance-payment request cohort matrix: the request
// count and total net employee-requested amount (request_amount - fee) for a
// given period (for_month), cycle-day, and status. It is demand from actual
// advance_payment_requests, not quota/max_adv_amount.
type CohortRow struct {
	ForMonth     string `gorm:"column:for_month"`
	CycleDay     int    `gorm:"column:cycle_day"`
	Status       string `gorm:"column:status"`
	RequestCount int64  `gorm:"column:request_count"`
	TotalAmount  int64  `gorm:"column:total_amount"`
}

// AdvancePaymentCycleForecastState is the authoritative in-progress cycle state
// used to bound a pace-adjusted wallet forecast.
type AdvancePaymentCycleForecastState struct {
	MaxAdvanceAmount  uint64 `gorm:"column:max_advance_amount"`
	UsedRequestAmount uint64 `gorm:"column:used_request_amount"`
}

type AdvancePaymentStatsSummary struct {
	TotalRequests         int64
	TotalPending          int64
	TotalApproved         int64
	TotalCancelled        int64
	TotalFailed           int64
	TotalPaid             int64
	TotalAmount           uint64
	TotalPaidAmount       uint64
	TotalPendingAmount    uint64
	TotalFailedAmount     uint64
	TotalCancelledAmount  uint64
	TotalFee              uint64
	TotalFeeEarned        uint64
	TotalProviderFee      uint64
	AvgProcessingTimeSecs float64
	CompletedUnder30s     int64
	Completed30sTo2m      int64
	Completed2mTo5m       int64
	Completed5mTo15m      int64
	CompletedOver15m      int64
}

// EmployeePendingRequests represents pending requests grouped by employee for export
type EmployeePendingRequests struct {
	EmployeeID    uint64   `json:"employee_id"`
	EmployeeName  string   `json:"employee_name"`
	EmployeeCCCD  string   `json:"employee_cccd"`
	ProjectID     uint64   `json:"project_id"`
	ProjectCode   string   `json:"project_code"`
	AccountNumber string   `json:"account_number"`
	AccountName   string   `json:"account_name"`
	BankName      string   `json:"bank_name"`
	TotalAmount   uint64   `json:"total_amount"`
	TotalFee      uint64   `json:"total_fee"`
	NetAmount     uint64   `json:"net_amount"`
	RequestIDs    []uint64 `json:"request_ids"`
}

// AdvancePaymentRequestFilters represents filtering options for advance payment request queries
type AdvancePaymentRequestFilters struct {
	Status     *string
	ProjectID  *uint64
	EmployeeID *uint64
	FromDate   *time.Time
	ToDate     *time.Time
	ForMonth   *string
	Search     string
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
}

// AdvancePaymentHistoryItem represents an item in the employee's history
type AdvancePaymentHistoryItem struct {
	ID            uint                        `json:"id"`
	RequestAmount uint64                      `json:"request_amount"`
	Fee           uint64                      `json:"fee"`
	NetAmount     uint64                      `json:"net_amount"`
	Status        AdvancePaymentRequestStatus `json:"status"`
	ForMonth      string                      `json:"for_month"`
	CreatedAt     time.Time                   `json:"created_at"`
	PaidAt        *time.Time                  `json:"paid_at"`
	ProjectName   string                      `json:"project_name"`
	ProjectCode   string                      `json:"project_code"`
}

// AdminAdvancePaymentSummary represents a summary of advance payments for admin dashboard
type AdminAdvancePaymentSummary struct {
	TotalPending int64      `json:"total_pending"`
	TotalAmount  uint64     `json:"total_amount"`
	TotalFee     uint64     `json:"total_fee"`
	TotalNet     uint64     `json:"total_net"`
	FromDate     *time.Time `json:"from_date"`
	ToDate       *time.Time `json:"to_date"`
}
