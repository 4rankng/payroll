package services

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The service applies the revenue_paid split in Go, so these stubs only have to
// hand back a fixed row set — they deliberately ignore the filters they receive.
type stubTimesheetRepo struct {
	domain.TimesheetRepository
	rows []*domain.Timesheet
}

func (s *stubTimesheetRepo) List(context.Context, domain.TimesheetFilters) ([]*domain.Timesheet, error) {
	return s.rows, nil
}

type stubProjectRepo struct {
	domain.ProjectRepository
	projects []*domain.Project
}

func (s *stubProjectRepo) List(context.Context, domain.ProjectFilters) ([]*domain.Project, error) {
	return s.projects, nil
}

func (s *stubProjectRepo) GetByIDs(context.Context, []uint) (map[uint]*domain.Project, error) {
	byID := make(map[uint]*domain.Project, len(s.projects))
	for _, p := range s.projects {
		byID[p.ID] = p
	}
	return byID, nil
}

type stubEmployeeRepo struct {
	domain.EmployeeRepository
}

func tevProject() *domain.Project {
	return &domain.Project{
		ID:               67,
		Name:             "TEV",
		Code:             "CX002",
		ProjectStatus:    domain.ProjectStatusRunning,
		SalaryPeriodFrom: 21,
		SalaryPeriodTo:   20,
	}
}

func paidRow(id uint, paidAmount int64, settled bool) *domain.Timesheet {
	return &domain.Timesheet{
		ID:          id,
		PaidAmount:  paidAmount,
		RevenuePaid: settled,
		Employee:    &domain.Employee{},
	}
}

// Regression: exporting a fully settled period used to filter out every row
// (revenue_paid=true) and hand back an empty workbook. The date-range export is
// a period statement, so settled rows must stay in it.
func TestGetPayrollReportForProjectsKeepsSettledRows(t *testing.T) {
	timesheets := &stubTimesheetRepo{rows: []*domain.Timesheet{
		paidRow(1, 100, false),
		paidRow(2, 200, true),
	}}
	projects := &stubProjectRepo{projects: []*domain.Project{tevProject()}}
	svc := NewPayrollReportByProjectService(timesheets, projects, &stubEmployeeRepo{})

	from := time.Date(2026, 7, 21, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 8, 20, 0, 0, 0, 0, time.Local)

	data, err := svc.GetPayrollReportForProjects(context.Background(), []uint{67}, from, to)
	require.NoError(t, err)
	require.Len(t, data, 1, "a fully settled period must still export a statement")

	assert.Equal(t, int64(300), data[0].TotalAmount)
	assert.Equal(t, []uint{1, 2}, data[0].TimesheetIDs)
}

// The same rows through the atDate path feed the reconciliation email, where
// showing an already-collected amount would re-bill the client.
func TestGetProjectsForPayrollReportDropsSettledRows(t *testing.T) {
	timesheets := &stubTimesheetRepo{rows: []*domain.Timesheet{
		paidRow(1, 100, false),
		paidRow(2, 200, true),
	}}
	projects := &stubProjectRepo{projects: []*domain.Project{tevProject()}}
	svc := NewPayrollReportByProjectService(timesheets, projects, &stubEmployeeRepo{})

	// Day 26 clears isProjectEligibleOnDay for salary_period_to=20.
	atDate := time.Date(2026, 8, 26, 0, 0, 0, 0, time.Local)

	data, err := svc.GetProjectsForPayrollReport(context.Background(), atDate)
	require.NoError(t, err)
	require.Len(t, data, 1)

	assert.Equal(t, int64(100), data[0].TotalAmount, "billing view must not re-bill settled rows")
	assert.Equal(t, []uint{1}, data[0].TimesheetIDs)
}
