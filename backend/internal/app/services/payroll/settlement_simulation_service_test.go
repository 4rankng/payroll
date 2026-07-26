package payroll

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	pkgClock "api-server/internal/pkg/clock"
)

type simulationTimesheetRepository struct {
	domain.TimesheetRepository
	rows []*domain.Timesheet
	err  error
}

func (r *simulationTimesheetRepository) List(context.Context, domain.TimesheetFilters) ([]*domain.Timesheet, error) {
	return r.rows, r.err
}

type simulationReportSelector struct {
	reports []*domainServices.ProjectReportData
	err     error
}

func (s *simulationReportSelector) GetProjectsForPayrollReport(context.Context, time.Time) ([]*domainServices.ProjectReportData, error) {
	return s.reports, s.err
}

type simulationLedgerReader struct {
	rangeTotal       int64
	transactionTotal int64
	err              error
	rangeCalls       int
	transactionCalls int
	transactionIDs   []uint
}

// GetAccountTotalInRange keeps the fake compatible with the broken
// implementation so the regression test proves the old date-window behavior
// fails before the transaction-scoped fix is applied.
func (r *simulationLedgerReader) GetAccountTotalInRange(context.Context, string, time.Time, time.Time) (int64, error) {
	r.rangeCalls++
	return r.rangeTotal, r.err
}

func (r *simulationLedgerReader) GetAccountTotalForTransactions(_ context.Context, _ string, transactionIDs []uint) (int64, error) {
	r.transactionCalls++
	r.transactionIDs = append([]uint(nil), transactionIDs...)
	return r.transactionTotal, r.err
}

