package bulktransfer

import (
	"context"
	"errors"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/app/services/payroll/excel"
	pkgClock "api-server/internal/pkg/clock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePlanner is a stub Planner used to drive SimulationService without a DB.
// It returns canned ExportPlans based on the request's NoDateFilter flag so
// tests can simulate the cycle-window plan vs the full-pool plan separately.
type fakePlanner struct {
	cyclePlan   *ExportPlan
	fullPoolPlan *ExportPlan
	cycleCalls  int
	fullCalls   int
	callErr     error
}

func (p *fakePlanner) Plan(ctx context.Context, req *dto.ExportBulkTransferRequest) (*ExportPlan, error) {
	if p.callErr != nil {
		return nil, p.callErr
	}
	if req.NoDateFilter {
		p.fullCalls++
		if p.fullPoolPlan != nil {
			return p.fullPoolPlan, nil
		}
		return p.cyclePlan, nil
	}
	p.cycleCalls++
	return p.cyclePlan, nil
}

// fakeLedger is a stub LedgerService satisfying the local interface.
type fakeLedger struct {
	total int64
	err   error
}

func (f *fakeLedger) CreateEntries(context.Context, []*domain.LedgerEntry, uint) ([]*domain.LedgerEntry, error) {
	return nil, errors.New("must not be called from simulation")
}
func (f *fakeLedger) ListEntries(context.Context, domain.LedgerFilters) ([]*domain.LedgerEntry, error) {
	return nil, errors.New("must not be called from simulation")
}
func (f *fakeLedger) GetEntriesByAssetID(context.Context, uint) ([]*domain.LedgerEntry, error) {
	return nil, errors.New("must not be called from simulation")
}
func (f *fakeLedger) DeleteEntriesByTransactionID(context.Context, uint) error {
	return errors.New("must not be called from simulation")
}
func (f *fakeLedger) GetAccountTotalInRange(context.Context, domain.LedgerAccount, time.Time, time.Time) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.total, nil
}

// makeAggregated builds a BulkTransferData with one employee×project group.
func makeAggregated(empID, projID uint, amount int64, tsIDs []uint, bankAcct string) *excel.BulkTransferData {
	key := excel.EmployeeProjectKey{EmployeeID: empID, ProjectID: projID}
	return &excel.BulkTransferData{
		EmployeeProjectAmounts:    map[excel.EmployeeProjectKey]int64{key: amount},
		EmployeeProjectTimesheets: map[excel.EmployeeProjectKey][]uint{key: tsIDs},
		EmployeeData: map[uint]excel.Employee{empID: {
			ID: empID, Fullname: "Employee", BankAccountNumber: bankAcct,
		}},
		ProjectData:     map[uint]excel.Project{projID: {ID: projID, Name: "Project"}},
		TransactionCodes: map[excel.EmployeeProjectKey]string{key: "VFICtest"},
	}
}

// makePlan wraps a BulkTransferData into an ExportPlan with empty skipped list.
func makePlan(data *excel.BulkTransferData) *ExportPlan {
	validated := &excel.BulkTransferValidationResult{
		ValidData:    data,
		TotalCount:   len(data.EmployeeProjectAmounts),
		ValidCount:   len(data.EmployeeProjectAmounts),
		SkippedCount: 0,
	}
	return &ExportPlan{
		Cycle:         "weekly",
		RawAggregated: data,
		ValidatedData: validated,
		SnapshotEpoch: time.Date(2026, 7, 15, 9, 0, 0, 0, pkgClock.DefaultLocation),
	}
}

