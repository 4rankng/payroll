package wallet_bulk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"

	"github.com/xuri/excelize/v2"
)

// KQ sheet/column layout (matches the reference KQ Chuyen Tien.xlsx).
// Sheet `data`, rows 1-4 header, row 5+ data, 9 columns A-I.
const (
	kqSheetName        = "data"
	kqTitleRow         = 1
	kqRefRow           = 2
	kqDateRow          = 3
	kqHeaderRow        = 4
	kqFirstDataRow     = 5
	vficInvoicePrefix  = "VFIC"
	ftPendingMessage   = "Đang chờ FT"
)

// kqHeaders is the row-4 header set.
var kqHeaders = []string{
	"STT",
	"Số tài khoản",
	"Tên người thụ hưởng",
	"Ngân hàng thụ hưởng",
	"Số tiền",
	"Nội dung",
	"Phí",
	"Trạng thái",
	"FT / Ghi chú",
}

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

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	// Rename default Sheet1 → data.
	if err := f.SetSheetName("Sheet1", kqSheetName); err != nil {
		return nil, fmt.Errorf("rename sheet: %w", err)
	}

	// Title block (rows 1-3).
	now := g.clock()
	if err := setCell(f, "A1", fmt.Sprintf("Kết quả chuyển tiền - %s", batch.Filename)); err != nil {
		return nil, err
	}
	if err := setCell(f, "A2", fmt.Sprintf("Mã lô: WB%d", batch.ID)); err != nil {
		return nil, err
	}
	if err := setCell(f, "A3", fmt.Sprintf("Ngày: %s", now.Format("02/01/2006 15:04:05"))); err != nil {
		return nil, err
	}

	// Header row 4.
	for i, h := range kqHeaders {
		col := columnLetter(i)
		cell := fmt.Sprintf("%s%d", col, kqHeaderRow)
		if err := setCell(f, cell, h); err != nil {
			return nil, err
		}
	}

	// Data rows starting at row 5.
	for idx, row := range rows {
		excelRow := kqFirstDataRow + idx
		order := 0
		if row.BulkTransferOrder != nil {
			order = int(*row.BulkTransferOrder)
		}
		displayName := displayNames[order]
		if displayName == "" {
			// Fallback: SWIFT code is better than nothing.
			displayName = row.RecipientBank
		}

		cells := []struct {
			col string
			val any
		}{
			{"A", order},
			{"B", row.RecipientAccountNo},
			{"C", row.RecipientName},
			{"D", displayName},
			{"E", row.RequestedAmount},
			{"F", descriptionOr(row.Description, row.RequestID)},
			{"G", row.Fee},
			{"H", statusToVietnamese(row.Status)},
			{"I", invoiceOrPending(row)},
		}
		for _, c := range cells {
			cell := fmt.Sprintf("%s%d", c.col, excelRow)
			if err := setCell(f, cell, c.val); err != nil {
				return nil, err
			}
		}
	}

	// Column widths.
	widths := map[string]float64{
		"A": 6, "B": 22, "C": 28, "D": 36, "E": 18,
		"F": 40, "G": 12, "H": 16, "I": 28,
	}
	for col, w := range widths {
		if err := f.SetColWidth(kqSheetName, col, col, w); err != nil {
			return nil, fmt.Errorf("set col width %s: %w", col, err)
		}
	}

	// Sheet dimension.
	lastRow := kqFirstDataRow + len(rows) - 1
	if lastRow < kqHeaderRow {
		lastRow = kqHeaderRow
	}
	if err := f.SetSheetDimension(kqSheetName, fmt.Sprintf("A1:I%d", lastRow)); err != nil {
		return nil, fmt.Errorf("set sheet dimension: %w", err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write xlsx: %w", err)
	}
	return buf.Bytes(), nil
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

// columnLetter converts 0-indexed column → A, B, ..., I.
func columnLetter(idx int) string {
	if idx < 0 || idx > 25 {
		return "?"
	}
	return string(rune('A' + idx))
}

// setCell writes a value to a cell on kqSheetName.
func setCell(f *excelize.File, cell string, val any) error {
	if err := f.SetCellValue(kqSheetName, cell, val); err != nil {
		return fmt.Errorf("set cell %s: %w", cell, err)
	}
	return nil
}

// FilenameForKQ builds the download filename per spec.
func FilenameForKQ(batchID uint64, now time.Time) string {
	return fmt.Sprintf("KQ_Chuyen_Tien_%d_%s.xlsx", batchID, now.Format("20060102_150405"))
}
