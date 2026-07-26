package payroll

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	pkgClock "api-server/internal/pkg/clock"
	"api-server/internal/pkg/timeutil"
)

// Simulation verdicts.
const (
	VerdictAnToan     = "AN_TOAN_DE_XUAT" // safe — full pool covered, reconciled
	VerdictCanKiemTra = "CAN_KIEM_TRA"    // needs review — remainders or reconciliation drift
)

// SettlementSimulationService projects N future exports starting from an
// admin-chosen date, reusing the EXACT same selection logic the production
// "Xuất sao kê" endpoint (GET /timesheets/payroll/report) uses.
//
// Model (mirrors the real admin workflow):
//   - Admin picks a start_date (e.g. 2026-07-26).
//   - Server generates N export dates by stepping forward cadence_days
//     (default 7) each time.
//   - For each export date, calls PayrollReportByProjectService.
//     GetProjectsForPayrollReport(ctx, exportDate) — the SAME function the
//     real export handler calls. Zero drift: whatever production would pick
//     up, the sim picks up.
//   - After each export, the paid-but-unreconciled timesheets it covered
//     are assumed reconciled (revenue_paid=true) and removed from the pool.
//   - The full pool (denominator) is queried directly: every paid timesheet
//     with revenue_paid=false, across ALL projects, no date filter. This
//     avoids the day-of-month project-eligibility rule (which would return
//     zero projects on days 11–23) for the pool question.
//   - Remainders = full pool − covered; each carries employee/project/date.
//
// Strictly read-only: only calls GetProjectsForPayrollReport + a read-only
// timesheet List + GetAccountTotalForTransactions. Zero writes.
type SettlementSimulationService struct {
	reportService ReportSelector
	timesheetRepo domain.TimesheetRepository
	ledgerService LedgerSumReader
	clock         pkgClock.Clock
	logger        *slog.Logger
}

// ReportSelector is the read-only surface SettlementSimulationService consumes
// from PayrollReportByProjectService. Declared as an interface for testability.
type ReportSelector interface {
	GetProjectsForPayrollReport(ctx context.Context, atDate time.Time) ([]*domainServices.ProjectReportData, error)
}

// LedgerSumReader is the read-only ledger reconciliation subset.
type LedgerSumReader interface {
	GetAccountTotalForTransactions(ctx context.Context, account string, transactionIDs []uint) (int64, error)
}

// NewSettlementSimulationService constructs the service.
func NewSettlementSimulationService(reportService ReportSelector, tsRepo domain.TimesheetRepository, ledgerService LedgerSumReader, clock pkgClock.Clock) *SettlementSimulationService {
	return &SettlementSimulationService{
		reportService: reportService,
		timesheetRepo: tsRepo,
		ledgerService: ledgerService,
		clock:         clock,
		logger:        observability.GetLogger().With("component", "SettlementSimulationService"),
	}
}

