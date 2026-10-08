package integration

import (
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/integration"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// SelfCheckinHandler handles the chatbot's self check-in flow: OTP → verify →
// enable/disable update.
type SelfCheckinHandler struct {
	service *integration.SelfCheckinService
}

// NewSelfCheckinHandler constructs the handler.
func NewSelfCheckinHandler(service *integration.SelfCheckinService) *SelfCheckinHandler {
	return &SelfCheckinHandler{service: service}
}

// RequestOTP
// @Summary Send a self check-in OTP
// @Description Resolve the phone, dispatch a Zalo ZNS OTP, and return the employee's eligible assignments so the conversation needs one round-trip.
// @Tags integration
// @Security ApiKeyAuth
// @Param body body dto.SelfCheckinOTPRequest true "Employee phone"
// @Success 200 {object} response.SuccessResponse
// @Router /integration/self-checkin/otp [post]
func (h *SelfCheckinHandler) RequestOTP(c *gin.Context) {
	var req dto.SelfCheckinOTPRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	result, err := h.service.SelfCheckinOTP(c.Request.Context(), req.Phone)
	if err != nil {
		h.writeError(c, err)
		return
	}

	out := dto.SelfCheckinOTPResponse{
		Found:             result.Found,
		OTPSent:           result.OTPSent,
		SessionID:         result.SessionID,
		ExpiresIn:         result.ExpiresIn,
		OTPLength:         otpLength,
		EmployeeName:      result.EmployeeName,
		DeliveryErrorCode: result.DeliveryErrorCode,
		Assignments:       mapAssignments(result.Assignments),
	}
	if result.FailureReason != "" {
		reason := result.FailureReason
		out.FailureReason = &reason
	}

	response.Success(c, out, selfCheckinOTPMessage(result))
}

// selfCheckinOTPMessage returns a message coherent with the reported outcome,
// so the human-readable text never contradicts the structured flags. The
// not-found / disabled / delivery-failure wording is shared with the reset
// flow; only the success line is flow-specific.
func selfCheckinOTPMessage(r *integration.SelfCheckinRequestResult) string {
	switch r.FailureReason {
	case "":
		return constants.MsgSelfCheckinOTPSentVN
	case integration.FailureAccountNotFound:
		return constants.MsgIntegrationAccountNotFoundVN
	case integration.FailureZaloDisabled:
		return constants.MsgIntegrationOTPDisabledVN
	default:
		return constants.MsgIntegrationOTPSendFailedVN
	}
}

// VerifyOTP
// @Summary Verify a self check-in OTP
// @Description Consume the OTP session and return a single-use action token.
// @Tags integration
// @Security ApiKeyAuth
// @Param body body dto.SelfCheckinVerifyRequest true "Session id + code"
// @Success 200 {object} response.SuccessResponse
// @Router /integration/self-checkin/verify [post]
func (h *SelfCheckinHandler) VerifyOTP(c *gin.Context) {
	var req dto.SelfCheckinVerifyRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	result, err := h.service.SelfCheckinVerify(c.Request.Context(), req.SessionID, req.Code)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, dto.SelfCheckinVerifyResponse{
		Verified:    true,
		ActionToken: result.ActionToken,
		ExpiresIn:   result.ExpiresIn,
	}, constants.MsgSelfCheckinOTPVerifiedVN)
}

