package infrastructure

import (
	"context"
	"log/slog"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// AuditService provides centralized audit logging for file operations
type AuditService struct {
	eventBus domain.EventBus
	logger   *slog.Logger
}

// NewAuditService creates a new AuditService
func NewAuditService(eventBus domain.EventBus) *AuditService {
	return &AuditService{
		eventBus: eventBus,
		logger:   observability.GetLogger(),
	}
}

// LogFileImport logs a data import event
func (s *AuditService) LogFileImport(ctx context.Context, dataType string, recordCount int, fileName string) error {
	if s.eventBus == nil {
		return nil
	}

	event := domain.NewDataImportedEvent(ctx, dataType, recordCount, fileName)
	if err := s.eventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("failed to emit import audit event",
			"data_type", dataType,
			"file_name", fileName,
			"error", err,
		)
		return err
	}

	s.logger.Info("import audit event emitted",
		"data_type", dataType,
		"record_count", recordCount,
		"file_name", fileName,
	)

	return nil
}

// LogFileExport logs a data export event
func (s *AuditService) LogFileExport(ctx context.Context, dataType string, recordCount int, fileName string) error {
	if s.eventBus == nil {
		return nil
	}

	event := domain.NewDataExportedEvent(ctx, dataType, recordCount, fileName)
	if err := s.eventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("failed to emit export audit event",
			"data_type", dataType,
			"file_name", fileName,
			"error", err,
		)
		return err
	}

	s.logger.Info("export audit event emitted",
		"data_type", dataType,
		"record_count", recordCount,
		"file_name", fileName,
	)

	return nil
}

// LogAssetCreated logs an asset creation event
func (s *AuditService) LogAssetCreated(ctx context.Context, asset *domain.Asset) error {
	if s.eventBus == nil {
		return nil
	}

	event := domain.NewAssetCreatedEvent(ctx, asset)
	if err := s.eventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("failed to emit asset created audit event",
			"asset_id", asset.ID,
			"file_name", asset.Filename,
			"error", err,
		)
		return err
	}

	s.logger.Info("asset created audit event emitted",
		"asset_id", asset.ID,
		"file_name", asset.Filename,
	)

	return nil
}

// LogAssetDeleted logs an asset deletion event
func (s *AuditService) LogAssetDeleted(ctx context.Context, asset *domain.Asset) error {
	if s.eventBus == nil {
		return nil
	}

	event := domain.NewAssetDeletedEvent(ctx, asset)
	if err := s.eventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("failed to emit asset deleted audit event",
			"asset_id", asset.ID,
			"file_name", asset.Filename,
			"error", err,
		)
		return err
	}

	s.logger.Info("asset deleted audit event emitted",
		"asset_id", asset.ID,
		"file_name", asset.Filename,
	)

	return nil
}
