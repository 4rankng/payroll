package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// The stubs embed their interface so an unexpected call to any other method
// panics, keeping each test honest about which repository surface it touches.
type assignmentUpdateRepoStub struct {
	domain.ProjectEmployeeRepository
	byID        map[uint]*domain.ProjectEmployee
	overlapping []*domain.ProjectEmployee
}

func (s *assignmentUpdateRepoStub) GetByID(_ context.Context, id uint) (*domain.ProjectEmployee, error) {
	if a, ok := s.byID[id]; ok {
		return a, nil
	}
	return nil, domain.NewNotFoundError(constants.MsgAssignmentNotFoundVN)
}

func (s *assignmentUpdateRepoStub) GetOverlappingAssignments(_ context.Context, _ uint, _ *time.Time, _ *time.Time) ([]*domain.ProjectEmployee, error) {
	return s.overlapping, nil
}

type assignmentUpdateEmployeeStub struct {
	domain.EmployeeRepository
}

func (s *assignmentUpdateEmployeeStub) GetByID(_ context.Context, _ uint) (*domain.Employee, error) {
	return &domain.Employee{}, nil
}

type assignmentUpdateProjectStub struct {
	domain.ProjectRepository
}

func (s *assignmentUpdateProjectStub) GetByID(_ context.Context, _ uint) (*domain.Project, error) {
	return &domain.Project{}, nil
}

type assignmentUpdateCheckerStub struct {
	domain.TimesheetAssignmentChecker
	countTimesheets int64
	inRangeResult   bool
}

func (s *assignmentUpdateCheckerStub) CountTimesheets(_ context.Context, _, _ uint) (int64, error) {
	return s.countTimesheets, nil
}

func (s *assignmentUpdateCheckerStub) HasNonEditableTimesheetsInRange(_ context.Context, _, _ uint, _, _ *time.Time) (bool, error) {
	return s.inRangeResult, nil
}

func (s *assignmentUpdateCheckerStub) HasNonEditableTimesheetsAfterDate(_ context.Context, _, _ uint, _ time.Time) (bool, error) {
	panic("HasNonEditableTimesheetsAfterDate must not be called on the update path")
}

func parseDay(s string) time.Time {
	day, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return day
}

func parseDayPtr(s string) *time.Time {
	day := parseDay(s)
	return &day
}

func mkAssignment(id uint, start, last string) *domain.ProjectEmployee {
	a := &domain.ProjectEmployee{
		ID:         id,
		ProjectID:  19,
		EmployeeID: 966,
		StartDate:  parseDay(start),
	}
	if last != "" {
		a.LastDate = parseDayPtr(last)
	}
	return a
}

func newUpdateService(repo *assignmentUpdateRepoStub, checker domain.TimesheetAssignmentChecker) *EmployeeAssignmentService {
	return NewEmployeeAssignmentService(repo, &assignmentUpdateEmployeeStub{}, &assignmentUpdateProjectStub{}, checker)
}

func TestValidateAssignmentForUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		current     *domain.ProjectEmployee
		updated     *domain.ProjectEmployee
		checker     *assignmentUpdateCheckerStub
		overlapping []*domain.ProjectEmployee
		wantCode    string
		wantMsg     string
	}{
		{
			name:    "allows extending end date even with paid timesheets",
			current: mkAssignment(1, "2026-07-02", "2026-09-09"),
			updated: mkAssignment(1, "2026-07-02", "2026-10-31"),
			checker: &assignmentUpdateCheckerStub{},
		},
		{
			name:    "allows clearing end date to open-ended with paid timesheets",
			current: mkAssignment(1, "2026-07-02", "2026-09-09"),
			updated: mkAssignment(1, "2026-07-02", ""),
			checker: &assignmentUpdateCheckerStub{},
		},
		{
			name:     "blocks shrinking end date over paid timesheets",
			current:  mkAssignment(1, "2026-07-02", "2026-09-09"),
			updated:  mkAssignment(1, "2026-07-02", "2026-08-25"),
			checker:  &assignmentUpdateCheckerStub{inRangeResult: true},
			wantCode: "VALIDATION_ERROR",
			wantMsg:  constants.MsgAssignmentUpdateExcludesPaidTimesheetsVN,
		},
		{
			name:    "allows shrinking end date when dropped window is clean",
			current: mkAssignment(1, "2026-07-02", "2026-09-09"),
			updated: mkAssignment(1, "2026-07-02", "2026-08-25"),
			checker: &assignmentUpdateCheckerStub{inRangeResult: false},
		},
		{
			name:     "blocks moving start date later over paid timesheets",
			current:  mkAssignment(1, "2026-07-02", "2026-09-09"),
			updated:  mkAssignment(1, "2026-08-01", "2026-09-09"),
			checker:  &assignmentUpdateCheckerStub{inRangeResult: true},
			wantCode: "VALIDATION_ERROR",
			wantMsg:  constants.MsgAssignmentUpdateExcludesPaidTimesheetsVN,
		},
		{
			name:    "allows moving start date earlier",
			current: mkAssignment(1, "2026-07-02", "2026-09-09"),
			updated: mkAssignment(1, "2026-06-15", "2026-09-09"),
			checker: &assignmentUpdateCheckerStub{},
		},
		{
			name:     "blocks setting an end date on an open assignment over paid timesheets",
			current:  mkAssignment(1, "2026-07-02", ""),
			updated:  mkAssignment(1, "2026-07-02", "2026-08-31"),
			checker:  &assignmentUpdateCheckerStub{inRangeResult: true},
			wantCode: "VALIDATION_ERROR",
			wantMsg:  constants.MsgAssignmentUpdateExcludesPaidTimesheetsVN,
		},
		{
			name:        "preserves same-project overlap conflict",
			current:     mkAssignment(1, "2026-07-02", "2026-09-09"),
			updated:     mkAssignment(1, "2026-07-02", "2026-09-09"),
			checker:     &assignmentUpdateCheckerStub{countTimesheets: 3},
			overlapping: []*domain.ProjectEmployee{mkAssignment(2, "2026-08-01", "2026-08-31")},
			wantCode:    "CONFLICT",
			wantMsg:     constants.MsgAssignmentOverlapVN,
		},
		{
			name:    "ignores overlapping assignment on a different project",
			current: mkAssignment(1, "2026-07-02", "2026-09-09"),
			updated: mkAssignment(1, "2026-07-02", "2026-09-09"),
			checker: &assignmentUpdateCheckerStub{countTimesheets: 3},
			overlapping: []*domain.ProjectEmployee{
				{ID: 2, ProjectID: 68, EmployeeID: 966, StartDate: parseDay("2026-08-01")},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &assignmentUpdateRepoStub{
				byID:        map[uint]*domain.ProjectEmployee{1: tc.current},
				overlapping: tc.overlapping,
			}
			svc := newUpdateService(repo, tc.checker)

			err := svc.ValidateAssignmentForUpdate(context.Background(), tc.updated)

			if tc.wantCode == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			domainErr, ok := err.(*domain.DomainError)
			require.True(t, ok, "expected *domain.DomainError, got %T", err)
			require.Equal(t, tc.wantCode, domainErr.Code)
			require.True(t, strings.Contains(domainErr.Message, tc.wantMsg))
		})
	}
}

// ---------------------------------------------------------------------------
// Window bounds: the guard must query the checker ONLY for the boundary that
// actually shrinks, and with exact day-granular bounds. An off-by-one here is
// invisible to the table above (its stub ignores the dates), so these tests
// capture and assert the from/to arguments verbatim.
// ---------------------------------------------------------------------------

type rangeBoundsQuery struct{ from, to *time.Time }

func formatDay(p *time.Time) string {
	if p == nil {
		return "<nil>"
	}
	return p.Format("2006-01-02")
}

// boundsCapturingChecker embeds the table-test stub (so CountTimesheets and
// the AfterDate panic-guard keep working) and records every range query.
type boundsCapturingChecker struct {
	assignmentUpdateCheckerStub
	inRange bool
	err     error
	calls   []rangeBoundsQuery
}

func (s *boundsCapturingChecker) HasNonEditableTimesheetsInRange(_ context.Context, _, _ uint, from, to *time.Time) (bool, error) {
	s.calls = append(s.calls, rangeBoundsQuery{from: from, to: to})
	if s.err != nil {
		return false, s.err
	}
	return s.inRange, nil
}

