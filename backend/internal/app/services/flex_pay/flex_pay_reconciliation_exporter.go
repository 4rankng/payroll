package flex_pay

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	appconfig "api-server/internal/app/services/config"
	"api-server/internal/app/services/excel"
	serviceports "api-server/internal/domain/ports/services"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	pkgConstants "api-server/internal/pkg/constants"

	"github.com/xuri/excelize/v2"
)

const (
	internalSheetName   = "INTERNAL"
	internalTypeAdvance = "type:advance_payment"
)

// TransferBankInfoProvider supplies beneficiary bank details for FlexPay
// reconciliation exports. Implemented by *config.SettingsConfigService.
type TransferBankInfoProvider interface {
	GetTransferBankInfo(ctx context.Context) appconfig.TransferBankInfo
}

// FlexPayReconciliationExporter handles Excel generation for FlexPay reconciliation report
type FlexPayReconciliationExporter struct {
	logger       *slog.Logger
	bankSettings TransferBankInfoProvider
}

// NewFlexPayReconciliationExporter creates a new exporter
func NewFlexPayReconciliationExporter(bankSettings TransferBankInfoProvider) *FlexPayReconciliationExporter {
	return &FlexPayReconciliationExporter{
		logger:       observability.GetLogger(),
		bankSettings: bankSettings,
	}
}

// resolveBankInfo returns the configured beneficiary bank details, falling
// back to defaults when no settings provider is bound.
func (e *FlexPayReconciliationExporter) resolveBankInfo(ctx context.Context) appconfig.TransferBankInfo {
	if e.bankSettings == nil {
		return appconfig.DefaultTransferBankInfo()
	}
	return e.bankSettings.GetTransferBankInfo(ctx)
}

// BankInfoForStatement exposes the configured beneficiary bank details
// (including the visibility flag) to statement email builders that hold the
// exporter but not the settings service directly.
func (e *FlexPayReconciliationExporter) BankInfoForStatement(ctx context.Context) appconfig.TransferBankInfo {
	return e.resolveBankInfo(ctx)
}

