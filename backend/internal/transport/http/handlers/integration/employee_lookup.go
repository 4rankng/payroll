package integration

import (
	"api-server/internal/app/dto"
	"api-server/internal/app/services/integration"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// EmployeeLookupHandler handles the chatbot's employee detail lookup.
type EmployeeLookupHandler struct {
	service *integration.Service
}

// NewEmployeeLookupHandler constructs the handler.
func NewEmployeeLookupHandler(service *integration.Service) *EmployeeLookupHandler {
	return &EmployeeLookupHandler{service: service}
}

// LookupEmployee
// @Summary Look up an employee by mobile
// @Description Resolve a phone to the employee's name, CCCD, and mobile for the chatbot to verify.
// @Tags integration
// @Security ApiKeyAuth
// @Param body body dto.IntegrationLookupRequest true "Employee phone"
// @Success 200 {object} response.SuccessResponse
// @Router /integration/employee/lookup [post]
func (h *EmployeeLookupHandler) LookupEmployee(c *gin.Context) {
	var req dto.IntegrationLookupRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	result, err := h.service.LookupEmployee(c.Request.Context(), req.Phone)
	if err != nil {
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgZaloResetStoreDownVN)
		return
	}

	message := constants.MsgIntegrationEmployeeNotFoundVN
	if result.Found {
		message = constants.MsgIntegrationEmployeeFoundVN
	}
	response.Success(c, dto.IntegrationLookupResponse{
		Found:        result.Found,
		EmployeeName: result.EmployeeName,
		CCCD:         result.CCCD,
		Mobile:       result.Mobile,
	}, message)
}
