package persistence

import (
	"context"
	"fmt"
	"math"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CashForecastSnapshotRepository struct {
	db *gorm.DB
}

type cashForecastOutcomeItemRow struct {
	ID             uint64    `gorm:"primaryKey;type:bigint unsigned"`
	TargetFromDate time.Time `gorm:"type:date;not null;uniqueIndex:uq_cash_forecast_outcome_item"`
	TargetToDate   time.Time `gorm:"type:date;not null;uniqueIndex:uq_cash_forecast_outcome_item"`
	TimesheetID    uint      `gorm:"not null;uniqueIndex:uq_cash_forecast_outcome_item"`
	Amount         int64     `gorm:"not null"`
	OutcomeSource  string    `gorm:"type:varchar(64);not null"`
	CreatedAt      time.Time
}

func (cashForecastOutcomeItemRow) TableName() string { return "cash_forecast_outcome_items" }

func NewCashForecastSnapshotRepository(db *gorm.DB) *CashForecastSnapshotRepository {
	return &CashForecastSnapshotRepository{db: db}
}

// Upsert refreshes the forecast values for one identity without clearing an
// outcome that may already have been resolved by an export event.
func (r *CashForecastSnapshotRepository) Upsert(ctx context.Context, snapshot *domain.CashForecastSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("cash forecast snapshot is nil")
	}
	if snapshot.ScopeKey != domain.CashForecastCompanyScope {
		return fmt.Errorf("cash forecast snapshot scope must be %q", domain.CashForecastCompanyScope)
	}
	if snapshot.CycleKey == "" || snapshot.ModelVersion == "" {
		return fmt.Errorf("cash forecast snapshot identity is incomplete")
	}
	snapshot.TargetFromDate = dateOnly(snapshot.TargetFromDate)
	snapshot.TargetToDate = dateOnly(snapshot.TargetToDate)
	if snapshot.TargetFromDate.After(snapshot.TargetToDate) {
		return fmt.Errorf("cash forecast snapshot target range is inverted")
	}

	identity := map[string]any{
		"scope_key": snapshot.ScopeKey, "cycle_key": snapshot.CycleKey,
		"cycle_day": snapshot.CycleDay, "model_version": snapshot.ModelVersion,
	}
	updates := map[string]any{
		"target_from_date": snapshot.TargetFromDate, "target_to_date": snapshot.TargetToDate,
		"horizon_days": snapshot.HorizonDays, "observed_approved_amount": snapshot.ObservedApprovedAmount,
		"pending_amount": snapshot.PendingAmount, "expected_pending_amount": snapshot.ExpectedPendingAmount,
		"expected_future_amount": snapshot.ExpectedFutureAmount, "expected_payout_amount": snapshot.ExpectedPayoutAmount,
		"recommended_reserve_amount": snapshot.RecommendedReserveAmount,
		"interval_lower_amount":      snapshot.IntervalLowerAmount, "interval_upper_amount": snapshot.IntervalUpperAmount,
		"generated_at": snapshot.GeneratedAt,
	}

	// Refresh only unresolved measurements. Once an actual is attached, the
	// prediction is immutable so later dashboard reads cannot rewrite history.
	updateUnresolved := func() *gorm.DB {
		return r.db.WithContext(ctx).Model(&domain.CashForecastSnapshot{}).
			Where(identity).Where("actual_payout_amount IS NULL").Updates(updates)
	}
	if result := updateUnresolved(); result.Error != nil {
		return fmt.Errorf("update unresolved cash forecast snapshot: %w", result.Error)
	} else if result.RowsAffected > 0 {
		return nil
	}

	created := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(snapshot)
	if created.Error != nil {
		return fmt.Errorf("upsert cash forecast snapshot: %w", created.Error)
	}
	if created.RowsAffected > 0 {
		return nil
	}
	// A concurrent unresolved insert may have won after our first update. One
	// retry refreshes it; a resolved row intentionally remains untouched.
	if result := updateUnresolved(); result.Error != nil {
		return fmt.Errorf("retry unresolved cash forecast snapshot update: %w", result.Error)
	}
	return nil
}

func (r *CashForecastSnapshotRepository) ListResolvedResiduals(ctx context.Context, query domain.CashForecastResolvedQuery) ([]domain.CashForecastResidual, error) {
	rows, err := r.listResolved(ctx, query)
	if err != nil {
		return nil, err
	}

	result := make([]domain.CashForecastResidual, 0, len(rows))
	for _, row := range rows {
		if row.ActualPayoutAmount == nil || row.ResolvedAt == nil {
			continue
		}
		result = append(result, domain.CashForecastResidual{
			CycleKey:                 row.CycleKey,
			CycleDay:                 row.CycleDay,
			HorizonDays:              row.HorizonDays,
			ExpectedPayoutAmount:     row.ExpectedPayoutAmount,
			RecommendedReserveAmount: row.RecommendedReserveAmount,
			IntervalLowerAmount:      row.IntervalLowerAmount,
			IntervalUpperAmount:      row.IntervalUpperAmount,
			ActualPayoutAmount:       *row.ActualPayoutAmount,
			ResidualAmount:           *row.ActualPayoutAmount - row.ExpectedPayoutAmount,
			GeneratedAt:              row.GeneratedAt,
			ResolvedAt:               *row.ResolvedAt,
		})
	}
	return result, nil
}

