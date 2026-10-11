package config

import (
	"context"
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	"api-server/internal/domain"
	llmport "api-server/internal/infra/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSettingReader struct {
	setting *domain.Settings
	err     error
	calls   int
	locks   int
}

func (s *stubSettingReader) GetSettingByKey(context.Context, string) (*domain.Settings, error) {
	return s.setting, s.err
}

func (s *stubSettingReader) GetSettingByKeyAuthoritative(context.Context, string) (*domain.Settings, error) {
	s.calls++
	return s.setting, s.err
}

func (s *stubSettingReader) GetSettingByKeyAuthoritativeForUpdate(context.Context, string) (*domain.Settings, error) {
	s.locks++
	return s.setting, s.err
}

func numberSetting(value *string) *domain.Settings {
	return &domain.Settings{
		Key:       SettingKeyBulkTransferWorkbookLimit,
		Value:     value,
		ValueType: domain.ValueTypeNumber,
	}
}

func selfCheckInPercentSetting(value *string) *domain.Settings {
	return &domain.Settings{
		Key:       SettingKeySelfCheckInAdvancePercent,
		Value:     value,
		ValueType: domain.ValueTypeNumber,
	}
}

func selfCheckInAdvanceHoldSetting(value *string) *domain.Settings {
	return &domain.Settings{
		Key:       SettingKeySelfCheckInAdvanceHold,
		Value:     value,
		ValueType: domain.ValueTypeNumber,
	}
}

func TestSettingsConfigServiceGetBulkTransferWorkbookLimit(t *testing.T) {
	valid := "275000000"
	reader := &stubSettingReader{setting: numberSetting(&valid)}
	service := NewSettingsConfigService(reader)

	assert.Equal(t, int64(275_000_000), service.GetBulkTransferWorkbookLimit(context.Background()))
	assert.Equal(t, 1, reader.calls)

	updated := "350000000"
	reader.setting = numberSetting(&updated)
	assert.Equal(t, int64(350_000_000), service.GetBulkTransferWorkbookLimit(context.Background()))
	assert.Equal(t, 2, reader.calls, "the money-moving threshold must bypass the local cache")
}

func TestSettingsConfigServiceGetBulkTransferWorkbookLimitFallsBackToDefault(t *testing.T) {
	tests := []struct {
		name    string
		setting *domain.Settings
		err     error
	}{
		{name: "missing", err: errors.New("not found")},
		{name: "nil value", setting: numberSetting(nil)},
		{name: "decimal", setting: numberSetting(stringPointer("400000000.5"))},
		{name: "plus sign", setting: numberSetting(stringPointer("+400000000"))},
		{name: "leading zero", setting: numberSetting(stringPointer("0400000000"))},
		{name: "below minimum", setting: numberSetting(stringPointer("1"))},
		{name: "overflow", setting: numberSetting(stringPointer("9223372036854775808"))},
		{name: "surrounding whitespace", setting: numberSetting(stringPointer(" 400000000 "))},
		{
			name: "wrong type",
			setting: &domain.Settings{
				Key:       SettingKeyBulkTransferWorkbookLimit,
				Value:     stringPointer("400000000"),
				ValueType: domain.ValueTypeString,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSettingsConfigService(&stubSettingReader{setting: tt.setting, err: tt.err})
			assert.Equal(t, DefaultBulkTransferWorkbookLimit, service.GetBulkTransferWorkbookLimit(context.Background()))
		})
	}
}

func TestParseBulkTransferWorkbookLimitBoundaries(t *testing.T) {
	for _, value := range []int64{MinBulkTransferWorkbookLimit, DefaultBulkTransferWorkbookLimit, MaxBulkTransferWorkbookLimit} {
		raw := strconv.FormatInt(value, 10)
		parsed, err := parseBulkTransferWorkbookLimit(numberSetting(&raw))
		require.NoError(t, err)
		assert.Equal(t, value, parsed)
	}

	for _, value := range []int64{1, MinBulkTransferWorkbookLimit - 1, MaxBulkTransferWorkbookLimit + 1, math.MaxInt64} {
		raw := strconv.FormatInt(value, 10)
		_, err := parseBulkTransferWorkbookLimit(numberSetting(&raw))
		require.Error(t, err)
	}
}

func TestValidateBusinessSettingValidatesNamedFinancialControls(t *testing.T) {
	invalid := "not-a-number"
	require.Error(t, validateBusinessSetting(numberSetting(&invalid)))
	require.Error(t, validateBusinessSetting(selfCheckInPercentSetting(&invalid)))
	require.Error(t, validateBusinessSetting(selfCheckInAdvanceHoldSetting(&invalid)))

	unrelated := &domain.Settings{
		Key:       "some_other_number",
		Value:     &invalid,
		ValueType: domain.ValueTypeNumber,
	}
	require.NoError(t, validateBusinessSetting(unrelated))
}

