package dto

// Vendor-facing DTOs for the read-only vendor timesheet API (/api/v1/vendor/*).
//
// These are deliberately separate from the internal dto.TimesheetResponse
// family. The vendor contract is keyed on business codes (project code,
// CCCD) rather than internal numeric IDs, so the vendor never learns or
// caches our primary keys. The payload is full-fidelity because the caller is
// a sister company that reconciles against it — pay amounts and personal data
// are in scope — while RevenueReceivable/RevenuePaid stay out because those
// are client-billing figures rather than worker payouts.

// VendorProjectItem is one project in the vendor's vocabulary.
type VendorProjectItem struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	ClientName string `json:"client_name"`
	Status     string `json:"status"`
}

// VendorEmployeeItem is one employee in the vendor's vocabulary, keyed by
// CCCD. ProjectCodes lets the vendor build the project↔employee relationship
// without a third call.
//
// The OnePay bank-validation fields (BankAccountStatus /
// BankAccountInvalidReason) are deliberately not exposed: they describe our
// internal payment-rail state, not the vendor's reconciliation view.
type VendorEmployeeItem struct {
	CCCD              string   `json:"cccd"`
	Fullname          string   `json:"fullname"`
	Mobile            string   `json:"mobile"`
	Email             *string  `json:"email"`
	Address           string   `json:"address"`
	DateOfBirth       *string  `json:"date_of_birth"`
	BankID            *uint    `json:"bank_id"`
	BankAccountName   string   `json:"bank_account_name"`
	BankAccountNumber string   `json:"bank_account_number"`
	ProjectCodes      []string `json:"project_codes"`
}

// VendorTimesheetItem is one timesheet row. Money fields are included because
// the sister vendor reconciles what the company owes against what it has
// paid; payment_reference is included so both sides can tie a row to the same
// bank transaction.
//
// Status and RejectionReason (the internal approval workflow) are not
// exposed — the vendor's reconciliation is a payment concern, not an approval
// workflow.
//
// RevenueReceivable / RevenuePaid are intentionally absent — see the package
// comment.
type VendorTimesheetItem struct {
	ProjectCode      string  `json:"project_code"`
	CCCD             string  `json:"cccd"`
	EmployeeName     string  `json:"employee_name"`
	Date             string  `json:"date"`   // YYYY-MM-DD
	HoursWorked      float64 `json:"hours_worked"`
	PayType          string  `json:"pay_type"` // Day_type.shift_type, e.g. ngày_thường.TCN
	PayRate          int64   `json:"payrate"`  // VND
	Amount           int64   `json:"amount"`   // VND, computed total
	PaidAmount       int64   `json:"paid_amount"`
	PaymentStatus    string  `json:"payment_status"`
	PaymentReference *string `json:"payment_reference"` // nil until paid
	PaidAt           *string `json:"paid_at"`
}
