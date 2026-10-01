package advance_payment

import (
	"context"
	"log/slog"

	"api-server/internal/domain"
)

// checkInPeriodIndex answers, per employee, "was this salary period funded by
// self check-in?" for one import run.
//
// Two independent facts make a period a check-in period, and either is enough:
//
//  1. the assignment is under self check-in and was already under it during
//     that period (domain.ProjectEmployee.IsCheckInPeriod reads the recorded
//     start day), or
//  2. the period's quota row already carries check-in earnings — the only fact
//     that survives the employee being switched off again, since disabling
//     never deletes earnings already banked.
//
// Answers are memoized per employee: a workbook carries one row per employee
// and the same person can appear on several sheets.
type checkInPeriodIndex struct {
	repo     domain.AdvancePaymentRepository
	logger   *slog.Logger
	forMonth string
	// hasEarnings caches fact (2). A lookup error is cached as "has earnings"
	// so a failing read can never turn a check-in period into a workbook period
	// and let a second source in.
	hasEarnings map[uint64]bool
}

// newCheckInPeriodIndex builds the index for one import. The per-employee
// earnings lookup is deferred to isCheckInPeriod, so a workbook that only
// touches non-check-in employees costs no extra queries.
func newCheckInPeriodIndex(ctx context.Context, repo domain.AdvancePaymentRepository, logger *slog.Logger, forMonth string) *checkInPeriodIndex {
	return &checkInPeriodIndex{
		repo:        repo,
		logger:      logger,
		forMonth:    forMonth,
		hasEarnings: make(map[uint64]bool),
	}
}

// isCheckInPeriod reports whether the workbook must leave this employee alone
// for this period.
func (i *checkInPeriodIndex) isCheckInPeriod(ctx context.Context, assignment *domain.ProjectEmployee, employeeID uint) bool {
	if assignment == nil {
		return false
	}
	// A period before the employee started self check-in is still a workbook
	// period, so an employee enabled later must not block an older upload.
	inPeriod, err := assignment.IsCheckInPeriod(i.forMonth)
	if err != nil {
		// An unparseable period cannot be reasoned about; refuse to add a second
		// source rather than guess.
		return true
	}
	if inPeriod {
		return true
	}

	if cached, ok := i.hasEarnings[uint64(employeeID)]; ok {
		return cached
	}
	rows, err := i.repo.GetByEmployeeAndMonth(ctx, uint64(employeeID), i.forMonth)
	if err != nil {
		i.logger.Warn("failed to read existing check-in earnings for period, treating the period as check-in",
			"employee_id", employeeID, "for_month", i.forMonth, "error", err)
		i.hasEarnings[uint64(employeeID)] = true
		return true
	}
	for _, row := range rows {
		if row.ProjectID == assignment.ProjectID && row.HasCheckInEarnings() {
			i.hasEarnings[uint64(employeeID)] = true
			return true
		}
	}
	i.hasEarnings[uint64(employeeID)] = false
	return false
}

// periodHasCheckInEarnings reports whether the period was funded from
// check-in/out earnings for the employee in any project. Such a period is
// served by the self-check-in flow only, so the same period never has two
// request paths.
func periodHasCheckInEarnings(rows []*domain.AdvancePayment) bool {
	for _, row := range rows {
		if row.HasCheckInEarnings() {
			return true
		}
	}
	return false
}

// periodConflicts returns the first quota row of a period funded by both
// pipelines. A mixed row must never be spendable: the request would draw
// against the sum of two independent sources.
func periodConflicts(rows []*domain.AdvancePayment) *domain.AdvancePayment {
	for _, row := range rows {
		if row.HasMixedSources() {
			return row
		}
	}
	return nil
}
