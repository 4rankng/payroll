package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmployeeMobileRequired(t *testing.T) {
	// Missing row = policy off: rollout is an admin decision, not a deploy artifact.
	missing := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{}})
	assert.False(t, missing.IsEmployeeMobileRequired(context.Background()))

	on := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeyEmployeeMobileRequired: "true",
	}})
	assert.True(t, on.IsEmployeeMobileRequired(context.Background()))

	off := NewSettingsConfigService(&keyedStubSettingReader{values: map[string]string{
		SettingKeyEmployeeMobileRequired: "false",
	}})
	assert.False(t, off.IsEmployeeMobileRequired(context.Background()))
}
