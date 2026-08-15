package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/excel"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/notification"
	"api-server/internal/app/services/settlement"
	"api-server/internal/pkg/clock"
)

type TransactionHandler struct {
	logger               *slog.Logger
	transactionService   *settlement.TransactionService
	bulkTransferFileRepo domain.BulkTransferFileRepository
	timesheetRepo        domain.TimesheetRepository
	notificationService  *notification.NotificationService
	eventBus             domain.EventBus
	clock                clock.Clock
}

func NewTransactionHandler(
	transactionService *settlement.TransactionService,
	bulkTransferFileRepo domain.BulkTransferFileRepository,
	timesheetRepo domain.TimesheetRepository,
	notificationService *notification.NotificationService,
	eventBus domain.EventBus,
	clk clock.Clock,
) *TransactionHandler {
	if clk == nil {
		clk = clock.New()
	}
	return &TransactionHandler{
		logger:               slog.Default(),
		transactionService:   transactionService,
		bulkTransferFileRepo: bulkTransferFileRepo,
		timesheetRepo:        timesheetRepo,
		notificationService:  notificationService,
		eventBus:             eventBus,
		clock:                clk,
	}
}

// CreateTransaction creates a new transaction with automatic double-entry
// @Summary Create transaction
// @Description Creates a user-facing transaction and automatically generates balanced ledger entries
// @Tags transactions
// @Accept json
// @Produce json
// @Param request body dto.CreateTransactionRequest true "Transaction data"
// @Success 201 {object} response.SuccessResponse{data=dto.TransactionWithLedgerResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /transactions [post]
func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	var req dto.CreateTransactionRequest

	// Log raw request body for troubleshooting
	h.logger.Info("Received transaction creation request")

	if !helpers.BindJSON(c, &req) {
		return
	}

	// Get user ID from context
	userIDInterface, exists := c.Get(constants.CtxUserID)
	if !exists {
		h.logger.Error("User ID not found in context")
		response.Forbidden(c, "Không tìm thấy thông tin người dùng")
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok || userID == 0 {
		h.logger.Error("Invalid user ID type in context")
		response.Forbidden(c, "Không tìm thấy thông tin người dùng")
		return
	}

	h.logger.Info("Creating transaction",
		"type", req.TransactionType,
		"amount", req.Amount,
		"party", req.Party,
		"status", req.Status,
		"userID", userID)

	// Create domain transaction
	txn := &domain.Transaction{
		Description:     req.Description,
		TransactionType: req.TransactionType,
		Amount:          req.Amount,
		Party:           req.Party,
		Status:          req.Status,
		AssetID:         req.AssetID,
		UserID:          req.UserID,
		CreatedBy:       userID,
	}

	// Create transaction with ledger entries
	createdTxn, ledgerEntries, err := h.transactionService.CreateTransaction(c.Request.Context(), txn)
	if err != nil {
		if domain.IsValidationError(err) {
			h.logger.Error("Transaction validation failed", "error", err, "userID", userID)
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error("Failed to create transaction", "error", err, "userID", userID)
		response.InternalServerError(c, "Không thể tạo giao dịch")
		return
	}

	h.logger.Info("Transaction created successfully",
		"transactionID", createdTxn.ID,
		"ledgerEntriesCount", len(ledgerEntries),
		"userID", userID)

	// Build response
	resp := dto.TransactionWithLedgerResponse{
		Transaction:   h.mapToTransactionResponse(createdTxn),
		LedgerEntries: h.mapToLedgerEntriesResponse(ledgerEntries),
	}

	response.SuccessCreated(c, resp, "Tạo giao dịch thành công")
}

// GetTransaction retrieves a single transaction by ID
// @Summary Get transaction
// @Description Get a transaction by ID with related data
// @Tags transactions
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 200 {object} response.SuccessResponse{data=dto.TransactionResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /transactions/{id} [get]
func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID giao dịch không hợp lệ")
	if !ok {
		return
	}

	txn, err := h.transactionService.GetTransaction(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			h.logger.Error("Transaction not found", "transactionID", id)
			response.NotFound(c, "Không tìm thấy giao dịch")
			return
		}
		h.logger.Error("Failed to get transaction", "transactionID", id, "error", err)
		response.InternalServerError(c, "Không thể lấy giao dịch")
		return
	}

	h.logger.Info("Retrieved transaction successfully", "transactionID", id)
	response.Success(c, h.mapToTransactionResponse(txn), "Lấy giao dịch thành công")
}

