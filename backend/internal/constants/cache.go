package constants

import "time"

// Cache TTL constants for all services
// Transaction, Settlement, and Ledger related caches use 3 seconds for near real-time updates
const (
	// Transaction Cache TTLs (3 seconds for real-time updates)
	TransactionDetailCacheTTL = 3 * time.Second
	TransactionListCacheTTL   = 3 * time.Second

	// Settlement Cache TTLs (3 seconds for real-time updates)
	SettlementDetailCacheTTL = 3 * time.Second
	SettlementListCacheTTL   = 3 * time.Second

	// Ledger Cache TTLs (3 seconds for real-time updates)
	LedgerSummaryCacheTTL = 3 * time.Second
	LedgerEntriesCacheTTL = 3 * time.Second
	LedgerBalanceCacheTTL = 3 * time.Second

	// Asset/Upload Cache TTLs (3 seconds for real-time updates)
	AssetListCacheTTL   = 3 * time.Second
	AssetDetailCacheTTL = 3 * time.Second
	SettingsCacheTTL    = 3 * time.Second

	// Dashboard Cache TTLs (5 minutes for advisory dashboard aggregates — they
	// only change when timesheets/settlements mutate, and every mutation path
	// already invalidates "dashboard:*" via the cache invalidation handler, so
	// the TTL only bounds rebuild frequency for unrelated traffic)
	DashboardSummaryCacheTTL       = 5 * time.Minute
	DashboardFinancialCacheTTL     = 5 * time.Minute
	DashboardSalaryDistributionTTL = 1 * time.Hour // Salary distribution changes less frequently
	CashFlowSummaryCacheTTL        = 15 * time.Second
	CheckInHealthCacheTTL          = 60 * time.Second // Health metrics need fresh data
)

// Other constants
const (
	// Timesheet Cache TTLs (15 seconds for timesheet data)
	TimesheetSummaryCacheTTL = 15 * time.Second
	TimesheetListCacheTTL    = 15 * time.Second

	// Employee/Project Cache TTLs (15 seconds for reference data)
	EmployeeListCacheTTL = 15 * time.Second
	// ProjectListCacheTTL is longer than the employee list TTL: projects change
	// rarely and every mutation path (CRUD, assignment changes, cron auto
	// activate/complete) publishes a domain event that the cache invalidation
	// handler maps to "projects:list*" / "projects:count*", so staleness is
	// bounded by event coverage rather than the TTL alone.
	ProjectListCacheTTL = 60 * time.Second
	// AdBannerCacheTTL bounds the employee ad-banner resolve cache. Campaign
	// volume is tiny (tens), so a single live-list entry serves everyone and
	// staleness after an admin edit is at most one TTL.
	AdBannerCacheTTL = 60 * time.Second

	// Employee Summary Cache TTLs (different intervals based on data volatility)
	EmployeeTimesheetSummaryCacheTTL = 5 * time.Minute  // Timesheet data changes frequently
	EmployeePayrollSummaryCacheTTL   = 15 * time.Minute // Payroll data is more stable

	// Validation Cache TTLs (optimized for validation operations)
	EmployeeAssignmentCacheTTL = 5 * time.Minute  // Employee assignment validation data
	PayrateCacheTTL            = 10 * time.Minute // Payrate validation data

	// Idempotency Cache TTLs (longer duration for request deduplication)
	IdempotencyKeyTTL    = 7 * 24 * time.Hour // 7 days for idempotency keys
	IdempotencyResultTTL = 1 * time.Hour      // 1 hour for idempotency results
	IdempotencyLockTTL   = 5 * time.Minute    // 5 minutes for processing lock

	// General Cache TTLs
	DefaultShortCacheTTL  = 3 * time.Second  // For real-time data
	DefaultMediumCacheTTL = 15 * time.Second // For dashboard data
	DefaultLongCacheTTL   = 5 * time.Minute  // For static configuration
)
