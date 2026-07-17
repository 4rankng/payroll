package persistence

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCashForecastTestRepository(t *testing.T) (*CashForecastSnapshotRepository, *gorm.DB) {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000&_time_format=sqlite"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&domain.CashForecastSnapshot{}, &cashForecastOutcomeItemRow{}); err != nil {
		t.Fatalf("migrate snapshot: %v", err)
	}
	return NewCashForecastSnapshotRepository(db), db
}

func cashForecastTestDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, clock.DefaultLocation)
}

func cashForecastTestSnapshot(cycleDay int, expected int64) *domain.CashForecastSnapshot {
	return &domain.CashForecastSnapshot{
		ScopeKey:                 domain.CashForecastCompanyScope,
		CycleKey:                 "2026-07-ky1",
		CycleDay:                 cycleDay,
		ModelVersion:             "cash-v2",
		TargetFromDate:           cashForecastTestDate(2026, time.July, 1),
		TargetToDate:             cashForecastTestDate(2026, time.July, 7),
		HorizonDays:              3,
		ObservedApprovedAmount:   1_000,
		PendingAmount:            500,
		ExpectedPendingAmount:    300,
		ExpectedFutureAmount:     200,
		ExpectedPayoutAmount:     expected,
		RecommendedReserveAmount: 1_700,
		IntervalLowerAmount:      1_200,
		IntervalUpperAmount:      1_800,
		GeneratedAt:              cashForecastTestDate(2026, time.July, 4).Add(12 * time.Hour),
	}
}

