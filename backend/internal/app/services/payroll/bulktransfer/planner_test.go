package bulktransfer

import (
	"context"
	"errors"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/domain"
	pkgClock "api-server/internal/pkg/clock"
	"api-server/internal/pkg/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubRepoBundle is a minimal in-memory fake for the four read repos the
// planner depends on. Enough surface to exercise Plan() end-to-end without
// a live DB; intentionally narrow so tests stay readable.
type stubRepoBundle struct {
	timesheets       []*domain.Timesheet
	employees        map[uint]*domain.Employee
	projects         map[uint]*domain.Project
	assignments      map[excel.EmployeeProjectKey]*domain.ProjectEmployee
	timesheetByID    map[uint]*domain.Timesheet
	weeklyPercentage float64
	listFilters      []domain.TimesheetFilters
}

type stubTimesheetRepo struct{ b *stubRepoBundle }

func (r *stubTimesheetRepo) List(ctx context.Context, filters domain.TimesheetFilters) ([]*domain.Timesheet, error) {
	r.b.listFilters = append(r.b.listFilters, filters)
	out := make([]*domain.Timesheet, 0, len(r.b.timesheets))
	for _, ts := range r.b.timesheets {
		// honor status + payment status + project filters; dates ignored for simplicity
		if !containsStatus(filters.TimesheetStatus, ts.Status) {
			continue
		}
		if !containsPaymentStatus(filters.PaymentStatus, ts.PaymentStatus) {
			continue
		}
		if len(filters.ProjectIDs) > 0 && !containsUint(filters.ProjectIDs, ts.ProjectID) {
			continue
		}
		// ForcePayroll: stub treats it the same as the base path — both lists
		// come from the same timesheets slice, the planner de-dups the union.
		out = append(out, ts)
	}
	return out, nil
}
func (r *stubTimesheetRepo) GetByIDs(ctx context.Context, ids []uint) ([]*domain.Timesheet, error) {
	out := make([]*domain.Timesheet, 0, len(ids))
	for _, id := range ids {
		if ts, ok := r.b.timesheetByID[id]; ok {
			out = append(out, ts)
		}
	}
	return out, nil
}
func (r *stubTimesheetRepo) GetByProjectAndEmployee(context.Context, uint, uint, time.Time, time.Time) ([]*domain.Timesheet, error) {
	return nil, errors.New("not used")
}
func (r *stubTimesheetRepo) GetByEmployeeAndPeriod(context.Context, uint, time.Time, time.Time) ([]*domain.Timesheet, error) {
	return nil, errors.New("not used")
}
func (r *stubTimesheetRepo) CountDistinctWorkingDays(context.Context, uint, uint, time.Time, time.Time) (int, error) {
	return 0, errors.New("not used")
}
func (r *stubTimesheetRepo) BulkUpdatePaymentStatus(context.Context, []domain.PaymentStatusUpdate) error {
	return errors.New("must not be called from planner")
}
func (r *stubTimesheetRepo) BulkUpdateRevenueReceivable(context.Context, map[uint]int64) error {
	return errors.New("must not be called from planner")
}
func (r *stubTimesheetRepo) BulkUpdateTransactionID(context.Context, uint, []uint) error {
	return errors.New("must not be called from planner")
}
func (r *stubTimesheetRepo) Update(context.Context, *domain.Timesheet) error {
	return errors.New("must not be called from planner")
}

type stubEmployeeRepo struct{ b *stubRepoBundle }

func (r *stubEmployeeRepo) GetByID(ctx context.Context, id uint) (*domain.Employee, error) {
	if e, ok := r.b.employees[id]; ok {
		return e, nil
	}
	return nil, errors.New("employee not found")
}
func (r *stubEmployeeRepo) GetByIDs(context.Context, []int64) ([]*domain.Employee, error) {
	return nil, errors.New("not used")
}
func (r *stubEmployeeRepo) GetByBankAccountNumber(context.Context, string) (*domain.Employee, error) {
	return nil, errors.New("not used")
}

type stubProjectRepo struct{ b *stubRepoBundle }

func (r *stubProjectRepo) GetByID(ctx context.Context, id uint) (*domain.Project, error) {
	if p, ok := r.b.projects[id]; ok {
		return p, nil
	}
	return nil, errors.New("project not found")
}
func (r *stubProjectRepo) List(context.Context, domain.ProjectFilters) ([]*domain.Project, error) {
	out := make([]*domain.Project, 0, len(r.b.projects))
	for _, p := range r.b.projects {
		out = append(out, p)
	}
	return out, nil
}

type stubProjectEmployeeRepo struct{ b *stubRepoBundle }

func (r *stubProjectEmployeeRepo) GetActiveAssignmentByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	key := excel.EmployeeProjectKey{EmployeeID: employeeID, ProjectID: projectID}
	if a, ok := r.b.assignments[key]; ok {
		return a, nil
	}
	return nil, errors.New("assignment not found")
}
func (r *stubProjectEmployeeRepo) GetActiveAssignmentsByProjectsAndEmployees(context.Context, []uint, []uint) ([]*domain.ProjectEmployee, error) {
	return nil, errors.New("not used")
}

