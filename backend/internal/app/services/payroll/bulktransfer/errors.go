package bulktransfer

import "fmt"

// ErrFileAlreadyProcessed indicates the file has already been processed
type ErrFileAlreadyProcessed struct {
	EntryCount int
}

func (e *ErrFileAlreadyProcessed) Error() string {
	return fmt.Sprintf("file has already been processed with %d ledger entries", e.EntryCount)
}

// ErrInvalidFileFormat indicates the file format is not recognized
type ErrInvalidFileFormat struct {
	Reason string
}

func (e *ErrInvalidFileFormat) Error() string {
	return fmt.Sprintf("invalid file format: %s", e.Reason)
}

// ErrRowValidation indicates a row failed validation
type ErrRowValidation struct {
	Row    int
	Field  string
	Reason string
}

func (e *ErrRowValidation) Error() string {
	return fmt.Sprintf("row %d validation failed for field %s: %s", e.Row, e.Field, e.Reason)
}

// ErrEmployeeNotFound indicates no employee matches the account number
type ErrEmployeeNotFound struct {
	AccountNumber string
}

func (e *ErrEmployeeNotFound) Error() string {
	return fmt.Sprintf("employee not found with account number: %s", e.AccountNumber)
}

// ErrTimesheetNotFound indicates no timesheets match the criteria
type ErrTimesheetNotFound struct {
	Criteria string
}

func (e *ErrTimesheetNotFound) Error() string {
	return fmt.Sprintf("no timesheets found: %s", e.Criteria)
}

// ErrTimesheetValidation indicates a timesheet validation failure
type ErrTimesheetValidation struct {
	TimesheetID uint
	Reason      string
}

func (e *ErrTimesheetValidation) Error() string {
	return fmt.Sprintf("timesheet %d validation failed: %s", e.TimesheetID, e.Reason)
}

// ErrInsufficientData indicates required data is missing
type ErrInsufficientData struct {
	Field string
}

func (e *ErrInsufficientData) Error() string {
	return fmt.Sprintf("required field missing: %s", e.Field)
}
