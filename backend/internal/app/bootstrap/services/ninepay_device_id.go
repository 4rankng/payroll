package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"

	"api-server/internal/app/services/config"
	"api-server/internal/domain"
)

// ninepayDeviceIDKey is the Settings table key under which the
// auto-generated device_id is persisted. One row per merchant
// deployment — survives container restarts and is shared across
// horizontally-scaled backend instances.
const ninepayDeviceIDKey = "ninepay_web_device_id"

// resolveNinepayDeviceID returns the device_id to send in the 9pay
// merchant-portal login form. Resolution order:
//  1. Env override (envValue) — highest priority, lets operators force
//     a known value during migrations or per-merchant rotations.
//  2. Persisted value in the Settings table — set on first call.
//  3. Newly generated 32-hex (16 random bytes), persisted then returned.
//
// Generates a stable per-deployment device_id without operator
// involvement, while still allowing override via env when needed.
func resolveNinepayDeviceID(ctx context.Context, settingsSvc *config.SettingsService, envValue string, logger *slog.Logger) (string, error) {
	if envValue != "" {
		return envValue, nil
	}
	if settingsSvc == nil {
		return "", fmt.Errorf("settings service required to resolve device_id")
	}

	existing, err := settingsSvc.GetSettingByKey(ctx, ninepayDeviceIDKey)
	if err == nil && existing != nil && existing.Value != nil && *existing.Value != "" {
		return *existing.Value, nil
	}
	// Treat any error other than "not found" as fatal — silently
	// regenerating on a transient DB error would risk producing a
	// fresh device_id when one already exists.
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return "", fmt.Errorf("read setting %q: %w", ninepayDeviceIDKey, err)
	}

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate device_id: %w", err)
	}
	value := hex.EncodeToString(buf)

	v := value
	created, err := settingsSvc.CreateSetting(ctx, &domain.Settings{
		Key:       ninepayDeviceIDKey,
		Value:     &v,
		ValueType: domain.ValueTypeString,
	}, 0)
	if err != nil {
		return "", fmt.Errorf("persist setting %q: %w", ninepayDeviceIDKey, err)
	}
	if logger != nil {
		logger.Info("ninepay: generated and persisted device_id",
			"settings_id", created.ID, "key", ninepayDeviceIDKey)
	}
	return value, nil
}
