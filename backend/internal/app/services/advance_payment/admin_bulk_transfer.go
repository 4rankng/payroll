package advance_payment

import (
	"api-server/internal/pkg/clock"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	pkgConstants "api-server/internal/pkg/constants"
)

func (s *Service) saveBulkTransferFile(ctx context.Context, fileDataArray []dto.BulkTransferFileData, totalAmount int64, fromDate, toDate time.Time, forMonth string, userID uint) (string, error) {
	fileUUID := uuid.New()
	uuidStr := strings.ReplaceAll(fileUUID.String(), "-", "")

	filename := fmt.Sprintf("%s_%s_%s", pkgConstants.MBankPrefix, pkgConstants.CycleFlexible, uuidStr)

	sort.Slice(fileDataArray, func(i, j int) bool {
		return fileDataArray[i].STT < fileDataArray[j].STT
	})

	var (
		dataJSON string
	)

	if len(fileDataArray) == 0 {
		dataJSON = "{}"
	} else {
		_ = s.calculateChecksum(fileDataArray)

		dataBytes, err := json.Marshal(fileDataArray)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal data")
		}
		dataJSON = string(dataBytes)
	}

	file := &domain.BulkTransferFile{
		Filename:          filename,
		Cycle:             domain.StringPtr(pkgConstants.CycleFlexible),
		CreatedBy:         userID,
		FromDate:          &fromDate,
		ToDate:            &toDate,
		ForMonth:          &forMonth,
		TransactionsCount: len(fileDataArray),
		CompletedCount:    0,
		FailedCount:       0,
		TransferAmount:    totalAmount,
		Data:              dataJSON,
	}

	if err := s.config.BulkTransferFileRepo.Create(ctx, file); err != nil {
		return "", errors.Wrap(err, "failed to create bulk transfer file record")
	}

	return filename, nil
}

func (s *Service) calculateChecksum(data []dto.BulkTransferFileData) string {
	if len(data) == 0 {
		return ""
	}

	transactionCodes := make([]string, 0, len(data))
	for _, d := range data {
		if d.TransactionCode != "" {
			transactionCodes = append(transactionCodes, d.TransactionCode)
		}
	}

	if len(transactionCodes) == 0 {
		return ""
	}

	sort.Strings(transactionCodes)
	concatenated := strings.Join(transactionCodes, "")
	hash := sha256.Sum256([]byte(concatenated))
	return hex.EncodeToString(hash[:])
}