func TestValidateAssignmentForUpdateWindowBounds(t *testing.T) {
	// Employee-966 shape: window 2026-07-02 → 2026-09-09, weekly pay.
	current := mkAssignment(1, "2026-07-02", "2026-09-09")

	run := func(updated *domain.ProjectEmployee, checker *boundsCapturingChecker) error {
		repo := &assignmentUpdateRepoStub{byID: map[uint]*domain.ProjectEmployee{1: current}}
		svc := newUpdateService(repo, checker)
		return svc.ValidateAssignmentForUpdate(context.Background(), updated)
	}

	t.Run("start moved later queries old start through day before new start", func(t *testing.T) {
		// Updated assignment clears the end (nil) so the end-side guard stays
		// silent and the start window is observed in isolation.
		checker := &boundsCapturingChecker{}
		require.NoError(t, run(mkAssignment(1, "2026-08-01", ""), checker))
		require.Len(t, checker.calls, 1, "exactly one range query expected")
		require.Equal(t, "2026-07-02", formatDay(checker.calls[0].from), "from must be the old start")
		require.Equal(t, "2026-07-31", formatDay(checker.calls[0].to), "to must be the day before the new start")
	})

	t.Run("start moved earlier queries nothing", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		require.NoError(t, run(mkAssignment(1, "2026-06-15", ""), checker))
		require.Empty(t, checker.calls)
	})

	t.Run("end cleared queries nothing", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		require.NoError(t, run(mkAssignment(1, "2026-07-02", ""), checker))
		require.Empty(t, checker.calls, "clearing the end date must not consult the paid-timesheet guard")
	})

	t.Run("end shrunk queries day after new end with unbounded upper", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		require.NoError(t, run(mkAssignment(1, "2026-07-02", "2026-08-31"), checker))
		require.Len(t, checker.calls, 1, "exactly one range query expected")
		require.Equal(t, "2026-09-01", formatDay(checker.calls[0].from), "from must be the day after the new end")
		require.Nil(t, checker.calls[0].to, "upper bound is unbounded: every paid timesheet after the new end is checked")
	})

	t.Run("end extended still verifies no paid work lies beyond it", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		require.NoError(t, run(mkAssignment(1, "2026-07-02", "2026-12-31"), checker))
		require.Len(t, checker.calls, 1, "a finite end always consults the guard, extension included")
		require.Equal(t, "2027-01-01", formatDay(checker.calls[0].from))
		require.Nil(t, checker.calls[0].to)
	})

	t.Run("finite end unchanged still verifies no paid work lies beyond it", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		require.NoError(t, run(mkAssignment(1, "2026-07-02", "2026-09-09"), checker))
		require.Len(t, checker.calls, 1)
		require.Equal(t, "2026-09-10", formatDay(checker.calls[0].from))
		require.Nil(t, checker.calls[0].to)
	})

	t.Run("insufficient extension leaving paid work beyond the new end is rejected", func(t *testing.T) {
		// Spec rule: no approved/paid timesheet may fall outside the resulting
		// window, so extending past the old end is only enough when nothing
		// paid sits beyond the new end either.
		checker := &boundsCapturingChecker{inRange: true}
		err := run(mkAssignment(1, "2026-07-02", "2026-09-30"), checker)
		require.Error(t, err)
		domainErr, ok := err.(*domain.DomainError)
		require.True(t, ok)
		require.Equal(t, "VALIDATION_ERROR", domainErr.Code)
	})

	t.Run("open-ended assignment shrunk to finite queries an unbounded upper", func(t *testing.T) {
		open := mkAssignment(1, "2026-07-02", "")
		checker := &boundsCapturingChecker{}
		repo := &assignmentUpdateRepoStub{byID: map[uint]*domain.ProjectEmployee{1: open}}
		svc := newUpdateService(repo, checker)
		require.NoError(t, svc.ValidateAssignmentForUpdate(context.Background(), mkAssignment(1, "2026-07-02", "2026-08-31")))
		require.Len(t, checker.calls, 1)
		require.Equal(t, "2026-09-01", formatDay(checker.calls[0].from))
		require.Nil(t, checker.calls[0].to, "old end is unbounded, so the upper bound must be nil")
	})

	t.Run("both boundaries shrink clean queries exactly two windows", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		require.NoError(t, run(mkAssignment(1, "2026-07-15", "2026-08-31"), checker))
		require.Len(t, checker.calls, 2)
		require.Equal(t, "2026-07-02", formatDay(checker.calls[0].from))
		require.Equal(t, "2026-07-14", formatDay(checker.calls[0].to))
		require.Equal(t, "2026-09-01", formatDay(checker.calls[1].from))
		require.Nil(t, checker.calls[1].to)
	})

	t.Run("checker error propagates", func(t *testing.T) {
		boom := errors.New("db down")
		checker := &boundsCapturingChecker{err: boom}
		err := run(mkAssignment(1, "2026-07-02", "2026-08-31"), checker)
		require.ErrorIs(t, err, boom)
	})
}

