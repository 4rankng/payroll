package dto

import "time"

// EmployeeImportResponse represents the response when starting an import
type EmployeeImportResponse struct {
	ImportID string `json:"import_id"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

// EmployeeImportStatus represents the status of an employee import job
type EmployeeImportStatus struct {
	ImportID      string     `json:"import_id"`
	Status        string     `json:"status"` // pending, processing, completed, failed
	TotalRows     int        `json:"total_rows"`
	ProcessedRows int        `json:"processed_rows"`
	CreatedCount  int        `json:"created_count"`
	UpdatedCount  int        `json:"updated_count"`
	ErrorCount    int        `json:"error_count"`
	Errors        []RowError `json:"errors"`
	Percentage    int        `json:"percentage"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// RowError represents an error for a specific row in the import
type RowError struct {
	RowNumber int    `json:"row_number"`
	CCCD      string `json:"cccd,omitempty"`
	Message   string `json:"message"`
}

// EmployeeImportRow represents a single row from the employee import Excel
type EmployeeImportRow struct {
	RowNumber       int
	Fullname        string
	CCCD            string
	Email           *string
	Mobile          string
	Address         string
	DateOfBirth     *string
	BankAccount     string
	BankAccountName string
	BankName        string
	ProjectCode     string
	Position        string
	StartDate       string
	PaymentSchedule string
}

// EmployeeImportProgress represents the progress data stored in Redis
type EmployeeImportProgress struct {
	ImportID      string     `json:"import_id"`
	Status        string     `json:"status"`
	TotalRows     int        `json:"total_rows"`
	ProcessedRows int        `json:"processed_rows"`
	CreatedCount  int        `json:"created_count"`
	UpdatedCount  int        `json:"updated_count"`
	ErrorCount    int        `json:"error_count"`
	Errors        []RowError `json:"errors"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	UserID        uint       `json:"user_id"`
	FilePath      string     `json:"file_path"`
}

// EmployeeImportResult represents the final result of an import job
type EmployeeImportResult struct {
	TotalRows    int        `json:"total_rows"`
	CreatedCount int        `json:"created_count"`
	UpdatedCount int        `json:"updated_count"`
	ErrorCount   int        `json:"error_count"`
	Errors       []RowError `json:"errors"`
}
