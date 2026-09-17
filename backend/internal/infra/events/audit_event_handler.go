package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"api-server/internal/domain"
	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
)

// AuditEventHandler listens to domain events and enqueues audit:log:write asynq tasks.
// The worker (AuditLogWriteWorker) performs the actual DB insert with retries.
type AuditEventHandler struct {
	asynqClient asynqinfra.AuditEnqueuer
	logger      *slog.Logger
}

// NewAuditEventHandler creates a new audit event handler.
func NewAuditEventHandler(asynqClient asynqinfra.AuditEnqueuer) *AuditEventHandler {
	return &AuditEventHandler{
		asynqClient: asynqClient,
		logger:      observability.GetLogger(),
	}
}

// Handle processes a domain event and enqueues an audit log write task.
func (h *AuditEventHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	message := event.GetAuditMessage()
	if message == "" {
		return nil // skip events with no audit intent
	}

	// Drop rows with no actor (cron jobs, system writes, background workers
	// without a user context). Login attempts are the lone exception: failed
	// logins legitimately have user_id=0 and we still want them recorded.
	if event.UserID() == 0 && event.GetAction() != domain.AuditActionLogin {
		return nil
	}

	entityID := event.AggregateID()
	var entityIDPtr *uint
	if entityID != 0 {
		entityIDPtr = &entityID
	}

	meta := h.buildMetadata(event)

	var metaJSON string
	if len(meta) > 0 {
		b, err := json.Marshal(meta)
		if err != nil {
			h.logger.Error("audit: failed to marshal metadata", "error", err)
		} else {
			metaJSON = string(b)
		}
	}

	payload := asynqinfra.AuditLogWritePayload{
		UserID:       event.UserID(),
		Action:       string(event.GetAction()),
		EntityType:   string(event.GetEntityType()),
		EntityID:     entityIDPtr,
		Message:      message,
		IPAddress:    event.GetIPAddress(),
		UserAgent:    event.GetUserAgent(),
		MetadataJSON: metaJSON,
		CreatedAt:    clock.Now(),
	}

	return h.asynqClient.EnqueueAuditLogWrite(payload)
}

