package services

import (
	"api-server/internal/pkg/clock"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain/wallet"
)

var (
	reconJobsMu sync.RWMutex
	reconJobs   = map[string]*wallet.ReconciliationJob{}
)

type walletService struct {
	topupRepo   wallet.WalletTopupRepository
	paymentRepo wallet.WalletPaymentRepository
	registry    providerResolver
	mu          sync.Mutex
	logger      *slog.Logger
}

// providerResolver decouples walletService from the concrete disbursement
// registry while still resolving the active provider dynamically.
type providerResolver interface {
	ActiveProviderName(ctx context.Context) (string, error)
	GetProviderBalance(ctx context.Context) (int64, error)
}

func NewWalletService(
	topupRepo wallet.WalletTopupRepository,
	paymentRepo wallet.WalletPaymentRepository,
	registry providerResolver,
) wallet.WalletService {
	if registry == nil {
		panic("wallet: registry is required (fail-fast: no default provider)")
	}
	return &walletService{
		topupRepo:   topupRepo,
		paymentRepo: paymentRepo,
		registry:    registry,
		logger:      slog.Default(),
	}
}

func (s *walletService) GetBalance(ctx context.Context) (*wallet.WalletBalance, error) {
	topupTotal, err := s.topupRepo.Sum(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get topup total: %w", err)
	}

	completedPayments, err := s.paymentRepo.SumByStatuses(ctx, []string{"completed"})
	if err != nil {
		return nil, fmt.Errorf("failed to get completed payments total: %w", err)
	}

	pendingPayments, err := s.paymentRepo.SumByStatuses(ctx, []string{"pending", "authorised"})
	if err != nil {
		return nil, fmt.Errorf("failed to get pending payments total: %w", err)
	}

	// Unreconciled failed payments are in limbo — money might still have
	// gone through. Include them in pending_out and subtract from available.
	unreconciledFailed, err := s.paymentRepo.SumUnreconciledByStatuses(ctx, []string{"failed"})
	if err != nil {
		return nil, fmt.Errorf("failed to get unreconciled failed payments total: %w", err)
	}

	return &wallet.WalletBalance{
		Available:  topupTotal - completedPayments - unreconciledFailed,
		PendingIn:  0, // topups are confirmed on insert
		PendingOut: pendingPayments + unreconciledFailed,
		Currency:   "VND",
		AsOf:       clock.Now().Format(time.RFC3339),
	}, nil
}

func (s *walletService) SyncBalance(ctx context.Context, userID uint64) (*wallet.SyncBalanceResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	providerBalance, err := s.registry.GetProviderBalance(ctx)
	if err != nil {
		// Map infrastructure errors to domain-level sentinel errors
		// so the handler can return the correct HTTP status code.
		if errors.Is(err, disbursement.ErrNoActiveProvider) {
			return nil, wallet.ErrProviderNotConfigured
		}
		if errors.Is(err, disbursement.ErrBalanceNotSupported) {
			return nil, wallet.ErrProviderBalanceUnsupported
		}
		return nil, fmt.Errorf("lỗi lấy số dư từ nhà cung cấp: %w", err)
	}

	localBalance, err := s.GetBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("lỗi lấy số dư hệ thống: %w", err)
	}

	result := &wallet.SyncBalanceResult{
		ProviderBalance: providerBalance,
		LocalBalance:    localBalance.Available,
	}

	diff := providerBalance - localBalance.Available
	if diff != 0 {
		now := clock.Now()
		note := fmt.Sprintf("[Đồng bộ NCC] %d → %d", localBalance.Available, providerBalance)
		req := wallet.CreateWalletTopupRequest{
			Amount:     diff,
			BankRef:    fmt.Sprintf("SYNC-%s", now.Format("20060102-150405")),
			OccurredAt: now,
			Note:       &note,
		}
		if _, err := s.CreateTopup(ctx, req, userID); err != nil {
			return nil, fmt.Errorf("điều chỉnh số dư thất bại: %w", err)
		}
		result.Adjusted = true

		// Refresh local balance after adjustment
		if refreshed, err := s.GetBalance(ctx); err == nil {
			result.LocalBalance = refreshed.Available
		}
	}

	s.logger.Info("wallet balance sync completed",
		"provider_balance", providerBalance,
		"local_balance", result.LocalBalance,
		"adjusted", result.Adjusted,
	)

	return result, nil
}

