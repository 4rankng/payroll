package notification

import (
	"api-server/internal/pkg/clock"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/config"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/services"
	pkgConstants "api-server/internal/pkg/constants"
	auditctx "api-server/internal/pkg/context"
)

// EmailService coordinates email delivery via a configurable provider.
type EmailService struct {
	cfg              config.NotificationConfig
	delivery         domain.EmailDeliveryPort
	payrollReportSvc services.PayrollReportPort
	notificationRepo domain.NotificationRepository
	userRepo         domain.UserRepository
	eventPublisher   domain.EmailEventPublisher
	assetSvc         AssetStoragePort
	logger           *slog.Logger
}

// AssetStoragePort is a minimal interface for saving files as assets.
type AssetStoragePort interface {
	UploadAssetFromBytes(ctx context.Context, data []byte, filename string, uploadType string, uploadedBy uint) (*domain.Asset, error)
}

// NewEmailService constructs a new EmailService instance.
func NewEmailService(
	cfg config.NotificationConfig,
	delivery domain.EmailDeliveryPort,
	payrollReportSvc services.PayrollReportPort,
	notificationRepo domain.NotificationRepository,
	userRepo domain.UserRepository,
	eventPublisher domain.EmailEventPublisher,
	assetSvc AssetStoragePort,
	logger *slog.Logger,
) *EmailService {
	return &EmailService{
		cfg:              cfg,
		delivery:         delivery,
		payrollReportSvc: payrollReportSvc,
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		eventPublisher:   eventPublisher,
		assetSvc:         assetSvc,
		logger:           logger,
	}
}

// SendGenericEmail sends an arbitrary email supplied by administrators.
func (s *EmailService) SendGenericEmail(ctx context.Context, payload *dto.SendEmailRequest) (string, error) {
	if err := payload.Validate(); err != nil {
		return "", domain.NewValidationError(err.Error())
	}

	msg, err := s.buildGenericMessage(payload)
	if err != nil {
		return "", err
	}

	return s.dispatch(ctx, msg)
}

// SendPayrollReportEmail generates the payroll report for the specified date range and emails it.
func (s *EmailService) SendPayrollReportEmail(ctx context.Context, payload *dto.SendPayrollReportEmailRequest) (string, error) {
	if payload == nil {
		return "", domain.NewValidationError(constants.MsgRequestPayloadRequiredVN)
	}

	reportAtDate, err := payload.ParseReportAtDate()
	if err != nil {
		return "", domain.NewValidationError(err.Error())
	}

	reportData, err := s.payrollReportSvc.GetProjectsForPayrollReport(ctx, reportAtDate)
	if err != nil {
		return "", domain.NewInternalError(constants.MsgFailedToFetchPayrollReportDataVN, err)
	}

	if len(reportData) == 0 {
		return "", domain.NewValidationError(constants.MsgNoPayrollDataForPeriodVN)
	}

	reportBytes, summary, err := s.payrollReportSvc.GenerateExcel(reportData, reportAtDate)
	if err != nil {
		return "", domain.NewInternalError(constants.MsgFailedToGeneratePayrollEmailReportVN, err)
	}

	msg, err := s.buildPayrollReportMessage(payload, reportAtDate, summary, reportBytes)
	if err != nil {
		return "", err
	}

	// Collect timesheet IDs from report data for metadata storage
	timesheetIDSet := make(map[uint]struct{})
	for _, pd := range reportData {
		for _, id := range pd.TimesheetIDs {
			timesheetIDSet[id] = struct{}{}
		}
	}
	timesheetIDs := make([]uint, 0, len(timesheetIDSet))
	for id := range timesheetIDSet {
		timesheetIDs = append(timesheetIDs, id)
	}

	var payrollMeta *domain.PayrollEmailMetadata
	if summary != nil {
		feePercentage := summary.FeePercentage
		if feePercentage <= 0 {
			feePercentage = 0.02
		}
		payrollMeta = &domain.PayrollEmailMetadata{
			TotalAmount:   summary.TotalAmount,
			FeePercentage: feePercentage,
			FeeAmount:     summary.FeeAmount,
			TotalWithFee:  summary.TotalWithFee,
			ReportAtDate:  reportAtDate.Format("2006-01-02"),
			TimesheetIDs:  timesheetIDs,
			CC:            chooseAddresses(payload.Cc, s.cfg.DefaultCC),
			BCC:           chooseAddresses(payload.Bcc, s.cfg.DefaultBCC),
		}

		// Save the Excel file as an asset so it can be downloaded later
		if s.assetSvc != nil && len(reportBytes) > 0 {
			var senderID uint
			if uid := auditctx.GetUserID(ctx); uid != nil && *uid > 0 {
				senderID = *uid
			}
			filename := fmt.Sprintf("sao_ke_tt_%s.xlsx", reportAtDate.Format("2006-01-02"))
			if savedAsset, err := s.assetSvc.UploadAssetFromBytes(ctx, reportBytes, filename, "sao_ke", senderID); err != nil {
				s.logger.Warn("failed to save sao ke file as asset", "error", err)
			} else {
				payrollMeta.SaoKeAssetID = &savedAsset.ID
			}
		}
	}

	return s.dispatchWithMeta(ctx, msg, payrollMeta)
}

