package domain

// Assignment-related error codes
const (
	ErrCodeAssignmentOverlap    = "ASSIGNMENT_OVERLAP"
	ErrCodeAssignmentReplaced   = "ASSIGNMENT_REPLACED"
	ErrCodeAssignmentTerminated = "ASSIGNMENT_TERMINATED"
	ErrCodeTimesheetExists      = "TIMESHEET_EXISTS"
	ErrCodeInvalidDateRange     = "INVALID_DATE_RANGE"
	ErrCodeInvalidTransition    = "INVALID_TRANSITION"
	ErrCodeConflictResolution   = "CONFLICT_RESOLUTION_FAILED"
)

// NewValidationErrorWithCode creates a validation error with an error code
func NewValidationErrorWithCode(code, message string) *DomainError {
	return &DomainError{
		Type:    "VALIDATION_ERROR",
		Code:    code,
		Message: message,
		Cause:   ErrValidation,
	}
}

// NewConflictErrorWithCode creates a conflict error with an error code
func NewConflictErrorWithCode(code, message string) *DomainError {
	return &DomainError{
		Type:    "CONFLICT",
		Code:    code,
		Message: message,
		Cause:   ErrConflict,
	}
}

// Common error builders for timesheet domain
func NewTimesheetExistsError() *DomainError {
	return NewConflictErrorWithCode(ErrCodeTimesheetExists, "Timesheet entry already exists")
}

func NewAssignmentOverlapError(message string) *DomainError {
	return NewConflictErrorWithCode(ErrCodeAssignmentOverlap, message)
}

func NewInvalidDateRangeError(message string) *DomainError {
	return NewValidationErrorWithCode(ErrCodeInvalidDateRange, message)
}

func NewInvalidTransitionError(message string) *DomainError {
	return NewValidationErrorWithCode(ErrCodeInvalidTransition, message)
}
