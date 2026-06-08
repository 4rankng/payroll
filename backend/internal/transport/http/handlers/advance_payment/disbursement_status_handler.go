package advance_payment

import (
	"strconv"

	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/domain/wallet"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetDisbursementStatus returns the current status of an advance payment request.
// Used by the admin UI to poll progress after a manual retry — reports the
// advance_request status AND the latest wallet_payment status so the frontend
// can show the real transfer outcome (e.g. sync failure from provider) even when
// the advance_request itself is still APPROVED.
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

	// Base response from the advance request
	isTerminal := req.IsCompleted() || req.IsFailed() || req.IsCancelled()
	out := gin.H{
		"requestId":     requestID,
		"requestStatus": string(req.Status),
		"isTerminal":    isTerminal,
		"paymentRef":    req.PaymentReference,
		"paidAt":        req.PaidAt,
	}

	// Supplement with the latest wallet_payment when the request is still
	// in-progress (APPROVED). This surfaces sync failures from the provider
	// that the frontend otherwise wouldn't see.
	if !isTerminal && h.walletPaymentRepo != nil {
		payments, _, _ := h.walletPaymentRepo.List(ctx, wallet.WalletPaymentFilter{
			EntityID: &requestID,
			Page:     1,
			PageSize: 1,
		})
		if len(payments) > 0 {
			latest := payments[0]
			wpTerminal := latest.Status == string(domaintx.StateCompleted) ||
				latest.Status == string(domaintx.StateFailed) ||
				latest.Status == string(domaintx.StateReversed)
			out["paymentStatus"] = latest.Status
			out["isTerminal"] = wpTerminal
			out["paymentErrorCode"] = latest.ErrorCode
			out["paymentErrorMessage"] = latest.ErrorMessage
			if latest.InvoiceNo != nil {
				out["paymentRef"] = *latest.InvoiceNo
			}
		}
	}

	response.Success(c, out, "")
}
