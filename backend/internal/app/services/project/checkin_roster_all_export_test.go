package project

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
)

type allRostersProjectRepo struct {
	domain.ProjectRepository
	projects map[uint]*domain.Project
	ids      []uint
}

func (f *allRostersProjectRepo) GetByIDs(_ context.Context, ids []uint) (map[uint]*domain.Project, error) {
	f.ids = ids
	return f.projects, nil
}

// TestExportAllCheckInRostersQueriesEveryProject pins the contract the day-9
// job depends on: ProjectID 0 reaches the repository (all projects), the
// project name lookup covers exactly the distinct assignment projects, and the
// returned count matches the row count.
func TestExportAllCheckInRostersQueriesEveryProject(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	repo := &checkInExportRepo{
		result: &domain.CheckInConfigurationResult{
			Employees: []domain.CheckInConfigurationEmployee{
				{ProjectID: 2, EmployeeName: "Bui Van Bao", CheckInStartDate: &start},
				{ProjectID: 1, EmployeeName: "Nguyen Van An"},
			},
		},
	}
	projectRepo := &allRostersProjectRepo{
		projects: map[uint]*domain.Project{
			1: {ID: 1, Name: "VFIC"},
			2: {ID: 2, Name: "Admin"},
		},
	}
	svc := newCheckInExportService(repo, projectRepo)

	f, count, err := svc.ExportAllCheckInRosters(context.Background(), domain.CheckInConfigurationStatusEnabled, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ExportAllCheckInRosters: %v", err)
	}
	if f == nil {
		t.Fatal("expected a workbook")
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if repo.query.ProjectID != 0 {
		t.Fatalf("ProjectID = %d, want 0 (all projects)", repo.query.ProjectID)
	}
	if len(projectRepo.ids) != 2 {
		t.Fatalf("GetByIDs got %d ids, want 2", len(projectRepo.ids))
	}

	rows, err := f.GetRows(checkInExportSheet)
	if err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if len(rows) != 5 { // title + subtitle + header + 2 employees
		t.Fatalf("rows = %d, want 5", len(rows))
	}
	if rows[2][1] != "Dự án" {
		t.Fatalf("header B = %q, want Dự án", rows[2][1])
	}
	// Sorted by project name: Admin before VFIC.
	if rows[3][1] != "Admin" || rows[3][2] != "Bui Van Bao" {
		t.Fatalf("row 4 = %v, want Admin/Bui Van Bao", rows[3])
	}
	if rows[4][1] != "VFIC" || rows[4][2] != "Nguyen Van An" {
		t.Fatalf("row 5 = %v, want VFIC/Nguyen Van An", rows[4])
	}
	// Start date only on the row that has one. GetRows trims trailing empty
	// cells, so a row without the date simply ends after column C.
	if rows[3][5] != "01/08/2026" {
		t.Fatalf("start date cell = %q, want 01/08/2026", rows[3][5])
	}
	if len(rows[4]) > 5 && rows[4][5] != "" {
		t.Fatalf("missing start date should stay empty, got %q", rows[4][5])
	}
}

// TestExportAllCheckInRostersEmptyReturnsNothing checks the job's skip path:
// no employees means no file and no error, never a validation error that would
// log a failure every month.
func TestExportAllCheckInRostersEmptyReturnsNothing(t *testing.T) {
	repo := &checkInExportRepo{result: &domain.CheckInConfigurationResult{}}
	svc := newCheckInExportService(repo, &allRostersProjectRepo{})

	f, count, err := svc.ExportAllCheckInRosters(context.Background(), domain.CheckInConfigurationStatusEnabled, time.Now())
	if err != nil {
		t.Fatalf("empty roster must not error: %v", err)
	}
	if f != nil || count != 0 {
		t.Fatalf("f=%v count=%d, want nil/0", f, count)
	}
}
