package zaloconnect

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/zalo"
	"api-server/internal/pkg/clock"
)

type fakeSalaryNotificationRepo struct {
	mu     sync.Mutex
	nextID uint
	byID   map[uint]*domain.FlexPaySalaryNotification
	byKey  map[string]uint
}

func newFakeSalaryNotificationRepo() *fakeSalaryNotificationRepo {
	return &fakeSalaryNotificationRepo{byID: map[uint]*domain.FlexPaySalaryNotification{}, byKey: map[string]uint{}}
}

func salaryKey(n *domain.FlexPaySalaryNotification) string {
	return fmt.Sprintf("%d:%d:%d:%s", n.AssetID, n.ProjectID, n.EmployeeID, n.TemplateID)
}

func (r *fakeSalaryNotificationRepo) CreateIfAbsent(_ context.Context, n *domain.FlexPaySalaryNotification) (*domain.FlexPaySalaryNotification, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := salaryKey(n)
	if id, ok := r.byKey[key]; ok {
		copy := *r.byID[id]
		return &copy, false, nil
	}
	r.nextID++
	copy := *n
	copy.ID = r.nextID
	r.byID[copy.ID] = &copy
	r.byKey[key] = copy.ID
	return &copy, true, nil
}

func (r *fakeSalaryNotificationRepo) Claim(_ context.Context, id uint, now, leaseUntil time.Time) (*domain.FlexPaySalaryNotification, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := r.byID[id]
	if n == nil || (n.Status != domain.FlexPaySalaryNotificationPending && (n.Status != domain.FlexPaySalaryNotificationProcessing || n.LeaseExpiresAt == nil || !n.LeaseExpiresAt.Before(now))) {
		return nil, false, nil
	}
	n.Status = domain.FlexPaySalaryNotificationProcessing
	n.Attempt++
	n.LeaseExpiresAt = &leaseUntil
	copy := *n
	return &copy, true, nil
}

func (r *fakeSalaryNotificationRepo) ReleaseForRetry(_ context.Context, id uint, attempt uint, reason string) error {
	r.finish(id, attempt, domain.FlexPaySalaryNotificationPending, time.Time{}, reason, 0, "")
	return nil
}

func (r *fakeSalaryNotificationRepo) MarkSent(_ context.Context, id uint, attempt uint, at time.Time, msgID string) error {
	r.finish(id, attempt, domain.FlexPaySalaryNotificationSent, at, "", 0, msgID)
	return nil
}

func (r *fakeSalaryNotificationRepo) MarkFailed(_ context.Context, id uint, attempt uint, at time.Time, reason string, code int) error {
	r.finish(id, attempt, domain.FlexPaySalaryNotificationFailed, at, reason, code, "")
	return nil
}

func (r *fakeSalaryNotificationRepo) MarkSuppressed(_ context.Context, id uint, attempt uint, at time.Time, reason string, code int) error {
	r.finish(id, attempt, domain.FlexPaySalaryNotificationSuppressed, at, reason, code, "")
	return nil
}

func (r *fakeSalaryNotificationRepo) finish(id uint, attempt uint, status string, at time.Time, reason string, code int, msgID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := r.byID[id]
	if n == nil || n.Attempt != attempt || n.Status != domain.FlexPaySalaryNotificationProcessing {
		return
	}
	n.Status, n.LeaseExpiresAt, n.LastError, n.ProviderCode, n.ProviderMsgID = status, nil, reason, code, msgID
	if !at.IsZero() {
		n.SentAt = &at
	}
}

func (r *fakeSalaryNotificationRepo) ListRecoverable(_ context.Context, now time.Time, _ int) ([]*domain.FlexPaySalaryNotification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*domain.FlexPaySalaryNotification
	for _, n := range r.byID {
		if n.Status == domain.FlexPaySalaryNotificationPending || (n.Status == domain.FlexPaySalaryNotificationProcessing && n.LeaseExpiresAt != nil && n.LeaseExpiresAt.Before(now)) {
			copy := *n
			result = append(result, &copy)
		}
	}
	return result, nil
}

type fakeSalaryNotificationEnqueuer struct {
	ids []uint
	err error
}

func (e *fakeSalaryNotificationEnqueuer) EnqueueFlexPaySalaryNotification(id uint) error {
	e.ids = append(e.ids, id)
	return e.err
}

func testSalaryRecipient() SalaryNotificationRecipient {
	return SalaryNotificationRecipient{ProjectID: 7, EmployeeID: 9, EmployeeName: "Nguyễn Việt Dũng", Mobile: "0357210887", Amount: 1000000, ExpiryDate: time.Date(2026, 6, 18, 0, 0, 0, 0, clock.DefaultLocation)}
}

func testSalaryService(repo *fakeSalaryNotificationRepo, sender zalo.Sender, enqueue *fakeSalaryNotificationEnqueuer, enabled func(context.Context) (bool, error), now time.Time) *SalaryNotificationDeliveryService {
	return NewSalaryNotificationDeliveryService(repo, sender, enqueue, enabled, clock.NewFake(now), nil)
}

