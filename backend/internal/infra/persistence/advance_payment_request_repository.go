package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AdvancePaymentRequestRepository struct {
	*BaseRepository
	filterBuilder *common.FilterBuilder
	errorHandler  *common.RepoErrorHandler
}

func NewAdvancePaymentRequestRepository(db *Database) domain.AdvancePaymentRequestRepository {
	return &AdvancePaymentRequestRepository{
		BaseRepository: NewBaseRepository(db),
		filterBuilder:  common.NewFilterBuilder(db.DB),
		errorHandler:   common.NewRepoErrorHandler(),
	}
}

func (r *AdvancePaymentRequestRepository) Create(ctx context.Context, req *domain.AdvancePaymentRequest) error {
	return r.DB.WithContext(ctx).Create(req).Error
}

func (r *AdvancePaymentRequestRepository) GetByID(ctx context.Context, id uint64) (*domain.AdvancePaymentRequest, error) {
	var req domain.AdvancePaymentRequest
	err := r.DB.WithContext(ctx).
		Preload("AdvancePayment").
		Preload("Project").
		Preload("Employee").
		First(&req, id).Error

	if err != nil {
		return nil, r.errorHandler.HandleGetError(err, "advance_payment_request", id)
	}

	return &req, nil
}

func (r *AdvancePaymentRequestRepository) GetByEmployee(ctx context.Context, employeeID uint64, limit, offset int, fromDate, toDate *time.Time, forMonth *string) ([]*domain.AdvancePaymentRequest, int64, error) {
	var requests []*domain.AdvancePaymentRequest
	var total int64

	query := r.DB.WithContext(ctx).Model(&domain.AdvancePaymentRequest{}).
		Where("advance_payment_requests.employee_id = ?", employeeID)

	// Salary-period scope (preferred for employee history): group requests by the
	// for_month they are charged to, not the calendar day they were submitted. A
	// request created early in a month can belong to the previous salary period
	// (days 1–8 charge to the previous period), so created_at bounds would
	// misplace it. for_month lives on the parent advance_payments row, so we join
	// through adv_pay_id (every request has one, including PENDING).
	if forMonth != nil {
		query = query.
			Joins("JOIN advance_payments ap ON advance_payment_requests.adv_pay_id = ap.id").
			Where("ap.for_month = ?", *forMonth)
	}

	// Optional date-range fallback (inclusive created_at bounds), kept for callers
	// that still scope by submission date.
	if fromDate != nil {
		query = query.Where("advance_payment_requests.created_at >= ?", *fromDate)
	}
	if toDate != nil {
		query = query.Where("advance_payment_requests.created_at <= ?", *toDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Preload("Project").
		Order("advance_payment_requests.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&requests).Error

	if err != nil {
		return nil, 0, r.errorHandler.HandleListError(err, "advance_payment_request")
	}

	return requests, total, nil
}

func (r *AdvancePaymentRequestRepository) CountPending(ctx context.Context) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).Model(&domain.AdvancePaymentRequest{}).
		Where("status = ?", domain.AdvancePaymentStatusPending).
		Count(&count).Error
	if err != nil {
		return 0, r.errorHandler.HandleGetError(err, "advance_payment_request", nil)
	}
	return count, nil
}

func (r *AdvancePaymentRequestRepository) GetPendingSummary(ctx context.Context, forMonth string) (int64, int64, uint64, error) {
	type summaryResult struct {
		RequestCount  int64
		EmployeeCount int64
		TotalAmount   uint64
	}
	var result summaryResult
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Select("COUNT(*) as request_count, COUNT(DISTINCT employee_id) as employee_count, COALESCE(SUM(request_amount), 0) as total_amount").
		Joins("JOIN advance_payments ap ON advance_payment_requests.adv_pay_id = ap.id").
		Where("advance_payment_requests.status IN (?, ?)", domain.AdvancePaymentStatusPending, domain.AdvancePaymentStatusApproved).
		Where("ap.for_month = ?", forMonth).
		Scan(&result).Error
	if err != nil {
		return 0, 0, 0, r.errorHandler.HandleGetError(err, "advance_payment_request", nil)
	}
	return result.RequestCount, result.EmployeeCount, result.TotalAmount, nil
}

func (r *AdvancePaymentRequestRepository) CountCompletedProjectsByMonth(ctx context.Context, forMonth string) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Select("COUNT(DISTINCT project_id)").
		Joins("JOIN advance_payments ap ON advance_payment_requests.adv_pay_id = ap.id").
		Where("advance_payment_requests.status = ?", domain.AdvancePaymentStatusCompleted).
		Where("ap.for_month = ?", forMonth).
		Scan(&count).Error
	if err != nil {
		return 0, r.errorHandler.HandleGetError(err, "advance_payment_request", nil)
	}
	return count, nil
}