// buildMetadata returns event-specific metadata for audit logs.
func (h *AuditEventHandler) buildMetadata(event domain.DomainEvent) map[string]interface{} {
	switch e := event.(type) {
	// --- User events ---
	case domain.UserCreatedEvent:
		return map[string]interface{}{
			"username": e.Username,
			"fullname": e.Fullname,
			"email":    e.Email,
			"role":     string(e.Role),
		}
	case domain.UserUpdatedEvent:
		if len(e.ChangedFields) > 0 {
			return map[string]interface{}{"changed_fields": e.ChangedFields}
		}
	case domain.UserDeletedEvent:
		return map[string]interface{}{
			"username": e.Username,
			"fullname": e.Fullname,
		}
	case domain.PasswordChangedEvent:
		return map[string]interface{}{"method": e.Method}
	case domain.UserLoginEvent:
		if !e.Success {
			meta := map[string]interface{}{
				"success": false,
				"reason":  e.Reason,
			}
			if e.AttemptedIdentifier != "" {
				meta["attempted_identifier"] = e.AttemptedIdentifier
			}
			if e.Location != nil {
				meta["location"] = e.Location
			}
			return meta
		}
		// Successful login — store identifier used and location if available
		meta := map[string]interface{}{}
		if e.AttemptedIdentifier != "" && e.AttemptedIdentifier != e.Username {
			meta["login_identifier"] = e.AttemptedIdentifier
		}
		if e.Location != nil {
			meta["location"] = e.Location
		}
		if len(meta) > 0 {
			return meta
		}
	case domain.UserLogoutEvent:
		return map[string]interface{}{"username": e.Username}

	// --- Employee events ---
	case domain.EmployeeCreatedEvent:
		return map[string]interface{}{
			"fullname": e.Fullname,
			"cccd":     e.CCCD,
		}
	case domain.EmployeeUpdatedEvent:
		if len(e.ChangedFields) > 0 {
			return map[string]interface{}{"changed_fields": e.ChangedFields}
		}
	case domain.EmployeeDeletedEvent:
		return map[string]interface{}{
			"fullname": e.Fullname,
			"cccd":     e.CCCD,
		}
	case domain.EmployeeProjectAssignmentsRemovedEvent:
		return map[string]interface{}{
			"fullname": e.Fullname,
			"cccd":     e.CCCD,
		}
	case domain.EmployeeProfileUpdatedEvent:
		if len(e.ChangedFields) > 0 {
			return map[string]interface{}{"changed_fields": e.ChangedFields}
		}
	case domain.EmployeeNameUpdatedEvent:
		return map[string]interface{}{
			"old_fullname": e.OldFullName,
			"new_fullname": e.NewFullName,
			"cccd":         e.CCCD,
		}

	// --- Project events ---
	case domain.ProjectCreatedEvent:
		return map[string]interface{}{
			"name":        e.Name,
			"code":        e.Code,
			"client_name": e.ClientName,
		}
	case domain.ProjectUpdatedEvent:
		if len(e.ChangedFields) > 0 {
			return map[string]interface{}{"changed_fields": e.ChangedFields}
		}
	case domain.ProjectDeletedEvent:
		return map[string]interface{}{
			"name": e.Name,
			"code": e.Code,
		}

	// --- ProjectEmployee events ---
	case domain.ProjectEmployeeCreatedEvent:
		return map[string]interface{}{
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"role":          e.Role,
		}
	case domain.ProjectEmployeeUpdatedEvent:
		return map[string]interface{}{
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"status":        e.Status,
		}
	case domain.ProjectEmployeeDeletedEvent:
		return map[string]interface{}{
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
			"employee_name": e.EmployeeName,
			"reason":        e.Reason,
		}

	// --- Timesheet events ---
	case domain.TimesheetCreatedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
			"date":          e.Date.Format("2006-01-02"),
		}
	case domain.TimesheetUpdatedEvent:
		if len(e.ChangedFields) > 0 {
			return map[string]interface{}{"changed_fields": e.ChangedFields}
		}
	case domain.TimesheetDeletedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
		}
	case domain.TimesheetApprovedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
			"approver_id":   e.ApproverID,
		}
	case domain.TimesheetRejectedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
			"approver_id":   e.ApproverID,
		}
	case domain.TimesheetBulkApprovedEvent:
		return map[string]interface{}{
			"count":       e.Count,
			"project_id":  e.ProjectID,
			"approver_id": e.ApproverID,
		}
	case domain.TimesheetBulkRejectedEvent:
		return map[string]interface{}{
			"count":       e.Count,
			"project_id":  e.ProjectID,
			"approver_id": e.ApproverID,
		}
	case domain.TimesheetBulkResetEvent:
		return map[string]interface{}{
			"count":      e.Count,
			"project_id": e.ProjectID,
			"reset_by":   e.ResetBy,
		}
	case domain.TimesheetBulkCreatedEvent:
		return map[string]interface{}{
			"count":      e.Count,
			"project_id": e.ProjectID,
			"created_by": e.CreatedBy,
		}

	// --- Bank events ---
	case domain.BankCreatedEvent:
		return map[string]interface{}{"name": e.Name}
	case domain.BankUpdatedEvent:
		if len(e.ChangedFields) > 0 {
			return map[string]interface{}{"changed_fields": e.ChangedFields}
		}
	case domain.BankDeletedEvent:
		return map[string]interface{}{"name": e.Name}

	// --- Payrate events ---
	case domain.PayrateCreatedEvent:
		return map[string]interface{}{
			"project_id":   e.ProjectID,
			"project_name": e.ProjectName,
		}
	case domain.PayrateUpdatedEvent:
		return map[string]interface{}{
			"project_id":   e.ProjectID,
			"project_name": e.ProjectName,
		}
	case domain.PayrateDeletedEvent:
		return map[string]interface{}{
			"project_id":   e.ProjectID,
			"project_name": e.ProjectName,
		}

	// --- Transaction events ---
	case domain.TransactionCreatedEvent:
		return map[string]interface{}{
			"amount": e.Amount,
			"type":   e.Type,
		}
	case domain.TransactionUpdatedEvent:
		if len(e.ChangedFields) > 0 {
			return map[string]interface{}{"changed_fields": e.ChangedFields}
		}
	case domain.TransactionDeletedEvent:
		return map[string]interface{}{
			"amount":        e.Amount,
			"employee_name": e.EmployeeName,
			"employee_id":   e.EmployeeID,
		}
	case domain.TransactionSettledEvent:
		return map[string]interface{}{
			"amount":           e.Amount,
			"settlement_id":    e.SettlementID,
			"transaction_desc": e.TransactionDesc,
		}

	// --- Settlement events ---
	case domain.SettlementCreatedEvent:
		return map[string]interface{}{
			"transaction_id":   e.TransactionID,
			"amount":           e.Amount,
			"settlement_uuid":  e.SettlementUUID,
			"transaction_code": e.TransactionCode,
		}

	// --- Loan events ---
	case domain.LoanCreatedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"amount":        e.Amount,
		}
	case domain.LoanUpdatedEvent:
		if len(e.ChangedFields) > 0 {
			return map[string]interface{}{"changed_fields": e.ChangedFields}
		}
	case domain.LoanDeletedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"amount":        e.Amount,
		}
	case domain.LoanApprovedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"amount":        e.Amount,
		}
	case domain.LoanRejectedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"reason":        e.Reason,
		}
	case domain.LoanDisbursedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"amount":        e.Amount,
		}

	// --- Asset events ---
	case domain.AssetCreatedEvent:
		return map[string]interface{}{"file_name": e.FileName}
	case domain.AssetDeletedEvent:
		return map[string]interface{}{"file_name": e.FileName}

	// --- Lender events ---
	case domain.LenderCreatedEvent:
		return map[string]interface{}{"name": e.Name}
	case domain.LenderUpdatedEvent:
		if len(e.ChangedFields) > 0 {
			return map[string]interface{}{"changed_fields": e.ChangedFields}
		}
	case domain.LenderDeletedEvent:
		return map[string]interface{}{"name": e.Name}

	// --- Ledger events ---
	case domain.LedgerEntryCreatedEvent:
		return map[string]interface{}{
			"account": e.Account,
			"amount":  e.Amount,
		}
	case domain.LedgerEntryUpdatedEvent:
		return map[string]interface{}{
			"account": e.Account,
			"amount":  e.Amount,
		}
	case domain.LedgerEntryDeletedEvent:
		return map[string]interface{}{"account": e.Account}

	// --- Settings events ---
	case domain.SettingsCreatedEvent:
		return map[string]interface{}{"key": e.Key}
	case domain.SettingsUpdatedEvent:
		return map[string]interface{}{"key": e.Key}
	case domain.SettingsDeletedEvent:
		return map[string]interface{}{"key": e.Key}

	// --- Fee schedule events ---
	case domain.AdvancePaymentFeeScheduleCreatedEvent:
		return feeScheduleMetadata(e.ScheduleID, e.EffectiveDate, e.Summary)
	case domain.AdvancePaymentFeeScheduleUpdatedEvent:
		return feeScheduleMetadata(e.ScheduleID, e.EffectiveDate, e.Summary)
	case domain.AdvancePaymentFeeScheduleDeletedEvent:
		return feeScheduleMetadata(e.ScheduleID, e.EffectiveDate, e.Summary)
	case domain.DisbursementFeeScheduleCreatedEvent:
		return feeScheduleMetadata(e.ScheduleID, e.EffectiveDate, e.Summary)
	case domain.DisbursementFeeScheduleUpdatedEvent:
		return feeScheduleMetadata(e.ScheduleID, e.EffectiveDate, e.Summary)
	case domain.DisbursementFeeScheduleDeletedEvent:
		return feeScheduleMetadata(e.ScheduleID, e.EffectiveDate, e.Summary)
	case domain.WeeklyPaymentFeeScheduleCreatedEvent:
		return feeScheduleMetadata(e.ScheduleID, e.EffectiveDate, e.Summary)
	case domain.WeeklyPaymentFeeScheduleUpdatedEvent:
		return feeScheduleMetadata(e.ScheduleID, e.EffectiveDate, e.Summary)
	case domain.WeeklyPaymentFeeScheduleDeletedEvent:
		return feeScheduleMetadata(e.ScheduleID, e.EffectiveDate, e.Summary)

	// --- Import/Export events ---
	case domain.DataImportedEvent:
		return map[string]interface{}{
			"data_type":    e.DataType,
			"record_count": e.RecordCount,
			"file_name":    e.FileName,
		}
	case domain.DataExportedEvent:
		return map[string]interface{}{
			"data_type":    e.DataType,
			"record_count": e.RecordCount,
			"file_name":    e.FileName,
		}

	// --- Bulk Transfer events ---
	case domain.BulkTransferFileExportedEvent:
		return map[string]interface{}{
			"filename":           e.Filename,
			"cycle":              e.Cycle,
			"from_date":          e.FromDate,
			"to_date":            e.ToDate,
			"for_month":          e.ForMonth,
			"transactions_count": e.TransactionsCount,
			"total_amount":       e.TotalAmount,
		}
	case domain.BulkTransferResultImportedEvent:
		return map[string]interface{}{
			"asset_id":        e.AssetID,
			"filename":        e.Filename,
			"total_transfers": e.TotalTransfers,
			"completed":       e.CompletedCount,
			"failed":          e.FailedCount,
			"total_amount":    e.TotalAmount,
			"is_duplicate":    e.IsDuplicate,
		}
	case domain.BulkTransferFileDownloadedEvent:
		return map[string]interface{}{
			"bulk_transfer_file_id": e.BulkTransferFileID,
			"asset_id":              e.AssetID,
			"filename":              e.Filename,
		}
	case domain.PayrollHistoriesExportedEvent:
		return map[string]interface{}{
			"from_date": e.FromDate,
			"to_date":   e.ToDate,
			"filename":  e.Filename,
		}
	case domain.BulkTransferResultParsedEvent:
		return map[string]interface{}{
			"filename":     e.Filename,
			"total":        e.TotalTransfers,
			"completed":    e.CompletedCount,
			"failed":       e.FailedCount,
			"total_amount": e.TotalAmount,
		}
	case domain.BulkTransferPaymentStatusUpdatedEvent:
		return map[string]interface{}{
			"bulk_file_id":   e.BulkFileID,
			"payment_status": e.PaymentStatus,
			"total_updated":  e.TotalUpdated,
		}
	case domain.BulkTransferTransactionCreatedEvent:
		return map[string]interface{}{
			"bulk_file_id":   e.BulkFileID,
			"transaction_id": e.TransactionID,
			"amount":         e.Amount,
			"asset_id":       e.AssetID,
			"filename":       e.Filename,
		}

	// --- Settlement Upload events ---
	case domain.SettlementUploadProcessedEvent:
		return map[string]interface{}{
			"asset_id":          e.AssetID,
			"filename":          e.Filename,
			"settlement_amount": e.SettlementAmount,
			"processed_by":      e.ProcessedBy,
		}
	case domain.SettlementAppliedFromUploadEvent:
		return map[string]interface{}{
			"transaction_id": e.TransactionID,
			"amount":         e.Amount,
			"source":         e.Source,
			"file_id":        e.FileID,
			"filename":       e.Filename,
		}
	case domain.TimesheetsRevenuePaidFromInternalEvent:
		return map[string]interface{}{
			"file_id":  e.FileID,
			"filename": e.Filename,
		}

	// --- Access events ---
	case domain.EmployeeAccessGrantedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
			"role":          e.Role,
			"granted_by":    e.GrantedBy,
		}
	case domain.EmployeeAccessRevokedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
			"revoked_by":    e.RevokedBy,
		}
	case domain.ProjectAccessGrantedEvent:
		return map[string]interface{}{
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"role":          e.Role,
			"granted_by":    e.GrantedBy,
		}
	case domain.ProjectAccessRevokedEvent:
		return map[string]interface{}{
			"project_id":    e.ProjectID,
			"project_name":  e.ProjectName,
			"employee_id":   e.EmployeeID,
			"employee_name": e.EmployeeName,
			"revoked_by":    e.RevokedBy,
		}

	// --- Project Employees Synced ---
	case domain.ProjectEmployeesSyncedEvent:
		return map[string]interface{}{
			"employee_id":   e.EmployeeID,
			"old_fullname":  e.OldFullName,
			"new_fullname":  e.NewFullName,
			"updated_count": e.UpdatedCount,
		}

	// --- Timesheet Edit Request events ---
	case domain.TimesheetEditRequestCreatedEvent:
		return map[string]interface{}{
			"timesheet_id": e.TimesheetID,
			"requested_by": e.RequestedBy,
			"reason":       e.Reason,
		}
	case domain.TimesheetEditRequestUpdatedEvent:
		return map[string]interface{}{
			"request_id":   e.RequestID,
			"timesheet_id": e.TimesheetID,
			"requested_by": e.RequestedBy,
			"new_status":   e.NewStatus,
			"reason":       e.Reason,
		}

	// --- Timesheet Marking ---
	case domain.TimesheetMarkingEvent:
		return map[string]interface{}{
			"marked_as": e.MarkedAs,
			"source":    e.Source,
		}
	case domain.TimesheetBulkExternallyPaidEvent:
		return map[string]interface{}{
			"count":     e.Count,
			"reference": e.Reference,
		}
	}
	return nil
}

func feeScheduleMetadata(scheduleID, effectiveDate, summary string) map[string]interface{} {
	return map[string]interface{}{
		"schedule_id":    scheduleID,
		"effective_date": effectiveDate,
		"summary":        summary,
	}
}

// CanHandle returns true if this handler can process the given event type
func (h *AuditEventHandler) CanHandle(eventType string) bool {
	// Handle all events that have audit messages
	// If an event has GetAuditMessage() returning non-empty string, we handle it
	return true
}
