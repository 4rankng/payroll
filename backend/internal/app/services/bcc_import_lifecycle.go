package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

const maxBCCUploadSize = 10 << 20

type deferBCCTerminalMetadataKey struct{}

func (s *BCCImportService) ensureEmployeeUserAccount(ctx context.Context, employeeID uint) error {
	if s.employeeService == nil {
		return errors.New("employee service is not configured")
	}
	return s.employeeService.EnsureEmployeeUserAccount(ctx, employeeID)
}

// AcceptUpload durably stores a BCC upload and returns before spreadsheet
// processing starts. The idempotency key belongs to one uploader and one exact
// project/month/file fingerprint.
func (s *BCCImportService) AcceptUpload(
	ctx context.Context,
	fileData io.Reader,
	filename string,
	projectID uint,
	uploaderID uint,
	uploaderRole string,
	forMonth string,
	includeFlexibleEmployees bool,
	idempotencyKey string,
) (*BCCImportResult, error) {
	data, err := io.ReadAll(io.LimitReader(fileData, maxBCCUploadSize+1))
	if err != nil {
		return nil, fmt.Errorf("BCCImportService.AcceptUpload: read file: %w", err)
	}
	if len(data) > maxBCCUploadSize {
		return nil, fmt.Errorf("file quá lớn (tối đa 10MB)")
	}
	if _, _, err := parseForMonth(forMonth); err != nil {
		return nil, fmt.Errorf("tháng không hợp lệ: %w", err)
	}

	fingerprint := bccRequestFingerprint(projectID, forMonth, includeFlexibleEmployees, data)
	if existing, lookupErr := s.importJobRepo.GetByIdempotencyKey(ctx, uploaderID, idempotencyKey); lookupErr == nil {
		if existing.RequestFingerprint != fingerprint {
			return nil, ErrBCCIdempotencyConflict
		}
		return s.resultForAsset(ctx, existing.AssetID)
	} else if !domain.IsNotFoundError(lookupErr) {
		return nil, lookupErr
	}

	stored, err := s.fileStorage.StoreBytes(data, filename, domain.UploadTypePartnerBCCImport)
	if err != nil {
		return nil, fmt.Errorf("BCCImportService.AcceptUpload: store file: %w", err)
	}

	now := clock.Now()
	stats := BCCImportStats{
		ProjectID:    projectID,
		OriginalName: filename,
		ForMonth:     forMonth,
		Status:       domain.TimesheetImportStatusPending,
	}
	metadata, err := json.Marshal(stats)
	if err != nil {
		_ = s.fileStorage.Delete(stored.FilePath)
		return nil, fmt.Errorf("BCCImportService.AcceptUpload: metadata: %w", err)
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(data))
	activeScope := fmt.Sprintf("%d:%s", projectID, forMonth)
	asset := &domain.Asset{
		Filename:   stored.OriginalFilename,
		FilePath:   stored.FilePath,
		UploadType: domain.UploadTypePartnerBCCImport,
		Checksum:   &checksum,
		UploadedBy: uploaderID,
		Metadata:   stringPointer(string(metadata)),
	}

	err = s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		created, createErr := s.assetRepo.Create(txCtx, asset)
		if createErr != nil {
			return createErr
		}
		asset = created
		return s.importJobRepo.Create(txCtx, &domain.TimesheetImportJob{
			AssetID:                  asset.ID,
			ProjectID:                projectID,
			ForMonth:                 forMonth,
			UploadedBy:               uploaderID,
			UploaderRole:             uploaderRole,
			IncludeFlexibleEmployees: includeFlexibleEmployees,
			Status:                   domain.TimesheetImportStatusPending,
			IdempotencyKey:           idempotencyKey,
			RequestFingerprint:       fingerprint,
			ActiveScopeKey:           &activeScope,
			CreatedAt:                now,
			UpdatedAt:                now,
		})
	})
	if err != nil {
		_ = s.fileStorage.Delete(stored.FilePath)
		if existing, lookupErr := s.importJobRepo.GetByIdempotencyKey(ctx, uploaderID, idempotencyKey); lookupErr == nil {
			if existing.RequestFingerprint != fingerprint {
				return nil, ErrBCCIdempotencyConflict
			}
			return s.resultForAsset(ctx, existing.AssetID)
		}
		if strings.Contains(strings.ToLower(err.Error()), "active_scope") ||
			strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, ErrBCCImportScopeBusy
		}
		return nil, err
	}

	if enqueueErr := s.importEnqueuer.EnqueueBCCImport(asset.ID); enqueueErr != nil {
		// The durable pending row is an outbox. The periodic recovery task will
		// enqueue it after a transient Redis/queue outage.
		slog.Error("BCCImport: initial enqueue failed; recovery will retry",
			"asset_id", asset.ID, "error", enqueueErr)
	}
	return buildResult(stats, asset.ID, uploaderID, asset.CreatedAt), nil
}

