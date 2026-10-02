package bulktransfer

import (
	"context"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/excelkit"
)

// KQResultFile is a generated "Kết quả chuyển khoản" workbook plus the
// metadata the download handler reports in X- headers.
type KQResultFile struct {
	ExcelBytes  []byte
	Filename    string
	TotalCount  int
	TotalAmount int64
}

// KQResultGenerator turns an OUTBOUND transfer listing ("chuyển lô" file —
// the MBank manual template or the OnePay `eMB_BulkPayment` workbook) into the
// bank RESULT workbook the system already accepts back on
// POST /payrolls/bulk-transfer-result.
//
// The conversion is deliberately value-preserving: every listing line becomes
// one KQ line carrying the same account, beneficiary, bank, amount and — the
// part that makes the round trip work — the same VFIC transaction code in the
// payment-detail column. TransactionCodeStrategy matches rows by that code,
// so a generated KQ re-imports against the batch it was generated from.
//
// Every generated line starts as "Thành công" with the FT / note column left
// blank: the admin edits the workbook (marking real failures and pasting the
// bank's FT numbers) before uploading it.
type KQResultGenerator struct {
	excelConverter ExcelConverter
	clock          func() time.Time
}

// NewKQResultGenerator constructs the generator. clockFn defaults to
// clock.Now when nil.
func NewKQResultGenerator(excelConverter ExcelConverter, clockFn func() time.Time) *KQResultGenerator {
	if clockFn == nil {
		clockFn = clock.Now
	}
	return &KQResultGenerator{excelConverter: excelConverter, clock: clockFn}
}

// GenerateFromTransferListing reads the uploaded chuyenlo file and returns
// the equivalent KQ result workbook.
func (g *KQResultGenerator) GenerateFromTransferListing(
	ctx context.Context,
	fileHeader *multipart.FileHeader,
) (*KQResultFile, error) {
	rows, err := g.readListingRows(fileHeader)
	if err != nil {
		return nil, err
	}

	headerRow, cols, ok := excelkit.LocateTransferListingHeader(rows)
	if !ok {
		return nil, domain.NewValidationError(constants.MsgTransferListingHeaderNotFoundVN)
	}

	kqRows := make([]excelkit.KQRow, 0, len(rows)-headerRow-1)
	var totalAmount int64

	for i := headerRow + 1; i < len(rows); i++ {
		row := rows[i]

		accountNumber := listingCell(row, cols.AccountNumber)
		transactionRef := listingCell(row, cols.TransactionRef)

		// A listing pads rows (totals, footers, blanks); a line needs both the
		// beneficiary account and the transaction code to be re-importable.
		if accountNumber == "" || transactionRef == "" {
			continue
		}

		order := len(kqRows) + 1
		if cols.Order >= 0 {
			if parsed, err := strconv.Atoi(listingCell(row, cols.Order)); err == nil {
				order = parsed
			}
		}

		amount := parseTransferListingAmount(listingCell(row, cols.Amount))
		totalAmount += amount

		kqRows = append(kqRows, excelkit.KQRow{
			Order:          order,
			AccountNumber:  accountNumber,
			AccountName:    listingCell(row, cols.AccountName),
			BankName:       listingCell(row, cols.BankName),
			Amount:         amount,
			TransactionRef: transactionRef,
			Fee:            0,
			Status:         kqDefaultSuccessLabel,
			Reference:      "",
		})
	}

	if len(kqRows) == 0 {
		return nil, domain.NewValidationError(constants.MsgTransferListingNoRowsVN)
	}

	now := g.clock()
	sourceName := strings.TrimSuffix(fileHeader.Filename, ".xlsx")

	excelBytes, err := excelkit.WriteKQWorkbook(excelkit.KQTitle{
		Title:    "Kết quả chuyển tiền - " + sourceName,
		Ref:      fmt.Sprintf("Nguồn: %s", sourceName),
		DateLine: excelkit.KQDateLine(now),
	}, kqRows)
	if err != nil {
		return nil, fmt.Errorf("write KQ workbook: %w", err)
	}

	return &KQResultFile{
		ExcelBytes:  excelBytes,
		Filename:    FilenameForKQResult(now),
		TotalCount:  len(kqRows),
		TotalAmount: totalAmount,
	}, nil
}

// readListingRows loads the first sheet of the uploaded workbook through the
// shared ExcelConverter (which applies the repo's decompression caps).
func (g *KQResultGenerator) readListingRows(fileHeader *multipart.FileHeader) ([][]string, error) {
	excelFile, err := g.excelConverter.ProcessExcelFile(fileHeader)
	if err != nil {
		return nil, fmt.Errorf("process excel file: %w", err)
	}

	sheets := g.excelConverter.GetSheetNames(excelFile)
	if len(sheets) == 0 {
		return nil, domain.NewValidationError(constants.MsgTransferListingHeaderNotFoundVN)
	}

	rows, err := g.excelConverter.GetRowsFromSheet(excelFile, sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read sheet %s: %w", sheets[0], err)
	}

	return rows, nil
}

// kqDefaultSuccessLabel is the KQ column-H text for a settled transfer. It
// must stay in sync with the labels TransactionCodeStrategy.parseStatus
// recognises ("thành công" → completed).

const kqDefaultSuccessLabel = "Thành công"

// FilenameForKQResult builds the download filename for a generated KQ.
func FilenameForKQResult(now time.Time) string {
	return fmt.Sprintf("KQ_CK_%s.xlsx", now.Format("20060102_150405"))
}

// listingCell reads one mapped column from a row, guarding short rows
// (excelize trims trailing empty cells) and absent columns.
func listingCell(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[col])
}

// parseTransferListingAmount converts a listing amount cell to VND. Bank
// listings write plain digits, but thousands separators and a currency
// suffix both show up in hand-edited files.
func parseTransferListingAmount(s string) int64 {
	cleaned := strings.ReplaceAll(s, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "đ", "")
	cleaned = strings.ReplaceAll(cleaned, "₫", "")
	cleaned = strings.ReplaceAll(cleaned, "VND", "")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		return 0
	}

	amount, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		// Vietnamese spreadsheets also group thousands with dots
		// ("1.500.000"), which ParseFloat rejects. A lone decimal point
		// ("930000.5") already parsed above, so reaching here means the
		// dots are separators — strip them and retry once.
		amount, err = strconv.ParseFloat(strings.ReplaceAll(cleaned, ".", ""), 64)
		if err != nil {
			return 0
		}
	}

	return int64(amount)
}
