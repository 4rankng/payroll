package config

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/infra/persistence"
	"api-server/internal/pkg/secret"

	"github.com/alicebob/miniredis/v2"
)

type fakeSettingsEventBus struct{}

func (fakeSettingsEventBus) Publish(context.Context, ...domain.DomainEvent) error { return nil }
func (fakeSettingsEventBus) Subscribe(string, domain.EventHandler)                {}
func (fakeSettingsEventBus) SubscribeAll(domain.EventHandler)                     {}

type fakeSettingsRepoForUpdate struct {
	rows map[uint]*domain.Settings
}

func newFakeSettingsRepoForUpdate(rows ...*domain.Settings) *fakeSettingsRepoForUpdate {
	out := &fakeSettingsRepoForUpdate{rows: make(map[uint]*domain.Settings, len(rows))}
	for _, row := range rows {
		cp := *row
		out.rows[row.ID] = &cp
	}
	return out
}

func (r *fakeSettingsRepoForUpdate) snapshot() map[uint]*domain.Settings {
	out := make(map[uint]*domain.Settings, len(r.rows))
	for id, row := range r.rows {
		cp := *row
		out[id] = &cp
	}
	return out
}

func (r *fakeSettingsRepoForUpdate) restore(rows map[uint]*domain.Settings) {
	r.rows = rows
}

func (r *fakeSettingsRepoForUpdate) Create(context.Context, *domain.Settings) error {
	panic("unexpected Create call")
}

func (r *fakeSettingsRepoForUpdate) GetByID(_ context.Context, id uint) (*domain.Settings, error) {
	row, ok := r.rows[id]
	if !ok {
		return nil, domain.NewNotFoundError("setting not found")
	}
	cp := *row
	return &cp, nil
}

func (r *fakeSettingsRepoForUpdate) GetByIDForUpdate(ctx context.Context, id uint) (*domain.Settings, error) {
	return r.GetByID(ctx, id)
}

func (r *fakeSettingsRepoForUpdate) GetByKey(_ context.Context, key string) (*domain.Settings, error) {
	for _, row := range r.rows {
		if row.Key == key {
			cp := *row
			return &cp, nil
		}
	}
	return nil, domain.NewNotFoundError("setting not found")
}

func (r *fakeSettingsRepoForUpdate) GetByKeyForUpdate(ctx context.Context, key string) (*domain.Settings, error) {
	return r.GetByKey(ctx, key)
}

func (r *fakeSettingsRepoForUpdate) Update(_ context.Context, setting *domain.Settings) error {
	cp := *setting
	r.rows[setting.ID] = &cp
	return nil
}

func (r *fakeSettingsRepoForUpdate) CompareAndSwapValue(_ context.Context, key, currentValue, nextValue string, valueType domain.SettingsValueType) (bool, error) {
	for id, setting := range r.rows {
		if setting.Key != key || setting.Value == nil || *setting.Value != currentValue {
			continue
		}
		copySetting := *setting
		copySetting.Value = &nextValue
		copySetting.ValueType = valueType
		r.rows[id] = &copySetting
		return true, nil
	}
	return false, nil
}

// EncryptProtectedValues is a no-op here: the fake stores values in memory and
// these tests exercise the update flow, not the startup secret backfill (that
// lives in the persistence layer, covered by settings_repository_secrets_test).
func (r *fakeSettingsRepoForUpdate) EncryptProtectedValues(context.Context) ([]string, error) {
	return nil, nil
}

func (r *fakeSettingsRepoForUpdate) Delete(context.Context, uint) error {
	panic("unexpected Delete call")
}

func (r *fakeSettingsRepoForUpdate) List(context.Context, domain.SettingsFilters) ([]*domain.Settings, error) {
	panic("unexpected List call")
}

func (r *fakeSettingsRepoForUpdate) Count(context.Context, domain.SettingsFilters) (int64, error) {
	panic("unexpected Count call")
}

type fakeQuotaRepoForUpdate struct {
	rows         []*domain.AdvancePayment
	recomputeErr error
	calls        int
}

