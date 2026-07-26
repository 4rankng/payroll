package config

import (
	"context"
	"errors"
	"math"
	"strconv"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSettingReader struct {
	setting *domain.Settings
	err     error
	calls   int
}

func (s *stubSettingReader) GetSettingByKey(context.Context, string) (*domain.Settings, error) {
	return s.setting, s.err
}

func (s *stubSettingReader) GetSettingByKeyAuthoritative(context.Context, string) (*domain.Settings, error) {
	s.calls++
	return s.setting, s.err
}

func numberSetting(value *string) *domain.Settings {
	return &domain.Settings{
		Key:       SettingKeyBulkTransferWorkbookLimit,
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
	for _, value := range []int64{MinBulkTransferWorkbookLimit, DefaultBulkTransferWorkbookLimit, math.MaxInt64} {
		raw := strconv.FormatInt(value, 10)
		parsed, err := parseBulkTransferWorkbookLimit(numberSetting(&raw))
		require.NoError(t, err)
		assert.Equal(t, value, parsed)
	}
}

func TestValidateBusinessSettingOnlyAppliesToWorkbookLimit(t *testing.T) {
	invalid := "not-a-number"
	require.Error(t, validateBusinessSetting(numberSetting(&invalid)))

	unrelated := &domain.Settings{
		Key:       "some_other_number",
		Value:     &invalid,
		ValueType: domain.ValueTypeNumber,
	}
	require.NoError(t, validateBusinessSetting(unrelated))
}

func stringPointer(value string) *string {
	return &value
}
