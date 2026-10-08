package settlement

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func bbdsDetailPOFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(filepath.Join("..", "..", "..", "..", "tests", "fixtures", name)))
	require.NoError(t, err)
	return data
}

// bbdsDetailPOExpectedSummary pins the summary the parser must derive from
// the BBDS_DETAIL_PO export (250 transactions, uniform 3.850₫ fee, September
// 2026 period inferred from the transaction dates).
func bbdsDetailPOExpectedSummary(t *testing.T, report *onePayFeeReport) {
	t.Helper()
	require.Equal(t, "VFICPO", report.summary.MerchantID)
	require.Equal(t, "2026-09-01", report.summary.PeriodFrom)
	require.Equal(t, "2026-09-30", report.summary.PeriodTo)
	require.Equal(t, "Tháng 09.2026", report.summary.PeriodLabel)
	require.Equal(t, 250, report.summary.TransactionCount)
	require.Equal(t, int64(3850), report.summary.FeePerTransaction)
	require.Equal(t, int64(962500), report.summary.TotalFee)
	require.Equal(t, int64(1054445434), report.summary.DetailTotalAmount)
	require.Equal(t, "ONEPAY-FEE:VFICPO:2026-09", report.summary.ImportReference)
	require.Len(t, report.details, 250)

	first := report.details[0]
	require.Equal(t, "ttd9cadb855c974b", first.fundTransferID)
	require.Equal(t, "F12BC02987BF", first.opTransactionID)
	require.Equal(t, "0389435262", first.beneficiaryAccount)
	require.Equal(t, int64(345000), first.amount)
	require.Equal(t, int64(3850), first.fixFee)
	require.Equal(t, int64(3850), first.rowTotalFee)
}

// TestParseOnePayFeeReportFromLegacyXLS is the end-to-end byte-path check:
// OnePay's .xls (BIFF) export parses identically to the .xlsx copy.
func TestParseOnePayFeeReportFromLegacyXLS(t *testing.T) {
	t.Parallel()

	report, issues, err := parseOnePayFeeReport(bytes.NewReader(bbdsDetailPOFixture(t, "BBDS_DETAIL_PO_1791366534312_4065.xls")))
	require.NoError(t, err)
	require.Empty(t, issues)
	require.NotNil(t, report)
	bbdsDetailPOExpectedSummary(t, report)
}

// TestParseOnePayFeeReportFromStrictOOXMLXLSX proves the same export in
// strict-OOXML .xlsx form parses to the identical summary.
func TestParseOnePayFeeReportFromStrictOOXMLXLSX(t *testing.T) {
	t.Parallel()

	report, issues, err := parseOnePayFeeReport(bytes.NewReader(bbdsDetailPOFixture(t, "BBDS_DETAIL_PO_1791366534312_4065.xlsx")))
	require.NoError(t, err)
	require.Empty(t, issues)
	require.NotNil(t, report)
	bbdsDetailPOExpectedSummary(t, report)
}

// TestParseOnePayFeeReportDerivesSummaryWithoutPHIThangSheet exercises the
// summary-less derivation in-memory: deleting the PHI THANG sheet from the
// known monthly layout must still produce a complete, consistent summary.
func TestParseOnePayFeeReportDerivesSummaryWithoutPHIThangSheet(t *testing.T) {
	t.Parallel()

	wb := buildOnePayFeeWorkbookWithVietnameseDetailLayout()
	require.NoError(t, wb.DeleteSheet("PHI THANG"))

	var buf bytes.Buffer
	require.NoError(t, wb.Write(&buf))

	report, issues, err := parseOnePayFeeReport(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Empty(t, issues)
	require.NotNil(t, report)

	require.Equal(t, "VFICPO", report.summary.MerchantID)
	// Derived period = min/max of the transaction dates (05/08 and 31/08),
	// not the summary banner range the original layout carried.
	require.Equal(t, "2026-08-05", report.summary.PeriodFrom)
	require.Equal(t, "2026-08-31", report.summary.PeriodTo)
	require.Equal(t, "Tháng 08.2026", report.summary.PeriodLabel)
	require.Equal(t, 2, report.summary.TransactionCount)
	require.Equal(t, int64(3850), report.summary.FeePerTransaction)
	require.Equal(t, int64(7700), report.summary.TotalFee)
	require.Equal(t, int64(12550160), report.summary.DetailTotalAmount)
	require.Equal(t, "ONEPAY-FEE:VFICPO:2026-08", report.summary.ImportReference)
}

// TestParseOnePayFeeReportRejectsWorkbookWithoutAnyFeeSource: no PHI THANG
// sheet and a detail sheet without fee columns cannot produce a fee report.
func TestParseOnePayFeeReportRejectsWorkbookWithoutAnyFeeSource(t *testing.T) {
	t.Parallel()

	wb := buildOnePayFeeWorkbookWithVietnameseDetailLayout()
	require.NoError(t, wb.DeleteSheet("PHI THANG"))
	// Drop the fee-column headers (Phí XLGD/Total Fee, cols N and O) so the
	// detail sheet matches the template but carries no fee source.
	require.NoError(t, wb.SetCellValue("CHI TIET THANG", "N9", ""))
	require.NoError(t, wb.SetCellValue("CHI TIET THANG", "N8", ""))
	require.NoError(t, wb.SetCellValue("CHI TIET THANG", "O9", ""))
	require.NoError(t, wb.SetCellValue("CHI TIET THANG", "O8", ""))

	var buf bytes.Buffer
	require.NoError(t, wb.Write(&buf))

	_, issues, err := parseOnePayFeeReport(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.NotEmpty(t, issues)
	require.Equal(t, "missing_sheet", issues[0].Code)
	require.Contains(t, issues[0].Message, "PHI THANG")
}

// TestParseOnePayFeeReportDetectsInconsistentPerRowFee: a tampered row fee
// must surface as a reconciliation issue in the summary-less mode.
func TestParseOnePayFeeReportDetectsInconsistentPerRowFee(t *testing.T) {
	t.Parallel()

	wb := buildOnePayFeeWorkbookWithVietnameseDetailLayout()
	require.NoError(t, wb.DeleteSheet("PHI THANG"))
	require.NoError(t, wb.SetCellValue("CHI TIET THANG", "N11", int64(5000))) // row 2 fix fee

	var buf bytes.Buffer
	require.NoError(t, wb.Write(&buf))

	report, issues, err := parseOnePayFeeReport(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, issues, issue("fee_per_transaction_inconsistent", 11, "tt25bf8659cf3b4e", "Phí XLGD 5000 khác phí 3850 của các dòng khác"))
}
