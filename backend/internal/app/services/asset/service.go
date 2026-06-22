package asset

import (
	"api-server/internal/constants"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/infra/storage"
)

type AssetService struct {
	AssetRepo    domain.AssetRepository
	FileStorage  storage.FileStorage
	MaxFileSize  int64
	AllowedTypes []string
	EventBus     domain.EventBus
}

func NewAssetService(
	assetRepo domain.AssetRepository,
	fileStorage storage.FileStorage,
	maxFileSize int64,
	eventBus domain.EventBus,
) *AssetService {
	return &AssetService{
		AssetRepo:   assetRepo,
		FileStorage: fileStorage,
		MaxFileSize: maxFileSize,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "application/pdf",
			"application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"text/plain"},
		EventBus: eventBus,
	}
}

func (s *AssetService) UploadAsset(ctx context.Context, file multipart.File, header *multipart.FileHeader, req domain.AssetUploadRequest, uploadedBy uint) (*domain.Asset, error) {
	// Validate upload type. upload_type is used as the first path segment when
	// storing the file, so it must be one of the fixed, traversal-free constants
	// to prevent arbitrary file writes outside the storage root.
	if !domain.IsValidUploadType(req.UploadType) {
		return nil, domain.NewValidationError(constants.MsgInvalidUploadTypeVN)
	}

	// Validate file size
	if err := storage.ValidateFileSize(header, s.MaxFileSize); err != nil {
		return nil, domain.NewValidationError(err.Error())
	}

	// Validate file type
	if err := storage.ValidateFileType(header); err != nil {
		return nil, domain.NewValidationError(err.Error())
	}

	var checksum *string

	// Calculate checksum to prevent duplicates for all upload types
	// Reset file pointer to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return nil, domain.NewInternalError(constants.MsgFailedToResetFilePointerForChecksumVN, err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, domain.NewInternalError(constants.MsgFailedToCalculateFileChecksumVN, err)
	}

	checksumValue := fmt.Sprintf("%x", hash.Sum(nil))
	checksum = &checksumValue

	var filePath string
	var isNewFile bool

	// Check if file with same checksum already exists for deduplication
	existingAsset, err := s.AssetRepo.GetByChecksum(ctx, checksumValue, req.UploadType)
	if err == nil {
		// File with same checksum exists, reuse the physical file path
		filePath = existingAsset.FilePath
		isNewFile = false
	} else if !domain.IsNotFoundError(err) {
		// If error is not "not found", it's a real error
		return nil, fmt.Errorf("failed to check for duplicate file: %w", err)
	} else {
		// File doesn't exist, store it
		// Reset file pointer again for storage
		if _, err := file.Seek(0, 0); err != nil {
			return nil, domain.NewInternalError(constants.MsgFailedToResetFilePointerForStorageVN, err)
		}

		storedFile, err := s.FileStorage.Store(file, header, req.UploadType)
		if err != nil {
			return nil, domain.NewInternalError(constants.MsgFailedToStoreFileVN, err)
		}
		filePath = storedFile.FilePath
		isNewFile = true
	}

	// Create new asset record with current filename (may reuse existing file path)
	asset := &domain.Asset{
		Filename:   header.Filename,
		FilePath:   filePath,
		UploadType: req.UploadType,
		Checksum:   checksum,
		UploadedBy: uploadedBy,
	}

	// Save to database
	createdAsset, err := s.AssetRepo.Create(ctx, asset)
	if err != nil {
		// Clean up file on database error only if we just stored it
		if isNewFile {
			if deleteErr := s.FileStorage.Delete(filePath); deleteErr != nil {
				observability.GetLogger().Warn("failed to clean up file on database error", "error", deleteErr)
			}
		}
		return nil, fmt.Errorf("failed to create asset record: %w", err)
	}

	// Publish domain event
	event := domain.NewAssetCreatedEvent(ctx, createdAsset)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		logger := observability.GetLogger()
		logger.Warn("Failed to publish AssetCreated event", "assetID", createdAsset.ID, "error", err)
	}

	return createdAsset, nil
}

// UploadAssetFromBytes stores raw bytes as an asset record (no multipart needed).
func (s *AssetService) UploadAssetFromBytes(ctx context.Context, data []byte, filename string, uploadType string, uploadedBy uint) (*domain.Asset, error) {
	if !domain.IsValidUploadType(uploadType) {
		return nil, domain.NewValidationError(constants.MsgInvalidUploadTypeVN)
	}

	storedFile, err := s.FileStorage.StoreBytes(data, filename, uploadType)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgFailedToStoreFileBytesVN, err)
	}

	// Compute checksum
	hash := sha256.New()
	hash.Write(data)
	checksumValue := fmt.Sprintf("%x", hash.Sum(nil))

	asset := &domain.Asset{
		Filename:   filename,
		FilePath:   storedFile.FilePath,
		UploadType: uploadType,
		Checksum:   &checksumValue,
		UploadedBy: uploadedBy,
	}

	createdAsset, err := s.AssetRepo.Create(ctx, asset)
	if err != nil {
		_ = s.FileStorage.Delete(storedFile.FilePath)
		return nil, fmt.Errorf("failed to create asset record: %w", err)
	}

	event := domain.NewAssetCreatedEvent(ctx, createdAsset)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		logger := observability.GetLogger()
		logger.Warn("Failed to publish AssetCreated event", "assetID", createdAsset.ID, "error", err)
	}

	return createdAsset, nil
}

func (s *AssetService) GetAsset(ctx context.Context, id uint) (*domain.Asset, error) {
	return s.AssetRepo.GetByID(ctx, id)
}

func (s *AssetService) ListAssets(ctx context.Context, filters domain.AssetFilters) ([]*domain.Asset, error) {
	return s.AssetRepo.List(ctx, filters)
}

func (s *AssetService) CountAssets(ctx context.Context, filters domain.AssetFilters) (int64, error) {
	return s.AssetRepo.Count(ctx, filters)
}

func (s *AssetService) CheckFileExists(filePath string) (bool, string, error) {
	exists := s.FileStorage.Exists(filePath)
	fullPath := s.FileStorage.GetFilePath(filePath)
	return exists, fullPath, nil
}

func (s *AssetService) CleanupOrphanedAssets(ctx context.Context, olderThan time.Time, performedBy uint) (int, error) {
	orphanedAssets, err := s.AssetRepo.FindOrphaned(ctx, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to find orphaned assets: %w", err)
	}

	cleanedCount := 0
	for _, asset := range orphanedAssets {
		// Only delete physical file if no other assets reference it
		count, err := s.AssetRepo.CountByFilePath(ctx, asset.FilePath)
		if err != nil {
			observability.GetLogger().Warn("failed to check for other assets using file",
				"filePath", asset.FilePath, "error", err)
		} else if count == 0 {
			// No other assets reference this file, safe to delete
			if err := s.FileStorage.Delete(asset.FilePath); err != nil {
				observability.GetLogger().Warn("failed to delete physical file",
					"filePath", asset.FilePath, "error", err)
			}
		}

		cleanedCount++
	}

	return cleanedCount, nil
}
