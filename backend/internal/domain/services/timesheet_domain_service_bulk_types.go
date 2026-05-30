package services

// PreviewError represents a single validation error tied to an employee
type PreviewError struct {
	EmployeeID uint   // 0 means not tied to a specific employee
	Date       string // "YYYY-MM-DD" — the specific date that caused the error, empty if not date-specific
	Message    string // Vietnamese error message
}
