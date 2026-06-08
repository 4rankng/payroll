package advance_payment

import (
	"strconv"

	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetDisbursementStatus returns the current status of an advance payment request.
// Used by the admin UI to poll progress after a manual retry — the request status
// transitions from APPROVED → COMPLETED/FAILED as the disbursement worker processes it.
//
// GET /api/v1/advance-payments/:id/disbursement-status
func (h *AdvancePaymentHandler) GetDisbursementStatus(c *gin.Context) {
	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseUint(requestIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "ID yêu cầu không hợp lệ")
		return
	}

	ctx := c.Request.Context()

	req, err := h.service.GetConfig().AdvancePaymentRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		response.NotFound(c, "Không tìm thấy yêu cầu")
		return
	}

	isTerminal := req.IsCompleted() || req.IsFailed() || req.IsCancelled()

	response.Success(c, gin.H{
		"requestId":     requestID,
		"requestStatus": string(req.Status),
		"isTerminal":    isTerminal,
		"paymentRef":    req.PaymentReference,
		"paidAt":        req.PaidAt,
	}, "")
}