func (s *walletService) GetTransactions(ctx context.Context, filter wallet.TransactionFilter) ([]*wallet.UnifiedTransaction, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 50
	}

	var all []*wallet.UnifiedTransaction

	// Fetch topups (unless filtered to payments only, or status excludes completed)
	if (filter.Type == "" || filter.Type == "topup") && (filter.Status == "" || filter.Status == "completed") {
		topupFilter := wallet.WalletTopupFilter{
			StartDate: parseTimePtr(filter.FromDate),
			EndDate:   parseTimePtr(filter.ToDate),
			Page:      1,
			PageSize:  1000, // fetch enough to merge
		}
		topups, _, err := s.topupRepo.List(ctx, topupFilter)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list topups: %w", err)
		}
		for _, t := range topups {
			note := ""
			if t.Note != nil {
				note = *t.Note
			}
			all = append(all, &wallet.UnifiedTransaction{
				ID:         t.ID,
				Type:       "topup",
				Amount:     t.Amount,
				Status:     "completed",
				OccurredAt: t.OccurredAt.Format(time.RFC3339),
				Reference:  t.BankRef,
				Note:       note,
			})
		}
	}

	// Fetch payments (unless filtered to topups only)
	if filter.Type == "" || filter.Type == "payment" {
		paymentFilter := wallet.WalletPaymentFilter{
			Status:    filter.Status,
			StartDate: parseTimePtr(filter.FromDate),
			EndDate:   parseTimePtr(filter.ToDate),
			Page:      1,
			PageSize:  1000,
		}
		payments, _, err := s.paymentRepo.List(ctx, paymentFilter)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list payments: %w", err)
		}
		for _, p := range payments {
			desc := ""
			if p.Description != nil {
				desc = *p.Description
			}
			all = append(all, &wallet.UnifiedTransaction{
				ID:           p.ID,
				Type:         "payment",
				Amount:       -p.RequestedAmount,
				Status:       p.Status,
				OccurredAt:   p.CreatedAt.Format(time.RFC3339),
				Reference:    p.RequestID,
				Counterparty: fmt.Sprintf("%s — %s %s", p.RecipientName, p.RecipientBank, p.RecipientAccountNo),
				Note:         desc,
			})
		}
	}

	// Sort by occurred_at descending
	sort.Slice(all, func(i, j int) bool {
		return all[i].OccurredAt > all[j].OccurredAt
	})

	// Paginate
	total := int64(len(all))
	start := (filter.Page - 1) * filter.PageSize
	if start >= len(all) {
		return []*wallet.UnifiedTransaction{}, total, nil
	}
	end := start + filter.PageSize
	if end > len(all) {
		end = len(all)
	}

	return all[start:end], total, nil
}

func (s *walletService) CreateTopup(ctx context.Context, req wallet.CreateWalletTopupRequest, userID uint64) (*wallet.WalletTopup, error) {
	// Check idempotency
	existing, err := s.topupRepo.GetByBankRef(ctx, req.BankRef)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("duplicate bank_ref: topup with bank_ref %q already exists (id=%d)", req.BankRef, existing.ID)
	}

	topup := &wallet.WalletTopup{
		Amount:     req.Amount,
		BankRef:    req.BankRef,
		OccurredAt: req.OccurredAt,
		Note:       req.Note,
		CreatedBy:  userID,
	}

	if err := s.topupRepo.Create(ctx, topup); err != nil {
		return nil, fmt.Errorf("failed to create wallet topup: %w", err)
	}

	return topup, nil
}

func (s *walletService) GetTopupByID(ctx context.Context, id uint64) (*wallet.WalletTopup, error) {
	return s.topupRepo.GetByID(ctx, id)
}

func (s *walletService) ListTopups(ctx context.Context, filter wallet.WalletTopupFilter) ([]*wallet.WalletTopup, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	return s.topupRepo.List(ctx, filter)
}

func (s *walletService) GetPaymentByID(ctx context.Context, id uint64) (*wallet.WalletPayment, error) {
	return s.paymentRepo.GetByID(ctx, id)
}

func (s *walletService) GetPaymentByTxnID(ctx context.Context, txnID string) (*wallet.WalletPayment, error) {
	return s.paymentRepo.GetByTxnID(ctx, txnID)
}

func (s *walletService) GetPaymentByRequestID(ctx context.Context, requestID string) (*wallet.WalletPayment, error) {
	return s.paymentRepo.GetByRequestID(ctx, requestID)
}

func (s *walletService) ListPayments(ctx context.Context, filter wallet.WalletPaymentFilter) ([]*wallet.WalletPayment, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	return s.paymentRepo.List(ctx, filter)
}

