package config

import (
	"api-server/internal/pkg/clock"
	"context"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	llmport "api-server/internal/infra/llm"
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
	SettingKeyWalletBalanceAlert        = "wallet_balance_alert_threshold_vnd"
	SettingKeyTransferBankHolder        = "transfer_bank_account_holder"
	SettingKeyTransferBankNumber        = "transfer_bank_account_number"
	SettingKeyTransferBankName          = "transfer_bank_name"
	SettingKeyTransferBankVisible       = "transfer_bank_visible"

	// FlexPay (OnePay early-salary) reconciliation statements print their own
	// beneficiary account, configured separately from the weekly payroll
	// statement account above.
	SettingKeyFlexPayTransferBankHolder  = "flexpay_transfer_bank_account_holder"
	SettingKeyFlexPayTransferBankNumber  = "flexpay_transfer_bank_account_number"
	SettingKeyFlexPayTransferBankName    = "flexpay_transfer_bank_name"
	SettingKeyFlexPayTransferBankVisible = "flexpay_transfer_bank_visible"

	// MiMo LLM assistant settings. SettingKeyMimoAPIKey is sealed at rest and
	// masked in every API response — its literal twin is secret.MimoAPIKey
	// (pinned by TestMimoSettingsKeysMatchSecretPackage). The model keys are
	// runtime overrides over the built-in allowlist so a vendor model rename
	// costs a settings edit, not a redeploy.
	SettingKeyMimoAPIKey        = "mimo.api_key"
	SettingKeyMimoBaseURL       = "mimo.base_url"
	SettingKeyMimoDefaultModel  = "mimo.default_model"
	SettingKeyMimoFallbackModel = "mimo.fallback_model"
	// Per-role chat surface toggles (admin-controlled): missing rows mean
	// ENABLED — the switches exist to turn a role off, not to require opt-in.
	SettingKeyMimoChatEnabledAdmin   = "mimo.chat_enabled_admin"
	SettingKeyMimoChatEnabledPartner = "mimo.chat_enabled_partner"
)

// TransferBankInfo carries the beneficiary identity printed on payroll
// statements (email body and Excel attachment). Hidden=true suppresses the
// beneficiary block entirely: Excel cells are cleared and statement emails
// become informational-only (no transfer instructions). The zero value keeps
// the block visible so existing struct literals stay backward compatible.
type TransferBankInfo struct {
	Holder string
	Number string
	Name   string
	// Hidden is driven by the transfer_bank_visible setting (row missing or
	// unparsable => visible, i.e. Hidden=false).
	Hidden bool
}