// UpdateTransactionEvidence updates only the evidence fields (url, asset_id) of a transaction
// @Summary Update transaction evidence
// @Description Update only evidence fields (url, asset_id). Other fields are immutable.
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Param request body dto.UpdateTransactionEvidenceRequest true "Evidence update"
// @Success 200 {object} response.SuccessResponse{data=dto.TransactionResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /transactions/{id} [put]
func (h *TransactionHandler) UpdateTransactionEvidence(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID giao dịch không hợp lệ")
	if !ok {
		return
	}

	var req dto.UpdateTransactionEvidenceRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	// Validate: at least one of url or asset_id must be provided
	if req.URL == nil && req.AssetID == nil {
		response.BadRequest(c, "Yêu cầu cung cấp ít nhất một trong hai trường: url hoặc asset_id")
		return
	}

	// Perform update via service
	txn, err := h.transactionService.UpdateTransactionEvidence(c.Request.Context(), uint(id), req.URL, req.AssetID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			h.logger.Error("Transaction not found for evidence update", "transactionID", id)
			response.NotFound(c, "Không tìm thấy giao dịch")
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error("Failed to update transaction evidence", "transactionID", id, "error", err)
		response.InternalServerError(c, "Không thể cập nhật chứng từ giao dịch")
		return
	}

	response.Success(c, h.mapToTransactionResponse(txn), "Cập nhật chứng từ giao dịch thành công")
}

// ListTransactions retrieves transactions with filters and pagination
// @Summary List transactions
// @Description List transactions with optional filters
// @Tags transactions
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Param sortBy query string false "Sort by field (created_at, amount, party, etc.)" default(created_at)
// @Param sortOrder query string false "Sort order (asc, desc)" default(desc)
// @Param transaction_type query string false "Transaction type (expense or revenue)"
// @Param status query string false "Status (pending or settled)"
// @Param party query string false "Party name"
// @Param fromDate query string false "From date (YYYY-MM-DD)"
// @Param toDate query string false "To date (YYYY-MM-DD)"
// @Param created_by query int false "Created by user ID"
// @Success 200 {object} response.SuccessResponse{data=dto.TransactionListResponse}
// @Failure 500 {object} response.ErrorResponse
// @Router /transactions [get]
func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	filters := h.parseTransactionFilters(c)

	h.logger.Info("Listing transactions",
		"page", filters.Page,
		"pageSize", filters.PageSize,
		"sortBy", filters.SortBy)

	transactions, err := h.transactionService.ListTransactions(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("Failed to list transactions", "error", err, "filters", filters)
		response.InternalServerError(c, "Không thể lấy danh sách giao dịch")
		return
	}

	total, err := h.transactionService.CountTransactions(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("Failed to count transactions", "error", err, "filters", filters)
		response.InternalServerError(c, "Không thể đếm giao dịch")
		return
	}

	h.logger.Info("Listed transactions successfully", "count", len(transactions), "total", total)

	// Map to response
	transactionResponses := make([]dto.TransactionResponse, len(transactions))
	for i, txn := range transactions {
		transactionResponses[i] = h.mapToTransactionResponse(txn)
	}

	// Calculate pagination
	pagination := helpers.CalculatePagination(filters.Page, filters.PageSize, total)

	if len(transactionResponses) == 0 {
		response.SuccessEmptyWithPagination(c, "Không tìm thấy giao dịch nào", pagination)
		return
	}

	response.SuccessWithPagination(c, transactionResponses, "Lấy danh sách giao dịch thành công", pagination)
}

