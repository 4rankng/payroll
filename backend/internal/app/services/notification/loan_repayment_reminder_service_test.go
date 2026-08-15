package notification

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/pkg/utils"
)

type fakeReminderScheduleRepo struct {
	rows      []*domain.LoanRepaymentReminder
	gotStart  time.Time
	gotEnd    time.Time
	callCount int
	listErr   error
}

func (f *fakeReminderScheduleRepo) ListPendingForReminder(ctx context.Context, start, end time.Time) ([]*domain.LoanRepaymentReminder, error) {
	f.callCount++
	f.gotStart = start
	f.gotEnd = end
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.rows, nil
}

type fakeReminderUserRepo struct {
	admins []*domain.User
}

func (f *fakeReminderUserRepo) ListByRole(ctx context.Context, role domain.UserRole) ([]*domain.User, error) {
	return f.admins, nil
}

type fakeReminderNotifier struct {
	role    domain.UserRole
	nType   domain.NotificationType
	title   string
	message string
	calls   int
	err     error
}

func (f *fakeReminderNotifier) NotifyUsersByRole(ctx context.Context, role domain.UserRole, nType domain.NotificationType, title, message string) error {
	f.calls++
	f.role = role
	f.nType = nType
	f.title = title
	f.message = message
	return f.err
}

type fakeReminderEmail struct {
	payload *dto.SendEmailRequest
	calls   int
	err     error
}

func (f *fakeReminderEmail) SendGenericEmail(ctx context.Context, payload *dto.SendEmailRequest) (string, error) {
	f.calls++
	f.payload = payload
	return "msg-id", f.err
}

var reminderLocation = time.FixedZone("ICT", 7*3600)

func TestLoanRepaymentReminderService_NoSchedulesSendsNothing(t *testing.T) {
	schedules := &fakeReminderScheduleRepo{rows: nil}
	notifier := &fakeReminderNotifier{}
	email := &fakeReminderEmail{}
	service := NewLoanRepaymentReminderService(schedules, &fakeReminderUserRepo{}, notifier, email, nil)

	now := time.Date(2026, 8, 14, 9, 0, 0, 0, reminderLocation)
	count, err := service.Send(context.Background(), now)
	if err != nil {
		t.Fatalf("Send with no schedules: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 schedules, got %d", count)
	}
	if notifier.calls != 0 {
		t.Fatalf("notification channel must not fire when nothing is due, got %d calls", notifier.calls)
	}
	if email.calls != 0 {
		t.Fatalf("email channel must not fire when nothing is due, got %d calls", email.calls)
	}
}

func TestLoanRepaymentReminderService_ConsolidatesContentAndRecipients(t *testing.T) {
	schedules := &fakeReminderScheduleRepo{rows: []*domain.LoanRepaymentReminder{
		{ScheduleID: 3, LoanCode: "LOAN-2026-003", LenderName: "Ngân hàng C", Period: 2, Amount: 2000000, DueDate: time.Date(2026, 8, 14, 0, 0, 0, 0, reminderLocation)},
		{ScheduleID: 2, LoanCode: "LOAN-2026-002", LenderName: "Ngân hàng B", Period: 3, Amount: 5000000, DueDate: time.Date(2026, 8, 15, 0, 0, 0, 0, reminderLocation)},
		{ScheduleID: 1, LoanCode: "LOAN-2026-001", LenderName: "Ngân hàng A", Period: 1, Amount: 1000000, DueDate: time.Date(2026, 8, 15, 0, 0, 0, 0, reminderLocation)},
	}}
	notifier := &fakeReminderNotifier{}
	email := &fakeReminderEmail{}
	users := &fakeReminderUserRepo{admins: []*domain.User{
		{Email: new("admin@tingting.vip")},
		{Email: new("  Admin@TingTing.vip  ")}, // duplicate after trim+lowercase
		{Email: new("   ")},                    // blank, skipped
		{Email: nil},                           // nil, skipped
	}}
	service := NewLoanRepaymentReminderService(schedules, users, notifier, email, nil)

	now := time.Date(2026, 8, 14, 9, 0, 0, 0, reminderLocation)
	count, err := service.Send(context.Background(), now)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 schedules, got %d", count)
	}

	if notifier.calls != 1 {
		t.Fatalf("expected exactly 1 consolidated notification, got %d", notifier.calls)
	}
	if notifier.role != domain.RoleAdmin {
		t.Fatalf("expected RoleAdmin, got %v", notifier.role)
	}
	if notifier.nType != domain.NotificationTypeLoanInterestDue {
		t.Fatalf("expected NotificationTypeLoanInterestDue, got %v", notifier.nType)
	}
	if !strings.Contains(notifier.title, "14/08/2026") || !strings.Contains(notifier.title, "15/08/2026") {
		t.Fatalf("title should carry today 14/08/2026 and tomorrow 15/08/2026, got %q", notifier.title)
	}
	// Consolidation: sorted loan codes, lender names, vi-VN amounts, and total.
	lower := notifier.message
	for _, want := range []string{
		"LOAN-2026-001",
		"Ngân hàng A",
		utils.FormatVND(1000000),
		"LOAN-2026-002",
		"Ngân hàng B",
		utils.FormatVND(5000000),
		utils.FormatVND(8000000),
	} {
		if !strings.Contains(lower, want) {
			t.Fatalf("notification body missing %q:\n%s", want, lower)
		}
	}
	if strings.Index(lower, "LOAN-2026-001") > strings.Index(lower, "LOAN-2026-002") {
		t.Fatalf("loans must be sorted by loan code:\n%s", lower)
	}

	if email.calls != 1 {
		t.Fatalf("expected exactly 1 consolidated email, got %d", email.calls)
	}
	got := email.payload.Recipients
	if len(got) != 1 || got[0] != "admin@tingting.vip" {
		t.Fatalf("expected trimmed deduped recipients [admin@tingting.vip], got %#v", got)
	}
	if email.payload.Subject != notifier.title {
		t.Fatalf("email subject should match notification title, got %q vs %q", email.payload.Subject, notifier.title)
	}
	if email.payload.HTMLBody == "" || email.payload.TextBody == "" {
		t.Fatal("email should carry both HTML and text bodies")
	}
	if !strings.Contains(email.payload.HTMLBody, "LOAN-2026-001") || !strings.Contains(email.payload.HTMLBody, "<ul>") {
		t.Fatalf("HTML body should list loans as a list:\n%s", email.payload.HTMLBody)
	}
}

