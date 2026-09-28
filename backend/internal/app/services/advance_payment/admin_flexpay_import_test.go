package advance_payment

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"api-server/internal/app/services/employee"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// TestDedupeAdvancePayments_LastWriteWins reproduces the flexible payroll
// template shape the ops team uses: the same employee can appear in multiple
// sheets within one workbook (e.g. "UL" then "UL (2)"), and the trailing
// occurrence holds the freshest amount. The deduper must collapse to one
// record per (project, employee) and keep the LAST value seen — otherwise
// BatchUpsert ends up issuing conflicting INSERTs for the same unique key
// and the order of resolution becomes implementation-defined.
func TestDedupeAdvancePayments_LastWriteWins(t *testing.T) {
	first := &domain.AdvancePayment{ProjectID: 58, EmployeeID: 638, ForMonth: "2026-04", UploadDate: "2026-04-15", MaxAdvAmount: 2460000}
	second := &domain.AdvancePayment{ProjectID: 58, EmployeeID: 638, ForMonth: "2026-04", UploadDate: "2026-04-15", MaxAdvAmount: 2580000}
	other := &domain.AdvancePayment{ProjectID: 58, EmployeeID: 444, ForMonth: "2026-04", UploadDate: "2026-04-15", MaxAdvAmount: 9060000}

	got := dedupeAdvancePayments([]*domain.AdvancePayment{first, second, other})

	if len(got) != 2 {
		t.Fatalf("expected 2 deduped records, got %d", len(got))
	}

	for _, ap := range got {
		if ap.EmployeeID == 638 && ap.MaxAdvAmount != 2580000 {
			t.Errorf("employee 638: expected last-write-wins amount 2580000, got %d", ap.MaxAdvAmount)
		}
		if ap.EmployeeID == 444 && ap.MaxAdvAmount != 9060000 {
			t.Errorf("employee 444: amount should be unchanged at 9060000, got %d", ap.MaxAdvAmount)
		}
	}
}

func TestDedupeAdvancePayments_PreservesOrder(t *testing.T) {
	a := &domain.AdvancePayment{ProjectID: 1, EmployeeID: 10, MaxAdvAmount: 100}
	b := &domain.AdvancePayment{ProjectID: 1, EmployeeID: 20, MaxAdvAmount: 200}
	c := &domain.AdvancePayment{ProjectID: 1, EmployeeID: 30, MaxAdvAmount: 300}

	got := dedupeAdvancePayments([]*domain.AdvancePayment{a, b, c})

	if len(got) != 3 {
		t.Fatalf("expected 3 records, got %d", len(got))
	}
	if got[0].EmployeeID != 10 || got[1].EmployeeID != 20 || got[2].EmployeeID != 30 {
		t.Errorf("expected order [10,20,30], got [%d,%d,%d]", got[0].EmployeeID, got[1].EmployeeID, got[2].EmployeeID)
	}
}

func TestDedupeAdvancePayments_Empty(t *testing.T) {
	if got := dedupeAdvancePayments(nil); len(got) != 0 {
		t.Errorf("expected empty slice for nil input, got %d", len(got))
	}
	if got := dedupeAdvancePayments([]*domain.AdvancePayment{}); len(got) != 0 {
		t.Errorf("expected empty slice for empty input, got %d", len(got))
	}
}

// TestDedupeAdvancePayments_DifferentProjectsSameEmployee verifies that the
// dedup key is (project, employee) — the same employee assigned to two
// projects gets two separate records, not one.
func TestDedupeAdvancePayments_DifferentProjectsSameEmployee(t *testing.T) {
	p1 := &domain.AdvancePayment{ProjectID: 58, EmployeeID: 638, MaxAdvAmount: 100}
	p2 := &domain.AdvancePayment{ProjectID: 59, EmployeeID: 638, MaxAdvAmount: 200}

	got := dedupeAdvancePayments([]*domain.AdvancePayment{p1, p2})

	if len(got) != 2 {
		t.Fatalf("expected 2 records (different projects), got %d", len(got))
	}
}

func TestResolveUploadDate_PrefersAssetCreatedAtOverFrozenClock(t *testing.T) {
	fake := clock.NewAutoFake()
	clock.SetGlobal(fake)
	t.Cleanup(func() {
		clock.SetGlobal(clock.New())
	})

	fake.Set(time.Date(2026, 6, 6, 9, 0, 0, 0, clock.DefaultLocation))

	uploadedAt := time.Date(2026, 6, 20, 17, 37, 28, 0, time.FixedZone("+08", 8*60*60))

	got := resolveUploadDate(uploadedAt)
	if got != "2026-06-20" {
		t.Fatalf("expected upload date 2026-06-20 from asset created_at, got %s", got)
	}
}

