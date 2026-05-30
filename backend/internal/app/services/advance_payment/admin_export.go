package advance_payment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	pkgConstants "api-server/internal/pkg/constants"
)

func (s *Service) ExportToExcel(ctx context.Context, fromDate, toDate time.Time, userID uint) (*excelize.File, error) {
	currentMonth := GetCurrentMonth()

	grouped, err := s.config.AdvancePaymentRequestRepo.GetPendingGroupedByEmployee(ctx, currentMonth)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pending requests")
	}

	// Open MBank template first to handle empty case
	f, err := excelize.OpenFile(pkgConstants.MBankTemplatePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open MBank template")
	}

	if err := s.clearExistingDataMBank(f); err != nil {
		_ = f.Close()
		return nil, errors.Wrap(err, "failed to clear existing data")
	}

	// Return empty template if no records
	if len(grouped) == 0 {
		s.logger.Info("no pending requests to export, returning empty template", "forMonth", currentMonth)
		return f, nil
	}

	validatedGroups := make([]*domain.EmployeePendingRequests, 0, len(grouped))
	for _, g := range grouped {
		maxAdv, err := s.config.AdvancePaymentRepo.SumMaxAdvByEmployeeMonth(ctx, g.EmployeeID, currentMonth)
		if err != nil {
			s.logger.Error("failed to get max advance limit, skipping employee", "employee_id", g.EmployeeID, "error", err)
			continue
		}
		// Use SumPendingAndCompletedByEmployeeMonth to get total active budget usage.
		// g.NetAmount is already included in this sum (it's the pending requests for this employee).
		usedTotal, err := s.config.AdvancePaymentRequestRepo.SumPendingAndCompletedByEmployeeMonth(ctx, g.EmployeeID, currentMonth)
		if err != nil {
			s.logger.Error("failed to get used total, skipping employee", "employee_id", g.EmployeeID, "error", err)
			continue
		}
		if maxAdv > 0 && usedTotal > maxAdv {
			return nil, domain.NewValidationError(
				fmt.Sprintf("Nhân viên %s vượt hạn mức: tối đa %d, đã sử dụng %d",
					g.EmployeeName, maxAdv, usedTotal))
		}
		validatedGroups = append(validatedGroups, g)
	}

	for _, g := range validatedGroups {
		if err := s.config.AdvancePaymentRequestRepo.BatchUpdateStatus(
			ctx,
			g.RequestIDs,
			domain.AdvancePaymentStatusApproved,
			"",
			nil,
		); err != nil {
			return nil, errors.Wrap(err, "failed to approve requests")
		}
		s.logger.Info("marked requests as approved",
			"employee_id", g.EmployeeID,
			"request_ids", g.RequestIDs,
		)
	}

	// Load all existing transaction codes once for uniqueness checking
	existingCodes, err2 := s.config.TransactionCodeRepo.GetAllCodes(ctx)
	if err2 != nil {
		_ = f.Close()
		return nil, errors.Wrap(err2, "failed to load existing transaction codes")
	}

	row := 3
	stt := 1
	var totalAmount int64
	var fileDataArray []dto.BulkTransferFileData
	var transactionCodesToCreate []*domain.TransactionCode

	for _, g := range validatedGroups {
		txCode := GenerateUniqueTransactionCode(existingCodes, true)

		codeData := domain.TransactionCodeData{
			FlexPay: &domain.FlexPayData{RequestIDs: g.RequestIDs},
		}
		dataBytes, _ := json.Marshal(codeData)

		transactionCodesToCreate = append(transactionCodesToCreate, &domain.TransactionCode{
			Code: txCode,
			Data: dataBytes,
		})
		existingCodes[txCode] = struct{}{} // Track newly generated code

		cellUpdates := []struct {
			cell  string
			value any
		}{
			{fmt.Sprintf("A%d", row), stt},
			{fmt.Sprintf("B%d", row), g.AccountNumber},
			{fmt.Sprintf("C%d", row), g.AccountName},
			{fmt.Sprintf("D%d", row), g.BankName},
			{fmt.Sprintf("E%d", row), g.NetAmount},
			{fmt.Sprintf("F%d", row), txCode},
		}

		for _, update := range cellUpdates {
			if err := f.SetCellValue(pkgConstants.MBank_SheetName, update.cell, update.value); err != nil {
				_ = f.Close()
				return nil, errors.Wrap(err, fmt.Sprintf("failed to set cell %s", update.cell))
			}
		}

		advPayReqIDs := make([]uint, len(g.RequestIDs))
		for i, id := range g.RequestIDs {
			advPayReqIDs[i] = uint(id)
		}

		fileData := dto.BulkTransferFileData{
			STT:             stt,
			EmployeeID:      uint(g.EmployeeID),
			ProjectID:       uint(g.ProjectID),
			AdvPayReqIDs:    advPayReqIDs,
			AccountNumber:   g.AccountNumber,
			AccountName:     g.AccountName,
			BankName:        g.BankName,
			Amount:          int64(g.NetAmount),
			TransactionCode: txCode,
		}
		fileDataArray = append(fileDataArray, fileData)
		totalAmount += int64(g.NetAmount)

		row++
		stt++
	}

	// Bulk insert all transaction codes after file is generated
	if err := s.config.TransactionCodeRepo.CreateBatch(ctx, transactionCodesToCreate); err != nil {
		_ = f.Close()
		return nil, errors.Wrap(err, "failed to create transaction codes")
	}

	filename, err := s.saveBulkTransferFile(ctx, fileDataArray, totalAmount, fromDate, toDate, currentMonth, userID)
	if err != nil {
		_ = f.Close()
		return nil, errors.Wrap(err, "failed to save bulk transfer file")
	}

	s.logger.Info("exported advance payment requests to MBank template",
		"filename", filename,
		"total_transactions", len(fileDataArray),
		"total_amount", totalAmount,
	)

	return f, nil
}

func (s *Service) clearExistingDataMBank(f *excelize.File) error {
	// Remove all data rows (row 3 onward) from the template.
	// We delete from the bottom up so that row indices don't shift as we go.
	// Row 1-2 are headers; data starts from row 3.
	rows, err := f.GetRows(pkgConstants.MBank_SheetName)
	if err != nil {
		return errors.Wrap(err, "failed to get rows for clearing")
	}

	lastRow := len(rows)
	for clearRow := lastRow; clearRow >= 3; clearRow-- {
		if err := f.RemoveRow(pkgConstants.MBank_SheetName, clearRow); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to remove row %d", clearRow))
		}
	}
	return nil
}
