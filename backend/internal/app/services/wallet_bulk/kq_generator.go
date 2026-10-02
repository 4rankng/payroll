package wallet_bulk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/pkg/excelkit"
)

const (
	// kqSheetName aliases the shared KQ result layout so package tests can
	// address the generated sheet.
	kqSheetName       = excelkit.KQSheetName
	vficInvoicePrefix = "VFIC"
	ftPendingMessage  = "Đang chờ FT"
)

// KQExcelGenerator produces the "KQ Chuyen Tien" .xlsx for a finished batch.
//
// Layout per plan.md phase-04:
//   - Column D reads the Vietnamese display name from bulk_transfer_batches.data
//     JSON (matched by bulk_transfer_order), NOT a SWIFT reverse-lookup.
//   - Column G reads wallet_payments.fee (stamped by Initiate).
//   - Column H maps wallet_payments.status → Vietnamese.
//   - Column I reads invoice_no; if it still starts with "VFIC" (FT not yet
//     received via IPN), display "Đang chờ FT" instead.
type KQExcelGenerator struct {
	clock func() time.Time
}

// NewKQExcelGenerator constructs the generator. clock defaults to clock.Now.
func NewKQExcelGenerator(clockFn func() time.Time) *KQExcelGenerator {
	if clockFn == nil {
		clockFn = time.Now
	}
	return &KQExcelGenerator{clock: clockFn}
}

// Generate builds the .xlsx for the given batch + wallet_payments rows.
// Rows are already ordered by bulk_transfer_order ASC by the caller.
//
// The workbook layout itself lives in excelkit (WriteKQWorkbook) — this
// function only maps wallet_payments onto the shared row shape.
func (g *KQExcelGenerator) Generate(
	ctx context.Context,
	batch *domain.BulkTransferBatch,
	rows []*domaintx.WalletPayment,
) ([]byte, error) {
	// Build the orderNo → display name map from batch.data JSON.
	displayNames, err := buildDisplayNameMap(batch.Data)
	if err != nil {
		return nil, fmt.Errorf("build display name map: %w", err)
	}

	kqRows := make([]excelkit.KQRow, 0, len(rows))
	for _, row := range rows {
		order := 0
		if row.BulkTransferOrder != nil {
			order = int(*row.BulkTransferOrder)
		}
		displayName := displayNames[order]
		if displayName == "" {
			// Fallback: SWIFT code is better than nothing.
			displayName = row.RecipientBank
		}

		kqRows = append(kqRows, excelkit.KQRow{
			Order:          order,
			AccountNumber:  row.RecipientAccountNo,
			AccountName:    row.RecipientName,
			BankName:       displayName,
			Amount:         row.RequestedAmount,
			TransactionRef: descriptionOr(row.Description, row.RequestID),
			Fee:            row.Fee,
			Status:         statusToVietnamese(row.Status),
			Reference:      invoiceOrPending(row),
		})
	}

	now := g.clock()
	return excelkit.WriteKQWorkbook(excelkit.KQTitle{
		Title:    fmt.Sprintf("Kết quả chuyển tiền - %s", batch.Filename),
		Ref:      fmt.Sprintf("Mã lô: WB%d", batch.ID),
		DateLine: excelkit.KQDateLine(now),
	}, kqRows)
}

// buildDisplayNameMap unmarshals batch.data JSON → []BulkTransferRow and
// builds orderNo → Bank (Vietnamese display name).
func buildDisplayNameMap(dataJSON string) (map[int]string, error) {
	if strings.TrimSpace(dataJSON) == "" {
		return map[int]string{}, nil
	}
	var rows []BulkTransferRow
	if err := json.Unmarshal([]byte(dataJSON), &rows); err != nil {
		return nil, fmt.Errorf("unmarshal batch.data: %w", err)
	}
	m := make(map[int]string, len(rows))
	for _, r := range rows {
		m[r.OrderNo] = r.Bank
	}
	return m, nil
}

// statusToVietnamese maps wallet_payments.status → KQ column H label.
func statusToVietnamese(s domaintx.State) string {
	switch s {
	case domaintx.StatePending, domaintx.StateVerified, domaintx.StateAuthorised:
		return "Đang xử lý"
	case domaintx.StateCompleted:
		return "Thành công"
	case domaintx.StateFailed:
		return "Thất bại"
	case domaintx.StateReversed:
		return "Đã hoàn"
	default:
		return string(s)
	}
}

// invoiceOrPending returns the FT number for column I, or "Đang chờ FT" when
// invoice_no still starts with VFIC (FT not yet received via IPN), or the
// error message for failed rows.
func invoiceOrPending(row *domaintx.WalletPayment) string {
	switch row.Status {
	case domaintx.StateFailed:
		if row.ErrorMessage != nil && *row.ErrorMessage != "" {
			return *row.ErrorMessage
		}
		return "Thất bại"
	case domaintx.StateReversed:
		inv := row.GetInvoiceNo()
		if inv != "" {
			return inv + " (REVERSED)"
		}
		return "Đã hoàn"
	case domaintx.StateCompleted:
		inv := row.GetInvoiceNo()
		// Initiate stamps invoice_no := request_id at INSERT. If IPN hasn't
		// yet overwritten it with the real FT number, the cell still starts
		// with VFIC — surface "Đang chờ FT" instead so admin knows.
		if inv == "" || strings.HasPrefix(inv, vficInvoicePrefix) {
			return ftPendingMessage
		}
		return inv
	default:
		// pending/verified/authorised — no FT yet.
		return ""
	}
}

// descriptionOr returns the row's description, falling back to request_id.
func descriptionOr(desc *string, fallback string) string {
	if desc != nil && *desc != "" {
		return *desc
	}
	return fallback
}

// FilenameForKQ builds the download filename per spec.
func FilenameForKQ(batchID uint64, now time.Time) string {
	return fmt.Sprintf("KQ_Chuyen_Tien_%d_%s.xlsx", batchID, now.Format("20060102_150405"))
}
