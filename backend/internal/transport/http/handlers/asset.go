package handlers

import (
	"fmt"
	"net/url"
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/asset"
)

type AssetHandler struct {
	assetService *asset.AssetService
}

func NewAssetHandler(assetService *asset.AssetService) *AssetHandler {
	return &AssetHandler{
		assetService: assetService,
	}
}

func (h *AssetHandler) UploadAsset(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Parse form data
	var form dto.AssetUploadForm
	if err := c.ShouldBind(&form); err != nil {
		response.BadRequest(c, constants.MsgInvalidFormDataVN)
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, constants.MsgNoFileUploadedAssetVN)
		return
	}
	defer func() {
		_ = file.Close() // Ignore error on defer
	}()

	// Create upload request
	uploadReq := domain.AssetUploadRequest{
		UploadType: form.UploadType,
	}

	// Upload asset
	asset, err := h.assetService.UploadAsset(c.Request.Context(), file, header, uploadReq, userID.(uint))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Convert to response DTO
	assetResponse := dto.AssetUploadResponse{
		ID:         asset.ID,
		Filename:   asset.Filename,
		UploadType: asset.UploadType,
		UploadedBy: asset.UploadedBy,
		CreatedAt:  asset.CreatedAt,
	}

	response.SuccessCreated(c, assetResponse, constants.MsgAssetUploadedSuccessfullyVN)
}

func (h *AssetHandler) GetAsset(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgAssetIDMustBeValidNumberVN)
		return
	}

	asset, err := h.assetService.GetAsset(c.Request.Context(), uint(id))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	assetResponse := dto.AssetResponse{
		ID:         asset.ID,
		Filename:   asset.Filename,
		UploadType: asset.UploadType,
		UploadedBy: asset.UploadedBy,
		CreatedAt:  asset.CreatedAt,
	}

	response.Success(c, assetResponse, constants.MsgAssetRetrievedSuccessfullyVN)
}

func (h *AssetHandler) ListAssets(c *gin.Context) {
	var req dto.AssetListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	if req.SortBy == "" {
		req.SortBy = "created_at"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Build filters
	filters := domain.AssetFilters{
		Limit:     req.Limit,
		Offset:    req.Offset,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	if req.UploadType != "" {
		filters.UploadType = &req.UploadType
	}
	if req.UploadedBy != 0 {
		filters.UploadedBy = &req.UploadedBy
	}

	assets, err := h.assetService.ListAssets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListAssetsVN)
		return
	}

	total, err := h.assetService.CountAssets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountAssetsVN)
		return
	}

	// Convert to response DTOs
	var assetResponses []dto.AssetResponse
	for _, asset := range assets {
		assetResponses = append(assetResponses, dto.AssetResponse{
			ID:         asset.ID,
			Filename:   asset.Filename,
			UploadType: asset.UploadType,
			UploadedBy: asset.UploadedBy,
			CreatedAt:  asset.CreatedAt,
		})
	}

	// Calculate pagination
	page := (filters.Offset / filters.Limit) + 1
	totalPages := (int(total) + filters.Limit - 1) / filters.Limit

	if len(assetResponses) == 0 {
		pagination := response.Pagination{
			Page:         page,
			PageSize:     filters.Limit,
			TotalPages:   totalPages,
			TotalRecords: int(total),
		}
		response.SuccessEmptyWithPagination(c, constants.MsgNoAssetsFoundVN, pagination)
		return
	}

	pagination := response.Pagination{
		Page:         page,
		PageSize:     filters.Limit,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	message := constants.MsgAssetsRetrievedSuccessfullyVN
	if page > 1 {
		message = fmt.Sprintf(constants.MsgPageOfAssetsRetrievedVN, strconv.Itoa(page))
	}

	response.SuccessWithPagination(c, assetResponses, message, pagination)
}

// DownloadAsset serves the actual file for download
func (h *AssetHandler) DownloadAsset(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgAssetIDMustBeValidNumberVN)
		return
	}

	// Get asset metadata first
	asset, err := h.assetService.GetAsset(c.Request.Context(), uint(id))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Check if file exists and get full path
	exists, fullPath, err := h.assetService.CheckFileExists(asset.FilePath)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToAccessAssetFileVN)
		return
	}
	if !exists {
		response.NotFound(c, constants.MsgAssetFileNotFoundVN)
		return
	}

	// Serve the file
	// Sanitize filename to prevent header injection attacks
	safeFilename := url.QueryEscape(asset.Filename)
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", safeFilename))
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Expires", "0")
	c.Header("Cache-Control", "must-revalidate")
	c.Header("Pragma", "public")

	// Stream the file
	c.File(fullPath)
}