func TestSettlementSimulationReconcilesCoveredReceivablesByTransaction(t *testing.T) {
	txIncluded := uint(10)
	txRemainder := uint(20)
	project := &domain.Project{ID: 1, Name: "Dự án thử nghiệm"}
	includedOne := simulationTimesheet(1, 1000, 1020, &txIncluded, project)
	includedTwo := simulationTimesheet(2, 500, 510, &txIncluded, project)
	remainder := simulationTimesheet(3, 340, 347, &txRemainder, project)

	ledger := &simulationLedgerReader{
		rangeTotal:       -352_097_029,
		transactionTotal: 1530,
	}
	service := NewSettlementSimulationService(
		&simulationReportSelector{reports: []*domainServices.ProjectReportData{{
			Project:      project,
			TimesheetIDs: []uint{includedOne.ID, includedTwo.ID},
			Timesheets:   []*domain.Timesheet{includedOne, includedTwo},
		}}},
		&simulationTimesheetRepository{rows: []*domain.Timesheet{includedOne, includedTwo, remainder}},
		ledger,
		pkgClock.NewFake(time.Date(2026, time.July, 26, 0, 0, 0, 0, pkgClock.DefaultLocation)),
	)

	result, err := service.Simulate(context.Background(), &dto.SimulateSettlementRequest{
		StartDate:   "2026-07-26",
		ExportCount: 1,
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}

	if result.Summary.TotalIncludedAmount != 1500 {
		t.Fatalf("included paid amount = %d, want 1500", result.Summary.TotalIncludedAmount)
	}
	if result.Reconciliation.ExportedTotal != 1530 {
		t.Fatalf("reconciliation comparator = %d, want covered revenue receivable 1530", result.Reconciliation.ExportedTotal)
	}
	if result.Reconciliation.LedgerReceivable != 1530 || result.Reconciliation.Delta != 0 || !result.Reconciliation.Reconciled {
		t.Fatalf("reconciliation = %+v, want an exact transaction-scoped match", result.Reconciliation)
	}
	if ledger.rangeCalls != 0 {
		t.Fatalf("date-window ledger calls = %d, want 0", ledger.rangeCalls)
	}
	if ledger.transactionCalls != 1 {
		t.Fatalf("transaction-scoped ledger calls = %d, want 1", ledger.transactionCalls)
	}
	if !slices.Equal(ledger.transactionIDs, []uint{txIncluded}) {
		t.Fatalf("transaction IDs = %v, want only covered transaction %d", ledger.transactionIDs, txIncluded)
	}
	if result.Summary.RemainingAmount != remainder.PaidAmount {
		t.Fatalf("remaining amount = %d, want %d", result.Summary.RemainingAmount, remainder.PaidAmount)
	}
}

func TestSettlementSimulationMissingTransactionLinkRemainsMismatch(t *testing.T) {
	project := &domain.Project{ID: 1, Name: "Dự án thử nghiệm"}
	included := simulationTimesheet(1, 1000, 1020, nil, project)
	ledger := &simulationLedgerReader{}
	service := NewSettlementSimulationService(
		&simulationReportSelector{reports: []*domainServices.ProjectReportData{{
			Project:      project,
			TimesheetIDs: []uint{included.ID},
			Timesheets:   []*domain.Timesheet{included},
		}}},
		&simulationTimesheetRepository{rows: []*domain.Timesheet{included}},
		ledger,
		pkgClock.NewFake(time.Date(2026, time.July, 26, 0, 0, 0, 0, pkgClock.DefaultLocation)),
	)

	result, err := service.Simulate(context.Background(), &dto.SimulateSettlementRequest{
		StartDate:   "2026-07-26",
		ExportCount: 1,
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}

	if result.Reconciliation.ExportedTotal != 1020 {
		t.Fatalf("reconciliation comparator = %d, want 1020", result.Reconciliation.ExportedTotal)
	}
	if result.Reconciliation.LedgerReceivable != 0 || result.Reconciliation.Delta != -1020 || result.Reconciliation.Reconciled {
		t.Fatalf("reconciliation = %+v, want missing ledger mismatch", result.Reconciliation)
	}
	if len(ledger.transactionIDs) != 0 {
		t.Fatalf("transaction IDs = %v, want none for an unlinked timesheet", ledger.transactionIDs)
	}
	if !hasSimulationWarning(result.Warnings, "LEDGER_TRANSACTION_LINK_MISSING") {
		t.Fatalf("warnings = %+v, want LEDGER_TRANSACTION_LINK_MISSING", result.Warnings)
	}
}

func TestSettlementSimulationMissingZeroValueTransactionLinkCannotReconcile(t *testing.T) {
	project := &domain.Project{ID: 1, Name: "Dự án thử nghiệm"}
	included := simulationTimesheet(1, 0, 0, nil, project)
	ledger := &simulationLedgerReader{}
	service := NewSettlementSimulationService(
		&simulationReportSelector{reports: []*domainServices.ProjectReportData{{
			Project:      project,
			TimesheetIDs: []uint{included.ID},
			Timesheets:   []*domain.Timesheet{included},
		}}},
		&simulationTimesheetRepository{rows: []*domain.Timesheet{included}},
		ledger,
		pkgClock.NewFake(time.Date(2026, time.July, 26, 0, 0, 0, 0, pkgClock.DefaultLocation)),
	)

	result, err := service.Simulate(context.Background(), &dto.SimulateSettlementRequest{
		StartDate:   "2026-07-26",
		ExportCount: 1,
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}

	if result.Reconciliation.Reconciled {
		t.Fatalf("reconciliation = %+v, missing evidence must not reconcile even at zero value", result.Reconciliation)
	}
	if !hasSimulationWarning(result.Warnings, "LEDGER_TRANSACTION_LINK_MISSING") {
		t.Fatalf("warnings = %+v, want LEDGER_TRANSACTION_LINK_MISSING", result.Warnings)
	}
}

func TestSettlementSimulationAcceptsEstablishedRoundingTolerance(t *testing.T) {
	txID := uint(10)
	project := &domain.Project{ID: 1, Name: "Dự án thử nghiệm"}
	includedOne := simulationTimesheet(1, 1000, 1020, &txID, project)
	includedTwo := simulationTimesheet(2, 500, 510, &txID, project)
	ledger := &simulationLedgerReader{transactionTotal: 1545}
	service := NewSettlementSimulationService(
		&simulationReportSelector{reports: []*domainServices.ProjectReportData{{
			Project:      project,
			TimesheetIDs: []uint{includedOne.ID, includedTwo.ID},
			Timesheets:   []*domain.Timesheet{includedOne, includedTwo},
		}}},
		&simulationTimesheetRepository{rows: []*domain.Timesheet{includedOne, includedTwo}},
		ledger,
		pkgClock.NewFake(time.Date(2026, time.July, 26, 0, 0, 0, 0, pkgClock.DefaultLocation)),
	)

	result, err := service.Simulate(context.Background(), &dto.SimulateSettlementRequest{
		StartDate:   "2026-07-26",
		ExportCount: 1,
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}

	if result.Reconciliation.Delta != 15 || !result.Reconciliation.Reconciled {
		t.Fatalf("reconciliation = %+v, want 15 VND rounding difference accepted", result.Reconciliation)
	}
}

func TestSettlementSimulationExcludesPartiallyCoveredTransactionsFromLedgerComparison(t *testing.T) {
	partialTxID := uint(10)
	completeTxID := uint(11)
	project := &domain.Project{ID: 1, Name: "Dự án thử nghiệm"}
	partialCovered := simulationTimesheet(1, 1000, 1020, &partialTxID, project)
	partialRemainder := simulationTimesheet(2, 500, 510, &partialTxID, project)
	completeCovered := simulationTimesheet(3, 500, 510, &completeTxID, project)
	ledger := &simulationLedgerReader{transactionTotal: 510}
	service := NewSettlementSimulationService(
		&simulationReportSelector{reports: []*domainServices.ProjectReportData{{
			Project:      project,
			TimesheetIDs: []uint{partialCovered.ID, completeCovered.ID},
			Timesheets:   []*domain.Timesheet{partialCovered, completeCovered},
		}}},
		&simulationTimesheetRepository{rows: []*domain.Timesheet{partialCovered, partialRemainder, completeCovered}},
		ledger,
		pkgClock.NewFake(time.Date(2026, time.July, 26, 0, 0, 0, 0, pkgClock.DefaultLocation)),
	)

	result, err := service.Simulate(context.Background(), &dto.SimulateSettlementRequest{
		StartDate:   "2026-07-26",
		ExportCount: 1,
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}

	if result.Reconciliation.ExportedTotal != completeCovered.RevenueReceivable {
		t.Fatalf("reconciliation comparator = %d, want only complete transaction amount %d", result.Reconciliation.ExportedTotal, completeCovered.RevenueReceivable)
	}
	if !slices.Equal(ledger.transactionIDs, []uint{completeTxID}) {
		t.Fatalf("transaction IDs = %v, want only complete transaction %d", ledger.transactionIDs, completeTxID)
	}
	if result.Reconciliation.Reconciled {
		t.Fatalf("reconciliation = %+v, partial transaction scope must not claim a complete match", result.Reconciliation)
	}
	if !hasSimulationWarning(result.Warnings, "LEDGER_PARTIAL_TRANSACTION_SCOPE") {
		t.Fatalf("warnings = %+v, want LEDGER_PARTIAL_TRANSACTION_SCOPE", result.Warnings)
	}
	if message := simulationWarningMessage(result.Warnings, "LEDGER_PARTIAL_TRANSACTION_SCOPE"); !strings.Contains(message, "1 giao dịch") {
		t.Fatalf("partial transaction warning = %q, want affected transaction count", message)
	}
}

func TestSettlementSimulationLedgerReadErrorCannotReportReconciled(t *testing.T) {
	txID := uint(10)
	project := &domain.Project{ID: 1, Name: "Dự án thử nghiệm"}
	included := simulationTimesheet(1, 1000, 1020, &txID, project)
	ledger := &simulationLedgerReader{err: errors.New("ledger unavailable")}
	service := NewSettlementSimulationService(
		&simulationReportSelector{reports: []*domainServices.ProjectReportData{{
			Project:      project,
			TimesheetIDs: []uint{included.ID},
			Timesheets:   []*domain.Timesheet{included},
		}}},
		&simulationTimesheetRepository{rows: []*domain.Timesheet{included}},
		ledger,
		pkgClock.NewFake(time.Date(2026, time.July, 26, 0, 0, 0, 0, pkgClock.DefaultLocation)),
	)

	result, err := service.Simulate(context.Background(), &dto.SimulateSettlementRequest{
		StartDate:   "2026-07-26",
		ExportCount: 1,
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}

	if result.Reconciliation.Reconciled {
		t.Fatalf("reconciliation = %+v, ledger errors must not report success", result.Reconciliation)
	}
	if !hasSimulationWarning(result.Warnings, "LEDGER_RECONCILIATION_FAILED") {
		t.Fatalf("warnings = %+v, want LEDGER_RECONCILIATION_FAILED", result.Warnings)
	}
}

func simulationTimesheet(id uint, paidAmount, revenueReceivable int64, transactionID *uint, project *domain.Project) *domain.Timesheet {
	return &domain.Timesheet{
		ID:                id,
		ProjectID:         project.ID,
		EmployeeID:        1,
		Date:              time.Date(2026, time.July, 20, 0, 0, 0, 0, pkgClock.DefaultLocation),
		PaymentStatus:     domain.PaymentStatusPaid,
		PaidAmount:        paidAmount,
		RevenueReceivable: revenueReceivable,
		TransactionID:     transactionID,
		Project:           project,
		Employee:          &domain.Employee{ID: 1, Fullname: "Nhân viên thử nghiệm"},
	}
}

func hasSimulationWarning(warnings []dto.SimWarning, code string) bool {
	for _, warning := range warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}

func simulationWarningMessage(warnings []dto.SimWarning, code string) string {
	for _, warning := range warnings {
		if warning.Code == code {
			return warning.Message
		}
	}
	return ""
}