func TestSettingsConfigServiceGetSelfCheckInAdvanceHoldDuration(t *testing.T) {
	valid := "6"
	reader := &stubSettingReader{setting: selfCheckInAdvanceHoldSetting(&valid)}
	service := NewSettingsConfigService(reader)

	assert.Equal(t, 6*time.Hour, service.GetSelfCheckInAdvanceHoldDuration(context.Background()))
	assert.Equal(t, 1, reader.calls)

	updated := "0"
	reader.setting = selfCheckInAdvanceHoldSetting(&updated)
	assert.Equal(t, time.Duration(0), service.GetSelfCheckInAdvanceHoldDuration(context.Background()))
	assert.Equal(t, 2, reader.calls, "the credit schedule must bypass the local cache")
}

func TestSettingsConfigServiceGetSelfCheckInAdvanceHoldDurationFallsBackToDefault(t *testing.T) {
	tests := []struct {
		name    string
		setting *domain.Settings
		err     error
	}{
		{name: "missing", err: errors.New("not found")},
		{name: "nil value", setting: selfCheckInAdvanceHoldSetting(nil)},
		{name: "negative", setting: selfCheckInAdvanceHoldSetting(stringPointer("-1"))},
		{name: "decimal", setting: selfCheckInAdvanceHoldSetting(stringPointer("24.5"))},
		{name: "leading zero", setting: selfCheckInAdvanceHoldSetting(stringPointer("024"))},
		{name: "over maximum", setting: selfCheckInAdvanceHoldSetting(stringPointer("721"))},
		{
			name: "wrong type",
			setting: &domain.Settings{
				Key:       SettingKeySelfCheckInAdvanceHold,
				Value:     stringPointer("24"),
				ValueType: domain.ValueTypeString,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSettingsConfigService(&stubSettingReader{setting: tt.setting, err: tt.err})
			assert.Equal(t, DefaultSelfCheckInAdvanceHold, service.GetSelfCheckInAdvanceHoldDuration(context.Background()))
		})
	}
}

func TestParseSelfCheckInAdvanceHoldHoursBoundaries(t *testing.T) {
	for _, value := range []uint64{0, 24, MaxSelfCheckInAdvanceHoldHours} {
		raw := strconv.FormatUint(value, 10)
		parsed, err := parseSelfCheckInAdvanceHoldHours(selfCheckInAdvanceHoldSetting(&raw))
		require.NoError(t, err)
		assert.Equal(t, value, parsed)
	}
}

func TestSettingsConfigServiceGetSelfCheckInAdvancePercentage(t *testing.T) {
	valid := "85"
	reader := &stubSettingReader{setting: selfCheckInPercentSetting(&valid)}
	service := NewSettingsConfigService(reader)

	assert.Equal(t, uint64(85), service.GetSelfCheckInAdvancePercentage(context.Background()))
	assert.Equal(t, 1, reader.calls)
	assert.Equal(t, 0, reader.locks)
}

func TestSettingsConfigServiceGetSelfCheckInAdvancePercentageForUpdateLocks(t *testing.T) {
	valid := "72"
	reader := &stubSettingReader{setting: selfCheckInPercentSetting(&valid)}
	service := NewSettingsConfigService(reader)

	got, err := service.GetSelfCheckInAdvancePercentageForUpdate(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(72), got)
	assert.Equal(t, 0, reader.calls)
	assert.Equal(t, 1, reader.locks)
}

func TestSettingsConfigServiceGetSelfCheckInAdvancePercentageForUpdatePropagatesLockFailure(t *testing.T) {
	reader := &stubSettingReader{err: errors.New("database unavailable")}
	service := NewSettingsConfigService(reader)

	_, err := service.GetSelfCheckInAdvancePercentageForUpdate(context.Background())
	require.ErrorContains(t, err, "database unavailable")
	assert.Equal(t, 1, reader.locks)
}

