package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const bccImportLockTTL = 30 * time.Second

func (s *BCCImportService) acquireImportLock(ctx context.Context, projectID uint, forMonth string) (string, error) {
	lockKey := fmt.Sprintf("bcc_import:lock:%d:%s", projectID, forMonth)
	lockValue := fmt.Sprintf("%d-%d", projectID, time.Now().UnixNano())
	acquired, err := s.redis.SetNX(ctx, lockKey, lockValue, bccImportLockTTL).Result()
	if err != nil {
		return "", fmt.Errorf("failed to acquire import lock: %w", err)
	}
	if !acquired {
		return "", fmt.Errorf("một thao tác import khác đang chạy cho dự án %d tháng %s, vui lòng thử lại sau", projectID, forMonth)
	}
	return lockValue, nil
}

func (s *BCCImportService) releaseImportLock(ctx context.Context, projectID uint, forMonth, lockValue string) {
	lockKey := fmt.Sprintf("bcc_import:lock:%d:%s", projectID, forMonth)
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := s.redis.Eval(ctx, script, []string{lockKey}, lockValue).Result()
	if err != nil {
		slog.Warn("BCCImport: failed to release import lock", "key", lockKey, "error", err)
	}
}
