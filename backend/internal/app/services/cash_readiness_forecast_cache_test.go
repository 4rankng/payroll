package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
)

// cacheFake embeds the interface so only the methods the forecast touches need
// real implementations; any other call would panic on a nil embedded method.
type cacheFake struct {
	domain.CacheServiceUseCase
	store map[string]domain.CashReadiness
	sets  int
}

func (f *cacheFake) Get(_ context.Context, key string, dest interface{}) error {
	v, ok := f.store[key]
	if !ok {
		return errors.New("cache miss")
	}
	*dest.(*domain.CashReadiness) = v
	return nil
}

func (f *cacheFake) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	f.store[key] = *value.(*domain.CashReadiness)
	f.sets++
	return nil
}

func (f *cacheFake) GenerateDashboardCacheKey(operation string, params ...string) string {
	return "dashboard:" + operation
}

// TestGetCashReadiness_ServesRepeatedReadsFromCache verifies the advisory
// forecast is recomputed once per scope and later reads hit the cache — the
// months-long cohort scan must not run per request.
func TestGetCashReadiness_ServesRepeatedReadsFromCache(t *testing.T) {
	reader := &fakeTimesheetReader{cohort: []domain.TimesheetAccrualDailyRow{
		forecastRow("2026-07-02", "2026-07-01", "2026-07-02", domain.TimesheetStatusApproved, 1, 10, 100),
	}}
	measurement := &fakeCashMeasurement{}
	cache := &cacheFake{store: map[string]domain.CashReadiness{}}

	svc := NewCashReadinessForecastService(
		reader, &fakeWallet{balance: &wallet.WalletBalance{Available: 1_000_000}},
		NewTimesheetAccrualProvider(), clock.NewFake(july3), config.CashForecastConfig{}, nil, measurement,
	).WithCacheService(cache)

	empty := domain.TimesheetFilters{}
	if _, err := svc.GetCashReadiness(context.Background(), empty); err != nil {
		t.Fatalf("first read: %v", err)
	}
	first, err := svc.GetCashReadiness(context.Background(), empty)
	if err != nil {
		t.Fatalf("second read: %v", err)
	}

	if reader.cohortCalls != 1 {
		t.Fatalf("cohort scan ran %d times, want 1 (second read must be a cache hit)", reader.cohortCalls)
	}
	if len(measurement.upserts) != 1 {
		t.Fatalf("snapshot persisted %d times, want 1 (cache hits must not rewrite)", len(measurement.upserts))
	}
	if cache.sets != 1 {
		t.Fatalf("cache populated %d times, want 1", cache.sets)
	}
	if first == nil {
		t.Fatal("cached read returned nil")
	}
}

// TestGetCashReadiness_ScopesDoNotShareCacheEntries verifies two different
// filter scopes produce two distinct cache keys — a partner scoped forecast
// must never be served the company-wide projection.
func TestGetCashReadiness_ScopesDoNotShareCacheEntries(t *testing.T) {
	reader := &fakeTimesheetReader{cohort: []domain.TimesheetAccrualDailyRow{
		forecastRow("2026-07-02", "2026-07-01", "2026-07-02", domain.TimesheetStatusApproved, 1, 10, 100),
	}}

	svc := NewCashReadinessForecastService(
		reader, &fakeWallet{balance: &wallet.WalletBalance{Available: 1_000_000}},
		NewTimesheetAccrualProvider(), clock.NewFake(july3), config.CashForecastConfig{}, nil,
	).WithCacheService(&cacheFake{store: map[string]domain.CashReadiness{}})

	company := domain.TimesheetFilters{}
	scoped := domain.TimesheetFilters{ProjectIDs: []uint{7}}
	if _, err := svc.GetCashReadiness(context.Background(), company); err != nil {
		t.Fatalf("company read: %v", err)
	}
	if _, err := svc.GetCashReadiness(context.Background(), scoped); err != nil {
		t.Fatalf("scoped read: %v", err)
	}

	if reader.cohortCalls != 2 {
		t.Fatalf("cohort scan ran %d times, want 2 (scopes must not share an entry)", reader.cohortCalls)
	}
}