// GenerateExcel creates a multi-sheet Excel file with FlexPay reconciliation reports
// Uses the same template as the existing timesheet sao ke exporter
func (e *FlexPayReconciliationExporter) GenerateExcel(ctx context.Context, reportData []*domainServices.ProjectFlexPayReportData, atDate time.Time) ([]byte, *serviceports.PayrollReportSummary, error) {
	e.logger.Info("Starting Excel generation",
		"projectCount", len(reportData),
		"atDate", atDate.Format("2006-01-02"))

	// Open template file - using the same template path
	f, err := excelize.OpenFile(pkgConstants.PayrollReportByProjectTemplatePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open template: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			observability.GetLogger().Warn("failed to close Excel file", "error", closeErr)
		}
	}()

	excelService := excel.NewExportService()

	// Setup styles
	dataStyleWhite, err := excelService.SetupDataStyle(f, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create white data style: %w", err)
	}

	dataStyleGray, err := excelService.SetupDataStyle(f, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gray data style: %w", err)
	}

	centerStyleWhite, err := setupCenterMiddleStyle(f, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create white center style: %w", err)
	}

	centerStyleGray, err := setupCenterMiddleStyle(f, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gray center style: %w", err)
	}

	currencyStyleWhite, err := excelService.SetupCurrencyStyle(f, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create white currency style: %w", err)
	}

	currencyStyleGray, err := excelService.SetupCurrencyStyle(f, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gray currency style: %w", err)
	}

	// Process each project
	var totalAmountAllProjects int64
	var totalRequestedAmountAllProjects int64
	templateSheetName := "Sample Project"

	// Get template sheet index for copying
	templateIndex, err := f.GetSheetIndex(templateSheetName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get template sheet index: %w", err)
	}
	if err := excelService.RemoveTablesFromSheet(f, templateSheetName); err != nil {
		return nil, nil, fmt.Errorf("failed to remove template tables: %w", err)
	}

	projectSheetNames := make([]string, 0, len(reportData))
	for idx, projectData := range reportData {
		newSheetName := projectData.Project.Name
		if len(newSheetName) > 31 {
			// Excel sheet name limit is 31 characters
			newSheetName = newSheetName[:31]
		}
		projectSheetNames = append(projectSheetNames, newSheetName)

		if idx == 0 {
			continue
		}

		// Create a new sheet for this project
		sheetIndex, err := f.NewSheet(newSheetName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create sheet: %w", err)
		}

		// Copy ALL formatting from template (merged cells, row heights, styles, etc.)
		if err := f.CopySheet(templateIndex, sheetIndex); err != nil {
			return nil, nil, fmt.Errorf("failed to copy sheet: %w", err)
		}
	}

	if len(projectSheetNames) > 0 {
		if err := f.SetSheetName(templateSheetName, projectSheetNames[0]); err != nil {
			return nil, nil, fmt.Errorf("failed to rename template sheet: %w", err)
		}
	}

	for idx, projectData := range reportData {
		newSheetName := projectSheetNames[idx]

		e.logger.Info("Creating sheet for project",
			"projectID", projectData.Project.ID,
			"projectName", projectData.Project.Name,
			"sheetName", newSheetName,
			"employeeCount", projectData.EmployeeCount,
			"totalAmount", projectData.TotalAmount)

		sheetIndex, err := f.GetSheetIndex(newSheetName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get project sheet index: %w", err)
		}

		f.SetActiveSheet(sheetIndex)

		// Set column F width for payment date (YYYY-MM-DD format needs ~14 width)
		if err := f.SetColWidth(newSheetName, "F", "F", 14); err != nil {
			return nil, nil, fmt.Errorf("failed to set column F width: %w", err)
		}

		// Populate project sheet
		if err := e.populateProjectSheet(f, newSheetName, projectData, dataStyleWhite, dataStyleGray, centerStyleWhite, centerStyleGray, currencyStyleWhite, currencyStyleGray); err != nil {
			return nil, nil, fmt.Errorf("failed to populate project sheet: %w", err)
		}

		// Auto-size columns D and E based on content
		if err := e.autoSizeColumnsForSheet(f, newSheetName, projectData); err != nil {
			return nil, nil, fmt.Errorf("failed to auto-size columns: %w", err)
		}

		e.logger.Info("Successfully populated project sheet",
			"projectID", projectData.Project.ID,
			"projectName", projectData.Project.Name)

		totalAmountAllProjects += projectData.TotalAmount
		totalRequestedAmountAllProjects += projectData.TotalRequestedAmount
	}

	// Update Summary sheet
	summary := e.buildSummary(totalAmountAllProjects, totalRequestedAmountAllProjects, atDate)

	if err := e.updateSummarySheet(ctx, f, reportData, summary, atDate, currencyStyleWhite, currencyStyleGray); err != nil {
		return nil, nil, fmt.Errorf("failed to update summary sheet: %w", err)
	}

	summaryIndex, err := f.GetSheetIndex("Summary")
	if err != nil {
		summaryIndex = -1
	}

	// Write to buffer
	if err := addInternalSheet(f, collectRequestIDs(reportData)); err != nil {
		return nil, nil, fmt.Errorf("failed to add INTERNAL sheet: %w", err)
	}

	if summaryIndex >= 0 {
		f.SetActiveSheet(summaryIndex)
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write Excel file: %w", err)
	}
	excelBytes, err := excelService.SanitizeWorkbookXML(buffer.Bytes())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sanitize Excel file: %w", err)
	}

	e.logger.Info("Excel generation completed successfully",
		"totalProjects", len(reportData),
		"totalAmount", totalAmountAllProjects,
		"fileSize", len(excelBytes))

	return excelBytes, summary, nil
}

func (e *FlexPayReconciliationExporter) buildSummary(totalPaid int64, totalRequested int64, atDate time.Time) *serviceports.PayrollReportSummary {
	// Due date is fixed to 17th of the current month
	dueDate := time.Date(atDate.Year(), atDate.Month(), 17, 0, 0, 0, 0, atDate.Location())
	feeAmount := totalRequested - totalPaid

	return &serviceports.PayrollReportSummary{
		TotalAmount:    totalPaid,
		FeePercentage:  0,
		FeeAmount:      feeAmount,
		TotalWithFee:   totalRequested,
		DueDate:        dueDate,
		FormattedRange: atDate.Format("02/01/2006"),
	}
}

