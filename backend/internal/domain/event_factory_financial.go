package domain

import (
	"context"
	"fmt"
)

// NewTransactionCreatedEvent creates a TransactionCreatedEvent.
// Actor falls back to txn.CreatedBy when ctx has none (async worker path).
func NewTransactionCreatedEvent(ctx context.Context, txn *Transaction) TransactionCreatedEvent {
	var employeeName string
	relatedType := new(string)
	relatedID := new(uint)

	if txn.UserID != nil {
		*relatedType = "user"
		*relatedID = *txn.UserID
	} else if txn.LoanID != nil {
		*relatedType = "loan"
		*relatedID = *txn.LoanID
	}

	entityName := fmt.Sprintf("%d||%s", txn.Amount, string(txn.TransactionType))
	auditMessage := BuildAuditMessage(AuditActionCreate, EntityTypeTransaction, entityName)
	return TransactionCreatedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "TransactionCreated", txn.ID, txn.CreatedBy, AuditActionCreate, EntityTypeTransaction, auditMessage),
		Amount:       txn.Amount,
		Type:         string(txn.TransactionType),
		EmployeeID:   0,
		EmployeeName: employeeName,
		RelatedType:  relatedType,
		RelatedID:    relatedID,
	}
}

// NewTransactionCreatedEventWithAudit creates a TransactionCreatedEvent with audit message.
// Actor falls back to txn.CreatedBy when ctx has none (async worker path).
func NewTransactionCreatedEventWithAudit(ctx context.Context, txn *Transaction, auditMessage string) TransactionCreatedEvent {
	var employeeName string
	relatedType := new(string)
	relatedID := new(uint)

	if txn.UserID != nil {
		*relatedType = "user"
		*relatedID = *txn.UserID
	} else if txn.LoanID != nil {
		*relatedType = "loan"
		*relatedID = *txn.LoanID
	}

	return TransactionCreatedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "TransactionCreated", txn.ID, txn.CreatedBy, AuditActionCreate, EntityTypeTransaction, auditMessage),
		Amount:       txn.Amount,
		Type:         string(txn.TransactionType),
		EmployeeID:   0,
		EmployeeName: employeeName,
		RelatedType:  relatedType,
		RelatedID:    relatedID,
	}
}

// NewTransactionUpdatedEvent creates a TransactionUpdatedEvent
func NewTransactionUpdatedEvent(ctx context.Context, txn *Transaction, original *Transaction) TransactionUpdatedEvent {
	entityName := fmt.Sprintf("%d||%s", txn.Amount, string(txn.TransactionType))
	auditMessage := BuildAuditMessage(AuditActionUpdate, EntityTypeTransaction, entityName)
	return TransactionUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "TransactionUpdated", txn.ID, txn.CreatedBy, AuditActionUpdate, EntityTypeTransaction, auditMessage),
		Amount:        txn.Amount,
		EmployeeID:    0,
		EmployeeName:  "",
		ChangedFields: CompareTransactions(original, txn),
	}
}

// NewTransactionDeletedEvent creates a TransactionDeletedEvent
func NewTransactionDeletedEvent(ctx context.Context, txn *Transaction) TransactionDeletedEvent {
	entityName := fmt.Sprintf("%d||%s", txn.Amount, string(txn.TransactionType))
	auditMessage := BuildAuditMessage(AuditActionDelete, EntityTypeTransaction, entityName)
	return TransactionDeletedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "TransactionDeleted", txn.ID, txn.CreatedBy, AuditActionDelete, EntityTypeTransaction, auditMessage),
		Amount:       txn.Amount,
		EmployeeName: "",
		EmployeeID:   0,
	}
}

