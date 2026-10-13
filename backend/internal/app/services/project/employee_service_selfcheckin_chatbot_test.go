package project

import (
	"context"
	"fmt"
	"testing"
	"time"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// The chatbot self check-in matrix (SetSelfCheckinViaChatbot). Enables follow
// the day-9 rule and reuse the admin enable path (payrate precheck, upload
// quota conflict); disables are always deferred to the 1st of the next month
// and never zero quota at queue time — the sweep owns that, using the
// effective month.

// pendingDisableOn seeds an enabled row with a queued disable.
func pendingDisableOn(effective time.Time) func(*domain.ProjectEmployee) {
	return func(a *domain.ProjectEmployee) {
		a.CheckInEnabled = true
		disabled := false
		a.PendingCheckInEnabled = &disabled
		a.CheckInEffectiveFrom = &effective
	}
}

func TestSetSelfCheckinEnableDayBoundary(t *testing.T) {
	tests := []struct {
		name          string
		day           int
		wantImmediate bool
		wantEffective time.Time
		wantEnabled   bool
		wantStartDate *time.Time
	}{
		{
			name:          "day 5 activates immediately with the 1st as start date",
			day:           5,
			wantImmediate: true,
			wantEffective: time.Date(2026, 10, 1, 0, 0, 0, 0, clock.DefaultLocation),
			wantEnabled:   true,
			wantStartDate: &[]time.Time{time.Date(2026, 10, 1, 0, 0, 0, 0, clock.DefaultLocation)}[0],
		},
		{
			name:          "day 15 queues for the 1st of the next month",
			day:           15,
			wantImmediate: false,
			wantEffective: time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation),
			wantEnabled:   false,
			wantStartDate: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFakeClock(t, time.Date(2026, 10, tt.day, 10, 0, 0, 0, clock.DefaultLocation))
			repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
			svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

			result, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, true, 42)
			if err != nil {
				t.Fatalf("enable on day %d: %v", tt.day, err)
			}

			if result.Kind != "enable" {
				t.Errorf("Kind = %q, want enable", result.Kind)
			}
			if result.Immediate != tt.wantImmediate {
				t.Errorf("Immediate = %v, want %v", result.Immediate, tt.wantImmediate)
			}
			if !result.EffectiveFrom.Equal(tt.wantEffective) {
				t.Errorf("EffectiveFrom = %v, want %v", result.EffectiveFrom, tt.wantEffective)
			}
			a := repo.assignment
			if a.CheckInEnabled != tt.wantEnabled {
				t.Errorf("CheckInEnabled = %v, want %v", a.CheckInEnabled, tt.wantEnabled)
			}
			switch {
			case tt.wantStartDate == nil && a.CheckInStartDate != nil:
				t.Errorf("CheckInStartDate = %v, want nil until activation", a.CheckInStartDate)
			case tt.wantStartDate != nil && (a.CheckInStartDate == nil || !a.CheckInStartDate.Equal(*tt.wantStartDate)):
				t.Errorf("CheckInStartDate = %v, want %v (1st of the start month)", a.CheckInStartDate, tt.wantStartDate)
			}
			if len(repo.saved) != 1 {
				t.Errorf("saved = %d rows, want 1", len(repo.saved))
			}
		})
	}
}

func TestSetSelfCheckinEnableRefusals(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(*domain.ProjectEmployee)
		wantMsg string
		// stillHolds must be true after the refusal — the pin that the refused
		// request left the row untouched.
		stillHolds func(*domain.ProjectEmployee) bool
	}{
		{
			name:       "already enabled",
			seed:       func(a *domain.ProjectEmployee) { a.CheckInEnabled = true },
			wantMsg:    constants.MsgSelfCheckinAlreadyEnabledVN,
			stillHolds: func(a *domain.ProjectEmployee) bool { return a.CheckInEnabled },
		},
		{
			name: "pending enable already queued",
			seed: func(a *domain.ProjectEmployee) {
				if err := a.RequestCheckInEnable(time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation)); err != nil {
					t.Fatalf("seed: %v", err)
				}
			},
			wantMsg:    constants.MsgSelfCheckinPendingExistsVN,
			stillHolds: func(a *domain.ProjectEmployee) bool { return a.HasPendingCheckInEnable() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFakeClock(t, time.Date(2026, 10, 5, 10, 0, 0, 0, clock.DefaultLocation))
			repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
			tt.seed(repo.assignment)
			svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

			result, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, true, 42)

			if err == nil {
				t.Fatal("refusal expected")
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantMsg)
			}
			if result != nil {
				t.Errorf("result = %+v, want nil on refusal", result)
			}
			if !tt.stillHolds(repo.assignment) {
				t.Error("refused request must not change the row")
			}
			if len(repo.saved) != 0 {
				t.Errorf("refused request must not persist, saved = %d", len(repo.saved))
			}
		})
	}
}