func (s *Service) ProcessBankResult(ctx context.Context, file *excelize.File) (*dto.AdvancePaymentUploadResultResponse, error) {
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("no sheets found in Excel file")
	}

	rows, err := file.GetRows(sheets[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to read Excel file")
	}

	now := clock.Now()
	var resultItems []dto.AdvancePaymentResultItem
	completedCount := 0
	failedCount := 0

	var grandTotalNetAmount, grandTotalFee int64
	var detectedForMonth string
	var allCompletedRequestIDs []uint64

	for _, br := range extractBankResultRows(rows) {
		txCode := br.TxCode
		status := br.Status
		paymentRef := br.PaymentRef

		tc, err := s.config.TransactionCodeRepo.GetByCode(ctx, txCode)
		if err != nil {
			s.logger.Warn("transaction code not found", "code", txCode)
			continue
		}

		var codeData domain.TransactionCodeData
		if err := json.Unmarshal(tc.Data, &codeData); err != nil {
			s.logger.Error("failed to parse transaction code data", "error", err)
			continue
		}

		requestStatus := domain.AdvancePaymentStatusCompleted
		if status == "FAILED" || status == "THẤT BẠI" {
			requestStatus = domain.AdvancePaymentStatusFailed
		}

		// Look up employee info for result
		var employeeName, employeeBank, employeeAccountNumber, employeeCCCD string
		var amount int64

		if tc != nil {
			if err := json.Unmarshal(tc.Data, &codeData); err == nil && len(codeData.GetRequestIDs()) > 0 {
				req, err := s.config.AdvancePaymentRequestRepo.GetByID(ctx, codeData.GetRequestIDs()[0])
				if err == nil {
					amount = int64(req.NetAmount)
					employee, err := s.config.EmployeeRepo.GetByID(ctx, req.EmployeeID)
					if err == nil {
						employeeName = employee.Fullname
						employeeCCCD = employee.CCCD
						employeeAccountNumber = employee.BankAccountNumber
						if employee.BankID != nil {
							bank, err := s.config.BankRepo.GetByID(ctx, *employee.BankID)
							if err == nil {
								employeeBank = bank.BranchName
							}
						}
					}
				}
			}
		}

		paymentStatus := "paid"
		paidAtStr := now.Format(time.RFC3339)
		if requestStatus == domain.AdvancePaymentStatusFailed {
			paymentStatus = "failed"
			failedCount++
		} else {
			completedCount++
		}

		resultItems = append(resultItems, dto.AdvancePaymentResultItem{
			Row:                   br.Row,
			EmployeeName:          employeeName,
			EmployeeBank:          employeeBank,
			EmployeeAccountNumber: employeeAccountNumber,
			EmployeeCCCD:          employeeCCCD,
			Amount:                fmt.Sprintf("%d", amount),
			PaymentStatus:         paymentStatus,
			PaidAt:                &paidAtStr,
		})

		if err := s.config.AdvancePaymentRequestRepo.BatchUpdateStatus(ctx, codeData.GetRequestIDs(), requestStatus, paymentRef, &now); err != nil {
			s.logger.Error("failed to update request status", "error", err, "request_ids", codeData.GetRequestIDs())
			continue
		}

		// Notify employees about advance payment status change
		if s.config.EmployeeNotifier != nil {
			for _, reqID := range codeData.GetRequestIDs() {
				req, err := s.config.AdvancePaymentRequestRepo.GetByID(ctx, reqID)
				if err != nil {
					continue
				}
				s.config.EmployeeNotifier.NotifyAdvancePaymentStatusChanged(ctx, req.EmployeeID, requestStatus, req.RequestAmount)
			}
		}

		if requestStatus == domain.AdvancePaymentStatusCompleted {
			for _, reqID := range codeData.GetRequestIDs() {
				req, err := s.config.AdvancePaymentRequestRepo.GetByID(ctx, reqID)
				if err != nil {
					continue
				}
				grandTotalNetAmount += int64(req.NetAmount)
				grandTotalFee += int64(req.Fee)

				if detectedForMonth == "" && req.AdvPayID > 0 {
					advPay, err := s.config.AdvancePaymentRepo.GetByID(ctx, uint64(req.AdvPayID))
					if err == nil && advPay.ForMonth != "" {
						detectedForMonth = advPay.ForMonth
					}
				}
			}
			allCompletedRequestIDs = append(allCompletedRequestIDs, codeData.GetRequestIDs()...)
		}

		s.logger.Info("updated advance payment requests",
			"tx_code", txCode,
			"request_ids", codeData.GetRequestIDs(),
			"status", requestStatus,
		)
	}

	// Create ONE consolidated transaction + ledger entries for the entire file
	if grandTotalNetAmount > 0 {
		partnerCompany := s.config.GetPartnerCompany(ctx)
		receivableAmount := grandTotalNetAmount + grandTotalFee

		transaction := &domain.Transaction{
			Amount:          receivableAmount,
			TransactionType: domain.TransactionTypeRevenue,
			Description:     fmt.Sprintf("Ứng lương TingTing %s", detectedForMonth),
			Status:          domain.TransactionStatusPending,
			Party:           partnerCompany,
			CreatedBy:       constants.SystemUserID,
		}
		if err := s.config.TransactionRepo.Create(ctx, transaction); err != nil {
			s.logger.Error("failed to create consolidated transaction", "error", err)
		} else {
			if err := s.config.AdvancePaymentRequestRepo.UpdateSettlementTransactionID(ctx, allCompletedRequestIDs, transaction.ID); err != nil {
				s.logger.Error("failed to link requests to transaction", "error", err)
			}

			ledgerEntries := domain.BuildDisbursementSettlementEntries(
				transaction.ID, grandTotalNetAmount, grandTotalFee,
				partnerCompany, constants.SystemUserID, now,
			)
			if err := s.config.LedgerRepo.CreateTransaction(ctx, ledgerEntries); err != nil {
				s.logger.Error("failed to create ledger entries", "error", err)
			}

			s.logger.Info("created consolidated transaction and ledger entries for advance payment",
				"transaction_id", transaction.ID,
				"receivable_amount", receivableAmount,
				"cash_out_amount", grandTotalNetAmount,
				"fee_amount", grandTotalFee,
				"request_count", len(allCompletedRequestIDs))
		}
	}

	return &dto.AdvancePaymentUploadResultResponse{
		Data:         resultItems,
		TotalTxn:     len(resultItems),
		CompletedTxn: completedCount,
		FailedTxn:    failedCount,
	}, nil
}
