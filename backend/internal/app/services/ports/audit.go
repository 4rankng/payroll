package ports

import (
	"context"

	"api-server/internal/domain"
)

// AuditPort defines the interface for audit operations
// This allows domains to log audit events without coupling to infrastructure
type AuditPort interface {
	LogAction(ctx context.Context, action string, entityType string, entityID uint, metadata map[string]interface{}) error
	GetAuditLogs(ctx context.Context, filters map[string]interface{}) ([]*domain.AuditLog, error)
}
