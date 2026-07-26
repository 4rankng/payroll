package main

import (
	"context"
	"fmt"
	"os"
	"time"

	appConfig "api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/infra/events"
	"api-server/internal/infra/persistence"
	auditctx "api-server/internal/pkg/context"
)

const (
	transactionID = uint(99)
	timesheetID   = uint(11579)
	proofAssetID  = uint(122)
	settlementVND = int64(346800)
)

func main() {
	ctx := auditctx.WithUserID(context.Background(), 1)

	cfg, err := appConfig.Load()
	must(err)

	db, err := persistence.NewDatabase(persistence.DatabaseConfig{
		Driver:          cfg.DB.Driver,
		DSN:             cfg.DB.DSN,
		MaxIdleConns:    1,
		MaxOpenConns:    2,
		ConnMaxLifetime: time.Minute,
	})
	must(err)
	defer func() { must(db.Close()) }()

	projectEmployeeRepo := persistence.NewProjectEmployeeRepository(db)
	timesheetRepo := persistence.NewTimesheetRepository(db, projectEmployeeRepo)
	transactionRepo := persistence.NewTransactionRepository(db.DB)
	settlementRepo := persistence.NewSettlementRepository(db.DB)
	ledgerRepo := persistence.NewLedgerEntryRepository(db)

	var timesheet domain.Timesheet
	must(db.WithContext(ctx).First(&timesheet, timesheetID).Error)
	if timesheet.TransactionID == nil ||
		*timesheet.TransactionID != transactionID ||
		timesheet.PaidAmount != 340000 ||
		timesheet.RevenueReceivable != settlementVND ||
		timesheet.RevenuePaid {
		fail("timesheet precondition failed: id=%d transaction=%v paid=%d receivable=%d revenue_paid=%t",
			timesheet.ID, timesheet.TransactionID, timesheet.PaidAmount, timesheet.RevenueReceivable, timesheet.RevenuePaid)
	}

	var transaction domain.Transaction
	must(db.WithContext(ctx).First(&transaction, transactionID).Error)
	if transaction.Amount != settlementVND ||
		transaction.SettledAmount != 0 ||
		transaction.Status != domain.TransactionStatusPending {
		fail("transaction precondition failed: id=%d amount=%d settled=%d status=%s",
			transaction.ID, transaction.Amount, transaction.SettledAmount, transaction.Status)
	}

	var settlementCount int64
	must(db.WithContext(ctx).Model(&domain.Settlement{}).
		Where("transaction_id = ?", transactionID).
		Count(&settlementCount).Error)
	if settlementCount != 0 {
		fail("transaction %d already has %d settlements", transactionID, settlementCount)
	}

	var proofAssetCount int64
	must(db.WithContext(ctx).Model(&domain.Asset{}).
		Where("id = ?", proofAssetID).
		Count(&proofAssetCount).Error)
	if proofAssetCount != 1 {
		fail("proof asset %d not found", proofAssetID)
	}

	handler := events.NewSettlementEventHandler(
		db.DB,
		timesheetRepo,
		transactionRepo,
		settlementRepo,
		ledgerRepo,
		nil,
		nil,
		nil,
	)
	must(handler.ApplySettlement(
		ctx,
		transactionID,
		settlementVND,
		proofAssetID,
		"sao_ke_tt_2026-04.xlsx (recovery for omitted transaction #99)",
		[]uint{timesheetID},
	))

	var repairedTimesheet domain.Timesheet
	must(db.WithContext(ctx).First(&repairedTimesheet, timesheetID).Error)
	var repairedTransaction domain.Transaction
	must(db.WithContext(ctx).First(&repairedTransaction, transactionID).Error)
	var repairedSettlement domain.Settlement
	must(db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		First(&repairedSettlement).Error)

	if !repairedTimesheet.RevenuePaid ||
		repairedTransaction.Status != domain.TransactionStatusSettled ||
		repairedTransaction.SettledAmount != settlementVND ||
		repairedSettlement.Amount != settlementVND ||
		repairedSettlement.ProofAssetID == nil ||
		*repairedSettlement.ProofAssetID != proofAssetID {
		fail("postcondition failed: revenue_paid=%t status=%s settled=%d settlement=%d proof=%v",
			repairedTimesheet.RevenuePaid,
			repairedTransaction.Status,
			repairedTransaction.SettledAmount,
			repairedSettlement.Amount,
			repairedSettlement.ProofAssetID)
	}

	fmt.Printf("repaired timesheet=%d transaction=%d settlement=%d amount=%d proof_asset=%d\n",
		timesheetID, transactionID, repairedSettlement.ID, settlementVND, proofAssetID)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