// Update
// @Summary Apply a self check-in enable/disable
// @Description Consume the action token and apply the change through the shared row-locked service method, returning the verdict.
// @Tags integration
// @Security ApiKeyAuth
// @Param body body dto.SelfCheckinUpdateRequest true "Action token + project + direction"
// @Success 200 {object} response.SuccessResponse
// @Router /integration/self-checkin/update [post]
func (h *SelfCheckinHandler) Update(c *gin.Context) {
	var req dto.SelfCheckinUpdateRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	actorID := uint(0)
	if v, ok := c.Get(constants.CtxAPIKeyID); ok {
		if id, ok := v.(uint); ok {
			actorID = id
		}
	}

	// req.Enable is a required pointer binding, so dereferencing is safe here.
	result, err := h.service.SelfCheckinUpdate(c.Request.Context(), req.ActionToken, req.ProjectID, *req.Enable, actorID)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, dto.SelfCheckinUpdateResponse{
		Success:                 true,
		Kind:                    result.Kind,
		Immediate:               result.Immediate,
		EffectiveFrom:           effectiveFrom(result),
		CancelledPendingEnable:  result.CancelledPendingEnable,
		CancelledPendingDisable: result.CancelledPendingDisable,
	}, constants.MsgSelfCheckinUpdateSuccessVN)
}

// effectiveFrom returns nil for a zero EffectiveFrom (a cancelled queued
// enable) so the JSON field is omitted instead of reporting a bogus date.
func effectiveFrom(r *integration.SelfCheckinUpdateResult) *time.Time {
	if r.EffectiveFrom.IsZero() {
		return nil
	}
	t := r.EffectiveFrom
	return &t
}

// writeError maps service domain errors to HTTP responses.
// Status
// @Summary Report the employee's current self check-in state
// @Description Read-only: peek the verified action token (never consumed) and return each supported project's on/off flag, start date and queued change.
// @Tags integration
// @Security ApiKeyAuth
// @Param body body dto.SelfCheckinStatusRequest true "Verified action token"
// @Success 200 {object} response.SuccessResponse
// @Router /integration/self-checkin/status [post]
func (h *SelfCheckinHandler) Status(c *gin.Context) {
	var req dto.SelfCheckinStatusRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	result, err := h.service.SelfCheckinStatus(c.Request.Context(), req.ActionToken)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, dto.SelfCheckinStatusResponse{
		Found:        result.Found,
		EmployeeName: result.EmployeeName,
		Assignments:  mapAssignments(result.Assignments),
	}, "ok")
}

func (h *SelfCheckinHandler) writeError(c *gin.Context, err error) {
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

// mapAssignments converts the service's assignment payload to its DTO shape.
func mapAssignments(items []integration.SelfCheckinAssignment) []dto.SelfCheckinAssignmentDTO {
	if len(items) == 0 {
		return nil
	}
	out := make([]dto.SelfCheckinAssignmentDTO, 0, len(items))
	for _, item := range items {
		dtoItem := dto.SelfCheckinAssignmentDTO{
			ProjectID:        item.ProjectID,
			ProjectName:      item.ProjectName,
			CheckInEnabled:   item.CheckInEnabled,
			CheckInStartDate: item.CheckInStartDate,
			PaymentSchedule:  item.PaymentSchedule,
		}
		if item.PendingChange != nil {
			dtoItem.PendingChange = &dto.SelfCheckinPendingChangeDTO{
				Type:          item.PendingChange.Type,
				EffectiveFrom: item.PendingChange.EffectiveFrom,
			}
		}
		if item.Target != nil {
			dtoItem.CheckInTarget = &dto.SelfCheckinTargetDTO{
				RadiusMeters: item.Target.RadiusMeters,
				Gates:        item.Target.Gates,
			}
		}
		for _, w := range item.ShiftWindows {
			dtoItem.ShiftWindows = append(dtoItem.ShiftWindows, dto.SelfCheckinShiftWindowDTO{
				ShiftStart:          w.ShiftStart,
				ShiftEnd:            w.ShiftEnd,
				CheckInWindowStart:  w.CheckInWindowStart,
				CheckInWindowEnd:    w.CheckInWindowEnd,
				CheckOutWindowStart: w.CheckOutWindowStart,
				CheckOutWindowEnd:   w.CheckOutWindowEnd,
				ShiftName:           w.Name,
			})
		}
		out = append(out, dtoItem)
	}
	return out
}
