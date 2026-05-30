package domain

import (
	"errors"
)

var (
	// Common domain errors
	ErrNotFound        = errors.New("không tìm thấy tài nguyên")
	ErrAlreadyExists   = errors.New("tài nguyên đã tồn tại")
	ErrInvalidInput    = errors.New("đầu vào không hợp lệ")
	ErrUnauthorized    = errors.New("không được phép")
	ErrForbidden       = errors.New("bị cấm")
	ErrValidation      = errors.New("xác thực thất bại")
	ErrInternal        = errors.New("lỗi máy chủ nội bộ")
	ErrConflict        = errors.New("xung đột tài nguyên")
	ErrTooManyRequests = errors.New("quá nhiều yêu cầu")
)

// DomainError represents a domain-specific error with additional context
type DomainError struct {
	Type    string
	Code    string // Error code
	Message string
	Cause   error
	Context map[string]interface{}
}

func (e *DomainError) Error() string {
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

func (e *DomainError) WithContext(key string, value interface{}) *DomainError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// Error constructors
func NewNotFoundError(message string) *DomainError {
	return &DomainError{
		Type:    "NOT_FOUND",
		Code:    "NOT_FOUND",
		Message: message,
		Cause:   ErrNotFound,
	}
}

func NewNotFoundErrorWithCode(code, message string) *DomainError {
	return &DomainError{
		Type:    "NOT_FOUND",
		Code:    code,
		Message: message,
		Cause:   ErrNotFound,
	}
}

func NewValidationError(message string) *DomainError {
	return &DomainError{
		Type:    "VALIDATION_ERROR",
		Code:    "VALIDATION_ERROR",
		Message: message,
		Cause:   ErrValidation,
	}
}

func NewUnauthorizedError(message string) *DomainError {
	return &DomainError{
		Type:    "UNAUTHORIZED",
		Code:    "UNAUTHORIZED",
		Message: message,
		Cause:   ErrUnauthorized,
	}
}

func NewUnauthorizedErrorWithCode(code, message string) *DomainError {
	return &DomainError{
		Type:    "UNAUTHORIZED",
		Code:    code,
		Message: message,
		Cause:   ErrUnauthorized,
	}
}

func NewForbiddenError(message string) *DomainError {
	return &DomainError{
		Type:    "FORBIDDEN",
		Code:    "FORBIDDEN",
		Message: message,
		Cause:   ErrForbidden,
	}
}

func NewForbiddenErrorWithCode(code, message string) *DomainError {
	return &DomainError{
		Type:    "FORBIDDEN",
		Code:    code,
		Message: message,
		Cause:   ErrForbidden,
	}
}

func NewConflictError(message string) *DomainError {
	return &DomainError{
		Type:    "CONFLICT",
		Code:    "CONFLICT",
		Message: message,
		Cause:   ErrConflict,
	}
}

func NewInternalError(message string, cause error) *DomainError {
	return &DomainError{
		Type:    "INTERNAL_ERROR",
		Code:    "INTERNAL_ERROR",
		Message: message,
		Cause:   cause,
	}
}

func NewInternalErrorWithCode(code, message string, cause error) *DomainError {
	return &DomainError{
		Type:    "INTERNAL_ERROR",
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Error type checkers
func IsNotFoundError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Type == "NOT_FOUND"
	}
	return errors.Is(err, ErrNotFound)
}

func IsValidationError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Type == "VALIDATION_ERROR"
	}
	return errors.Is(err, ErrValidation)
}

func IsUnauthorizedError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Type == "UNAUTHORIZED"
	}
	return errors.Is(err, ErrUnauthorized)
}

func IsForbiddenError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Type == "FORBIDDEN"
	}
	return errors.Is(err, ErrForbidden)
}

func IsConflictError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Type == "CONFLICT"
	}
	return errors.Is(err, ErrConflict)
}

func IsInternalError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Type == "INTERNAL_ERROR"
	}
	return errors.Is(err, ErrInternal)
}