// An enable arriving while a disable is queued supersedes it: the queued
// disable is cancelled and the running service simply continues from its
// original start date — on any day of the month, because there is nothing
// left to schedule. The verdict carries CancelledPendingDisable so the bot
// confirms "vẫn bật" instead of claiming a fresh activation.
func TestSetSelfCheckinEnableSupersedesQueuedDisable(t *testing.T) {
	queuedFor := time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation)
	runningSince := time.Date(2026, 9, 1, 0, 0, 0, 0, clock.DefaultLocation)

	t.Run("day 5 keeps the service running from its original start", func(t *testing.T) {
		setFakeClock(t, time.Date(2026, 10, 5, 10, 0, 0, 0, clock.DefaultLocation))
		repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
		pendingDisableOn(queuedFor)(repo.assignment)
		repo.assignment.CheckInStartDate = &runningSince
		svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

		result, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, true, 42)
		if err != nil {
			t.Fatalf("enable over queued disable: %v", err)
		}

		a := repo.assignment
		if !a.CheckInEnabled {
			t.Error("the service must stay on after the supersede")
		}
		if a.HasPendingCheckInChange() {
			t.Error("the queued disable must be cancelled")
		}
		if a.CheckInStartDate == nil || !a.CheckInStartDate.Equal(runningSince) {
			t.Errorf("CheckInStartDate = %v, want the original %v", a.CheckInStartDate, runningSince)
		}
		if result == nil || !result.Immediate || result.Kind != "enable" || !result.EffectiveFrom.Equal(runningSince) {
			t.Errorf("result = %+v, want enable effective from the original start %v", result, runningSince)
		}
		if !result.CancelledPendingDisable {
			t.Error("CancelledPendingDisable = false, want true so the bot can confirm the service stayed on")
		}
		if len(repo.saved) != 1 {
			t.Errorf("saved = %d rows, want 1", len(repo.saved))
		}
	})

	t.Run("day 15 behaves the same — nothing needs scheduling", func(t *testing.T) {
		setFakeClock(t, time.Date(2026, 10, 15, 10, 0, 0, 0, clock.DefaultLocation))
		repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
		pendingDisableOn(queuedFor)(repo.assignment)
		repo.assignment.CheckInStartDate = &runningSince
		svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

		result, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, true, 42)
		if err != nil {
			t.Fatalf("enable over queued disable: %v", err)
		}
		if result == nil || !result.Immediate || !result.EffectiveFrom.Equal(runningSince) || !result.CancelledPendingDisable {
			t.Errorf("result = %+v, want immediate enable from %v with the queued disable cancelled", result, runningSince)
		}
		if repo.assignment.HasPendingCheckInEnable() {
			t.Error("the day-9 rule must not queue a new enable for a service that is already on")
		}
	})

	t.Run("a legacy row without a recorded start still answers", func(t *testing.T) {
		// Rows enabled before CheckInStartDate existed carry no start day; the
		// verdict must not dereference it and quotes the computed day-1.
		setFakeClock(t, time.Date(2026, 10, 5, 10, 0, 0, 0, clock.DefaultLocation))
		repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
		pendingDisableOn(queuedFor)(repo.assignment)
		svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

		result, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, true, 42)
		if err != nil {
			t.Fatalf("enable over queued disable on a legacy row: %v", err)
		}
		if result == nil || !result.Immediate || !result.EffectiveFrom.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, clock.DefaultLocation)) {
			t.Errorf("result = %+v, want immediate enable quoting the current month's day 1", result)
		}
		if !result.CancelledPendingDisable {
			t.Error("CancelledPendingDisable = false, want true")
		}
	})
}

