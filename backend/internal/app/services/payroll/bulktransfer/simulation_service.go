package bulktransfer

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	pkgClock "api-server/internal/pkg/clock"
	"api-server/internal/pkg/timeutil"
)

// Simulation verdicts. KHONG_THE_TAT_TOAN is reserved for a future FlexPay
// scope where the op-loss class is meaningful; in payroll-only scope every
// remainder is unpaid wages (the eligible pool is payment_status IN (pending,
// failed), i.e. not yet paid) so only the first two verdicts can fire.
const (
	VerdictAnToan       = "AN_TOAN_DE_XUAT" // safe to export — full pool covered, no blocking, reconciled
	VerdictCanKiemTra   = "CAN_KIEM_TRA"    // needs review — remainders, blocking findings, or reconciliation drift
	VerdictKhongThe     = "KHONG_THE_TAT_TOAN"
	priorCycleWarning   = "PRIOR_CYCLE_STRAGGLERS"
	priorCycleWarningMsg = "Có giao dịch thuộc kỳ trước vẫn chưa thanh toán — sẽ KHÔNG tự động bao phủ. Xem 'Còn lại'."
)

// SimulationService projects the current payroll cycle + next N−1 cycles by
// reusing the production ExportPlanner.Plan() for each cycle's date window,
// then runs one additional full-pool Plan() to compute the coverage verdict.
//
// Strictly read-only: Simulate() only calls ExportPlanner.Plan() (pure, by
// Phase 1 invariant), GetAccountTotalInRange (read-only SUM), and the
// excel.ValidateAndFilterBulkTransferData helper. Zero writes. The grep
// assertion test in Phase 5 enforces no write-call substrings appear here.
type SimulationService struct {
	planner      Planner       // indirection for testability
	ledgerService LedgerService // local interface — has GetAccountTotalInRange
	clock        pkgClock.Clock
	logger       *slog.Logger
}

// Planner is the read-only surface SimulationService consumes from ExportPlanner.
// Declared as an interface so tests can substitute a fake without spinning a DB.
type Planner interface {
	Plan(ctx context.Context, req *dto.ExportBulkTransferRequest) (*ExportPlan, error)
}

// LedgerSumReader is the read-only ledger subset SimulationService consumes.
// Declared as an interface for testability — the production LedgerService
// (bulktransfer.LedgerService) satisfies it via GetAccountTotalInRange.
type LedgerSumReader interface {
	GetAccountTotalInRange(ctx context.Context, account domain.LedgerAccount, from, to time.Time) (int64, error)
}

// NewSimulationService constructs the service. planner is typically
// exportService.Planner(); ledgerService is the bulktransfer.LedgerService
// (which now carries GetAccountTotalInRange).
func NewSimulationService(planner Planner, ledgerService LedgerService, clock pkgClock.Clock) *SimulationService {
	return &SimulationService{
		planner:       planner,
		ledgerService: ledgerService,
		clock:         clock,
		logger:        observability.GetLogger().With("component", "SettlementSimulationService"),
	}
}