// ReconciliationEmailParams contains all data needed to send an advance payment reconciliation email.
type ReconciliationEmailParams struct {
	ForMonth   string
	Recipients []string
	CC         []string
	BCC        []string
	ExcelBytes []byte
	HTMLBody   string
	TextBody   string
	Subject    string
	Summary    *services.PayrollReportSummary
}

// SendAdvancePaymentReconciliationEmail sends a reconciliation email with asset + metadata storage.
func (s *EmailService) SendAdvancePaymentReconciliationEmail(ctx context.Context, params *ReconciliationEmailParams) (string, error) {
	recipients := chooseAddresses(params.Recipients, s.cfg.DefaultRecipients)
	if len(recipients) == 0 {
		return "", domain.NewValidationError(constants.MsgRecipientsCannotBeEmptyVN)
	}

	toAddrs, err := parseAddresses(recipients)
	if err != nil {
		return "", err
	}

	ccAddrs, _ := parseAddresses(chooseAddresses(params.CC, s.cfg.DefaultCC))
	bccAddrs, _ := parseAddresses(chooseAddresses(params.BCC, s.cfg.DefaultBCC))

	fromAddr := domain.EmailAddress{Name: s.cfg.FromName, Address: strings.ToLower(strings.TrimSpace(s.cfg.FromEmail))}

	msg := &domain.EmailMessage{
		Kind:     domain.EmailKindAdvancePaymentReport,
		From:     fromAddr,
		To:       toAddrs,
		CC:       ccAddrs,
		BCC:      bccAddrs,
		Subject:  params.Subject,
		HTMLBody: params.HTMLBody,
		TextBody: params.TextBody,
		Attachments: []domain.EmailAttachment{
			{
				Filename:    fmt.Sprintf("sao_ke_tt_%s.xlsx", params.ForMonth),
				ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				Content:     params.ExcelBytes,
			},
		},
	}

	if err := msg.Validate(); err != nil {
		return "", err
	}

	var payrollMeta *domain.PayrollEmailMetadata
	if params.Summary != nil {
		feePercentage := params.Summary.FeePercentage
		if feePercentage <= 0 {
			feePercentage = 0.02
		}
		payrollMeta = &domain.PayrollEmailMetadata{
			TotalAmount:   params.Summary.TotalAmount,
			FeePercentage: feePercentage,
			FeeAmount:     params.Summary.FeeAmount,
			TotalWithFee:  params.Summary.TotalWithFee,
			ReportAtDate:  params.ForMonth,
			CC:            chooseAddresses(params.CC, s.cfg.DefaultCC),
			BCC:           chooseAddresses(params.BCC, s.cfg.DefaultBCC),
		}

		if s.assetSvc != nil && len(params.ExcelBytes) > 0 {
			var senderID uint
			if uid := auditctx.GetUserID(ctx); uid != nil && *uid > 0 {
				senderID = *uid
			}
			filename := fmt.Sprintf("sao_ke_ung_luong_%s.xlsx", params.ForMonth)
			if savedAsset, err := s.assetSvc.UploadAssetFromBytes(ctx, params.ExcelBytes, filename, "sao_ke", senderID); err != nil {
				s.logger.Warn("failed to save reconciliation file as asset", "error", err)
			} else {
				payrollMeta.SaoKeAssetID = &savedAsset.ID
			}
		}
	}

	return s.dispatchWithMeta(ctx, msg, payrollMeta)
}

