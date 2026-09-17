package payroll

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	appconfig "api-server/internal/app/services/config"
	"api-server/internal/app/services/excel"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	pkgConstants "api-server/internal/pkg/constants"
	"api-server/internal/pkg/timeutil"

	"github.com/xuri/excelize/v2"
)

// PayrollReportSummary captures key figures needed for presentation layers.
type PayrollReportSummary struct {
	TotalAmount    int64
	FeePercentage  float64
	FeeAmount      int64
	TotalWithFee   int64
	DueDate        time.Time
	FormattedRange string
}

// PayrollReportExcelData holds the Excel rows and associated timesheet IDs.
type PayrollReportExcelData struct {
	Rows         [][]interface{}
	TimesheetIDs []uint
}

// SettingsConfigProvider exposes the subset of settings config methods required by the exporter.
type SettingsConfigProvider interface {
	GetWeeklyPaymentPercentage(ctx context.Context) float64
	GetWeeklyPaymentFeePercentage(ctx context.Context) float64
	GetTransferBankInfo(ctx context.Context) appconfig.TransferBankInfo
}

// PayrollReportExporter centralizes payroll report Excel generation so HTTP handlers
// and email services reuse the same implementation.
type PayrollReportExporter struct {
	settingsConfigService SettingsConfigProvider
}

// NewPayrollReportExporter builds a new exporter.
func NewPayrollReportExporter(settingsConfigService SettingsConfigProvider) *PayrollReportExporter {
	return &PayrollReportExporter{settingsConfigService: settingsConfigService}
}

type payrollAggregate struct {
	name      string
	cccd      string
	date      time.Time
	projects  map[string]struct{}
	paidTotal int64
}

// BuildPayrollReportData aggregates report entries into the Excel-friendly structure.
func (e *PayrollReportExporter) BuildPayrollReportData(ctx context.Context, entries []*domainServices.PayrollReportEntry) *PayrollReportExcelData {
	aggregates := map[string]*payrollAggregate{}
	timesheetIDSet := make(map[uint]struct{})

	for _, entry := range entries {
		if entry == nil || entry.PaymentDate == nil {
			continue
		}

		for _, tsID := range entry.TimesheetIDs {
			if tsID == 0 {
				continue
			}
			timesheetIDSet[tsID] = struct{}{}
		}

		empKey := entry.EmployeeCCCD
		if empKey == "" {
			empKey = entry.EmployeeName
		}

		dateOnly := entry.PaymentDate.Format("2006-01-02")
		key := empKey + "|" + dateOnly

		aggregate, ok := aggregates[key]
		if !ok {
			aggregate = &payrollAggregate{
				name:     entry.EmployeeName,
				cccd:     entry.EmployeeCCCD,
				date:     timeutil.StartOfDay(*entry.PaymentDate),
				projects: map[string]struct{}{},
			}
			aggregates[key] = aggregate
		}

		aggregate.paidTotal += entry.PaidAmount
		if entry.ProjectName != "" {
			aggregate.projects[entry.ProjectName] = struct{}{}
		}
	}

	items := make([]*payrollAggregate, 0, len(aggregates))
	for _, v := range aggregates {
		items = append(items, v)
	}

	sortAggregated(items)

	excelService := excel.NewExportService()
	var data [][]interface{}

	for idx, ag := range items {
		date := ag.date
		paymentDate := excelService.FormatDate(&date)

		projectNames := make([]string, 0, len(ag.projects))
		for project := range ag.projects {
			projectNames = append(projectNames, project)
		}
		sortStrings(projectNames)

		row := []interface{}{idx + 1, paymentDate, ag.name, ag.cccd, strings.Join(projectNames, ", "), ag.paidTotal}
		data = append(data, row)
	}

	sortedTimesheetIDs := make([]uint, 0, len(timesheetIDSet))
	for tsID := range timesheetIDSet {
		sortedTimesheetIDs = append(sortedTimesheetIDs, tsID)
	}
	sort.Slice(sortedTimesheetIDs, func(i, j int) bool {
		return sortedTimesheetIDs[i] < sortedTimesheetIDs[j]
	})

	return &PayrollReportExcelData{
		Rows:         data,
		TimesheetIDs: sortedTimesheetIDs,
	}
}