func (r *AdvancePaymentRequestRepository) GetPendingByDateRange(ctx context.Context, fromDate, toDate time.Time) ([]*domain.AdvancePaymentRequest, error) {
	var requests []*domain.AdvancePaymentRequest

	err := r.DB.WithContext(ctx).
		Where("status = ? AND created_at >= ? AND created_at <= ?",
			domain.AdvancePaymentStatusPending, fromDate, toDate).
		Preload("AdvancePayment").
		Preload("Project").
		Preload("Employee").
		Preload("Employee.Bank").
		Order("created_at ASC").
		Find(&requests).Error

	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "advance_payment_request")
	}

	return requests, nil
}

func (r *AdvancePaymentRequestRepository) GetPendingGroupedByEmployee(ctx context.Context, forMonth string) ([]*domain.EmployeePendingRequests, error) {
	// Use intermediate struct to capture raw SQL results
	type rawResult struct {
		EmployeeID    uint64 `gorm:"column:employee_id"`
		EmployeeName  string `gorm:"column:employee_name"`
		EmployeeCCCD  string `gorm:"column:employee_cccd"`
		ProjectID     uint64 `gorm:"column:project_id"`
		ProjectCode   string `gorm:"column:project_code"`
		AccountNumber string `gorm:"column:account_number"`
		AccountName   string `gorm:"column:account_name"`
		BankName      string `gorm:"column:bank_name"`
		TotalAmount   uint64 `gorm:"column:total_amount"`
		TotalFee      uint64 `gorm:"column:total_fee"`
		NetAmount     uint64 `gorm:"column:net_amount"`
		RequestIDsRaw string `gorm:"column:request_ids"`
	}

	var rawResults []*rawResult

	// Include both PENDING and APPROVED requests in the current pay cycle (for_month).
	// Exclude requests that already have a wallet_payment (being processed by 9Pay).
	err := r.DB.WithContext(ctx).
		Raw(`
			SELECT
				e.id as employee_id,
				e.fullname as employee_name,
				e.cccd as employee_cccd,
				p.id as project_id,
				p.code as project_code,
				e.bank_account_number as account_number,
				e.bank_account_name as account_name,
				b.branch_name as bank_name,
				SUM(apr.request_amount) as total_amount,
				SUM(apr.fee) as total_fee,
				SUM(apr.net_amount) as net_amount,
				GROUP_CONCAT(apr.id) as request_ids
			FROM advance_payment_requests apr
			JOIN advance_payments ap ON apr.adv_pay_id = ap.id
			JOIN employees e ON apr.employee_id = e.id
			JOIN projects p ON apr.project_id = p.id
			LEFT JOIN banks b ON e.bank_id = b.id
			WHERE apr.status IN (?, ?)
				AND ap.for_month = ?
				AND COALESCE((
						SELECT wp.status FROM wallet_payments wp
						WHERE wp.entity_id = apr.id
						ORDER BY wp.created_at DESC
						LIMIT 1
					), '') IN ('', 'failed', 'reversed')
			GROUP BY e.id, p.id
			ORDER BY e.fullname ASC
		`, domain.AdvancePaymentStatusPending, domain.AdvancePaymentStatusApproved, forMonth).
		Scan(&rawResults).Error

	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "advance_payment_request")
	}

	// Convert raw results to domain struct, parsing the comma-separated request IDs
	results := make([]*domain.EmployeePendingRequests, len(rawResults))
	for i, raw := range rawResults {
		results[i] = &domain.EmployeePendingRequests{
			EmployeeID:    raw.EmployeeID,
			EmployeeName:  raw.EmployeeName,
			EmployeeCCCD:  raw.EmployeeCCCD,
			ProjectID:     raw.ProjectID,
			ProjectCode:   raw.ProjectCode,
			AccountNumber: raw.AccountNumber,
			AccountName:   raw.AccountName,
			BankName:      raw.BankName,
			TotalAmount:   raw.TotalAmount,
			TotalFee:      raw.TotalFee,
			NetAmount:     raw.NetAmount,
			RequestIDs:    parseRequestIDs(raw.RequestIDsRaw),
		}
	}

	return results, nil
}