// NewTransactionSettledEvent creates a TransactionSettledEvent
func NewTransactionSettledEvent(ctx context.Context, txn *Transaction, settlement *Settlement) TransactionSettledEvent {
	return TransactionSettledEvent{
		BaseEvent:       newBaseEvent(ctx, "TransactionSettled", txn.ID, AuditActionSettle, EntityTypeTransaction),
		Amount:          settlement.Amount,
		SettlementID:    settlement.ID,
		TransactionDesc: txn.Description,
	}
}

// NewLoanCreatedEvent creates a LoanCreatedEvent
func NewLoanCreatedEvent(ctx context.Context, loan *Loan) LoanCreatedEvent {
	lenderName := ""
	if loan.Lender != nil {
		lenderName = loan.Lender.Name
	}
	entityName := fmt.Sprintf("%s||%d", lenderName, loan.PrincipalAmount)
	auditMessage := BuildAuditMessage(AuditActionCreate, EntityTypeLoan, entityName)
	return LoanCreatedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "LoanCreated", loan.ID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeLoan, auditMessage),
		EmployeeID:   loan.LenderID,
		EmployeeName: lenderName,
		Amount:       loan.PrincipalAmount,
	}
}

// NewLoanUpdatedEvent creates a LoanUpdatedEvent
func NewLoanUpdatedEvent(ctx context.Context, loan *Loan, original *Loan) LoanUpdatedEvent {
	lenderName := ""
	if loan.Lender != nil {
		lenderName = loan.Lender.Name
	}
	entityName := fmt.Sprintf("%s||%d", lenderName, loan.PrincipalAmount)
	auditMessage := BuildAuditMessage(AuditActionUpdate, EntityTypeLoan, entityName)
	return LoanUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "LoanUpdated", loan.ID, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypeLoan, auditMessage),
		EmployeeID:    loan.LenderID,
		EmployeeName:  lenderName,
		Amount:        loan.PrincipalAmount,
		ChangedFields: CompareLoans(original, loan),
	}
}

// NewLoanDeletedEvent creates a LoanDeletedEvent
func NewLoanDeletedEvent(ctx context.Context, loan *Loan) LoanDeletedEvent {
	lenderName := ""
	if loan.Lender != nil {
		lenderName = loan.Lender.Name
	}
	entityName := fmt.Sprintf("%s||%d", lenderName, loan.PrincipalAmount)
	auditMessage := BuildAuditMessage(AuditActionDelete, EntityTypeLoan, entityName)
	return LoanDeletedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "LoanDeleted", loan.ID, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeLoan, auditMessage),
		EmployeeID:   loan.LenderID,
		EmployeeName: lenderName,
		Amount:       loan.PrincipalAmount,
	}
}

// NewLoanApprovedEvent creates a LoanApprovedEvent
func NewLoanApprovedEvent(ctx context.Context, loan *Loan) LoanApprovedEvent {
	lenderName := ""
	if loan.Lender != nil {
		lenderName = loan.Lender.Name
	}
	entityName := fmt.Sprintf("%s||%d", lenderName, loan.PrincipalAmount)
	auditMessage := BuildAuditMessage(AuditActionApprove, EntityTypeLoan, entityName)
	return LoanApprovedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "LoanApproved", loan.ID, getUserIDFromContext(ctx), AuditActionApprove, EntityTypeLoan, auditMessage),
		EmployeeID:   loan.LenderID,
		EmployeeName: lenderName,
		Amount:       loan.PrincipalAmount,
	}
}

// NewLoanRejectedEvent creates a LoanRejectedEvent
func NewLoanRejectedEvent(ctx context.Context, loan *Loan, reason string) LoanRejectedEvent {
	lenderName := ""
	if loan.Lender != nil {
		lenderName = loan.Lender.Name
	}
	// For rejected loans, use lenderName only (reason is stored separately)
	auditMessage := BuildAuditMessage(AuditActionReject, EntityTypeLoan, lenderName)
	return LoanRejectedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "LoanRejected", loan.ID, getUserIDFromContext(ctx), AuditActionReject, EntityTypeLoan, auditMessage),
		EmployeeID:   loan.LenderID,
		EmployeeName: lenderName,
		Reason:       reason,
	}
}

