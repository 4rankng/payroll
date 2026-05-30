package infrastructure

import (
	"context"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         uint
	Action     string
	EntityType string
	EntityID   uint
	UserID     uint
	Metadata   map[string]interface{}
	CreatedAt  interface{}
}

// AuditPort defines the interface for audit logging operations
type AuditPort interface {
	Log(ctx context.Context, action string, entityType string, entityID uint, metadata map[string]interface{}) error
	GetLogs(ctx context.Context, filters map[string]interface{}) ([]*AuditLog, error)
}
