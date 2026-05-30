package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidationErrorWithCode(t *testing.T) {
	code := "TEST_CODE"
	message := "Test validation error message"

	err := NewValidationErrorWithCode(code, message)

	assert.NotNil(t, err)
	assert.Equal(t, "VALIDATION_ERROR", err.Type)
	assert.Equal(t, code, err.Code)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, ErrValidation, err.Cause)
}

func TestNewConflictErrorWithCode(t *testing.T) {
	code := "CONFLICT_CODE"
	message := "Test conflict error message"

	err := NewConflictErrorWithCode(code, message)

	assert.NotNil(t, err)
	assert.Equal(t, "CONFLICT", err.Type)
	assert.Equal(t, code, err.Code)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, ErrConflict, err.Cause)
}

func TestNewTimesheetExistsError(t *testing.T) {
	err := NewTimesheetExistsError()

	assert.NotNil(t, err)
	assert.Equal(t, "CONFLICT", err.Type)
	assert.Equal(t, ErrCodeTimesheetExists, err.Code)
	assert.Equal(t, "Timesheet entry already exists", err.Message)
	assert.Equal(t, ErrConflict, err.Cause)
}

func TestNewAssignmentOverlapError(t *testing.T) {
	message := "Assignment overlaps with existing assignment"
	err := NewAssignmentOverlapError(message)

	assert.NotNil(t, err)
	assert.Equal(t, "CONFLICT", err.Type)
	assert.Equal(t, ErrCodeAssignmentOverlap, err.Code)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, ErrConflict, err.Cause)
}

func TestNewInvalidDateRangeError(t *testing.T) {
	message := "Invalid date range"
	err := NewInvalidDateRangeError(message)

	assert.NotNil(t, err)
	assert.Equal(t, "VALIDATION_ERROR", err.Type)
	assert.Equal(t, ErrCodeInvalidDateRange, err.Code)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, ErrValidation, err.Cause)
}

func TestNewInvalidTransitionError(t *testing.T) {
	message := "Invalid state transition"
	err := NewInvalidTransitionError(message)

	assert.NotNil(t, err)
	assert.Equal(t, "VALIDATION_ERROR", err.Type)
	assert.Equal(t, ErrCodeInvalidTransition, err.Code)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, ErrValidation, err.Cause)
}