type fakeAttendanceRepoForSettingsUpdate struct {
	domain.AttendanceRepository
	schedules []domain.QuotaCreditSchedule
	hold      time.Duration
	calls     int
	err       error
}

func (r *fakeAttendanceRepoForSettingsUpdate) RecalculatePendingQuotaCreditSchedules(_ context.Context, hold time.Duration) ([]domain.QuotaCreditSchedule, error) {
	r.calls++
	r.hold = hold
	if r.err != nil {
		return nil, r.err
	}
	return r.schedules, nil
}

type fakeQuotaCreditTaskEnqueuer struct {
	schedules []domain.QuotaCreditSchedule
}

func (e *fakeQuotaCreditTaskEnqueuer) EnqueueCreditQuota(attendanceID uint, at time.Time) error {
	e.schedules = append(e.schedules, domain.QuotaCreditSchedule{AttendanceID: attendanceID, EligibleAt: at})
	return nil
}

func (r *fakeQuotaRepoForUpdate) snapshot() []*domain.AdvancePayment {
	out := make([]*domain.AdvancePayment, 0, len(r.rows))
	for _, row := range r.rows {
		cp := *row
		out = append(out, &cp)
	}
	return out
}

func (r *fakeQuotaRepoForUpdate) restore(rows []*domain.AdvancePayment) {
	r.rows = rows
}

func (r *fakeQuotaRepoForUpdate) RecomputeActiveCheckInMaxAdvance(_ context.Context, advancePercentage uint64) error {
	r.calls++
	if r.recomputeErr != nil {
		return r.recomputeErr
	}
	for _, row := range r.rows {
		row.MaxAdvAmount = (row.Salary * advancePercentage) / 100
	}
	return nil
}

func (r *fakeQuotaRepoForUpdate) Create(context.Context, *domain.AdvancePayment) error {
	panic("unexpected Create call")
}

func (r *fakeQuotaRepoForUpdate) Upsert(context.Context, *domain.AdvancePayment) error {
	panic("unexpected Upsert call")
}

func (r *fakeQuotaRepoForUpdate) GetByID(context.Context, uint64) (*domain.AdvancePayment, error) {
	panic("unexpected GetByID call")
}

func (r *fakeQuotaRepoForUpdate) GetByEmployeeAndMonth(context.Context, uint64, string) ([]*domain.AdvancePayment, error) {
	panic("unexpected GetByEmployeeAndMonth call")
}

func (r *fakeQuotaRepoForUpdate) GetMonthsByEmployee(context.Context, uint64) ([]string, error) {
	panic("unexpected GetMonthsByEmployee call")
}

func (r *fakeQuotaRepoForUpdate) SumMaxAdvByEmployeeMonth(context.Context, uint64, string) (uint64, error) {
	panic("unexpected SumMaxAdvByEmployeeMonth call")
}

func (r *fakeQuotaRepoForUpdate) SumSalaryAndMaxAdvByEmployeeMonth(context.Context, uint64, string) (uint64, uint64, error) {
	panic("unexpected SumSalaryAndMaxAdvByEmployeeMonth call")
}

func (r *fakeQuotaRepoForUpdate) SumPendingEarningsByEmployeeMonth(context.Context, uint64, string) (uint64, error) {
	panic("unexpected SumPendingEarningsByEmployeeMonth call")
}

func (r *fakeQuotaRepoForUpdate) BatchCreate(context.Context, []*domain.AdvancePayment) error {
	panic("unexpected BatchCreate call")
}

func (r *fakeQuotaRepoForUpdate) BatchUpsert(context.Context, []*domain.AdvancePayment) error {
	panic("unexpected BatchUpsert call")
}

func (r *fakeQuotaRepoForUpdate) Update(context.Context, *domain.AdvancePayment) error {
	panic("unexpected Update call")
}

func (r *fakeQuotaRepoForUpdate) AccumulateSalary(context.Context, uint64, int64, uint64) error {
	panic("unexpected AccumulateSalary call")
}

