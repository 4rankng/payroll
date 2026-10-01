package project

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
)

type checkInExportRepo struct {
	domain.ProjectEmployeeRepository
	result *domain.CheckInConfigurationResult
	query  domain.CheckInConfigurationQuery
}

func (f *checkInExportRepo) GetCheckInConfiguration(_ context.Context, query domain.CheckInConfigurationQuery) (*domain.CheckInConfigurationResult, error) {
	f.query = query
	return f.result, nil
}

type checkInExportProjectRepo struct {
	domain.ProjectRepository
	project *domain.Project
}

func (f *checkInExportProjectRepo) GetByID(_ context.Context, _ uint) (*domain.Project, error) {
	return f.project, nil
}

func newCheckInExportService(repo domain.ProjectEmployeeRepository, projectRepo domain.ProjectRepository) *ProjectEmployeeService {
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})
	svc.projectRepo = projectRepo
	return svc
}

func TestBuildCheckInRosterWorkbookWritesRequestedColumns(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	f := buildCheckInRosterWorkbook(
		&domain.Project{Name: "Nhà máy Hải Phòng", Code: "HP1"},
		[]domain.CheckInConfigurationEmployee{
			{
				EmployeeName:     "Nguyễn Văn An",
				EmployeeCCCD:     "001201000101",
				EmployeeMobile:   "0366178061",
				CheckInStartDate: &start,
			},
			{
				EmployeeName:   "Trần Thị Bình",
				EmployeeCCCD:   "001201000102",
				EmployeeMobile: "0987654321",
			},
		},
	)

	rows, err := f.GetRows(checkInExportSheet)
	if err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("rows = %d (title + subtitle + header + 2 employees), want 5", len(rows))
	}
	if got := rows[2]; len(got) != 5 || got[0] != "STT" || got[1] != "Họ và tên" || got[2] != "CCCD" ||
		got[3] != "Số điện thoại" || got[4] != "Ngày bắt đầu tự chấm công" {
		t.Fatalf("header row = %v", got)
	}
	if got := rows[3]; got[1] != "Nguyễn Văn An" || got[2] != "001201000101" ||
		got[3] != "0366178061" || got[4] != "01/08/2026" {
		t.Errorf("employee row = %v", got)
	}
	// An employee enabled before the start date existed has no known start day;
	// the cell must stay empty rather than show a fabricated date.
	withoutStart := rows[4]
	if withoutStart[1] != "Trần Thị Bình" || withoutStart[3] != "0987654321" {
		t.Errorf("employee row without a start date = %v", withoutStart)
	}
	if cell, _ := f.GetCellValue(checkInExportSheet, "E5"); cell != "" {
		t.Errorf("start date cell E5 = %q, want empty", cell)
	}
	if title, _ := f.GetCellValue(checkInExportSheet, "A1"); title != "DANH SÁCH NHÂN VIÊN TỰ CHẤM CÔNG" {
		t.Errorf("title = %q", title)
	}
	if subtitle, _ := f.GetCellValue(checkInExportSheet, "A2"); subtitle != "Dự án: Nhà máy Hải Phòng (HP1)" {
		t.Errorf("subtitle = %q", subtitle)
	}
}

func TestExportCheckInEmployeesSortsByNameAndDropsPagination(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	repo := &checkInExportRepo{result: &domain.CheckInConfigurationResult{
		Employees: []domain.CheckInConfigurationEmployee{
			{EmployeeName: "Vũ Zezuto", EmployeeCCCD: "3", EmployeeMobile: "03", CheckInStartDate: &start},
			{EmployeeName: "An Bình", EmployeeCCCD: "1", EmployeeMobile: "01", CheckInStartDate: &start},
			{EmployeeName: "Mười Mẻ", EmployeeCCCD: "2", EmployeeMobile: "02", CheckInStartDate: &start},
		},
		Total: 3,
	}}
	svc := newCheckInExportService(repo, &checkInExportProjectRepo{project: &domain.Project{ID: 5, Name: "Dự án 5", Code: "P5"}})

	file, err := svc.ExportCheckInEmployees(
		context.Background(),
		5,
		domain.CheckInConfigurationStatusEnabled,
		"an",
		time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	defer func() { _ = file.Close() }()

	if repo.query.Limit != 0 || repo.query.Offset != 0 {
		t.Errorf("export query must not paginate, got limit=%d offset=%d", repo.query.Limit, repo.query.Offset)
	}
	if repo.query.Status != domain.CheckInConfigurationStatusEnabled || repo.query.Search != "an" {
		t.Errorf("export query filters = status %q search %q, want the on-screen filters", repo.query.Status, repo.query.Search)
	}
	if want := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC); !repo.query.MonthStart.Equal(want) {
		t.Errorf("month window starts %v, want %v", repo.query.MonthStart, want)
	}

	rows, err := file.GetRows(checkInExportSheet)
	if err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if got := []string{rows[3][1], rows[4][1], rows[5][1]}; got[0] != "An Bình" || got[1] != "Mười Mẻ" || got[2] != "Vũ Zezuto" {
		t.Errorf("roster order = %v, want sorted by name", got)
	}
}

func TestExportCheckInEmployeesRejectsEmptySelection(t *testing.T) {
	repo := &checkInExportRepo{result: &domain.CheckInConfigurationResult{Total: 0}}
	svc := newCheckInExportService(repo, &checkInExportProjectRepo{project: &domain.Project{ID: 5, Name: "Dự án 5"}})

	_, err := svc.ExportCheckInEmployees(
		context.Background(),
		5,
		domain.CheckInConfigurationStatusEnabled,
		"",
		time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("exporting an empty group must fail instead of writing an empty workbook")
	}
	if !domain.IsValidationError(err) {
		t.Errorf("error = %v, want a validation error the handler renders as 400", err)
	}
}