// parseRequestIDs converts a comma-separated string of IDs to []uint64
func parseRequestIDs(s string) []uint64 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	ids := make([]uint64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func (r *AdvancePaymentRequestRepository) SumCompletedByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (uint64, error) {
	var result struct {
		Total uint64
	}
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Joins("JOIN advance_payments ap ON advance_payment_requests.adv_pay_id = ap.id").
		Select("COALESCE(SUM(advance_payment_requests.request_amount), 0) as total").
		Where("advance_payment_requests.employee_id = ? AND ap.for_month = ? AND advance_payment_requests.status = ?",
			employeeID, forMonth, domain.AdvancePaymentStatusCompleted).
		Scan(&result).Error

	return result.Total, err
}

func (r *AdvancePaymentRequestRepository) SumPendingAndCompletedByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (uint64, error) {
	var result struct {
		Total uint64
	}
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Joins("JOIN advance_payments ap ON advance_payment_requests.adv_pay_id = ap.id").
		Select("COALESCE(SUM(advance_payment_requests.request_amount), 0) as total").
		Where("advance_payment_requests.employee_id = ? AND ap.for_month = ? AND advance_payment_requests.status IN ?",
			employeeID, forMonth, []domain.AdvancePaymentRequestStatus{
				domain.AdvancePaymentStatusPending,
				domain.AdvancePaymentStatusApproved,
				domain.AdvancePaymentStatusCompleted,
			}).
		Scan(&result).Error

	return result.Total, err
}

func (r *AdvancePaymentRequestRepository) SumPendingByEmployeeMonth(ctx context.Context, employeeID uint64, forMonth string) (uint64, error) {
	var result struct {
		Total uint64
	}
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Joins("JOIN advance_payments ap ON advance_payment_requests.adv_pay_id = ap.id").
		Select("COALESCE(SUM(request_amount), 0) as total").
		Where("advance_payment_requests.employee_id = ? AND ap.for_month = ? AND advance_payment_requests.status IN ?",
			employeeID, forMonth, []domain.AdvancePaymentRequestStatus{
				domain.AdvancePaymentStatusPending,
				domain.AdvancePaymentStatusApproved,
			}).
		Scan(&result).Error

	return result.Total, err
}

func (r *AdvancePaymentRequestRepository) UpdateStatus(ctx context.Context, id uint64, status domain.AdvancePaymentRequestStatus, paymentRef string, paidAt *time.Time) error {
	updates := map[string]any{
		"status":            status,
		"payment_reference": paymentRef,
	}
	if paidAt != nil {
		updates["paid_at"] = paidAt
	}

	return r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *AdvancePaymentRequestRepository) BatchUpdateStatus(ctx context.Context, ids []uint64, status domain.AdvancePaymentRequestStatus, paymentRef string, paidAt *time.Time) error {
	if len(ids) == 0 {
		return nil
	}

	updates := map[string]any{
		"status":            status,
		"payment_reference": paymentRef,
	}
	if paidAt != nil {
		updates["paid_at"] = paidAt
	}

	return r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Where("id IN ?", ids).
		Updates(updates).Error
}

func (r *AdvancePaymentRequestRepository) Cancel(ctx context.Context, id uint64) error {
	return r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Where("id = ? AND status IN ?", id, []domain.AdvancePaymentRequestStatus{
			domain.AdvancePaymentStatusPending,
			domain.AdvancePaymentStatusApproved,
		}).
		Update("status", domain.AdvancePaymentStatusCancelled).Error
}

func (r *AdvancePaymentRequestRepository) List(ctx context.Context, filters domain.AdvancePaymentRequestFilters) ([]*domain.AdvancePaymentRequest, int64, error) {
	var requests []*domain.AdvancePaymentRequest
	var total int64

	query := r.DB.WithContext(ctx).Model(&domain.AdvancePaymentRequest{}).
		Preload("AdvancePayment").
		Preload("Project").
		Preload("Employee")

	// Apply filters
	query = r.applyFilters(query, filters)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting and pagination
	query = r.filterBuilder.ApplySorting(query, filters.SortBy, filters.SortOrder, "created_at")
	query = r.filterBuilder.ApplyPagination(query, filters.Limit, filters.Offset)

	err := query.Find(&requests).Error
	if err != nil {
		return nil, 0, r.errorHandler.HandleListError(err, "advance_payment_request")
	}

	return requests, total, nil
}

func (r *AdvancePaymentRequestRepository) GetByIDs(ctx context.Context, ids []uint64) ([]*domain.AdvancePaymentRequest, error) {
	if len(ids) == 0 {
		return []*domain.AdvancePaymentRequest{}, nil
	}

	// Chunk at DefaultChunkSize to stay under MySQL's packet limit at >1000 IDs.
	var requests []*domain.AdvancePaymentRequest
	for i := 0; i < len(ids); i += common.DefaultChunkSize {
		end := i + common.DefaultChunkSize
		if end > len(ids) {
			end = len(ids)
		}
		var batchReqs []*domain.AdvancePaymentRequest
		if err := r.getDB(ctx).
			Where("id IN ?", ids[i:end]).
			Preload("AdvancePayment").
			Preload("Project").
			Preload("Employee").
			Find(&batchReqs).Error; err != nil {
			return nil, r.errorHandler.HandleListError(err, "advance_payment_request")
		}
		requests = append(requests, batchReqs...)
	}

	return requests, nil
}

// GetByIDsLean is GetByIDs without preloading relations, for callers that only
// need scalar fields (e.g. settlement aggregation in the wallet worker).
func (r *AdvancePaymentRequestRepository) GetByIDsLean(ctx context.Context, ids []uint64) ([]*domain.AdvancePaymentRequest, error) {
	if len(ids) == 0 {
		return []*domain.AdvancePaymentRequest{}, nil
	}

	var requests []*domain.AdvancePaymentRequest
	for i := 0; i < len(ids); i += common.DefaultChunkSize {
		end := i + common.DefaultChunkSize
		if end > len(ids) {
			end = len(ids)
		}
		var batchReqs []*domain.AdvancePaymentRequest
		if err := r.getDB(ctx).
			Where("id IN ?", ids[i:end]).
			Find(&batchReqs).Error; err != nil {
			return nil, r.errorHandler.HandleListError(err, "advance_payment_request")
		}
		requests = append(requests, batchReqs...)
	}

	return requests, nil
}

// getDB returns the tx-scoped DB when a transaction is propagated on the context
// (so this repo participates in domain.TransactionManager transactions), else
// the base DB. Mirrors transactionRepository.getDB / settlement_repository.
func (r *AdvancePaymentRequestRepository) getDB(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}

func (r *AdvancePaymentRequestRepository) applyFilters(query *gorm.DB, filters domain.AdvancePaymentRequestFilters) *gorm.DB {
	if filters.Status != nil {
		query = query.Where("advance_payment_requests.status = ?", *filters.Status)
	}
	if filters.ProjectID != nil {
		query = query.Where("advance_payment_requests.project_id = ?", *filters.ProjectID)
	}
	if filters.EmployeeID != nil {
		query = query.Where("advance_payment_requests.employee_id = ?", *filters.EmployeeID)
	}
	if filters.FromDate != nil {
		query = query.Where("advance_payment_requests.created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("advance_payment_requests.created_at <= ?", *filters.ToDate)
	}
	if filters.ForMonth != nil {
		query = query.Joins("JOIN advance_payments ap ON advance_payment_requests.adv_pay_id = ap.id").
			Where("ap.for_month = ?", *filters.ForMonth)
	}
	if filters.Search != "" {
		searchTerm := "%" + filters.Search + "%"
		query = query.Joins("JOIN employees e ON advance_payment_requests.employee_id = e.id").
			Where("e.fullname LIKE ? OR e.cccd LIKE ?", searchTerm, searchTerm)
	}

	return query
}

func (r *AdvancePaymentRequestRepository) GetStatsSummary(ctx context.Context, fromDate, toDate time.Time, forMonth string) (*domain.AdvancePaymentStatsSummary, error) {
	type statsResult struct {
		TotalRequests         int64
		TotalPending          int64
		TotalApproved         int64
		TotalCancelled        int64
		TotalFailed           int64
		TotalPaid             int64
		TotalAmount           uint64
		TotalPaidAmount       uint64
		TotalPendingAmount    uint64
		TotalFailedAmount     uint64
		TotalCancelledAmount  uint64
		TotalFee              uint64
		TotalFeeEarned        uint64
		TotalProviderFee      uint64
		AvgProcessingTimeSecs float64
		CompletedUnder30s     int64
	}

	var result statsResult

	query := r.DB.WithContext(ctx)

	selectClause := `
			SELECT
				COUNT(advance_payment_requests.id) as total_requests,
				SUM(CASE WHEN advance_payment_requests.status IN ('PENDING', 'APPROVED') THEN 1 ELSE 0 END) as total_pending,
				SUM(CASE WHEN advance_payment_requests.status = 'APPROVED' THEN 1 ELSE 0 END) as total_approved,
				SUM(CASE WHEN advance_payment_requests.status = 'CANCELLED' THEN 1 ELSE 0 END) as total_cancelled,
				SUM(CASE WHEN advance_payment_requests.status = 'FAILED' THEN 1 ELSE 0 END) as total_failed,
				SUM(CASE WHEN advance_payment_requests.status = 'COMPLETED' THEN 1 ELSE 0 END) as total_paid,
				SUM(CASE WHEN advance_payment_requests.status = 'COMPLETED' THEN advance_payment_requests.request_amount ELSE 0 END) as total_amount,
				SUM(CASE WHEN advance_payment_requests.status = 'COMPLETED' THEN advance_payment_requests.request_amount - advance_payment_requests.fee ELSE 0 END) as total_paid_amount,
				SUM(CASE WHEN advance_payment_requests.status IN ('PENDING', 'APPROVED') THEN advance_payment_requests.request_amount ELSE 0 END) as total_pending_amount,
				SUM(CASE WHEN advance_payment_requests.status = 'FAILED' THEN advance_payment_requests.request_amount ELSE 0 END) as total_failed_amount,
				SUM(CASE WHEN advance_payment_requests.status = 'CANCELLED' THEN advance_payment_requests.request_amount ELSE 0 END) as total_cancelled_amount,
				SUM(advance_payment_requests.fee) as total_fee,
				SUM(CASE WHEN advance_payment_requests.status = 'COMPLETED' THEN advance_payment_requests.fee - COALESCE(advance_payment_requests.provider_fee, 0) ELSE 0 END) as total_fee_earned,
				SUM(CASE WHEN advance_payment_requests.status = 'COMPLETED' THEN COALESCE(advance_payment_requests.provider_fee, 0) ELSE 0 END) as total_provider_fee,
					COALESCE(AVG(CASE WHEN advance_payment_requests.status = 'COMPLETED' AND advance_payment_requests.paid_at IS NOT NULL THEN TIMESTAMPDIFF(SECOND, advance_payment_requests.created_at, advance_payment_requests.paid_at) END), 0) as avg_processing_time_secs,
					SUM(CASE WHEN advance_payment_requests.status = 'COMPLETED' AND advance_payment_requests.paid_at IS NOT NULL AND TIMESTAMPDIFF(SECOND, advance_payment_requests.created_at, advance_payment_requests.paid_at) <= 30 THEN 1 ELSE 0 END) as completed_under30s
				FROM advance_payment_requests
	`

	if forMonth != "" {
		selectClause += ` JOIN advance_payments ap ON advance_payment_requests.adv_pay_id = ap.id WHERE ap.for_month = ?`
		query = query.Raw(selectClause, forMonth)
	} else if !fromDate.IsZero() && !toDate.IsZero() {
		selectClause += ` WHERE advance_payment_requests.created_at >= ? AND advance_payment_requests.created_at <= ?`
		query = query.Raw(selectClause, fromDate, toDate)
	} else {
		query = query.Raw(selectClause)
	}

	err := query.Scan(&result).Error

	if err != nil {
		return nil, r.errorHandler.HandleListError(err, "advance_payment_request")
	}

	return &domain.AdvancePaymentStatsSummary{
		TotalRequests:         result.TotalRequests,
		TotalPending:          result.TotalPending,
		TotalApproved:         result.TotalApproved,
		TotalCancelled:        result.TotalCancelled,
		TotalFailed:           result.TotalFailed,
		TotalPaid:             result.TotalPaid,
		TotalAmount:           result.TotalAmount,
		TotalPaidAmount:       result.TotalPaidAmount,
		TotalPendingAmount:    result.TotalPendingAmount,
		TotalFailedAmount:     result.TotalFailedAmount,
		TotalCancelledAmount:  result.TotalCancelledAmount,
		TotalFee:              result.TotalFee,
		TotalFeeEarned:        result.TotalFeeEarned,
		TotalProviderFee:      result.TotalProviderFee,
		AvgProcessingTimeSecs: result.AvgProcessingTimeSecs,
		CompletedUnder30s:     result.CompletedUnder30s,
	}, nil
}

// MarkReceivableSettled marks the specified requests as receivable settled
func (r *AdvancePaymentRequestRepository) MarkReceivableSettled(ctx context.Context, ids []uint64, settledAt time.Time) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	result := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Where("id IN ?", ids).
		Where("receivable_settled_at IS NULL"). // Only update if not already settled
		Update("receivable_settled_at", settledAt)

	if result.Error != nil {
		return 0, r.errorHandler.HandleUpdateError(result.Error, "advance_payment_request", ids)
	}

	return result.RowsAffected, nil
}

// UpdateSettlementTransactionID links requests to their settlement transaction.
// Routed through getDB so it commits with the surrounding settlement tx.
func (r *AdvancePaymentRequestRepository) UpdateSettlementTransactionID(ctx context.Context, ids []uint64, transactionID uint) error {
	if len(ids) == 0 {
		return nil
	}

	return r.getDB(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Where("id IN ?", ids).
		Update("settlement_transaction_id", transactionID).Error
}

// ClaimPendingForDisbursement atomically locks PENDING requests using
// FOR UPDATE SKIP LOCKED and sets them to APPROVED within a single
// transaction. Returns the claimed requests with Employee and Bank preloaded.
func (r *AdvancePaymentRequestRepository) ClaimPendingForDisbursement(ctx context.Context, limit int) ([]*domain.AdvancePaymentRequest, error) {
	var requests []*domain.AdvancePaymentRequest

	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock PENDING rows, skipping any already locked by admin export
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", domain.AdvancePaymentStatusPending).
			Order("created_at ASC").
			Limit(limit).
			Preload("AdvancePayment").
			Preload("Employee").
			Preload("Employee.Bank").
			Preload("Project").
			Find(&requests).Error; err != nil {
			return fmt.Errorf("claim pending: %w", err)
		}

		if len(requests) == 0 {
			return nil
		}

		ids := make([]uint64, len(requests))
		for i, req := range requests {
			ids[i] = uint64(req.ID)
		}

		// Atomically approve within the same transaction
		now := clock.Now()
		if err := tx.Model(&domain.AdvancePaymentRequest{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"status":     domain.AdvancePaymentStatusApproved,
				"updated_at": now,
			}).Error; err != nil {
			return fmt.Errorf("claim pending: update to approved: %w", err)
		}

		// Update in-memory structs so caller has correct status
		for _, req := range requests {
			req.Status = domain.AdvancePaymentStatusApproved
			req.UpdatedAt = now
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return requests, nil
}

// GetOrphanedApproved returns APPROVED requests that have no corresponding
// wallet_payment (for the given provider) and were approved more than 5
// minutes ago. The time buffer prevents picking up requests currently being
// processed by an active worker. An empty provider string skips the provider
// filter (legacy fallback).
//
// A request is only considered "processed" if it has a wallet_payment NOT in
// a terminal-failure state (failed/reversed). This allows the poller to retry
// requests whose payment attempt failed (e.g. name_mismatch) — the failed row
// stays as an audit trail, but the request is re-enqueued with a fresh
// request_id so the execute worker creates a new wallet_payment attempt.
func (r *AdvancePaymentRequestRepository) GetOrphanedApproved(ctx context.Context, limit int, provider string) ([]*domain.AdvancePaymentRequest, error) {
	var requests []*domain.AdvancePaymentRequest

	// Use clock.Now() (app timezone, Asia/Ho_Chi_Minh) instead of MySQL's
	// NOW() (which may be UTC) so the 5-minute buffer is compared against the
	// same timezone that wrote updated_at. A MySQL/Go timezone mismatch would
	// otherwise make stale requests look fresh (or vice-versa).
	cutoff := clock.Now().Add(-5 * time.Minute)

	q := r.DB.WithContext(ctx).
		Where("status = ?", domain.AdvancePaymentStatusApproved).
		Where("updated_at < ?", cutoff)

	if provider != "" {
		q = q.Where("NOT EXISTS (SELECT 1 FROM wallet_payments wp WHERE wp.entity_id = advance_payment_requests.id AND wp.provider = ? AND wp.status NOT IN ('failed', 'reversed'))", provider)
	} else {
		q = q.Where("NOT EXISTS (SELECT 1 FROM wallet_payments wp WHERE wp.entity_id = advance_payment_requests.id AND wp.status NOT IN ('failed', 'reversed'))")
	}

	err := q.
		Order("created_at ASC").
		Limit(limit).
		Preload("Employee").
		Preload("Employee.Bank").
		Preload("Project").
		Find(&requests).Error

	if err != nil {
		return nil, fmt.Errorf("orphan recovery: %w", err)
	}

	return requests, nil
}

// GetTotalProviderFee returns the all-time sum of provider_fee for COMPLETED requests.
func (r *AdvancePaymentRequestRepository) GetTotalProviderFee(ctx context.Context) (uint64, error) {
	var total uint64
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Select("COALESCE(SUM(provider_fee), 0)").
		Where("status = ?", domain.AdvancePaymentStatusCompleted).
		Scan(&total).Error
	return total, err
}

// GetTotalFeeEarned returns the all-time sum of (fee - provider_fee) for COMPLETED requests.
func (r *AdvancePaymentRequestRepository) GetTotalFeeEarned(ctx context.Context) (uint64, error) {
	var total uint64
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Select("COALESCE(SUM(fee - COALESCE(provider_fee, 0)), 0)").
		Where("status = ?", domain.AdvancePaymentStatusCompleted).
		Scan(&total).Error
	return total, err
}

// ResetToPending atomically resets APPROVED requests back to PENDING.
// Used by the poller when wallet balance is insufficient to process claimed requests.
// Only resets requests that are still APPROVED (idempotent, safe for concurrent pollers).
func (r *AdvancePaymentRequestRepository) ResetToPending(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}

	return r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Where("id IN ? AND status = ?", ids, domain.AdvancePaymentStatusApproved).
		Updates(map[string]any{
			"status":     domain.AdvancePaymentStatusPending,
			"updated_at": clock.Now(),
		}).Error
}

// CountStuckPending returns the number of PENDING requests created before olderThan.
// A non-zero count signals a stalled disbursement worker.
// Scoped to check-in-enabled employees to match the self-checkin dashboard.
func (r *AdvancePaymentRequestRepository) CountStuckPending(ctx context.Context, olderThan time.Time) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Where("status = ? AND created_at < ?", domain.AdvancePaymentStatusPending, olderThan).
		Where(checkInEnabledScope("advance_payment_requests")).
		Count(&count).Error
	if err != nil {
		return 0, r.errorHandler.HandleGetError(err, "advance_payment_request", "stuck_pending")
	}
	return count, nil
}

// CountStuckPendingInWindow returns PENDING requests older than olderThan within [since, until).
// Scoped to check-in-enabled employees to match the self-checkin dashboard.
func (r *AdvancePaymentRequestRepository) CountStuckPendingInWindow(ctx context.Context, olderThan, since, until time.Time) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Where("status = ? AND created_at < ? AND created_at >= ? AND created_at < ?", domain.AdvancePaymentStatusPending, olderThan, since, until).
		Where(checkInEnabledScope("advance_payment_requests")).
		Count(&count).Error
	if err != nil {
		return 0, r.errorHandler.HandleGetError(err, "advance_payment_request", "stuck_pending_window")
	}
	return count, nil
}