func (r *fakeQuotaRepoForUpdate) ZeroOutQuota(context.Context, uint, uint, string) error {
	panic("unexpected ZeroOutQuota call")
}

func (r *fakeQuotaRepoForUpdate) BatchZeroOutQuota(context.Context, uint, []uint, string) error {
	panic("unexpected BatchZeroOutQuota call")
}

func (r *fakeQuotaRepoForUpdate) GetLatestForMonth(context.Context) (string, error) {
	panic("unexpected GetLatestForMonth call")
}

func (r *fakeQuotaRepoForUpdate) GetEmployeeAdvanceStats(context.Context, domain.EmployeeAdvanceStatsFilters) ([]*domain.EmployeeAdvanceStats, int64, error) {
	panic("unexpected GetEmployeeAdvanceStats call")
}

func (r *fakeQuotaRepoForUpdate) GetAvailableMonths(context.Context) ([]*domain.AvailableMonth, error) {
	panic("unexpected GetAvailableMonths call")
}

func (r *fakeQuotaRepoForUpdate) GetEmployeeByID(context.Context, uint64) (*domain.Employee, error) {
	panic("unexpected GetEmployeeByID call")
}

func (r *fakeQuotaRepoForUpdate) GetEmployeesByIDs(context.Context, []uint64) (map[uint64]*domain.Employee, error) {
	panic("unexpected GetEmployeesByIDs call")
}

func (r *fakeQuotaRepoForUpdate) HasDataForMonth(context.Context, string) (bool, error) {
	panic("unexpected HasDataForMonth call")
}

func (r *fakeQuotaRepoForUpdate) GetQuotaAnomalies(context.Context, string, string, uint64) ([]domain.QuotaAnomaly, error) {
	panic("unexpected GetQuotaAnomalies call")
}

func (r *fakeQuotaRepoForUpdate) CountQuotaAnomalies(context.Context, string, string, uint64) (int, error) {
	panic("unexpected CountQuotaAnomalies call")
}

func (r *fakeQuotaRepoForUpdate) SumSalaryAndMaxAdvForMonth(context.Context, string) (uint64, uint64, error) {
	panic("unexpected SumSalaryAndMaxAdvForMonth call")
}

type fakeSettingsTxManager struct {
	settings *fakeSettingsRepoForUpdate
	quota    *fakeQuotaRepoForUpdate
}

func (tm fakeSettingsTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	settingsSnapshot := tm.settings.snapshot()
	quotaSnapshot := tm.quota.snapshot()
	if err := fn(ctx); err != nil {
		tm.settings.restore(settingsSnapshot)
		tm.quota.restore(quotaSnapshot)
		return err
	}
	return nil
}

func (tm fakeSettingsTxManager) WithTransactionResult(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	settingsSnapshot := tm.settings.snapshot()
	quotaSnapshot := tm.quota.snapshot()
	result, err := fn(ctx)
	if err != nil {
		tm.settings.restore(settingsSnapshot)
		tm.quota.restore(quotaSnapshot)
	}
	return result, err
}

func TestSettingsServiceUpdateSettingRecomputesSelfCheckInQuota(t *testing.T) {
	settingsRepo := newFakeSettingsRepoForUpdate(&domain.Settings{
		ID:        1,
		Key:       SettingKeySelfCheckInAdvancePercent,
		Value:     stringPointer("70"),
		ValueType: domain.ValueTypeNumber,
	})
	quotaRepo := &fakeQuotaRepoForUpdate{rows: []*domain.AdvancePayment{
		{ID: 1, Salary: 1_000_000, MaxAdvAmount: 700_000},
	}}
	service := NewSettingsService(
		settingsRepo,
		quotaRepo,
		nil,
		fakeSettingsTxManager{settings: settingsRepo, quota: quotaRepo},
		nil,
		fakeSettingsEventBus{},
		nil,
	)

	updated, err := service.UpdateSetting(context.Background(), 1, map[string]any{
		"value": "85",
	}, 7)
	if err != nil {
		t.Fatalf("UpdateSetting returned error: %v", err)
	}
	if updated.Value == nil || *updated.Value != "85" {
		t.Fatalf("updated value = %v, want 85", updated.Value)
	}
	if quotaRepo.calls != 1 {
		t.Fatalf("expected 1 quota recompute, got %d", quotaRepo.calls)
	}
	if quotaRepo.rows[0].MaxAdvAmount != 850_000 {
		t.Fatalf("self-check-in row max_adv_amount = %d, want 850000", quotaRepo.rows[0].MaxAdvAmount)
	}
}