// populateProjectSheet fills in the project sheet with data
func (e *FlexPayReconciliationExporter) populateProjectSheet(
	f *excelize.File,
	sheetName string,
	projectData *domainServices.ProjectFlexPayReportData,
	dataStyleWhite, dataStyleGray, centerStyleWhite, centerStyleGray, currencyStyleWhite, currencyStyleGray int,
) error {
	// C1: Project name
	if err := f.SetCellValue(sheetName, "C1", projectData.Project.Name); err != nil {
		return err
	}

	// C2: Total distinct employees
	if err := f.SetCellValue(sheetName, "C2", projectData.EmployeeCount); err != nil {
		return err
	}

	// C3: Salary period (DD/MM -> DD/MM/YY)
	periodStr := fmt.Sprintf("%s -> %s",
		projectData.SalaryPeriodFrom.Format("02/01"),
		projectData.SalaryPeriodTo.Format("02/01/06"))
	if err := f.SetCellValue(sheetName, "C3", periodStr); err != nil {
		return err
	}

	// C4: Total money paid with Vietnamese currency format
	if err := f.SetCellValue(sheetName, "C4", projectData.TotalRequestedAmount); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetName, "C4", "C4", currencyStyleWhite); err != nil {
		return err
	}

	// Populate employee data starting from row 9
	for idx, employeeData := range projectData.EmployeeData {
		rowNum := 9 + idx
		styleID := dataStyleWhite
		currencyStyle := currencyStyleWhite
		centerStyle := centerStyleWhite
		if idx%2 == 1 {
			styleID = dataStyleGray
			currencyStyle = currencyStyleGray
			centerStyle = centerStyleGray
		}

		// A: STT (sequential number)
		cellA := fmt.Sprintf("A%d", rowNum)
		if err := f.SetCellValue(sheetName, cellA, idx+1); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheetName, cellA, cellA, styleID); err != nil {
			return err
		}

		// B: Employee Name
		cellB := fmt.Sprintf("B%d", rowNum)
		if err := f.SetCellValue(sheetName, cellB, employeeData.EmployeeName); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheetName, cellB, cellB, styleID); err != nil {
			return err
		}

		// C: Employee CCCD (center-middle)
		cellC := fmt.Sprintf("C%d", rowNum)
		if err := f.SetCellValue(sheetName, cellC, employeeData.EmployeeCCCD); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheetName, cellC, cellC, centerStyle); err != nil {
			return err
		}

		// D: Project name (center-middle)
		cellD := fmt.Sprintf("D%d", rowNum)
		if err := f.SetCellValue(sheetName, cellD, employeeData.ProjectName); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheetName, cellD, cellD, centerStyle); err != nil {
			return err
		}

		// E: Total money requested by employee with Vietnamese currency format (center-middle)
		cellE := fmt.Sprintf("E%d", rowNum)
		if err := f.SetCellValue(sheetName, cellE, employeeData.TotalRequested); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheetName, cellE, cellE, currencyStyle); err != nil {
			return err
		}

		// F: Payment date (center-middle)
		cellF := fmt.Sprintf("F%d", rowNum)
		if employeeData.PaymentDate != nil {
			if err := f.SetCellValue(sheetName, cellF, employeeData.PaymentDate.Format("02/01/2006")); err != nil {
				return err
			}
		} else {
			if err := f.SetCellValue(sheetName, cellF, ""); err != nil {
				return err
			}
		}
		if err := f.SetCellStyle(sheetName, cellF, cellF, centerStyle); err != nil {
			return err
		}
	}

	return nil
}

