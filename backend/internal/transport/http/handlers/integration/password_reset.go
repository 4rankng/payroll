// Package integration exposes the API-key-authenticated endpoints the external
// chatbot calls: a three-step password reset and an employee detail lookup.
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

// otpLength is the fixed OTP code length reported to the caller.
const otpLength = 6

// PasswordResetHandler handles the chatbot's three-step reset flow.
type PasswordResetHandler struct {
	service *integration.Service
}

// NewPasswordResetHandler constructs the handler.
func NewPasswordResetHandler(service *integration.Service) *PasswordResetHandler {
	return &PasswordResetHandler{service: service}
}

// RequestOTP
// @Summary Send a password-reset OTP
// @Description Resolve the phone and dispatch a Zalo ZNS OTP; report outcomes explicitly.
// @Tags integration
// @Security ApiKeyAuth
// @Param body body dto.IntegrationOTPRequest true "Employee phone"
// @Success 200 {object} response.SuccessResponse
// @Router /integration/password-reset/otp [post]
func (h *PasswordResetHandler) RequestOTP(c *gin.Context) {
	var req dto.IntegrationOTPRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	result, err := h.service.RequestOTP(c.Request.Context(), req.Phone)
	if err != nil {
		h.writeError(c, err)
		return
	}

	out := dto.IntegrationOTPResponse{
		Found:             result.Found,
		OTPSent:           result.OTPSent,
		SessionID:         result.SessionID,
		ExpiresIn:         result.ExpiresIn,
		OTPLength:         otpLength,
		EmployeeName:      result.EmployeeName,
		DeliveryErrorCode: result.DeliveryErrorCode,
	}
	if result.FailureReason != "" {
		reason := result.FailureReason
		out.FailureReason = &reason
	}

	response.Success(c, out, otpMessage(result))
}

// otpMessage returns a message coherent with the reported outcome, so the
// human-readable text never contradicts the structured flags.
func otpMessage(r *integration.RequestResult) string {
	switch r.FailureReason {
	case "":
		return constants.MsgIntegrationOTPSentVN
	case integration.FailureAccountNotFound:
		return constants.MsgIntegrationAccountNotFoundVN
	case integration.FailureZaloDisabled:
		return constants.MsgIntegrationOTPDisabledVN
	default:
		return constants.MsgIntegrationOTPSendFailedVN
	}
}

// VerifyOTP
// @Summary Verify a password-reset OTP
// @Description Consume the OTP session and return a single-use reset token.
// @Tags integration
// @Security ApiKeyAuth
// @Param body body dto.IntegrationVerifyRequest true "Session id + code"
// @Success 200 {object} response.SuccessResponse
// @Router /integration/password-reset/verify [post]
func (h *PasswordResetHandler) VerifyOTP(c *gin.Context) {
	var req dto.IntegrationVerifyRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	result, err := h.service.VerifyOTP(c.Request.Context(), req.SessionID, req.Code)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, dto.IntegrationVerifyResponse{
		Verified:   true,
		ResetToken: result.ResetToken,
		ExpiresIn:  result.ExpiresIn,
	}, constants.MsgIntegrationOTPVerifiedVN)
}

// ResetPassword
// @Summary Reset a password with a verified token
// @Description Consume the reset token and set (or generate) the new password.
// @Tags integration
// @Security ApiKeyAuth
// @Param body body dto.IntegrationResetRequest true "Reset token + optional new password"
// @Success 200 {object} response.SuccessResponse
// @Router /integration/password-reset/reset [post]
func (h *PasswordResetHandler) ResetPassword(c *gin.Context) {
	var req dto.IntegrationResetRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	actorID := uint(0)
	if v, ok := c.Get(constants.CtxAPIKeyID); ok {
		if id, ok := v.(uint); ok {
			actorID = id
		}
	}
	actorName := "api key"
	if v, ok := c.Get(constants.CtxAPIKeyName); ok {
		if name, ok := v.(string); ok && name != "" {
			actorName = name
		}
	}

	result, err := h.service.ResetPassword(c.Request.Context(), req.ResetToken, req.NewPassword, actorID, actorName)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, dto.IntegrationResetResponse{
		Username:     result.Username,
		NewPassword:  result.NewPassword,
		EmployeeName: result.EmployeeName,
	}, constants.MsgIntegrationResetSuccessVN)
}

// writeError maps service domain errors to HTTP responses.
func (h *PasswordResetHandler) writeError(c *gin.Context, err error) {
	switch {
	case domain.IsUnauthorizedError(err):
		response.Unauthorized(c, err.Error())
	case domain.IsValidationError(err):
		response.BadRequest(c, err.Error())
	case domain.IsNotFoundError(err):
		response.NotFound(c, err.Error())
	default:
		response.InternalServerError(c, constants.MsgZaloResetStoreDownVN)
	}
}