// ---------------------------------------------------------------------------
// Pure guard directly: overlap variants the impure table cannot express.
// ---------------------------------------------------------------------------

func TestValidateAssignmentUpdateWithDataOverlapVariants(t *testing.T) {
	svc := NewEmployeeAssignmentService(nil, nil, nil, nil)
	updated := mkAssignment(1, "2026-07-02", "2026-12-31")

	t.Run("skips the assignment itself", func(t *testing.T) {
		self := mkAssignment(1, "2026-07-02", "2026-12-31")
		require.NoError(t, svc.ValidateAssignmentUpdateWithData(updated, []*domain.ProjectEmployee{self}, map[uint]int64{1: 5}, false, false))
	})

	t.Run("allows same-project overlap when the other assignment has no timesheets", func(t *testing.T) {
		other := mkAssignment(2, "2026-08-01", "2026-08-31")
		require.NoError(t, svc.ValidateAssignmentUpdateWithData(updated, []*domain.ProjectEmployee{other}, map[uint]int64{2: 0}, false, false))
	})

	t.Run("rejects when only the start window is dirty", func(t *testing.T) {
		err := svc.ValidateAssignmentUpdateWithData(updated, nil, nil, true, false)
		require.Error(t, err)
		domainErr, ok := err.(*domain.DomainError)
		require.True(t, ok)
		require.Equal(t, "VALIDATION_ERROR", domainErr.Code)
	})
}

// ---------------------------------------------------------------------------
// Base validation preserved: early rejects never reach the paid-timesheet guard.
// ---------------------------------------------------------------------------

type notFoundEmployeeStub struct {
	domain.EmployeeRepository
}

func (notFoundEmployeeStub) GetByID(_ context.Context, _ uint) (*domain.Employee, error) {
	return nil, domain.NewNotFoundError(constants.MsgEmployeeNotFoundVN)
}

func TestValidateAssignmentForUpdateBaseValidation(t *testing.T) {
	t.Run("last date before start date rejected before any range query", func(t *testing.T) {
		checker := &boundsCapturingChecker{}
		repo := &assignmentUpdateRepoStub{byID: map[uint]*domain.ProjectEmployee{1: mkAssignment(1, "2026-07-02", "2026-09-09")}}
		svc := newUpdateService(repo, checker)
		err := svc.ValidateAssignmentForUpdate(context.Background(), mkAssignment(1, "2026-07-02", "2026-07-01"))
		require.Error(t, err)
		domainErr, ok := err.(*domain.DomainError)
		require.True(t, ok)
		require.Equal(t, "VALIDATION_ERROR", domainErr.Code)
		require.Empty(t, checker.calls, "dates reject must fire before the paid-timesheet guard")
	})

	t.Run("missing employee surfaces as not found", func(t *testing.T) {
		svc := NewEmployeeAssignmentService(
			&assignmentUpdateRepoStub{},
			notFoundEmployeeStub{},
			&assignmentUpdateProjectStub{},
			&assignmentUpdateCheckerStub{},
		)
		err := svc.ValidateAssignmentForUpdate(context.Background(), mkAssignment(1, "2026-07-02", ""))
		require.Error(t, err)
		domainErr, ok := err.(*domain.DomainError)
		require.True(t, ok)
		require.Equal(t, "NOT_FOUND", domainErr.Code)
	})
}

// The rule: last recorded timesheet + 1 day when the employee has history,
// else the 1st of the current month — never "today", which would reject
// same-month BCC uploads covering earlier days (2026-09-17 CBS incident).
func TestSuggestAssignmentStart(t *testing.T) {
	now := time.Date(2026, 9, 17, 15, 0, 0, 0, time.UTC)

	cases := []struct {
		name   string
		latest *time.Time
		want   time.Time
	}{
		{"no history falls back to month start", nil, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
		{"mid-month history continues next day", parseDayPtr("2026-09-10"), time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)},
		{"history equals today starts tomorrow", parseDayPtr("2026-09-17"), time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)},
		{"previous-month history still continues next day", parseDayPtr("2026-08-31"), time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SuggestAssignmentStart(tc.latest, now)
			if !got.Equal(tc.want) {
				t.Fatalf("SuggestAssignmentStart(%v, %v) = %v, want %v", tc.latest, now, got, tc.want)
			}
		})
	}
}
