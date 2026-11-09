package dashboard

import (
	"context"
	"sort"
	"sync/atomic"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/infra/persistence/repositories"
)

// Precomputed bank-usage summary — the dashboard cards never run the legacy
// per-employee prefix walk on the request path.
//
// Semantics (must match the legacy queries exactly, because the fold is
// subtle):
//   - overall (project_id = 0): every paid timesheet of every employee who has
//     at least one ACTIVE assignment, each timesheet counted exactly once,
//     grouped by the employee's bank. A timesheet belongs to exactly one
//     (employee, project) bucket of the scan, so summing every bucket of an
//     employee reproduces the legacy employee-level totals.
//   - per project: the (employee, project) pair must be active AND the
//     timesheet must belong to that project — the legacy
//     GetBankUsageByProject/GetBankUsageAllProjects join.
//
// Employees with paid timesheets but no active assignment are excluded from
// both views (the legacy INNER JOIN on project_employees drops them).

const (
	// bankUsageSummaryMaxAge bounds how stale a served snapshot may be. The
	// asynq refresher runs every 2m; 15m means three missed refreshes still
	// serve warm data before a request falls back to the legacy path.
	bankUsageSummaryMaxAge = 15 * time.Minute
	// bankUsageRefreshTimeout bounds one inline refresh kicked from a request.
	bankUsageRefreshTimeout = 45 * time.Second
)

// bankUsageRefreshInFlight collapses concurrent stale-read fallbacks into one
// background refresh.
var bankUsageRefreshInFlight atomic.Bool

// RefreshBankUsageSummary recomputes the whole snapshot. Idempotent and
// snapshot-consistent (read scan + pairs, then one transactional swap).
func (s *Service) RefreshBankUsageSummary(ctx context.Context) error {
	scan, err := s.TimesheetAnalyticsRepo.ScanBankUsageByEmployeeProjectBank(ctx)
	if err != nil {
		return err
	}
	pairs, err := s.TimesheetAnalyticsRepo.ActiveProjectPairs(ctx)
	if err != nil {
		return err
	}
	return s.TimesheetAnalyticsRepo.ReplaceBankUsageSummary(ctx, foldBankUsageSummary(scan, pairs), time.Now())
}

// bankUsageSummaryFresh fetches a scope and reports whether it may be served.
// On stale/missing snapshots it kicks one background refresh so the next
// request finds warm data.
func (s *Service) bankUsageSummaryFresh(ctx context.Context, projectID *uint) ([]repositories.BankUsageSummaryRow, bool) {
	rows, computedMillis, err := s.TimesheetAnalyticsRepo.GetBankUsageSummary(ctx, projectID)
	if err == nil && computedMillis > 0 && time.Since(time.UnixMilli(computedMillis)) <= bankUsageSummaryMaxAge {
		return rows, true
	}
	if err != nil {
		s.logger.Error("failed to read bank usage summary", "error", err)
	}
	s.kickBankUsageRefresh()
	return nil, false
}

// kickBankUsageRefresh starts at most one detached refresh; the request path
// never waits for the 90s-class legacy scans.
func (s *Service) kickBankUsageRefresh() {
	if !bankUsageRefreshInFlight.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer bankUsageRefreshInFlight.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), bankUsageRefreshTimeout)
		defer cancel()
		if err := s.RefreshBankUsageSummary(ctx); err != nil {
			s.logger.Error("inline bank usage summary refresh failed", "error", err)
		}
	}()
}

// bankUsageAllProjectsFromSummary assembles the all-projects response from
// one summary read: scope 0 rows become the overall card, the rest group into
// project cards ordered by name (the legacy ORDER BY p.name). nil when the
// snapshot carries no live project rows and no overall buckets.
func (s *Service) bankUsageAllProjectsFromSummary(ctx context.Context, rows []repositories.BankUsageSummaryRow) *dto.BankUsageAllProjectsResponse {
	var overallRows []repositories.BankUsageSummaryRow
	projectRows := map[uint64][]repositories.BankUsageSummaryRow{}
	for _, row := range rows {
		if row.ProjectID == 0 {
			overallRows = append(overallRows, row)
			continue
		}
		projectRows[row.ProjectID] = append(projectRows[row.ProjectID], row)
	}
	if len(overallRows) == 0 && len(projectRows) == 0 {
		return nil
	}

	overallItems, overallEmployees, overallPaid, overallTransfers := bankUsageItemsFromSummary(overallRows)

	names, err := s.TimesheetAnalyticsRepo.ActiveProjectNames(ctx)
	if err != nil {
		s.logger.Error("failed to read project names for bank usage summary", "error", err)
		names = map[uint64]string{}
	}

	projects := make([]dto.ProjectBankUsageItem, 0, len(projectRows))
	for projectID, buckets := range projectRows {
		projectName, live := names[projectID]
		if !live {
			// The legacy join drops rows of deleted projects.
			continue
		}
		items, totalEmployees, _, _ := bankUsageItemsFromSummary(buckets)
		projects = append(projects, dto.ProjectBankUsageItem{
			ProjectID:      uint(projectID),
			ProjectName:    projectName,
			Banks:          items,
			TotalEmployees: totalEmployees,
		})
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].ProjectName < projects[j].ProjectName })

	return &dto.BankUsageAllProjectsResponse{
		Projects: projects,
		Overall: dto.BankUsageResponse{
			Banks:          overallItems,
			TotalEmployees: overallEmployees,
			TotalPaidVND:   overallPaid,
			TotalTransfers: overallTransfers,
		},
	}
}

