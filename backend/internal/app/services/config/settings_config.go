package config

import (
	"api-server/internal/pkg/clock"
	"context"
	"sync"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// Setting keys for business configuration
const (
	SettingKeyWeeklyPaymentPercentage  = "bulk_transfer_payment_percentage" // Reusing existing key for weekly
	SettingKeyMonthlyPaymentPercentage = "monthly_payment_percentage"
	SettingKeyPartnerCompany           = "partner_company"
	SettingKeyAdvancePaymentPercentage = "advance_payment_percentage"
)

// Default values
const (
	DefaultWeeklyPaymentPercentage  = 0.70
	DefaultMonthlyPaymentPercentage = 0.70
	DefaultAdvanceCashFeePercentage = 0.02
	DefaultPartnerCompany           = "VFIC Manpower"
	DefaultAdvancePaymentPercentage = 0.60                       // 60% max advance
	DefaultAdvancePaymentFeeMin     = 10000                      // 10,000 VND minimum fee
	CacheTTL                        = constants.SettingsCacheTTL // Use centralized cache TTL
)

// Validation constants for advance payment
const (
	MinAdvanceRequestAmount = 10000 // 10,000 VND minimum request
)

// cacheEntry stores cached value with expiration time
type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

// FeeScheduleResolver is the read-side surface of the advance-payment fee
// schedule that legacy callers depend on. Implemented by *FeeScheduleService;
// bound at bootstrap via BindFeeScheduleResolver after both services exist
// (FeeScheduleService is created later in the wiring graph).
type FeeScheduleResolver interface {
	FeePercentageAt(ctx context.Context, at time.Time) float64
	MinFeeAt(ctx context.Context, at time.Time) uint64
}

// SettingsConfigService provides methods to retrieve business configuration settings
type SettingsConfigService struct {
	settingsService *SettingsService
	cache           sync.Map // key -> *cacheEntry
	feeSchedule     FeeScheduleResolver
}

// NewSettingsConfigService creates a new settings configuration service
func NewSettingsConfigService(settingsService *SettingsService) *SettingsConfigService {
	return &SettingsConfigService{
		settingsService: settingsService,
		cache:           sync.Map{},
	}
}

// BindFeeScheduleResolver wires the fee-schedule data source. After binding,
// GetAdvanceCashFeePercentage and GetAdvancePaymentFeeMin read from the
// active schedule instead of the legacy flat-rate settings keys (which the
// 044 migration retired). Safe to call once at bootstrap; not goroutine-safe
// for repeated rebinding.
func (s *SettingsConfigService) BindFeeScheduleResolver(r FeeScheduleResolver) {
	s.feeSchedule = r
}

// getFromCache retrieves a value from cache if valid
func (s *SettingsConfigService) getFromCache(key string) (interface{}, bool) {
	if entry, ok := s.cache.Load(key); ok {
		cached := entry.(*cacheEntry)
		if clock.Now().Before(cached.expiration) {
			return cached.value, true
		}
		// Expired, remove from cache
		s.cache.Delete(key)
	}
	return nil, false
}

// setCache stores a value in cache with TTL
func (s *SettingsConfigService) setCache(key string, value interface{}) {
	s.cache.Store(key, &cacheEntry{
		value:      value,
		expiration: clock.Now().Add(CacheTTL),
	})
}

// InvalidateCache clears all cached settings (useful when settings are updated)
func (s *SettingsConfigService) InvalidateCache() {
	s.cache.Range(func(key, value interface{}) bool {
		s.cache.Delete(key)
		return true
	})
}

// GetWeeklyPaymentPercentage retrieves the weekly payment percentage setting (with caching)
// Returns the configured value or default (0.70) if not found or invalid
func (s *SettingsConfigService) GetWeeklyPaymentPercentage(ctx context.Context) float64 {
	// Check cache first
	if cached, ok := s.getFromCache(SettingKeyWeeklyPaymentPercentage); ok {
		return cached.(float64)
	}

	// Cache miss, fetch from database
	setting, err := s.settingsService.GetSettingByKey(ctx, SettingKeyWeeklyPaymentPercentage)
	if err != nil {
		observability.GetLogger().Warn("failed to get weekly payment percentage setting, using default", "key", SettingKeyWeeklyPaymentPercentage, "error", err)
		return DefaultWeeklyPaymentPercentage
	}

	value, err := setting.GetFloatValue()
	if err != nil {
		observability.GetLogger().Warn("failed to parse weekly payment percentage, using default", "key", SettingKeyWeeklyPaymentPercentage, "error", err)
		return DefaultWeeklyPaymentPercentage
	}

	// Validate range (0-1)
	if value < 0 || value > 1 {
		observability.GetLogger().Warn("weekly payment percentage out of range, using default", "key", SettingKeyWeeklyPaymentPercentage, "value", value)
		return DefaultWeeklyPaymentPercentage
	}

	// Store in cache
	s.setCache(SettingKeyWeeklyPaymentPercentage, value)

	return value
}

// GetMonthlyPaymentPercentage retrieves the monthly payment percentage setting (with caching)
// Returns the configured value or default (0.70) if not found or invalid
func (s *SettingsConfigService) GetMonthlyPaymentPercentage(ctx context.Context) float64 {
	// Check cache first
	if cached, ok := s.getFromCache(SettingKeyMonthlyPaymentPercentage); ok {
		return cached.(float64)
	}

	// Cache miss, fetch from database
	setting, err := s.settingsService.GetSettingByKey(ctx, SettingKeyMonthlyPaymentPercentage)
	if err != nil {
		observability.GetLogger().Warn("failed to get monthly payment percentage setting, using default", "key", SettingKeyMonthlyPaymentPercentage, "error", err)
		return DefaultMonthlyPaymentPercentage
	}

	value, err := setting.GetFloatValue()
	if err != nil {
		observability.GetLogger().Warn("failed to parse monthly payment percentage, using default", "key", SettingKeyMonthlyPaymentPercentage, "error", err)
		return DefaultMonthlyPaymentPercentage
	}

	// Validate range (0-1)
	if value < 0 || value > 1 {
		observability.GetLogger().Warn("monthly payment percentage out of range, using default", "key", SettingKeyMonthlyPaymentPercentage, "value", value)
		return DefaultMonthlyPaymentPercentage
	}

	// Store in cache
	s.setCache(SettingKeyMonthlyPaymentPercentage, value)

	return value
}

// GetPaymentPercentageForSchedule retrieves the payment percentage based on payment schedule
func (s *SettingsConfigService) GetPaymentPercentageForSchedule(ctx context.Context, schedule string) float64 {
	switch schedule {
	case string(domain.PaymentScheduleMonthly):
		return s.GetMonthlyPaymentPercentage(ctx)
	case string(domain.PaymentScheduleWeekly):
		return s.GetWeeklyPaymentPercentage(ctx)
	default:
		// Default to weekly if schedule is unknown
		return s.GetWeeklyPaymentPercentage(ctx)
	}
}

// GetAdvanceCashFeePercentage returns the headline (first-tier) advance cash
// fee percentage as a fraction (0.02 == 2%). Once the FeeScheduleResolver is
// bound, this reads the active schedule's first-tier percentage; unbound
// callers fall back to the default.
func (s *SettingsConfigService) GetAdvanceCashFeePercentage(ctx context.Context) float64 {
	if s.feeSchedule != nil {
		return s.feeSchedule.FeePercentageAt(ctx, clock.Now())
	}
	return DefaultAdvanceCashFeePercentage
}

// GetPartnerCompany retrieves the partner company name setting (with caching)
// Returns the configured value or default ("VFIC Manpower") if not found or invalid
func (s *SettingsConfigService) GetPartnerCompany(ctx context.Context) string {
	// Check cache first
	if cached, ok := s.getFromCache(SettingKeyPartnerCompany); ok {
		return cached.(string)
	}

	// Cache miss, fetch from database
	setting, err := s.settingsService.GetSettingByKey(ctx, SettingKeyPartnerCompany)
	if err != nil {
		observability.GetLogger().Warn("failed to get partner company setting, using default", "key", SettingKeyPartnerCompany, "error", err)
		return DefaultPartnerCompany
	}

	value := setting.GetStringValue()
	if value == "" {
		observability.GetLogger().Warn("partner company setting is empty, using default", "key", SettingKeyPartnerCompany)
		return DefaultPartnerCompany
	}

	// Store in cache
	s.setCache(SettingKeyPartnerCompany, value)

	return value
}

// GetAdvancePaymentPercentage retrieves the advance payment percentage setting (with caching)
// Returns the configured value or default (0.60) if not found or invalid
func (s *SettingsConfigService) GetAdvancePaymentPercentage(ctx context.Context) float64 {
	// Check cache first
	if cached, ok := s.getFromCache(SettingKeyAdvancePaymentPercentage); ok {
		return cached.(float64)
	}

	// Cache miss, fetch from database
	setting, err := s.settingsService.GetSettingByKey(ctx, SettingKeyAdvancePaymentPercentage)
	if err != nil {
		return DefaultAdvancePaymentPercentage
	}

	value, err := setting.GetFloatValue()
	if err != nil {
		return DefaultAdvancePaymentPercentage
	}

	// Validate range (0-1)
	if value < 0 || value > 1 {
		return DefaultAdvancePaymentPercentage
	}

	// Store in cache
	s.setCache(SettingKeyAdvancePaymentPercentage, value)

	return value
}

// GetAdvancePaymentFeeMin returns the minimum advance payment fee in VND from
// the active fee schedule. Falls back to the default when no resolver is
// bound (boot order / tests).
func (s *SettingsConfigService) GetAdvancePaymentFeeMin(ctx context.Context) uint64 {
	if s.feeSchedule != nil {
		return s.feeSchedule.MinFeeAt(ctx, clock.Now())
	}
	return DefaultAdvancePaymentFeeMin
}