func TestSettingsConfigServiceGetSelfCheckInAdvancePercentageFallsBackToDefault(t *testing.T) {
	tests := []struct {
		name    string
		setting *domain.Settings
		err     error
	}{
		{name: "missing", err: errors.New("not found")},
		{name: "nil value", setting: selfCheckInPercentSetting(nil)},
		{name: "zero", setting: selfCheckInPercentSetting(stringPointer("0"))},
		{name: "leading zero", setting: selfCheckInPercentSetting(stringPointer("070"))},
		{name: "spaces", setting: selfCheckInPercentSetting(stringPointer(" 70 "))},
		{name: "decimal", setting: selfCheckInPercentSetting(stringPointer("70.5"))},
		{name: "plus sign", setting: selfCheckInPercentSetting(stringPointer("+70"))},
		{name: "overflow range", setting: selfCheckInPercentSetting(stringPointer("101"))},
		{
			name: "wrong type",
			setting: &domain.Settings{
				Key:       SettingKeySelfCheckInAdvancePercent,
				Value:     stringPointer("70"),
				ValueType: domain.ValueTypeString,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSettingsConfigService(&stubSettingReader{setting: tt.setting, err: tt.err})
			assert.Equal(t, DefaultSelfCheckInAdvancePercent, service.GetSelfCheckInAdvancePercentage(context.Background()))
		})
	}
}

func TestParseSelfCheckInAdvancePercentBoundaries(t *testing.T) {
	for _, value := range []uint64{1, DefaultSelfCheckInAdvancePercent, 100} {
		raw := strconv.FormatUint(value, 10)
		parsed, err := parseSelfCheckInAdvancePercent(selfCheckInPercentSetting(&raw))
		require.NoError(t, err)
		assert.Equal(t, value, parsed)
	}
}

func stringPointer(value string) *string {
	return &value
}

func stringSetting(key, value string) *domain.Settings {
	v := value
	return &domain.Settings{
		Key:       key,
		Value:     &v,
		ValueType: domain.ValueTypeString,
	}
}

func TestGetTransferBankInfoReturnsConfiguredValues(t *testing.T) {
	values := map[string]string{
		SettingKeyTransferBankHolder: "TEN CONG TY MOI",
		SettingKeyTransferBankNumber: "999888777",
		SettingKeyTransferBankName:   "Ngân hàng Test (TT)",
	}
	reader := &keyedStubSettingReader{values: values}
	service := NewSettingsConfigService(reader)

	info := service.GetTransferBankInfo(context.Background())
	assert.Equal(t, "TEN CONG TY MOI", info.Holder)
	assert.Equal(t, "999888777", info.Number)
	assert.Equal(t, "Ngân hàng Test (TT)", info.Name)
	// Row missing => default visible.
	assert.False(t, info.Hidden)
}

func TestGetTransferBankInfoHiddenWhenToggleFalse(t *testing.T) {
	values := map[string]string{
		SettingKeyTransferBankHolder:  "TEN CONG TY MOI",
		SettingKeyTransferBankNumber:  "999888777",
		SettingKeyTransferBankName:    "Ngân hàng Test (TT)",
		SettingKeyTransferBankVisible: "false",
	}
	reader := &keyedStubSettingReader{values: values}
	service := NewSettingsConfigService(reader)

	info := service.GetTransferBankInfo(context.Background())
	assert.True(t, info.Hidden, "transfer_bank_visible=false must hide the beneficiary block")

	visibleService := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeyTransferBankVisible: "true",
	}})
	assert.False(t, visibleService.GetTransferBankInfo(context.Background()).Hidden)

	// Non-"true" values hide; empty value behaves like a missing row (visible).
	zeroService := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeyTransferBankVisible: "0",
	}})
	assert.True(t, zeroService.GetTransferBankInfo(context.Background()).Hidden)
	emptyService := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeyTransferBankVisible: "",
	}})
	assert.False(t, emptyService.GetTransferBankInfo(context.Background()).Hidden)
}

type keyedStubSettingReader struct {
	values map[string]string
}

func (s *keyedStubSettingReader) GetSettingByKey(_ context.Context, key string) (*domain.Settings, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return stringSetting(key, value), nil
}

func (s *keyedStubSettingReader) GetSettingByKeyAuthoritative(ctx context.Context, key string) (*domain.Settings, error) {
	return s.GetSettingByKey(ctx, key)
}

func (s *keyedStubSettingReader) GetSettingByKeyAuthoritativeForUpdate(ctx context.Context, key string) (*domain.Settings, error) {
	return s.GetSettingByKey(ctx, key)
}

func TestGetTransferBankInfoFallsBackToDefaults(t *testing.T) {
	service := NewSettingsConfigService(&stubSettingReader{setting: nil, err: errors.New("not found")})

	info := service.GetTransferBankInfo(context.Background())
	assert.Equal(t, DefaultTransferBankHolder, info.Holder)
	assert.Equal(t, DefaultTransferBankNumber, info.Number)
	assert.Equal(t, DefaultTransferBankName, info.Name)
	assert.False(t, info.Hidden, "missing toggle row must default to visible")
}

