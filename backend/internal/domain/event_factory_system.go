package domain

import (
	"context"

	auditctx "api-server/internal/pkg/context"

	"github.com/google/uuid"
)

// NewDataImportedEvent creates a DataImportedEvent
func NewDataImportedEvent(ctx context.Context, dataType string, recordCount int, fileName string) DataImportedEvent {
	return DataImportedEvent{
		BaseEvent:   newBaseEvent(ctx, "DataImported", 0, AuditActionImport, EntityTypeAsset),
		DataType:    dataType,
		RecordCount: recordCount,
		FileName:    fileName,
	}
}

// NewDataExportedEvent creates a DataExportedEvent
func NewDataExportedEvent(ctx context.Context, dataType string, recordCount int, fileName string) DataExportedEvent {
	return DataExportedEvent{
		BaseEvent:   newBaseEvent(ctx, "DataExported", 0, AuditActionExport, EntityTypeAsset),
		DataType:    dataType,
		RecordCount: recordCount,
		FileName:    fileName,
	}
}

// NewBankCreatedEvent creates a BankCreatedEvent
func NewBankCreatedEvent(ctx context.Context, bank *Bank) BankCreatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeBank, actorName, bank.BranchName)
	return BankCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "BankCreated", bank.ID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeBank, auditMessage),
		Name:      bank.BranchName,
	}
}

// NewBankUpdatedEvent creates a BankUpdatedEvent
func NewBankUpdatedEvent(ctx context.Context, bank *Bank, original *Bank) BankUpdatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypeBank, actorName, bank.BranchName)
	return BankUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "BankUpdated", bank.ID, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypeBank, auditMessage),
		Name:          bank.BranchName,
		ChangedFields: CompareBanks(original, bank),
	}
}

// NewBankDeletedEvent creates a BankDeletedEvent
func NewBankDeletedEvent(ctx context.Context, bank *Bank) BankDeletedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeBank, actorName, bank.BranchName)
	return BankDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "BankDeleted", bank.ID, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeBank, auditMessage),
		Name:      bank.BranchName,
	}
}

// NewPayrateCreatedEvent creates a PayrateCreatedEvent
// Note: Payrate struct doesn't have EmployeeID or Rate - using ProjectID and 0 for now
func NewPayrateCreatedEvent(ctx context.Context, payrate *Payrate) PayrateCreatedEvent {
	projectName := ""
	if payrate.Project.ID != 0 {
		projectName = payrate.Project.Name
	}
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypePayrate, actorName, projectName)
	return PayrateCreatedEvent{
		BaseEvent:   newBaseEventWithActor(ctx, "PayrateCreated", payrate.ID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypePayrate, auditMessage),
		ProjectID:   payrate.ProjectID,
		ProjectName: projectName,
		EmployeeID:  0, // Payrate is per project, not per employee
		Rate:        0, // Rate is in PayrateJSON field, not a simple int64
	}
}

// NewPayrateUpdatedEvent creates a PayrateUpdatedEvent
func NewPayrateUpdatedEvent(ctx context.Context, payrate *Payrate) PayrateUpdatedEvent {
	projectName := ""
	if payrate.Project.ID != 0 {
		projectName = payrate.Project.Name
	}
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypePayrate, actorName, projectName)
	return PayrateUpdatedEvent{
		BaseEvent:   newBaseEventWithActor(ctx, "PayrateUpdated", payrate.ID, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypePayrate, auditMessage),
		ProjectID:   payrate.ProjectID,
		ProjectName: projectName,
		EmployeeID:  0,
		Rate:        0,
	}
}

// NewPayrateDeletedEvent creates a PayrateDeletedEvent
func NewPayrateDeletedEvent(ctx context.Context, payrate *Payrate) PayrateDeletedEvent {
	projectName := ""
	if payrate.Project.ID != 0 {
		projectName = payrate.Project.Name
	}
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypePayrate, actorName, projectName)
	return PayrateDeletedEvent{
		BaseEvent:   newBaseEventWithActor(ctx, "PayrateDeleted", payrate.ID, getUserIDFromContext(ctx), AuditActionDelete, EntityTypePayrate, auditMessage),
		ProjectID:   payrate.ProjectID,
		ProjectName: projectName,
		EmployeeID:  0,
	}
}