// SettleTransaction creates a settlement for a pending transaction (supports partial settlements)
// @Summary Settle transaction
// @Description Create a settlement for a pending transaction with proof attachment (supports partial settlements)
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Param request body dto.SettleTransactionRequest true "Settlement details"
// @Success 200 {object} response.SuccessResponse{data=dto.TransactionWithSettlementsResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /transactions/{id}/settle [post]
func (h *TransactionHandler) SettleTransaction(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID giao dịch không hợp lệ")
	if !ok {
		return
	}

	// Get user ID from context
	userIDInterface, exists := c.Get(constants.CtxUserID)
	if !exists {
		h.logger.Error("User ID not found in context for settlement")
		response.Forbidden(c, "Không tìm thấy thông tin người dùng")
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok || userID == 0 {
		h.logger.Error("Invalid user ID type in context for settlement")
		response.Forbidden(c, "Không tìm thấy thông tin người dùng")
		return
	}

	// Parse request body
	var req dto.SettleTransactionRequest
	if !helpers.BindJSONWithError(c, &req, "Dữ liệu yêu cầu không hợp lệ") {
		return
	}

	h.logger.Info("Creating settlement", "transactionID", id, "amount", utils.FormatVND(req.Amount), "createdBy", userID)

	// Parse settlement date
	settlementDate, err := time.ParseInLocation("2006-01-02", req.SettlementDate, time.Local)
	if err != nil {
		h.logger.Error("Invalid settlement date format", "date", req.SettlementDate, "error", err)
		response.BadRequest(c, "Định dạng ngày thanh toán không hợp lệ (sử dụng YYYY-MM-DD)")
		return
	}

	// Create settlement domain object
	settlement := &domain.Settlement{
		Amount:         req.Amount,
		SettlementDate: settlementDate,
		ProofURL:       req.ProofURL,
		ProofAssetID:   req.ProofAssetID,
		PaymentMethod:  req.PaymentMethod,
		Notes:          req.Notes,
		CreatedBy:      userID,
	}

	// Create settlement
	txn, createdSettlement, ledgerEntries, err := h.transactionService.CreateSettlement(c.Request.Context(), uint(id), settlement)
	if err != nil {
		if domain.IsNotFoundError(err) {
			h.logger.Error("Transaction not found for settlement", "transactionID", id)
			response.NotFound(c, "Không tìm thấy giao dịch")
			return
		}
		if domain.IsValidationError(err) {
			h.logger.Error("Settlement validation failed", "transactionID", id, "error", err)
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error("Failed to create settlement", "transactionID", id, "error", err)
		response.InternalServerError(c, "Không thể tạo thanh toán")
		return
	}

	h.logger.Info("Settlement created successfully",
		"transactionID", id,
		"settlementID", createdSettlement.ID,
		"amount", utils.FormatVND(createdSettlement.Amount),
		"remaining", utils.FormatVND(txn.GetRemainingAmount()))

	// Build response
	settlementResp := h.mapToSettlementResponse(createdSettlement)
	resp := dto.TransactionWithSettlementsResponse{
		Transaction:   h.mapToTransactionResponse(txn),
		Settlement:    &settlementResp,
		LedgerEntries: h.mapToLedgerEntriesResponse(ledgerEntries),
	}

	response.Success(c, resp, "Thanh toán giao dịch thành công")
}

// DeleteTransaction cancels (soft-deletes) a pending transaction together with its ledger entries.
// Only transactions in the "pending" status can be deleted — settled transactions must be reversed.
// @Summary Cancel pending transaction
// @Description Soft-deletes a pending transaction and its ledger entries. Settled transactions must use reverse instead.
// @Tags transactions
// @Produce json
// @Param id path int true "Transaction ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /transactions/{id} [delete]
func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID giao dịch không hợp lệ")
	if !ok {
		return
	}

	// Load transaction first to enforce status guard. Soft-delete is destructive
	// (cascades to ledger entries), so we must refuse anything that already moved money.
	txn, err := h.transactionService.GetTransaction(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			h.logger.Error("Transaction not found for deletion", "transactionID", id)
			response.NotFound(c, "Không tìm thấy giao dịch")
			return
		}
		h.logger.Error("Failed to get transaction for deletion", "transactionID", id, "error", err)
		response.InternalServerError(c, "Không thể lấy giao dịch")
		return
	}

	if txn.IsReversed() {
		response.BadRequest(c, "Giao dịch đã được đảo ngược, không thể hủy")
		return
	}

	// Domain rule (transaction.go CanReverse): only pending transactions should be deleted;
	// settled/partially_settled transactions must use reverse to keep the audit trail.
	if !txn.IsPending() {
		response.BadRequest(c, "Chỉ có thể hủy giao dịch chờ thanh toán. Giao dịch đã thanh toán vui lòng dùng Đảo ngược.")
		return
	}

	h.logger.Info("Canceling pending transaction", "transactionID", id)

	if err := h.transactionService.DeleteTransaction(c.Request.Context(), uint(id)); err != nil {
		if domain.IsNotFoundError(err) {
			h.logger.Error("Transaction not found during delete", "transactionID", id)
			response.NotFound(c, "Không tìm thấy giao dịch")
			return
		}
		h.logger.Error("Failed to delete transaction", "transactionID", id, "error", err)
		response.InternalServerError(c, "Không thể hủy giao dịch")
		return
	}

	h.logger.Info("Transaction canceled successfully", "transactionID", id)
	response.SuccessEmpty(c, "Hủy giao dịch thành công")
}

