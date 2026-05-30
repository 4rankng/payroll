package bank

import (
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ListBanks lists all banks with pagination and filtering
// @Summary List banks
// @Description Get a paginated list of banks with filtering options. Inactive banks are not returned by default.
// @Tags banks
// @Accept json
// @Produce json
// @Param page query int false "Page number, starts from 1" default(1)
// @Param pageSize query int false "Items per page (max: 100)" default(20)
// @Param sortBy query string false "Field to sort by" default(branch_name)
// @Param sortOrder query string false "Sort direction asc or desc" default(asc)
// @Param status query string false "Filter by status (active, inactive)"
// @Param search query string false "Search by branch name"
// @Param limit query int false "Legacy: Limit number of results" default(20)
// @Param offset query int false "Legacy: Offset for pagination" default(0)
// @Param sort_by query string false "Legacy: Sort by field" default(created_at)
// @Param sort_order query string false "Legacy: Sort order (asc/desc)" default(desc)
// @Success 200 {object} dto.ListBanksResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /banks [get]
func (h *Handler) ListBanks(c *gin.Context) {
	// Initialize filters with new pagination defaults
	filters := domain.BankFilters{
		Limit:     20, // default page size
		Offset:    0,  // default offset
		SortBy:    "branch_name",
		SortOrder: "asc",
	}

	// Handle page/pageSize pagination (new model)
	page := 1
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	pageSize := 20
	if ps := c.Query("pageSize"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil && size > 0 && size <= 500 {
			pageSize = size
		}
	}

	// Convert to offset/limit for repository
	filters.Limit = pageSize
	filters.Offset = (page - 1) * pageSize

	// Support legacy offset/limit parameters
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filters.Offset = o
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 500 {
			filters.Limit = l
		}
	}

	// Handle sorting with validation to prevent SQL injection
	if sortBy := c.Query("sortBy"); sortBy != "" {
		allowedSortFields := []string{
			"id", "branch_name", "bank_code", "bin",
		}
		validSortBy := false
		for _, field := range allowedSortFields {
			if sortBy == field {
				validSortBy = true
				break
			}
		}
		if validSortBy {
			filters.SortBy = sortBy
		}
	}
	// Support legacy sort_by
	if sortBy := c.Query("sort_by"); sortBy != "" {
		allowedSortFields := []string{
			"id", "branch_name", "bank_code", "bin",
		}
		validSortBy := false
		for _, field := range allowedSortFields {
			if sortBy == field {
				validSortBy = true
				break
			}
		}
		if validSortBy {
			filters.SortBy = sortBy
		}
	}

	if sortOrder := c.Query("sortOrder"); sortOrder != "" {
		// Validate sortOrder to prevent SQL injection (case insensitive)
		if sortOrder == "asc" || sortOrder == "desc" || sortOrder == "ASC" || sortOrder == "DESC" {
			filters.SortOrder = sortOrder
		}
	}
	// Support legacy sort_order
	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		// Validate sortOrder to prevent SQL injection (case insensitive)
		if sortOrder == "asc" || sortOrder == "desc" || sortOrder == "ASC" || sortOrder == "DESC" {
			filters.SortOrder = sortOrder
		}
	}

	// Handle search
	if search := c.Query("search"); search != "" {
		filters.Search = search
	}

	// Get banks and total count
	banks, total, err := h.bankService.ListBanks(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToRetrieveBanksVN)
		return
	}

	// Convert to response DTOs
	bankResponses := make([]dto.BankResponse, len(banks))
	for i, bank := range banks {
		bankResponses[i] = h.buildBankResponse(bank)
	}

	// Calculate pagination
	totalPages := (int(total) + pageSize - 1) / pageSize

	if len(bankResponses) == 0 {
		pagination := response.Pagination{
			Page:         page,
			PageSize:     pageSize,
			TotalPages:   totalPages,
			TotalRecords: int(total),
		}
		response.SuccessEmptyWithPagination(c, "No banks found", pagination)
		return
	}

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	message := "Banks retrieved successfully"
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of banks retrieved successfully"
	}

	response.SuccessWithPagination(c, bankResponses, message, pagination)
}
