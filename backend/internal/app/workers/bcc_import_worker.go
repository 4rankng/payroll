package workers

import (
	"context"

	appservices "api-server/internal/app/services"
	"api-server/internal/app/services/infrastructure"
	auditctx "api-server/internal/pkg/context"
)

type BCCImportWorker struct {
	service      *appservices.BCCImportService
	auditService *infrastructure.AuditService
}

func NewBCCImportWorker(
	service *appservices.BCCImportService,
	auditService *infrastructure.AuditService,
) *BCCImportWorker {
	return &BCCImportWorker{service: service, auditService: auditService}
}

func (w *BCCImportWorker) ProcessJob(ctx context.Context, assetID uint) error {
	if err := w.service.ProcessPendingJob(ctx, assetID); err != nil {
		return err
	}
	if w.auditService == nil {
		return nil
	}
	result, pending, err := w.service.PendingTerminalAudit(ctx, assetID)
	if err != nil || !pending {
		return err
	}
	dataType := "partner_bcc_import"
	recordCount := result.CreatedCount
	if result.Status == "failed" {
		dataType = "partner_bcc_import_failed"
		recordCount = result.ErrorCount
	}
	ctx = auditctx.WithUserID(ctx, result.UploadedBy)
	if err := w.auditService.LogFileImport(ctx, dataType, recordCount, result.OriginalName); err != nil {
		return err
	}
	return w.service.MarkTerminalAuditLogged(ctx, assetID)
}

func (w *BCCImportWorker) Recover(ctx context.Context) error {
	return w.service.RecoverPendingJobs(ctx)
}