// Default values
const (
	DefaultWeeklyPaymentPercentage    = 0.70
	DefaultMonthlyPaymentPercentage   = 0.70
	DefaultAdvanceCashFeePercentage   = 0.02
	DefaultWeeklyPaymentFeePercentage = 0.02
	DefaultPartnerCompany             = "VFIC Manpower"
	DefaultAdvancePaymentPercentage   = 0.60  // 60% max advance
	DefaultAdvancePaymentFeeMin       = 10000 // 10,000 VND minimum fee
	DefaultBulkTransferWorkbookLimit  = int64(400_000_000)
	DefaultSelfCheckInAdvancePercent  = domain.DefaultSelfCheckInAdvancePercentage
	DefaultSelfCheckInAdvanceHold     = domain.QuotaCreditHoldDuration
	MinBulkTransferWorkbookLimit      = int64(100_000_000)
	MaxBulkTransferWorkbookLimit      = int64(500_000_000)
	MaxSelfCheckInAdvanceHoldHours    = uint64(720)

	// Bounds for the wallet low-balance alert threshold. The alert fires once
	// per downward crossing of this value; defaults to 50 million VND.
	DefaultWalletBalanceAlertThreshold = int64(50_000_000)
	MinWalletBalanceAlertThreshold     = int64(1_000_000)
	MaxWalletBalanceAlertThreshold     = int64(1_000_000_000)
	CacheTTL                           = constants.SettingsCacheTTL // Use centralized cache TTL

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

// WeeklyFeeScheduleResolver is the read-side surface of the weekly-payment
// fee schedule. Implemented by *payroll.WeeklyPaymentFeeScheduleService;
// bound at bootstrap via BindWeeklyFeeScheduleResolver. Kept as an interface
// here because package config cannot import package payroll (cycle).
type WeeklyFeeScheduleResolver interface {
	PercentageAt(ctx context.Context, at time.Time) float64
}

type SettingReader interface {
	GetSettingByKey(ctx context.Context, key string) (*domain.Settings, error)
	GetSettingByKeyAuthoritative(ctx context.Context, key string) (*domain.Settings, error)
	GetSettingByKeyAuthoritativeForUpdate(ctx context.Context, key string) (*domain.Settings, error)
}

// SettingsConfigService provides methods to retrieve business configuration settings
type SettingsConfigService struct {
	settingsService   SettingReader
	cache             sync.Map // key -> *cacheEntry
	feeSchedule       FeeScheduleResolver
	weeklyFeeSchedule WeeklyFeeScheduleResolver
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
	if err != nil || value < MinBulkTransferWorkbookLimit || value > MaxBulkTransferWorkbookLimit {
		return 0, domain.NewValidationError("giới hạn tổng tiền file Chuyển lô phải từ 100.000.000 đ đến 500.000.000 đ")
	}
	return value, nil
}

// GetWalletBalanceAlertThreshold returns the wallet balance under which every
// administrator is warned to top up. Reads bypass the local and Redis cache
// layers so a completed Admin update governs the very next evaluation.
// Missing or invalid persisted data falls back to the default.
func (s *SettingsConfigService) GetWalletBalanceAlertThreshold(ctx context.Context) int64 {
	if s.settingsService == nil {
		return DefaultWalletBalanceAlertThreshold
	}

	setting, err := s.settingsService.GetSettingByKeyAuthoritative(ctx, SettingKeyWalletBalanceAlert)
	if err != nil {
		observability.GetLogger().Warn(
			"failed to get wallet balance alert threshold setting, using default",
			"key", SettingKeyWalletBalanceAlert,
			"error", err,
		)
		return DefaultWalletBalanceAlertThreshold
	}

	value, err := parseWalletBalanceAlertThreshold(setting)
	if err != nil {
		observability.GetLogger().Warn(
			"invalid wallet balance alert threshold setting, using default",
			"key", SettingKeyWalletBalanceAlert,
			"error", err,
		)
		return DefaultWalletBalanceAlertThreshold
	}
	return value
}

func parseWalletBalanceAlertThreshold(setting *domain.Settings) (int64, error) {
	if setting == nil || setting.Value == nil {
		return 0, domain.NewValidationError("ngưỡng cảnh báo số dư ví tiền không được để trống")
	}
	if setting.ValueType != domain.ValueTypeNumber {
		return 0, domain.NewValidationError("ngưỡng cảnh báo số dư ví tiền phải là số")
	}

	raw := *setting.Value
	if raw == "" || strings.TrimSpace(raw) != raw || raw[0] == '0' || strings.IndexFunc(raw, func(r rune) bool {
		return r < '0' || r > '9'
	}) >= 0 {
		return 0, domain.NewValidationError("ngưỡng cảnh báo số dư ví tiền phải là số nguyên")
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < MinWalletBalanceAlertThreshold || value > MaxWalletBalanceAlertThreshold {
		return 0, domain.NewValidationError("ngưỡng cảnh báo số dư ví tiền phải từ 1.000.000 đ đến 1.000.000.000 đ")
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

// BindWeeklyFeeScheduleResolver binds the weekly-payment fee schedule service
// so GetWeeklyPaymentFeePercentage reads the decoupled weekly schedule
// instead of the shared legacy default.
func (s *SettingsConfigService) BindWeeklyFeeScheduleResolver(r WeeklyFeeScheduleResolver) {
	s.weeklyFeeSchedule = r
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

// GetWeeklyPaymentFeePercentage returns the service fee percentage (as a
// fraction, 0.02 == 2%) charged on weekly salary disbursements. Once the
// WeeklyFeeScheduleResolver is bound, this reads the weekly schedule's active
// entry for today; unbound callers fall back to the same 2% default the
// shared schedule provided before the two fee domains were decoupled.
func (s *SettingsConfigService) GetWeeklyPaymentFeePercentage(ctx context.Context) float64 {
	if s.weeklyFeeSchedule != nil {
		return s.weeklyFeeSchedule.PercentageAt(ctx, clock.Now())
	}
	return DefaultWeeklyPaymentFeePercentage
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
// weekly payroll statement emails and Excel attachments, plus the visibility
// flag. Unconfigured rows fall back to the MB bank defaults; the visibility
// flag defaults to true.
func (s *SettingsConfigService) GetTransferBankInfo(ctx context.Context) TransferBankInfo {
	return s.getTransferBankInfo(ctx,
		SettingKeyTransferBankHolder,
		SettingKeyTransferBankNumber,
		SettingKeyTransferBankName,
		SettingKeyTransferBankVisible,
		DefaultTransferBankInfo(),
	)
}

// GetFlexPayTransferBankInfo retrieves the beneficiary bank details printed on
// FlexPay (OnePay early-salary) reconciliation statements — email body and
// Excel attachment. Each field falls back to the weekly payroll account when
// its own row is unset, so environments that never configured a separate
// FlexPay account keep printing exactly what they printed before the two
// accounts were split.
func (s *SettingsConfigService) GetFlexPayTransferBankInfo(ctx context.Context) TransferBankInfo {
	return s.getTransferBankInfo(ctx,
		SettingKeyFlexPayTransferBankHolder,
		SettingKeyFlexPayTransferBankNumber,
		SettingKeyFlexPayTransferBankName,
		SettingKeyFlexPayTransferBankVisible,
		s.GetTransferBankInfo(ctx),
	)
}

// getTransferBankInfo resolves one beneficiary block from its four setting
// keys, using fallback for every row that is missing or empty.
func (s *SettingsConfigService) getTransferBankInfo(
	ctx context.Context,
	holderKey, numberKey, nameKey, visibleKey string,
	fallback TransferBankInfo,
) TransferBankInfo {
	return TransferBankInfo{
		Holder: s.getTransferBankString(ctx, holderKey, fallback.Holder),
		Number: s.getTransferBankString(ctx, numberKey, fallback.Number),
		Name:   s.getTransferBankString(ctx, nameKey, fallback.Name),
		Hidden: !s.getTransferBankBool(ctx, visibleKey, !fallback.Hidden),
	}
}

// getTransferBankBool reads one boolean setting with caching, falling back to
// defaultValue when the row is missing or holds an unrecognized value.
func (s *SettingsConfigService) getTransferBankBool(ctx context.Context, key string, defaultValue bool) bool {
	if s.settingsService == nil {
		return defaultValue
	}

	if cached, ok := s.getFromCache(key); ok {
		if v, ok := cached.(bool); ok {
			return v
		}
		return defaultValue
	}

	setting, err := s.settingsService.GetSettingByKey(ctx, key)
	if err != nil {
		// Cache the fallback too (short TTL): the transfer_bank_visible row
		// does not exist until an admin first saves the toggle, and without
		// negative caching every sao-ke email/export pays a DB round-trip
		// for the same not-found answer. Staleness is bounded by the TTL.
		observability.GetLogger().Warn("failed to get boolean setting, using default", "key", key, "error", err)
		s.setCache(key, defaultValue)
		return defaultValue
	}

	raw := strings.TrimSpace(setting.GetStringValue())
	if raw == "" {
		// Empty row value behaves like a missing row (default), matching the
		// frontend toggle's parse.
		s.setCache(key, defaultValue)
		return defaultValue
	}
	// Only an explicit "true" (any case) keeps the block visible; anything
	// else ("false", "0", garbage) hides it — same rule the frontend applies.
	value := strings.EqualFold(raw, "true")
	s.setCache(key, value)

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

// --- MiMo LLM assistant settings -------------------------------------------
//
// Reads are authoritative (repository, no cache): an admin key rotation or
// model change must govern the very next LLM call, and the API key must never
// land in a cache payload — the same rule SettingsService applies to
// secret.IsProtectedKey rows.

// GetMimoAPIKey returns the MiMo API key, or "" when unconfigured. The value
// is a secret: never log, never cache, never return in API responses (the DTO
// masks it); only the LLM client factory consumes it.
func (s *SettingsConfigService) GetMimoAPIKey(ctx context.Context) string {
	if s.settingsService == nil {
		return ""
	}
	setting, err := s.settingsService.GetSettingByKeyAuthoritative(ctx, SettingKeyMimoAPIKey)
	if err != nil {
		// Not-found is the normal "not configured yet" state — log at info
		// only when it is something else. Never include the value.
		if !domain.IsNotFoundError(err) {
			observability.GetLogger().Warn("failed to get mimo api key setting", "key", SettingKeyMimoAPIKey, "error", err)
		}
		return ""
	}
	if setting == nil || setting.Value == nil {
		return ""
	}
	return strings.TrimSpace(*setting.Value)
}

// GetMimoBaseURL returns the provider base URL (version segment included).
// An unset, malformed, or non-HTTP(S) override falls back to the production
// endpoint — a typo here must not silently black-hole assistant traffic.
func (s *SettingsConfigService) GetMimoBaseURL(ctx context.Context) string {
	if s.settingsService == nil {
		return llmport.DefaultBaseURL
	}
	setting, err := s.settingsService.GetSettingByKeyAuthoritative(ctx, SettingKeyMimoBaseURL)
	if err != nil {
		if !domain.IsNotFoundError(err) {
			observability.GetLogger().Warn("failed to get mimo base url setting, using default", "key", SettingKeyMimoBaseURL, "error", err)
		}
		return llmport.DefaultBaseURL
	}
	if setting == nil || setting.Value == nil {
		return llmport.DefaultBaseURL
	}
	raw := strings.TrimSpace(strings.TrimRight(*setting.Value, "/"))
	if raw == "" {
		return llmport.DefaultBaseURL
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		observability.GetLogger().Warn("invalid mimo base url setting, using default", "key", SettingKeyMimoBaseURL)
		return llmport.DefaultBaseURL
	}
	return raw
}

// GetMimoDefaultModel returns the model routed for ordinary LLM calls. Invalid
// or retired values (e.g. a leftover mimo-v2.5* row) fall back to the built-in
// default so stale config can never route production traffic to a retired
// family.
// GetMimoChatEnabledForRole reports whether the chat surface is enabled for
// the role. Missing or unreadable rows default to enabled; unknown roles are
// always denied. Reads the authoritative path (no cache) so an admin toggle
// applies on the next request.
func (s *SettingsConfigService) GetMimoChatEnabledForRole(ctx context.Context, role string) bool {
	switch role {
	case "admin":
		return s.mimoChatEnabled(ctx, SettingKeyMimoChatEnabledAdmin)
	case "partner":
		return s.mimoChatEnabled(ctx, SettingKeyMimoChatEnabledPartner)
	default:
		return false
	}
}

func (s *SettingsConfigService) mimoChatEnabled(ctx context.Context, key string) bool {
	if s.settingsService == nil {
		return true
	}
	setting, err := s.settingsService.GetSettingByKeyAuthoritative(ctx, key)
	if err != nil || setting.Value == nil {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(*setting.Value)) {
	case "false", "0", "off", "no":
		return false
	}
	return true
}

func (s *SettingsConfigService) GetMimoDefaultModel(ctx context.Context) string {
	return s.getMimoModel(ctx, SettingKeyMimoDefaultModel, llmport.DefaultModel)
}

// GetMimoFallbackModel returns the model used when the primary model is not
// found (the vendor renamed its model family). Invalid values fall back to the
// built-in fallback.
func (s *SettingsConfigService) GetMimoFallbackModel(ctx context.Context) string {
	return s.getMimoModel(ctx, SettingKeyMimoFallbackModel, llmport.DefaultFallbackModel)
}

// getMimoModel reads one model routing setting with the allowlist enforced on
// the read path.
func (s *SettingsConfigService) getMimoModel(ctx context.Context, key, defaultValue string) string {
	if s.settingsService == nil {
		return defaultValue
	}
	setting, err := s.settingsService.GetSettingByKeyAuthoritative(ctx, key)
	if err != nil {
		if !domain.IsNotFoundError(err) {
			observability.GetLogger().Warn("failed to get mimo model setting, using default", "key", key, "error", err)
		}
		return defaultValue
	}
	if setting == nil || setting.Value == nil {
		return defaultValue
	}
	raw := strings.TrimSpace(*setting.Value)
	if raw == "" {
		return defaultValue
	}
	if _, err := llmport.ResolveModel(raw, ""); err != nil {
		observability.GetLogger().Warn("invalid mimo model setting, using default", "key", key, "error", err)
		return defaultValue
	}
	return raw
}

// parseMimoAPIKey validates a MiMo API key write. The value is required and
// bounded; it is sealed at rest by the settings repository (secret.MimoAPIKey).
func parseMimoAPIKey(setting *domain.Settings) error {
	if setting == nil || setting.Value == nil {
		return domain.NewValidationError("Khóa bí mật MiMo không được để trống")
	}
	raw := strings.TrimSpace(*setting.Value)
	if raw == "" {
		return domain.NewValidationError("Khóa bí mật MiMo không được để trống")
	}
	if len(raw) > 512 {
		return domain.NewValidationError("Khóa bí mật MiMo tối đa 512 ký tự")
	}
	return nil
}

// parseMimoBaseURL validates an optional provider base-URL override. Empty
// means "use the production endpoint"; a set value must be absolute HTTP(S).
func parseMimoBaseURL(setting *domain.Settings) error {
	if setting == nil || setting.Value == nil {
		return nil
	}
	raw := strings.TrimSpace(strings.TrimRight(*setting.Value, "/"))
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return domain.NewValidationError("URL API MiMo không hợp lệ — phải là địa chỉ http:// hoặc https:// đầy đủ")
	}
	return nil
}

// parseMimoModel validates a model routing write against the allowlist.
// Empty resets the override to the built-in default. Retired families
// (mimo-v2.5*) are rejected here — config validation is the last line of
// defence before a stale value could route production traffic.
func parseMimoModel(setting *domain.Settings) error {
	if setting == nil || setting.Value == nil {
		return nil
	}
	raw := strings.TrimSpace(*setting.Value)
	if raw == "" {
		return nil
	}
	if err := llmport.ValidateModel(raw); err != nil {
		return domain.NewValidationError(err.Error())
	}
	return nil
}