type stubSettings struct {
	pct             float64
	limit           int64
	failOnLimitRead bool
}

func (s *stubSettings) GetWeeklyPaymentPercentage(context.Context) float64  { return s.pct }
func (s *stubSettings) GetMonthlyPaymentPercentage(context.Context) float64 { return s.pct }
func (s *stubSettings) GetAdvanceCashFeePercentage(context.Context) float64 { return 0 }
func (s *stubSettings) GetPartnerCompany(context.Context) string            { return "TestCo" }
func (s *stubSettings) GetPaymentPercentageForSchedule(_ context.Context, _ string) float64 {
	return s.pct
}
func (s *stubSettings) GetBulkTransferWorkbookLimit(context.Context) int64 {
	if s.failOnLimitRead {
		panic("OnePay must not read the manual MBank workbook limit")
	}
	if s.limit >= 2 {
		return s.limit
	}
	return 400_000_000
}

// helpers
func containsStatus(s []domain.TimesheetStatus, v domain.TimesheetStatus) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
func containsPaymentStatus(s []domain.PaymentStatus, v domain.PaymentStatus) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
func containsUint(s []uint, v uint) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func newPlannerWithStub(b *stubRepoBundle) *ExportPlanner {
	tsRepo := &stubTimesheetRepo{b: b}
	return &ExportPlanner{
		timesheetRepo:       tsRepo,
		employeeRepo:        &stubEmployeeRepo{b: b},
		projectRepo:         &stubProjectRepo{b: b},
		projectEmployeeRepo: &stubProjectEmployeeRepo{b: b},
		periodCalculator:    nil, // weekly mode does not use it
		excelService:        excel.NewService(&stubSettings{pct: b.weeklyPercentage}),
	}
}

func seedStub() *stubRepoBundle {
	tsUpdated := time.Date(2026, 7, 12, 9, 0, 0, 0, pkgClock.DefaultLocation)
	b := &stubRepoBundle{
		employees: map[uint]*domain.Employee{
			7: {ID: 7, BankAccountNumber: "12345678", BankAccountName: "NGUYEN VAN A", Bank: &domain.Bank{ID: 1, BranchName: "MBank", BankCode: "MB"}},
			9: {ID: 9, BankAccountNumber: "", BankAccountName: "TRAN B", Bank: &domain.Bank{ID: 1, BranchName: "MBank", BankCode: "MB"}}, // missing account number → excluded
		},
		projects: map[uint]*domain.Project{
			12: {ID: 12, Name: "Site A"},
		},
		assignments: map[excel.EmployeeProjectKey]*domain.ProjectEmployee{
			{EmployeeID: 7, ProjectID: 12}: {ID: 1, EmployeeID: 7, ProjectID: 12, PaymentSchedule: string(domain.PaymentScheduleWeekly)},
			{EmployeeID: 9, ProjectID: 12}: {ID: 2, EmployeeID: 9, ProjectID: 12, PaymentSchedule: string(domain.PaymentScheduleWeekly)},
		},
		weeklyPercentage: 1.0, // 100% so ValidateAndFilter doesn't trim
	}
	b.timesheets = []*domain.Timesheet{
		{ID: 101, EmployeeID: 7, ProjectID: 12, Amount: 5000000, Status: domain.TimesheetStatusApproved, PaymentStatus: domain.PaymentStatusPending, UpdatedAt: tsUpdated},
		{ID: 102, EmployeeID: 7, ProjectID: 12, Amount: 3000000, Status: domain.TimesheetStatusApproved, PaymentStatus: domain.PaymentStatusPending, UpdatedAt: tsUpdated},
		{ID: 110, EmployeeID: 9, ProjectID: 12, Amount: 2000000, Status: domain.TimesheetStatusApproved, PaymentStatus: domain.PaymentStatusPending, UpdatedAt: tsUpdated},
	}
	b.timesheetByID = map[uint]*domain.Timesheet{}
	for _, ts := range b.timesheets {
		b.timesheetByID[ts.ID] = ts
	}
	return b
}