func (r *CashForecastSnapshotRepository) GetAccuracy(ctx context.Context, query domain.CashForecastResolvedQuery) (*domain.CashForecastAccuracy, error) {
	residuals, err := r.ListResolvedResiduals(ctx, query)
	if err != nil {
		return nil, err
	}
	accuracy := &domain.CashForecastAccuracy{SampleCount: len(residuals)}
	if len(residuals) == 0 {
		return accuracy, nil
	}

	var absoluteError, actualTotal, signedForecastError int64
	var reserveShortfalls, intervalHits int
	for _, row := range residuals {
		absoluteError += absInt64(row.ResidualAmount)
		actualTotal += row.ActualPayoutAmount
		signedForecastError += row.ExpectedPayoutAmount - row.ActualPayoutAmount
		if row.ActualPayoutAmount > row.RecommendedReserveAmount {
			reserveShortfalls++
		}
		if row.ActualPayoutAmount >= row.IntervalLowerAmount && row.ActualPayoutAmount <= row.IntervalUpperAmount {
			intervalHits++
		}
	}
	accuracy.MeanAbsoluteError = int64(math.Round(float64(absoluteError) / float64(len(residuals))))
	if actualTotal > 0 {
		accuracy.WAPE = float64(absoluteError) / float64(actualTotal)
		accuracy.Bias = float64(signedForecastError) / float64(actualTotal)
	}
	accuracy.ReserveShortfallRate = float64(reserveShortfalls) / float64(len(residuals))
	accuracy.IntervalCoverage = float64(intervalHits) / float64(len(residuals))
	return accuracy, nil
}

// ResolveWeeklyExport accumulates unique exported timesheets for the exact work
// date range. Split files add their distinct rows; retries are ignored by the
// unique range/timesheet key, preventing both undercount and double-counting.
func (r *CashForecastSnapshotRepository) ResolveWeeklyExport(ctx context.Context, fromDate, toDate time.Time, items []domain.CashForecastOutcomeItem, outcomeSource string) (int64, error) {
	fromDate = dateOnly(fromDate)
	toDate = dateOnly(toDate)
	if fromDate.After(toDate) {
		return 0, fmt.Errorf("cash forecast outcome range is inverted")
	}
	if outcomeSource == "" {
		return 0, fmt.Errorf("cash forecast outcome source is required")
	}
	if len(items) == 0 {
		return 0, nil
	}
	rows := make([]cashForecastOutcomeItemRow, 0, len(items))
	for _, item := range items {
		if item.TimesheetID == 0 || item.Amount <= 0 {
			return 0, fmt.Errorf("cash forecast outcome item must have a positive timesheet id and amount")
		}
		rows = append(rows, cashForecastOutcomeItemRow{
			TargetFromDate: fromDate, TargetToDate: toDate,
			TimesheetID: item.TimesheetID, Amount: item.Amount, OutcomeSource: outcomeSource,
		})
	}

	now := clock.Now()
	var affected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize split-file outcomes for the same target range. Without this
		// lock, concurrent exports could each observe only their own inserted
		// items and leave the snapshot below the final distinct total.
		var lockedScopes []string
		if err := tx.Model(&domain.CashForecastSnapshot{}).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("scope_key = ? AND target_from_date = ? AND target_to_date = ?", domain.CashForecastCompanyScope, fromDate, toDate).
			Pluck("scope_key", &lockedScopes).Error; err != nil {
			return fmt.Errorf("lock cash forecast snapshots for outcome: %w", err)
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
			return fmt.Errorf("insert distinct cash forecast outcome items: %w", err)
		}
		var actualAmount int64
		if err := tx.Model(&cashForecastOutcomeItemRow{}).
			Where("target_from_date = ? AND target_to_date = ?", fromDate, toDate).
			Select("COALESCE(SUM(amount), 0)").Scan(&actualAmount).Error; err != nil {
			return fmt.Errorf("sum distinct cash forecast outcome items: %w", err)
		}
		result := tx.Model(&domain.CashForecastSnapshot{}).
			Where("scope_key = ? AND target_from_date = ? AND target_to_date = ?", domain.CashForecastCompanyScope, fromDate, toDate).
			Where("actual_payout_amount IS NULL OR actual_payout_amount < ?", actualAmount).
			Updates(map[string]any{
				"actual_payout_amount": actualAmount,
				"outcome_source":       outcomeSource,
				"resolved_at":          now,
				"updated_at":           now,
			})
		if result.Error != nil {
			return fmt.Errorf("resolve weekly cash forecast outcome: %w", result.Error)
		}
		affected = result.RowsAffected
		return nil
	})
	return affected, err
}

func (r *CashForecastSnapshotRepository) listResolved(ctx context.Context, query domain.CashForecastResolvedQuery) ([]domain.CashForecastSnapshot, error) {
	if query.ScopeKey == "" {
		query.ScopeKey = domain.CashForecastCompanyScope
	}
	if query.ScopeKey != domain.CashForecastCompanyScope {
		return nil, fmt.Errorf("cash forecast accuracy scope must be %q", domain.CashForecastCompanyScope)
	}
	if query.Limit <= 0 {
		query.Limit = 52
	}
	db := r.db.WithContext(ctx).
		Where("scope_key = ? AND model_version = ? AND horizon_days = ?", query.ScopeKey, query.ModelVersion, query.HorizonDays).
		Where("actual_payout_amount IS NOT NULL AND resolved_at IS NOT NULL").
		Order("generated_at DESC").
		Limit(query.Limit)
	var rows []domain.CashForecastSnapshot
	if err := db.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list resolved cash forecast snapshots: %w", err)
	}
	return rows, nil
}

func dateOnly(value time.Time) time.Time {
	local := value.In(clock.DefaultLocation)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, clock.DefaultLocation)
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

var _ domain.CashForecastSnapshotRepository = (*CashForecastSnapshotRepository)(nil)