func TestGetFlexPayTransferBankInfoUsesItsOwnKeys(t *testing.T) {
	service := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeyTransferBankHolder:         "CONG TY LUONG TUAN",
		SettingKeyTransferBankNumber:         "111222333",
		SettingKeyTransferBankName:           "Ngân hàng Tuần",
		SettingKeyFlexPayTransferBankHolder:  "CONG TY FLEXPAY",
		SettingKeyFlexPayTransferBankNumber:  "444555666",
		SettingKeyFlexPayTransferBankName:    "Ngân hàng FlexPay",
		SettingKeyFlexPayTransferBankVisible: "false",
	}})

	weekly := service.GetTransferBankInfo(context.Background())
	assert.Equal(t, "CONG TY LUONG TUAN", weekly.Holder)
	assert.Equal(t, "111222333", weekly.Number)
	assert.False(t, weekly.Hidden, "the weekly block stays visible")

	flexPay := service.GetFlexPayTransferBankInfo(context.Background())
	assert.Equal(t, "CONG TY FLEXPAY", flexPay.Holder)
	assert.Equal(t, "444555666", flexPay.Number)
	assert.Equal(t, "Ngân hàng FlexPay", flexPay.Name)
	assert.True(t, flexPay.Hidden, "flexpay_transfer_bank_visible=false must hide only the FlexPay block")
}

func TestGetFlexPayTransferBankInfoFallsBackToWeeklyAccount(t *testing.T) {
	service := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeyTransferBankHolder:  "CONG TY LUONG TUAN",
		SettingKeyTransferBankNumber:  "111222333",
		SettingKeyTransferBankName:    "Ngân hàng Tuần",
		SettingKeyTransferBankVisible: "false",
	}})

	info := service.GetFlexPayTransferBankInfo(context.Background())
	assert.Equal(t, "CONG TY LUONG TUAN", info.Holder)
	assert.Equal(t, "111222333", info.Number)
	assert.Equal(t, "Ngân hàng Tuần", info.Name)
	assert.True(t, info.Hidden, "an unconfigured FlexPay block mirrors the weekly visibility")
}

func TestGetFlexPayTransferBankInfoFallsBackToDefaults(t *testing.T) {
	service := NewSettingsConfigService(&stubSettingReader{setting: nil, err: errors.New("not found")})

	info := service.GetFlexPayTransferBankInfo(context.Background())
	assert.Equal(t, DefaultTransferBankHolder, info.Holder)
	assert.Equal(t, DefaultTransferBankNumber, info.Number)
	assert.Equal(t, DefaultTransferBankName, info.Name)
	assert.False(t, info.Hidden)
}

func TestSettingsConfigServiceInvalidateCacheReReadsTransferBank(t *testing.T) {
	reader := &keyedStubSettingReader{values: map[string]string{
		SettingKeyTransferBankHolder: "CONG TY CU",
		SettingKeyTransferBankNumber: "111222333",
		SettingKeyTransferBankName:   "Ngân hàng Cũ",
	}}
	service := NewSettingsConfigService(reader)

	require.Equal(t, "CONG TY CU", service.GetTransferBankInfo(context.Background()).Holder)

	// A persisted change stays invisible while the in-process cache is warm...
	reader.values[SettingKeyTransferBankHolder] = "CONG TY MOI"
	assert.Equal(t, "CONG TY CU", service.GetTransferBankInfo(context.Background()).Holder)

	// ...so a committed settings mutation must clear it, otherwise the next
	// statement export would print the account the admin just replaced.
	service.InvalidateCache()
	assert.Equal(t, "CONG TY MOI", service.GetTransferBankInfo(context.Background()).Holder)
}

// --- MiMo LLM settings accessors -------------------------------------------

func TestMimoAccessorsFallBackToDefaults(t *testing.T) {
	service := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{}})

	assert.Equal(t, "", service.GetMimoAPIKey(context.Background()))
	assert.Equal(t, llmport.DefaultBaseURL, service.GetMimoBaseURL(context.Background()))
	assert.Equal(t, llmport.DefaultModel, service.GetMimoDefaultModel(context.Background()))
	assert.Equal(t, llmport.DefaultFallbackModel, service.GetMimoFallbackModel(context.Background()))
}