// updateSummarySheet updates the Summary sheet with totals and project breakdown
func (e *FlexPayReconciliationExporter) updateSummarySheet(
	ctx context.Context,
	f *excelize.File,
	reportData []*domainServices.ProjectFlexPayReportData,
	summary *serviceports.PayrollReportSummary,
	atDate time.Time,
	currencyStyleWhite, currencyStyleGray int,
) error {
	summarySheet := "Summary"
	// Override the static beneficiary bank rows with the configured values,
	// or clear both values and labels when the beneficiary block is hidden.
	bankInfo := e.resolveBankInfo(ctx)
	if !bankInfo.Hidden {
		if err := f.SetCellValue(summarySheet, "E8", bankInfo.Holder); err != nil {
			return fmt.Errorf("failed to set E8 bank holder: %w", err)
		}
		if err := f.SetCellValue(summarySheet, "E9", bankInfo.Number); err != nil {
			return fmt.Errorf("failed to set E9 bank number: %w", err)
		}
		if err := f.SetCellValue(summarySheet, "E10", bankInfo.Name); err != nil {
			return fmt.Errorf("failed to set E10 bank name: %w", err)
		}
	} else {
		// Clear values, labels, and the "TÀI KHOẢN THỤ HƯỞNG" heading row
		// directly above (D7/E7 in the sao ke Summary template) so no orphan
		// beneficiary banner prints over the blanked rows.
		for _, cell := range []string{"E8", "E9", "E10", "D8", "D9", "D10", "D7", "E7"} {
			if err := f.SetCellValue(summarySheet, cell, ""); err != nil {
				return fmt.Errorf("failed to clear bank cell %s: %w", cell, err)
			}
		}
	}
	whiteRowStyleID, err := e.buildSummaryRowStyle(f, false)
	if err != nil {
		return err
	}
	alternateRowStyleID, err := e.buildSummaryRowStyle(f, true)
	if err != nil {
		return err
	}

	e.logger.Info("Populating Summary sheet",
		"totalAmount", summary.TotalWithFee,
		"projectCount", len(reportData),
		"dueDate", summary.DueDate.Format("02/01/2006"))

	// B7:C8 is covered by the embedded provider banner in the workbook template.
	if err := f.SetCellValue(summarySheet, "B7", ""); err != nil {
		return err
	}

	// D2, E2: Tiền CTy Phải Trả
	if err := f.SetCellValue(summarySheet, "D2", "Tiền CTy Phải Trả"); err != nil {
		return err
	}
	if err := f.SetCellValue(summarySheet, "E2", summary.TotalWithFee); err != nil {
		return err
	}
	if err := f.SetCellStyle(summarySheet, "E2", "E2", currencyStyleWhite); err != nil {
		return err
	}

	// D3, E3: Hạn Thanh Toán
	if err := f.SetCellValue(summarySheet, "D3", "Hạn Thanh Toán"); err != nil {
		return err
	}
	if err := f.SetCellValue(summarySheet, "E3", summary.DueDate.Format("02/01/2006")); err != nil {
		return err
	}

	// Clear D4, D5, E4, E5
	if err := f.SetCellValue(summarySheet, "D4", ""); err != nil {
		return err
	}
	if err := f.SetCellValue(summarySheet, "E4", ""); err != nil {
		return err
	}
	if err := f.SetCellValue(summarySheet, "D5", ""); err != nil {
		return err
	}
	if err := f.SetCellValue(summarySheet, "E5", ""); err != nil {
		return err
	}

	// Populate project breakdown starting from row 15
	e.logger.Info("Populating project breakdown in Summary sheet",
		"startRow", 15,
		"projectCount", len(reportData))

	for idx, projectData := range reportData {
		rowNum := 15 + idx

		// A15+: STT (sequential number)
		cellA := fmt.Sprintf("A%d", rowNum)
		if err := f.SetCellValue(summarySheet, cellA, idx+1); err != nil {
			return err
		}

		// B15+: Salary period from (DD/MM/YYYY)
		cellB := fmt.Sprintf("B%d", rowNum)
		periodFrom := projectData.SalaryPeriodFrom.Format("02/01/2006")
		if err := f.SetCellValue(summarySheet, cellB, periodFrom); err != nil {
			return err
		}

		// C15+: Salary period to (DD/MM/YYYY)
		cellC := fmt.Sprintf("C%d", rowNum)
		periodTo := projectData.SalaryPeriodTo.Format("02/01/2006")
		if err := f.SetCellValue(summarySheet, cellC, periodTo); err != nil {
			return err
		}

		// D15+: Project name
		cellD := fmt.Sprintf("D%d", rowNum)
		if err := f.SetCellValue(summarySheet, cellD, projectData.Project.Name); err != nil {
			return err
		}

		// E15+: Total money requested with Vietnamese currency format
		cellE := fmt.Sprintf("E%d", rowNum)
		if err := f.SetCellValue(summarySheet, cellE, projectData.TotalRequestedAmount); err != nil {
			return err
		}

		// Apply professional styling with alternating background colors
		styleID := whiteRowStyleID
		currencyStyle := currencyStyleWhite
		if idx%2 == 1 {
			styleID = alternateRowStyleID
			currencyStyle = currencyStyleGray
		}
		// Apply regular style to columns A-D
		startCell := fmt.Sprintf("A%d", rowNum)
		endCell := fmt.Sprintf("D%d", rowNum)
		if err := f.SetCellStyle(summarySheet, startCell, endCell, styleID); err != nil {
			return err
		}
		// Apply currency style to column E
		if err := f.SetCellStyle(summarySheet, cellE, cellE, currencyStyle); err != nil {
			return err
		}
	}

	e.logger.Info("Summary sheet population completed",
		"projectBreakdownRows", len(reportData))

	return nil
}