// ProcessPendingJob claims and processes one durable import. Duplicate Asynq
// deliveries are harmless because only one pending/expired job can be claimed.
func (s *BCCImportService) ProcessPendingJob(ctx context.Context, assetID uint) error {
	job, err := s.importJobRepo.GetByAssetID(ctx, assetID)
	if err != nil {
		return err
	}
	if job.IsTerminal() {
		return nil
	}
	attempt, claimed, err := s.importJobRepo.Claim(ctx, assetID, clock.Now().Add(15*time.Minute))
	if err != nil || !claimed {
		return err
	}

	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return s.importJobRepo.Fail(ctx, assetID, attempt, err.Error(), clock.Now())
		}
		_ = s.importJobRepo.ReleaseForRetry(ctx, assetID, attempt, err.Error())
		return err
	}
	if asset.Metadata != nil {
		var persisted BCCImportStats
		if json.Unmarshal([]byte(*asset.Metadata), &persisted) == nil {
			switch persisted.Status {
			case domain.TimesheetImportStatusCompleted:
				return s.importJobRepo.Complete(ctx, assetID, attempt, clock.Now())
			case domain.TimesheetImportStatusFailed:
				return s.importJobRepo.Fail(ctx, assetID, attempt, FirstErrorReason(persisted.ErrorDetail), clock.Now())
			}
		}
	}
	data, err := os.ReadFile(s.fileStorage.GetFilePath(asset.FilePath))
	if err != nil {
		result := failedBCCImportResult(
			asset,
			job,
			fmt.Sprintf("không thể đọc tệp BCC đã lưu: %v", err),
		)
		return s.finalizeImportJob(ctx, assetID, attempt, result)
	}
	processingStats := BCCImportStats{
		ProjectID:    job.ProjectID,
		OriginalName: asset.Filename,
		ForMonth:     job.ForMonth,
		Status:       domain.TimesheetImportStatusProcessing,
	}
	if err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.importJobRepo.LockProcessingAttempt(txCtx, assetID, attempt); err != nil {
			return err
		}
		return s.updateAssetMetadata(txCtx, assetID, &processingStats)
	}); err != nil {
		_ = s.importJobRepo.ReleaseForRetry(ctx, assetID, attempt, err.Error())
		return err
	}

	var result *BCCImportResult
	rollbackAndFinalize := false
	processErr := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.importJobRepo.LockProcessingAttempt(txCtx, assetID, attempt); err != nil {
			return err
		}
		processingCtx := context.WithValue(txCtx, deferBCCTerminalMetadataKey{}, true)
		var importErr error
		result, importErr = s.processAssetData(
			processingCtx,
			data,
			asset.Filename,
			job.ProjectID,
			job.UploadedBy,
			job.UploaderRole,
			asset,
			job.ForMonth,
			job.IncludeFlexibleEmployees,
		)
		if importErr != nil {
			rollbackAndFinalize = result != nil
			return importErr
		}
		return s.finalizeImportJob(txCtx, assetID, attempt, result)
	})
	if processErr != nil {
		if rollbackAndFinalize && result != nil {
			if finalErr := s.finalizeImportJob(ctx, assetID, attempt, result); finalErr != nil {
				_ = s.importJobRepo.ReleaseForRetry(ctx, assetID, attempt, finalErr.Error())
				return finalErr
			}
			return nil
		}
		_ = s.importJobRepo.ReleaseForRetry(ctx, assetID, attempt, processErr.Error())
		return processErr
	}
	return nil
}

func (s *BCCImportService) finalizeImportJob(
	ctx context.Context,
	assetID uint,
	attempt uint,
	result *BCCImportResult,
) error {
	if result == nil {
		return fmt.Errorf("BCCImportService.finalizeImportJob: missing result")
	}
	processedAt := clock.Now()
	return s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.importJobRepo.LockProcessingAttempt(txCtx, assetID, attempt); err != nil {
			return err
		}
		stats := BCCImportStats{
			ProjectID:    result.ProjectID,
			OriginalName: result.OriginalName,
			ForMonth:     result.ForMonth,
			Status:       result.Status,
			TotalRows:    result.TotalRows,
			CreatedCount: result.CreatedCount,
			SkippedCount: result.SkippedCount,
			ErrorCount:   result.ErrorCount,
			ErrorDetail:  result.ErrorDetail,
			ProcessedAt:  result.ProcessedAt,
		}
		if err := s.updateAssetMetadata(txCtx, assetID, &stats); err != nil {
			return err
		}
		if result.Status == domain.TimesheetImportStatusFailed {
			return s.importJobRepo.Fail(txCtx, assetID, attempt, FirstErrorReason(result.ErrorDetail), processedAt)
		}
		return s.importJobRepo.Complete(txCtx, assetID, attempt, processedAt)
	})
}

func (s *BCCImportService) RecoverPendingJobs(ctx context.Context) error {
	jobs, err := s.importJobRepo.ListRecoverable(ctx, 100, clock.Now())
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err := s.importEnqueuer.EnqueueBCCImport(job.AssetID); err != nil {
			slog.Warn("BCCImport: recovery enqueue failed", "asset_id", job.AssetID, "error", err)
		}
	}
	return nil
}