// Simulate projects the current cycle + next N−1 cycles and computes a
// full-pool coverage verdict. Read-only.
func (s *SimulationService) Simulate(ctx context.Context, req *dto.SimulateSettlementRequest) (*dto.SimulationResult, error) {
	// 1. Resolve starting cycle and project count.
	cycleCount := req.ProjectedCycleCount
	if cycleCount < 1 {
		cycleCount = 4
	}
	if cycleCount > 6 {
		cycleCount = 6
	}

	now := s.clock.Now()
	startCycle := pkgClock.NextTimesheetPayCycle(now)

	// 2. Project each cycle by calling the production planner with that cycle's
	//    date window. Accumulate covered timesheet IDs for the full-pool verdict.
	cycles := make([]dto.CycleProjection, 0, cycleCount)
	coveredTimesheetIDs := make(map[uint]struct{})
	var snapshotEpoch time.Time
	var spanFrom, spanTo time.Time
	current := startCycle

	for i := 0; i < cycleCount; i++ {
		fromDate, toDate := current.CycleWindow()
		cycleReq := buildCycleRequest(req, fromDate, toDate)

		plan, err := s.planner.Plan(ctx, cycleReq)
		if err != nil {
			return nil, fmt.Errorf("cycle Kỳ %d: %w", current.Ky, err)
		}

		projection := buildCycleProjection(i+1, current, plan)
		cycles = append(cycles, projection)

		// Track coverage + snapshot + span across cycles.
		for _, ids := range plan.RawAggregated.EmployeeProjectTimesheets {
			for _, id := range ids {
				coveredTimesheetIDs[id] = struct{}{}
			}
		}
		if plan.SnapshotEpoch.After(snapshotEpoch) {
			snapshotEpoch = plan.SnapshotEpoch
		}
		if i == 0 {
			spanFrom = fromDate
		}
		spanTo = toDate

		current = pkgClock.NextPayCycleAfter(current)
	}

	// 3. Full-pool scan: Plan with NoDateFilter=true. Same selection logic,
	//    no date window — every outstanding approved timesheet.
	fullPoolReq := buildFullPoolRequest(req)
	fullPoolPlan, err := s.planner.Plan(ctx, fullPoolReq)
	if err != nil {
		return nil, fmt.Errorf("full-pool scan: %w", err)
	}
	if fullPoolPlan.SnapshotEpoch.After(snapshotEpoch) {
		snapshotEpoch = fullPoolPlan.SnapshotEpoch
	}

	// 4. Remainders = full pool minus covered. Every remainder is unpaid wages
	//    by construction (eligible pool excludes payment_status=paid).
	remainders := buildRemainders(fullPoolPlan, coveredTimesheetIDs)

	// 5. Reconciliation: SUM(debit − credit) over receivable account for the
	//    projected span, compared to the total the N cycles will export.
	var totalIncluded int64
	for _, c := range cycles {
		totalIncluded += c.IncludedAmount
	}
	receivable, recErr := s.ledgerService.GetAccountTotalInRange(ctx, domain.AccountReceivable, spanFrom, spanTo)
	if recErr != nil {
		// Reconciliation is best-effort; surface as a warning rather than abort.
		s.logger.Warn("simulation: receivable query failed", "error", recErr)
		receivable = 0
	}
	reconciliation := dto.ReconciliationResult{
		ExportedTotal:    totalIncluded,
		LedgerReceivable: receivable,
		Delta:            receivable - totalIncluded,
		Reconciled:       receivable == totalIncluded,
	}

	// 6. Verdict (2 actionable states in payroll scope).
	hasBlocking := cyclesHaveBlocking(cycles)
	warnings := buildWarnings(len(remainders) > 0)
	verdict := computeVerdict(remainders, hasBlocking, reconciliation, warnings)

	// 7. Summary.
	summary := buildSummary(fullPoolPlan, cycles, remainders)

	return &dto.SimulationResult{
		SnapshotEpoch:       snapshotEpoch,
		StartingCycle:       buildCycleMeta(startCycle),
		ProjectedCycleCount: cycleCount,
		Verdict:             verdict,
		Summary:             summary,
		Reconciliation:      reconciliation,
		Cycles:              cycles,
		Remainders:          remainders,
		Warnings:            warnings,
	}, nil
}

// buildCycleRequest clones the simulation request scoped to a cycle's window.
// Uses weekly cycle mode (Kỳ cycles are weekly payment windows).
func buildCycleRequest(req *dto.SimulateSettlementRequest, from, to time.Time) *dto.ExportBulkTransferRequest {
	return &dto.ExportBulkTransferRequest{
		ProjectIDs:  append([]uint(nil), req.ProjectIDs...),
		EmployeeIDs: append([]uint(nil), req.EmployeeIDs...),
		FromDate:    from.Format(timeutil.DateFormat),
		ToDate:      to.Format(timeutil.DateFormat),
		CreatedBy:   req.CreatedBy,
	}
}

// buildFullPoolRequest returns a planner request with NoDateFilter=true so
// Plan() selects every outstanding approved timesheet regardless of cycle.
func buildFullPoolRequest(req *dto.SimulateSettlementRequest) *dto.ExportBulkTransferRequest {
	return &dto.ExportBulkTransferRequest{
		ProjectIDs:   append([]uint(nil), req.ProjectIDs...),
		EmployeeIDs:  append([]uint(nil), req.EmployeeIDs...),
		CreatedBy:    req.CreatedBy,
		NoDateFilter: true,
	}
}