// The supersede on a running row keeps the service on with its original start
// (pinned above at the public level). This pins the remaining branch shape at
// the helper level: an INACTIVE row carrying a queued disable — a state the
// service cannot produce today (a queued disable only exists on enabled rows)
// but tolerates — gets the disable cancelled and a real enable queued or
// activated per the day-9 rule.
func TestApplySelfCheckinChatbotChangeSupersedesQueuedDisableOnInactiveRow(t *testing.T) {
	queuedFor := time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation)

	t.Run("day 5 activates immediately", func(t *testing.T) {
		setFakeClock(t, time.Date(2026, 10, 5, 10, 0, 0, 0, clock.DefaultLocation))
		assignment := newPendingAssignment()
		pendingDisableOn(queuedFor)(assignment)
		assignment.CheckInEnabled = false // anomaly state, unreachable via the service today
		svc := newCheckinPendingService(&checkinPendingAssignmentRepo{}, &checkinPendingAdvanceRepo{})

		result, err := svc.applySelfCheckinChatbotChange(context.Background(), assignment, true)
		if err != nil {
			t.Fatalf("enable: %v", err)
		}

		if !assignment.CheckInEnabled {
			t.Error("the superseded enable must activate on a day 1-8 request")
		}
		if assignment.HasPendingCheckInChange() {
			t.Error("the queued disable must be cancelled by the supersede branch")
		}
		if assignment.CheckInStartDate == nil || !assignment.CheckInStartDate.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, clock.DefaultLocation)) {
			t.Errorf("CheckInStartDate = %v, want 2026-10-01", assignment.CheckInStartDate)
		}
		if result == nil || !result.Immediate || result.EffectiveFrom != (time.Date(2026, 10, 1, 0, 0, 0, 0, clock.DefaultLocation)) {
			t.Errorf("result = %+v, want immediate enable effective 2026-10-01", result)
		}
		if result == nil || !result.CancelledPendingDisable {
			t.Error("CancelledPendingDisable = false, want true")
		}
	})

	t.Run("day 15 queues for next month", func(t *testing.T) {
		setFakeClock(t, time.Date(2026, 10, 15, 10, 0, 0, 0, clock.DefaultLocation))
		assignment := newPendingAssignment()
		pendingDisableOn(queuedFor)(assignment)
		assignment.CheckInEnabled = false // anomaly state, unreachable via the service today
		svc := newCheckinPendingService(&checkinPendingAssignmentRepo{}, &checkinPendingAdvanceRepo{})

		result, err := svc.applySelfCheckinChatbotChange(context.Background(), assignment, true)
		if err != nil {
			t.Fatalf("enable: %v", err)
		}

		if assignment.CheckInEnabled || !assignment.HasPendingCheckInEnable() {
			t.Error("a day 9+ enable must queue, not activate")
		}
		if result == nil || result.Immediate || !result.EffectiveFrom.Equal(queuedFor) {
			t.Errorf("result = %+v, want queued enable effective %v", result, queuedFor)
		}
		if result == nil || !result.CancelledPendingDisable {
			t.Error("CancelledPendingDisable = false, want true")
		}
	})
}

// The upload-quota conflict is checked before the enable path touches a queued
// disable, so a conflicted enable must leave the disable queued (the same
// unreachable-via-public-path anomaly state as above, pinned at the helper).
func TestApplySelfCheckinChatbotChangeQuotaConflictKeepsQueuedDisable(t *testing.T) {
	setFakeClock(t, time.Date(2026, 10, 5, 10, 0, 0, 0, clock.DefaultLocation))
	stamped := uint(7)
	advanceRepo := &checkinPendingAdvanceRepo{rows: []*domain.AdvancePayment{
		{ProjectID: 5, EmployeeID: 99, LastAppliedAssetID: &stamped},
	}}
	assignment := newPendingAssignment()
	pendingDisableOn(time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation))(assignment)
	assignment.CheckInEnabled = false // anomaly state, unreachable via the service today
	svc := newCheckinPendingService(&checkinPendingAssignmentRepo{}, advanceRepo)

	result, err := svc.applySelfCheckinChatbotChange(context.Background(), assignment, true)

	if err == nil {
		t.Fatalf("upload conflict expected, got %+v", result)
	}
	if !assignment.HasPendingCheckInDisable() {
		t.Error("the quota conflict must not consume the queued disable")
	}
}