// CountByStatusInWindow returns request counts grouped by status for rows created in [since, until).
// Only the statuses relevant to the health dashboard are populated; absent keys are zero.
func (r *AdvancePaymentRequestRepository) CountByStatusInWindow(ctx context.Context, since, until time.Time) (map[domain.AdvancePaymentRequestStatus]int64, error) {
	type row struct {
		Status string `gorm:"column:status"`
		Cnt    int64  `gorm:"column:cnt"`
	}
	var rows []row
	err := r.DB.WithContext(ctx).
		Model(&domain.AdvancePaymentRequest{}).
		Select("status, COUNT(*) as cnt").
		Where("created_at >= ? AND created_at < ?", since, until).
		Where(checkInEnabledScope("advance_payment_requests")).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, r.errorHandler.HandleGetError(err, "advance_payment_request", "by_status_window")
	}
	out := make(map[domain.AdvancePaymentRequestStatus]int64, len(rows))
	for _, r := range rows {
		out[domain.AdvancePaymentRequestStatus(r.Status)] = r.Cnt
	}
	return out, nil
}

// GetCohortByMonths returns cohort rows (for_month × cycle-day × status) for the
// given for_months, scoped to flexible-schedule project assignments via a
// project-correlated join on project_employees.
//
// cycle_day is 1-indexed from the period start (day 20 of for_month) and is derived
// from created_at in Asia/Ho_Chi_Minh: created_at is stored UTC on prod, so without
// CONVERT_TZ the day boundary shifts and rows land on the wrong cycle-day. The
// caller clamps rows to [1, maxCycleDay(for_month)]. total_amount is intentionally
// request_amount - fee from actual employee requests; never use max_adv_amount
// here because that is quota, not demand.
func (r *AdvancePaymentRequestRepository) GetCohortByMonths(ctx context.Context, forMonths []string) ([]domain.CohortRow, error) {
	if len(forMonths) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(forMonths))
	args := make([]any, len(forMonths))
	for i, m := range forMonths {
		placeholders[i] = "?"
		args[i] = m
	}
	query := fmt.Sprintf(`
		SELECT
		  ap.for_month                                                       AS for_month,
		  DATEDIFF(DATE(CONVERT_TZ(apr.created_at, '+00:00', '+07:00')),
		           DATE(CONCAT(ap.for_month, '-20'))) + 1                     AS cycle_day,
		  apr.status                                                         AS status,
		  COUNT(*)                                                           AS request_count,
		  COALESCE(SUM(apr.request_amount - COALESCE(apr.fee, 0)), 0)        AS total_amount
		FROM advance_payment_requests apr
		JOIN advance_payments ap ON apr.adv_pay_id = ap.id
		JOIN project_employees pe
		  ON pe.employee_id = apr.employee_id
		 AND pe.project_id   = apr.project_id
		 AND pe.deleted_at IS NULL
		 AND pe.payment_schedule = 'flexible'
		WHERE ap.for_month IN (%s)
		GROUP BY ap.for_month, cycle_day, apr.status
	`, strings.Join(placeholders, ","))

	var rows []domain.CohortRow
	if err := r.DB.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, r.errorHandler.HandleListError(err, "advance_payment_request_cohort")
	}
	return rows, nil
}

