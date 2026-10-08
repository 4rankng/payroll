package excelkit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func bbdsFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(filepath.Join("..", "..", "..", "tests", "fixtures", name)))
	require.NoError(t, err)
	return data
}

// TestOpenWorkbookConvertsLegacyXLS verifies the OLE2 → excelize conversion
// against a real OnePay BBDS_DETAIL_PO export: sheet names survive, BIFF8
// compressed Latin-1 strings are repaired to UTF-8, and numeric cells keep
// their values.
func TestOpenWorkbookConvertsLegacyXLS(t *testing.T) {
	t.Parallel()

	f, err := OpenWorkbook(bytes.NewReader(bbdsFixture(t, "BBDS_DETAIL_PO_1791366534312_4065.xls")))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	require.Equal(t, []string{"CHI TIET THANG"}, f.GetSheetList())

	rows, err := f.GetRows("CHI TIET THANG")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 10)

	// Vietnamese headers stored as 8-bit BIFF strings must come back as
	// valid UTF-8, not raw Latin-1 bytes.
	require.Equal(t, "Ngân hàng", rows[7][9])
	require.Equal(t, "Phí XLGD", rows[7][13])
	require.Equal(t, "Tổng phí", rows[7][14])

	// Numbers stay numeric through the conversion.
	require.Equal(t, "345000", rows[9][12])
	require.Equal(t, "30/09/2026 21:03:22", rows[9][6])
}

// TestOpenWorkbookReadsStrictOOXLSX pins support for the strict-OOXML
// dialect OnePay ships alongside the .xls export; openpyxl-class readers
// reject it, excelize must not.
func TestOpenWorkbookReadsStrictOOXLSX(t *testing.T) {
	t.Parallel()

	f, err := OpenWorkbook(bytes.NewReader(bbdsFixture(t, "BBDS_DETAIL_PO_1791366534312_4065.xlsx")))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	require.Equal(t, []string{"CHI TIET THANG"}, f.GetSheetList())
	rows, err := f.GetRows("CHI TIET THANG")
	require.NoError(t, err)
	require.Equal(t, "Beneficiary Bank", rows[8][9])
}

func TestOpenWorkbookPassesXLSXThroughUnchanged(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	wb := excelize.NewFile()
	require.NoError(t, wb.SetCellValue("Sheet1", "A1", "SLGD"))
	require.NoError(t, wb.SetCellValue("Sheet1", "B1", int64(3850)))
	_, err := wb.NewSheet("CHI TIET THANG")
	require.NoError(t, err)
	require.NoError(t, wb.Write(&buf))

	f, err := OpenWorkbook(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	require.Equal(t, []string{"Sheet1", "CHI TIET THANG"}, f.GetSheetList())
	value, err := f.GetCellValue("Sheet1", "B1")
	require.NoError(t, err)
	require.Equal(t, "3850", value)
}

func TestOpenWorkbookRejectsNonExcelBytes(t *testing.T) {
	t.Parallel()

	_, err := OpenWorkbook(strings.NewReader("definitely not a workbook"))
	require.Error(t, err)
}
