package services

import (
	"time"
)

// BCCImportStats holds common import statistics shared between service result,
// stored metadata, and handler DTOs.
type BCCImportStats struct {
	ProjectID    uint       `json:"project_id"`
	OriginalName string     `json:"original_name"`
	ForMonth     string     `json:"for_month"`
	Status       string     `json:"status"`
	TotalRows    int        `json:"total_rows"`
	CreatedCount int        `json:"created_count"`
	SkippedCount int        `json:"skipped_count"`
	ErrorCount   int        `json:"error_count"`
	ErrorDetail  *string    `json:"error_detail,omitempty"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
}

// BCCImportResult is returned by ProcessUpload and serialized to the API response.
type BCCImportResult struct {
	BCCImportStats
	ID         uint      `json:"id"`
	UploadedBy uint      `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// buildResult creates a BCCImportResult from stats and asset fields.
func buildResult(stats BCCImportStats, id, uploaderID uint, createdAt time.Time) *BCCImportResult {
	return &BCCImportResult{
		BCCImportStats: stats,
		ID:             id,
		UploadedBy:     uploaderID,
		CreatedAt:      createdAt,
	}
}

// bccImportContext holds the shared setup results for BCC import flows.
// Both single-position and multi-position flows perform the same month parsing,
// lock acquisition, and payrate resolution — this struct consolidates that setup.
// resolver is the shared per-date payrate authority (PAYR-36): every path
// resolves rates through it by each entry's own work date, and position
// validation goes through its month union.
type bccImportContext struct {
	year       int
	month      time.Month
	monthStart time.Time
	resolver   *payrateResolver
}

// posCorrection records a pending position update to apply after the STK transaction.
// Applying outside the transaction via ProjectEmployeeService.UpdateAssignmentPosition
// ensures cache invalidation, event publishing, and timesheet recalculation while
// using a targeted column update to avoid full-row Save() overwrites.
type posCorrection struct {
	assignmentID uint
	employeeID   uint
	oldPosition  string
	newPosition  string
	employeeName string
}
