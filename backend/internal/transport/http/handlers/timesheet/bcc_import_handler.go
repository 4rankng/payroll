package timesheet

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	appservices "api-server/internal/app/services"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/project"
	"api-server/internal/domain"
	"api-server/internal/infra/storage"
	auditctx "api-server/internal/pkg/context"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// bccImportItem is the DTO returned by list/detail endpoints.
type bccImportItem struct {
	appservices.BCCImportStats
	ID         uint      `json:"id"`
	UploadedBy uint      `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// bccMetadata mirrors the metadata stored in Asset.Metadata.
type bccMetadata = appservices.BCCImportStats

// BCCImportHandler handles partner BCC attendance file import endpoints.
type BCCImportHandler struct {
	bccImportService     *appservices.BCCImportService
	assetRepo            domain.AssetRepository
	fileStorage          storage.FileStorage
	projectPermissionSvc *project.ProjectPermissionService
	auditService         *infrastructure.AuditService
}

func NewBCCImportHandler(
	svc *appservices.BCCImportService,
	assetRepo domain.AssetRepository,
	fs storage.FileStorage,
	projectPermissionSvc *project.ProjectPermissionService,
	auditService *infrastructure.AuditService,
) *BCCImportHandler {
	return &BCCImportHandler{
		bccImportService:     svc,
		assetRepo:            assetRepo,
		fileStorage:          fs,
		projectPermissionSvc: projectPermissionSvc,
		auditService:         auditService,
	}
}

// UploadBCC handles POST /timesheets/partner-import
func (h *BCCImportHandler) UploadBCC(c *gin.Context) {
	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}
	userRole, ok := helpers.GetUserRoleOrRespond(c)
	if !ok {
		return
	}

	projectIDStr := c.PostForm("project_id")
	if projectIDStr == "" {
		response.BadRequest(c, "project_id là bắt buộc")
		return
	}
	projectID64, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil || projectID64 == 0 {
		response.BadRequest(c, "project_id không hợp lệ")
		return
	}
	projectID := uint(projectID64)

	forMonth := c.PostForm("for_month")
	if forMonth == "" {
		response.BadRequest(c, "for_month là bắt buộc (YYYY-MM)")
		return
	}

	// Partner users must have write access to the target project.
	if userRole == string(domain.RolePartner) {
		canModify, err := h.projectPermissionSvc.CanUserModifyProject(c.Request.Context(), projectID, userID)
		if err != nil || !canModify {
			response.Forbidden(c, "Bạn không có quyền import cho dự án này")
			return
		}
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "Vui lòng tải lên file BCC (.xlsx)")
		return
	}
	defer func() { _ = file.Close() }()

	if !isExcelFile(header.Filename) {
		response.BadRequest(c, "File phải có định dạng .xlsx")
		return
	}

	importCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Minute)
	defer cancel()

	result, err := h.bccImportService.ProcessUpload(
		importCtx,
		file,
		header.Filename,
		projectID,
		userID,
		userRole,
		forMonth,
	)
	if err != nil && result == nil {
		response.InternalServerError(c, err.Error())
		return
	}

	// Capture audit context before spawning goroutines — gin recycles the
	// request context after the handler returns, and header.Filename is
	// borrowed from the multipart form which is tied to the request body.
	auditCtx := auditctx.WithUserID(context.Background(), userID)
	auditCtx = auditctx.WithIPAddress(auditCtx, c.ClientIP())
	auditCtx = auditctx.WithUserAgent(auditCtx, c.GetHeader("User-Agent"))
	filename := header.Filename

	if result.Status == "failed" {
		if h.auditService != nil {
			go func() {
				_ = h.auditService.LogFileImport(auditCtx, "partner_bcc_import_failed", result.ErrorCount, filename)
			}()
		}
		response.Success(c, result, "Import thất bại: "+appservices.FirstErrorReason(result.ErrorDetail))
		return
	}

	if h.auditService != nil {
		go func() {
			_ = h.auditService.LogFileImport(auditCtx, "partner_bcc_import", result.CreatedCount, filename)
		}()
	}
	msg := buildImportSummaryMessage(result)
	response.Success(c, result, msg)
}

// ListPartnerImports handles GET /timesheets/partner-import
func (h *BCCImportHandler) ListPartnerImports(c *gin.Context) {
	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}
	userRole, ok := helpers.GetUserRoleOrRespond(c)
	if !ok {
		return
	}

	bccType := domain.UploadTypePartnerBCCImport
	filters := domain.AssetFilters{
		UploadType:    &bccType,
		MetadataQuery: make(map[string]string),
	}

	if pid := c.Query("project_id"); pid != "" {
		filters.MetadataQuery["project_id"] = pid
	}
	if fm := c.Query("for_month"); fm != "" {
		filters.MetadataQuery["for_month"] = fm
	}

	// Partners only see their own uploads; admin sees all.
	if userRole == string(domain.RolePartner) {
		filters.UploadedBy = &userID
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	filters.Limit = pageSize
	filters.Offset = (page - 1) * pageSize

	assets, err := h.assetRepo.List(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể tải danh sách lịch sử import")
		return
	}

	total, err := h.assetRepo.Count(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể tải danh sách lịch sử import")
		return
	}

	items := make([]*bccImportItem, 0, len(assets))
	for _, a := range assets {
		item := assetToImportItem(a)
		if item != nil {
			items = append(items, item)
		}
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}
	response.SuccessWithPagination(c, items, "Lấy danh sách thành công", pagination)
}

// GetPartnerImport handles GET /timesheets/partner-import/:id
func (h *BCCImportHandler) GetPartnerImport(c *gin.Context) {
	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}
	userRole, ok := helpers.GetUserRoleOrRespond(c)
	if !ok {
		return
	}

	id, ok := parseBCCImportID(c)
	if !ok {
		return
	}
	asset, err := h.assetRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	if asset.UploadType != domain.UploadTypePartnerBCCImport {
		response.NotFound(c, "Không tìm thấy file import")
		return
	}

	if userRole == string(domain.RolePartner) && asset.UploadedBy != userID {
		response.Forbidden(c, "Bạn không có quyền truy cập file import này")
		return
	}

	item := assetToImportItem(asset)
	if item == nil {
		response.NotFound(c, "Không tìm thấy file import")
		return
	}
	response.Success(c, item, "Lấy thông tin thành công")
}

// DownloadPartnerImport handles GET /timesheets/partner-import/:id/download
func (h *BCCImportHandler) DownloadPartnerImport(c *gin.Context) {
	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}
	userRole, ok := helpers.GetUserRoleOrRespond(c)
	if !ok {
		return
	}

	id, ok := parseBCCImportID(c)
	if !ok {
		return
	}
	asset, err := h.assetRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	if asset.UploadType != domain.UploadTypePartnerBCCImport {
		response.NotFound(c, "Không tìm thấy file import")
		return
	}

	if userRole == string(domain.RolePartner) && asset.UploadedBy != userID {
		response.Forbidden(c, "Bạn không có quyền tải file import này")
		return
	}

	if !h.fileStorage.Exists(asset.FilePath) {
		response.NotFound(c, "File không còn tồn tại trên server")
		return
	}
	fullPath := h.fileStorage.GetFilePath(asset.FilePath)

	// Use original name from metadata if available.
	originalName := asset.Filename
	if asset.Metadata != nil {
		var meta bccMetadata
		if json.Unmarshal([]byte(*asset.Metadata), &meta) == nil && meta.OriginalName != "" {
			originalName = meta.OriginalName
		}
	}
	c.FileAttachment(fullPath, originalName)
}

// ─── helpers ────────────────────────────────────────────────────────────────

func parseBCCImportID(c *gin.Context) (uint, bool) {
	v, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || v == 0 {
		response.BadRequest(c, "ID không hợp lệ")
		return 0, false
	}
	return uint(v), true
}

func assetToImportItem(a *domain.Asset) *bccImportItem {
	if a.Metadata == nil {
		return nil
	}
	var meta bccMetadata
	if err := json.Unmarshal([]byte(*a.Metadata), &meta); err != nil {
		return nil
	}
	return &bccImportItem{
		BCCImportStats: meta,
		ID:             a.ID,
		UploadedBy:     a.UploadedBy,
		CreatedAt:      a.CreatedAt,
	}
}

func buildImportSummaryMessage(r *appservices.BCCImportResult) string {
	return "Import thành công: " +
		strconv.Itoa(r.CreatedCount) + " timesheet tạo mới, " +
		strconv.Itoa(r.SkippedCount) + " bỏ qua, " +
		strconv.Itoa(r.ErrorCount) + " lỗi"
}
