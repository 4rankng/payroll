package dto

// ImportRowIssue is the internal row-level error contract shared by file
// importers (BCC timesheets, OnePay fee reports, employee imports, wallet
// bulk, FlexPay). Modeled on OnePayFeeReportIssue — the most structured
// regime already in production.
//
// Each island adapts it to its existing external JSON shape (ImportError,
// RowError, OnePayFeeReportIssue); external payloads are unchanged. Codes
// are machine-readable so handlers can branch on them without parsing
// Vietnamese message text.
type ImportRowIssue struct {
	// Code identifies the failure class, e.g. "duplicate_import",
	// "missing_column", "amount_mismatch", "unknown_employee".
	Code string `json:"code"`
	// Message is the human-readable explanation (Vietnamese, user-facing).
	Message string `json:"message"`
	// Row is the 1-based workbook row the issue belongs to (0 when unknown).
	Row int `json:"row"`
	// Reference locates the record the issue is about — a CCCD, transaction
	// reference, or import key ("" when unknown).
	Reference string `json:"reference,omitempty"`
}
