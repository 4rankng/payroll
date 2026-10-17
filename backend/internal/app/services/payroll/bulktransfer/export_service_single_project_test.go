package bulktransfer

import (
	"context"
	"testing"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The weekly Chuyển lô file is keyed by (employee, project), so an employee
// with approved timesheets in two projects in one cycle produced two rows with
// the same bank account and two transaction codes — the October 2026 accident
// (six employees filed under both PQC Hải Phòng and PQC Nam Định). Export must
// refuse the batch instead of handing the admin a file that pays twice.
func TestExportRefusesOneEmployeeInTwoProjectsInOneCycle(t *testing.T) {
	bundle := seedStub()
	bundle.employees[7].Fullname = "Nguyễn Thị Loan"
	bundle.projects[13] = &domain.Project{ID: 13, Name: "Site B"}
	bundle.assignments[excel.EmployeeProjectKey{EmployeeID: 7, ProjectID: 13}] = &domain.ProjectEmployee{
		ID: 3, EmployeeID: 7, ProjectID: 13, PaymentSchedule: string(domain.PaymentScheduleWeekly),
	}
	second := &domain.Timesheet{
		ID: 103, EmployeeID: 7, ProjectID: 13, Amount: 250_000,
		Status: domain.TimesheetStatusApproved, PaymentStatus: domain.PaymentStatusPending,
		UpdatedAt: bundle.timesheets[0].UpdatedAt,
	}
	bundle.timesheets = append(bundle.timesheets, second)
	bundle.timesheetByID[second.ID] = second

	excelService := excel.NewService(&stubSettings{pct: 1})
	planner := newPlannerWithStub(bundle)
	planner.excelService = excelService
	transactionCodes := &stubTransactionCodeRepo{}
	eventBus := &recordingEventBus{}
	service := &ExportService{
		planner:             planner,
		transactionCodeRepo: transactionCodes,
		excelService:        excelService,
		eventBus:            eventBus,
	}

	response, err := service.Export(context.Background(), &dto.ExportBulkTransferRequest{
		FromDate:  "2026-07-08",
		ToDate:    "2026-07-14",
		CreatedBy: 1,
	})

	require.Error(t, err)
	assert.True(t, domain.IsValidationError(err), "want a validation error, got %v", err)
	assert.Contains(t, err.Error(), "Nguyễn Thị Loan")
	assert.Contains(t, err.Error(), "Site A")
	assert.Contains(t, err.Error(), "Site B")
	assert.Nil(t, response)
	assert.Empty(t, transactionCodes.created, "a refused export must not mint transaction codes")
	assert.Empty(t, eventBus.events, "a refused export must not publish an export outcome")
}
