package persistence

import (
	"context"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// Zalo reachability report.
//
// Why the classification runs in Go instead of SQL: the invalid_mobile state is
// decided by phone.NormalizeVietnameseMobile, the single shape rule phone-based
// login also uses. Re-implementing that rule in SQL (regexp + prefix rewriting)
// would create a second, silently divergent definition of "a usable Vietnamese
// mobile". The employee table is small (order of a thousand active rows), so the
// repository reads the candidate signals once and classifies them in Go. The
// project filter stays in SQL, and pagination is applied to the classified rows
// because the shape test cannot be expressed in SQL.

// reachabilityLatestZNSJoin attaches the employee's MOST RECENT salary
// notification. "Most recent" is created_at with id as the tie-break, so two
// rows written in the same second still resolve deterministically. An employee
// whose latest row is not suppressed (for example a later successful send) is
// reachable and must not be reported.
const reachabilityLatestZNSJoin = `LEFT JOIN flexpay_salary_notifications z ON z.id = (
	SELECT n.id FROM flexpay_salary_notifications n
	WHERE n.employee_id = employees.id
	ORDER BY n.created_at DESC, n.id DESC
	LIMIT 1
)`

// reachabilitySharedMobileJoin counts ACTIVE employees per mobile value,
// including the row itself, so a count above 1 means at least one other active
// employee shares the number. Empty mobiles are excluded from the tally.
const reachabilitySharedMobileJoin = `LEFT JOIN (
	SELECT mobile, COUNT(*) AS mobile_matches
	FROM employees
	WHERE deleted_at IS NULL AND COALESCE(mobile, '') <> ''
	GROUP BY mobile
) shared_mobile ON shared_mobile.mobile = employees.mobile`

// reachabilityCandidate is one employee with the raw signals the domain
// classifier needs.
type reachabilityCandidate struct {
	EmployeeID    uint   `gorm:"column:employee_id"`
	Fullname      string `gorm:"column:fullname"`
	CCCD          string `gorm:"column:cccd"`
	Mobile        string `gorm:"column:mobile"`
	ZNSNoLinked   bool   `gorm:"column:zns_no_linked"`
	MobileMatches int64  `gorm:"column:mobile_matches"`
}

// reachabilityAmountRow carries the per-employee money exposure.
type reachabilityAmountRow struct {
	EmployeeID        uint  `gorm:"column:employee_id"`
	PaidAmount        int64 `gorm:"column:paid_amount"`
	OutstandingAmount int64 `gorm:"column:outstanding_amount"`
}

// reachabilityAssignmentRow is one current project assignment of one employee.
type reachabilityAssignmentRow struct {
	EmployeeID  uint   `gorm:"column:employee_id"`
	ProjectName string `gorm:"column:project_name"`
}

// ListEmployeeReachability returns the employees matching the filters, ordered
// by employee id, with their money exposure and current project names.
//
// The state filter and pagination are applied after classification, which is why
// this method does the offset/limit handling itself rather than in SQL. Counts
// describing the whole cohort come from CountEmployeeReachability.
func (r *EmployeeRepository) ListEmployeeReachability(ctx context.Context, filters domain.EmployeeReachabilityFilters) ([]*domain.EmployeeReachability, error) {
	candidates, err := r.reachabilityCandidates(ctx, filters)
	if err != nil {
		return nil, err
	}

	rows := classifiedReachabilityRows(candidates, filters.State)
	rows = paginateReachabilityRows(rows, filters.Limit, filters.Offset)

	if err := r.attachReachabilityProjects(ctx, rows); err != nil {
		return nil, err
	}
	if err := r.attachReachabilityAmounts(ctx, rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// CountEmployeeReachability counts employees per state over the project-scoped
// cohort. The State filter is deliberately ignored: the counters exist so the
// caller can show the full worklist next to a filtered page.
func (r *EmployeeRepository) CountEmployeeReachability(ctx context.Context, filters domain.EmployeeReachabilityFilters) (*domain.EmployeeReachabilitySummary, error) {
	candidates, err := r.reachabilityCandidates(ctx, filters)
	if err != nil {
		return nil, err
	}

	summary := &domain.EmployeeReachabilitySummary{}
	for _, candidate := range candidates {
		states := classifyCandidate(candidate)
		if len(states) == 0 {
			continue
		}
		summary.Employees++
		for _, state := range states {
			switch state {
			case domain.EmployeeReachabilityStateNoPhone:
				summary.NoPhone++
			case domain.EmployeeReachabilityStateNotLinked:
				summary.NotLinked++
			case domain.EmployeeReachabilityStateDuplicateMobile:
				summary.DuplicateMobile++
			case domain.EmployeeReachabilityStateInvalidMobile:
				summary.InvalidMobile++
			}
		}
	}
	return summary, nil
}

// reachabilityCandidates loads every active employee with the raw signals the
// classifier needs, scoped by the project filter and ordered by employee id.
func (r *EmployeeRepository) reachabilityCandidates(ctx context.Context, filters domain.EmployeeReachabilityFilters) ([]reachabilityCandidate, error) {
	query := r.dbForContext(ctx).
		Table("employees").
		Select(`employees.id AS employee_id,
			employees.fullname AS fullname,
			employees.cccd AS cccd,
			COALESCE(employees.mobile, '') AS mobile,
			CASE WHEN z.status = ? AND z.provider_code = ? THEN 1 ELSE 0 END AS zns_no_linked,
			COALESCE(shared_mobile.mobile_matches, 0) AS mobile_matches`,
			domain.FlexPaySalaryNotificationSuppressed, domain.ZaloNoLinkedAccountProviderCode).
		Joins(reachabilityLatestZNSJoin).
		Joins(reachabilitySharedMobileJoin).
		Where("employees.deleted_at IS NULL")

	if filters.ProjectID != nil {
		// Current assignments only: started and not yet ended, mirroring the
		// "current projects" rule used by the employee list queries.
		query = query.Where(`EXISTS (
			SELECT 1 FROM project_employees pe
			WHERE pe.employee_id = employees.id
			  AND pe.project_id = ?
			  AND pe.deleted_at IS NULL
			  AND pe.last_date IS NULL
			  AND pe.start_date < ?
		)`, *filters.ProjectID, reachabilityStartDateCutoff())
	}

	var candidates []reachabilityCandidate
	err := query.Order("employees.id ASC").Find(&candidates).Error
	if err != nil {
		return nil, err
	}
	return candidates, nil
}

// reachabilityStartDateCutoff returns tomorrow's date in the business timezone.
// It is passed as a bare date string so the comparison against start_date has
// identical semantics on MySQL (DATETIME) and sqlite (ISO text): a bare date
// string compares as "before the start of that day", which keeps an assignment
// starting today included without depending on the server timezone.
func reachabilityStartDateCutoff() string {
	return clock.Now().AddDate(0, 0, 1).Format("2006-01-02")
}

func classifyCandidate(candidate reachabilityCandidate) []string {
	return domain.ClassifyReachabilityStates(domain.EmployeeReachabilitySignals{
		Mobile:                       candidate.Mobile,
		ZNSSuppressedNoLinkedAccount: candidate.ZNSNoLinked,
		MobileMatches:                candidate.MobileMatches,
	})
}

// classifiedReachabilityRows classifies candidates and keeps the ones in state
// (every state when state is empty), preserving the candidate order.
func classifiedReachabilityRows(candidates []reachabilityCandidate, state string) []*domain.EmployeeReachability {
	rows := make([]*domain.EmployeeReachability, 0, len(candidates))
	for _, candidate := range candidates {
		states := classifyCandidate(candidate)
		if len(states) == 0 {
			continue
		}
		if state != "" && !hasReachabilityState(states, state) {
			continue
		}
		rows = append(rows, &domain.EmployeeReachability{
			EmployeeID: candidate.EmployeeID,
			Fullname:   candidate.Fullname,
			CCCD:       candidate.CCCD,
			Mobile:     candidate.Mobile,
			Projects:   []string{},
			States:     states,
		})
	}
	return rows
}

func hasReachabilityState(states []string, state string) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

func paginateReachabilityRows(rows []*domain.EmployeeReachability, limit, offset int) []*domain.EmployeeReachability {
	if offset > 0 {
		if offset >= len(rows) {
			return []*domain.EmployeeReachability{}
		}
		rows = rows[offset:]
	}
	if limit > 0 && limit < len(rows) {
		rows = rows[:limit]
	}
	return rows
}

// attachReachabilityProjects fills in the current project names of the given
// rows. Employees without a current assignment keep an empty (non-nil) list.
func (r *EmployeeRepository) attachReachabilityProjects(ctx context.Context, rows []*domain.EmployeeReachability) error {
	if len(rows) == 0 {
		return nil
	}

	employeeIDs := make([]uint, len(rows))
	for i, row := range rows {
		employeeIDs[i] = row.EmployeeID
	}

	var assignments []reachabilityAssignmentRow
	err := r.dbForContext(ctx).
		Table("project_employees").
		Select("project_employees.employee_id AS employee_id, projects.name AS project_name").
		Joins("INNER JOIN projects ON projects.id = project_employees.project_id").
		Where("project_employees.deleted_at IS NULL").
		Where("projects.deleted_at IS NULL").
		Where("project_employees.employee_id IN ?", employeeIDs).
		Where("project_employees.last_date IS NULL").
		Where("project_employees.start_date < ?", reachabilityStartDateCutoff()).
		Order("project_employees.employee_id ASC, projects.name ASC").
		Scan(&assignments).Error
	if err != nil {
		return err
	}

	byEmployee := make(map[uint][]string, len(rows))
	for _, assignment := range assignments {
		names := byEmployee[assignment.EmployeeID]
		// The query is ordered by name, so a repeat can only be the previous
		// entry: one employee can hold two assignments on the same project.
		if len(names) > 0 && names[len(names)-1] == assignment.ProjectName {
			continue
		}
		byEmployee[assignment.EmployeeID] = append(names, assignment.ProjectName)
	}
	for _, row := range rows {
		if names, ok := byEmployee[row.EmployeeID]; ok {
			row.Projects = names
		}
	}
	return nil
}

// attachReachabilityAmounts fills in paid and outstanding salary per row in a
// single grouped query. Rows with no timesheets keep zero amounts.
func (r *EmployeeRepository) attachReachabilityAmounts(ctx context.Context, rows []*domain.EmployeeReachability) error {
	if len(rows) == 0 {
		return nil
	}

	employeeIDs := make([]uint, len(rows))
	for i, row := range rows {
		employeeIDs[i] = row.EmployeeID
	}

	// The canonical "still needs operational processing" cohort, reused so the
	// reported exposure matches the dashboard backlog rather than a local
	// redefinition of unpaid.
	outstanding := domain.NewOperationalPaymentBacklogTimesheetFilters()

	var amounts []reachabilityAmountRow
	err := r.dbForContext(ctx).
		Table("timesheets").
		Select(`timesheets.employee_id AS employee_id,
			COALESCE(SUM(CASE WHEN timesheets.payment_status = ? THEN timesheets.paid_amount ELSE 0 END), 0) AS paid_amount,
			COALESCE(SUM(CASE WHEN timesheets.timesheet_status IN ? AND timesheets.payment_status IN ? THEN timesheets.amount ELSE 0 END), 0) AS outstanding_amount`,
			domain.PaymentStatusPaid, outstanding.TimesheetStatus, outstanding.PaymentStatus).
		Where("timesheets.deleted_at IS NULL").
		Where("timesheets.employee_id IN ?", employeeIDs).
		Group("timesheets.employee_id").
		Scan(&amounts).Error
	if err != nil {
		return err
	}

	byEmployee := make(map[uint]reachabilityAmountRow, len(amounts))
	for _, amount := range amounts {
		byEmployee[amount.EmployeeID] = amount
	}
	for _, row := range rows {
		if amount, ok := byEmployee[row.EmployeeID]; ok {
			row.PaidAmount = amount.PaidAmount
			row.OutstandingAmount = amount.OutstandingAmount
		}
	}
	return nil
}
