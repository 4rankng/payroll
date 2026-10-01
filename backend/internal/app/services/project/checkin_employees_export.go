package project

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"

	"api-server/internal/domain"
)

// checkInExportSheet is the worksheet name of the self check-in roster.
const checkInExportSheet = "DSC"

// ExportCheckInEmployees builds the admin's printable self check-in roster for a
// project: full name, CCCD, mobile and the day the service started.
//
// It reuses the check-in configuration query (same status filter, search and
// month window the admin is looking at) but drops pagination, so the workbook
// holds the whole filtered group. Rows are sorted by name: the list query
// orders by attendance volume, which is a screen concern, not a roster one.
func (s *ProjectEmployeeService) ExportCheckInEmployees(
	ctx context.Context,
	projectID uint,
	status domain.CheckInConfigurationStatus,
	search string,
	selectedMonth time.Time,
) (*excelize.File, error) {
	monthStart, monthEnd, asOfDate := selectedCheckInMonthWindow(selectedMonth)
	result, err := s.projectEmployeeRepo.GetCheckInConfiguration(ctx, domain.CheckInConfigurationQuery{
		ProjectID:  projectID,
		MonthStart: monthStart,
		MonthEnd:   monthEnd,
		AsOfDate:   asOfDate,
		Status:     status,
		Search:     search,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load check-in employees for export")
	}

	employees := result.Employees
	if len(employees) == 0 {
		return nil, domain.NewValidationError("Không có nhân viên nào trong nhóm đã chọn")
	}
	sort.SliceStable(employees, func(i, j int) bool {
		return employees[i].EmployeeName < employees[j].EmployeeName
	})

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load project for export")
	}

	return buildCheckInRosterWorkbook(project, employees), nil
}

// buildCheckInRosterWorkbook renders the roster sheet. It is a pure function so
// the layout is verifiable without a database.
func buildCheckInRosterWorkbook(project *domain.Project, employees []domain.CheckInConfigurationEmployee) *excelize.File {
	f := excelize.NewFile()
	index, err := f.NewSheet(checkInExportSheet)
	if err != nil {
		// A brand-new file always accepts this sheet name; fall back to the
		// default sheet rather than returning a nil workbook to the handler.
		index = f.GetActiveSheetIndex()
	}
	f.SetActiveSheet(index)
	_ = f.DeleteSheet("Sheet1")

	const lastColumn = "E"

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "1F497D"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Size: 12, Color: "595959"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	borderParams := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"2F75B5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    borderParams,
	})
	centerDataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    borderParams,
	})
	leftDataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    borderParams,
	})

	_ = f.SetCellValue(checkInExportSheet, "A1", "DANH SÁCH NHÂN VIÊN TỰ CHẤM CÔNG")
	_ = f.MergeCell(checkInExportSheet, "A1", lastColumn+"1")
	_ = f.SetCellStyle(checkInExportSheet, "A1", lastColumn+"1", titleStyle)
	_ = f.SetRowHeight(checkInExportSheet, 1, 24)

	projectLabel := project.Name
	if project.Code != "" {
		projectLabel = fmt.Sprintf("%s (%s)", project.Name, project.Code)
	}
	_ = f.SetCellValue(checkInExportSheet, "A2", fmt.Sprintf("Dự án: %s", projectLabel))
	_ = f.MergeCell(checkInExportSheet, "A2", lastColumn+"2")
	_ = f.SetCellStyle(checkInExportSheet, "A2", lastColumn+"2", subtitleStyle)
	_ = f.SetRowHeight(checkInExportSheet, 2, 18)

	headers := []string{"STT", "Họ và tên", "CCCD", "Số điện thoại", "Ngày bắt đầu tự chấm công"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		_ = f.SetCellValue(checkInExportSheet, cell, header)
	}
	_ = f.SetCellStyle(checkInExportSheet, "A3", lastColumn+"3", headerStyle)
	_ = f.SetRowHeight(checkInExportSheet, 3, 22)

	_ = f.SetPanes(checkInExportSheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      3,
		TopLeftCell: "A4",
		ActivePane:  "bottomLeft",
	})

	for i, employee := range employees {
		row := i + 4
		_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("A%d", row), i+1)
		_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("B%d", row), employee.EmployeeName)
		_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("C%d", row), employee.EmployeeCCCD)
		_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("D%d", row), employee.EmployeeMobile)
		// A start date the row never recorded is genuinely unknown (enabled
		// before the column existed, with no attendance to recover it from);
		// leave the cell empty rather than printing a made-up day.
		if employee.CheckInStartDate != nil {
			_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("E%d", row), employee.CheckInStartDate.Format("02/01/2006"))
		}
		_ = f.SetRowHeight(checkInExportSheet, row, 20)

		_ = f.SetCellStyle(checkInExportSheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), centerDataStyle)
		_ = f.SetCellStyle(checkInExportSheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), leftDataStyle)
		_ = f.SetCellStyle(checkInExportSheet, fmt.Sprintf("C%d", row), fmt.Sprintf("E%d", row), centerDataStyle)
	}

	_ = f.SetColWidth(checkInExportSheet, "A", "A", 6)
	_ = f.SetColWidth(checkInExportSheet, "B", "B", 28)
	_ = f.SetColWidth(checkInExportSheet, "C", "C", 18)
	_ = f.SetColWidth(checkInExportSheet, "D", "D", 16)
	_ = f.SetColWidth(checkInExportSheet, "E", "E", 24)

	return f
}
