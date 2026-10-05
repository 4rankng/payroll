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
)

// A salary period may be funded by the admin workbook OR by self check-in/out.
// These tests pin the import side of that rule: the workbook must leave a
// check-in period alone, including the case the flag alone used to miss — an
// employee whose check-in earnings for the period are already banked even
// though the assignment is not flagged as check-in any more.

func newPeriodImportService(
	project *domain.Project,
	assignments map[string]*domain.ProjectEmployee,
	employees map[string]*domain.Employee,
	advanceRepo *flexpayImportAdvanceRepo,
) *Service {
	employeeRepo := &flexpayImportEmployeeRepo{byCCCD: employees}
	projectRepo := &flexpayImportProjectRepo{byCode: map[string]*domain.Project{project.Code: project}}
	empService := employee.NewEmployeeService(&employee.Config{
		EmployeeRepo:  employeeRepo,
		TimesheetRepo: &flexpayImportTimesheetRepo{},
	})

	return NewService(&Config{
		AdvancePaymentRepo:  advanceRepo,
		EmployeeRepo:        employeeRepo,
		ProjectRepo:         projectRepo,
		ProjectEmployeeRepo: &flexpayImportProjectEmployeeRepo{byProjectEmployee: assignments},
		EmployeeService:     empService,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// periodWorkbook builds a single-row UL workbook for one employee. The bank and
// mobile values mirror the stored employee so the import performs no update.
func periodWorkbook(t *testing.T, emp *domain.Employee, projectCode, amount string) *excelize.File {
	t.Helper()
	file := excelize.NewFile()
	_ = file.SetSheetName("Sheet1", "UL")
	_ = file.SetCellValue("UL", "B1", "Mã nhân viên")
	row := []string{
		"", emp.CCCD, projectCode, "phổ thông", emp.Mobile,
		emp.Fullname, emp.BankAccountName, emp.BankAccountNumber, "", amount,
	}
	for col, val := range row {
		cell, _ := excelize.CoordinatesToCellName(col+1, 2)
		_ = file.SetCellValue("UL", cell, val)
	}
	return file
}

func periodEmployee(id uint, cccd, fullname, mobile string) *domain.Employee {
	userID := uint(77)
	return &domain.Employee{
		ID: id, Fullname: fullname, CCCD: cccd, Mobile: mobile,
		BankAccountNumber: mobile, BankAccountName: fullname, UserID: &userID,
	}
}

func TestImportFlexPayFile_SkipsPeriodWithBankedCheckInEarnings(t *testing.T) {
	project := &domain.Project{ID: 10, Code: "PROJ1"}
	emp := periodEmployee(2, "098765432109", "TRAN VAN B", "0987654321")
	// The assignment is NOT check-in enabled any more (switched off after the
	// month), but the month's earnings are already banked. The period is still
	// a check-in period, so the workbook must not add a second source.
	assignment := &domain.ProjectEmployee{
		ProjectID: project.ID, EmployeeID: emp.ID,
		PaymentSchedule: string(domain.PaymentScheduleFlexible),
	}
	advanceRepo := &flexpayImportAdvanceRepo{byEmployeeMonth: map[string][]*domain.AdvancePayment{
		fmt.Sprintf("%d:2026-09", emp.ID): {{
			ProjectID: project.ID, EmployeeID: emp.ID, ForMonth: "2026-09",
			Salary: 4_000_000, MaxAdvAmount: 2_800_000,
		}},
	}}
	svc := newPeriodImportService(
		project,
		map[string]*domain.ProjectEmployee{fmt.Sprintf("%d:%d", project.ID, emp.ID): assignment},
		map[string]*domain.Employee{emp.CCCD: emp},
		advanceRepo,
	)

	result, err := svc.ImportFlexPayFile(
		context.Background(),
		periodWorkbook(t, emp, project.Code, "3000000"),
		"2026-09", 55, 1, time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), nil,
	)
	if err != nil {
		t.Fatalf("ImportFlexPayFile: %v", err)
	}

	if len(advanceRepo.upserted) != 0 {
		t.Fatalf("a period with banked check-in earnings must not receive a workbook quota, got %+v", advanceRepo.upserted)
	}
	if result.AdvancePaymentsCreated != 0 {
		t.Fatalf("AdvancePaymentsCreated = %d, want 0", result.AdvancePaymentsCreated)
	}
}

func TestImportFlexPayFile_SkipsOnlyPeriodsSpentOnCheckIn(t *testing.T) {
	project := &domain.Project{ID: 10, Code: "PROJ1"}
	emp := periodEmployee(3, "011111111111", "LE THI C", "0900000003")
	// Check-in started on 2026-09-15: September is a check-in period, August
	// was not.
	assignment := &domain.ProjectEmployee{
		ProjectID: project.ID, EmployeeID: emp.ID,
		PaymentSchedule:  string(domain.PaymentScheduleFlexible),
		CheckInEnabled:   true,
		CheckInStartDate: datePtr(2026, 9, 15),
	}
	advanceRepo := &flexpayImportAdvanceRepo{byEmployeeMonth: map[string][]*domain.AdvancePayment{}}
	svc := newPeriodImportService(
		project,
		map[string]*domain.ProjectEmployee{fmt.Sprintf("%d:%d", project.ID, emp.ID): assignment},
		map[string]*domain.Employee{emp.CCCD: emp},
		advanceRepo,
	)

	if _, err := svc.ImportFlexPayFile(
		context.Background(),
		periodWorkbook(t, emp, project.Code, "1000000"),
		"2026-08", 55, 1, time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC), nil,
	); err != nil {
		t.Fatalf("ImportFlexPayFile(august): %v", err)
	}
	if len(advanceRepo.upserted) != 1 {
		t.Fatalf("a period before the check-in start must still receive the workbook quota, got %d rows", len(advanceRepo.upserted))
	}

	advanceRepo.upserted = nil
	if _, err := svc.ImportFlexPayFile(
		context.Background(),
		periodWorkbook(t, emp, project.Code, "1000000"),
		"2026-09", 55, 1, time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC), nil,
	); err != nil {
		t.Fatalf("ImportFlexPayFile(september): %v", err)
	}
	if len(advanceRepo.upserted) != 0 {
		t.Fatalf("a period the employee spent on check-in must be skipped, got %d rows", len(advanceRepo.upserted))
	}
}

func datePtr(year int, month time.Month, day int) *time.Time {
	d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &d
}