// bankUsageItemsFromSummary shapes summary buckets into the response DTO with
// the same percentage math as the legacy loops.
func bankUsageItemsFromSummary(rows []repositories.BankUsageSummaryRow) ([]dto.BankUsageItem, int, int64, int) {
	if len(rows) == 0 {
		return nil, 0, 0, 0
	}
	items := make([]dto.BankUsageItem, 0, len(rows))
	totalEmployees := 0
	var totalPaid int64
	var totalTransfers int
	for _, row := range rows {
		totalEmployees += row.EmployeeCount
		totalPaid += int64(row.TotalPaid)
		totalTransfers += row.TransferCount
	}
	for _, row := range rows {
		pct := 0.0
		if totalEmployees > 0 {
			pct = float64(row.EmployeeCount) / float64(totalEmployees) * 100
		}
		items = append(items, dto.BankUsageItem{
			BankName:      row.BankName,
			EmployeeCount: row.EmployeeCount,
			TotalPaidVND:  int64(row.TotalPaid),
			TransferCount: row.TransferCount,
			Percentage:    roundPct(pct),
		})
	}
	return items, totalEmployees, totalPaid, totalTransfers
}

// foldBankUsageSummary projects the single (employee, project, bank) scan onto
// the overall + per-project views using the active-assignment sets. Emission
// order matches the legacy ORDER BY (employee_count DESC).
func foldBankUsageSummary(
	scan []repositories.EmployeeProjectBankUsageRow,
	pairs []repositories.ActiveProjectPair,
) []repositories.BankUsageSummaryRow {
	type activeKey struct{ employeeID, projectID uint64 }

	activeEmployees := make(map[uint64]struct{}, len(pairs))
	activePairs := make(map[activeKey]struct{}, len(pairs))
	for _, pair := range pairs {
		activeEmployees[pair.EmployeeID] = struct{}{}
		activePairs[activeKey{pair.EmployeeID, pair.ProjectID}] = struct{}{}
	}

	type bucket struct {
		employees map[uint64]struct{}
		paid      float64
		transfers int64
	}
	newBucket := func() *bucket {
		return &bucket{employees: map[uint64]struct{}{}}
	}
	add := func(b *bucket, employeeID uint64, paid float64, transfers int64) {
		b.employees[employeeID] = struct{}{}
		b.paid += paid
		b.transfers += transfers
	}

	overall := map[string]*bucket{}
	byProject := map[uint64]map[string]*bucket{}

	for _, row := range scan {
		if _, active := activeEmployees[row.EmployeeID]; !active {
			continue
		}
		overallBucket, ok := overall[row.BankName]
		if !ok {
			overallBucket = newBucket()
			overall[row.BankName] = overallBucket
		}
		add(overallBucket, row.EmployeeID, row.TotalPaid, row.TransferCount)

		if _, pairActive := activePairs[activeKey{row.EmployeeID, row.ProjectID}]; pairActive {
			projectBuckets, ok := byProject[row.ProjectID]
			if !ok {
				projectBuckets = map[string]*bucket{}
				byProject[row.ProjectID] = projectBuckets
			}
			projectBucket, ok := projectBuckets[row.BankName]
			if !ok {
				projectBucket = newBucket()
				projectBuckets[row.BankName] = projectBucket
			}
			add(projectBucket, row.EmployeeID, row.TotalPaid, row.TransferCount)
		}
	}

	rows := make([]repositories.BankUsageSummaryRow, 0, len(overall)+len(byProject)*4)
	emit := func(projectID uint64, buckets map[string]*bucket) {
		for bankName, b := range buckets {
			rows = append(rows, repositories.BankUsageSummaryRow{
				ProjectID:     projectID,
				BankName:      bankName,
				EmployeeCount: len(b.employees),
				TotalPaid:     b.paid,
				TransferCount: int(b.transfers),
			})
		}
	}
	emit(0, overall)
	for projectID, buckets := range byProject {
		emit(projectID, buckets)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].ProjectID != rows[j].ProjectID {
			return rows[i].ProjectID < rows[j].ProjectID
		}
		if rows[i].EmployeeCount != rows[j].EmployeeCount {
			return rows[i].EmployeeCount > rows[j].EmployeeCount
		}
		return rows[i].BankName < rows[j].BankName
	})
	return rows
}