func TestGetMimoAPIKeyReadsAuthoritativeAndTrims(t *testing.T) {
	reader := &keyedStubSettingReader{values: map[string]string{
		SettingKeyMimoAPIKey: "  sk-live-fake-key  ",
	}}
	service := NewSettingsConfigService(reader)
	require.Equal(t, "sk-live-fake-key", service.GetMimoAPIKey(context.Background()))

	// The key must never ride the in-process cache: a rotation governs the
	// next read, and a decrypted secret must not sit in the cache map.
	reader.values[SettingKeyMimoAPIKey] = "sk-rotated-fake"
	require.Equal(t, "sk-rotated-fake", service.GetMimoAPIKey(context.Background()))
}

func TestMimoModelSettingsRejectRetiredFamilies(t *testing.T) {
	// A leftover pre-deprecation row must never route production traffic.
	service := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeyMimoDefaultModel:  "mimo-v2.5-flash",
		SettingKeyMimoFallbackModel: "mimo-v2.5-pro",
	}})
	assert.Equal(t, llmport.DefaultModel, service.GetMimoDefaultModel(context.Background()))
	assert.Equal(t, llmport.DefaultFallbackModel, service.GetMimoFallbackModel(context.Background()))
}

func TestMimoModelSettingsAcceptAllowlistOverrides(t *testing.T) {
	service := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeyMimoDefaultModel:  llmport.ModelPro,
		SettingKeyMimoFallbackModel: llmport.ModelFlash,
	}})
	assert.Equal(t, llmport.ModelPro, service.GetMimoDefaultModel(context.Background()))
	assert.Equal(t, llmport.ModelFlash, service.GetMimoFallbackModel(context.Background()))
}

func TestMimoBaseURLValidation(t *testing.T) {
	for _, bad := range []string{"not a url", "ftp://example.com", "example.com", "/relative"} {
		service := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
			SettingKeyMimoBaseURL: bad,
		}})
		assert.Equal(t, llmport.DefaultBaseURL, service.GetMimoBaseURL(context.Background()),
			"invalid base URL %q must fall back to the production endpoint", bad)
	}

	service := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeyMimoBaseURL: "https://proxy.internal/v1/",
	}})
	assert.Equal(t, "https://proxy.internal/v1", service.GetMimoBaseURL(context.Background()))
}

func TestValidateBusinessSettingMimoKeys(t *testing.T) {
	// API key: required, bounded.
	empty := "   "
	require.Error(t, validateBusinessSetting(&domain.Settings{Key: SettingKeyMimoAPIKey, Value: &empty, ValueType: domain.ValueTypeString}))
	missing := &domain.Settings{Key: SettingKeyMimoAPIKey, ValueType: domain.ValueTypeString}
	require.Error(t, validateBusinessSetting(missing))
	goodKey := "sk-live-fake-key"
	require.NoError(t, validateBusinessSetting(&domain.Settings{Key: SettingKeyMimoAPIKey, Value: &goodKey, ValueType: domain.ValueTypeString}))

	// Model routing: retired families rejected with a clear Vietnamese error.
	retired := "mimo-v2.5-flash"
	err := validateBusinessSetting(&domain.Settings{Key: SettingKeyMimoDefaultModel, Value: &retired, ValueType: domain.ValueTypeString})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mimo-v2.5")
	assert.Contains(t, err.Error(), "ngừng hỗ trợ")
	require.Error(t, validateBusinessSetting(&domain.Settings{Key: SettingKeyMimoFallbackModel, Value: &retired, ValueType: domain.ValueTypeString}))

	pro := llmport.ModelPro
	require.NoError(t, validateBusinessSetting(&domain.Settings{Key: SettingKeyMimoDefaultModel, Value: &pro, ValueType: domain.ValueTypeString}))
	flash := llmport.ModelFlash
	require.NoError(t, validateBusinessSetting(&domain.Settings{Key: SettingKeyMimoFallbackModel, Value: &flash, ValueType: domain.ValueTypeString}))

	// Empty model value resets the override to the built-in default.
	reset := ""
	require.NoError(t, validateBusinessSetting(&domain.Settings{Key: SettingKeyMimoDefaultModel, Value: &reset, ValueType: domain.ValueTypeString}))

	// Base URL: optional, but must be absolute HTTP(S) when set.
	badURL := "not a url"
	require.Error(t, validateBusinessSetting(&domain.Settings{Key: SettingKeyMimoBaseURL, Value: &badURL, ValueType: domain.ValueTypeString}))
	goodURL := "https://api.xiaomimimo.com/v1"
	require.NoError(t, validateBusinessSetting(&domain.Settings{Key: SettingKeyMimoBaseURL, Value: &goodURL, ValueType: domain.ValueTypeString}))
}