func TestSetSelfCheckinEnableUploadQuotaConflict(t *testing.T) {
	uploadRow := func(projectID uint, lastApplied uint) *domain.AdvancePayment {
		return &domain.AdvancePayment{ProjectID: projectID, EmployeeID: 99, LastAppliedAssetID: &lastApplied}
	}
	wantConflict := func(month string) string {
		return fmt.Sprintf(constants.MsgCheckInEnableUploadConflictVN, clock.FormatMonthDisplay(month))
	}

	tests := []struct {
		name      string
		day       int
		rows      []*domain.AdvancePayment
		wantError string
	}{
		{
			name:      "upload-funded start month is refused",
			day:       5,
			rows:      []*domain.AdvancePayment{uploadRow(5, 7)},
			wantError: wantConflict("2026-10"),
		},
		{
			name:      "the checked period is the START month, not the current one",
			day:       15,
			rows:      []*domain.AdvancePayment{uploadRow(5, 7)},
			wantError: wantConflict("2026-11"),
		},
		{
			name:      "another project's upload quota does not block",
			day:       5,
			rows:      []*domain.AdvancePayment{uploadRow(6, 7)},
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFakeClock(t, time.Date(2026, 10, tt.day, 10, 0, 0, 0, clock.DefaultLocation))
			advanceRepo := &checkinPendingAdvanceRepo{rows: tt.rows}
			repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
			svc := newCheckinPendingService(repo, advanceRepo)

			result, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, true, 42)

			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("upload conflict expected, got result %+v", result)
				}
				if err.Error() != tt.wantError {
					t.Errorf("error = %q, want %q", err.Error(), tt.wantError)
				}
				if len(repo.saved) != 0 {
					t.Errorf("conflicted request must not persist, saved = %d", len(repo.saved))
				}
				return
			}
			if err != nil {
				t.Fatalf("enable should pass: %v", err)
			}
			if result == nil {
				t.Fatal("expected a result")
			}
		})
	}

	t.Run("check-in earnings without an upload stamp do not block", func(t *testing.T) {
		setFakeClock(t, time.Date(2026, 10, 5, 10, 0, 0, 0, clock.DefaultLocation))
		advanceRepo := &checkinPendingAdvanceRepo{rows: []*domain.AdvancePayment{
			{ProjectID: 5, EmployeeID: 99, Salary: 500_000},
		}}
		repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
		svc := newCheckinPendingService(repo, advanceRepo)

		result, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, true, 42)
		if err != nil {
			t.Fatalf("earnings-only quota must not conflict: %v", err)
		}
		if result == nil || !result.Immediate {
			t.Errorf("result = %+v, want immediate enable", result)
		}
	})
}