// parseTransactionFilters parses query parameters into TransactionFilters
// Supported params: page, pageSize, sortBy, sortOrder, account, party, fromDate, toDate, created_by
func (h *TransactionHandler) parseTransactionFilters(c *gin.Context) domain.TransactionFilters {
	filters := domain.TransactionFilters{
		Page:      1,
		PageSize:  100,
		SortBy:    "created_at",
		SortOrder: "DESC",
	}

	// Pagination
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		filters.Page = page
	}

	if pageSize, err := strconv.Atoi(c.Query("pageSize")); err == nil && pageSize > 0 {
		if pageSize > 100 {
			pageSize = 100 // Max 100 as per standards
		}
		filters.PageSize = pageSize
	}

	// Convert page/pageSize to limit/offset for repository
	filters.Limit = filters.PageSize
	filters.Offset = (filters.Page - 1) * filters.PageSize

	// Sorting
	if sortBy := c.Query("sortBy"); sortBy != "" {
		filters.SortBy = sortBy
	}

	if sortOrder := c.Query("sortOrder"); sortOrder != "" {
		filters.SortOrder = sortOrder
	}

	// Transaction type filter
	if transactionType := c.Query("transaction_type"); transactionType != "" {
		filters.TransactionType = &transactionType
	}

	// Status filter
	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}

	// Party filter
	if party := c.Query("party"); party != "" {
		filters.Party = &party
	}

	// Search filter (description and party)
	if search := c.Query("search"); search != "" {
		filters.Search = search
	}

	// Date range filters
	loc, _ := time.LoadLocation("Local")
	if fromDateStr := c.Query("fromDate"); fromDateStr != "" {
		if fromDate, err := time.ParseInLocation("2006-01-02", fromDateStr, loc); err == nil {
			filters.FromDate = &fromDate
		}
	}

	if toDateStr := c.Query("toDate"); toDateStr != "" {
		if toDate, err := time.ParseInLocation("2006-01-02", toDateStr, loc); err == nil {
			filters.ToDate = &toDate
		}
	}

	// CreatedBy filter
	if createdByStr := c.Query("created_by"); createdByStr != "" {
		if createdBy, err := strconv.ParseUint(createdByStr, 10, 32); err == nil {
			uid := uint(createdBy)
			filters.CreatedBy = &uid
		}
	}

	return filters
}