// TestPlanner_Plan_Weekly_AggregatesAndValidates is the Phase 1 regression
// guard: Plan() must select, aggregate, and validate identically to the
// pre-refactor ExportService.Export() body. Verifies:
//   - eligible timesheets aggregated by employee×project
//   - employee 9 excluded (missing bank account number) — ValidateAndFilter
//   - SnapshotEpoch = max(timesheet.updated_at)
//   - cycle label = "weekly"
func TestPlanner_Plan_Weekly_AggregatesAndValidates(t *testing.T) {
	b := seedStub()
	p := newPlannerWithStub(b)

	plan, err := p.Plan(context.Background(), &dto.ExportBulkTransferRequest{
		FromDate:  "2026-07-08",
		ToDate:    "2026-07-14",
		CreatedBy: 1,
	})
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, "weekly", plan.Cycle)
	assert.False(t, plan.IsMonthly)

	// Two distinct employee×project keys exist, but only employee 7 passes
	// ValidateAndFilter (employee 9 has no bank account number).
	require.NotNil(t, plan.ValidatedData)
	assert.Equal(t, 2, plan.ValidatedData.TotalCount, "total aggregated keys")
	assert.Equal(t, 1, plan.ValidatedData.ValidCount, "valid keys after bank-info filter")
	assert.Equal(t, 1, plan.ValidatedData.SkippedCount, "skipped keys")

	// Employee 7 aggregated amount = 5,000,000 + 3,000,000
	key7 := excel.EmployeeProjectKey{EmployeeID: 7, ProjectID: 12}
	assert.Equal(t, int64(8000000), plan.RawAggregated.EmployeeProjectAmounts[key7])
	assert.Equal(t, []uint{101, 102}, plan.RawAggregated.EmployeeProjectTimesheets[key7])

	// SnapshotEpoch = the seeded updated_at.
	assert.Equal(t, time.Date(2026, 7, 12, 9, 0, 0, 0, pkgClock.DefaultLocation), plan.SnapshotEpoch)

	// Date-only filters must use the same location as the loc=Local MySQL DSN.
	// UTC midnight would become local 08:00 and omit the entire first day.
	require.NotEmpty(t, b.listFilters)
	require.NotNil(t, b.listFilters[0].FromDate)
	require.NotNil(t, b.listFilters[0].ToDate)
	assert.Equal(t, pkgClock.DefaultLocation, b.listFilters[0].FromDate.Location())
	assert.Equal(t, pkgClock.DefaultLocation, b.listFilters[0].ToDate.Location())
	assert.Equal(t, "2026-07-08", b.listFilters[0].FromDate.Format(timeutil.DateFormat))
	assert.Equal(t, "2026-07-14", b.listFilters[0].ToDate.Format(timeutil.DateFormat))
}

// TestPlanner_Plan_EmptyPool_NoError verifies the empty-eligible-pool case:
// Plan returns a non-nil plan with zero items, no error.
func TestPlanner_Plan_EmptyPool_NoError(t *testing.T) {
	b := &stubRepoBundle{
		employees:        map[uint]*domain.Employee{},
		projects:         map[uint]*domain.Project{},
		assignments:      map[excel.EmployeeProjectKey]*domain.ProjectEmployee{},
		timesheetByID:    map[uint]*domain.Timesheet{},
		weeklyPercentage: 1.0,
	}
	p := newPlannerWithStub(b)
	plan, err := p.Plan(context.Background(), &dto.ExportBulkTransferRequest{
		FromDate: "2026-07-08", ToDate: "2026-07-14", CreatedBy: 1,
	})
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.NotNil(t, plan.ValidatedData)
	assert.Equal(t, 0, plan.ValidatedData.TotalCount)
	assert.Equal(t, 0, plan.ValidatedData.ValidCount)
	assert.True(t, plan.SnapshotEpoch.IsZero(), "empty pool → zero snapshot")
}

func TestBuildForecastOutcomeItems_DeduplicatesOnlyValidatedExactRangeRows(t *testing.T) {
	key := excel.EmployeeProjectKey{EmployeeID: 7, ProjectID: 12}
	validation := &excel.BulkTransferValidationResult{ValidData: &excel.BulkTransferData{
		EmployeeProjectAmounts:    map[excel.EmployeeProjectKey]int64{key: 1_200},
		EmployeeProjectTimesheets: map[excel.EmployeeProjectKey][]uint{key: {101, 102, 103}},
	}}
	selected := []*domain.Timesheet{
		{ID: 101, EmployeeID: 7, ProjectID: 12, Date: time.Date(2026, time.July, 8, 0, 0, 0, 0, time.UTC), Amount: 100},
		{ID: 102, EmployeeID: 7, ProjectID: 12, Date: time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC), Amount: 200},
		// Force-included backlog is in the bank file but outside the forecast range.
		{ID: 103, EmployeeID: 7, ProjectID: 12, Date: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), Amount: 900},
		// This row failed validation and must not become a measured outcome.
		{ID: 104, EmployeeID: 8, ProjectID: 12, Date: time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC), Amount: 500},
	}
	items := buildForecastOutcomeItems(&dto.ExportBulkTransferRequest{
		FromDate: "2026-07-08", ToDate: "2026-07-14",
	}, false, selected, validation, 0.70)

	require.Equal(t, []domain.CashForecastOutcomeItem{
		{TimesheetID: 101, Amount: 70},
		{TimesheetID: 102, Amount: 140},
	}, items)
}
