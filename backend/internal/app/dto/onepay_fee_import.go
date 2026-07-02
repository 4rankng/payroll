package dto

import "time"

type OnePayFeeReportIssue struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Row       int    `json:"row,omitempty"`
	Reference string `json:"reference,omitempty"`
}

type OnePayFeeReportSummary struct {
	MerchantID          string `json:"merchant_id"`
	MerchantName        string `json:"merchant_name"`
	PeriodLabel         string `json:"period_label"`
	PeriodFrom          string `json:"period_from"`
	PeriodTo            string `json:"period_to"`
	TransactionCount    int    `json:"transaction_count"`
	FeePerTransaction   int64  `json:"fee_per_transaction"`
	TotalFee            int64  `json:"total_fee"`
	DetailTotalAmount   int64  `json:"detail_total_amount"`
	AppRecordedFeeTotal int64  `json:"app_recorded_fee_total"`
	ImportReference     string `json:"import_reference"`
}

type OnePayFeeImportResponse struct {
	Summary         OnePayFeeReportSummary `json:"summary"`
	TransactionID   uint                   `json:"transaction_id"`
	TransactionCode string                 `json:"transaction_code"`
	LedgerEntryIDs  []uint                 `json:"ledger_entry_ids"`
	CreatedAt       time.Time              `json:"created_at"`
}
