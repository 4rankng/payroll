package notification

import (
	"context"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
)

type fakeActivationAssignmentRepo struct {
	assignments []*domain.ProjectEmployee
	err         error
	queriedDate *time.Time
}

func (f *fakeActivationAssignmentRepo) GetEmployeesWithPendingCheckInEnableEffective(ctx context.Context, effectiveDate time.Time) ([]*domain.ProjectEmployee, error) {
	d := time.Date(effectiveDate.Year(), effectiveDate.Month(), effectiveDate.Day(), 0, 0, 0, 0, effectiveDate.Location())
	f.queriedDate = &d
	return f.assignments, f.err
}

type fakeActivationNotificationRepo struct {
	exists      bool
	queriedUser uint
	queriedType domain.NotificationType
	queriedDay  time.Time
}

func (f *fakeActivationNotificationRepo) HasNotificationForUserOn(ctx context.Context, userID uint, notificationType domain.NotificationType, day time.Time) (bool, error) {
	f.queriedUser = userID
	f.queriedType = notificationType
	f.queriedDay = day
	return f.exists, nil
}

type fakeActivationNotifier struct {
	created []struct {
		userID uint
		nType  domain.NotificationType
		title  string
		body   string
	}
}

func (f *fakeActivationNotifier) CreateNotification(ctx context.Context, userID uint, notificationType domain.NotificationType, title, message string) error {
	f.created = append(f.created, struct {
		userID uint
		nType  domain.NotificationType
		title  string
		body   string
	}{userID, notificationType, title, message})
	return nil
}

func newActivationService(t *testing.T, assignments []*domain.ProjectEmployee, repo *fakeActivationNotificationRepo, notifier *fakeActivationNotifier) *CheckInActivationReminderService {
	t.Helper()
	return NewCheckInActivationReminderService(
		&fakeActivationAssignmentRepo{assignments: assignments},
		notifier,
		repo,
		nil,
	)
}

func pendingEnableAssignment(userID *uint) *domain.ProjectEmployee {
	return &domain.ProjectEmployee{
		Employee:              domain.Employee{UserID: userID},
		PendingCheckInEnabled: boolPtr(true),
	}
}

func boolPtr(b bool) *bool { return &b }

func uintPtr(u uint) *uint { return &u }

func TestSendExitsOutsideMonthEnd(t *testing.T) {
	// Tomorrow is not day 1 -> the guard must exit before even querying.
	cases := []struct {
		name string
		now  time.Time
	}{
		{"mid-month", time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)},
		{"leap-feb-28", time.Date(2028, 2, 28, 9, 0, 0, 0, time.UTC)}, // tomorrow = Feb 29
		{"first-day", time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)},   // tomorrow = Nov 2
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeActivationAssignmentRepo{}
			notifier := &fakeActivationNotifier{}
			svc := newActivationService(t, nil, &fakeActivationNotificationRepo{}, notifier)
			// Point the fake at a repo that would return data if queried.
			svc.assignments = repo

			count, err := svc.Send(context.Background(), tc.now)
			if err != nil {
				t.Fatalf("Send: %v", err)
			}
			if count != 0 {
				t.Errorf("expected 0 notifications, got %d", count)
			}
			if repo.queriedDate != nil {
				t.Error("expected no repository query on a non-month-end day")
			}
			if len(notifier.created) != 0 {
				t.Errorf("expected no notifications, got %d", len(notifier.created))
			}
		})
	}
}