// AdvancePaymentReminderData holds data for the advance payment reminder email template.
type AdvancePaymentReminderData struct {
	Date               string
	TotalRequests      string
	TotalEmployees     string
	TotalAmount        string
	Requests           []AdvancePaymentReminderRequest
	TotalRequestAmount string
	TotalFee           string
	TotalNetAmount     string
}

// AdvancePaymentReminderRequest represents a single row in the reminder email table.
type AdvancePaymentReminderRequest struct {
	STT           int
	EmployeeCCCD  string
	EmployeeName  string
	ProjectCode   string
	RequestAmount string
	Fee           string
	NetAmount     string
}

// SendAdvancePaymentReminder sends an email notifying recipients about pending advance payment requests.
func (s *EmailService) SendAdvancePaymentReminder(ctx context.Context, data *AdvancePaymentReminderData, recipients []string) (string, error) {
	htmlBody, err := renderAdvancePaymentReminderTemplate(data)
	if err != nil {
		return "", domain.NewInternalError("Không thể render template email nhắc nhở ứng lương", err)
	}

	msg, err := s.buildGenericMessage(&dto.SendEmailRequest{
		Recipients: recipients,
		Subject:    fmt.Sprintf("Nhắc nhở: Có %s yêu cầu ứng lương đang chờ xử lý - %s", data.TotalRequests, data.Date),
		HTMLBody:   htmlBody,
		TextBody:   htmlToPlainText(htmlBody),
	})
	if err != nil {
		return "", err
	}

	return s.dispatch(ctx, msg)
}

func renderAdvancePaymentReminderTemplate(data *AdvancePaymentReminderData) (string, error) {
	content, err := os.ReadFile(pkgConstants.AdvancePaymentReminderTemplatePath)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("advance_payment_reminder").Parse(string(content))
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", err
	}

	return builder.String(), nil
}

