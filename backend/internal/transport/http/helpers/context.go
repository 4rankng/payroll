package helpers

import (
	"errors"

	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

var (
	// ErrUserNotFound is returned when user ID is not found in context
	ErrUserNotFound = errors.New("user not found in context")
	// ErrInvalidUserID is returned when user ID in context is invalid
	ErrInvalidUserID = errors.New("invalid user ID in context")
	// ErrRoleNotFound is returned when user role is not found in context
	ErrRoleNotFound = errors.New("user role not found in context")
	// ErrInvalidRole is returned when user role in context is invalid
	ErrInvalidRole = errors.New("invalid user role in context")
)

// GetUserID extracts the user ID from the Gin context.
// Returns an error if the user ID is not found or is invalid.
func GetUserID(c *gin.Context) (uint, error) {
	userIDInterface, exists := c.Get(constants.CtxUserID)
	if !exists {
		return 0, ErrUserNotFound
	}

	userID, ok := userIDInterface.(uint)
	if !ok || userID == 0 {
		return 0, ErrInvalidUserID
	}

	return userID, nil
}

// GetUserRole extracts the user role from the Gin context.
// Returns an error if the role is not found or is invalid.
func GetUserRole(c *gin.Context) (string, error) {
	roleInterface, exists := c.Get(constants.CtxUserRole)
	if !exists {
		return "", ErrRoleNotFound
	}

	role, ok := roleInterface.(string)
	if !ok || role == "" {
		return "", ErrInvalidRole
	}

	return role, nil
}

// GetUserIDOrRespond extracts the user ID from context and sends an error response if not found.
// Returns the user ID and true if successful, or 0 and false if not found (response already sent).
// This is a convenience method for handlers that need to extract user ID and respond with error if not found.
func GetUserIDOrRespond(c *gin.Context) (uint, bool) {
	userID, err := GetUserID(c)
	if err != nil {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return 0, false
	}
	return userID, true
}

// GetUserRoleOrRespond extracts the user role from context and sends an error response if not found.
// Returns the role and true if successful, or empty string and false if not found (response already sent).
// This is a convenience method for handlers that need to extract user role and respond with error if not found.
func GetUserRoleOrRespond(c *gin.Context) (string, bool) {
	role, err := GetUserRole(c)
	if err != nil {
		response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
		return "", false
	}
	return role, true
}

// MustGetUserID extracts the user ID from context and panics if not found.
// This should only be used in contexts where the user ID is guaranteed to exist (after auth middleware).
// For most cases, use GetUserID or GetUserIDOrRespond instead.
func MustGetUserID(c *gin.Context) uint {
	userID, err := GetUserID(c)
	if err != nil {
		panic("user ID not found in context: " + err.Error())
	}
	return userID
}

// MustGetUserRole extracts the user role from context and panics if not found.
// This should only be used in contexts where the role is guaranteed to exist (after auth middleware).
// For most cases, use GetUserRole or GetUserRoleOrRespond instead.
func MustGetUserRole(c *gin.Context) string {
	role, err := GetUserRole(c)
	if err != nil {
		panic("user role not found in context: " + err.Error())
	}
	return role
}
