package advance_payment

import (
	"api-server/internal/app/dto"
	"api-server/internal/app/services/flex_pay"
	"api-server/internal/app/services/notification"
	"api-server/internal/constants"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"
	"api-server/internal/transport/http/response"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// SendReconciliationEmail handles sending FlexPay reconciliation email
// POST /api/v1/advance-payments/reconciliation/send-email
func (h *AdvancePaymentHandler) SendReconciliationEmail(c *gin.Context) {
	var req dto.SendFlexPayReconciliationEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestParametersVN)
		return
	}

	if _, err := time.Parse("2006-01", req.ForMonth); err != nil {
		response.BadRequest(c, constants.MsgInvalidMonthFormatVN)
		return
	}

	ctx := c.Request.Context()
	logger := observability.GetLogger()

	logger.Info("Starting FlexPay reconciliation email sending",
		"forMonth", req.ForMonth,
		"recipients_count", len(req.Recipients),
		"user_id", c.GetString("user_id"))

	// Get completed requests for the month
	reportData, err := h.flexPayReconciliationService.GetCompletedRequestsByMonth(ctx, req.ForMonth)
	if err != nil {
		logger.Error("Failed to get completed requests", "error", err)
		response.InternalServerError(c, constants.MsgFailedToGenerateReconciliationVN)
		return
	}

	if len(reportData) == 0 {
		logger.Info("No completed requests found for the month", "forMonth", req.ForMonth)
		response.Success(c, nil, "Không có dữ liệu Nhận lương sớm 24/7 cho tháng được chọn")
		return
	}

	// Parse the month for due date calculation
	parsedMonth, _ := time.Parse("2006-01", req.ForMonth)
	atDate := time.Date(parsedMonth.Year(), parsedMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Generate Excel file
	excelBytes, summary, err := h.flexPayReconciliationExporter.GenerateExcel(reportData, atDate)
	if err != nil {
		logger.Error("Failed to generate Excel", "error", err)
		response.InternalServerError(c, constants.MsgFailedToGenerateReconciliationFileVN)
		return
	}

	// Cancel all pending requests for the month
	cancelledCount, _, err := h.flexPayReconciliationService.CancelAllPendingRequests(ctx, req.ForMonth)
	if err != nil {
		logger.Error("Failed to cancel pending requests", "error", err)
	}

	// Use default recipients if none provided
	recipients := req.Recipients
	if len(recipients) == 0 {
		recipients = []string{"frankng.sg@gmail.com"}
	}

	// Generate email body - due date is end of the NEXT month (e.g. April advance → due end of May)
	endOfNextMonth := time.Date(parsedMonth.Year(), parsedMonth.Month()+2, 0, 0, 0, 0, 0, time.UTC)
	dueDate := endOfNextMonth.Format("02/01/2006")
	totalCollect := utils.FormatNumber(summary.TotalWithFee) + " đ"

	htmlBody, textBody := flex_pay.BuildSaoKeEmailBodies(req.ForMonth, dueDate, totalCollect)

	// Send email with asset + metadata storage
	emailID, err := h.emailService.SendAdvancePaymentReconciliationEmail(ctx, &notification.ReconciliationEmailParams{
		ForMonth:   req.ForMonth,
		Recipients: recipients,
		CC:         req.Cc,
		BCC:        req.Bcc,
		ExcelBytes: excelBytes,
		HTMLBody:   htmlBody,
		TextBody:   textBody,
		Subject:    fmt.Sprintf("Sao kê thanh toán LG Display - %s", h.clock.Now().Format("02/01/2006")),
		Summary:    summary,
	})
	if err != nil {
		logger.Error("Failed to send reconciliation email", "error", err)
		response.InternalServerError(c, constants.MsgFailedToSendReconciliationEmailVN)
		return
	}

	logger.Info("FlexPay reconciliation email sent successfully",
		"forMonth", req.ForMonth,
		"email_id", emailID,
		"cancelledCount", cancelledCount)

	response.Success(c, dto.ReconciliationEmailResult{
		EmailID:        emailID,
		CancelledCount: cancelledCount,
	}, "Email sent successfully")
}
