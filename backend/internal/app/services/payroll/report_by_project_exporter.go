package payroll

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	appconfig "api-server/internal/app/services/config"
	"api-server/internal/app/services/excel"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	pkgConstants "api-server/internal/pkg/constants"

	"github.com/xuri/excelize/v2"
)

// TransferBankInfoProvider supplies beneficiary bank details for payroll
// statement Excel exports. Implemented by *config.SettingsConfigService.
type TransferBankInfoProvider interface {
	GetTransferBankInfo(ctx context.Context) appconfig.TransferBankInfo
}

// PayrollReportByProjectExporter handles Excel generation for payroll report by project
type PayrollReportByProjectExporter struct {
	logger       *slog.Logger
	bankSettings TransferBankInfoProvider
}

// NewPayrollReportByProjectExporter creates a new exporter
func NewPayrollReportByProjectExporter(bankSettings TransferBankInfoProvider) *PayrollReportByProjectExporter {
	return &PayrollReportByProjectExporter{
		logger:       observability.GetLogger(),
		bankSettings: bankSettings,
	}
}

// GenerateExcel creates a multi-sheet Excel file with project payroll reports
func (e *PayrollReportByProjectExporter) GenerateExcel(ctx context.Context, reportData []*domainServices.ProjectReportData, atDate time.Time) ([]byte, *PayrollReportSummary, error) {
	e.logger.Info("Starting Excel generation",
		"projectCount", len(reportData),
		"atDate", atDate.Format("2006-01-02"))

	// Open template file
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
	}

	// Update Summary sheet
	summary := e.buildSummary(totalAmountAllProjects, atDate)

	if err := e.updateSummarySheet(ctx, f, reportData, summary, atDate, currencyStyleWhite, currencyStyleGray); err != nil {
		return nil, nil, fmt.Errorf("failed to update summary sheet: %w", err)
	}

	summaryIndex, err := f.GetSheetIndex("Summary")
	if err != nil {
		summaryIndex = -1
	}

	// Write to buffer
	if err := addInternalSheet(f, collectProjectTimesheetIDs(reportData)); err != nil {
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

// resolveBankInfo returns the configured beneficiary bank details, falling
// back to defaults when no settings provider is bound.
func (e *PayrollReportByProjectExporter) resolveBankInfo(ctx context.Context) appconfig.TransferBankInfo {
	if e.bankSettings == nil {
		return appconfig.DefaultTransferBankInfo()
	}
	return e.bankSettings.GetTransferBankInfo(ctx)
}

func (e *PayrollReportByProjectExporter) buildSummary(totalAmount int64, atDate time.Time) *PayrollReportSummary {
	feePercentage := 0.02
	feeAmount := (totalAmount * 2) / 100
	totalWithFee := totalAmount + feeAmount

	return &PayrollReportSummary{
		TotalAmount:    totalAmount,
		FeePercentage:  feePercentage,
		FeeAmount:      feeAmount,
		TotalWithFee:   totalWithFee,
		DueDate:        atDate.AddDate(0, 0, 14),
		FormattedRange: atDate.Format("02/01/2006"),
	}
}

// populateProjectSheet fills in the project sheet with data
func (e *PayrollReportByProjectExporter) populateProjectSheet(
	f *excelize.File,
	sheetName string,
	projectData *domainServices.ProjectReportData,
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
	if err := f.SetCellValue(sheetName, "C4", projectData.TotalAmount); err != nil {
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

		// E: Total money paid to employee with Vietnamese currency format (center-middle)
		cellE := fmt.Sprintf("E%d", rowNum)
		if err := f.SetCellValue(sheetName, cellE, employeeData.TotalPaid); err != nil {
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

	// Timesheet detail breakdown
	if len(projectData.Timesheets) > 0 {
		detailStartRow := 9 + len(projectData.EmployeeData) + 2

		// Section header style: dark blue bold with bottom border
		sectionHeaderStyle, err := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Size: 12, Family: "Calibri", Color: "2F5496"},
			Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
			Border: []excelize.Border{
				{Type: "bottom", Color: "2F5496", Style: 2},
			},
		})
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", detailStartRow), "CHI TIẾT CHẤM CÔNG"); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", detailStartRow), fmt.Sprintf("H%d", detailStartRow), sectionHeaderStyle); err != nil {
			return err
		}
		if err := f.SetRowHeight(sheetName, detailStartRow, 28); err != nil {
			return err
		}

		// Column header style: dark blue bg with white bold text
		colHeaderStyle, err := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Size: 10, Family: "Calibri", Color: "FFFFFF"},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"2F5496"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
			Border: []excelize.Border{
				{Type: "left", Color: "1F3864", Style: 1},
				{Type: "top", Color: "1F3864", Style: 1},
				{Type: "bottom", Color: "1F3864", Style: 1},
				{Type: "right", Color: "1F3864", Style: 1},
			},
		})
		if err != nil {
			return err
		}

		// Column headers: STT | Họ tên | Ngày công | Vị trí | Ngày làm | Ca làm | Giờ | Số tiền
		colHeaderRow := detailStartRow + 1
		colHeaders := []string{"STT", "Họ tên", "Ngày công", "Vị trí", "Ngày làm", "Ca làm", "Giờ", "Số tiền"}
		for colIdx, header := range colHeaders {
			col := string(rune('A' + colIdx))
			cell := fmt.Sprintf("%s%d", col, colHeaderRow)
			if err := f.SetCellValue(sheetName, cell, header); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetName, cell, cell, colHeaderStyle); err != nil {
				return err
			}
		}

		// Sort timesheets by work date, then employee name
		sortedTimesheets := make([]*domain.Timesheet, len(projectData.Timesheets))
		copy(sortedTimesheets, projectData.Timesheets)
		sort.Slice(sortedTimesheets, func(i, j int) bool {
			if !sortedTimesheets[i].Date.Equal(sortedTimesheets[j].Date) {
				return sortedTimesheets[i].Date.Before(sortedTimesheets[j].Date)
			}
			nameI := ""
			nameJ := ""
			if sortedTimesheets[i].Employee != nil {
				nameI = sortedTimesheets[i].Employee.FormattedFullname()
			}
			if sortedTimesheets[j].Employee != nil {
				nameJ = sortedTimesheets[j].Employee.FormattedFullname()
			}
			return nameI < nameJ
		})

		// Detail rows
		for idx, ts := range sortedTimesheets {
			rowNum := colHeaderRow + 1 + idx
			detailStyle := dataStyleWhite
			detailCurrency := currencyStyleWhite
			detailCenter := centerStyleWhite
			if idx%2 == 1 {
				detailStyle = dataStyleGray
				detailCurrency = currencyStyleGray
				detailCenter = centerStyleGray
			}

			// A: STT
			cellA := fmt.Sprintf("A%d", rowNum)
			if err := f.SetCellValue(sheetName, cellA, idx+1); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetName, cellA, cellA, detailCenter); err != nil {
				return err
			}

			// B: Employee name
			cellB := fmt.Sprintf("B%d", rowNum)
			empName := ""
			if ts.Employee != nil {
				empName = ts.Employee.FormattedFullname()
			}
			if err := f.SetCellValue(sheetName, cellB, empName); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetName, cellB, cellB, detailStyle); err != nil {
				return err
			}

			// C: Work date
			cellC := fmt.Sprintf("C%d", rowNum)
			if err := f.SetCellValue(sheetName, cellC, ts.Date.Format("02/01/2006")); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetName, cellC, cellC, detailCenter); err != nil {
				return err
			}

			// Split paytype "position.dayType.shift" into 3 parts
			parts := strings.Split(ts.PayType, ".")
			viTri := ""
			ngayLam := ""
			caLam := ""
			for i, p := range parts {
				switch i {
				case 0:
					viTri = p
				case 1:
					ngayLam = p
				case 2:
					caLam = p
				}
			}

			// D: Vị trí (position)
			cellD := fmt.Sprintf("D%d", rowNum)
			if err := f.SetCellValue(sheetName, cellD, viTri); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetName, cellD, cellD, detailCenter); err != nil {
				return err
			}

			// E: Ngày làm (day type)
			cellE := fmt.Sprintf("E%d", rowNum)
			if err := f.SetCellValue(sheetName, cellE, ngayLam); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetName, cellE, cellE, detailCenter); err != nil {
				return err
			}

			// F: Ca làm (shift)
			cellF := fmt.Sprintf("F%d", rowNum)
			if err := f.SetCellValue(sheetName, cellF, caLam); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetName, cellF, cellF, detailCenter); err != nil {
				return err
			}

			// G: Hours worked
			cellG := fmt.Sprintf("G%d", rowNum)
			if err := f.SetCellValue(sheetName, cellG, ts.HoursWorked); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetName, cellG, cellG, detailCenter); err != nil {
				return err
			}

			// H: Paid amount
			cellH := fmt.Sprintf("H%d", rowNum)
			if err := f.SetCellValue(sheetName, cellH, ts.PaidAmount); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetName, cellH, cellH, detailCurrency); err != nil {
				return err
			}
		}

		// Set column widths for detail columns
		if err := f.SetColWidth(sheetName, "D", "D", 14); err != nil {
			return err
		}
		if err := f.SetColWidth(sheetName, "E", "E", 16); err != nil {
			return err
		}
		if err := f.SetColWidth(sheetName, "F", "F", 18); err != nil {
			return err
		}
		if err := f.SetColWidth(sheetName, "G", "G", 8); err != nil {
			return err
		}
		if err := f.SetColWidth(sheetName, "H", "H", 15); err != nil {
			return err
		}
	}

	return nil
}