// GetEmailHistory returns paginated email delivery records sourced from notification logs.
func (s *EmailService) GetEmailHistory(ctx context.Context, limit, offset int) ([]dto.EmailHistoryEntry, int64, error) {
	if s.notificationRepo == nil {
		return nil, 0, domain.NewInternalError(constants.MsgEmailHistoryRepoNotConfiguredVN, nil)
	}
	if limit <= 0 {
		return nil, 0, domain.NewValidationError(constants.MsgLimitMustBeGreaterThanZeroVN)
	}
	if offset < 0 {
		return nil, 0, domain.NewValidationError(constants.MsgOffsetCannotBeNegativeVN)
	}

	notifications, err := s.notificationRepo.ListByChannel(ctx, domain.NotificationChannelEmail, limit, offset)
	if err != nil {
		return nil, 0, domain.NewInternalError(constants.MsgFailedToRetrieveEmailHistoryVN, err)
	}

	total, err := s.notificationRepo.CountByChannel(ctx, domain.NotificationChannelEmail)
	if err != nil {
		return nil, 0, domain.NewInternalError(constants.MsgFailedToCountEmailHistoryVN, err)
	}

	senders := s.buildSenderLookup(ctx, notifications)

	history := make([]dto.EmailHistoryEntry, 0, len(notifications))
	for _, notification := range notifications {
		recipients := s.parseNotificationRecipients(notification.EmailRecipients)

		var messageID *string
		if notification.ResendMessageID != nil {
			trimmed := strings.TrimSpace(*notification.ResendMessageID)
			if trimmed != "" {
				messageIDCopy := trimmed
				messageID = &messageIDCopy
			}
		}

		senderName := ""
		senderUsername := ""
		if sender, ok := senders[notification.SenderID]; ok && sender != nil {
			senderName = sender.Fullname
			senderUsername = sender.Username
		} else if notification.SenderID == constants.SystemUserID {
			senderName = s.cfg.FromName
			senderUsername = strings.TrimSpace(s.cfg.FromEmail)
		}

		entry := dto.EmailHistoryEntry{
			ID:             notification.ID,
			Subject:        notification.Title,
			Body:           notification.Message,
			MessageID:      messageID,
			Recipients:     recipients,
			SenderID:       notification.SenderID,
			SenderName:     senderName,
			SenderUsername: senderUsername,
			Type:           string(notification.Type),
			Channel:        string(notification.Channel),
			SentAt:         notification.CreatedAt,
		}

		// Parse and attach payroll financial metadata if present
		if notification.Metadata != nil && *notification.Metadata != "" {
			var meta domain.PayrollEmailMetadata
			if err := json.Unmarshal([]byte(*notification.Metadata), &meta); err == nil {
				entry.SettledAt = meta.SettledAt
				entry.PayrollMeta = &dto.PayrollEmailMetaDTO{
					TotalAmount:   meta.TotalAmount,
					FeePercentage: meta.FeePercentage,
					FeeAmount:     meta.FeeAmount,
					TotalWithFee:  meta.TotalWithFee,
					ReportAtDate:  meta.ReportAtDate,
					TimesheetIDs:  meta.TimesheetIDs,
					SaoKeAssetID:  meta.SaoKeAssetID,
					CC:            meta.CC,
					BCC:           meta.BCC,
				}
			}
		}

		history = append(history, entry)
	}

	return history, total, nil
}

func (s *EmailService) buildSenderLookup(ctx context.Context, notifications []*domain.Notification) map[uint]*domain.User {
	if len(notifications) == 0 {
		return map[uint]*domain.User{}
	}

	result := make(map[uint]*domain.User, len(notifications))
	for _, notification := range notifications {
		if notification == nil {
			continue
		}

		if notification.SenderID == 0 {
			continue
		}

		// Prefer the preloaded sender when available.
		if notification.Sender != nil {
			result[notification.SenderID] = notification.Sender
			continue
		}

		if s.userRepo == nil {
			continue
		}

		if _, exists := result[notification.SenderID]; exists {
			continue
		}

		user, err := s.userRepo.GetByID(ctx, notification.SenderID)
		if err != nil {
			if !domain.IsNotFoundError(err) && s.logger != nil {
				s.logger.Warn("failed to resolve notification sender", "error", err, "senderID", notification.SenderID)
			}
			continue
		}

		result[notification.SenderID] = user
	}

	return result
}