// NewLoanDisbursedEvent creates a LoanDisbursedEvent
func NewLoanDisbursedEvent(ctx context.Context, loan *Loan) LoanDisbursedEvent {
	lenderName := ""
	if loan.Lender != nil {
		lenderName = loan.Lender.Name
	}
	entityName := fmt.Sprintf("%s||%d", lenderName, loan.PrincipalAmount)
	auditMessage := BuildAuditMessage(AuditActionCreate, EntityTypeLoan, entityName)
	return LoanDisbursedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "LoanDisbursed", loan.ID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeLoan, auditMessage),
		EmployeeID:   loan.LenderID,
		EmployeeName: lenderName,
		Amount:       loan.PrincipalAmount,
	}
}

// NewSettlementCreatedEvent creates a SettlementCreatedEvent
func NewSettlementCreatedEvent(ctx context.Context, settlement *Settlement, txn *Transaction) SettlementCreatedEvent {
	var transactionCode *string
	if txn != nil {
		transactionCode = txn.TransactionCode
	}
	return SettlementCreatedEvent{
		BaseEvent:       newBaseEvent(ctx, "SettlementCreated", settlement.ID, AuditActionCreate, EntityTypeTransaction),
		TransactionID:   settlement.TransactionID,
		Amount:          settlement.Amount,
		SettlementUUID:  settlement.SettlementUUID,
		TransactionCode: transactionCode,
	}
}

// NewSettlementCreatedEventWithAudit creates a SettlementCreatedEvent with audit message
func NewSettlementCreatedEventWithAudit(ctx context.Context, settlement *Settlement, txn *Transaction, auditMessage string) SettlementCreatedEvent {
	var transactionCode *string
	if txn != nil {
		transactionCode = txn.TransactionCode
	}
	return SettlementCreatedEvent{
		BaseEvent:       newBaseEventWithAudit(ctx, "SettlementCreated", settlement.ID, AuditActionCreate, EntityTypeTransaction, auditMessage),
		TransactionID:   settlement.TransactionID,
		Amount:          settlement.Amount,
		SettlementUUID:  settlement.SettlementUUID,
		TransactionCode: transactionCode,
	}
}

// NewSettlementUploadProcessedEvent creates a SettlementUploadProcessedEvent
func NewSettlementUploadProcessedEvent(
	ctx context.Context,
	assetID uint,
	filename string,
	timesheetIDs []uint,
	settlementAmount int64,
	transactionMap map[uint][]uint,
	processedBy uint,
) SettlementUploadProcessedEvent {
	return SettlementUploadProcessedEvent{
		BaseEvent:        newBaseEvent(ctx, "SettlementUploadProcessed", assetID, AuditActionCreate, EntityTypeAsset),
		AssetID:          assetID,
		Filename:         filename,
		TimesheetIDs:     timesheetIDs,
		SettlementAmount: settlementAmount,
		TransactionMap:   transactionMap,
		ProcessedBy:      processedBy,
	}
}

// NewSettlementAppliedFromUploadEvent creates a SettlementAppliedFromUploadEvent.
// timesheetIDs are the timesheets to mark revenue_paid=1 atomically with settlement creation.
func NewSettlementAppliedFromUploadEvent(
	ctx context.Context,
	transactionID uint,
	amount int64,
	fileID uint,
	filename string,
	timesheetIDs []uint,
) SettlementAppliedFromUploadEvent {
	return SettlementAppliedFromUploadEvent{
		BaseEvent:     newBaseEvent(ctx, "SettlementAppliedFromUpload", transactionID, AuditActionSettle, EntityTypeTransaction),
		TransactionID: transactionID,
		Amount:        amount,
		Source:        "upload",
		FileID:        fileID,
		Filename:      filename,
		TimesheetIDs:  timesheetIDs,
	}
}
