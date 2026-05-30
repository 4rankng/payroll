package dto

// FlexPayReconciliationExportRequest represents the request parameters for FlexPay reconciliation export
type FlexPayReconciliationExportRequest struct {
	ForMonth string `form:"forMonth" binding:"required"`
}

// FlexPayReconciliationExportResponse represents the response after generating the reconciliation report
type FlexPayReconciliationExportResponse struct {
	Message        string `json:"message"`
	RowCount       int64  `json:"row_count"`
	CancelledCount int64  `json:"cancelled_count"`
	Filename       string `json:"filename"`
}
