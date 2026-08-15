package dto

import (
	"time"
)

// CreateLoanRequest represents the request to create a new loan (supports both auto-interest and custom-schedule)
type CreateLoanRequest struct {
	LenderID              uint                       `json:"lender_id" binding:"required"`
	PrincipalAmount       int64                      `json:"principal_amount" binding:"required,gt=0"`
	TermMonths            int                        `json:"term_months" binding:"omitempty,gt=0"` // Required for auto-interest loans
	StartDate             string                     `json:"start_date" binding:"required"`        // YYYY-MM-DD
	PaymentDayOfMonth     uint8                      `json:"payment_day_of_month" binding:"omitempty,gte=1,lte=28"`
	Description           *string                    `json:"description"`
	DisburseNow           bool                       `json:"disburse_now"`
	DisbursementReference *string                    `json:"disbursement_reference"`
	Schedules             []RepaymentScheduleRequest `json:"schedules,omitempty"` // For custom schedule loans
}

// RepaymentScheduleRequest represents a single repayment schedule item in a request
type RepaymentScheduleRequest struct {
	DueDate string `json:"due_date" binding:"required"` // YYYY-MM-DD
	Amount  int64  `json:"amount" binding:"required,gt=0"`
}

// CreateCustomScheduleLoanRequest represents the request to create a custom schedule loan
type CreateCustomScheduleLoanRequest struct {
	LenderID         uint                       `json:"lender_id" binding:"required"`
	PrincipalAmount  int64                      `json:"principal_amount" binding:"required,gt=0"`
	Description      *string                    `json:"description"`
	DisbursementDate string                     `json:"disbursement_date" binding:"required"` // YYYY-MM-DD
	DisburseNow      bool                       `json:"disburse_now"`
	Schedules        []RepaymentScheduleRequest `json:"schedules" binding:"required,min=1"`
}

// DisburseLoanRequest represents the request to disburse a loan
type DisburseLoanRequest struct {
	DisbursementDate      string  `json:"disbursement_date" binding:"required"` // YYYY-MM-DD
	DisbursementReference *string `json:"disbursement_reference"`
}

// RepayLoanRequest represents the request to repay loan principal
type RepayLoanRequest struct {
	Amount           int64   `json:"amount" binding:"required,gt=0"`
	PaymentDate      string  `json:"payment_date" binding:"required"` // YYYY-MM-DD
	PaymentReference *string `json:"payment_reference"`
	Notes            *string `json:"notes"`
}

// UpdateLoanRequest represents the request to update a loan
type UpdateLoanRequest struct {
	Description       *string `json:"description"`
	PaymentDayOfMonth *uint8  `json:"payment_day_of_month" binding:"omitempty,gte=1,lte=28"`
}

// LenderBriefResponse represents brief lender info
type LenderBriefResponse struct {
	ID    uint    `json:"id"`
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
}

// LoanResponse represents the response containing loan data (for list view)
type LoanResponse struct {
	ID                   uint                `json:"id"`
	LoanCode             string              `json:"loan_code"`
	Lender               LenderBriefResponse `json:"lender"`
	PrincipalAmount      int64               `json:"principal_amount"`
	OutstandingPrincipal int64               `json:"outstanding_principal"`
	InterestRateBps      int                 `json:"interest_rate_bps"`
	TotalInterestPaid    int64               `json:"total_interest_paid"`
	TermMonths           int                 `json:"term_months"`
	DisbursementDate     *string             `json:"disbursement_date,omitempty"`
	NextPaymentDate      *string             `json:"next_payment_date,omitempty"`
	NextPaymentAmount    *int64              `json:"next_payment_amount,omitempty"`
	PaymentDayOfMonth    uint8               `json:"payment_day_of_month"`
	Status               string              `json:"status"`
	Description          *string             `json:"description"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
}

// LoanDetailResponse represents the detailed response containing loan data with schedules (for single loan view)
type LoanDetailResponse struct {
	ID                   uint                        `json:"id"`
	LoanCode             string                      `json:"loan_code"`
	Lender               LenderBriefResponse         `json:"lender"`
	PrincipalAmount      int64                       `json:"principal_amount"`
	OutstandingPrincipal int64                       `json:"outstanding_principal"`
	InterestRateBps      int                         `json:"interest_rate_bps"`
	TotalInterestPaid    int64                       `json:"total_interest_paid"`
	TermMonths           int                         `json:"term_months"`
	DisbursementDate     *string                     `json:"disbursement_date,omitempty"`
	NextPaymentDate      *string                     `json:"next_payment_date,omitempty"`
	NextPaymentAmount    *int64                      `json:"next_payment_amount,omitempty"`
	PaymentDayOfMonth    uint8                       `json:"payment_day_of_month"`
	Status               string                      `json:"status"`
	Description          *string                     `json:"description"`
	Schedules            []RepaymentScheduleResponse `json:"schedules"`
	CreatedAt            time.Time                   `json:"created_at"`
	UpdatedAt            time.Time                   `json:"updated_at"`
}

// DisburseLoanResponse represents the response after disbursing a loan
type DisburseLoanResponse struct {
	LoanID          uint   `json:"loan_id"`
	DisbursedAmount int64  `json:"disbursed_amount"`
	LedgerEntryIDs  []uint `json:"ledger_entry_ids"`
}

// RepayLoanResponse represents the response after repaying a loan
type RepayLoanResponse struct {
	LoanID               uint   `json:"loan_id"`
	RepaidAmount         int64  `json:"repaid_amount"`
	OutstandingPrincipal int64  `json:"outstanding_principal"`
	Status               string `json:"status"`
	LedgerEntryIDs       []uint `json:"ledger_entry_ids"`
}

// ScheduleItemResponse represents a single payment schedule item
type ScheduleItemResponse struct {
	Period  int    `json:"period"`
	DueDate string `json:"due_date"` // YYYY-MM-DD
	Type    string `json:"type"`     // "interest" or "principal"
	Amount  int64  `json:"amount"`
	Status  string `json:"status"` // "pending" or "paid"
}

// RepaymentScheduleResponse represents a single repayment schedule item in a response
type RepaymentScheduleResponse struct {
	ID              uint    `json:"id"`
	Period          int     `json:"period"`
	DueDate         string  `json:"due_date"` // YYYY-MM-DD
	Amount          int64   `json:"amount"`
	PrincipalAmount int64   `json:"principal_amount"`
	InterestAmount  int64   `json:"interest_amount"`
	Status          string  `json:"status"`            // "pending" or "paid"
	PaidAt          *string `json:"paid_at,omitempty"` // YYYY-MM-DDTHH:MM:SSZ
}

// ProcessScheduledPaymentRequest represents the request to process a scheduled payment
type ProcessScheduledPaymentRequest struct {
	ScheduleID       uint    `json:"schedule_id" binding:"required"`
	PaymentDate      string  `json:"payment_date" binding:"required"` // YYYY-MM-DD
	PaymentReference *string `json:"payment_reference"`
	Notes            *string `json:"notes"`
}

// ProcessScheduledPaymentResponse represents the response after processing a scheduled payment
type ProcessScheduledPaymentResponse struct {
	LoanID               uint   `json:"loan_id"`
	SchedulePeriod       int    `json:"schedule_period"`
	PaidAmount           int64  `json:"paid_amount"`
	OutstandingPrincipal int64  `json:"outstanding_principal"`
	Status               string `json:"status"`
	LedgerEntryIDs       []uint `json:"ledger_entry_ids"`
}