func TestSendActsOnMonthEndAndNotifies(t *testing.T) {
	// Oct 31: tomorrow = Nov 1 (day 1) -> the acting run.
	notifier := &fakeActivationNotifier{}
	repo := &fakeActivationNotificationRepo{}
	uid := uint(77)
	assignments := []*domain.ProjectEmployee{pendingEnableAssignment(uintPtr(uid))}
	svc := newActivationService(t, assignments, repo, notifier)

	count, err := svc.Send(context.Background(), time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if count != 1 || len(notifier.created) != 1 {
		t.Fatalf("expected 1 notification, got count=%d created=%d", count, len(notifier.created))
	}
	created := notifier.created[0]
	if created.userID != uid {
		t.Errorf("wrong recipient: %d", created.userID)
	}
	if created.nType != domain.NotificationTypeCheckInActivationReminder {
		t.Errorf("wrong notification type: %s", created.nType)
	}
	if !strings.Contains(created.body, "01/11/2026") {
		t.Errorf("message should announce the effective date, got: %s", created.body)
	}
}

func TestSendCoversEveryMonthLength(t *testing.T) {
	// The days-28-31 cron + guard must produce exactly one acting run for
	// every month shape (30/31 days, Feb leap and non-leap).
	cases := []struct {
		name   string
		now    time.Time
		active bool
	}{
		{"oct-31", time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC), true},
		{"sep-30", time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC), true},
		{"feb-28-nonleap", time.Date(2027, 2, 28, 9, 0, 0, 0, time.UTC), true},
		{"feb-28-leap-skips", time.Date(2028, 2, 28, 9, 0, 0, 0, time.UTC), false},
		{"feb-29-leap-acts", time.Date(2028, 2, 29, 9, 0, 0, 0, time.UTC), true},
		{"apr-30", time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			notifier := &fakeActivationNotifier{}
			uid := uint(5)
			svc := newActivationService(t, []*domain.ProjectEmployee{pendingEnableAssignment(uintPtr(uid))}, &fakeActivationNotificationRepo{}, notifier)

			if _, err := svc.Send(context.Background(), tc.now); err != nil {
				t.Fatalf("Send: %v", err)
			}
			got := len(notifier.created)
			if tc.active && got != 1 {
				t.Errorf("expected the acting run to notify once, got %d", got)
			}
			if !tc.active && got != 0 {
				t.Errorf("expected a silent guard exit, got %d notifications", got)
			}
		})
	}
}

func TestSendSkipsEmployeeWithoutLinkedUser(t *testing.T) {
	notifier := &fakeActivationNotifier{}
	assignments := []*domain.ProjectEmployee{
		pendingEnableAssignment(nil),        // no user account -> skip + log
		pendingEnableAssignment(uintPtr(9)), // linked -> notify
	}
	svc := newActivationService(t, assignments, &fakeActivationNotificationRepo{}, notifier)

	count, err := svc.Send(context.Background(), time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if count != 1 || len(notifier.created) != 1 {
		t.Fatalf("expected exactly the linked employee to be notified, got count=%d created=%d", count, len(notifier.created))
	}
	if notifier.created[0].userID != 9 {
		t.Errorf("wrong recipient: %d", notifier.created[0].userID)
	}
}

func TestSendDedupsSameDayRepeat(t *testing.T) {
	// AC-06: a same-day re-run must not duplicate the reminder.
	notifier := &fakeActivationNotifier{}
	uid := uint(77)
	repo := &fakeActivationNotificationRepo{exists: true}
	svc := newActivationService(t,
		[]*domain.ProjectEmployee{pendingEnableAssignment(uintPtr(uid))},
		repo,
		notifier,
	)

	now := time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)
	count, err := svc.Send(context.Background(), now)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if count != 0 || len(notifier.created) != 0 {
		t.Errorf("expected dedup to suppress the reminder, got count=%d created=%d", count, len(notifier.created))
	}
	// The dedup probe must target the right user, type, and day window.
	if repo.queriedUser != uid || repo.queriedType != domain.NotificationTypeCheckInActivationReminder {
		t.Errorf("dedup probe wrong user/type: %d/%s", repo.queriedUser, repo.queriedType)
	}
	if repo.queriedDay.Format("2006-01-02") != "2026-10-31" {
		t.Errorf("dedup probe wrong day: %s", repo.queriedDay)
	}
}