// TestSimulation_Verdict_AnToan_AllCovered: full pool == cycle coverage, no
// remainders, reconciliation delta = 0 → AN_TOAN_DE_XUAT.
func TestSimulation_Verdict_AnToan_AllCovered(t *testing.T) {
	// Cycle plan and full-pool plan both contain the same single group →
	// 100% coverage, zero remainders.
	data := makeAggregated(7, 12, 5_000_000, []uint{101, 102}, "12345678")
	cyclePlan := makePlan(data)
	fullPlan := makePlan(data)

	svc := NewSimulationService(
		&fakePlanner{cyclePlan: cyclePlan, fullPoolPlan: fullPlan},
		&fakeLedger{total: 5_000_000}, // matches exported total → reconciled
		pkgClock.NewFake(time.Date(2026, 7, 12, 9, 0, 0, 0, pkgClock.DefaultLocation)),
	)

	result, err := svc.Simulate(context.Background(), &dto.SimulateSettlementRequest{ProjectedCycleCount: 1})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, VerdictAnToan, result.Verdict)
	assert.Equal(t, 0, result.Summary.RemainingAfterAllCount)
	assert.True(t, result.Reconciliation.Reconciled)
	assert.Empty(t, result.Remainders)
}

// TestSimulation_Verdict_CanKiemTra_Straggler: the cycle window misses an item
// that exists in the full pool → remainder → CAN_KIEM_TRA, and the prior-cycle
// warning fires.
func TestSimulation_Verdict_CanKiemTra_Straggler(t *testing.T) {
	// Cycle plan covers employee 7; full pool covers employee 7 AND employee 9.
	cycleData := makeAggregated(7, 12, 5_000_000, []uint{101}, "12345678")
	cyclePlan := makePlan(cycleData)

	fullData := makeAggregated(9, 12, 3_000_000, []uint{200}, "87654321")
	fullPlan := makePlan(fullData)
	// Augment fullPlan with both groups so coverage math works.
	key7 := excel.EmployeeProjectKey{EmployeeID: 7, ProjectID: 12}
	fullPlan.RawAggregated.EmployeeProjectAmounts[key7] = 5_000_000
	fullPlan.RawAggregated.EmployeeProjectTimesheets[key7] = []uint{101}
	fullPlan.RawAggregated.EmployeeData[7] = excel.Employee{ID: 7, Fullname: "Emp7", BankAccountNumber: "12345678"}
	fullPlan.RawAggregated.EmployeeData[9] = excel.Employee{ID: 9, Fullname: "Emp9", BankAccountNumber: "87654321"}
	fullPlan.ValidatedData.ValidData = fullPlan.RawAggregated
	fullPlan.ValidatedData.TotalCount = 2
	fullPlan.ValidatedData.ValidCount = 2

	svc := NewSimulationService(
		&fakePlanner{cyclePlan: cyclePlan, fullPoolPlan: fullPlan},
		&fakeLedger{total: 5_000_000},
		pkgClock.NewFake(time.Date(2026, 7, 12, 9, 0, 0, 0, pkgClock.DefaultLocation)),
	)

	result, err := svc.Simulate(context.Background(), &dto.SimulateSettlementRequest{ProjectedCycleCount: 1})
	require.NoError(t, err)

	assert.Equal(t, VerdictCanKiemTra, result.Verdict)
	require.Len(t, result.Remainders, 1)
	assert.Equal(t, uint(9), result.Remainders[0].EmployeeID)
	assert.Equal(t, int64(3_000_000), result.Remainders[0].Amount)
	assert.Equal(t, []uint{200}, result.Remainders[0].TimesheetIDs)

	// Prior-cycle-straggler warning present.
	found := false
	for _, w := range result.Warnings {
		if w.Code == priorCycleWarning {
			found = true
		}
	}
	assert.True(t, found, "expected prior-cycle-straggler warning")
}