// buildCycleProjection assembles the per-cycle projection from a Plan result.
func buildCycleProjection(seq int, cycle pkgClock.TimesheetPayCycle, plan *ExportPlan) dto.CycleProjection {
	fromDate, toDate := cycle.CycleWindow()
	label := fmt.Sprintf("Kỳ %d", cycle.Ky)
	if seq == 1 {
		label += " (hiện tại)"
	}

	var includedAmount int64
	included := make([]dto.SimulationRow, 0)
	if plan.ValidatedData != nil && plan.ValidatedData.ValidData != nil {
		vd := plan.ValidatedData.ValidData
		for key, amount := range vd.EmployeeProjectAmounts {
			emp := vd.EmployeeData[key.EmployeeID]
			proj := vd.ProjectData[key.ProjectID]
			includedAmount += amount
			included = append(included, dto.SimulationRow{
				EmployeeID:           emp.ID,
				EmployeeName:         emp.Fullname,
				ProjectID:            proj.ID,
				ProjectName:          proj.Name,
				Amount:               amount,
				TimesheetIDs:         append([]uint(nil), vd.EmployeeProjectTimesheets[key]...),
				BankAccountMasked:    maskBankAccount(emp.BankAccountNumber),
			})
		}
	}
	sortRows(included)

	excluded := make([]dto.SimulationExcludedRow, 0)
	if plan.ValidatedData != nil {
		for _, skipped := range plan.ValidatedData.SkippedEmployees {
			excluded = append(excluded, dto.SimulationExcludedRow{
				EmployeeID:   skipped.EmployeeID,
				EmployeeName: skipped.EmployeeName,
				ProjectID:    skipped.ProjectID,
				ProjectName:  skipped.ProjectName,
				Reason:       skipped.Reason,
				InProduction: true,
			})
		}
	}
	sortExcluded(excluded)

	// Convert any sim-only findings on this cycle.
	findings := make([]dto.SimFinding, 0)
	// (Phase 2 sim-only validators live in simulation_validators.go; for now
	// the per-cycle findings list is empty — the production ValidateAndFilter
	// results surface as excluded rows above. Sim-only validators are added
	// incrementally.)

	return dto.CycleProjection{
		Sequence:          seq,
		Label:             label,
		FromDate:          fromDate.Format(timeutil.DateFormat),
		ToDate:            toDate.Format(timeutil.DateFormat),
		PayDate:           cycle.NextPayDate.Format(timeutil.DateFormat),
		IncludedCount:     len(included),
		IncludedAmount:    includedAmount,
		ExcludedCount:     len(excluded),
		RemainingAfter:    0, // computed at the result level; per-cycle not meaningful without status flips
		RemainingAmount:   0,
		Included:          included,
		Excluded:          excluded,
		Findings:          findings,
	}
}

// buildRemainders returns every full-pool item not covered by any cycle window.
// Every remainder is unpaid wages — the eligible pool excludes paid items.
func buildRemainders(fullPool *ExportPlan, covered map[uint]struct{}) []dto.RemainderRow {
	if fullPool == nil || fullPool.ValidatedData == nil || fullPool.ValidatedData.ValidData == nil {
		return []dto.RemainderRow{}
	}
	vd := fullPool.ValidatedData.ValidData
	out := make([]dto.RemainderRow, 0)

	for key, amount := range vd.EmployeeProjectAmounts {
		emp := vd.EmployeeData[key.EmployeeID]
		proj := vd.ProjectData[key.ProjectID]
		tsIDs := vd.EmployeeProjectTimesheets[key]

		// How many of this group's timesheets are NOT covered by any cycle?
		uncovered := make([]uint, 0, len(tsIDs))
		for _, id := range tsIDs {
			if _, ok := covered[id]; !ok {
				uncovered = append(uncovered, id)
			}
		}
		if len(uncovered) == 0 {
			continue
		}

		// Pro-rate amount by uncovered fraction (best-effort; if all uncovered
		// the full amount applies).
		amt := amount
		if len(uncovered) < len(tsIDs) {
			amt = amount * int64(len(uncovered)) / int64(len(tsIDs))
		}

		out = append(out, dto.RemainderRow{
			EmployeeID:        emp.ID,
			EmployeeName:      emp.Fullname,
			ProjectID:         proj.ID,
			ProjectName:       proj.Name,
			Amount:            amt,
			TimesheetIDs:      uncovered,
			Reason:            "Thuộc kỳ trước hoặc ngoài các kỳ mô phỏng — chưa được bao phủ",
			BankAccountMasked: maskBankAccount(emp.BankAccountNumber),
		})
	}
	sortRemainders(out)
	return out
}