// Simulate projects N exports and computes a coverage verdict.
// Simulate projects N future exports starting from an admin-chosen date.
// Export dates follow the real sao kê cadence: day-26 and day-2 of each month,
// alternating. The admin picks the first date (must be a 26 or 2); the server
// generates the next N-1 by walking the 26→2→26→2 sequence forward.
func (s *SettlementSimulationService) Simulate(ctx context.Context, req *dto.SimulateSettlementRequest) (*dto.SimulationResult, error) {
	// 1. Resolve + clamp parameters.
	startDate, err := parseStartDate(req.StartDate, s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	exportCount := clampInt(req.ExportCount, 1, 10, 4)

	// 2. Generate the N export dates by alternating day-26 and day-2.
	exportTimes := generateExportDates(startDate, exportCount)
	exportDates := make([]string, 0, exportCount)
	for _, t := range exportTimes {
		exportDates = append(exportDates, t.Format(timeutil.DateFormat))
	}

	// 3. Full pool: every paid-but-unreconciled timesheet, across ALL projects.
	//    Queried directly (not via GetProjectsForPayrollReport) because the
	//    report service applies a day-of-month project-eligibility rule that
	//    returns zero projects on days 11–23 — wrong for the "what's the total
	//    outstanding pool" question. This matches the same row filter
	//    getPaidTimesheets uses (payment_status=paid, revenue_paid=false).
	paidFalse := domain.PaymentStatusPaid
	fullPoolTs, err := s.timesheetRepo.List(ctx, domain.TimesheetFilters{
		PaymentStatus: []domain.PaymentStatus{paidFalse},
		Limit:         100000,
	})
	if err != nil {
		return nil, fmt.Errorf("full-pool scan: %w", err)
	}
	// getPaidTimesheets also excludes revenue_paid=true — filter in Go to match.
	fullPoolTs = filterRevenuePaidFalse(fullPoolTs)
	dateByID := make(map[uint]string, len(fullPoolTs))
	for _, ts := range fullPoolTs {
		if ts != nil {
			dateByID[ts.ID] = ts.Date.Format(timeutil.DateFormat)
		}
	}

	// 4. Run each export. Track covered IDs across exports (rolling: items
	//    covered by export k are excluded from export k+1's "newly covered").
	covered := make(map[uint]struct{})
	exports := make([]dto.ExportProjection, 0, exportCount)

	for i, exportDate := range exportTimes {
		reportData, err := s.reportService.GetProjectsForPayrollReport(ctx, exportDate)
		if err != nil {
			s.logger.Warn("simulation: export call failed, treating as empty", "export_date", exportDate, "error", err)
			reportData = nil
		}

		projection := buildExportProjectionFromReport(i+1, exportDate, reportData, covered, dateByID)
		exports = append(exports, projection)
		// Mark this export's items as covered for subsequent exports.
		for _, p := range reportData {
			for _, id := range p.TimesheetIDs {
				covered[id] = struct{}{}
			}
		}
	}

	// 5. Remainders = full pool − covered. Built from the raw timesheet slice.
	remainders := buildRemaindersFromTimesheets(fullPoolTs, covered, dateByID)

	// 6. Reconcile the projected timesheets against the exact same ledger
	//    transaction cohort. The export table shows employee paid_amount, while
	//    receivable ledger entries include the service fee, so the comparator
	//    must be RevenueReceivable rather than the wage principal.
	expectedReceivable, transactionIDs := buildReconciliationScope(fullPoolTs, covered)
	receivable, recErr := s.ledgerService.GetAccountTotalForTransactions(ctx, "receivable", transactionIDs)
	if recErr != nil {
		s.logger.Warn("simulation: receivable query failed", "error", recErr)
		receivable = 0
	}
	reconciliation := dto.ReconciliationResult{
		ExportedTotal:    expectedReceivable,
		LedgerReceivable: receivable,
		Delta:            receivable - expectedReceivable,
		Reconciled:       recErr == nil && receivable == expectedReceivable,
	}

	// 7. Summary + verdict.
	summary := buildSummaryFromTimesheets(fullPoolTs, exports, remainders, covered)
	warnings := buildWarnings(len(remainders) > 0, recErr != nil)
	verdict := computeVerdict(remainders, reconciliation, warnings)

	return &dto.SimulationResult{
		StartDate:      exportDates[0],
		ExportDates:    exportDates,
		Verdict:        verdict,
		Summary:        summary,
		Reconciliation: reconciliation,
		Exports:        exports,
		Remainders:     remainders,
		Warnings:       warnings,
	}, nil
}

// ---------- helpers ----------

// filterRevenuePaidFalse keeps only timesheets where RevenuePaid is false,
// matching getPaidTimesheets' revenue_paid=false filter.
func filterRevenuePaidFalse(tses []*domain.Timesheet) []*domain.Timesheet {
	out := make([]*domain.Timesheet, 0, len(tses))
	for _, ts := range tses {
		if ts == nil || ts.RevenuePaid {
			continue
		}
		out = append(out, ts)
	}
	return out
}

func parseStartDate(s string, now time.Time) (time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return time.Time{}, fmt.Errorf("start_date là bắt buộc (định dạng YYYY-MM-DD)")
	}
	t, err := timeutil.ParseBusinessDate(strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, fmt.Errorf("start_date không hợp lệ: %w", err)
	}
	return t, nil
}