// ReverseTransaction reverses a transaction by creating opposite ledger entries
// @Summary Reverse transaction
// @Description Reverses a transaction by creating opposite entries that cancel out the original
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path int true "Transaction ID"
// @Param request body dto.ReverseTransactionRequest true "Reversal details"
// @Success 200 {object} response.SuccessResponse{data=dto.ReverseTransactionResponse}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /transactions/{id}/reverse [post]
func (h *TransactionHandler) ReverseTransaction(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID giao dịch không hợp lệ")
	if !ok {
		return
	}

	// Get user ID from context
	userIDInterface, exists := c.Get(constants.CtxUserID)
	if !exists {
		h.logger.Error("User ID not found in context for reversal")
		response.Forbidden(c, "Không tìm thấy thông tin người dùng")
		return
	}
	userID, ok := userIDInterface.(uint)
	if !ok || userID == 0 {
		h.logger.Error("Invalid user ID type in context for reversal")
		response.Forbidden(c, "Không tìm thấy thông tin người dùng")
		return
	}

	// Parse request body
	var req dto.ReverseTransactionRequest
	if !helpers.BindJSONWithError(c, &req, "Dữ liệu yêu cầu không hợp lệ") {
		return
	}

	h.logger.Info("Reversing transaction", "transactionID", id, "reason", req.Reason, "userID", userID)

	// Reverse transaction
	originalTxn, reversalTxn, ledgerEntries, err := h.transactionService.ReverseTransaction(
		c.Request.Context(),
		uint(id),
		req.Reason,
		userID,
	)
	if err != nil {
		if domain.IsNotFoundError(err) {
			h.logger.Error("Transaction not found for reversal", "transactionID", id)
			response.NotFound(c, "Không tìm thấy giao dịch")
			return
		}
		if domain.IsValidationError(err) {
			h.logger.Error("Transaction reversal validation failed", "transactionID", id, "error", err)
			response.BadRequest(c, err.Error())
			return
		}
		h.logger.Error("Failed to reverse transaction", "transactionID", id, "error", err)
		response.InternalServerError(c, "Không thể đảo ngược giao dịch")
		return
	}

	h.logger.Info("Transaction reversed successfully",
		"originalID", originalTxn.ID,
		"reversalID", reversalTxn.ID,
		"userID", userID)

	// Build response
	resp := dto.ReverseTransactionResponse{
		OriginalTransaction: h.mapToTransactionResponse(originalTxn),
		ReversalTransaction: h.mapToTransactionResponse(reversalTxn),
		LedgerEntries:       h.mapToLedgerEntriesResponse(ledgerEntries),
	}

	response.Success(c, resp, "Đảo ngược giao dịch thành công")
}

// mapToTransactionResponse converts domain Transaction to DTO
func (h *TransactionHandler) mapToTransactionResponse(txn *domain.Transaction) dto.TransactionResponse {
	txnCode := ""
	if txn.TransactionCode != nil {
		txnCode = *txn.TransactionCode
	}

	// Determine status string
	status := string(txn.Status)
	if txn.IsPartiallySettled() {
		status = "partially_settled"
	}

	resp := dto.TransactionResponse{
		ID:                    txn.ID,
		Description:           txn.Description,
		TransactionCode:       txnCode,
		TransactionType:       txn.TransactionType,
		Amount:                txn.Amount,
		SettledAmount:         txn.SettledAmount,
		PendingAmount:         txn.GetRemainingAmount(),
		Party:                 txn.Party,
		Status:                status,
		AssetID:               txn.AssetID,
		ReversedTransactionID: txn.ReversedTransactionID,
		CreatedBy:             txn.CreatedBy,
		CreatedAt:             txn.CreatedAt.Format(time.RFC3339),
		UpdatedAt:             txn.UpdatedAt.Format(time.RFC3339),
	}

	if txn.Asset != nil {
		resp.Asset = txn.Asset
	}

	// Map settlements if loaded
	if len(txn.Settlements) > 0 {
		resp.Settlements = make([]dto.SettlementResponse, len(txn.Settlements))
		for i, settlement := range txn.Settlements {
			resp.Settlements[i] = h.mapToSettlementResponse(&settlement)
		}
	}

	return resp
}

