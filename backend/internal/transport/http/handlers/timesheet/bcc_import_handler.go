package timesheet

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	appservices "api-server/internal/app/services"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/project"
	"api-server/internal/domain"
	"api-server/internal/infra/storage"
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
	if parsedMonth, err := time.Parse("2006-01", forMonth); err != nil ||
		parsedMonth.Format("2006-01") != forMonth {
		response.BadRequest(c, "for_month không hợp lệ (YYYY-MM)")
		return
	}

	includeFlexibleEmployees, err := strconv.ParseBool(c.DefaultPostForm("include_flexible_employees", "false"))
	if err != nil {
		response.BadRequest(c, "include_flexible_employees không hợp lệ")
		return
	}
	if includeFlexibleEmployees && userRole != string(domain.RoleAdmin) {
		response.Forbidden(c, "Chỉ quản trị viên có thể import nhân viên lương linh hoạt")
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
	if header.Size > 10<<20 {
		response.BadRequest(c, "File quá lớn (tối đa 10MB)")
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" || len(idempotencyKey) > 128 {
		response.BadRequest(c, "Idempotency-Key là bắt buộc và không được vượt quá 128 ký tự")
		return
	}

	result, err := h.bccImportService.AcceptUpload(
		c.Request.Context(),
		file,
		header.Filename,
		projectID,
		userID,
		userRole,
		forMonth,
		includeFlexibleEmployees,
		idempotencyKey,
	)
	if errors.Is(err, appservices.ErrBCCIdempotencyConflict) {
		c.JSON(http.StatusUnprocessableEntity, response.ErrorResponse{
			Status: "error", HTTPStatus: http.StatusUnprocessableEntity, Message: err.Error(),
		})
		return
	}
	if errors.Is(err, appservices.ErrBCCImportScopeBusy) {
		response.Conflict(c, err.Error())
		return
	}
	if err != nil {
		slog.Error("BCC import acceptance failed", "project_id", projectID, "user_id", userID, "error", err)
		response.InternalServerError(c, "Không thể nhận tệp BCC. Vui lòng thử lại.")
		return
	}

	c.Header("Location", "/api/v1/timesheets/partner-import/"+strconv.FormatUint(uint64(result.ID), 10))
	c.JSON(http.StatusAccepted, response.SuccessResponse{
		Status:  "success",
		Data:    result,
		Message: "Đã nhận tệp BCC. Hệ thống đang xử lý trong nền.",
	})
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