// NewAssetCreatedEvent creates an AssetCreatedEvent
func NewAssetCreatedEvent(ctx context.Context, asset *Asset) AssetCreatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeAsset, actorName, asset.Filename)
	return AssetCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "AssetCreated", asset.ID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeAsset, auditMessage),
		FileName:  asset.Filename,
	}
}

// NewAssetDeletedEvent creates an AssetDeletedEvent
func NewAssetDeletedEvent(ctx context.Context, asset *Asset) AssetDeletedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeAsset, actorName, asset.Filename)
	return AssetDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "AssetDeleted", asset.ID, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeAsset, auditMessage),
		FileName:  asset.Filename,
	}
}

// NewLenderCreatedEvent creates a LenderCreatedEvent
func NewLenderCreatedEvent(ctx context.Context, lender *Lender) LenderCreatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeLender, actorName, lender.Name)
	return LenderCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "LenderCreated", lender.ID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeLender, auditMessage),
		Name:      lender.Name,
	}
}

// NewLenderUpdatedEvent creates a LenderUpdatedEvent
func NewLenderUpdatedEvent(ctx context.Context, lender *Lender, original *Lender) LenderUpdatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypeLender, actorName, lender.Name)
	return LenderUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "LenderUpdated", lender.ID, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypeLender, auditMessage),
		Name:          lender.Name,
		ChangedFields: CompareLenders(original, lender),
	}
}

// NewLenderDeletedEvent creates a LenderDeletedEvent
func NewLenderDeletedEvent(ctx context.Context, lender *Lender) LenderDeletedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeLender, actorName, lender.Name)
	return LenderDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "LenderDeleted", lender.ID, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeLender, auditMessage),
		Name:      lender.Name,
	}
}

// NewLedgerEntryCreatedEvent creates a LedgerEntryCreatedEvent.
// The actor is taken from entry.CreatedBy because async workers (e.g. bulk transfer)
// invoke this with context.Background() — the entry itself is the source of truth.
func NewLedgerEntryCreatedEvent(ctx context.Context, entry *LedgerEntry) LedgerEntryCreatedEvent {
	amount := entry.Debit
	if amount == 0 {
		amount = entry.Credit
	}
	actorName := auditctx.GetFullName(ctx)
	amountText := formatVND(amount)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeLedgerEntry, actorName, string(entry.Account), amountText)
	return LedgerEntryCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "LedgerEntryCreated", entry.ID, entry.CreatedBy, AuditActionCreate, EntityTypeLedgerEntry, auditMessage),
		Account:   string(entry.Account),
		Amount:    amount,
	}
}

// NewLedgerEntryUpdatedEvent creates a LedgerEntryUpdatedEvent
func NewLedgerEntryUpdatedEvent(ctx context.Context, entry *LedgerEntry) LedgerEntryUpdatedEvent {
	amount := entry.Debit
	if amount == 0 {
		amount = entry.Credit
	}
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypeLedgerEntry, actorName, string(entry.Account))
	return LedgerEntryUpdatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "LedgerEntryUpdated", entry.ID, entry.CreatedBy, AuditActionUpdate, EntityTypeLedgerEntry, auditMessage),
		Account:   string(entry.Account),
		Amount:    amount,
	}
}

// NewLedgerEntryDeletedEvent creates a LedgerEntryDeletedEvent
func NewLedgerEntryDeletedEvent(ctx context.Context, entry *LedgerEntry) LedgerEntryDeletedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeLedgerEntry, actorName, string(entry.Account))
	return LedgerEntryDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "LedgerEntryDeleted", entry.ID, entry.CreatedBy, AuditActionDelete, EntityTypeLedgerEntry, auditMessage),
		Account:   string(entry.Account),
	}
}

