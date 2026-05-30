package lender

import (
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/loan"
)

type Handler struct {
	lenderService *loan.LenderService
}

func NewHandler(lenderService *loan.LenderService) *Handler {
	return &Handler{
		lenderService: lenderService,
	}
}

// buildLenderResponse converts lender to response DTO
func (h *Handler) buildLenderResponse(lender *domain.Lender) dto.LenderResponse {
	return dto.LenderResponse{
		ID:                lender.ID,
		Name:              lender.Name,
		CCCD:              lender.CCCD,
		Email:             lender.Email,
		Mobile:            lender.Mobile,
		Notes:             lender.Notes,
		BankID:            lender.BankID,
		BankAccountNumber: lender.BankAccountNumber,
		BankAccountName:   lender.BankAccountName,
		CreatedAt:         lender.CreatedAt,
		UpdatedAt:         lender.UpdatedAt,
	}
}

// CreateLender creates a new lender
func (h *Handler) CreateLender(c *gin.Context) {
	var req dto.CreateLenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	_, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	lender := &domain.Lender{
		Name:              req.Name,
		CCCD:              req.CCCD,
		Email:             req.Email,
		Mobile:            req.Mobile,
		Notes:             req.Notes,
		BankID:            req.BankID,
		BankAccountNumber: req.BankAccountNumber,
		BankAccountName:   req.BankAccountName,
	}

	createdLender, err := h.lenderService.CreateLender(c.Request.Context(), lender)
	if err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "Không thể tạo người cho vay")
		return
	}

	lenderResponse := h.buildLenderResponse(createdLender)
	response.SuccessCreated(c, lenderResponse, constants.MsgLenderCreatedVN)
}

// GetLender retrieves a lender by ID
func (h *Handler) GetLender(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidLenderIDVN)
		return
	}

	lender, err := h.lenderService.GetLender(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgLenderNotFoundVN)
			return
		}
		response.InternalServerError(c, "Không thể lấy thông tin người cho vay")
		return
	}

	lenderResponse := h.buildLenderResponse(lender)
	response.Success(c, lenderResponse, constants.MsgLenderFetchedVN)
}

// ListLenders retrieves a paginated list of lenders
func (h *Handler) ListLenders(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "100"))
	sortBy := c.DefaultQuery("sortBy", "created_at")
	sortOrder := c.DefaultQuery("sortOrder", "DESC")
	search := c.Query("search")

	// Validate pagination parameters
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	if page < 1 {
		page = 1
	}

	filters := domain.LenderFilters{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	if search != "" {
		filters.Search = &search
	}

	lenders, total, err := h.lenderService.ListLenders(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể lấy danh sách người cho vay")
		return
	}

	// Convert to response DTOs
	lenderResponses := make([]dto.LenderResponse, len(lenders))
	for i, lender := range lenders {
		lenderResponses[i] = h.buildLenderResponse(lender)
	}

	// Build pagination and respond in documented format
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

	response.SuccessWithPagination(c, lenderResponses, constants.MsgLenderListFetchedVN, pagination)
}

// UpdateLender updates an existing lender
func (h *Handler) UpdateLender(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidLenderIDVN)
		return
	}

	var req dto.UpdateLenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	// Build update data map
	updateData := make(map[string]interface{})
	if req.Name != nil {
		updateData["name"] = *req.Name
	}
	if req.CCCD != nil {
		updateData["cccd"] = *req.CCCD
	}
	if req.Email != nil {
		updateData["email"] = *req.Email
	}
	if req.Mobile != nil {
		updateData["mobile"] = *req.Mobile
	}
	if req.Notes != nil {
		updateData["notes"] = *req.Notes
	}
	if req.BankID != nil {
		updateData["bank_id"] = *req.BankID
	}
	if req.BankAccountNumber != nil {
		updateData["bank_account_number"] = *req.BankAccountNumber
	}
	if req.BankAccountName != nil {
		updateData["bank_account_name"] = *req.BankAccountName
	}

	updatedLender, err := h.lenderService.UpdateLender(c.Request.Context(), uint(id), updateData)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgLenderNotFoundVN)
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "Không thể cập nhật người cho vay")
		return
	}

	lenderResponse := h.buildLenderResponse(updatedLender)
	response.Success(c, lenderResponse, constants.MsgLenderUpdatedVN)
}

// DeleteLender soft deletes a lender
func (h *Handler) DeleteLender(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidLenderIDVN)
		return
	}

	err = h.lenderService.DeleteLender(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgLenderNotFoundVN)
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "Không thể xóa người cho vay")
		return
	}

	response.Success(c, nil, constants.MsgLenderDeletedVN)
}
