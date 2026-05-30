package workers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/advance_payment"
	"api-server/internal/domain"
)

// ImportJobWorker processes import jobs asynchronously
type ImportJobWorker struct {
	logger            *slog.Logger
	assetRepo         domain.AssetRepository
	importService     *advance_payment.Service
	importProgressSvc *advance_payment.ImportProgressService
	storagePath       string
}

// NewImportJobWorker creates a new ImportJobWorker
func NewImportJobWorker(
	assetRepo domain.AssetRepository,
	importService *advance_payment.Service,
	importProgressSvc *advance_payment.ImportProgressService,
	storagePath string,
) *ImportJobWorker {
	return &ImportJobWorker{
		logger:            slog.Default().With("component", "ImportJobWorker"),
		assetRepo:         assetRepo,
		importService:     importService,
		importProgressSvc: importProgressSvc,
		storagePath:       storagePath,
	}
}

// ProcessJob processes an import job (called by Asynq handler)
func (w *ImportJobWorker) ProcessJob(ctx context.Context, jobID uint) error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("panic in import job processing", "job_id", jobID, "panic", r)
		}
	}()

	w.processJob(ctx, jobID)
	return nil
}

func (w *ImportJobWorker) processJob(ctx context.Context, jobID uint) {

	// Get and validate asset
	asset, err := w.getAndValidateAsset(ctx, jobID)
	if err != nil {
		return
	}

	// Auto-resolve for_month (metadata was always NULL)
	forMonth := advance_payment.GetCurrentMonth()
	if forMonth == "" {
		w.markJobFailed(ctx, asset, "Failed to auto-resolve for_month")
		return
	}

	w.logger.Info("starting import job processing", "job_id", jobID, "filename", asset.Filename, "for_month", forMonth)

	// Initialize import progress tracking
	if err := w.importProgressSvc.StartImport(ctx, asset.ID, forMonth); err != nil {
		w.markJobFailed(ctx, asset, fmt.Sprintf("Failed to start import: %v", err))
		return
	}

	w.logger.Info("import job started", "job_id", jobID, "filename", asset.Filename, "for_month", forMonth)

	// Process the Excel file
	result, err := w.processExcelFile(ctx, asset, forMonth)
	if err != nil {
		w.markJobFailed(ctx, asset, fmt.Sprintf("Failed to process file: %v", err))
		return
	}

	// Mark job as completed
	w.markJobCompleted(ctx, asset, result)

	w.logJobSuccess(jobID, result)
}

// getAndValidateAsset retrieves and validates the asset
func (w *ImportJobWorker) getAndValidateAsset(ctx context.Context, jobID uint) (*domain.Asset, error) {
	asset, err := w.assetRepo.GetByID(ctx, jobID)
	if err != nil {
		w.logger.Error("failed to get asset", "job_id", jobID, "error", err)
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}
	return asset, nil
}

// processExcelFile processes the Excel file and returns the result
func (w *ImportJobWorker) processExcelFile(ctx context.Context, asset *domain.Asset, forMonth string) (*dto.ImportFlexPayFileResult, error) {
	// Build full file path
	filePath := filepath.Join(w.storagePath, asset.FilePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Open the Excel file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			w.logger.Error("failed to close file", "error", closeErr)
		}
	}()

	// Parse with excelize
	xlsxFile, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Excel file: %w", err)
	}
	defer func() {
		if closeErr := xlsxFile.Close(); closeErr != nil {
			w.logger.Error("failed to close Excel file", "error", closeErr)
		}
	}()

	// Get createdBy from asset
	createdBy := asset.UploadedBy

	// Create progress callback for real-time updates
	progressCB := func(totalRows, processedRows int) {
		if err := w.importProgressSvc.UpdateProgress(ctx, asset.ID, totalRows, processedRows); err != nil {
			w.logger.Warn("failed to update progress", "job_id", asset.ID, "error", err)
		}
	}

	// Call the actual import service. asset.ID is the idempotency key —
	// re-enqueued duplicate uploads short-circuit inside the service via
	// advance_payment_imports.
	result, err := w.importService.ImportFlexPayFile(ctx, xlsxFile, forMonth, asset.ID, createdBy, progressCB)
	if err != nil {
		return nil, fmt.Errorf("failed to import flex pay file: %w", err)
	}

	// Mark as completed in import progress service
	if err := w.importProgressSvc.CompleteImport(ctx, asset.ID, result); err != nil {
		w.logger.Warn("failed to complete import progress", "job_id", asset.ID, "error", err)
	}

	return result, nil
}

func (w *ImportJobWorker) markJobFailed(ctx context.Context, asset *domain.Asset, errMsg string) {
	w.logger.Error("import job failed", "job_id", asset.ID, "error", errMsg)

	if err := w.importProgressSvc.MarkAsFailed(ctx, asset.ID, errMsg); err != nil {
		w.logger.Error("failed to mark import as failed", "job_id", asset.ID, "error", err)
	}
}

// markJobCompleted logs the completion of the job
func (w *ImportJobWorker) markJobCompleted(ctx context.Context, asset *domain.Asset, result *dto.ImportFlexPayFileResult) {
	w.logger.Info("import job completed",
		"job_id", asset.ID,
		"filename", asset.Filename,
		"total_rows", result.TotalRows,
		"employees_created", result.EmployeesCreated)
}

// logJobSuccess logs the successful completion of the job
func (w *ImportJobWorker) logJobSuccess(jobID uint, result *dto.ImportFlexPayFileResult) {
	w.logger.Info("import job completed successfully",
		"job_id", jobID,
		"total_rows", result.TotalRows,
		"employees_created", result.EmployeesCreated)
}
