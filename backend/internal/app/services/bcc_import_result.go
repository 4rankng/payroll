package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"
	"api-server/internal/pkg/clock"
)

func failedBCCImportResult(
	asset *domain.Asset,
	job *domain.TimesheetImportJob,
	reason string,
) *BCCImportResult {
	now := clock.Now()
	detail := marshalErrors([]domain.ImportError{{Reason: reason}})
	stats := BCCImportStats{
		ProjectID:    job.ProjectID,
		OriginalName: asset.Filename,
		ForMonth:     job.ForMonth,
		Status:       domain.TimesheetImportStatusFailed,
		ErrorCount:   1,
		ErrorDetail:  detail,
		ProcessedAt:  &now,
	}
	return buildResult(stats, asset.ID, job.UploadedBy, asset.CreatedAt)
}

// finalizeBCCImport persists the final stats to the asset's import metadata
// and builds the API result for a finished (completed or failed) import.
func (s *BCCImportService) finalizeBCCImport(
	ctx context.Context,
	asset *domain.Asset,
	uploaderID uint,
	stats BCCImportStats,
) (*BCCImportResult, error) {
	if err := s.updateAssetMetadata(ctx, asset.ID, &stats); err != nil {
		slog.Error("BCCImport: metadata update failed for final stats", "asset_id", asset.ID, "error", err)
	}
	return buildResult(stats, asset.ID, uploaderID, asset.CreatedAt), nil
}

func (s *BCCImportService) failWithImportErrors(
	ctx context.Context,
	createdAsset *domain.Asset,
	uploaderID uint,
	stats BCCImportStats,
	importErrors []domain.ImportError,
) (*BCCImportResult, error) {
	detail := marshalErrors(importErrors)
	now := clock.Now()
	stats.Status = domain.TimesheetImportStatusFailed
	stats.CreatedCount = 0
	stats.ErrorCount = len(importErrors)
	stats.ErrorDetail = detail
	stats.ProcessedAt = &now
	if err := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); err != nil {
		slog.Error("BCCImport: metadata update failed for bulk validation failure", "asset_id", createdAsset.ID, "error", err)
	}
	return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt),
		fmt.Errorf("import failed: %s", FirstErrorReason(detail))
}

func (s *BCCImportService) resultForAsset(ctx context.Context, assetID uint) (*BCCImportResult, error) {
	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if asset.Metadata == nil {
		return nil, fmt.Errorf("BCC import %d has no metadata", assetID)
	}
	var stats BCCImportStats
	if err := json.Unmarshal([]byte(*asset.Metadata), &stats); err != nil {
		return nil, err
	}
	return buildResult(stats, asset.ID, asset.UploadedBy, asset.CreatedAt), nil
}

func (s *BCCImportService) PendingTerminalAudit(ctx context.Context, assetID uint) (*BCCImportResult, bool, error) {
	job, err := s.importJobRepo.GetByAssetID(ctx, assetID)
	if err != nil {
		return nil, false, err
	}
	if !job.IsTerminal() || job.AuditLoggedAt != nil {
		return nil, false, nil
	}
	result, err := s.resultForAsset(ctx, assetID)
	return result, true, err
}

func (s *BCCImportService) MarkTerminalAuditLogged(ctx context.Context, assetID uint) error {
	_, err := s.importJobRepo.MarkTerminalAuditLogged(ctx, assetID, clock.Now())
	return err
}

func importErrorsFromBulkFailures(
	failures []domainservices.BulkCreateFailure,
	employeeNames map[uint]string,
) []domain.ImportError {
	errors := make([]domain.ImportError, 0, len(failures))
	for _, failure := range failures {
		employeeName := employeeNames[failure.Request.EmployeeID]
		if employeeName == "" {
			employeeName = "Nhân viên chưa xác định"
		}
		errors = append(errors, domain.ImportError{
			Employee: employeeName,
			Reason:   fmt.Sprintf("ngày %s: %s", failure.Request.Date, safeBulkFailureReason(failure.Error)),
		})
	}
	return errors
}

func safeBulkFailureReason(reason string) string {
	normalized := strings.ToLower(reason)
	switch {
	case strings.Contains(normalized, "ngày trong tương lai"):
		return "Ngày chấm công chưa đến"
	case strings.Contains(normalized, "phê duyệt"), strings.Contains(normalized, "approved"):
		return "Bảng chấm công đã được phê duyệt"
	case strings.Contains(normalized, "thanh toán"), strings.Contains(normalized, "paid"):
		return "Bảng chấm công đã thanh toán"
	case strings.Contains(normalized, "trùng"), strings.Contains(normalized, "duplicate"):
		return "Dữ liệu đã tồn tại"
	case strings.Contains(normalized, "không tìm thấy mức lương"):
		return "Chưa cấu hình mức lương phù hợp cho ca làm việc"
	default:
		return "Không thể tạo bảng chấm công"
	}
}

func safeWeeklyPaymentParseError(err error) string {
	reason := strings.ToLower(err.Error())
	if strings.Contains(reason, "nằm ngoài kỳ nhập") {
		return "Ngày trong tệp nằm ngoài tháng đã chọn. Vui lòng kiểm tra tệp."
	}
	return "Không thể đọc cấu trúc tệp chấm công. Vui lòng dùng đúng mẫu tệp."
}

func bccRequestFingerprint(projectID uint, forMonth string, includeFlexibleEmployees bool, data []byte) string {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%d:%s:%t:", projectID, forMonth, includeFlexibleEmployees)
	_, _ = hash.Write(data)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func stringPointer(value string) *string { return &value }