func TestSetSelfCheckinDisableMatrix(t *testing.T) {
	queuedFor := time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation)

	tests := []struct {
		name string
		seed func(t *testing.T, a *domain.ProjectEmployee)
		// wantErrMsg pins the Vietnamese verdict the bot relays; empty means
		// the call must succeed.
		wantErrMsg string
		// check receives the result, the assignment, and the advance repo.
		check func(t *testing.T, result *SelfCheckinChatbotResult, a *domain.ProjectEmployee, advanceRepo *checkinPendingAdvanceRepo, saved int)
	}{
		{
			name: "pending enable is cancelled, service stays off",
			seed: func(t *testing.T, a *domain.ProjectEmployee) {
				if err := a.RequestCheckInEnable(queuedFor); err != nil {
					t.Fatalf("seed: %v", err)
				}
			},
			check: func(t *testing.T, result *SelfCheckinChatbotResult, a *domain.ProjectEmployee, advanceRepo *checkinPendingAdvanceRepo, saved int) {
				if !result.CancelledPendingEnable {
					t.Errorf("CancelledPendingEnable = %v, want true", result.CancelledPendingEnable)
				}
				if result.Kind != "disable" {
					t.Errorf("Kind = %q, want disable", result.Kind)
				}
				if !result.EffectiveFrom.IsZero() {
					t.Errorf("EffectiveFrom = %v, want zero for a plain cancellation", result.EffectiveFrom)
				}
				if a.CheckInEnabled || a.HasPendingCheckInChange() {
					t.Errorf("row must end up off with nothing pending, got enabled=%v pending=%v", a.CheckInEnabled, a.HasPendingCheckInChange())
				}
				if len(advanceRepo.zeroedMonths) != 0 {
					t.Errorf("cancelling a never-active enable must not zero quota, got %v", advanceRepo.zeroedMonths)
				}
				if saved != 1 {
					t.Errorf("saved = %d rows, want 1", saved)
				}
			},
		},
		{
			name:       "already disabled",
			seed:       func(t *testing.T, a *domain.ProjectEmployee) {},
			wantErrMsg: constants.MsgSelfCheckinAlreadyDisabledVN,
			check: func(t *testing.T, result *SelfCheckinChatbotResult, a *domain.ProjectEmployee, _ *checkinPendingAdvanceRepo, saved int) {
				if result != nil {
					t.Errorf("result = %+v, want nil on refusal", result)
				}
				if saved != 0 {
					t.Errorf("refused disable must not persist, saved = %d", saved)
				}
			},
		},
		{
			name: "pending disable repeats the same verdict",
			seed: func(t *testing.T, a *domain.ProjectEmployee) { pendingDisableOn(queuedFor)(a) },
			check: func(t *testing.T, result *SelfCheckinChatbotResult, a *domain.ProjectEmployee, advanceRepo *checkinPendingAdvanceRepo, saved int) {
				if result.CancelledPendingEnable {
					t.Error("an active service with a queued disable is not a cancelled enable")
				}
				if result.Kind != "disable" || !result.EffectiveFrom.Equal(queuedFor) {
					t.Errorf("result = %+v, want disable effective %v (idempotent re-verdict)", result, queuedFor)
				}
				if a.CheckInEffectiveFrom == nil || !a.CheckInEffectiveFrom.Equal(queuedFor) {
					t.Errorf("queued date = %v, want unchanged %v", a.CheckInEffectiveFrom, queuedFor)
				}
				if !a.CheckInEnabled {
					t.Error("the service keeps running until the queued date arrives")
				}
				if len(advanceRepo.zeroedMonths) != 0 {
					t.Errorf("queueing must not zero quota, got %v", advanceRepo.zeroedMonths)
				}
				if saved != 1 {
					t.Errorf("saved = %d rows, want 1", saved)
				}
			},
		},
		{
			name: "enabled row queues for the 1st of the next month",
			seed: func(t *testing.T, a *domain.ProjectEmployee) { a.CheckInEnabled = true },
			check: func(t *testing.T, result *SelfCheckinChatbotResult, a *domain.ProjectEmployee, advanceRepo *checkinPendingAdvanceRepo, saved int) {
				if result.Kind != "disable" || !result.EffectiveFrom.Equal(queuedFor) {
					t.Errorf("result = %+v, want disable effective %v", result, queuedFor)
				}
				if !a.CheckInEnabled {
					t.Error("disable is deferred — the row must stay enabled until the sweep")
				}
				if !a.HasPendingCheckInDisable() {
					t.Error("a pending disable must be queued")
				}
				if a.CheckInEffectiveFrom == nil || !a.CheckInEffectiveFrom.Equal(queuedFor) {
					t.Errorf("queued date = %v, want %v", a.CheckInEffectiveFrom, queuedFor)
				}
				if len(advanceRepo.zeroedMonths) != 0 {
					t.Errorf("quota zeroing belongs to the sweep, not the queueing, got %v", advanceRepo.zeroedMonths)
				}
				if saved != 1 {
					t.Errorf("saved = %d rows, want 1", saved)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFakeClock(t, time.Date(2026, 10, 20, 10, 0, 0, 0, clock.DefaultLocation))
			advanceRepo := &checkinPendingAdvanceRepo{}
			repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
			tt.seed(t, repo.assignment)
			svc := newCheckinPendingService(repo, advanceRepo)

			result, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, false, 42)
			if tt.wantErrMsg != "" {
				if err == nil || err.Error() != tt.wantErrMsg {
					t.Fatalf("error = %v, want %q", err, tt.wantErrMsg)
				}
			} else if err != nil {
				t.Fatalf("disable: %v", err)
			}

			tt.check(t, result, repo.assignment, advanceRepo, len(repo.saved))
		})
	}
}