func TestLoanRepaymentReminderService_ChannelsAreIndependent(t *testing.T) {
	rows := []*domain.LoanRepaymentReminder{
		{ScheduleID: 1, LoanCode: "LOAN-2026-001", LenderName: "Bank A", Period: 1, Amount: 100},
	}

	notifierErr := &fakeReminderNotifier{err: errors.New("push down")}
	email := &fakeReminderEmail{}
	service := NewLoanRepaymentReminderService(
		&fakeReminderScheduleRepo{rows: rows},
		&fakeReminderUserRepo{admins: []*domain.User{{Email: new("admin@tingting.vip")}}},
		notifierErr,
		email,
		nil,
	)
	count, err := service.Send(context.Background(), time.Date(2026, 8, 14, 9, 0, 0, 0, reminderLocation))
	if err == nil {
		t.Fatal("push failure must surface as an error")
	}
	if count != 1 {
		t.Fatalf("schedule count must be reported even on delivery error, got %d", count)
	}
	if email.calls != 1 {
		t.Fatal("email must still be attempted when the notification channel fails")
	}

	notifierOK := &fakeReminderNotifier{}
	emailErr := &fakeReminderEmail{err: errors.New("smtp down")}
	service = NewLoanRepaymentReminderService(
		&fakeReminderScheduleRepo{rows: rows},
		&fakeReminderUserRepo{admins: []*domain.User{{Email: new("admin@tingting.vip")}}},
		notifierOK,
		emailErr,
		nil,
	)
	if _, err := service.Send(context.Background(), time.Date(2026, 8, 14, 9, 0, 0, 0, reminderLocation)); err == nil {
		t.Fatal("email failure must surface as an error")
	}
	if notifierOK.calls != 1 {
		t.Fatal("notification must still be attempted when the email channel fails")
	}
}

func TestLoanRepaymentReminderService_ComputesTomorrowWindowInNowLocation(t *testing.T) {
	cases := []struct {
		name        string
		now         time.Time
		wantStart   time.Time
		wantEndHour int
	}{
		{
			name:      "month rollover",
			now:       time.Date(2026, 1, 31, 9, 0, 0, 0, reminderLocation),
			wantStart: time.Date(2026, 2, 1, 0, 0, 0, 0, reminderLocation),
		},
		{
			name:      "year rollover",
			now:       time.Date(2026, 12, 31, 9, 0, 0, 0, reminderLocation),
			wantStart: time.Date(2027, 1, 1, 0, 0, 0, 0, reminderLocation),
		},
		{
			name:      "leap day window",
			now:       time.Date(2028, 2, 28, 9, 0, 0, 0, reminderLocation),
			wantStart: time.Date(2028, 2, 29, 0, 0, 0, 0, reminderLocation),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schedules := &fakeReminderScheduleRepo{}
			service := NewLoanRepaymentReminderService(schedules, &fakeReminderUserRepo{}, &fakeReminderNotifier{}, &fakeReminderEmail{}, nil)

			if _, err := service.Send(context.Background(), tc.now); err != nil {
				t.Fatalf("Send: %v", err)
			}
			wantStart := time.Date(tc.now.Year(), tc.now.Month(), tc.now.Day(), 0, 0, 0, 0, reminderLocation)
			if !schedules.gotStart.Equal(wantStart) {
				t.Fatalf("window start = %v, want %v", schedules.gotStart, wantStart)
			}
			wantEnd := wantStart.AddDate(0, 0, 2)
			if !schedules.gotEnd.Equal(wantEnd) {
				t.Fatalf("window end = %v, want %v", schedules.gotEnd, wantEnd)
			}
			if schedules.gotStart.Location() != reminderLocation {
				t.Fatal("window must be computed in now's location")
			}
		})
	}
}
