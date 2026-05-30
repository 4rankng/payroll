package bulktransfer

// Transfer status constants
const (
	TransferStatusFailed  = "x"
	TransferStatusSuccess = ""
)

// Result status constants for parsed bank results and stored transfer_status
const (
	ResultStatusCompleted = "completed"
	ResultStatusFailed    = "failed"
)

// Processing status constants
const (
	ProcessingStatusSuccess = "success"
	ProcessingStatusFailed  = "failed"
	ProcessingStatusWarning = "warning"
)

// Payment status constants
const (
	PaymentStatusPaid   = "paid"
	PaymentStatusFailed = "failed"
)

// Bulk transfer reference prefix
const (
	BulkTransferReferencePrefix = "BULK-TRANSFER-"
)