// GenerateExcel assembles the payroll report Excel file and returns its bytes along
// with a summary of monetary figures for downstream use (e.g., email template).
func (e *PayrollReportExporter) GenerateExcel(ctx context.Context, fromDate, toDate time.Time, data *PayrollReportExcelData) ([]byte, *PayrollReportSummary, error) {
	f, err := excelize.OpenFile(pkgConstants.PayrollReportTemplatePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open payroll template: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			observability.GetLogger().Warn("failed to close Excel file", "error", closeErr)
		}
	}()

	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return nil, nil, fmt.Errorf("payroll template has no sheets")
	}
	sheetName := sheetList[0]
	mainSheetIndex, _ := f.GetSheetIndex(sheetName)
	excelService := excel.NewExportService()

	// Override the static beneficiary bank rows with the configured values.
	bankInfo := e.resolveBankInfo(ctx)
	if err := writeBankInfoCells(f, sheetName, "E9", "E10", "E11", bankInfo); err != nil {
		return nil, nil, fmt.Errorf("failed to write bank info cells: %w", err)
	}

	dataStyleWhite, err := excelService.SetupDataStyle(f, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create white data style: %w", err)
	}

	dataStyleGray, err := excelService.SetupDataStyle(f, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gray data style: %w", err)
	}

	currencyStyleWhite, err := excelService.SetupCurrencyStyle(f, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create white currency style: %w", err)
	}

	currencyStyleGray, err := excelService.SetupCurrencyStyle(f, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gray currency style: %w", err)
	}

	summaryCurrencyStyle, err := excelService.SetupCurrencyStyle(f, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create summary currency style: %w", err)
	}

	dateRangeValue := fmt.Sprintf("%s - %s", fromDate.Format("02/01"), toDate.Format("02/01/2006"))
	if err := f.SetCellValue(sheetName, "E2", dateRangeValue); err != nil {
		return nil, nil, fmt.Errorf("failed to set E2 date range: %w", err)
	}

	if data == nil {
		data = &PayrollReportExcelData{}
	}

	var totalAmount int64
	for _, row := range data.Rows {
		if len(row) >= 6 {
			if amount, ok := row[5].(int64); ok {
				totalAmount += amount
			}
		}
	}

	if err := f.SetCellValue(sheetName, "E3", totalAmount); err != nil {
		return nil, nil, fmt.Errorf("failed to set E3 total amount: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "E3", "E3", summaryCurrencyStyle); err != nil {
		return nil, nil, fmt.Errorf("failed to set E3 currency style: %w", err)
	}

	feePercentage := e.settingsConfigService.GetWeeklyPaymentFeePercentage(ctx)
	feeAmount := int64(float64(totalAmount) * feePercentage)
	if err := f.SetCellValue(sheetName, "E4", feeAmount); err != nil {
		return nil, nil, fmt.Errorf("failed to set E4 advance cash fee: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "E4", "E4", summaryCurrencyStyle); err != nil {
		return nil, nil, fmt.Errorf("failed to set E4 currency style: %w", err)
	}

	totalWithFee := totalAmount + feeAmount
	if err := f.SetCellValue(sheetName, "E5", totalWithFee); err != nil {
		return nil, nil, fmt.Errorf("failed to set E5 total with fee: %w", err)
	}
	if err := f.SetCellStyle(sheetName, "E5", "E5", summaryCurrencyStyle); err != nil {
		return nil, nil, fmt.Errorf("failed to set E5 currency style: %w", err)
	}

	// Calculate the 16th of the month following the payment cycle
	// Use month arithmetic to avoid date overflow issues (e.g., Oct 31 + 1 month = Dec 1)
	nextMonth := toDate.Month() + 1
	nextYear := toDate.Year()
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	sixteenthNextMonth := time.Date(nextYear, nextMonth, 16, 0, 0, 0, 0, toDate.Location())
	if err := f.SetCellValue(sheetName, "E6", sixteenthNextMonth.Format("02/01/2006")); err != nil {
		return nil, nil, fmt.Errorf("failed to set E6 date: %w", err)
	}

	for i, rowData := range data.Rows {
		currentRow := 16 + i
		styleID := dataStyleWhite
		currencyStyle := currencyStyleWhite
		if i%2 == 1 {
			styleID = dataStyleGray
			currencyStyle = currencyStyleGray
		}

		for colIdx, val := range rowData {
			columnName := excelService.GetColumnName(colIdx)
			cell := fmt.Sprintf("%s%d", columnName, currentRow)
			if err := f.SetCellValue(sheetName, cell, val); err != nil {
				return nil, nil, fmt.Errorf("failed to set cell %s: %w", cell, err)
			}
			// Apply currency style to column F (index 5)
			cellStyleID := styleID
			if colIdx == 5 {
				cellStyleID = currencyStyle
			}
			if err := f.SetCellStyle(sheetName, cell, cell, cellStyleID); err != nil {
				return nil, nil, fmt.Errorf("failed to set style for cell %s: %w", cell, err)
			}
		}
	}

	if err := addInternalSheet(f, data.TimesheetIDs); err != nil {
		return nil, nil, fmt.Errorf("failed to add INTERNAL sheet: %w", err)
	}

	if mainSheetIndex >= 0 {
		f.SetActiveSheet(mainSheetIndex)
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write payroll report: %w", err)
	}

	summary := &PayrollReportSummary{
		TotalAmount:    totalAmount,
		FeePercentage:  feePercentage,
		FeeAmount:      feeAmount,
		TotalWithFee:   totalWithFee,
		DueDate:        sixteenthNextMonth,
		FormattedRange: dateRangeValue,
	}

	return buffer.Bytes(), summary, nil
}

// resolveBankInfo returns the configured beneficiary bank details, falling
// back to defaults when no settings provider is bound.
func (e *PayrollReportExporter) resolveBankInfo(ctx context.Context) appconfig.TransferBankInfo {
	if e.settingsConfigService == nil {
		return appconfig.DefaultTransferBankInfo()
	}
	return e.settingsConfigService.GetTransferBankInfo(ctx)
}

// writeBankInfoCells writes the beneficiary holder, account number, and bank
// name into the template's static bank rows. When the beneficiary block is
// hidden (bankInfo.Hidden == true), the value cells are cleared together
// with the template's static label cells in column D of the same rows and
// the section heading row directly above the holder row, so the statement
// prints no receiving-account information.
func writeBankInfoCells(f *excelize.File, sheetName, holderCell, numberCell, nameCell string, bankInfo appconfig.TransferBankInfo) error {
	cells := map[string]string{
		holderCell: bankInfo.Holder,
		numberCell: bankInfo.Number,
		nameCell:   bankInfo.Name,
	}
	if bankInfo.Hidden {
		for cell := range cells {
			if err := clearBankInfoCell(f, sheetName, cell); err != nil {
				return err
			}
		}
		// The heading sits one row above the holder row in both statement
		// templates (payroll_template.xlsx D8/E8 over E9-E11, the sao ke
		// Summary sheet D7/E7 over E8-E10); leaving it would print an orphan
		// "TÀI KHOẢN THỤ HƯỞNG" banner over the blanked rows.
		row, _, err := excelize.CellNameToCoordinates(holderCell)
		if err != nil {
			return fmt.Errorf("failed to parse holder cell %s: %w", holderCell, err)
		}
		headingRow := strconv.Itoa(row - 1)
		for _, col := range []string{"D", "E"} {
			if err := f.SetCellValue(sheetName, col+headingRow, ""); err != nil {
				return fmt.Errorf("failed to clear bank heading %s%s: %w", col, headingRow, err)
			}
		}
		return nil
	}
	for cell, value := range cells {
		if err := f.SetCellValue(sheetName, cell, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", cell, err)
		}
	}
	return nil
}

// clearBankInfoCell empties a beneficiary value cell plus the static template
// label in column D of the same row. Value cells are always in column E in
// both statement templates, so the label cell is D + row.
func clearBankInfoCell(f *excelize.File, sheetName, cell string) error {
	labelCell := "D" + cell[1:]
	if err := f.SetCellValue(sheetName, cell, ""); err != nil {
		return fmt.Errorf("failed to clear %s: %w", cell, err)
	}
	if err := f.SetCellValue(sheetName, labelCell, ""); err != nil {
		return fmt.Errorf("failed to clear label %s: %w", labelCell, err)
	}
	return nil
}

func sortStrings(values []string) {
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
}

func sortAggregated(items []*payrollAggregate) {
	sort.Slice(items, func(i, j int) bool {
		ai, aj := items[i], items[j]
		if ai.date.Equal(aj.date) {
			if ai.name == aj.name {
				return ai.cccd < aj.cccd
			}
			return ai.name < aj.name
		}
		return ai.date.After(aj.date)
	})
}