// NewSettingsCreatedEvent creates a SettingsCreatedEvent
func NewSettingsCreatedEvent(ctx context.Context, key string, entityID uint) SettingsCreatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeSettings, actorName, key)
	return SettingsCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "SettingsCreated", entityID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeSettings, auditMessage),
		Key:       key,
	}
}

// NewSettingsUpdatedEvent creates a SettingsUpdatedEvent
func NewSettingsUpdatedEvent(ctx context.Context, key string, entityID uint) SettingsUpdatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypeSettings, actorName, key)
	return SettingsUpdatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "SettingsUpdated", entityID, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypeSettings, auditMessage),
		Key:       key,
	}
}

// NewSettingsDeletedEvent creates a SettingsDeletedEvent
func NewSettingsDeletedEvent(ctx context.Context, key string, entityID uint) SettingsDeletedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeSettings, actorName, key)
	return SettingsDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "SettingsDeleted", entityID, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeSettings, auditMessage),
		Key:       key,
	}
}

// NewBulkTransferFileExportedEvent creates a BulkTransferFileExportedEvent.
// EntityID is set to 0 because the export aggregates many timesheets/transactions
// rather than mutating a single domain row; the filename + asset/transaction-code
// link in the audit metadata is what an investigator joins on.
func NewBulkTransferFileExportedEvent(
	ctx context.Context,
	actorUserID uint,
	filename string,
	cycle string,
	fromDate, toDate, forMonth string,
	transactionsCount int,
	totalAmount int64,
	companyWide bool,
	forecastOutcomeItems []CashForecastOutcomeItem,
) BulkTransferFileExportedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(
		AuditActionExport,
		EntityTypeBulkTransferFile,
		actorName,
		filename,
		formatVND(totalAmount),
	)
	return BulkTransferFileExportedEvent{
		BaseEvent:            newBaseEventWithActor(ctx, "BulkTransferFileExported", 0, actorUserID, AuditActionExport, EntityTypeBulkTransferFile, auditMessage),
		Filename:             filename,
		Cycle:                cycle,
		FromDate:             fromDate,
		ToDate:               toDate,
		ForMonth:             forMonth,
		TransactionsCount:    transactionsCount,
		TotalAmount:          totalAmount,
		CompanyWide:          companyWide,
		ForecastOutcomeItems: forecastOutcomeItems,
	}
}

// NewBulkTransferResultImportedEvent creates a BulkTransferResultImportedEvent.
// EntityID is set to assetID since the asset is the durable handle for the
// uploaded result file and what admins click through to in the UI.
//
// The audit message follows the user-facing template from audit_templates.go:
// "<actor> nhập file kết quả chuyển lô tạm ứng, tổng giao dịch <amount> ₫".
// Filename, completed/failed counts, and the duplicate flag stay on the
// event payload (and on audit metadata) for forensic queries.
func NewBulkTransferResultImportedEvent(
	ctx context.Context,
	actorUserID uint,
	assetID uint,
	filename string,
	totalTransfers, completedCount, failedCount int,
	totalAmount int64,
	isDuplicate bool,
) BulkTransferResultImportedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(
		AuditActionImport,
		EntityTypeBulkTransferFile,
		actorName,
		formatVND(totalAmount),
	)
	return BulkTransferResultImportedEvent{
		BaseEvent:      newBaseEventWithActor(ctx, "BulkTransferResultImported", assetID, actorUserID, AuditActionImport, EntityTypeBulkTransferFile, auditMessage),
		AssetID:        assetID,
		Filename:       filename,
		TotalTransfers: totalTransfers,
		CompletedCount: completedCount,
		FailedCount:    failedCount,
		TotalAmount:    totalAmount,
		IsDuplicate:    isDuplicate,
	}
}

