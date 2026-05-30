package bank

import (
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/asset"
)

type Handler struct {
	bankService *asset.BankService
}

func NewHandler(bankService *asset.BankService) *Handler {
	return &Handler{
		bankService: bankService,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Helper function to convert bank to response DTO
func (h *Handler) buildBankResponse(bank *domain.Bank) dto.BankResponse {
	return dto.BankResponse{
		ID:         bank.ID,
		BranchName: bank.BranchName,
		BankCode:   bank.BankCode,
		SwiftCode:  bank.SwiftCode,
		Bin:        bank.Bin,
	}
}

// CreateBank creates a new bank
// @Summary Create a new bank
// @Description Create a new bank with the provided information
// @Tags banks
// @Accept json
// @Produce json
// @Param bank body dto.CreateBankRequest true "Bank creation data"
// @Success 201 {object} dto.BankResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /banks [post]
func (h *Handler) CreateBank(c *gin.Context) {
	var req dto.CreateBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	bank := &domain.Bank{
		BranchName: req.BranchName,
		BankCode:   derefString(req.BankCode),
		SwiftCode:  derefString(req.SwiftCode),
		Bin:        derefString(req.Bin),
	}

	createdBank, err := h.bankService.CreateBank(c.Request.Context(), bank, userID.(uint))
	if err != nil {
		if domain.IsConflictError(err) {
			response.Conflict(c, err.Error())
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgFailedToCreateBankVN)
		return
	}

	bankResponse := h.buildBankResponse(createdBank)
	response.SuccessCreated(c, bankResponse, constants.MsgBankCreatedSuccessfullyVN)
}

// GetBank retrieves a bank by ID
// @Summary Get a bank by ID
// @Description Get detailed information about a specific bank
// @Tags banks
// @Accept json
// @Produce json
// @Param id path int true "Bank ID"
// @Success 200 {object} dto.BankResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /banks/{id} [get]
func (h *Handler) GetBank(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidBankIDVN)
		return
	}

	bank, err := h.bankService.GetBank(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgBankNotFoundVN)
			return
		}
		response.InternalServerError(c, constants.MsgFailedToRetrieveBankVN)
		return
	}

	bankResponse := h.buildBankResponse(bank)
	response.Success(c, bankResponse, constants.MsgBankRetrievedSuccessfullyVN)
}

// UpdateBank updates an existing bank
// @Summary Update a bank
// @Description Update bank information
// @Tags banks
// @Accept json
// @Produce json
// @Param id path int true "Bank ID"
// @Param bank body dto.UpdateBankRequest true "Bank update data"
// @Success 200 {object} dto.BankResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /banks/{id} [put]
func (h *Handler) UpdateBank(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidBankIDVN)
		return
	}

	var req dto.UpdateBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Build update data map
	updateData := make(map[string]interface{})
	if req.BranchName != nil {
		updateData["branch_name"] = *req.BranchName
	}
	if req.BankCode != nil {
		updateData["bank_code"] = *req.BankCode
	}
	if req.SwiftCode != nil {
		updateData["swift_code"] = *req.SwiftCode
	}
	if req.Bin != nil {
		updateData["bin"] = *req.Bin
	}
	if req.Status != nil {
		updateData["status"] = *req.Status
	}

	updatedBank, err := h.bankService.UpdateBank(c.Request.Context(), uint(id), updateData, userID.(uint))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgBankNotFoundVN)
			return
		}
		if domain.IsConflictError(err) {
			response.Conflict(c, err.Error())
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgFailedToUpdateBankVN)
		return
	}

	bankResponse := h.buildBankResponse(updatedBank)
	response.Success(c, bankResponse, constants.MsgBankUpdatedSuccessfullyVN)
}

// DeleteBank soft deletes a bank by setting its status to inactive
// @Summary Delete a bank
// @Description Soft delete a bank by setting its status to inactive
// @Tags banks
// @Accept json
// @Produce json
// @Param id path int true "Bank ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /banks/{id} [delete]
func (h *Handler) DeleteBank(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidBankIDVN)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	err = h.bankService.DeleteBank(c.Request.Context(), uint(id), userID.(uint))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgBankNotFoundVN)
			return
		}
		response.InternalServerError(c, constants.MsgFailedToDeleteBankVN)
		return
	}

	response.Success(c, nil, constants.MsgBankDeletedSuccessfullyVN)
}
