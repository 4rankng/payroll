package payroll

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

const (
	internalSheetName = "INTERNAL"
	// internalTypeTimesheet is half of a writer/reader contract: the
	// settlement parser (settlement/excel_parser.go InternalTypePrefix)
	// routes on this prefix when re-importing sao-kê exports. Change both
	// together.
	internalTypeTimesheet = "type:timesheet"
)

// addInternalSheet adds or replaces the INTERNAL sheet listing timesheet IDs.
func addInternalSheet(f *excelize.File, ids []uint) error {
	if f == nil {
		return fmt.Errorf("excel file must not be nil")
	}

	if idx, err := f.GetSheetIndex(internalSheetName); err == nil && idx >= 0 {
		if err := f.DeleteSheet(internalSheetName); err != nil {
			return fmt.Errorf("failed to delete existing %s sheet: %w", internalSheetName, err)
		}
	}

	if _, err := f.NewSheet(internalSheetName); err != nil {
		return fmt.Errorf("failed to create %s sheet: %w", internalSheetName, err)
	}

	// A1: type indicator for parser differentiation
	if err := f.SetCellValue(internalSheetName, "A1", internalTypeTimesheet); err != nil {
		return fmt.Errorf("failed to set type header: %w", err)
	}

	for i, tsID := range ids {
		cell := fmt.Sprintf("A%d", i+2)
		if err := f.SetCellValue(internalSheetName, cell, tsID); err != nil {
			return fmt.Errorf("failed to set cell %s: %w", cell, err)
		}
	}

	// Hide the INTERNAL sheet so it's not visible to users
	// This must be the last operation on the sheet to ensure it stays hidden
	if err := f.SetSheetVisible(internalSheetName, false); err != nil {
		return fmt.Errorf("failed to hide %s sheet: %w", internalSheetName, err)
	}

	return nil
}