func TestSetSelfCheckinPayratePrecheck(t *testing.T) {
	setFakeClock(t, time.Date(2026, 10, 20, 10, 0, 0, 0, clock.DefaultLocation))
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	enabled := newPendingAssignment()
	enabled.CheckInEnabled = true
	svc := NewProjectEmployeeService(
		&checkinPendingAssignmentRepo{assignment: enabled},
		nil, nil, nil, nil, nil,
		infrastructure.NewTransactionManager(db),
		&checkinPendingAdvanceRepo{},
		&checkinPendingPayrateRepo{hasRates: false},
		nil, nil, nil, nil, nil,
	)

	if _, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, true, 42); err == nil {
		t.Fatal("enable without project payrate must be rejected")
	} else if !domain.IsValidationError(err) {
		t.Errorf("error = %v, want a validation error mentioning the missing payrate", err)
	}

	// The precheck guards the enable direction only: a disable on the same
	// unpriced project still goes through (quota zeroing is the sweep's job).
	if _, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, false, 42); err != nil {
		t.Fatalf("disable must skip the payrate precheck: %v", err)
	}
	if !enabled.HasPendingCheckInDisable() {
		t.Error("disable on an unpriced project must still queue the pending disable")
	}
}

// An admin disable landing on a row that already has a queued disable is an
// idempotent no-op: nothing persisted, nothing zeroed, the queue untouched.
func TestAdminDisableWhilePendingDisableQueuedIsNoop(t *testing.T) {
	setFakeClock(t, time.Date(2026, 10, 20, 10, 0, 0, 0, clock.DefaultLocation))
	advanceRepo := &checkinPendingAdvanceRepo{}
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	pendingDisableOn(time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation))(repo.assignment)
	svc := newCheckinPendingService(repo, advanceRepo)

	if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, false, domain.CheckInStartMonthNextMonth, 1); err != nil {
		t.Fatalf("admin disable over a queued disable: %v", err)
	}

	a := repo.assignment
	if !a.CheckInEnabled {
		t.Error("the admin no-op must not turn the service off early")
	}
	if !a.HasPendingCheckInDisable() {
		t.Error("the queued disable must survive the admin no-op")
	}
	if len(repo.saved) != 0 {
		t.Errorf("no-op must not persist, saved = %d", len(repo.saved))
	}
	if len(advanceRepo.zeroedMonths) != 0 {
		t.Errorf("no-op must not zero quota, got %v", advanceRepo.zeroedMonths)
	}
}

// The bulk toggle follows the same rule and must not enroll the row in the
// batch quota zero-out.
func TestBulkAdminDisableWhilePendingDisableQueuedIsNoop(t *testing.T) {
	setFakeClock(t, time.Date(2026, 10, 20, 10, 0, 0, 0, clock.DefaultLocation))
	advanceRepo := &checkinPendingAdvanceRepo{}
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	pendingDisableOn(time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation))(repo.assignment)
	svc := newCheckinPendingService(repo, advanceRepo)

	if err := svc.BulkToggleCheckInEnabled(context.Background(), 5, []uint{99}, false, domain.CheckInStartMonthNextMonth, 1); err != nil {
		t.Fatalf("bulk disable over a queued disable: %v", err)
	}

	if len(repo.saved) != 0 {
		t.Errorf("bulk no-op must not persist, saved = %d", len(repo.saved))
	}
	if len(advanceRepo.zeroedMonths) != 0 {
		t.Errorf("bulk no-op must not enroll the row in BatchZeroOutQuota, got %v", advanceRepo.zeroedMonths)
	}
}