func (h *TransactionHandler) mapToSettlementResponse(settlement *domain.Settlement) dto.SettlementResponse {
	return dto.SettlementResponse{
		ID:             settlement.ID,
		TransactionID:  settlement.TransactionID,
		Amount:         settlement.Amount,
		SettlementDate: settlement.SettlementDate.Format("2006-01-02"),
		ProofURL:       settlement.ProofURL,
		ProofAssetID:   settlement.ProofAssetID,
		PaymentMethod:  settlement.PaymentMethod,
		Notes:          settlement.Notes,
		CreatedBy:      settlement.CreatedBy,
		CreatedAt:      settlement.CreatedAt.Format(time.RFC3339),
	}
}

// mapToLedgerEntriesResponse converts ledger entries to response format
func (h *TransactionHandler) mapToLedgerEntriesResponse(entries []*domain.LedgerEntry) []interface{} {
	result := make([]interface{}, len(entries))
	for i, entry := range entries {
		result[i] = map[string]interface{}{
			"id":      entry.ID,
			"date":    entry.Date.Format("2006-01-02"),
			"account": entry.Account,
			"party":   entry.Party,
			"debit":   entry.Debit,
			"credit":  entry.Credit,
			"balance": entry.Balance,
		}
	}
	return result
}

// GetTransactionMetadata returns transaction types and statuses with Vietnamese labels
// @Summary Get transaction metadata
// @Description Get transaction types and statuses with Vietnamese labels for frontend dropdowns
// @Tags transactions
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=dto.TransactionMetadataResponse}
// @Router /transactions/metadata [get]
func (h *TransactionHandler) GetTransactionMetadata(c *gin.Context) {
	metadata := dto.TransactionMetadataResponse{
		TransactionTypes: []dto.TransactionMetadata{
			{Type: string(domain.TransactionTypeExpense), Label: "Chi phí"},
			{Type: string(domain.TransactionTypeRevenue), Label: "Doanh thu"},
			{Type: string(domain.TransactionTypeCapital), Label: "Vốn"},
			{Type: string(domain.TransactionTypeWriteOff), Label: "Xóa nợ"},
			{Type: string(domain.TransactionTypeLoanDisbursement), Label: "Tiền vay nợ"},
			{Type: string(domain.TransactionTypeLoanRepayment), Label: "Thanh toán khoản vay"},
		},
		Statuses: []dto.TransactionMetadata{
			{Type: string(domain.TransactionStatusPending), Label: "Chờ thanh toán"},
			{Type: string(domain.TransactionStatusSettled), Label: "Đã thanh toán"},
			{Type: string(domain.TransactionStatusPartiallySettled), Label: "Thanh toán thiếu"},
		},
	}

	response.Success(c, metadata, "Lấy metadata giao dịch thành công")
}

