package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/notification"
)

// EmailHandler exposes email-related endpoints for administrators.
type EmailHandler struct {
	emailService *notification.EmailService
	asynqClient  *asynqinfra.Client
}

// NewEmailHandler constructs a new EmailHandler.
func NewEmailHandler(emailService *notification.EmailService, asynqClient *asynqinfra.Client) *EmailHandler {
	return &EmailHandler{emailService: emailService, asynqClient: asynqClient}
}

// SendGenericEmail handles POST /email/send for manual emails with optional attachments.
func (h *EmailHandler) SendGenericEmail(c *gin.Context) {
	if !isAdmin(c) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32 MB max memory
		response.BadRequest(c, constants.MsgInvalidFormDataVN)
		return
	}

	// Extract form fields
	req := dto.SendEmailRequest{
		From:       c.PostForm("from"),
		Recipients: c.PostFormArray("recipients"),
		Cc:         c.PostFormArray("cc"),
		Bcc:        c.PostFormArray("bcc"),
		Subject:    c.PostForm("subject"),
		HTMLBody:   c.PostForm("htmlBody"),
		TextBody:   c.PostForm("textBody"),
		ReplyTo:    c.PostForm("replyTo"),
	}

	// Validate basic request
	if err := req.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Process attachments if present
	form := c.Request.MultipartForm
	if form != nil && form.File["attachments"] != nil {
		files := form.File["attachments"]

		// Validate attachment count
		if len(files) > domain.MaxEmailAttachments {
			response.BadRequest(c, fmt.Sprintf("Tối đa %d tệp đính kèm được phép", domain.MaxEmailAttachments))
			return
		}

		attachments := make([]dto.AttachmentData, 0, len(files))
		for _, fileHeader := range files {
			// Validate file size (10 MB)
			if fileHeader.Size > domain.MaxEmailAttachmentSize {
				response.BadRequest(c, fmt.Sprintf("Tệp '%s' vượt quá giới hạn 10MB", fileHeader.Filename))
				return
			}

			// Validate file type
			if err := validateEmailAttachmentType(fileHeader.Filename); err != nil {
				response.BadRequest(c, err.Error())
				return
			}

			// Open and read file
			file, err := fileHeader.Open()
			if err != nil {
				response.BadRequest(c, fmt.Sprintf("Không thể đọc tệp '%s'", fileHeader.Filename))
				return
			}

			// Read file content
			content, err := io.ReadAll(file)
			_ = file.Close()
			if err != nil {
				response.BadRequest(c, fmt.Sprintf("Không thể đọc nội dung tệp '%s'", fileHeader.Filename))
				return
			}

			// Add to attachments
			attachments = append(attachments, dto.AttachmentData{
				Filename:    fileHeader.Filename,
				ContentType: fileHeader.Header.Get("Content-Type"),
				Content:     content,
			})
		}

		req.Attachments = attachments
	}

	messageID, err := h.emailService.SendGenericEmail(c.Request.Context(), &req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, dto.SendEmailResponse{MessageID: messageID}, constants.MsgEmailSentSuccessfullyVN)
}

// validateEmailAttachmentType checks if the file extension is in the allowlist.
func validateEmailAttachmentType(filename string) error {
	allowedExtensions := map[string]bool{
		".pdf":  true,
		".xlsx": true,
		".xls":  true,
		".doc":  true,
		".docx": true,
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".txt":  true,
		".csv":  true,
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExtensions[ext] {
		return fmt.Errorf("loại tệp %s không được phép. Chỉ chấp nhận: pdf, xlsx, xls, doc, docx, png, jpg, jpeg, gif, txt, csv", ext)
	}

	return nil
}

// SendPayrollReportEmail handles POST /timesheets/payroll/report/send-email.
func (h *EmailHandler) SendPayrollReportEmail(c *gin.Context) {
	if !isAdmin(c) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}

	var req dto.SendPayrollReportEmailRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	if _, err := req.ParseReportAtDate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}

	if h.asynqClient == nil {
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	taskID, duplicate, err := h.asynqClient.EnqueuePayrollReportEmail(req, userID, c.GetHeader("Idempotency-Key"))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, response.SuccessResponse{
		Status: "success",
		Data: gin.H{
			"taskId":          taskID,
			"idempotencyKey":  taskID,
			"alreadyQueued":   duplicate,
			"processingAsync": true,
		},
		Message: constants.MsgPayrollEmailQueuedSuccessfullyVN,
	})
}

// GetEmailHistory handles GET /email/history to provide paginated delivery logs.
func (h *EmailHandler) GetEmailHistory(c *gin.Context) {
	if !isAdmin(c) {
		response.Forbidden(c, constants.MsgForbiddenVN)
		return
	}

	page := 1
	if query := c.Query("page"); query != "" {
		value, err := strconv.Atoi(query)
		if err != nil || value <= 0 {
			response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
			return
		}
		page = value
	}

	pageSize := 20
	if query := c.Query("pageSize"); query != "" {
		value, err := strconv.Atoi(query)
		if err != nil || value <= 0 || value > 100 {
			response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
			return
		}
		pageSize = value
	}

	limit := pageSize
	offset := (page - 1) * pageSize

	history, total, err := h.emailService.GetEmailHistory(c.Request.Context(), limit, offset)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	var totalPages int
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	if len(history) == 0 {
		response.SuccessEmptyWithPagination(c, constants.MsgEmailHistoryRetrievedVN, pagination)
		return
	}

	response.SuccessWithPagination(c, history, constants.MsgEmailHistoryRetrievedVN, pagination)
}

func isAdmin(c *gin.Context) bool {
	role := c.GetString(constants.CtxUserRole)
	return role == string(domain.RoleAdmin)
}