func TestSettingsServiceUpdateSettingRollsBackWhenQuotaRecomputeFails(t *testing.T) {
	settingsRepo := newFakeSettingsRepoForUpdate(&domain.Settings{
		ID:        1,
		Key:       SettingKeySelfCheckInAdvancePercent,
		Value:     stringPointer("70"),
		ValueType: domain.ValueTypeNumber,
	})
	quotaRepo := &fakeQuotaRepoForUpdate{
		rows:         []*domain.AdvancePayment{{ID: 1, Salary: 1_000_000, MaxAdvAmount: 700_000}},
		recomputeErr: errors.New("boom"),
	}
	service := NewSettingsService(
		settingsRepo,
		quotaRepo,
		nil,
		fakeSettingsTxManager{settings: settingsRepo, quota: quotaRepo},
		nil,
		fakeSettingsEventBus{},
		nil,
	)

	if _, err := service.UpdateSetting(context.Background(), 1, map[string]any{"value": "85"}, 7); err == nil {
		t.Fatal("expected UpdateSetting to fail when quota recompute fails")
	}

	row, err := settingsRepo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if row.Value == nil || *row.Value != "70" {
		t.Fatalf("rolled-back setting value = %v, want 70", row.Value)
	}
	if quotaRepo.rows[0].MaxAdvAmount != 700_000 {
		t.Fatalf("rolled-back max_adv_amount = %d, want 700000", quotaRepo.rows[0].MaxAdvAmount)
	}
}

func TestSettingsServiceUpdateHoldRecalculatesCompletedPendingQuotaDeadlines(t *testing.T) {
	settingsRepo := newFakeSettingsRepoForUpdate(&domain.Settings{
		ID:        1,
		Key:       SettingKeySelfCheckInAdvanceHold,
		Value:     stringPointer("2"),
		ValueType: domain.ValueTypeNumber,
	})
	quotaRepo := &fakeQuotaRepoForUpdate{}
	attendanceRepo := &fakeAttendanceRepoForSettingsUpdate{schedules: []domain.QuotaCreditSchedule{{
		AttendanceID: 42,
		EligibleAt:   time.Date(2026, 8, 16, 19, 0, 0, 0, time.UTC),
	}}}
	tasks := &fakeQuotaCreditTaskEnqueuer{}
	service := NewSettingsService(
		settingsRepo,
		quotaRepo,
		attendanceRepo,
		fakeSettingsTxManager{settings: settingsRepo, quota: quotaRepo},
		nil,
		fakeSettingsEventBus{},
		tasks,
	)

	if _, err := service.UpdateSetting(context.Background(), 1, map[string]any{"value": "4"}, 7); err != nil {
		t.Fatalf("UpdateSetting returned error: %v", err)
	}
	if attendanceRepo.calls != 1 || attendanceRepo.hold != 4*time.Hour {
		t.Fatalf("hold reconciliation = %d calls at %v, want 1 call at 4h", attendanceRepo.calls, attendanceRepo.hold)
	}
	if len(tasks.schedules) != 1 || tasks.schedules[0].AttendanceID != 42 {
		t.Fatalf("recalculated quota task schedules = %+v, want attendance 42", tasks.schedules)
	}
}

