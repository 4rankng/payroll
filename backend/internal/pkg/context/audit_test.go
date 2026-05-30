package context

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithIPAddress(t *testing.T) {
	ctx := context.Background()
	ipAddress := "192.168.1.100"

	newCtx := WithIPAddress(ctx, ipAddress)
	assert.NotNil(t, newCtx)

	retrievedIP := GetIPAddress(newCtx)
	assert.Equal(t, ipAddress, retrievedIP)
}

func TestWithUserAgent(t *testing.T) {
	ctx := context.Background()
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

	newCtx := WithUserAgent(ctx, userAgent)
	assert.NotNil(t, newCtx)

	retrievedUA := GetUserAgent(newCtx)
	assert.Equal(t, userAgent, retrievedUA)
}

func TestWithAuditInfo(t *testing.T) {
	ctx := context.Background()
	ipAddress := "192.168.1.100"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

	newCtx := WithAuditInfo(ctx, ipAddress, userAgent)
	assert.NotNil(t, newCtx)

	retrievedIP := GetIPAddress(newCtx)
	assert.Equal(t, ipAddress, retrievedIP)

	retrievedUA := GetUserAgent(newCtx)
	assert.Equal(t, userAgent, retrievedUA)
}

func TestGetIPAddress_NilContext(t *testing.T) {
	var ctx context.Context
	retrievedIP := GetIPAddress(ctx)
	assert.Equal(t, "", retrievedIP)
}

func TestGetIPAddress_NoValue(t *testing.T) {
	ctx := context.Background()
	retrievedIP := GetIPAddress(ctx)
	assert.Equal(t, "", retrievedIP)
}

func TestGetUserAgent_NilContext(t *testing.T) {
	var ctx context.Context
	retrievedUA := GetUserAgent(ctx)
	assert.Equal(t, "", retrievedUA)
}

func TestGetUserAgent_NoValue(t *testing.T) {
	ctx := context.Background()
	retrievedUA := GetUserAgent(ctx)
	assert.Equal(t, "", retrievedUA)
}

func TestWithUserID(t *testing.T) {
	ctx := context.Background()
	var userID uint = 12345

	newCtx := WithUserID(ctx, userID)
	assert.NotNil(t, newCtx)

	retrievedUserID := GetUserID(newCtx)
	assert.NotNil(t, retrievedUserID)
	assert.Equal(t, userID, *retrievedUserID)
}

func TestGetUserID_NilContext(t *testing.T) {
	var ctx context.Context
	retrievedUserID := GetUserID(ctx)
	assert.Nil(t, retrievedUserID)
}

func TestGetUserID_NoValue(t *testing.T) {
	ctx := context.Background()
	retrievedUserID := GetUserID(ctx)
	assert.Nil(t, retrievedUserID)
}

func TestWithUserID_StandardKey(t *testing.T) {
	ctx := context.Background()
	var userID uint = 67890

	// Set using WithUserID, which sets both keys
	newCtx := WithUserID(ctx, userID)

	// Should be retrievable via GetUserID
	retrievedUserID := GetUserID(newCtx)
	assert.NotNil(t, retrievedUserID)
	assert.Equal(t, userID, *retrievedUserID)

	// Should also have the standard key set
	if val, ok := newCtx.Value(StandardUserIDKey).(uint); ok {
		assert.Equal(t, userID, val)
	} else {
		t.Error("StandardUserIDKey not set correctly")
	}
}

func TestContextChaining(t *testing.T) {
	ctx := context.Background()

	// Chain multiple context values
	ctx = WithIPAddress(ctx, "10.0.0.1")
	ctx = WithUserAgent(ctx, "TestAgent/1.0")
	ctx = WithUserID(ctx, 999)

	// Verify all values are preserved
	assert.Equal(t, "10.0.0.1", GetIPAddress(ctx))
	assert.Equal(t, "TestAgent/1.0", GetUserAgent(ctx))
	assert.Equal(t, uint(999), *GetUserID(ctx))
}

func TestWithAuditInfo_WithExistingUserID(t *testing.T) {
	ctx := context.Background()
	var userID uint = 123

	// Add user ID first
	ctx = WithUserID(ctx, userID)

	// Add audit info
	ctx = WithAuditInfo(ctx, "172.16.0.1", "CustomAgent/2.0")

	// All values should be present
	assert.Equal(t, "172.16.0.1", GetIPAddress(ctx))
	assert.Equal(t, "CustomAgent/2.0", GetUserAgent(ctx))
	assert.Equal(t, userID, *GetUserID(ctx))
}
