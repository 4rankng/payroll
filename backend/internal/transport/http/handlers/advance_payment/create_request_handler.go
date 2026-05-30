package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// CreateAdvancePaymentRequest creates a new advance payment request
func (h *AdvancePaymentHandler) CreateAdvancePaymentRequest(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUnauthorizedVN)
		return
	}

	var req dto.CreateAdvancePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	forMonth := req.ForMonth
	if forMonth == "" {
		forMonth = h.clock.Now().Format("2006-01")
	}

	result, err := h.service.CreateRequestByUserID(c.Request.Context(), uint64(userID.(uint)), req.Amount, forMonth)
	if err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.SuccessCreated(c, result, "Yêu cầu ứng lương đã được tạo thành công")
}
