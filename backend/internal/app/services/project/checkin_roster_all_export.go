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

// ExportAllCheckInRosters builds the partner-facing self check-in roster across
// EVERY project (query.ProjectID = 0), for the day-9 reminder job that prepares
// the file the admin downloads from Tổng quan and forwards to VFIC.
//
// Unlike the per-project export the configuration screen offers, this one adds
// a "Dự án" column so a single workbook stays unambiguous when the program runs
// on several projects. Empty selection returns (nil, 0, nil) instead of a
// validation error: for an automated job "nobody is enrolled" is a nothing-to-do
// signal, not a failure.
func (s *ProjectEmployeeService) ExportAllCheckInRosters(
	ctx context.Context,
	status domain.CheckInConfigurationStatus,
	selectedMonth time.Time,
) (*excelize.File, int, error) {
	monthStart, monthEnd, asOfDate := selectedCheckInMonthWindow(selectedMonth)
	result, err := s.projectEmployeeRepo.GetCheckInConfiguration(ctx, domain.CheckInConfigurationQuery{
		ProjectID:  0, // all projects
		MonthStart: monthStart,
		MonthEnd:   monthEnd,
		AsOfDate:   asOfDate,
		Status:     status,
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to load check-in roster for export")
	}
	employees := result.Employees
	if len(employees) == 0 {
		return nil, 0, nil
	}

	projectIDs := make([]uint, 0, len(employees))
	seen := map[uint]bool{}
	for _, employee := range employees {
		if !seen[employee.ProjectID] {
			seen[employee.ProjectID] = true
			projectIDs = append(projectIDs, employee.ProjectID)
		}
	}
	projects, err := s.projectRepo.GetByIDs(ctx, projectIDs)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to load projects for roster export")
	}

	// Sort by project name then employee name: the partner reads the file
	// grouped by project, and within a project alphabetically — neither the
	// list query's attendance ordering nor raw assignment order helps here.
	sort.SliceStable(employees, func(i, j int) bool {
		pi, pj := projects[employees[i].ProjectID], projects[employees[j].ProjectID]
		piName, pjName := "", ""
		if pi != nil {
			piName = pi.Name
		}
		if pj != nil {
			pjName = pj.Name
		}
		if piName != pjName {
			return piName < pjName
		}
		return employees[i].EmployeeName < employees[j].EmployeeName
	})

	return buildAllCheckInRostersWorkbook(projects, employees, selectedMonth), len(employees), nil
}

// buildAllCheckInRostersWorkbook renders the combined roster sheet with a
// per-row project column. Pure function, testable without a database.
func buildAllCheckInRostersWorkbook(
	projects map[uint]*domain.Project,
	employees []domain.CheckInConfigurationEmployee,
	period time.Time,
) *excelize.File {
	f := excelize.NewFile()
	index, err := f.NewSheet(checkInExportSheet)
	if err != nil {
		index = f.GetActiveSheetIndex()
	}
	f.SetActiveSheet(index)
	_ = f.DeleteSheet("Sheet1")

	const lastColumn = "F"

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

	_ = f.SetCellValue(checkInExportSheet, "A2", fmt.Sprintf("Kỳ: %s · Tất cả dự án", period.Format("01/2006")))
	_ = f.MergeCell(checkInExportSheet, "A2", lastColumn+"2")
	_ = f.SetCellStyle(checkInExportSheet, "A2", lastColumn+"2", subtitleStyle)
	_ = f.SetRowHeight(checkInExportSheet, 2, 18)

	headers := []string{"STT", "Dự án", "Họ và tên", "CCCD", "Số điện thoại", "Ngày bắt đầu tự chấm công"}
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
		projectName := ""
		if project := projects[employee.ProjectID]; project != nil {
			projectName = project.Name
		}
		_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("A%d", row), i+1)
		_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("B%d", row), projectName)
		_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("C%d", row), employee.EmployeeName)
		_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("D%d", row), employee.EmployeeCCCD)
		_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("E%d", row), employee.EmployeeMobile)
		if employee.CheckInStartDate != nil {
			_ = f.SetCellValue(checkInExportSheet, fmt.Sprintf("F%d", row), employee.CheckInStartDate.Format("02/01/2006"))
		}
		_ = f.SetRowHeight(checkInExportSheet, row, 20)

		_ = f.SetCellStyle(checkInExportSheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), centerDataStyle)
		_ = f.SetCellStyle(checkInExportSheet, fmt.Sprintf("B%d", row), fmt.Sprintf("C%d", row), leftDataStyle)
		_ = f.SetCellStyle(checkInExportSheet, fmt.Sprintf("D%d", row), fmt.Sprintf("F%d", row), centerDataStyle)
	}

	_ = f.SetColWidth(checkInExportSheet, "A", "A", 6)
	_ = f.SetColWidth(checkInExportSheet, "B", "B", 24)
	_ = f.SetColWidth(checkInExportSheet, "C", "C", 28)
	_ = f.SetColWidth(checkInExportSheet, "D", "D", 18)
	_ = f.SetColWidth(checkInExportSheet, "E", "E", 16)
	_ = f.SetColWidth(checkInExportSheet, "F", "F", 24)

	return f
}