// updateSummarySheet updates the Summary sheet with totals and project breakdown
func (e *PayrollReportByProjectExporter) updateSummarySheet(
	ctx context.Context,
	f *excelize.File,
	reportData []*domainServices.ProjectReportData,
	summary *PayrollReportSummary,
	atDate time.Time,
	currencyStyleWhite, currencyStyleGray int,
) error {
	summarySheet := "Summary"
	// Override the static beneficiary bank rows with the configured values.
	if err := writeBankInfoCells(f, summarySheet, "E8", "E9", "E10", e.resolveBankInfo(ctx)); err != nil {
		return fmt.Errorf("failed to write bank info cells: %w", err)
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
		"totalAmount", summary.TotalAmount,
		"projectCount", len(reportData),
		"dueDate", summary.DueDate.Format("02/01/2006"))

	// E2: Total amount paid to all projects with Vietnamese currency format
	if err := f.SetCellValue(summarySheet, "E2", summary.TotalAmount); err != nil {
		return err
	}
	if err := f.SetCellStyle(summarySheet, "E2", "E2", currencyStyleWhite); err != nil {
		return err
	}

	// E3: fee amount with Vietnamese currency format
	if err := f.SetCellValue(summarySheet, "E3", summary.FeeAmount); err != nil {
		return err
	}
	if err := f.SetCellStyle(summarySheet, "E3", "E3", currencyStyleWhite); err != nil {
		return err
	}

	// E4: E2 + E3 with Vietnamese currency format
	if err := f.SetCellValue(summarySheet, "E4", summary.TotalWithFee); err != nil {
		return err
	}
	if err := f.SetCellStyle(summarySheet, "E4", "E4", currencyStyleWhite); err != nil {
		return err
	}

	// E5: atDate + 14 days (formatted as DD/MM/YYYY)
	if err := f.SetCellValue(summarySheet, "E5", summary.DueDate.Format("02/01/2006")); err != nil {
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

		// E15+: Total money paid with Vietnamese currency format
		cellE := fmt.Sprintf("E%d", rowNum)
		if err := f.SetCellValue(summarySheet, cellE, projectData.TotalAmount); err != nil {
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
func (e *PayrollReportByProjectExporter) buildSummaryRowStyle(f *excelize.File, useAlternateColor bool) (int, error) {
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

func (e *PayrollReportByProjectExporter) autoSizeColumnsForSheet(f *excelize.File, sheetName string, projectData *domainServices.ProjectReportData) error {
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

	// Columns C and E: Fixed width 14 for currency format #,##0 ₫
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

func collectProjectTimesheetIDs(reportData []*domainServices.ProjectReportData) []uint {
	idSet := make(map[uint]struct{})

	for _, projectData := range reportData {
		if projectData == nil {
			continue
		}
		for _, tsID := range projectData.TimesheetIDs {
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
