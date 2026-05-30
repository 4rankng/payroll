package context

import (
	"context"
)

// Context keys for audit information
type auditContextKey string

const (
	auditIPAddressKey auditContextKey = "audit_ip_address"
	auditUserAgentKey auditContextKey = "audit_user_agent"
	auditUserIDKey    auditContextKey = "audit_user_id"
	auditFullNameKey  auditContextKey = "audit_full_name"
	StandardUserIDKey auditContextKey = "user_id"
	StandardIPKey     auditContextKey = "ip_address"
	StandardUAKey     auditContextKey = "user_agent"
)

// WithIPAddress adds IP address to the context for audit logging
func WithIPAddress(ctx context.Context, ipAddress string) context.Context {
	return context.WithValue(ctx, auditIPAddressKey, ipAddress)
}

// WithUserAgent adds user agent to the context for audit logging
func WithUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, auditUserAgentKey, userAgent)
}

// WithAuditInfo adds both IP address and user agent to the context for audit logging
func WithAuditInfo(ctx context.Context, ipAddress, userAgent string) context.Context {
	ctx = WithIPAddress(ctx, ipAddress)
	ctx = WithUserAgent(ctx, userAgent)
	return ctx
}

// WithFullName adds user full name to the context for audit logging
func WithFullName(ctx context.Context, fullName string) context.Context {
	return context.WithValue(ctx, auditFullNameKey, fullName)
}

// GetIPAddress extracts IP address from context for audit logging
func GetIPAddress(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if ipAddress, ok := ctx.Value(auditIPAddressKey).(string); ok {
		return ipAddress
	}

	return ""
}

// GetUserAgent extracts user agent from context for audit logging
func GetUserAgent(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if userAgent, ok := ctx.Value(auditUserAgentKey).(string); ok {
		return userAgent
	}

	return ""
}

// WithUserID adds user ID to the context for audit logging
func WithUserID(ctx context.Context, userID uint) context.Context {
	ctx = context.WithValue(ctx, auditUserIDKey, userID)
	// Mirror the standard key so older code paths reading it directly still work.
	ctx = context.WithValue(ctx, StandardUserIDKey, userID)
	return ctx
}

// GetUserID extracts user ID from context for audit logging
func GetUserID(ctx context.Context) *uint {
	if ctx == nil {
		return nil
	}

	if userID, ok := ctx.Value(auditUserIDKey).(uint); ok {
		return &userID
	}

	// Also check the standard key
	if userID, ok := ctx.Value(StandardUserIDKey).(uint); ok {
		return &userID
	}

	return nil
}

// GetUserIDOrZero extracts user ID from context, returning 0 if not present
func GetUserIDOrZero(ctx context.Context) uint {
	if id := GetUserID(ctx); id != nil {
		return *id
	}
	return 0
}

// GetFullName extracts user full name from context for audit logging
func GetFullName(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if fullName, ok := ctx.Value(auditFullNameKey).(string); ok {
		return fullName
	}

	return ""
}
