package settlement

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestGetSettlementAmount_FindsAmountByLabel(t *testing.T) {
	fixturePath := "../../../../tests/fixtures/sao_ke_tt_testdata.xlsx"
	f, err := excelize.OpenFile(fixturePath)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	parser := NewSettlementExcelParser()
	amount, err := parser.getSettlementAmount(f)
	require.NoError(t, err)
	assert.Equal(t, int64(48082800), amount, "should find 'Tiền CTy Phải Trả' amount from fixture")
}

func TestGetSettlementAmount_VariedRowPosition(t *testing.T) {
	tests := []struct {
		name       string
		labelRow   int
		amount     string
		wantAmount int64
	}{
		{name: "label at row 2 (new format)", labelRow: 2, amount: "78620000", wantAmount: 78620000},
		{name: "label at row 4 (old format)", labelRow: 4, amount: "48082800", wantAmount: 48082800},
		{name: "label at row 7", labelRow: 7, amount: "12345678", wantAmount: 12345678},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := excelize.NewFile()
			defer func() { _ = f.Close() }()

			labelCell := fmt.Sprintf("D%d", tc.labelRow)
			amountCell := fmt.Sprintf("E%d", tc.labelRow)
			require.NoError(t, f.SetCellStr(f.GetSheetName(0), labelCell, LabelSettlementAmount))
			require.NoError(t, f.SetCellStr(f.GetSheetName(0), amountCell, tc.amount))
			// Rename default sheet to Summary
			require.NoError(t, f.SetSheetName(f.GetSheetName(0), SheetSummary))

			parser := NewSettlementExcelParser()
			amount, err := parser.getSettlementAmount(f)
			require.NoError(t, err)
			assert.Equal(t, tc.wantAmount, amount)
		})
	}
}

func TestSaoKeTTTestData_HasInternalSheetWithTimesheetIDs(t *testing.T) {
	fixturePath := "../../../../tests/fixtures/sao_ke_tt_testdata.xlsx"

	// Verify the file exists
	fileInfo, err := os.Stat(fixturePath)
	require.NoError(t, err, "fixture file should exist")
	require.Greater(t, fileInfo.Size(), int64(0), "fixture file should not be empty")

	// Open the Excel file
	f, err := excelize.OpenFile(fixturePath)
	require.NoError(t, err, "should be able to open the Excel file")
	defer func() {
		_ = f.Close()
	}()

	// Check that INTERNAL sheet exists
	sheetList := f.GetSheetList()
	assert.Contains(t, sheetList, SheetInternal, "Excel file should contain INTERNAL sheet")

	// Verify INTERNAL sheet is hidden
	sheetIndex, err := f.GetSheetIndex(SheetInternal)
	require.NoError(t, err, "should be able to get INTERNAL sheet index")
	visible, err := f.GetSheetVisible(SheetInternal)
	require.NoError(t, err, "should be able to check sheet visibility")
	assert.False(t, visible, "INTERNAL sheet should be hidden")
	assert.GreaterOrEqual(t, sheetIndex, 0, "INTERNAL sheet should have a valid index")

	// Read all timesheet IDs from INTERNAL sheet (column A)
	rows, err := f.GetRows(SheetInternal)
	require.NoError(t, err, "should be able to read rows from INTERNAL sheet")
	require.Greater(t, len(rows), 0, "INTERNAL sheet should contain at least one row")

	var timesheetIDs []uint
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		cellValue := row[ColumnTimesheetID]
		if cellValue == "" {
			t.Fatalf("cell A%d should contain a timesheet ID, but was empty", i+1)
		}
		// Verify it's a valid number
		var id uint
		_, err := fmt.Sscanf(cellValue, "%d", &id)
		require.NoError(t, err, "cell A%d should contain a valid timesheet ID number, got: %s", i+1, cellValue)
		require.Greater(t, id, uint(0), "cell A%d should contain a positive timesheet ID, got: %d", i+1, id)
		timesheetIDs = append(timesheetIDs, id)
	}

	// Verify we have at least some timesheet IDs
	assert.Greater(t, len(timesheetIDs), 0, "INTERNAL sheet should contain at least one timesheet ID")

	// Verify no duplicate IDs
	idSet := make(map[uint]struct{})
	for _, id := range timesheetIDs {
		_, exists := idSet[id]
		assert.False(t, exists, "INTERNAL sheet should not contain duplicate timesheet IDs, found duplicate: %d", id)
		idSet[id] = struct{}{}
	}

	t.Logf("INTERNAL sheet contains %d unique timesheet IDs", len(timesheetIDs))
	for i, id := range timesheetIDs {
		t.Logf("  A%d: %d", i+1, id)
	}
}

func TestSaoKeTTTestData_ValidateRequiredSheets(t *testing.T) {
	fixturePath := "../../../../tests/fixtures/sao_ke_tt_testdata.xlsx"

	f, err := excelize.OpenFile(fixturePath)
	require.NoError(t, err, "should be able to open the Excel file")
	defer func() {
		_ = f.Close()
	}()

	sheetList := f.GetSheetList()

	// Verify Summary sheet exists
	assert.Contains(t, sheetList, SheetSummary, "Excel file should contain Summary sheet")

	// Verify INTERNAL sheet exists
	assert.Contains(t, sheetList, SheetInternal, "Excel file should contain INTERNAL sheet")

	// Verify INTERNAL sheet is hidden
	visible, err := f.GetSheetVisible(SheetInternal)
	require.NoError(t, err)
	assert.False(t, visible, "INTERNAL sheet should be hidden")

	// Verify Summary sheet is visible (it should be the active sheet)
	summaryIndex, err := f.GetSheetIndex(SheetSummary)
	require.NoError(t, err)
	activeSheetIndex := f.GetActiveSheetIndex()
	assert.Equal(t, summaryIndex, activeSheetIndex, "Summary sheet should be the active sheet")
}
