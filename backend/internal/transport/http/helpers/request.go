package helpers

import (
	"strconv"

	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// BindJSON attempts to bind the request body JSON to the provided struct.
// If binding fails, it sends a bad request response with the default error message.
// Returns true if successful, false otherwise (response already sent).
//
// Example usage:
//
//	var req dto.CreateEmployeeRequest
//	if !helpers.BindJSON(c, &req) {
//		return
//	}
func BindJSON(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return false
	}
	return true
}

// BindJSONWithError attempts to bind the request body JSON to the provided struct.
// If binding fails, it sends a bad request response with the custom error message.
// Returns true if successful, false otherwise (response already sent).
//
// Example usage:
//
//	var req dto.CreateProjectRequest
//	if !helpers.BindJSONWithError(c, &req, "Invalid project data") {
//		return
//	}
func BindJSONWithError(c *gin.Context, req interface{}, errMsg string) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.BadRequest(c, errMsg)
		return false
	}
	return true
}

// BindJSONWithValidation attempts to bind the request body JSON and runs a custom validator.
// If binding or validation fails, it sends a bad request response.
// Returns true if successful, false otherwise (response already sent).
//
// Example usage:
//
//	var req dto.CreateEmployeeRequest
//	if !helpers.BindJSONWithValidation(c, &req, func() error {
//		if req.Age < 18 {
//			return errors.New("employee must be 18 or older")
//		}
//		return nil
//	}) {
//		return
//	}
func BindJSONWithValidation(c *gin.Context, req interface{}, validator func() error) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return false
	}

	if validator != nil {
		if err := validator(); err != nil {
			response.BadRequest(c, err.Error())
			return false
		}
	}

	return true
}

// BindQuery attempts to bind query parameters to the provided struct.
// If binding fails, it sends a bad request response.
// Returns true if successful, false otherwise (response already sent).
//
// Example usage:
//
//	var filters dto.EmployeeFilters
//	if !helpers.BindQuery(c, &filters) {
//		return
//	}
func BindQuery(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		response.BadRequest(c, "Invalid query parameters")
		return false
	}
	return true
}

// BindURI attempts to bind URI parameters to the provided struct.
// If binding fails, it sends a bad request response.
// Returns true if successful, false otherwise (response already sent).
//
// Example usage:
//
//	var uri struct {
//		ID uint `uri:"id" binding:"required"`
//	}
//	if !helpers.BindURI(c, &uri) {
//		return
//	}
func BindURI(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindUri(req); err != nil {
		response.BadRequest(c, "Invalid URI parameters")
		return false
	}
	return true
}

// ParseIDParam parses a uint path parameter by name.
// Sends a BadRequest response and returns (0, false) on failure.
//
// Example usage:
//
//	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidIDFormatVN)
//	if !ok {
//		return
//	}
func ParseIDParam(c *gin.Context, paramName string, errMsg string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(paramName), 10, 32)
	if err != nil {
		response.BadRequest(c, errMsg)
		return 0, false
	}
	return uint(id), true
}
