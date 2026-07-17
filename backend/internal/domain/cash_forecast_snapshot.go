package domain

import (
	"context"
	"time"
)

const (
	// CashForecastCompanyScope is the only scope eligible for production
	// accuracy measurement. Filtered forecasts must never be reconciled against
	// a company-wide transfer export.
	CashForecastCompanyScope = "company"

	CashForecastOutcomeWeeklyExportDistinctTimesheets = "weekly_bulk_transfer_distinct_timesheets"
)

// CashForecastOutcomeItem is one validated timesheet included in a generated
// weekly bank file. TimesheetID deduplicates retries and split exports.
type CashForecastOutcomeItem struct {
	TimesheetID uint
	Amount      int64
}

// CashForecastSnapshot is one company-wide forecast made at a particular
// cycle-day decision horizon. All monetary values are integer VND.
type CashForecastSnapshot struct {
	ID                       uint64     `json:"id" gorm:"primaryKey;type:bigint unsigned"`
	ScopeKey                 string     `json:"scope_key" gorm:"type:varchar(64);not null;uniqueIndex:uq_cash_forecast_snapshot_identity"`
	CycleKey                 string     `json:"cycle_key" gorm:"type:varchar(32);not null;uniqueIndex:uq_cash_forecast_snapshot_identity"`
	CycleDay                 int        `json:"cycle_day" gorm:"type:smallint;not null;uniqueIndex:uq_cash_forecast_snapshot_identity"`
	ModelVersion             string     `json:"model_version" gorm:"type:varchar(64);not null;uniqueIndex:uq_cash_forecast_snapshot_identity"`
	TargetFromDate           time.Time  `json:"target_from_date" gorm:"type:date;not null"`
	TargetToDate             time.Time  `json:"target_to_date" gorm:"type:date;not null"`
	HorizonDays              int        `json:"horizon_days" gorm:"type:smallint;not null;index:idx_cash_forecast_resolved_horizon"`
	ObservedApprovedAmount   int64      `json:"observed_approved_amount" gorm:"not null"`
	PendingAmount            int64      `json:"pending_amount" gorm:"not null"`
	ExpectedPendingAmount    int64      `json:"expected_pending_amount" gorm:"not null"`
	ExpectedFutureAmount     int64      `json:"expected_future_amount" gorm:"not null"`
	ExpectedPayoutAmount     int64      `json:"expected_payout_amount" gorm:"not null"`
	RecommendedReserveAmount int64      `json:"recommended_reserve_amount" gorm:"not null"`
	IntervalLowerAmount      int64      `json:"interval_lower_amount" gorm:"not null"`
	IntervalUpperAmount      int64      `json:"interval_upper_amount" gorm:"not null"`
	GeneratedAt              time.Time  `json:"generated_at" gorm:"not null"`
	ActualPayoutAmount       *int64     `json:"actual_payout_amount,omitempty"`
	OutcomeSource            *string    `json:"outcome_source,omitempty" gorm:"type:varchar(64)"`
	ResolvedAt               *time.Time `json:"resolved_at,omitempty" gorm:"index:idx_cash_forecast_resolved_horizon"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

func (CashForecastSnapshot) TableName() string { return "cash_forecast_snapshots" }

// CashForecastResolvedQuery selects comparable, resolved production outcomes.
// HorizonDays is exact: calibration must compare forecasts made at the same
// decision horizon.
type CashForecastResolvedQuery struct {
	ScopeKey     string
	ModelVersion string
	HorizonDays  int
	Limit        int
}

// CashForecastResidual is the calibration input for one resolved snapshot.
// ResidualAmount is actual minus expected; positive values mean underforecast.
type CashForecastResidual struct {
	CycleKey                 string
	CycleDay                 int
	HorizonDays              int
	ExpectedPayoutAmount     int64
	RecommendedReserveAmount int64
	IntervalLowerAmount      int64
	IntervalUpperAmount      int64
	ActualPayoutAmount       int64
	ResidualAmount           int64
	GeneratedAt              time.Time
	ResolvedAt               time.Time
}

// CashForecastAccuracy summarizes resolved forecasts at one exact horizon.
type CashForecastAccuracy struct {
	SampleCount          int
	MeanAbsoluteError    int64
	WAPE                 float64
	Bias                 float64
	ReserveShortfallRate float64
	IntervalCoverage     float64
}

// CashForecastOutcomeResolver is kept narrow for the weekly export event
// handler and can later be replaced by a timesheet-ID-deduplicated resolver.
type CashForecastOutcomeResolver interface {
	ResolveWeeklyExport(ctx context.Context, fromDate, toDate time.Time, items []CashForecastOutcomeItem, outcomeSource string) (int64, error)
}

// CashForecastSnapshotRepository persists and reads measurement data only. It
// has no wallet mutation or disbursement capability.
type CashForecastSnapshotRepository interface {
	CashForecastOutcomeResolver
	Upsert(ctx context.Context, snapshot *CashForecastSnapshot) error
	ListResolvedResiduals(ctx context.Context, query CashForecastResolvedQuery) ([]CashForecastResidual, error)
	GetAccuracy(ctx context.Context, query CashForecastResolvedQuery) (*CashForecastAccuracy, error)
}