func TestCashForecastSnapshotUpsertIsIdempotentAndPreservesOutcome(t *testing.T) {
	repo, db := newCashForecastTestRepository(t)
	ctx := context.Background()

	first := cashForecastTestSnapshot(4, 1_500)
	if err := repo.Upsert(ctx, first); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	refreshed := cashForecastTestSnapshot(4, 1_525)
	if err := repo.Upsert(ctx, refreshed); err != nil {
		t.Fatalf("unresolved refresh: %v", err)
	}
	if _, err := repo.ResolveWeeklyExport(ctx, first.TargetFromDate, first.TargetToDate, []domain.CashForecastOutcomeItem{{TimesheetID: 1, Amount: 1_600}}, domain.CashForecastOutcomeWeeklyExportDistinctTimesheets); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	second := cashForecastTestSnapshot(4, 1_550)
	if err := repo.Upsert(ctx, second); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	var count int64
	if err := db.Model(&domain.CashForecastSnapshot{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("row count = %d, want 1", count)
	}
	var stored domain.CashForecastSnapshot
	if err := db.First(&stored).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if stored.ExpectedPayoutAmount != 1_525 {
		t.Fatalf("expected payout = %d, want immutable resolved value 1525", stored.ExpectedPayoutAmount)
	}
	if stored.ActualPayoutAmount == nil || *stored.ActualPayoutAmount != 1_600 {
		t.Fatalf("resolved outcome was cleared: %v", stored.ActualPayoutAmount)
	}
}

func TestCashForecastSnapshotRejectsFilteredMeasurementScope(t *testing.T) {
	repo, _ := newCashForecastTestRepository(t)
	snapshot := cashForecastTestSnapshot(4, 1_500)
	snapshot.ScopeKey = "project:42"
	if err := repo.Upsert(context.Background(), snapshot); err == nil {
		t.Fatal("filtered snapshot scope should not be persisted as company accuracy")
	}
	if _, err := repo.ListResolvedResiduals(context.Background(), domain.CashForecastResolvedQuery{
		ScopeKey: "project:42", ModelVersion: "cash-v2", HorizonDays: 3,
	}); err == nil {
		t.Fatal("filtered snapshot scope should not be read as company accuracy")
	}
}

func TestCashForecastResolveWeeklyExportAccumulatesDistinctTimesheets(t *testing.T) {
	repo, db := newCashForecastTestRepository(t)
	ctx := context.Background()
	matching := cashForecastTestSnapshot(4, 1_500)
	if err := repo.Upsert(ctx, matching); err != nil {
		t.Fatalf("upsert matching: %v", err)
	}
	nonMatching := cashForecastTestSnapshot(5, 2_000)
	nonMatching.CycleKey = "2026-07-ky2"
	nonMatching.TargetFromDate = cashForecastTestDate(2026, time.July, 8)
	nonMatching.TargetToDate = cashForecastTestDate(2026, time.July, 14)
	if err := repo.Upsert(ctx, nonMatching); err != nil {
		t.Fatalf("upsert nonmatching: %v", err)
	}

	for _, items := range [][]domain.CashForecastOutcomeItem{
		{{TimesheetID: 11, Amount: 1_000}},
		{{TimesheetID: 12, Amount: 500}},
		{{TimesheetID: 11, Amount: 1_000}}, // retry must not double-count
	} {
		if _, err := repo.ResolveWeeklyExport(ctx, matching.TargetFromDate, matching.TargetToDate, items, domain.CashForecastOutcomeWeeklyExportDistinctTimesheets); err != nil {
			t.Fatalf("resolve %#v: %v", items, err)
		}
	}
	var rows []domain.CashForecastSnapshot
	if err := db.Order("cycle_day").Find(&rows).Error; err != nil {
		t.Fatalf("load rows: %v", err)
	}
	if rows[0].ActualPayoutAmount == nil || *rows[0].ActualPayoutAmount != 1_500 {
		t.Fatalf("matching actual = %v, want distinct cumulative 1500", rows[0].ActualPayoutAmount)
	}
	if rows[0].OutcomeSource == nil || *rows[0].OutcomeSource != domain.CashForecastOutcomeWeeklyExportDistinctTimesheets {
		t.Fatalf("missing explicit outcome source: %v", rows[0].OutcomeSource)
	}
	if rows[1].ActualPayoutAmount != nil {
		t.Fatalf("different exact range was resolved: %v", *rows[1].ActualPayoutAmount)
	}
}

func TestCashForecastResidualsAndAccuracyUseExactHorizon(t *testing.T) {
	repo, _ := newCashForecastTestRepository(t)
	ctx := context.Background()

	first := cashForecastTestSnapshot(3, 900)
	first.RecommendedReserveAmount = 1_050
	first.IntervalLowerAmount = 800
	first.IntervalUpperAmount = 1_200
	if err := repo.Upsert(ctx, first); err != nil {
		t.Fatalf("upsert first: %v", err)
	}
	second := cashForecastTestSnapshot(4, 1_200)
	second.CycleKey = "2026-08-ky1"
	second.TargetFromDate = cashForecastTestDate(2026, time.August, 1)
	second.TargetToDate = cashForecastTestDate(2026, time.August, 7)
	second.RecommendedReserveAmount = 1_100
	second.IntervalLowerAmount = 1_000
	second.IntervalUpperAmount = 1_250
	if err := repo.Upsert(ctx, second); err != nil {
		t.Fatalf("upsert second: %v", err)
	}
	otherHorizon := cashForecastTestSnapshot(5, 9_999)
	otherHorizon.CycleKey = "2026-09-ky1"
	otherHorizon.HorizonDays = 2
	otherHorizon.TargetFromDate = cashForecastTestDate(2026, time.September, 1)
	otherHorizon.TargetToDate = cashForecastTestDate(2026, time.September, 7)
	if err := repo.Upsert(ctx, otherHorizon); err != nil {
		t.Fatalf("upsert other horizon: %v", err)
	}

	for index, tc := range []struct {
		from, to time.Time
		actual   int64
	}{
		{first.TargetFromDate, first.TargetToDate, 1_000},
		{second.TargetFromDate, second.TargetToDate, 1_300},
		{otherHorizon.TargetFromDate, otherHorizon.TargetToDate, 10_000},
	} {
		items := []domain.CashForecastOutcomeItem{{TimesheetID: uint(index + 100), Amount: tc.actual}}
		if _, err := repo.ResolveWeeklyExport(ctx, tc.from, tc.to, items, domain.CashForecastOutcomeWeeklyExportDistinctTimesheets); err != nil {
			t.Fatalf("resolve: %v", err)
		}
	}

	query := domain.CashForecastResolvedQuery{
		ScopeKey:     domain.CashForecastCompanyScope,
		ModelVersion: "cash-v2",
		HorizonDays:  3,
		Limit:        10,
	}
	residuals, err := repo.ListResolvedResiduals(ctx, query)
	if err != nil {
		t.Fatalf("residuals: %v", err)
	}
	if len(residuals) != 2 || residuals[0].ResidualAmount+residuals[1].ResidualAmount != 200 {
		t.Fatalf("unexpected residuals: %+v", residuals)
	}
	accuracy, err := repo.GetAccuracy(ctx, query)
	if err != nil {
		t.Fatalf("accuracy: %v", err)
	}
	if accuracy.SampleCount != 2 || accuracy.MeanAbsoluteError != 100 {
		t.Fatalf("unexpected accuracy counts: %+v", accuracy)
	}
	if accuracy.WAPE < 0.0869 || accuracy.WAPE > 0.0870 {
		t.Fatalf("WAPE = %f, want about 0.08696", accuracy.WAPE)
	}
	if accuracy.ReserveShortfallRate != 0.5 || accuracy.IntervalCoverage != 0.5 {
		t.Fatalf("unexpected rates: %+v", accuracy)
	}
}