func clampInt(v, lo, hi, def int) int {
	if v == 0 {
		return def
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// generateExportDates produces N export dates starting from startDate, walking
// the real sao kê cadence forward: day-26 → next-month day-2 → day-26 → ...
//
// The admin picks the first date (typically day 26 or day 2 of a month). The
// generator then alternates: if the current date is ~day 26, the next is day 2
// of the next month; if the current date is ~day 2, the next is day 26 of the
// same month. This matches the production export schedule encoded in
// isProjectEligibleOnDay (only days >= 24 and <= 10 trigger exports).
func generateExportDates(startDate time.Time, count int) []time.Time {
	out := make([]time.Time, 0, count)
	current := startDate
	for i := 0; i < count; i++ {
		out = append(out, current)
		current = nextSaoKeDate(current)
	}
	return out
}

// nextSaoKeDate returns the next export date after t, following the 26↔2 rule:
//   - If t is on or before day 10 → next is day 26 of the same month.
//   - Otherwise (day 11–31) → next is day 2 of the next month.
func nextSaoKeDate(t time.Time) time.Time {
	loc := t.Location()
	if t.Day() <= 10 {
		// Day-2 export done → next is day 26 of same month.
		return time.Date(t.Year(), t.Month(), 26, 0, 0, 0, 0, loc)
	}
	// Day-26 export done → next is day 2 of next month.
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, 1, 1)
}

// buildExportProjectionFromReport assembles one export's view from a
// PayrollReportByProjectService result. Timesheets already covered by a prior
// export are NOT re-counted in this export's Included.
func buildExportProjectionFromReport(seq int, exportDate time.Time, reports []*domainServices.ProjectReportData, covered map[uint]struct{}, dateByID map[uint]string) dto.ExportProjection {
	var includedAmount int64
	included := make([]dto.SimulationRow, 0)
	includedTsCount := 0

	for _, r := range reports {
		if r.Project == nil {
			continue
		}
		// Group newly-covered timesheets by employee within this project.
		byEmp := make(map[uint][]*domain.Timesheet)
		for _, ts := range r.Timesheets {
			if ts == nil {
				continue
			}
			if _, isCovered := covered[ts.ID]; isCovered {
				continue
			}
			byEmp[ts.EmployeeID] = append(byEmp[ts.EmployeeID], ts)
		}
		for empID, tses := range byEmp {
			ids := make([]uint, 0, len(tses))
			dates := make([]string, 0, len(tses))
			var amt int64
			empName := fmt.Sprintf("NV #%d", empID)
			for _, ts := range tses {
				ids = append(ids, ts.ID)
				if d, ok := dateByID[ts.ID]; ok {
					dates = append(dates, d)
				} else {
					dates = append(dates, ts.Date.Format(timeutil.DateFormat))
				}
				amt += ts.PaidAmount
				if ts.Employee != nil && ts.Employee.FormattedFullname() != "" {
					empName = ts.Employee.FormattedFullname()
				}
			}
			sort.Strings(dates)
			includedAmount += amt
			includedTsCount += len(ids)
			included = append(included, dto.SimulationRow{
				EmployeeID:     empID,
				EmployeeName:   empName,
				ProjectID:      r.Project.ID,
				ProjectName:    r.Project.Name,
				Amount:         amt,
				TimesheetIDs:   ids,
				TimesheetDates: dates,
			})
		}
	}
	sortRows(included)

	return dto.ExportProjection{
		Sequence:       seq,
		ExportDate:     exportDate.Format(timeutil.DateFormat),
		ToDate:         exportDate.Format(timeutil.DateFormat),
		IncludedCount:  includedTsCount,
		IncludedAmount: includedAmount,
		Included:       included,
	}
}

// buildRemaindersFromTimesheets returns every full-pool timesheet not covered
// by ANY export, grouped by employee×project. Each row carries the underlying
// work dates so the admin can identify exactly which days need attention.
func buildRemaindersFromTimesheets(fullPool []*domain.Timesheet, covered map[uint]struct{}, dateByID map[uint]string) []dto.RemainderRow {
	type key struct{ emp, proj uint }
	groups := make(map[key]struct {
		tses []*domain.Timesheet
	})
	order := make([]key, 0)

	for _, ts := range fullPool {
		if ts == nil {
			continue
		}
		if _, ok := covered[ts.ID]; ok {
			continue
		}
		k := key{emp: ts.EmployeeID, proj: ts.ProjectID}
		if _, exists := groups[k]; !exists {
			order = append(order, k)
		}
		g := groups[k]
		g.tses = append(g.tses, ts)
		groups[k] = g
	}

	out := make([]dto.RemainderRow, 0, len(order))
	for _, k := range order {
		tses := groups[k].tses
		ids := make([]uint, 0, len(tses))
		dates := make([]string, 0, len(tses))
		var amt int64
		empName := fmt.Sprintf("NV #%d", k.emp)
		projName := fmt.Sprintf("Dự án #%d", k.proj)
		for _, ts := range tses {
			ids = append(ids, ts.ID)
			if d, ok := dateByID[ts.ID]; ok {
				dates = append(dates, d)
			} else {
				dates = append(dates, ts.Date.Format(timeutil.DateFormat))
			}
			amt += ts.PaidAmount
			if ts.Employee != nil && ts.Employee.FormattedFullname() != "" {
				empName = ts.Employee.FormattedFullname()
			}
			if ts.Project != nil && ts.Project.Name != "" {
				projName = ts.Project.Name
			}
		}
		sort.Strings(dates)
		out = append(out, dto.RemainderRow{
			EmployeeID:     k.emp,
			EmployeeName:   empName,
			ProjectID:      k.proj,
			ProjectName:    projName,
			Amount:         amt,
			TimesheetIDs:   ids,
			TimesheetDates: dates,
			Reason:         "Chưa được bao phủ bởi các lần xuất đã mô phỏng",
		})
	}
	sortRemainders(out)
	return out
}

func computeVerdict(remainders []dto.RemainderRow, rec dto.ReconciliationResult, warnings []dto.SimWarning) string {
	hasActionableWarning := false
	for _, w := range warnings {
		if w.Code != "PRODUCTION_DOES_NOT_VALIDATE" {
			hasActionableWarning = true
			break
		}
	}
	switch {
	case len(remainders) > 0 || !rec.Reconciled || hasActionableWarning:
		return VerdictCanKiemTra
	default:
		return VerdictAnToan
	}
}

func buildWarnings(hasRemainders, reconciliationFailed bool) []dto.SimWarning {
	w := []dto.SimWarning{
		{Code: "PRODUCTION_DOES_NOT_VALIDATE", Message: "Sản xuất không kiểm tra mã ngân hàng, số tiền âm, hoặc trùng lặp. Xem chi tiết trong từng lần xuất."},
	}
	if hasRemainders {
		w = append(w, dto.SimWarning{Code: "UNCOVERED_REMAINDERS", Message: "Có giao dịch chưa được bao phủ sau các lần xuất đã mô phỏng — xem danh sách 'Còn lại'."})
	}
	if reconciliationFailed {
		w = append(w, dto.SimWarning{Code: "LEDGER_RECONCILIATION_FAILED", Message: "Không thể đọc dữ liệu sổ cái để đối soát. Vui lòng thử lại."})
	}
	return w
}

// buildReconciliationScope returns the projected receivable amount and exact
// ledger transaction cohort for covered timesheets. Unlinked covered rows stay
// in the expected amount so missing accounting evidence remains visible.
func buildReconciliationScope(fullPool []*domain.Timesheet, covered map[uint]struct{}) (int64, []uint) {
	var expectedReceivable int64
	seenTransactionIDs := make(map[uint]struct{})

	for _, ts := range fullPool {
		if ts == nil {
			continue
		}
		if _, ok := covered[ts.ID]; !ok {
			continue
		}

		expectedReceivable += ts.RevenueReceivable
		if ts.TransactionID != nil {
			seenTransactionIDs[*ts.TransactionID] = struct{}{}
		}
	}

	transactionIDs := make([]uint, 0, len(seenTransactionIDs))
	for id := range seenTransactionIDs {
		transactionIDs = append(transactionIDs, id)
	}
	sort.Slice(transactionIDs, func(i, j int) bool {
		return transactionIDs[i] < transactionIDs[j]
	})

	return expectedReceivable, transactionIDs
}

// buildSummaryFromTimesheets computes the answer-first totals from the raw
// timesheet pool. Eligible = full pool (timesheet count + sum of paid_amount).
func buildSummaryFromTimesheets(fullPool []*domain.Timesheet, exports []dto.ExportProjection, remainders []dto.RemainderRow, covered map[uint]struct{}) dto.SimulationSummary {
	eligibleTs := len(fullPool)
	var eligibleAmt int64
	// Distinct employee×project groups in the pool.
	seenGroups := make(map[struct{ emp, proj uint }]struct{})
	for _, ts := range fullPool {
		if ts == nil {
			continue
		}
		eligibleAmt += ts.PaidAmount
		seenGroups[struct{ emp, proj uint }{ts.EmployeeID, ts.ProjectID}] = struct{}{}
	}

	remTs := 0
	var remAmt int64
	for _, r := range remainders {
		remTs += len(r.TimesheetIDs)
		remAmt += r.Amount
	}

	var incAmt int64
	for _, e := range exports {
		incAmt += e.IncludedAmount
	}

	return dto.SimulationSummary{
		TotalEligibleTimesheets: eligibleTs,
		TotalEligibleGroups:     len(seenGroups),
		TotalEligibleAmount:     eligibleAmt,
		TotalIncludedTimesheets: len(covered),
		TotalIncludedAmount:     incAmt,
		RemainingTimesheets:     remTs,
		RemainingGroups:         len(remainders),
		RemainingAmount:         remAmt,
		AllSettled:              len(remainders) == 0,
	}
}

func sortRows(r []dto.SimulationRow) {
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
			return r[i].Amount > r[j].Amount
		}
		if r[i].EmployeeID != r[j].EmployeeID {
			return r[i].EmployeeID < r[j].EmployeeID
		}
		return r[i].ProjectID < r[j].ProjectID
	})
}