// An admin enable on a row with a queued disable supersedes it, mirroring the
// chatbot row: the queued disable is cancelled, the service stays on from its
// original start date, and the admin's start-month choice is irrelevant
// because nothing needs scheduling. No quota is zeroed either way.
func TestAdminEnableOverQueuedDisableSupersedes(t *testing.T) {
	queuedFor := time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation)
	runningSince := time.Date(2026, 9, 1, 0, 0, 0, 0, clock.DefaultLocation)

	t.Run("single toggle", func(t *testing.T) {
		setFakeClock(t, time.Date(2026, 10, 20, 10, 0, 0, 0, clock.DefaultLocation))
		advanceRepo := &checkinPendingAdvanceRepo{}
		repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
		pendingDisableOn(queuedFor)(repo.assignment)
		repo.assignment.CheckInStartDate = &runningSince
		svc := newCheckinPendingService(repo, advanceRepo)

		if err := svc.ToggleCheckInEnabled(context.Background(), 5, 99, true, domain.CheckInStartMonthNextMonth, 1); err != nil {
			t.Fatalf("admin enable over a queued disable: %v", err)
		}

		a := repo.assignment
		if !a.CheckInEnabled {
			t.Error("the service must stay on after the supersede")
		}
		if a.HasPendingCheckInChange() {
			t.Error("the queued disable must be cancelled")
		}
		if a.CheckInStartDate == nil || !a.CheckInStartDate.Equal(runningSince) {
			t.Errorf("CheckInStartDate = %v, want the original %v (never rewritten)", a.CheckInStartDate, runningSince)
		}
		if len(repo.saved) != 1 {
			t.Errorf("the cancellation must persist, saved = %d", len(repo.saved))
		}
		if len(advanceRepo.zeroedMonths) != 0 {
			t.Errorf("superseding must not zero quota, got %v", advanceRepo.zeroedMonths)
		}
	})

	t.Run("bulk toggle", func(t *testing.T) {
		setFakeClock(t, time.Date(2026, 10, 20, 10, 0, 0, 0, clock.DefaultLocation))
		advanceRepo := &checkinPendingAdvanceRepo{}
		repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
		pendingDisableOn(queuedFor)(repo.assignment)
		repo.assignment.CheckInStartDate = &runningSince
		svc := newCheckinPendingService(repo, advanceRepo)

		if err := svc.BulkToggleCheckInEnabled(context.Background(), 5, []uint{99}, true, domain.CheckInStartMonthNextMonth, 1); err != nil {
			t.Fatalf("bulk enable over a queued disable: %v", err)
		}

		a := repo.assignment
		if !a.CheckInEnabled || a.HasPendingCheckInChange() {
			t.Errorf("row must stay enabled with the queued disable cancelled, got enabled=%v pending=%v", a.CheckInEnabled, a.HasPendingCheckInChange())
		}
		if a.CheckInStartDate == nil || !a.CheckInStartDate.Equal(runningSince) {
			t.Errorf("CheckInStartDate = %v, want the original %v", a.CheckInStartDate, runningSince)
		}
		if len(repo.saved) != 1 {
			t.Errorf("the cancellation must persist, saved = %d", len(repo.saved))
		}
		if len(advanceRepo.zeroedMonths) != 0 {
			t.Errorf("superseding must not enroll the row in BatchZeroOutQuota, got %v", advanceRepo.zeroedMonths)
		}
	})
}

