package config

import (
	"api-server/internal/pkg/clock"
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// Setting keys for business configuration
const (
	SettingKeyWeeklyPaymentPercentage   = "bulk_transfer_payment_percentage" // Reusing existing key for weekly
	SettingKeyMonthlyPaymentPercentage  = "monthly_payment_percentage"
	SettingKeyPartnerCompany            = "partner_company"
	SettingKeyAdvancePaymentPercentage  = "advance_payment_percentage"
	SettingKeyBulkTransferWorkbookLimit = "bulk_transfer_workbook_limit_vnd"
	SettingKeySelfCheckInAdvancePercent = "self_check_in_advance_percentage"
	SettingKeySelfCheckInAdvanceHold    = "self_check_in_advance_hold_hours"
	SettingKeyTransferBankHolder        = "transfer_bank_account_holder"
	SettingKeyTransferBankNumber        = "transfer_bank_account_number"
	SettingKeyTransferBankName          = "transfer_bank_name"
)

// TransferBankInfo carries the beneficiary identity printed on payroll
// statements (email body and Excel attachment).
type TransferBankInfo struct {
	Holder string
	Number string
	Name   string
}

// Default values
const (
	DefaultWeeklyPaymentPercentage   = 0.70
	DefaultMonthlyPaymentPercentage  = 0.70
	DefaultAdvanceCashFeePercentage  = 0.02
	DefaultPartnerCompany            = "VFIC Manpower"
	DefaultAdvancePaymentPercentage  = 0.60  // 60% max advance
	DefaultAdvancePaymentFeeMin      = 10000 // 10,000 VND minimum fee
	DefaultBulkTransferWorkbookLimit = int64(400_000_000)
	DefaultSelfCheckInAdvancePercent = domain.DefaultSelfCheckInAdvancePercentage
	DefaultSelfCheckInAdvanceHold    = domain.QuotaCreditHoldDuration
	MinBulkTransferWorkbookLimit     = int64(2)
	MaxSelfCheckInAdvanceHoldHours   = uint64(720)
	CacheTTL                         = constants.SettingsCacheTTL // Use centralized cache TTL

	// Defaults for the beneficiary bank printed on payroll statements.
	DefaultTransferBankHolder = "CONG TY TNHH MTV GPPM TING TING"
	DefaultTransferBankNumber = "271866699"
	DefaultTransferBankName   = "Ngân hàng Quân đội (MB)"
)

// DefaultTransferBankInfo returns the fallback beneficiary bank details used
// when the settings rows are not configured.
func DefaultTransferBankInfo() TransferBankInfo {
	return TransferBankInfo{
		Holder: DefaultTransferBankHolder,
		Number: DefaultTransferBankNumber,
		Name:   DefaultTransferBankName,
	}
}

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

type SettingReader interface {
	GetSettingByKey(ctx context.Context, key string) (*domain.Settings, error)
	GetSettingByKeyAuthoritative(ctx context.Context, key string) (*domain.Settings, error)
	GetSettingByKeyAuthoritativeForUpdate(ctx context.Context, key string) (*domain.Settings, error)
}

// SettingsConfigService provides methods to retrieve business configuration settings
type SettingsConfigService struct {
	settingsService SettingReader
	cache           sync.Map // key -> *cacheEntry
	feeSchedule     FeeScheduleResolver
}

// NewSettingsConfigService creates a new settings configuration service
func NewSettingsConfigService(settingsService SettingReader) *SettingsConfigService {
	return &SettingsConfigService{
		settingsService: settingsService,
		cache:           sync.Map{},
	}
}

// GetBulkTransferWorkbookLimit returns the strict total threshold for each
// manual MBank Chuyển lô workbook. This value intentionally bypasses the local
// sync.Map and Redis cache-aside layers so a completed Admin update is
// authoritative for the next money-moving export.
func (s *SettingsConfigService) GetBulkTransferWorkbookLimit(ctx context.Context) int64 {
	if s.settingsService == nil {
		return DefaultBulkTransferWorkbookLimit
	}

	setting, err := s.settingsService.GetSettingByKeyAuthoritative(ctx, SettingKeyBulkTransferWorkbookLimit)
	if err != nil {
		observability.GetLogger().Warn(
			"failed to get bulk transfer workbook limit setting, using default",
			"key", SettingKeyBulkTransferWorkbookLimit,
			"error", err,
		)
		return DefaultBulkTransferWorkbookLimit
	}

	value, err := parseBulkTransferWorkbookLimit(setting)
	if err != nil {
		observability.GetLogger().Warn(
			"invalid bulk transfer workbook limit setting, using default",
			"key", SettingKeyBulkTransferWorkbookLimit,
			"error", err,
		)
		return DefaultBulkTransferWorkbookLimit
	}
	return value
}

func parseBulkTransferWorkbookLimit(setting *domain.Settings) (int64, error) {
	if setting == nil || setting.Value == nil {
		return 0, domain.NewValidationError("giới hạn tổng tiền file Chuyển lô không được để trống")
	}
	if setting.ValueType != domain.ValueTypeNumber {
		return 0, domain.NewValidationError("giới hạn tổng tiền file Chuyển lô phải là số")
	}

	raw := *setting.Value
	if raw == "" || strings.TrimSpace(raw) != raw || raw[0] == '0' || strings.IndexFunc(raw, func(r rune) bool {
		return r < '0' || r > '9'
	}) >= 0 {
		return 0, domain.NewValidationError("giới hạn tổng tiền file Chuyển lô phải là số nguyên")
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < MinBulkTransferWorkbookLimit {
		return 0, domain.NewValidationError("giới hạn tổng tiền file Chuyển lô phải là số nguyên từ 2 đ trở lên")
	}
	return value, nil
}

func parseSelfCheckInAdvancePercent(setting *domain.Settings) (uint64, error) {
	if setting == nil || setting.Value == nil {
		return 0, domain.NewValidationError("tỷ lệ ứng lương tự chấm công không được để trống")
	}
	if setting.ValueType != domain.ValueTypeNumber {
		return 0, domain.NewValidationError("tỷ lệ ứng lương tự chấm công phải là số")
	}

	raw := *setting.Value
	if raw == "" || strings.TrimSpace(raw) != raw || raw[0] == '0' || strings.IndexFunc(raw, func(r rune) bool {
		return r < '0' || r > '9'
	}) >= 0 {
		return 0, domain.NewValidationError("tỷ lệ ứng lương tự chấm công phải là số nguyên từ 1 đến 100")
	}

	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value < 1 || value > 100 {
		return 0, domain.NewValidationError("tỷ lệ ứng lương tự chấm công phải là số nguyên từ 1 đến 100")
	}
	return value, nil
}

func parseSelfCheckInAdvanceHoldHours(setting *domain.Settings) (uint64, error) {
	if setting == nil || setting.Value == nil {
		return 0, domain.NewValidationError("thời gian chờ ứng lương tự chấm công không được để trống")
	}
	if setting.ValueType != domain.ValueTypeNumber {
		return 0, domain.NewValidationError("thời gian chờ ứng lương tự chấm công phải là số")
	}

	raw := *setting.Value
	if raw == "" || strings.TrimSpace(raw) != raw || (len(raw) > 1 && raw[0] == '0') || strings.IndexFunc(raw, func(r rune) bool {
		return r < '0' || r > '9'
	}) >= 0 {
		return 0, domain.NewValidationError("thời gian chờ ứng lương tự chấm công phải là số nguyên từ 0 đến 720")
	}

	hours, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || hours > MaxSelfCheckInAdvanceHoldHours {
		return 0, domain.NewValidationError("thời gian chờ ứng lương tự chấm công phải là số nguyên từ 0 đến 720")
	}
	return hours, nil
}

// GetSelfCheckInAdvanceHoldDuration returns the authoritative post-checkout
// holding duration before self-check-in earnings become advanceable. Financial
// scheduling bypasses cache so a completed Admin update governs the next
// checkout and recovery sweep. Missing or invalid persisted data retains the
// established 24-hour behavior.
func (s *SettingsConfigService) GetSelfCheckInAdvanceHoldDuration(ctx context.Context) time.Duration {
	if s.settingsService == nil {
		return DefaultSelfCheckInAdvanceHold
	}

	setting, err := s.settingsService.GetSettingByKeyAuthoritative(ctx, SettingKeySelfCheckInAdvanceHold)
	if err != nil {
		observability.GetLogger().Warn(
			"failed to get self check-in advance hold setting, using default",
			"key", SettingKeySelfCheckInAdvanceHold,
			"error", err,
		)
		return DefaultSelfCheckInAdvanceHold
	}

	hours, err := parseSelfCheckInAdvanceHoldHours(setting)
	if err != nil {
		observability.GetLogger().Warn(
			"invalid self check-in advance hold setting, using default",
			"key", SettingKeySelfCheckInAdvanceHold,
			"error", err,
		)
		return DefaultSelfCheckInAdvanceHold
	}

	return time.Duration(hours) * time.Hour
}

// GetSelfCheckInAdvancePercentage returns the authoritative self-check-in
// advance percentage (1..100). Financial reads bypass cache so a completed
// Admin update applies on the next request; invalid/missing data falls back to
// the long-standing default.
func (s *SettingsConfigService) GetSelfCheckInAdvancePercentage(ctx context.Context) uint64 {
	return s.getSelfCheckInAdvancePercentage(ctx)
}

// GetSelfCheckInAdvancePercentageForUpdate returns the authoritative
// self-check-in advance percentage while locking the settings row inside the
// caller's transaction. Attendance quota credit uses this so salary banking and
// admin edits serialize on the same row lock.
func (s *SettingsConfigService) GetSelfCheckInAdvancePercentageForUpdate(ctx context.Context) (uint64, error) {
	if s.settingsService == nil {
		return DefaultSelfCheckInAdvancePercent, nil
	}
	setting, err := s.settingsService.GetSettingByKeyAuthoritativeForUpdate(ctx, SettingKeySelfCheckInAdvancePercent)
	if err != nil {
		return 0, err
	}
	return parseSelfCheckInAdvancePercent(setting)
}

func (s *SettingsConfigService) getSelfCheckInAdvancePercentage(ctx context.Context) uint64 {
	if s.settingsService == nil {
		return DefaultSelfCheckInAdvancePercent
	}

	setting, err := s.settingsService.GetSettingByKeyAuthoritative(ctx, SettingKeySelfCheckInAdvancePercent)
	if err != nil {
		observability.GetLogger().Warn(
			"failed to get self check-in advance percentage, using default",
			"key", SettingKeySelfCheckInAdvancePercent,
			"error", err,
		)
		return DefaultSelfCheckInAdvancePercent
	}

	value, err := parseSelfCheckInAdvancePercent(setting)
	if err != nil {
		observability.GetLogger().Warn(
			"invalid self check-in advance percentage, using default",
			"key", SettingKeySelfCheckInAdvancePercent,
			"error", err,
		)
		return DefaultSelfCheckInAdvancePercent
	}
	return value
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

// getTransferBankString reads one transfer-bank string setting with caching,
// falling back to defaultValue when the row is missing or empty.
func (s *SettingsConfigService) getTransferBankString(ctx context.Context, key, defaultValue string) string {
	if s.settingsService == nil {
		return defaultValue
	}

	if cached, ok := s.getFromCache(key); ok {
		return cached.(string)
	}

	setting, err := s.settingsService.GetSettingByKey(ctx, key)
	if err != nil {
		observability.GetLogger().Warn("failed to get transfer bank setting, using default", "key", key, "error", err)
		return defaultValue
	}

	value := setting.GetStringValue()
	if value == "" {
		observability.GetLogger().Warn("transfer bank setting is empty, using default", "key", key)
		return defaultValue
	}

	s.setCache(key, value)

	return value
}

// GetTransferBankInfo retrieves the beneficiary bank details printed on
// payroll statement emails and Excel attachments. Unconfigured rows fall back
// to the MB bank defaults.
func (s *SettingsConfigService) GetTransferBankInfo(ctx context.Context) TransferBankInfo {
	return TransferBankInfo{
		Holder: s.getTransferBankString(ctx, SettingKeyTransferBankHolder, DefaultTransferBankHolder),
		Number: s.getTransferBankString(ctx, SettingKeyTransferBankNumber, DefaultTransferBankNumber),
		Name:   s.getTransferBankString(ctx, SettingKeyTransferBankName, DefaultTransferBankName),
	}
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