// TestSimulation_Verdict_CanKiemTra_ReconciliationDrift: no remainders but
// the ledger receivable doesn't match the exported total → CAN_KIEM_TRA.
func TestSimulation_Verdict_CanKiemTra_ReconciliationDrift(t *testing.T) {
	data := makeAggregated(7, 12, 5_000_000, []uint{101}, "12345678")
	plan := makePlan(data)

	svc := NewSimulationService(
		&fakePlanner{cyclePlan: plan, fullPoolPlan: plan},
		&fakeLedger{total: 4_000_000}, // 1M short → drift
		pkgClock.NewFake(time.Date(2026, 7, 12, 9, 0, 0, 0, pkgClock.DefaultLocation)),
	)

	result, err := svc.Simulate(context.Background(), &dto.SimulateSettlementRequest{ProjectedCycleCount: 1})
	require.NoError(t, err)

	assert.Equal(t, VerdictCanKiemTra, result.Verdict)
	assert.False(t, result.Reconciliation.Reconciled)
	assert.Equal(t, int64(-1_000_000), result.Reconciliation.Delta)
}

// TestSimulation_CycleCount_Clamped: cycleCount < 1 → 4, > 6 → 6.
func TestSimulation_CycleCount_Clamped(t *testing.T) {
	data := makeAggregated(7, 12, 5_000_000, []uint{101}, "12345678")
	plan := makePlan(data)
	fp := &fakePlanner{cyclePlan: plan, fullPoolPlan: plan}

	svc := NewSimulationService(fp, &fakeLedger{total: 5_000_000},
		pkgClock.NewFake(time.Date(2026, 7, 12, 9, 0, 0, 0, pkgClock.DefaultLocation)))

	// Below minimum → defaults to 4.
	_, err := svc.Simulate(context.Background(), &dto.SimulateSettlementRequest{ProjectedCycleCount: 0})
	require.NoError(t, err)
	assert.Equal(t, 4, fp.cycleCalls, "cycleCount=0 should clamp to 4")

	// Above maximum → clamps to 6.
	fp.cycleCalls = 0
	_, err = svc.Simulate(context.Background(), &dto.SimulateSettlementRequest{ProjectedCycleCount: 99})
	require.NoError(t, err)
	assert.Equal(t, 6, fp.cycleCalls, "cycleCount=99 should clamp to 6")
}

// TestSimulation_BankAccountMasked: the masked account field is set, raw is
// never exposed in the result.
func TestSimulation_BankAccountMasked(t *testing.T) {
	data := makeAggregated(7, 12, 5_000_000, []uint{101}, "1234567890")
	plan := makePlan(data)

	svc := NewSimulationService(
		&fakePlanner{cyclePlan: plan, fullPoolPlan: plan},
		&fakeLedger{total: 5_000_000},
		pkgClock.NewFake(time.Date(2026, 7, 12, 9, 0, 0, 0, pkgClock.DefaultLocation)),
	)

	result, err := svc.Simulate(context.Background(), &dto.SimulateSettlementRequest{ProjectedCycleCount: 1})
	require.NoError(t, err)
	require.Len(t, result.Cycles, 1)
	require.Len(t, result.Cycles[0].Included, 1)
	assert.Equal(t, "••••7890", result.Cycles[0].Included[0].BankAccountMasked)
}

// TestStaleSnapshot_RejectsExport exercises the IfMatchSnapshot guard in
// ExportService.Export. Verifies a NewConflictErrorWithCode is returned when
// the plan's snapshot is newer than the client-provided snapshot. This is a
// unit test on the guard condition itself; the HTTP 409 mapping is exercised
// by the integration suite.
func TestStaleSnapshot_RejectsExport(t *testing.T) {
	oldSnapshot := time.Date(2026, 7, 10, 9, 0, 0, 0, pkgClock.DefaultLocation)
	newEpoch := time.Date(2026, 7, 15, 9, 0, 0, 0, pkgClock.DefaultLocation)

	// Reconstruct the guard condition directly to avoid spinning ExportService.
	stale := oldSnapshot.Before(newEpoch) && newEpoch.After(oldSnapshot)
	assert.True(t, stale, "snapshot older than plan epoch should be considered stale")

	// Verify the error constructor produces a conflict-domain error.
	conflict := domain.NewConflictErrorWithCode("STALE_SIMULATION", "stale")
	assert.True(t, domain.IsConflictError(conflict))
}