// ExportTransactions exports transactions to Excel format with optional date filtering
// @Summary Export transactions to Excel
// @Description Export transactions to Excel (XLSX) format with optional date range filtering
// @Tags transactions
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param fromDate query string false "From date (YYYY-MM-DD)"
// @Param toDate query string false "To date (YYYY-MM-DD)"
// @Success 200 {file} binary "Excel file"
// @Failure 500 {object} response.ErrorResponse
// @Router /transactions/export [get]
func (h *TransactionHandler) ExportTransactions(c *gin.Context) {
	// Parse filters - only fromDate and toDate as per documentation
	filters := domain.TransactionFilters{
		Limit:     10000, // Get all transactions for export
		Offset:    0,
		SortBy:    "created_at",
		SortOrder: "ASC",
	}

	// Require fromDate on export — without a lower bound the query is unbounded
	// (up to the 10k cap) and expensive on a growing table. ck:debug 2026-07-04.
	if c.Query("fromDate") == "" {
		response.BadRequest(c, "Tham số fromDate là bắt buộc khi xuất danh sách giao dịch")
		return
	}

	// Parse only the specified query parameters: fromDate and toDate
	loc, _ := time.LoadLocation("Local")
	if fromDateStr := c.Query("fromDate"); fromDateStr != "" {
		if fromDate, err := time.ParseInLocation("2006-01-02", fromDateStr, loc); err == nil {
			filters.FromDate = &fromDate
		}
	}

	if toDateStr := c.Query("toDate"); toDateStr != "" {
		if toDate, err := time.ParseInLocation("2006-01-02", toDateStr, loc); err == nil {
			filters.ToDate = &toDate
		}
	}

	h.logger.Info("Exporting transactions", "fromDate", filters.FromDate, "toDate", filters.ToDate)

	// Get transactions from service
	transactions, err := h.transactionService.ListTransactions(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("Failed to list transactions for export", "error", err, "filters", filters)
		response.InternalServerError(c, constants.MsgFailedToListTransactionsVN)
		return
	}

	h.logger.Info("Retrieved transactions for export", "count", len(transactions))

	// Create Excel file
	excelService := excel.NewExportService()
	f, err := excelService.CreateStyledWorkbook("Transactions")
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelFileVN)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			observability.GetLogger().Error("Error closing Excel file", "error", err)
		}
	}()

	// Define Vietnamese headers
	headers := []string{
		"STT",
		"Ngày",
		"Mô tả",
		"Loại",
		"Số tiền",
		"Đã thanh toán",
		"Quản lý",
		"Trạng thái",
		"Người tạo",
	}

	// Write headers
	if err := excelService.WriteHeaders(f, "Transactions", headers); err != nil {
		response.InternalServerError(c, constants.MsgFailedToWriteExcelHeadersVN)
		return
	}

	// Prepare data rows
	var data [][]interface{}

	for i, txn := range transactions {
		// Get creator name
		creatorName := fmt.Sprintf("User ID: %d", txn.CreatedBy)
		if txn.Creator.Fullname != "" {
			creatorName = txn.Creator.Fullname
		}

		// Get transaction type label
		txnType := txn.TransactionType.Label()

		// Get status label (with special case for partial settlement)
		status := txn.Status.Label()
		if txn.IsPartiallySettled() {
			status = "Thanh toán một phần"
		}

		// Create row data
		rowData := []interface{}{
			i + 1, // STT
			excelService.FormatDateTimeValue(txn.CreatedAt), // Ngày
			txnType,           // Loại
			txn.Amount,        // Số tiền
			txn.SettledAmount, // Đã thanh toán
			txn.Party,         // Quản lý
			status,            // Trạng thái
			creatorName,       // Người tạo
		}

		data = append(data, rowData)
	}

	// Currency columns (indices for Số tiền and Đã thanh toán)
	currencyColumns := []int{4, 5}

	// Write data rows
	for i, rowData := range data {
		rowNum := i + 2 // Start from row 2 (row 1 is headers)
		if err := excelService.WriteDataRow(f, "Transactions", rowNum, rowData, currencyColumns); err != nil {
			response.InternalServerError(c, constants.MsgFailedToWriteExcelDataVN)
			return
		}
	}

	// Auto-size columns
	if err := excelService.AutoSizeColumns(f, "Transactions", headers, data); err != nil {
		response.InternalServerError(c, constants.MsgFailedToAutoSizeColumnsVN)
		return
	}

	// Save to buffer
	buffer, err := excelService.SaveToBuffer(f)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCreateExcelBufferVN)
		return
	}

	// Generate filename with current timestamp
	filename := fmt.Sprintf("lich_su_giao_dich_%s.xlsx", h.clock.Now().Format("20060102_150405"))

	h.logger.Info("Export completed successfully", "filename", filename, "size", len(buffer))

	// Set headers for Excel file download
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(buffer)))

	// Send the Excel file
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}