func TestSettingsServiceUpdateHoldRollsBackWhenDeadlineRecalculationFails(t *testing.T) {
	settingsRepo := newFakeSettingsRepoForUpdate(&domain.Settings{
		ID:        1,
		Key:       SettingKeySelfCheckInAdvanceHold,
		Value:     stringPointer("2"),
		ValueType: domain.ValueTypeNumber,
	})
	quotaRepo := &fakeQuotaRepoForUpdate{}
	attendanceRepo := &fakeAttendanceRepoForSettingsUpdate{err: errors.New("database unavailable")}
	service := NewSettingsService(
		settingsRepo,
		quotaRepo,
		attendanceRepo,
		fakeSettingsTxManager{settings: settingsRepo, quota: quotaRepo},
		nil,
		fakeSettingsEventBus{},
		nil,
	)

	if _, err := service.UpdateSetting(context.Background(), 1, map[string]any{"value": "4"}, 7); err == nil {
		t.Fatal("expected UpdateSetting to fail when deadline reconciliation fails")
	}
	setting, err := settingsRepo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if setting.Value == nil || *setting.Value != "2" {
		t.Fatalf("setting value = %v, want rollback to 2", setting.Value)
	}
}

// A protected row is returned to clients without its value, so a client that
// PUTs the response back sends a null value. That null must mean "unchanged" for
// the secret and "clear" for an ordinary setting.
func TestSettingsServiceUpdateSettingNullValueSemantics(t *testing.T) {
	const credential = `{"app_id":"123","secret_key":"s3cr3t-key"}`

	settingsRepo := newFakeSettingsRepoForUpdate(
		&domain.Settings{ID: 1, Key: secret.ZaloCredentialsKey, Value: stringPointer(credential), ValueType: domain.ValueTypeJSON},
		&domain.Settings{ID: 2, Key: "transfer_bank_visible", Value: stringPointer("true"), ValueType: domain.ValueTypeString},
	)
	service := NewSettingsService(
		settingsRepo,
		&fakeQuotaRepoForUpdate{},
		nil,
		fakeSettingsTxManager{settings: settingsRepo, quota: &fakeQuotaRepoForUpdate{}},
		nil,
		fakeSettingsEventBus{},
		nil,
	)

	protected, err := service.UpdateSetting(context.Background(), 1, map[string]any{"value": nil}, 7)
	if err != nil {
		t.Fatalf("UpdateSetting(protected, null): %v", err)
	}
	if protected.Value == nil || *protected.Value != credential {
		t.Fatalf("protected value = %v, want the stored credential to survive a null update", protected.Value)
	}

	ordinary, err := service.UpdateSetting(context.Background(), 2, map[string]any{"value": nil}, 7)
	if err != nil {
		t.Fatalf("UpdateSetting(ordinary, null): %v", err)
	}
	if ordinary.Value != nil {
		t.Fatalf("ordinary value = %v, want the null update to clear it", *ordinary.Value)
	}
}

// The settings cache is Redis; caching a decrypted credential would recreate the
// plaintext exposure the repository now seals in MySQL.
func TestSettingsServiceDoesNotCacheProtectedSetting(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb, err := persistence.NewRedisClient(persistence.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatalf("connect miniredis: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	settingsRepo := newFakeSettingsRepoForUpdate(
		&domain.Settings{ID: 1, Key: secret.ZaloCredentialsKey, Value: stringPointer(`{"app_id":"123"}`), ValueType: domain.ValueTypeJSON},
		&domain.Settings{ID: 2, Key: "transfer_bank_visible", Value: stringPointer("true"), ValueType: domain.ValueTypeString},
	)
	service := NewSettingsService(
		settingsRepo,
		nil,
		nil,
		nil,
		infrastructure.NewCacheService(rdb),
		fakeSettingsEventBus{},
		nil,
	)

	if _, err := service.GetSettingByKey(context.Background(), "transfer_bank_visible"); err != nil {
		t.Fatalf("GetSettingByKey(ordinary): %v", err)
	}
	if _, err := service.GetSettingByKey(context.Background(), secret.ZaloCredentialsKey); err != nil {
		t.Fatalf("GetSettingByKey(protected): %v", err)
	}

	keys := mr.Keys()
	if len(keys) != 1 || !strings.Contains(keys[0], "transfer_bank_visible") {
		t.Fatalf("cached keys = %v, want only the ordinary setting", keys)
	}
}