func (s *BCCImportService) completeSkippedBCCImport(
	ctx context.Context,
	createdAsset *domain.Asset,
	uploaderID uint,
	stats BCCImportStats,
) (*BCCImportResult, error) {
	now := clock.Now()
	stats.Status = domain.TimesheetImportStatusCompleted
	stats.CreatedCount = 0
	stats.ErrorCount = 0
	stats.ErrorDetail = marshalErrors(nil)
	stats.ProcessedAt = &now
	if err := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); err != nil {
		return nil, fmt.Errorf("cập nhật kết quả BCC: %w", err)
	}
	return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt), nil
}

// ProcessUpload saves the BCC file, parses it, and creates timesheets.
// forMonth is "YYYY-MM" from the frontend; day numbers in the Excel belong to this month.
// Conflict policy: missing rows are created, pending rows are replaced, and reviewed/paid rows are skipped.
func (s *BCCImportService) ProcessUpload(
	ctx context.Context,
	fileData io.Reader,
	filename string,
	projectID uint,
	uploaderID uint,
	uploaderRole string,
	forMonth string,
) (*BCCImportResult, error) {
	// 1. Read file bytes (needed for both storage and parsing).
	data, err := io.ReadAll(io.LimitReader(fileData, maxBCCUploadSize+1))
	if err != nil {
		return nil, fmt.Errorf("BCCImportService.ProcessUpload: read file: %w", err)
	}
	if len(data) > maxBCCUploadSize {
		return nil, fmt.Errorf("BCCImportService.ProcessUpload: file quá lớn (tối đa 10MB)")
	}

	// 2. Save raw file to storage.
	stored, err := s.fileStorage.StoreBytes(data, filename, domain.UploadTypePartnerBCCImport)
	if err != nil {
		return nil, fmt.Errorf("BCCImportService.ProcessUpload: store file: %w", err)
	}

	// 3. Create asset record. On DB failure, clean up the orphaned file.
	var createdAsset *domain.Asset
	asset := &domain.Asset{
		Filename:   stored.OriginalFilename,
		FilePath:   stored.FilePath,
		UploadType: domain.UploadTypePartnerBCCImport,
		UploadedBy: uploaderID,
	}
	createdAsset, err = s.assetRepo.Create(ctx, asset)
	if err != nil {
		_ = s.fileStorage.Delete(stored.FilePath)
		return nil, fmt.Errorf("BCCImportService.ProcessUpload: create asset: %w", err)
	}

	return s.processAssetData(ctx, data, filename, projectID, uploaderID, uploaderRole, createdAsset, forMonth, false)
}

func (s *BCCImportService) updateAssetMetadata(ctx context.Context, assetID uint, meta *bccImportMetadata) error {
	if deferTerminal, _ := ctx.Value(deferBCCTerminalMetadataKey{}).(bool); deferTerminal &&
		(meta.Status == domain.TimesheetImportStatusCompleted ||
			meta.Status == domain.TimesheetImportStatusFailed) {
		return nil
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("BCCImport: failed to marshal metadata: %w", err)
	}
	if err := s.assetRepo.UpdateMetadata(ctx, assetID, string(b)); err != nil {
		return fmt.Errorf("BCCImport: failed to update asset metadata (id=%d): %w", assetID, err)
	}
	return nil
}

// prepareImportContext performs the shared setup: month parsing, distributed lock,
// and payrate lookup. Returns the context and a cleanup function that must be
// deferred by the caller to release the lock.
func (s *BCCImportService) prepareImportContext(ctx context.Context, projectID uint, effectiveMonth string) (*bccImportContext, func(), error) {
	// Use time.Local so the lookup date matches the MySQL driver's location: the DSN uses
	// loc=Local, so cfg.Loc == time.Local. clock.Now() is pinned to Asia/Ho_Chi_Minh, which
	// diverges from time.Local in UTC containers (e.g. the scratch prod image) and shifts the
	// day boundary by 7h — breaking `from_date <= monthStart` when the import month starts on
	// the payrate's effective date. Same loc=Local precedent as timesheet_entry_table.go.
	loc := time.Local
	year, month, err := parseForMonth(effectiveMonth)
	if err != nil {
		return nil, nil, fmt.Errorf("tháng không hợp lệ: %v", err)
	}
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, loc)

	lockValue, lockErr := s.acquireImportLock(ctx, projectID, effectiveMonth)
	if lockErr != nil {
		return nil, nil, lockErr
	}
	release := func() { s.releaseImportLock(ctx, projectID, effectiveMonth, lockValue) }

	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, projectID, monthStart)
	if err != nil {
		release()
		return nil, nil, fmt.Errorf("không tìm thấy bảng lương cho dự án: %v", err)
	}
	flatRates, err := payrate.Payrate.Flatten()
	if err != nil {
		release()
		return nil, nil, fmt.Errorf("lỗi phân tích cấu hình lương: %v", err)
	}

	return &bccImportContext{
		year:       year,
		month:      month,
		monthStart: monthStart,
		flatRates:  flatRates,
	}, release, nil
}