// The admin delete route (DELETE .../checkin-enabled → CancelPendingCheckInEnable)
// cancels a queued disable as well as a queued enable — intended per the plan
// — and leaves the service running.
func TestCancelPendingCheckInEnableRouteCancelsQueuedDisable(t *testing.T) {
	setFakeClock(t, time.Date(2026, 10, 20, 10, 0, 0, 0, clock.DefaultLocation))
	repo := &checkinPendingAssignmentRepo{assignment: newPendingAssignment()}
	pendingDisableOn(time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation))(repo.assignment)
	svc := newCheckinPendingService(repo, &checkinPendingAdvanceRepo{})

	if err := svc.CancelPendingCheckInEnable(context.Background(), 1, 1); err != nil {
		t.Fatalf("cancel queued disable: %v", err)
	}

	a := repo.assignment
	if a.HasPendingCheckInChange() {
		t.Error("the queued disable must be cleared")
	}
	if !a.CheckInEnabled {
		t.Error("cancelling the queued disable leaves the service running")
	}
	if len(repo.saved) != 1 {
		t.Errorf("saved = %d rows, want 1", len(repo.saved))
	}

	if err := svc.CancelPendingCheckInEnable(context.Background(), 1, 1); err == nil {
		t.Error("cancelling with nothing pending must fail")
	}
}

// Owner rule 2026-10-08: disable follows the same day-9 boundary as enable.
// Days 1-8 apply immediately (service off now, start date kept, quota zeroed
// from the current month); day 9+ queue for the 1st of next month.
func TestSetSelfCheckinDisableFollowsTheDay9Rule(t *testing.T) {
	tests := []struct {
		name          string
		day           int
		wantImmediate bool
		wantEffective time.Time
	}{
		{
			name:          "day 5 turns the service off now",
			day:           5,
			wantImmediate: true,
			wantEffective: time.Date(2026, 10, 1, 0, 0, 0, 0, clock.DefaultLocation),
		},
		{
			name:          "day 15 queues for the 1st of the next month",
			day:           15,
			wantImmediate: false,
			wantEffective: time.Date(2026, 11, 1, 0, 0, 0, 0, clock.DefaultLocation),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFakeClock(t, time.Date(2026, 10, tt.day, 10, 0, 0, 0, clock.DefaultLocation))
			a := newPendingAssignment()
			a.CheckInEnabled = true
			sept1 := time.Date(2026, 9, 1, 0, 0, 0, 0, clock.DefaultLocation)
			a.CheckInStartDate = &sept1
			adv := &checkinPendingAdvanceRepo{}
			repo := &checkinPendingAssignmentRepo{assignment: a}
			svc := newCheckinPendingService(repo, adv)

			result, err := svc.SetSelfCheckinViaChatbot(context.Background(), 5, 99, false, 42)
			if err != nil {
				t.Fatalf("disable on day %d: %v", tt.day, err)
			}
			if result.Kind != "disable" {
				t.Errorf("Kind = %q, want disable", result.Kind)
			}
			if result.Immediate != tt.wantImmediate {
				t.Errorf("Immediate = %v, want %v", result.Immediate, tt.wantImmediate)
			}
			if !result.EffectiveFrom.Equal(tt.wantEffective) {
				t.Errorf("EffectiveFrom = %v, want %v", result.EffectiveFrom, tt.wantEffective)
			}
			if tt.wantImmediate && a.CheckInEnabled {
				t.Errorf("immediate disable must leave the service off, got enabled=%v", a.CheckInEnabled)
			}
			if a.CheckInStartDate == nil || !a.CheckInStartDate.Equal(sept1) {
				t.Errorf("CheckInStartDate = %v, want it kept at 2026-09-01", a.CheckInStartDate)
			}
			if tt.wantImmediate {
				if a.PendingCheckInEnabled != nil || a.CheckInEffectiveFrom != nil {
					t.Error("immediate disable must not queue anything")
				}
				if len(adv.zeroedMonths) != 1 || adv.zeroedMonths[0] != "2026-10" {
					t.Errorf("zeroedMonths = %v, want exactly [2026-10]", adv.zeroedMonths)
				}
			} else {
				if a.PendingCheckInEnabled == nil || *a.PendingCheckInEnabled {
					t.Error("day 9+ must queue a pending disable")
				}
				if len(adv.zeroedMonths) != 0 {
					t.Errorf("queued disable must not zero quota at request time, got %v", adv.zeroedMonths)
				}
			}
		})
	}
}
