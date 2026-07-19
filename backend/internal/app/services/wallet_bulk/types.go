// Package wallet_bulk implements the Stage 2 wallet-page pipeline for the
// Wallet Bulk Transfer feature: parse the "Yêu cầu chuyển tiền" .xlsx
// (exported from /admin/timesheet), enqueue one asynq task per row, drive
// each row through the full 5-step OnePay transfer flow, and on batch
// completion book the successful payroll receivable/cash movement plus the
// aggregate OnePay fee. Failed employees remain visible but are excluded from
// salary receivable and timesheet links.
//
// The package is self-contained: it does NOT touch wallet_payments directly
// — the row worker goes through WalletPaymentService.Initiate (sole insertion
// point) so the fee schedule stamps correctly at INSERT time.
package wallet_bulk

import "errors"

// BulkTransferRow is the parsed shape of one row of the uploaded
// "Yêu cầu chuyển tiền" .xlsx. It is persisted (as JSON array) into
// bulk_transfer_batches.data so the outbox sweeper can re-enqueue tasks
// after a crash and the KQ generator can render original input order.
//
// All monetary amounts are int64 VND. SwiftCode is read directly from the
// input file's "Mã SWIFT" column — no BankRepository lookup here (the
// exporter already resolved it at Stage 1).
type BulkTransferRow struct {
	OrderNo       int    `json:"order_no"`
	AccountNo     string `json:"account_no"`
	AccountName   string `json:"account_name"`
	Bank          string `json:"bank"`           // Vietnamese display name (e.g. "Quân đội (MB)") for KQ column D
	SwiftCode     string `json:"swift_code"`     // read directly from input file
	Amount        int64  `json:"amount"`         // VND
	PaymentDetail string `json:"payment_detail"` // contains VFIC code
	VFICCode      string `json:"vfic_code"`      // extracted from PaymentDetail via regex
}

// Sentinel errors. Each maps to a specific HTTP status code in the
// upload handler (see phase-04-kq-excel-routes.md "Error → HTTP mapping").
var (
	// ErrEmptyFile is returned when the workbook has no data rows.
	ErrEmptyFile = errors.New("empty file")
	// ErrInvalidSheet is returned when the expected `eMB_BulkPayment`
	// sheet is missing.
	ErrInvalidSheet = errors.New("invalid sheet")
	// ErrUnknownExcelHeader is returned when the header row contains a
	// column we don't recognize. We fail loudly rather than guess mapping.
	ErrUnknownExcelHeader = errors.New("unknown excel header")
	// ErrMissingSwiftColumn is returned when the input file lacks the
	// "Mã SWIFT" column — admins should re-export from /admin/timesheet
	// to get a file with SWIFT codes pre-resolved.
	ErrMissingSwiftColumn = errors.New("missing swift column")
	// ErrInvalidSwift is returned when a SWIFT code cell fails ISO 9362
	// format validation (8 or 11 chars: BBBBCCLL[bbr]).
	ErrInvalidSwift = errors.New("invalid swift")
	// ErrMissingVFICCode is returned when PaymentDetail has no VFIC code
	// (regex `(?i)VFIC[0-9a-f]+` finds no match).
	ErrMissingVFICCode = errors.New("missing vfic code")
	// ErrInvalidAmount is returned when Amount < 100000 VND (OnePay min).
	ErrInvalidAmount = errors.New("invalid amount")
	// ErrTooManyRows is returned when the workbook exceeds 5,000 data rows.
	// Hard cap for zip-bomb defense.
	ErrTooManyRows = errors.New("too many rows")
)

// MaxRows is the hard cap on parsed rows (zip-bomb defense). 5,000 rows is
// well above any realistic bulk batch (typical is 50-500) but small enough
// that excelize can't OOM us with a maliciously-crafted 16GB unzip.
const MaxRows = 5000

// MinAmountVND is the per-row minimum (OnePay rejects below this).
const MinAmountVND = int64(100000)