// NewBulkTransferFileDownloadedEvent creates a BulkTransferFileDownloadedEvent
// for re-downloads of previously-generated bulk transfer files.
func NewBulkTransferFileDownloadedEvent(
	ctx context.Context,
	actorUserID uint,
	bulkTransferFileID uint,
	assetID uint,
	filename string,
) BulkTransferFileDownloadedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(
		AuditActionView,
		EntityTypeBulkTransferFile,
		actorName,
		filename,
	)
	return BulkTransferFileDownloadedEvent{
		BaseEvent:          newBaseEventWithActor(ctx, "BulkTransferFileDownloaded", bulkTransferFileID, actorUserID, AuditActionView, EntityTypeBulkTransferFile, auditMessage),
		BulkTransferFileID: bulkTransferFileID,
		AssetID:            assetID,
		Filename:           filename,
	}
}

// NewPayrollHistoriesExportedEvent creates a PayrollHistoriesExportedEvent
// for cross-employee payroll exports.
func NewPayrollHistoriesExportedEvent(
	ctx context.Context,
	actorUserID uint,
	fromDate, toDate, filename string,
) PayrollHistoriesExportedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(
		AuditActionView,
		EntityTypePayroll,
		actorName,
		fromDate,
		toDate,
	)
	return PayrollHistoriesExportedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "PayrollHistoriesExported", 0, actorUserID, AuditActionView, EntityTypePayroll, auditMessage),
		FromDate:  fromDate,
		ToDate:    toDate,
		Filename:  filename,
	}
}

// NewBulkTransferResultParsedEvent creates a BulkTransferResultParsedEvent
func NewBulkTransferResultParsedEvent(
	ctx context.Context,
	bulkFileID uint,
	assetID uint,
	filename string,
	totalTransfers int,
	completedCount int,
	failedCount int,
	totalAmount int64,
	parsedDataJSON string,
	updatedDataJSON string,
	processedBy uint,
) BulkTransferResultParsedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionBulkCreate, EntityTypeTransaction, actorName, filename, completedCount, failedCount)
	return BulkTransferResultParsedEvent{
		BaseEvent:       newBaseEventWithActor(ctx, "BulkTransferResultParsed", bulkFileID, processedBy, AuditActionBulkCreate, EntityTypeTransaction, auditMessage),
		BulkFileID:      bulkFileID,
		AssetID:         assetID,
		Filename:        filename,
		TotalTransfers:  totalTransfers,
		CompletedCount:  completedCount,
		FailedCount:     failedCount,
		TotalAmount:     totalAmount,
		ParsedDataJSON:  parsedDataJSON,
		UpdatedDataJSON: updatedDataJSON,
		ProcessedBy:     processedBy,
		EventID:         uuid.NewString(),
	}
}

// NewBulkTransferPaymentStatusUpdatedEvent creates a BulkTransferPaymentStatusUpdatedEvent
func NewBulkTransferPaymentStatusUpdatedEvent(
	ctx context.Context,
	bulkFileID uint,
	timesheetIDs []uint,
	paymentStatus string,
	totalUpdated int,
) BulkTransferPaymentStatusUpdatedEvent {
	return BulkTransferPaymentStatusUpdatedEvent{
		BaseEvent:     newBaseEvent(ctx, "BulkTransferPaymentStatusUpdated", bulkFileID, AuditActionUpdate, EntityTypeTransaction),
		BulkFileID:    bulkFileID,
		TimesheetIDs:  timesheetIDs,
		PaymentStatus: paymentStatus,
		TotalUpdated:  totalUpdated,
	}
}

// NewBulkTransferTransactionCreatedEvent creates a BulkTransferTransactionCreatedEvent
func NewBulkTransferTransactionCreatedEvent(
	ctx context.Context,
	bulkFileID uint,
	transactionID uint,
	amount int64,
	assetID uint,
	filename string,
) BulkTransferTransactionCreatedEvent {
	return BulkTransferTransactionCreatedEvent{
		BaseEvent:     newBaseEvent(ctx, "BulkTransferTransactionCreated", bulkFileID, AuditActionCreate, EntityTypeTransaction),
		BulkFileID:    bulkFileID,
		TransactionID: transactionID,
		Amount:        amount,
		AssetID:       assetID,
		Filename:      filename,
	}
}
