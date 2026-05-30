package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError_Error(t *testing.T) {
	err := NewValidationError("test message")
	assert.Equal(t, "test message", err.Message)
}

func TestNotFoundError_Error(t *testing.T) {
	err := NewNotFoundError("test message")
	assert.Equal(t, "test message", err.Message)
}

func TestUnauthorizedError_Error(t *testing.T) {
	err := NewUnauthorizedError("test message")
	assert.Equal(t, "test message", err.Message)
}

func TestForbiddenError_Error(t *testing.T) {
	err := NewForbiddenError("test message")
	assert.Equal(t, "test message", err.Message)
}

func TestConflictError_Error(t *testing.T) {
	err := NewConflictError("test message")
	assert.Equal(t, "test message", err.Message)
}

func TestInternalError_Error(t *testing.T) {
	innerErr := errors.New("inner error")
	err := NewInternalError("test message", innerErr)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, innerErr, err.Cause)
}

func TestIsNotFoundError(t *testing.T) {
	validationErr := NewValidationError("message")
	assert.False(t, IsNotFoundError(validationErr))

	notFoundErr := NewNotFoundError("message")
	assert.True(t, IsNotFoundError(notFoundErr))
}

func TestIsValidationError(t *testing.T) {
	notFoundErr := NewNotFoundError("message")
	assert.False(t, IsValidationError(notFoundErr))

	validationErr := NewValidationError("message")
	assert.True(t, IsValidationError(validationErr))
}

func TestIsUnauthorizedError(t *testing.T) {
	notFoundErr := NewNotFoundError("message")
	assert.False(t, IsUnauthorizedError(notFoundErr))

	unauthorizedErr := NewUnauthorizedError("message")
	assert.True(t, IsUnauthorizedError(unauthorizedErr))
}

func TestIsForbiddenError(t *testing.T) {
	validationErr := NewValidationError("message")
	assert.False(t, IsForbiddenError(validationErr))

	forbiddenErr := NewForbiddenError("message")
	assert.True(t, IsForbiddenError(forbiddenErr))
}

func TestIsConflictError(t *testing.T) {
	validationErr := NewValidationError("message")
	assert.False(t, IsConflictError(validationErr))

	conflictErr := NewConflictError("message")
	assert.True(t, IsConflictError(conflictErr))
}

func TestIsInternalError(t *testing.T) {
	validationErr := NewValidationError("message")
	assert.False(t, IsInternalError(validationErr))

	innerErr := errors.New("inner error")
	internalErr := NewInternalError("message", innerErr)
	assert.True(t, IsInternalError(internalErr))
}