// CreateWithBudgetCheck atomically creates an advance payment request only if the
// employee's total active requests (PENDING + APPROVED + COMPLETED) + the new request
// amount do not exceed the max advance limit for the given month.
//
// It uses SELECT ... FOR UPDATE to lock the employee's advance_payments rows,
// preventing concurrent requests from reading a stale budget.
// Returns domain.ErrBudgetExceeded if the limit would be breached.
func (r *AdvancePaymentRequestRepository) CreateWithBudgetCheck(ctx context.Context, req *domain.AdvancePaymentRequest, employeeID uint64, forMonth string) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Lock the employee's advance_payment rows for this month (budget source)
		var maxAdv struct{ Total uint64 }
		if err := tx.
			Model(&domain.AdvancePayment{}).
			Select("COALESCE(SUM(max_adv_amount), 0) as total").
			Where("employee_id = ? AND for_month = ?", employeeID, forMonth).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Scan(&maxAdv).Error; err != nil {
			return fmt.Errorf("budget check: lock advance_payments: %w", err)
		}

		if maxAdv.Total == 0 {
			return domain.NewValidationError("không tìm thấy thông tin hạn mức ứng lương cho tháng này")
		}

		// 2. Sum all active (non-terminal) requests for this employee/month
		var usedTotal struct{ Total uint64 }
		if err := tx.
			Table("advance_payment_requests").
			Joins("JOIN advance_payments ap ON advance_payment_requests.adv_pay_id = ap.id").
			Select("COALESCE(SUM(advance_payment_requests.request_amount), 0) as total").
			Where("advance_payment_requests.employee_id = ? AND ap.for_month = ? AND advance_payment_requests.status IN ?",
				employeeID, forMonth,
				[]domain.AdvancePaymentRequestStatus{
					domain.AdvancePaymentStatusPending,
					domain.AdvancePaymentStatusApproved,
					domain.AdvancePaymentStatusCompleted,
				},
			).
			Scan(&usedTotal).Error; err != nil {
			return fmt.Errorf("budget check: sum active requests: %w", err)
		}

		// 3. Check budget
		if usedTotal.Total+req.RequestAmount > maxAdv.Total {
			return domain.NewValidationError(fmt.Sprintf(
				"vượt hạn mức ứng lương: tối đa %d, đã sử dụng %d, yêu cầu %d",
				maxAdv.Total, usedTotal.Total, req.RequestAmount,
			))
		}

		// 4. Insert the new request
		if err := tx.Create(req).Error; err != nil {
			return fmt.Errorf("budget check: create request: %w", err)
		}

		return nil
	})
}