func TestSalaryNotificationScheduleIsDurableAndIdempotent(t *testing.T) {
	repo, queue := newFakeSalaryNotificationRepo(), &fakeSalaryNotificationEnqueuer{}
	svc := testSalaryService(repo, &MockZaloProvider{}, queue, mockEnabledCheck, time.Date(2026, 6, 1, 9, 0, 0, 0, clock.DefaultLocation))
	if err := svc.ScheduleImportRecipients(context.Background(), 50, []SalaryNotificationRecipient{testSalaryRecipient()}); err != nil {
		t.Fatal(err)
	}
	if err := svc.ScheduleImportRecipients(context.Background(), 50, []SalaryNotificationRecipient{testSalaryRecipient()}); err != nil {
		t.Fatal(err)
	}
	if len(repo.byID) != 1 {
		t.Fatalf("records = %d, want 1", len(repo.byID))
	}
	if len(queue.ids) != 2 || queue.ids[0] != queue.ids[1] {
		t.Fatalf("queue ids = %v, want same persisted record", queue.ids)
	}
}

func TestSalaryNotificationDeliverRetriesTransportFailure(t *testing.T) {
	repo, queue := newFakeSalaryNotificationRepo(), &fakeSalaryNotificationEnqueuer{}
	sender := &MockZaloProvider{SendFunc: func(context.Context, string, string, string, map[string]string) (zalo.SendResult, error) {
		return zalo.SendResult{}, errors.New("timeout")
	}}
	svc := testSalaryService(repo, sender, queue, mockEnabledCheck, time.Date(2026, 6, 1, 9, 0, 0, 0, clock.DefaultLocation))
	_ = svc.ScheduleImportRecipients(context.Background(), 51, []SalaryNotificationRecipient{testSalaryRecipient()})
	if err := svc.Deliver(context.Background(), queue.ids[0]); err == nil {
		t.Fatal("expected retry error")
	}
	if got := repo.byID[queue.ids[0]].Status; got != domain.FlexPaySalaryNotificationPending {
		t.Fatalf("status = %s, want pending", got)
	}
}

func TestSalaryNotificationDeliverRecordsSuccessAndStableTrackingID(t *testing.T) {
	repo, queue := newFakeSalaryNotificationRepo(), &fakeSalaryNotificationEnqueuer{}
	sender := &MockZaloProvider{SendFunc: func(_ context.Context, _ string, _ string, trackingID string, data map[string]string) (zalo.SendResult, error) {
		if trackingID != "flexpay-salary-zns:1" {
			t.Fatalf("tracking id = %q", trackingID)
		}
		if data["customer_name"] != "Nguyễn Việt Dũng" || data["max_amount"] != "1000000" || data["expiry_date"] != "18/06/2026" {
			t.Fatalf("unexpected template data: %#v", data)
		}
		return zalo.SendResult{MsgID: "zalo-message", ErrorCode: zalo.ErrOK}, nil
	}}
	svc := testSalaryService(repo, sender, queue, mockEnabledCheck, time.Date(2026, 6, 1, 9, 0, 0, 0, clock.DefaultLocation))
	_ = svc.ScheduleImportRecipients(context.Background(), 52, []SalaryNotificationRecipient{testSalaryRecipient()})
	if err := svc.Deliver(context.Background(), queue.ids[0]); err != nil {
		t.Fatal(err)
	}
	n := repo.byID[queue.ids[0]]
	if n.Status != domain.FlexPaySalaryNotificationSent || n.ProviderMsgID != "zalo-message" {
		t.Fatalf("unexpected final notification: %#v", n)
	}
}

func TestSalaryNotificationSuppressesTerminalBusinessError(t *testing.T) {
	repo, queue := newFakeSalaryNotificationRepo(), &fakeSalaryNotificationEnqueuer{}
	sender := &MockZaloProvider{SendFunc: func(context.Context, string, string, string, map[string]string) (zalo.SendResult, error) {
		return zalo.SendResult{ErrorCode: zalo.ErrNoZaloAccount, ErrorMsg: "not linked"}, nil
	}}
	svc := testSalaryService(repo, sender, queue, mockEnabledCheck, time.Date(2026, 6, 1, 9, 0, 0, 0, clock.DefaultLocation))
	_ = svc.ScheduleImportRecipients(context.Background(), 53, []SalaryNotificationRecipient{testSalaryRecipient()})
	if err := svc.Deliver(context.Background(), queue.ids[0]); err != nil {
		t.Fatal(err)
	}
	if got := repo.byID[queue.ids[0]].Status; got != domain.FlexPaySalaryNotificationSuppressed {
		t.Fatalf("status = %s, want suppressed", got)
	}
}

func TestSalaryNotificationRecoverRequeuesPending(t *testing.T) {
	repo, queue := newFakeSalaryNotificationRepo(), &fakeSalaryNotificationEnqueuer{err: errors.New("redis unavailable")}
	svc := testSalaryService(repo, &MockZaloProvider{}, queue, mockEnabledCheck, time.Date(2026, 6, 1, 9, 0, 0, 0, clock.DefaultLocation))
	_ = svc.ScheduleImportRecipients(context.Background(), 54, []SalaryNotificationRecipient{testSalaryRecipient()})
	queue.err = nil
	if err := svc.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(queue.ids) != 2 {
		t.Fatalf("enqueue calls = %d, want 2", len(queue.ids))
	}
}
