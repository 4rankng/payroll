package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetAvailableMonths returns list of months with flex pay data
func (h *AdvancePaymentHandler) GetAvailableMonths(c *gin.Context) {
	months, err := h.service.GetAvailableMonths(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	data := make([]dto.AvailableMonth, len(months))
	for i, m := range months {
		data[i] = dto.AvailableMonth{
			ForMonth:      m.ForMonth,
			EmployeeCount: m.EmployeeCount,
		}
	}

	response.Success(c, data, "Lấy danh sách tháng thành công")
}
