package flex_pay

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"

	"github.com/xuri/excelize/v2"
)

type FlexPaySettlementService struct {
	logger            *slog.Logger
	requestRepo       domain.AdvancePaymentRequestRepository
	advPayRepo        domain.AdvancePaymentRepository
	ledgerRepo        domain.LedgerEntryRepository
	transactionRepo   domain.TransactionRepository
	getPartnerCompany func(ctx context.Context) string
}

func NewFlexPaySettlementService(
	requestRepo domain.AdvancePaymentRequestRepository,
	advPayRepo domain.AdvancePaymentRepository,
	ledgerRepo domain.LedgerEntryRepository,
	transactionRepo domain.TransactionRepository,
	getPartnerCompany func(ctx context.Context) string,
) *FlexPaySettlementService {
	return &FlexPaySettlementService{
		logger:            observability.GetLogger(),
		requestRepo:       requestRepo,
		advPayRepo:        advPayRepo,
		ledgerRepo:        ledgerRepo,
		transactionRepo:   transactionRepo,
		getPartnerCompany: getPartnerCompany,
	}
}

type SettlementResult struct {
	Success      bool      `json:"success"`
	Message      string    `json:"message"`
	SettledCount int64     `json:"settled_count"`
	RequestIDs   []uint64  `json:"request_ids,omitempty"`
	SettledAt    time.Time `json:"settled_at"`
}

func (s *FlexPaySettlementService) ProcessSettlementFile(ctx context.Context, file *excelize.File) (*SettlementResult, error) {
	s.logger.Info("Starting reconciliation file settlement processing")

	requestIDs, err := s.extractRequestIDsFromInternalSheet(file)
	if err != nil {
		s.logger.Error("Failed to extract request IDs from INTERNAL sheet", "error", err)
		return nil, fmt.Errorf("failed to extract request IDs: %w", err)
	}

	if len(requestIDs) == 0 {
		s.logger.Info("No request IDs found in INTERNAL sheet")
		return &SettlementResult{
			Success:      true,
			Message:      "No request IDs found in the reconciliation file",
			SettledCount: 0,
			SettledAt:    clock.Now(),
		}, nil
	}

	s.logger.Info("Extracted request IDs from file", "count", len(requestIDs), "ids", requestIDs)

	requests, err := s.requestRepo.GetByIDs(ctx, requestIDs)
	if err != nil {
		s.logger.Error("Failed to get requests by IDs", "error", err)
		return nil, fmt.Errorf("failed to get requests: %w", err)
	}

	var totalSettledAmount int64
	var forMonth string
	for _, req := range requests {
		totalSettledAmount += int64(req.RequestAmount)
		if forMonth == "" && req.AdvPayID > 0 {
			advPay, err := s.advPayRepo.GetByID(ctx, uint64(req.AdvPayID))
			if err == nil && advPay.ForMonth != "" {
				forMonth = advPay.ForMonth
			}
		}
	}

	settledAt := clock.Now()
	settledCount, err := s.requestRepo.MarkReceivableSettled(ctx, requestIDs, settledAt)
	if err != nil {
		s.logger.Error("Failed to mark requests as settled", "error", err, "count", len(requestIDs))
		return nil, fmt.Errorf("failed to mark requests as settled: %w", err)
	}

	if totalSettledAmount > 0 && len(requests) > 0 {
		partnerCompany := s.getPartnerCompany(ctx)

		// Collect all unique pending transactions linked to the requests.
		// Each request may have its own settlement transaction, so we settle them all.
		txnAmounts := make(map[uint]int64)
		txnMap := make(map[uint]*domain.Transaction)
		for _, req := range requests {
			if req.SettlementTransactionID == nil || *req.SettlementTransactionID == 0 {
				continue
			}
			txnID := *req.SettlementTransactionID
			if _, seen := txnMap[txnID]; seen {
				txnAmounts[txnID] += int64(req.RequestAmount)
				continue
			}
			tx, err := s.transactionRepo.GetByID(ctx, txnID)
			if err != nil || tx.Status != domain.TransactionStatusPending {
				continue
			}
			txnMap[txnID] = tx
			txnAmounts[txnID] += int64(req.RequestAmount)
		}

		for txnID, tx := range txnMap {
			amount := txnAmounts[txnID]
			s.logger.Info("Settling advance payment transaction",
				"transaction_id", txnID,
				"amount", tx.Amount,
				"settlement_amount", amount)

			tx.Status = domain.TransactionStatusSettled
			tx.SettledAmount = tx.Amount
			if err := s.transactionRepo.Update(ctx, tx); err != nil {
				s.logger.Error("Failed to update transaction status to settled",
					"transaction_id", txnID, "error", err)
				continue
			}

			txIDCopy := txnID
			ledgerEntries := []*domain.LedgerEntry{
				{
					Date:          settledAt,
					Account:       domain.AccountCash,
					Party:         partnerCompany,
					Debit:         amount,
					Credit:        0,
					CreatedBy:     constants.SystemUserID,
					TransactionID: &txIDCopy,
				},
				{
					Date:          settledAt,
					Account:       domain.AccountReceivable,
					Party:         partnerCompany,
					Debit:         0,
					Credit:        amount,
					CreatedBy:     constants.SystemUserID,
					TransactionID: &txIDCopy,
				},
			}
			if err := s.ledgerRepo.CreateTransaction(ctx, ledgerEntries); err != nil {
				s.logger.Error("Failed to create settlement ledger entries",
					"transaction_id", txnID, "error", err)
			}

			s.logger.Info("Successfully settled advance payment transaction",
				"transaction_id", txnID, "settled_amount", amount)
		}

		if len(txnMap) == 0 {
			s.logger.Warn("No matching pending transactions found for settlement",
				"request_count", len(requests),
				"total_amount", totalSettledAmount)
		}
	}

	s.logger.Info("Successfully settled receivables",
		"settled_count", settledCount,
		"total_ids", len(requestIDs),
		"total_amount", totalSettledAmount,
		"settled_at", settledAt)

	return &SettlementResult{
		Success:      true,
		Message:      fmt.Sprintf("Successfully settled %d receivables", settledCount),
		SettledCount: settledCount,
		RequestIDs:   requestIDs,
		SettledAt:    settledAt,
	}, nil
}

func (s *FlexPaySettlementService) extractRequestIDsFromInternalSheet(f *excelize.File) ([]uint64, error) {
	const internalSheetName = "INTERNAL"

	sheetIndex, err := f.GetSheetIndex(internalSheetName)
	if err != nil || sheetIndex < 0 {
		return nil, fmt.Errorf("INTERNAL sheet not found in the uploaded file. Please upload the correct reconciliation file")
	}

	rows, err := f.GetRows(internalSheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read INTERNAL sheet: %w", err)
	}

	var requestIDs []uint64
	for i, row := range rows {
		if len(row) == 0 || row[0] == "" {
			continue
		}

		if i == 0 && isHeaderRow(row[0]) {
			continue
		}

		id, err := strconv.ParseUint(row[0], 10, 64)
		if err != nil {
			s.logger.Warn("Failed to parse request ID", "row", i+1, "value", row[0], "error", err)
			continue
		}

		requestIDs = append(requestIDs, id)
	}

	return requestIDs, nil
}

func isHeaderRow(value string) bool {
	if strings.HasPrefix(value, "type:") {
		return true
	}
	headerPatterns := []string{"request_id", "id", "ID", "Request ID", "REQUEST_ID"}
	for _, pattern := range headerPatterns {
		if value == pattern {
			return true
		}
	}
	return false
}
