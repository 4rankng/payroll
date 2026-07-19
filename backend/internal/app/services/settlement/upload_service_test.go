package settlement

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
)

// These tests cover the synchronous, self-verifying settlement path (settleSynchronously)
// that replaced the fire-and-forget async event publish. The key guarantee: a settlement
// that cannot be applied or that fails to flip a timesheet to revenue_paid=1 is surfaced as
// an error to the admin — never silently dropped (the txn 99 / timesheet 11579 bug).

// --- fakes ---

// fakeReader is a minimal domain.TimesheetReader. Only GetByIDs is exercised by
// settleSynchronously; its stored RevenuePaid flags are flipped by fakeApplier.
type fakeReader struct {
	ts map[uint]*domain.Timesheet
}

func (f *fakeReader) GetByIDs(_ context.Context, ids []uint) ([]*domain.Timesheet, error) {
	out := make([]*domain.Timesheet, 0, len(ids))
	for _, id := range ids {
		if t, ok := f.ts[id]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeReader) GetByID(context.Context, uint) (*domain.Timesheet, error) {
	return nil, nil
}
func (f *fakeReader) GetByIDsWithoutRelations(context.Context, []uint) ([]*domain.Timesheet, error) {
	return nil, nil
}
func (f *fakeReader) GetTimesheetDatesByIDs(_ context.Context, ids []uint) (map[uint]time.Time, error) {
	out := make(map[uint]time.Time, len(ids))
	for _, id := range ids {
		if t, ok := f.ts[id]; ok {
			out[id] = t.Date
		}
	}
	return out, nil
}
func (f *fakeReader) List(context.Context, domain.TimesheetFilters) ([]*domain.Timesheet, error) {
	return nil, nil
}
func (f *fakeReader) Count(context.Context, domain.TimesheetFilters) (int64, error) {
	return 0, nil
}
func (f *fakeReader) GetByProjectAndEmployee(context.Context, uint, uint, time.Time, time.Time) ([]*domain.Timesheet, error) {
	return nil, nil
}
func (f *fakeReader) GetByProject(context.Context, uint, time.Time, time.Time) ([]*domain.Timesheet, error) {
	return nil, nil
}
func (f *fakeReader) GetByEmployee(context.Context, uint, time.Time, time.Time) ([]*domain.Timesheet, error) {
	return nil, nil
}
func (f *fakeReader) GetByEmployeeAndPeriod(context.Context, uint, time.Time, time.Time) ([]*domain.Timesheet, error) {
	return nil, nil
}
func (f *fakeReader) GetByProjectEmployeeDate(context.Context, uint, uint, time.Time) ([]*domain.Timesheet, error) {
	return nil, nil
}
func (f *fakeReader) GetByTransactionID(context.Context, uint) ([]*domain.Timesheet, error) {
	return nil, nil
}

// fakeApplier implements SettlementApplier. It flips RevenuePaid on the given timesheets
// unless configured to fail (failOn) or to silently no-op (skip) for a transaction.
type fakeApplier struct {
	reader *fakeReader
	failOn map[uint]error // txnID -> error
	skip   map[uint]bool  // txnID -> succeed without flipping (simulate update no-op)
	calls  []uint
}

func (a *fakeApplier) ApplySettlement(_ context.Context, txnID uint, _ int64, _ uint, _ string, tsIDs []uint) error {
	a.calls = append(a.calls, txnID)
	if err, ok := a.failOn[txnID]; ok && err != nil {
		return err
	}
	if a.skip[txnID] {
		return nil
	}
	for _, id := range tsIDs {
		if t, ok := a.reader.ts[id]; ok {
			t.RevenuePaid = true
		}
	}
	return nil
}

type captureBus struct {
	published []string
}

func (b *captureBus) Publish(_ context.Context, ev ...domain.DomainEvent) error {
	for _, e := range ev {
		b.published = append(b.published, e.EventType())
	}
	return nil
}
func (b *captureBus) Subscribe(string, domain.EventHandler) {}
func (b *captureBus) SubscribeAll(domain.EventHandler)      {}

func newService(reader *fakeReader, applier *fakeApplier) *SettlementUploadService {
	return &SettlementUploadService{
		eventBus:          &captureBus{},
		timesheetReader:   reader,
		settlementApplier: applier,
	}
}

// --- tests ---

func TestSettleSynchronously_HappyPath(t *testing.T) {
	reader := &fakeReader{ts: map[uint]*domain.Timesheet{
		1: {ID: 1}, 2: {ID: 2}, 3: {ID: 3},
	}}
	applier := &fakeApplier{reader: reader, failOn: map[uint]error{}, skip: map[uint]bool{}}
	bus := &captureBus{}
	svc := &SettlementUploadService{
		eventBus:          bus,
		timesheetReader:   reader,
		settlementApplier: applier,
	}
	result := &SettlementValidationResult{
		Transactions:           map[uint]int64{10: 100, 20: 200},
		TimesheetIDs:           []uint{1, 2, 3},
		TimesheetToTransaction: map[uint]uint{1: 10, 2: 20, 3: 20},
	}

	if err := svc.settleSynchronously(context.Background(), result, 300, 99, "sao_ke.xlsx"); err != nil {
		t.Fatalf("expected nil error on happy path, got %v", err)
	}
	if len(applier.calls) != 2 {
		t.Fatalf("expected 2 synchronous ApplySettlement calls, got %d", len(applier.calls))
	}
	if len(bus.published) != 2 {
		t.Fatalf("expected 2 informational audit events, got %d", len(bus.published))
	}
	for _, id := range result.TimesheetIDs {
		if !reader.ts[id].RevenuePaid {
			t.Fatalf("timesheet %d was not marked revenue_paid=1", id)
		}
	}
}

// TestSettleSynchronously_ApplierFailureIsSurfaced is the core regression: a failed
// settlement MUST surface as an error, not be silently dropped by an async worker.
func TestSettleSynchronously_ApplierFailureIsSurfaced(t *testing.T) {
	reader := &fakeReader{ts: map[uint]*domain.Timesheet{1: {ID: 1}, 2: {ID: 2}}}
	applier := &fakeApplier{
		reader: reader,
		failOn: map[uint]error{10: errors.New("settlement worker exploded")},
		skip:   map[uint]bool{},
	}
	svc := newService(reader, applier)
	result := &SettlementValidationResult{
		Transactions:           map[uint]int64{10: 100, 20: 200},
		TimesheetIDs:           []uint{1, 2},
		TimesheetToTransaction: map[uint]uint{1: 10, 2: 20},
	}

	err := svc.settleSynchronously(context.Background(), result, 300, 99, "x.xlsx")
	if err == nil {
		t.Fatal("expected an error when ApplySettlement fails — must not be silently dropped")
	}
	if !strings.Contains(err.Error(), "đối soát lại file") {
		t.Fatalf("error should guide the admin to re-upload to finish, got: %v", err)
	}
}

// TestSettleSynchronously_DoubleCheckCatchesUnflippedTimesheet verifies the
// post-settle guarantee: if any INTERNAL timesheet did not reach revenue_paid=1
// (e.g. a BulkUpdate that affected 0 rows, or a mapping gap), the operation fails
// with a validation error naming the offending ID instead of reporting success.
func TestSettleSynchronously_DoubleCheckCatchesUnflippedTimesheet(t *testing.T) {
	reader := &fakeReader{ts: map[uint]*domain.Timesheet{1: {ID: 1}, 2: {ID: 2}}}
	// Transaction 10 "succeeds" but flips nothing (simulates a 0-row update / dropped flip).
	applier := &fakeApplier{reader: reader, failOn: map[uint]error{}, skip: map[uint]bool{10: true}}
	svc := newService(reader, applier)
	result := &SettlementValidationResult{
		Transactions:           map[uint]int64{10: 100, 20: 200},
		TimesheetIDs:           []uint{1, 2},
		TimesheetToTransaction: map[uint]uint{1: 10, 2: 20},
	}

	err := svc.settleSynchronously(context.Background(), result, 300, 99, "x.xlsx")
	if err == nil {
		t.Fatal("expected validation error when a timesheet is not flipped to revenue_paid")
	}
	if !domain.IsValidationError(err) {
		t.Fatalf("expected a validation error, got %T: %v", err, err)
	}
	// Timesheet 1 belongs to the skipped txn 10 and must be named in the error.
	if !strings.Contains(err.Error(), "1") {
		t.Fatalf("error should name the unflipped timesheet ID (1), got: %v", err)
	}
}

func TestSettleSynchronously_NilApplierIsRejected(t *testing.T) {
	svc := &SettlementUploadService{
		eventBus:        &captureBus{},
		timesheetReader: &fakeReader{ts: map[uint]*domain.Timesheet{}},
		// settlementApplier intentionally nil
	}
	result := &SettlementValidationResult{
		Transactions:           map[uint]int64{10: 100},
		TimesheetIDs:           []uint{1},
		TimesheetToTransaction: map[uint]uint{1: 10},
	}
	if err := svc.settleSynchronously(context.Background(), result, 100, 99, "x.xlsx"); err == nil {
		t.Fatal("expected error when settlement applier is not configured")
	}
}

func TestVerifyRevenuePaid_FlagsExistingUnpaid(t *testing.T) {
	// ID 1 settled, ID 2 exists but still unpaid → must be flagged.
	reader := &fakeReader{ts: map[uint]*domain.Timesheet{
		1: {ID: 1, RevenuePaid: true},
		2: {ID: 2, RevenuePaid: false},
	}}
	svc := &SettlementUploadService{timesheetReader: reader}
	err := svc.verifyRevenuePaid(context.Background(), []uint{1, 2})
	if err == nil {
		t.Fatal("expected validation error for an existing unpaid timesheet")
	}
	if !domain.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if !strings.Contains(err.Error(), "2") {
		t.Fatalf("error should name the unpaid timesheet ID 2, got: %v", err)
	}
}

func TestVerifyRevenuePaid_IgnoresConcurrentlyDeleted(t *testing.T) {
	// ID 1 settled, ID 3 vanished (concurrently deleted, not returned by reader).
	// No existing unpaid rows → must NOT fail: a deleted row is not a stranded receivable.
	reader := &fakeReader{ts: map[uint]*domain.Timesheet{
		1: {ID: 1, RevenuePaid: true},
	}}
	svc := &SettlementUploadService{timesheetReader: reader}
	if err := svc.verifyRevenuePaid(context.Background(), []uint{1, 3}); err != nil {
		t.Fatalf("expected no error when only missing rows are absent, got %v", err)
	}
}