func (s *walletService) ResolvePayment(ctx context.Context, id uint64, action string, reason string, adminUserID uint64) (*wallet.WalletPayment, error) {
	payment, err := s.paymentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	var newStatus string
	switch action {
	case "complete":
		newStatus = "completed"
	case "fail":
		newStatus = "failed"
	default:
		return nil, fmt.Errorf("invalid action: %s (expected complete or fail)", action)
	}

	now := clock.Now()
	var settledAt, reconciledAt *time.Time
	if newStatus == "completed" || newStatus == "failed" {
		settledAt = &now
		reconciledAt = &now
	}

	s.logger.Info("resolving wallet payment",
		"payment_id", id,
		"old_status", payment.Status,
		"new_status", newStatus,
		"reason", reason,
		"admin_user_id", adminUserID)

	if err := s.paymentRepo.UpdateStatus(ctx, id, newStatus, settledAt, reconciledAt); err != nil {
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

	payment.Status = newStatus
	payment.SettledAt = settledAt
	return payment, nil
}

func (s *walletService) UploadReconciliation(ctx context.Context, csvData []byte, userID uint64) (jobID string, err error) {
	csvData = bytes.TrimPrefix(csvData, []byte{0xEF, 0xBB, 0xBF})

	reader := csv.NewReader(bytes.NewReader(csvData))
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		return "", fmt.Errorf("failed to read CSV header: %w", err)
	}

	colMap := buildColumnMap(header)

	matched := 0
	unmatched := 0
	totalRows := 0
	var breakRows [][]string

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		totalRows++

		invoiceNo := strings.TrimPrefix(getCol(row, colMap, "Mã giao dịch"), "'")
		statusStr := getCol(row, colMap, "Trạng thái")
		feeStr := getCol(row, colMap, "Phí")
		descStr := getCol(row, colMap, "Nội dung chi hộ")

		if invoiceNo == "" {
			breakRows = append(breakRows, row)
			unmatched++
			continue
		}

		providerName, err := s.registry.ActiveProviderName(ctx)
		if err != nil {
			breakRows = append(breakRows, row)
			unmatched++
			s.logger.Warn("reconciliation: no active provider", "error", err)
			continue
		}
		payment, err := s.paymentRepo.GetByProviderInvoiceNo(ctx, providerName, invoiceNo)
		if err != nil || payment == nil {
			breakRows = append(breakRows, row)
			unmatched++
			continue
		}

		newStatus := "completed"
		if statusStr != "" && statusStr != "Thành công" {
			newStatus = "failed"
		}

		if payment.Status != newStatus {
			payment.Status = newStatus
			now := clock.Now()
			payment.SettledAt = &now

			if fee, err := strconv.ParseInt(feeStr, 10, 64); err == nil {
				payment.Fee = fee
			}

			if descStr != "" {
				payment.Description = &descStr
			}

			if err := s.paymentRepo.Update(ctx, payment); err != nil {
				s.logger.Error("failed to update payment from reconciliation", "invoice_no", invoiceNo, "error", err)
				continue
			}
		}

		matched++
	}

	s.logger.Info("reconciliation completed",
		"total_rows", totalRows,
		"matched", matched,
		"unmatched", unmatched,
		"user_id", userID,
	)

	jobID = fmt.Sprintf("recon-%d", clock.Now().UnixMilli())
	reconJobsMu.Lock()
	reconJobs[jobID] = &wallet.ReconciliationJob{
		ID:          jobID,
		Status:      "completed",
		TotalRows:   totalRows,
		Matched:     matched,
		Unmatched:   unmatched,
		Headers:     header,
		RawRows:     breakRows,
		CompletedAt: clock.Now().Format(time.RFC3339),
	}
	reconJobsMu.Unlock()

	return jobID, nil
}

func (s *walletService) GetReconciliationJobStatus(ctx context.Context, jobID string) (*wallet.ReconciliationJob, error) {
	reconJobsMu.RLock()
	defer reconJobsMu.RUnlock()
	if job, ok := reconJobs[jobID]; ok {
		return job, nil
	}
	return &wallet.ReconciliationJob{
		ID:     jobID,
		Status: "completed",
	}, nil
}

func (s *walletService) ExportReconciliationReport(ctx context.Context, month string) ([]byte, error) {
	// Parse month (format: "2026-05")
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, fmt.Errorf("invalid month format, expected YYYY-MM: %w", err)
	}

	// Reconciliation cycle: 16th of previous month to 15th of current month
	startDate := time.Date(t.Year(), t.Month(), 16, 0, 0, 0, 0, t.Location()).AddDate(0, -1, 0)
	endDate := time.Date(t.Year(), t.Month(), 15, 23, 59, 59, 0, t.Location())

	filter := wallet.WalletPaymentFilter{
		Status:    "completed",
		StartDate: &startDate,
		EndDate:   &endDate,
		Page:      1,
		PageSize:  10000,
	}

	payments, _, err := s.paymentRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments for export: %w", err)
	}

	// TODO: generate Excel using excelize library
	// For now, return CSV as placeholder
	var buf bytes.Buffer
	buf.WriteString("Mã giao dịch,Tên khách hàng,Số tài khoản,Ngân hàng,Số tiền yêu cầu chuyển,Phí chuyển,Trạng thái,Thời gian giao dịch\n")

	var totalAmount, totalFee int64
	for _, p := range payments {
		settledAt := ""
		if p.SettledAt != nil {
			settledAt = p.SettledAt.Format("2006-01-02 15:04:05")
		}
		invoiceNo := ""
		if p.InvoiceNo != nil {
			invoiceNo = *p.InvoiceNo
		}
		fmt.Fprintf(&buf, "%s,%s,%s,%s,%d,%d,Thành công,%s\n",
			invoiceNo, p.RecipientName, p.RecipientAccountNo,
			p.RecipientBank, p.RequestedAmount, p.Fee, settledAt)
		totalAmount += p.RequestedAmount
		totalFee += p.Fee
	}

	fmt.Fprintf(&buf, ",,,,,%d,%d,,\n", totalAmount, totalFee)

	return buf.Bytes(), nil
}

func buildColumnMap(header []string) map[string]int {
	m := make(map[string]int)
	for i, h := range header {
		m[h] = i
	}
	return m
}

func getCol(row []string, colMap map[string]int, header string) string {
	idx, ok := colMap[header]
	if !ok || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func parseTimePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}