func TestResolveUploadDate_FallsBackToBusinessClockWhenAssetTimestampMissing(t *testing.T) {
	fake := clock.NewAutoFake()
	clock.SetGlobal(fake)
	t.Cleanup(func() {
		clock.SetGlobal(clock.New())
	})

	fake.Set(time.Date(2026, 6, 6, 9, 0, 0, 0, clock.DefaultLocation))

	got := resolveUploadDate(time.Time{})
	if got != "2026-06-06" {
		t.Fatalf("expected fallback upload date 2026-06-06, got %s", got)
	}
}

func TestShouldNotifyFlexPayZNS_OnlyForRequestableFlexibleEmployees(t *testing.T) {
	eligible := &domain.ProjectEmployee{PaymentSchedule: string(domain.PaymentScheduleFlexible)}
	selfCheckIn := &domain.ProjectEmployee{PaymentSchedule: string(domain.PaymentScheduleFlexible), CheckInEnabled: true}
	weekly := &domain.ProjectEmployee{PaymentSchedule: string(domain.PaymentScheduleWeekly)}

	tests := []struct {
		name       string
		amount     uint64
		mobile     string
		assignment *domain.ProjectEmployee
		want       bool
	}{
		{name: "eligible flexible employee", amount: 1_000_000, mobile: "0366178061", assignment: eligible, want: true},
		{name: "self check-in employee", amount: 1_000_000, mobile: "0366178061", assignment: selfCheckIn},
		{name: "non-flexible employee", amount: 1_000_000, mobile: "0366178061", assignment: weekly},
		{name: "missing mobile", amount: 1_000_000, assignment: eligible},
		{name: "zero amount", mobile: "0366178061", assignment: eligible},
		{name: "missing assignment", amount: 1_000_000, mobile: "0366178061"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldNotifyFlexPayZNS(tt.amount, tt.mobile, tt.assignment); got != tt.want {
				t.Fatalf("shouldNotifyFlexPayZNS() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestImportFlexPayFile_SkipsCheckInEnabledEmployeeQuota pins the boundary
// between the two quota sources: the admin workbook may top up the quota of
// flexible-pay employees, but a self-check-in employee's quota comes from
// check-in/out earnings and the upload must leave it untouched (while still
// creating their employee/assignment rows and notification entry).
func TestImportFlexPayFile_SkipsCheckInEnabledEmployeeQuota(t *testing.T) {
	project := &domain.Project{ID: 10, Code: "PROJ1"}
	userID := uint(77)
	// Bank tuple and mobile match the workbook rows so the import performs no
	// employee update and no bank-verifier call.
	normalEmp := &domain.Employee{
		ID: 1, Fullname: "NGUYEN VAN A", CCCD: "012345678901",
		Mobile: "0912345678", BankAccountNumber: "0123456789",
		BankAccountName: "NGUYEN VAN A", UserID: &userID,
	}
	checkInEmp := &domain.Employee{
		ID: 2, Fullname: "TRAN VAN B", CCCD: "098765432109",
		Mobile: "0987654321", BankAccountNumber: "0987654321",
		BankAccountName: "TRAN VAN B", UserID: &userID,
	}

	employeeRepo := &flexpayImportEmployeeRepo{
		byCCCD: map[string]*domain.Employee{
			normalEmp.CCCD:  normalEmp,
			checkInEmp.CCCD: checkInEmp,
		},
	}
	projectRepo := &flexpayImportProjectRepo{byCode: map[string]*domain.Project{"PROJ1": project}}
	projectEmployeeRepo := &flexpayImportProjectEmployeeRepo{
		byProjectEmployee: map[string]*domain.ProjectEmployee{
			fmt.Sprintf("%d:%d", project.ID, normalEmp.ID): {
				ProjectID: project.ID, EmployeeID: normalEmp.ID,
				PaymentSchedule: string(domain.PaymentScheduleFlexible),
			},
			fmt.Sprintf("%d:%d", project.ID, checkInEmp.ID): {
				ProjectID: project.ID, EmployeeID: checkInEmp.ID,
				PaymentSchedule: string(domain.PaymentScheduleFlexible),
				CheckInEnabled:  true,
			},
		},
	}
	advanceRepo := &flexpayImportAdvanceRepo{}

	empService := employee.NewEmployeeService(&employee.Config{
		EmployeeRepo:  employeeRepo,
		TimesheetRepo: &flexpayImportTimesheetRepo{},
	})

	svc := NewService(&Config{
		AdvancePaymentRepo:  advanceRepo,
		EmployeeRepo:        employeeRepo,
		ProjectRepo:         projectRepo,
		ProjectEmployeeRepo: projectEmployeeRepo,
		EmployeeService:     empService,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	file := excelize.NewFile()
	file.SetSheetName("Sheet1", "UL")
	file.SetCellValue("UL", "B1", "Mã nhân viên")

	rows := [][]string{
		{"", "012345678901", "PROJ1", "phổ thông", "0912345678", "NGUYEN VAN A", "NGUYEN VAN A", "0123456789", "", "2000000"},
		{"", "098765432109", "PROJ1", "phổ thông", "0987654321", "TRAN VAN B", "TRAN VAN B", "0987654321", "", "3000000"},
	}
	for i, row := range rows {
		for col, val := range row {
			cell, _ := excelize.CoordinatesToCellName(col+1, i+2)
			file.SetCellValue("UL", cell, val)
		}
	}

	uploadedAt := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	result, err := svc.ImportFlexPayFile(context.Background(), file, "2026-09", 55, 1, uploadedAt, nil)
	if err != nil {
		t.Fatalf("ImportFlexPayFile returned error: %v", err)
	}

	if result.TotalRows != 2 {
		t.Fatalf("TotalRows = %d, want 2", result.TotalRows)
	}
	upserted := advanceRepo.upserted
	if len(upserted) != 1 {
		t.Fatalf("expected exactly 1 quota row in BatchUpsert, got %d (%+v)", len(upserted), upserted)
	}
	got := upserted[0]
	if got.EmployeeID != normalEmp.ID || got.ProjectID != project.ID {
		t.Fatalf("quota row belongs to employee %d/project %d, want the non-check-in employee 1/project 10", got.EmployeeID, got.ProjectID)
	}
	if got.MaxAdvAmount != 2_000_000 {
		t.Fatalf("MaxAdvAmount = %d, want 2000000", got.MaxAdvAmount)
	}
	if got.ForMonth != "2026-09" || got.UploadDate != "2026-09-28" {
		t.Fatalf("ForMonth/UploadDate = %s/%s, want 2026-09/2026-09-28", got.ForMonth, got.UploadDate)
	}
	for _, ap := range upserted {
		if ap.EmployeeID == checkInEmp.ID {
			t.Fatalf("self-check-in employee 2 quota must not be written by the import")
		}
	}
	if result.AdvancePaymentsCreated != 1 {
		t.Fatalf("AdvancePaymentsCreated = %d, want 1", result.AdvancePaymentsCreated)
	}
}

type flexpayImportEmployeeRepo struct {
	domain.EmployeeRepository
	byCCCD map[string]*domain.Employee
}

func (f *flexpayImportEmployeeRepo) GetByCCCD(_ context.Context, cccd string) (*domain.Employee, error) {
	if emp, ok := f.byCCCD[cccd]; ok {
		return emp, nil
	}
	return nil, domain.NewNotFoundError("nhân viên")
}

type flexpayImportProjectRepo struct {
	domain.ProjectRepository
	byCode map[string]*domain.Project
}

func (f *flexpayImportProjectRepo) GetByCode(_ context.Context, code string) (*domain.Project, error) {
	if p, ok := f.byCode[code]; ok {
		return p, nil
	}
	return nil, domain.NewNotFoundError("dự án")
}

type flexpayImportProjectEmployeeRepo struct {
	domain.ProjectEmployeeRepository
	byProjectEmployee map[string]*domain.ProjectEmployee
}

func (f *flexpayImportProjectEmployeeRepo) GetActiveAssignmentByProjectAndEmployee(_ context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	if pe, ok := f.byProjectEmployee[fmt.Sprintf("%d:%d", projectID, employeeID)]; ok {
		return pe, nil
	}
	return nil, domain.NewNotFoundError("phân công")
}

type flexpayImportAdvanceRepo struct {
	domain.AdvancePaymentRepository
	upserted []*domain.AdvancePayment
}

func (f *flexpayImportAdvanceRepo) BatchUpsert(_ context.Context, aps []*domain.AdvancePayment) error {
	f.upserted = append(f.upserted, aps...)
	return nil
}

type flexpayImportTimesheetRepo struct {
	domain.TimesheetRepository
}

func (f *flexpayImportTimesheetRepo) GetLatestTimesheetDateByEmployeeID(_ context.Context, _ uint) (*time.Time, error) {
	return nil, nil
}