func (s *EmailService) buildGenericMessage(payload *dto.SendEmailRequest) (*domain.EmailMessage, error) {
	fromAddr, err := s.resolveFromAddress(payload.From)
	if err != nil {
		return nil, err
	}

	recipients, err := parseAddresses(payload.Recipients)
	if err != nil {
		return nil, err
	}

	cc, err := parseAddresses(payload.Cc)
	if err != nil {
		return nil, err
	}

	bcc, err := parseAddresses(payload.Bcc)
	if err != nil {
		return nil, err
	}

	var replyTo *domain.EmailAddress
	if payload.ReplyTo != "" {
		reply, err := domain.ParseEmailAddress(payload.ReplyTo)
		if err != nil {
			return nil, err
		}
		replyTo = &reply
	}

	textBody := payload.TextBody
	if textBody == "" {
		textBody = htmlToPlainText(payload.HTMLBody)
	}

	// Map DTO attachments to domain attachments
	attachments := make([]domain.EmailAttachment, 0, len(payload.Attachments))
	for _, att := range payload.Attachments {
		attachments = append(attachments, domain.EmailAttachment{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Content:     att.Content,
		})
	}

	msg := &domain.EmailMessage{
		Kind:        domain.EmailKindGeneric,
		From:        fromAddr,
		ReplyTo:     replyTo,
		To:          recipients,
		CC:          cc,
		BCC:         bcc,
		Subject:     payload.Subject,
		HTMLBody:    payload.HTMLBody,
		TextBody:    textBody,
		Attachments: attachments,
	}

	if err := msg.Validate(); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *EmailService) buildPayrollReportMessage(payload *dto.SendPayrollReportEmailRequest, reportAtDate time.Time, summary *services.PayrollReportSummary, reportBytes []byte) (*domain.EmailMessage, error) {
	recipients := chooseAddresses(payload.Recipients, s.cfg.DefaultRecipients)
	if len(recipients) == 0 {
		return nil, domain.NewValidationError(constants.MsgRecipientsCannotBeEmptyVN)
	}

	toAddrs, err := parseAddresses(recipients)
	if err != nil {
		return nil, err
	}

	ccAddrs, err := parseAddresses(chooseAddresses(payload.Cc, s.cfg.DefaultCC))
	if err != nil {
		return nil, err
	}

	bccAddrs, err := parseAddresses(chooseAddresses(payload.Bcc, s.cfg.DefaultBCC))
	if err != nil {
		return nil, err
	}

	fromAddr := domain.EmailAddress{Name: s.cfg.FromName, Address: strings.ToLower(strings.TrimSpace(s.cfg.FromEmail))}

	if summary == nil {
		summary = &services.PayrollReportSummary{}
	}

	emailDay := clock.Now()
	subject := fmt.Sprintf("Dịch vụ thanh toán TingTing – %s", emailDay.Format("02/01/2006"))
	dueDate := emailDay.AddDate(0, 0, 14).Format("02/01/2006")
	feePercentValue := summary.FeePercentage
	if feePercentValue <= 0 {
		feePercentValue = 0.02
	}

	// Fall back to derived totals when summary is incomplete
	totalAmount := summary.TotalAmount
	feeAmountValue := summary.FeeAmount
	totalWithFee := summary.TotalWithFee
	if feeAmountValue == 0 {
		feeAmountValue = int64(float64(totalAmount) * feePercentValue)
	}
	if totalWithFee == 0 {
		totalWithFee = totalAmount + feeAmountValue
	}

	totalPaid := formatCurrencyVN(totalAmount) + " đ"
	feeAmount := formatCurrencyVN(feeAmountValue) + " đ"
	totalCollect := formatCurrencyVN(totalWithFee) + " đ"

	htmlBody, err := renderPayrollTemplate(dueDate, totalPaid, feeAmount, totalCollect)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgFailedToRenderPayrollEmailTemplateVN, err)
	}
	textBody := htmlToPlainText(htmlBody)

	attachment := domain.EmailAttachment{
		Filename:    fmt.Sprintf("sao_ke_tt_%s.xlsx", reportAtDate.Format("2006-01-02")),
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     reportBytes,
	}

	msg := &domain.EmailMessage{
		Kind:        domain.EmailKindPayrollReport,
		From:        fromAddr,
		To:          toAddrs,
		CC:          ccAddrs,
		BCC:         bccAddrs,
		Subject:     subject,
		HTMLBody:    htmlBody,
		TextBody:    textBody,
		Attachments: []domain.EmailAttachment{attachment},
		Metadata: map[string]string{
			"report_at_date": reportAtDate.Format("2006-01-02"),
			"statement_date": emailDay.Format("2006-01-02"),
			"due_date":       dueDate,
		},
	}

	if err := msg.Validate(); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *EmailService) dispatch(ctx context.Context, msg *domain.EmailMessage) (string, error) {
	return s.dispatchWithMeta(ctx, msg, nil)
}