// computeVerdict applies the 2-state payroll-scope verdict logic.
//
// The informational "production does not validate X" warning is always
// present (it tells the admin about sim-only checks) and must NOT by itself
// escalate the verdict — otherwise AN_TOAN_DE_XUAT would be unreachable.
// Only actionable signals escalate: actual remainders, blocking findings,
// reconciliation drift, or non-informational warnings (prior-cycle
// stragglers, etc.).
func computeVerdict(remainders []dto.RemainderRow, hasBlocking bool, rec dto.ReconciliationResult, warnings []dto.SimWarning) string {
	hasActionableWarning := false
	for _, w := range warnings {
		// PRODUCTION_DOES_NOT_VALIDATE is informational — always present.
		if w.Code != "PRODUCTION_DOES_NOT_VALIDATE" {
			hasActionableWarning = true
			break
		}
	}
	switch {
	case len(remainders) > 0 || hasBlocking || !rec.Reconciled || hasActionableWarning:
		return VerdictCanKiemTra
	default:
		return VerdictAnToan
	}
}

func cyclesHaveBlocking(cycles []dto.CycleProjection) bool {
	for _, c := range cycles {
		for _, f := range c.Findings {
			if f.Severity == "blocking" {
				return true
			}
		}
	}
	return false
}

func buildWarnings(hasStragglers bool) []dto.SimWarning {
	w := []dto.SimWarning{
		{Code: "PRODUCTION_DOES_NOT_VALIDATE", Message: "Sản xuất không kiểm tra mã ngân hàng, số tiền âm, hoặc trùng lặp giữa các kỳ. Xem chi tiết trong từng kỳ."},
	}
	if hasStragglers {
		w = append(w, dto.SimWarning{Code: priorCycleWarning, Message: priorCycleWarningMsg})
	}
	return w
}

func buildSummary(fullPool *ExportPlan, cycles []dto.CycleProjection, remainders []dto.RemainderRow) dto.SimulationSummary {
	var totalEligibleCount int
	var totalEligibleAmount int64
	if fullPool != nil && fullPool.ValidatedData != nil && fullPool.ValidatedData.ValidData != nil {
		totalEligibleCount = len(fullPool.ValidatedData.ValidData.EmployeeProjectAmounts)
		for _, amt := range fullPool.ValidatedData.ValidData.EmployeeProjectAmounts {
			totalEligibleAmount += amt
		}
	}

	var totalIncludedCount int
	var totalIncludedAmount int64
	for _, c := range cycles {
		totalIncludedCount += c.IncludedCount
		totalIncludedAmount += c.IncludedAmount
	}

	var remainderCount int
	var remainderAmount int64
	for _, r := range remainders {
		remainderCount++
		remainderAmount += r.Amount
	}

	return dto.SimulationSummary{
		TotalEligibleCount:       totalEligibleCount,
		TotalEligibleAmount:      totalEligibleAmount,
		TotalIncludedCount:       totalIncludedCount,
		TotalIncludedAmount:      totalIncludedAmount,
		RemainingAfterAllCount:   remainderCount,
		RemainingAfterAllAmount:  remainderAmount,
		AllSettled:               remainderCount == 0,
	}
}

func buildCycleMeta(c pkgClock.TimesheetPayCycle) dto.CycleMeta {
	from, to := c.CycleWindow()
	return dto.CycleMeta{
		Index:     c.Ky,
		MonthRef:  c.WorkMonth.Format(timeutil.DateFormat),
		FromDate:  from.Format(timeutil.DateFormat),
		ToDate:    to.Format(timeutil.DateFormat),
		PayDate:   c.NextPayDate.Format(timeutil.DateFormat),
	}
}

// maskBankAccount returns "••••<last4>" or "" when empty. The simulation API
// never returns raw bank account numbers (red-team Finding 10).
func maskBankAccount(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return strings.Repeat("•", len(s))
	}
	return "••••" + s[len(s)-4:]
}

// Deterministic sort orders so repeated simulations with the same data return
// the same row ordering.
func sortRows(r []dto.SimulationRow) {
	sort.Slice(r, func(i, j int) bool {
		if r[i].EmployeeID != r[j].EmployeeID {
			return r[i].EmployeeID < r[j].EmployeeID
		}
		return r[i].ProjectID < r[j].ProjectID
	})
}
func sortExcluded(r []dto.SimulationExcludedRow) {
	sort.Slice(r, func(i, j int) bool {
		if r[i].EmployeeID != r[j].EmployeeID {
			return r[i].EmployeeID < r[j].EmployeeID
		}
		return r[i].ProjectID < r[j].ProjectID
	})
}
func sortRemainders(r []dto.RemainderRow) {
	sort.Slice(r, func(i, j int) bool {
		if r[i].Amount != r[j].Amount {
			return r[i].Amount > r[j].Amount // largest unpaid first
		}
		if r[i].EmployeeID != r[j].EmployeeID {
			return r[i].EmployeeID < r[j].EmployeeID
		}
		return r[i].ProjectID < r[j].ProjectID
	})
}
