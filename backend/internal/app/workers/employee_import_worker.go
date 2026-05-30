package workers

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/infrastructure"
)

// EmployeeImportWorker processes employee import jobs asynchronously
type EmployeeImportWorker struct {
	importService   *employee.ImportService
	progressService *infrastructure.EmployeeImportProgressService
	storagePath     string
	logger          *slog.Logger
}

// NewEmployeeImportWorker creates a new employee import worker
func NewEmployeeImportWorker(
	importService *employee.ImportService,
	progressService *infrastructure.EmployeeImportProgressService,
	storagePath string,
) *EmployeeImportWorker {
	return &EmployeeImportWorker{
		importService:   importService,
		progressService: progressService,
		storagePath:     storagePath,
		logger:          slog.Default().With("component", "EmployeeImportWorker"),
	}
}

// ProcessJob processes a single import job (called by Asynq handler)
func (w *EmployeeImportWorker) ProcessJob(ctx context.Context, importID string) error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("panic in employee import job processing", "import_id", importID, "panic", r)
		}
	}()

	w.processJob(ctx, importID)
	return nil
}

// processJob processes a single import job
func (w *EmployeeImportWorker) processJob(ctx context.Context, importID string) {
	w.logger.Info("processing employee import job", "import_id", importID)

	// Get import progress to retrieve file path
	progress, err := w.progressService.GetProgress(ctx, importID)
	if err != nil {
		w.logger.Error("failed to get import progress", "import_id", importID, "error", err)
		w.markJobFailed(ctx, importID, "Failed to get import progress")
		return
	}

	// Mark job as processing
	if err := w.progressService.MarkAsProcessing(ctx, importID); err != nil {
		w.logger.Error("failed to mark job as processing", "import_id", importID, "error", err)
	}

	// Build full file path
	filePath := filepath.Join(w.storagePath, progress.FilePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		w.logger.Error("file not found", "import_id", importID, "file_path", filePath)
		w.markJobFailed(ctx, importID, "Import file not found")
		return
	}

	// Open the Excel file
	xlsxFile, err := excelize.OpenFile(filePath)
	if err != nil {
		w.logger.Error("failed to open Excel file", "import_id", importID, "error", err)
		w.markJobFailed(ctx, importID, "Failed to open Excel file")
		return
	}
	defer func() { _ = xlsxFile.Close() }()

	// Parse Excel to get rows
	rows, totalRows, err := w.importService.ParseEmployeeExcel(ctx, xlsxFile)
	if err != nil {
		w.logger.Error("failed to parse Excel file", "import_id", importID, "error", err)
		w.markJobFailed(ctx, importID, "Failed to parse Excel file")
		return
	}

	// Update total rows count in progress
	if err := w.progressService.SetTotalRows(ctx, importID, totalRows); err != nil {
		w.logger.Warn("failed to update total rows", "import_id", importID, "error", err)
	}

	w.logger.Info("parsed Excel file", "import_id", importID, "total_rows", totalRows)

	// Handle empty file case
	if totalRows == 0 {
		w.logger.Warn("no data rows found in Excel file", "import_id", importID)
		w.markJobFailed(ctx, importID, "No data rows found in Excel file")
		return
	}

	// Process in chunks of 50
	chunkSize := 50
	createdCount := 0
	updatedCount := 0
	errorCount := 0
	var allErrors []dto.RowError

	for i := 0; i < len(rows); i += chunkSize {
		end := i + chunkSize
		if end > len(rows) {
			end = len(rows)
		}

		chunk := rows[i:end]
		created, updated, errors, rowErrors, err := w.importService.ProcessImportChunk(
			ctx,
			importID,
			chunk,
			progress.UserID,
			i+1, // start row number
		)

		if err != nil {
			w.logger.Error("failed to process chunk", "import_id", importID, "error", err)
			// Continue with next chunk despite error
			continue
		}

		createdCount += created
		updatedCount += updated
		errorCount += errors
		allErrors = append(allErrors, rowErrors...)

		// Update progress after each chunk
		if err := w.progressService.UpdateProgress(ctx, importID, end, createdCount, updatedCount, errorCount, allErrors); err != nil {
			w.logger.Warn("failed to update progress", "import_id", importID, "error", err)
		}

		w.logger.Debug("processed chunk",
			"import_id", importID,
			"chunk_start", i+1,
			"chunk_end", end,
			"created", created,
			"updated", updated,
			"errors", errors,
		)
	}

	// Mark job as completed
	w.logger.Info("employee import job completed",
		"import_id", importID,
		"total_rows", totalRows,
		"created_count", createdCount,
		"updated_count", updatedCount,
		"error_count", errorCount,
	)

	// Final progress update to mark as completed
	if err := w.progressService.UpdateProgress(ctx, importID, totalRows, createdCount, updatedCount, errorCount, allErrors); err != nil {
		w.logger.Warn("failed to update final progress", "import_id", importID, "error", err)
	}
}

// markJobFailed marks a job as failed
func (w *EmployeeImportWorker) markJobFailed(ctx context.Context, importID string, errorMessage string) {
	if err := w.progressService.MarkAsFailed(ctx, importID, errorMessage); err != nil {
		w.logger.Error("failed to mark job as failed", "import_id", importID, "error", err)
	}
}