func (s *EmailService) dispatchWithMeta(ctx context.Context, msg *domain.EmailMessage, payrollMeta *domain.PayrollEmailMetadata) (string, error) {
	timeout := s.cfg.SendTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	sendCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := s.delivery.Send(sendCtx, msg)
	if err != nil {
		s.logger.Error("email send failed", "error", err, "subject", msg.Subject)
		return "", domain.NewInternalError(constants.MsgEmailDeliveryFailedVN, err)
	}

	// Publish email sent event for audit logging (Observer pattern)
	if s.eventPublisher != nil {
		var initiatedBy *uint
		if userID := auditctx.GetUserID(ctx); userID != nil && *userID > 0 {
			initiatedBy = userID
		}

		event := domain.NewEmailSentEvent(result, msg, initiatedBy)
		event.PayrollMetadata = payrollMeta

		if err := s.eventPublisher.PublishEmailSent(ctx, event); err != nil {
			s.logger.Error("failed to publish email sent event", "error", err, "message_id", result.MessageID)
		}
	}

	return result.MessageID, nil
}

func (s *EmailService) resolveFromAddress(raw string) (domain.EmailAddress, error) {
	if strings.TrimSpace(raw) == "" {
		return domain.EmailAddress{Name: s.cfg.FromName, Address: strings.ToLower(s.cfg.FromEmail)}, nil
	}
	return domain.ParseEmailAddress(raw)
}

func parseAddresses(values []string) ([]domain.EmailAddress, error) {
	addresses := make([]domain.EmailAddress, 0, len(values))
	for _, raw := range values {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		addr, err := domain.ParseEmailAddress(raw)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}
	return addresses, nil
}

func (s *EmailService) parseNotificationRecipients(raw *string) []dto.EmailRecipient {
	if raw == nil {
		return nil
	}

	payload := strings.TrimSpace(*raw)
	if payload == "" {
		return nil
	}

	var stored []domain.EmailAddress
	if err := json.Unmarshal([]byte(payload), &stored); err != nil {
		if s.logger != nil {
			s.logger.Warn("failed to parse stored email recipients", "error", err)
		}
		return nil
	}

	recipients := make([]dto.EmailRecipient, 0, len(stored))
	for _, recipient := range stored {
		recipients = append(recipients, dto.EmailRecipient{
			Name:    recipient.Name,
			Address: recipient.Address,
		})
	}

	return recipients
}

func htmlToPlainText(html string) string {
	if html == "" {
		return ""
	}
	replacer := strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n", "</p>", "\n", "</li>", "\n")
	plain := replacer.Replace(html)
	stripTags := regexp.MustCompile(`<[^>]+>`)
	plain = stripTags.ReplaceAllString(plain, "")
	plain = strings.ReplaceAll(plain, "&nbsp;", " ")
	plain = strings.TrimSpace(plain)
	return plain
}

func chooseAddresses(requested, defaults []string) []string {
	if len(requested) > 0 {
		return requested
	}
	return defaults
}

func renderPayrollTemplate(dueDate, totalPaid, feeAmount, totalCollect string) (string, error) {
	content, err := os.ReadFile(pkgConstants.PayrollEmailTemplatePath)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("payroll_email").Parse(string(content))
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	data := map[string]string{
		"DueDate":      dueDate,
		"TotalPaid":    totalPaid,
		"FeeAmount":    feeAmount,
		"TotalCollect": totalCollect,
	}

	if err := tmpl.Execute(&builder, data); err != nil {
		return "", err
	}

	return builder.String(), nil
}

func formatCurrencyVN(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	negative := false
	if len(s) > 0 && s[0] == '-' {
		negative = true
		s = s[1:]
	}
	var result strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteString(".")
		}
		result.WriteRune(r)
	}
	value := result.String()
	if negative {
		value = "-" + value
	}
	return value
}