// buildSummaryRowStyle creates styles for summary table rows with consistent typography
func (e *FlexPayReconciliationExporter) buildSummaryRowStyle(f *excelize.File, useAlternateColor bool) (int, error) {
	style := &excelize.Style{
		Font: &excelize.Font{
			Size:   12,
			Family: excel.DefaultFontName,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	}

	if useAlternateColor {
		style.Fill = excelize.Fill{
			Type:    "pattern",
			Color:   []string{excel.AlternateRowColor},
			Pattern: 1,
		}
	}

	return f.NewStyle(style)
}

func setupCenterMiddleStyle(f *excelize.File, useAlternateColor bool) (int, error) {
	style := &excelize.Style{
		Font: &excelize.Font{
			Size:   11,
			Family: "Calibri",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	}

	if useAlternateColor {
		style.Fill = excelize.Fill{
			Type:    "pattern",
			Color:   []string{"F2F2F2"},
			Pattern: 1,
		}
	}

	return f.NewStyle(style)
}

func (e *FlexPayReconciliationExporter) autoSizeColumnsForSheet(f *excelize.File, sheetName string, projectData *domainServices.ProjectFlexPayReportData) error {
	// Calculate max width for column D (Project Name)
	maxDWidth := float64(len("Project Name")) // header width
	for _, empData := range projectData.EmployeeData {
		if len(empData.ProjectName) > int(maxDWidth) {
			maxDWidth = float64(len(empData.ProjectName))
		}
	}
	widthD := maxDWidth*1.05 + 1
	if widthD < 10 {
		widthD = 10
	}
	if widthD > 50 {
		widthD = 50
	}

	// Columns C and E: Fixed width 15 for currency format
	widthCE := 15.0

	if err := f.SetColWidth(sheetName, "C", "C", widthCE); err != nil {
		return err
	}
	if err := f.SetColWidth(sheetName, "D", "D", widthD); err != nil {
		return err
	}
	if err := f.SetColWidth(sheetName, "E", "E", widthCE); err != nil {
		return err
	}

	return nil
}

func collectRequestIDs(reportData []*domainServices.ProjectFlexPayReportData) []uint {
	idSet := make(map[uint]struct{})

	for _, projectData := range reportData {
		if projectData == nil {
			continue
		}
		for _, tsID := range projectData.RequestIDs {
			if tsID == 0 {
				continue
			}
			idSet[tsID] = struct{}{}
		}
	}

	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	return ids
}

// addInternalSheet adds or replaces the INTERNAL sheet listing request IDs.
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
	if err := f.SetCellValue(internalSheetName, "A1", internalTypeAdvance); err != nil {
		return fmt.Errorf("failed to set type header: %w", err)
	}

	for i, reqID := range ids {
		cell := fmt.Sprintf("A%d", i+2)
		if err := f.SetCellValue(internalSheetName, cell, reqID); err != nil {
			return fmt.Errorf("failed to set cell %s: %w", cell, err)
		}
	}

	// Hide the INTERNAL sheet so it's not visible to users
	if err := f.SetSheetVisible(internalSheetName, false); err != nil {
		return fmt.Errorf("failed to hide %s sheet: %w", internalSheetName, err)
	}

	return nil
}
