package disbursement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/domain/wallet"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// classifyTransportError maps a Go error from the 9pay HTTP client into a
// human-readable error code. The client wraps errors with a "ninepay: ..." prefix
// so we can distinguish HTTP 5xx from timeouts, DNS failures, etc.
func classifyTransportError(err error) string {
	// Pre-flight validation rejections never reached the provider, so they
	// are neither transport nor provider errors. Classify them explicitly so
	// the stored error_code matches the disbursement worker's
	// "preflight_validation" (kept consistent for StatsByErrorCode grouping).
	if errors.Is(err, infrastructure.ErrPreflightValidation) {
		return "preflight_validation"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unexpected status 5"):
		return "http_5xx"
	case strings.Contains(msg, "unexpected status 4"):
		return "http_4xx"
	case strings.Contains(msg, "context deadline exceeded"),
		strings.Contains(msg, "context canceled"),
		strings.Contains(msg, "Client.Timeout"):
		return "timeout"
	case strings.Contains(msg, "no such host"),
		strings.Contains(msg, "i/o timeout"):
		return "dns_failure"
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "TLS handshake"):
		return "connection_error"
	default:
		return "network_error"
	}
}

// ManualDisbursementHandler exposes /admin/manual-disbursement endpoints for
// the "Chuyển tiền" admin page. The Initiate endpoint stamps created_by
// with the authenticated admin's user_id and uses a "tingting" request-id
// prefix for all manual disbursements.
type ManualDisbursementHandler struct {
	registry       *disbursement.Registry
	providerTxs    *disbursement.WalletPaymentService
	bankRepo       domain.BankRepository
	tcRepo         domain.TransactionCodeRepository
	walletSvc      wallet.WalletService
	logger         *slog.Logger
	providerLogger *slog.Logger
}

func NewManualDisbursementHandler(
	registry *disbursement.Registry,
	providerTxs *disbursement.WalletPaymentService,
	bankRepo domain.BankRepository,
	tcRepo domain.TransactionCodeRepository,
	walletSvc wallet.WalletService,
	logger *slog.Logger,
	providerLogger *slog.Logger,
) *ManualDisbursementHandler {
	// Do NOT return nil when registry is nil — the Banks endpoint
	// only needs bankRepo and should remain functional even when
	// no disbursement provider is configured. Other methods check
	// h.registry != nil before use and return appropriate errors.
	if logger == nil {
		logger = slog.Default()
	}
	return &ManualDisbursementHandler{
		registry:       registry,
		providerTxs:    providerTxs,
		bankRepo:       bankRepo,
		tcRepo:         tcRepo,
		walletSvc:      walletSvc,
		logger:         logger,
		providerLogger: providerLogger,
	}
}

// logProvider logs to the payment-gateway file logger.
func (h *ManualDisbursementHandler) logProvider(msg string, args ...any) {
	if h.providerLogger != nil {
		h.providerLogger.Info(msg, args...)
	}
}

// initiateRequest is the frontend payload for POST /admin/manual-disbursement.
type initiateRequest struct {
	RequestID   string `json:"request_id"`
	Amount      int64  `json:"amount"       binding:"required"`
	Description string `json:"description"  binding:"required"`
	BankCode    string `json:"bank_code"    binding:"required"`
	AccountNo   string `json:"account_no"   binding:"required"`
	AccountName string `json:"account_name" binding:"required"`
	AccountType string `json:"account_type"`
}

func formatVND(amount int64) string {
	abs := amount
	if abs < 0 {
		abs = -abs
	}
	s := strconv.FormatInt(abs, 10)
	parts := make([]string, 0, (len(s)-1)/3+1)
	for len(s) > 3 {
		parts = append(parts, s[len(s)-3:])
		s = s[:len(s)-3]
	}
	parts = append(parts, s)
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	result := strings.Join(parts, ".")
	if amount < 0 {
		return "-" + result
	}
	return result
}

const manualTxnPrefix = "TF"

func generateManualTxnCode() string {
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")
	if len(suffix) > 10 {
		suffix = suffix[:10]
	}
	return fmt.Sprintf("%s%s", manualTxnPrefix, suffix)
}

// manualDisbursementResponse mirrors the frontend's ManualDisbursementResponse type.
type manualDisbursementResponse struct {
	TxnID              string  `json:"txn_id"`
	RequestID          string  `json:"request_id"`
	InvoiceNo          string  `json:"invoice_no"`
	Status             string  `json:"status"`
	RequestedAmount    int64   `json:"requested_amount"`
	Fee                int64   `json:"fee"`
	RecipientName      string  `json:"recipient_name"`
	RecipientAccountNo string  `json:"recipient_account_no"`
	RecipientBank      string  `json:"recipient_bank"`
	Description        *string `json:"description"`
	ErrorCode          *string `json:"error_code"`
	ErrorMessage       *string `json:"error_message"`
	CreatedAt          string  `json:"created_at"`
	SettledAt          *string `json:"settled_at"`
	CreatedBy          *uint64 `json:"created_by"`
}

func toManualResponse(row *domaintx.WalletPayment) manualDisbursementResponse {
	resp := manualDisbursementResponse{
		TxnID:              row.TxnID.String(),
		RequestID:          row.RequestID,
		InvoiceNo:          row.GetInvoiceNo(),
		Status:             string(row.Status),
		RequestedAmount:    row.RequestedAmount,
		Fee:                row.Fee,
		RecipientName:      row.RecipientName,
		RecipientAccountNo: row.RecipientAccountNo,
		RecipientBank:      row.RecipientBank,
		Description:        row.Description,
		ErrorCode:          row.ErrorCode,
		ErrorMessage:       row.ErrorMessage,
		CreatedAt:          row.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		CreatedBy:          row.CreatedBy,
	}
	if row.SettledAt != nil {
		s := row.SettledAt.Format("2006-01-02T15:04:05.000Z")
		resp.SettledAt = &s
	}
	return resp
}

// bankInfoResponse mirrors the frontend's BankInfoResponse type.
type bankInfoResponse struct {
	BankCode  string `json:"bank_code"`
	SwiftCode string `json:"swift_code"`
	BankName  string `json:"bank_name"`
}

// Initiate handles POST /admin/manual-disbursement.
//
// Drives the full check-account → create-transfer flow, mirroring the debug
// rig but stamping created_by with the admin's user_id.
func (h *ManualDisbursementHandler) Initiate(c *gin.Context) {
	var body initiateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	if body.Amount < 50000 {
		response.BadRequest(c, "số tiền chuyển tối thiểu 50,000 VND")
		return
	}
	if body.AccountType == "" {
		body.AccountType = "0"
	}

	bal, err := h.walletSvc.GetBalance(c.Request.Context())
	if err != nil {
		h.logger.Error("manual disbursement: failed to get wallet balance", "error", err)
		response.InternalServerError(c, "không thể kiểm tra số dư ví")
		return
	}
	if bal.Available < body.Amount {
		response.BadRequest(c, fmt.Sprintf("số dư khả dụng không đủ: %s < %s", formatVND(bal.Available), formatVND(body.Amount)))
		return
	}

	provider, err := h.resolveProvider(c)
	if err != nil {
		return
	}

	adminUserID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}
	uid := uint64(adminUserID.(uint))

	txnCode := generateManualTxnCode()
	requestID := body.RequestID
	if requestID == "" {
		requestID = txnCode
	}
	swiftCode := h.resolveSwiftCode(c.Request.Context(), body.BankCode)

	codeData := domain.TransactionCodeData{
		ManualDisbursement: &domain.ManualDisbursementData{
			Amount:      body.Amount,
			BankCode:    body.BankCode,
			AccountNo:   body.AccountNo,
			AccountName: body.AccountName,
			Description: body.Description,
		},
	}
	dataBytes, _ := json.Marshal(codeData)
	if err := h.tcRepo.Create(c.Request.Context(), &domain.TransactionCode{
		Code: requestID,
		Data: dataBytes,
	}); err != nil {
		h.logger.Warn("manual disbursement: failed to store transaction code", "code", requestID, "error", err)
	}

	row, err := h.providerTxs.Initiate(c.Request.Context(), disbursement.InitiateInput{
		RequestID:          requestID,
		RequestedAmount:    body.Amount,
		RecipientName:      body.AccountName,
		RecipientAccountNo: body.AccountNo,
		RecipientBank:      body.BankCode,
		Description:        body.Description,
		CreatedBy:          &uid,
	})
	if err != nil {
		if errors.Is(err, disbursement.ErrDuplicatePaymentInProgress) {
			h.logger.Warn("manual disbursement: rejected — duplicate payment in progress",
				"account_no", body.AccountNo, "bank", body.BankCode)
			response.BadRequest(c, "Đã có giao dịch đang xử lý cho người nhận này. Vui lòng đợi giao dịch trước hoàn tất.")
			return
		}
		h.logger.Error("manual disbursement: initiate failed",
			"request_id", requestID, "error", err)
		response.InternalServerError(c, "failed to record transaction: "+err.Error())
		return
	}

	verifier, ok := provider.(infrastructure.AccountVerifier)
	if !ok {
		response.BadRequest(c, "provider does not support account verification")
		return
	}

	h.logProvider("onepay: account check request",
		"request_id", requestID,
		"bank_code", body.BankCode,
		"swift_code", swiftCode,
		"account_no", body.AccountNo,
		"account_name", body.AccountName,
		"account_type", body.AccountType)

	checkRequestID := "AC" + strings.TrimPrefix(requestID, "TF")
	checkResult, err := verifier.CheckAccount(c.Request.Context(), infrastructure.AccountCheckRequest{
		RequestID:   checkRequestID,
		BankCode:    body.BankCode,
		SwiftCode:   swiftCode,
		AccountNo:   body.AccountNo,
		AccountType: body.AccountType,
		AccountName: body.AccountName,
		Amount:      body.Amount,
	})
	if err != nil {
		h.logger.Warn("manual disbursement: check-account transport failed",
			"request_id", requestID, "error", err)
		response.InternalServerError(c, err.Error())
		return
	}

	h.logProvider("onepay: account check response",
		"request_id", requestID,
		"valid", checkResult.Valid,
		"account_name", checkResult.AccountName,
		"raw_error_code", checkResult.RawErrorCode,
		"raw_message", checkResult.RawMessage)

	if _, recordErr := h.providerTxs.RecordAccountCheck(c.Request.Context(), requestID, disbursement.AccountCheckOutcome{
		Verified:     checkResult.Valid,
		RawErrorCode: checkResult.RawErrorCode,
		RawMessage:   checkResult.RawMessage,
	}); recordErr != nil {
		h.logger.Warn("manual disbursement: record account-check failed",
			"request_id", requestID, "error", recordErr)
	}

	if !checkResult.Valid {
		h.logger.Info("manual disbursement: account rejected",
			"request_id", requestID,
			"raw_error_code", checkResult.RawErrorCode,
			"raw_message", checkResult.RawMessage)
		updatedRow, _ := h.providerTxs.GetByTxnID(c.Request.Context(), row.TxnID.String())
		if updatedRow != nil {
			row = updatedRow
		}
		response.Success(c, toManualResponse(row), "account verification failed; transfer not attempted")
		return
	}

	h.logProvider("onepay: transfer request",
		"request_id", requestID,
		"amount", body.Amount,
		"bank_code", body.BankCode,
		"swift_code", swiftCode,
		"account_no", body.AccountNo,
		"account_name", body.AccountName,
		"description", body.Description)

	result, err := provider.InitiateTransfer(c.Request.Context(), infrastructure.TransferRequest{
		RequestID:   requestID,
		Amount:      body.Amount,
		Description: body.Description,
		BankCode:    body.BankCode,
		SwiftCode:   swiftCode,
		AccountNo:   body.AccountNo,
		AccountName: body.AccountName,
		AccountType: body.AccountType,
	})
	if err != nil {
		h.logger.Warn("manual disbursement: transfer failed",
			"request_id", requestID, "error", err)
		if _, recordErr := h.providerTxs.RecordSyncResponse(c.Request.Context(), requestID, disbursement.SyncResult{
			Accepted:     false,
			RawErrorCode: classifyTransportError(err),
			RawMessage:   err.Error(),
			// Pre-flight validation rejection → transfer endpoint never called
			// → no fee charged → waive (zero) the stamped fee.
			FeeWaived: errors.Is(err, infrastructure.ErrPreflightValidation),
		}); recordErr != nil {
			h.logger.Warn("manual disbursement: failed to record sync failure",
				"request_id", requestID, "error", recordErr)
		}
		response.InternalServerError(c, err.Error())
		return
	}

	accepted := result.Status == infrastructure.TransferStatusSuccess ||
		result.Status == infrastructure.TransferStatusPending

	h.logProvider("onepay: transfer response",
		"request_id", requestID,
		"accepted", accepted,
		"provider_ref", result.ProviderRef,
		"status", string(result.Status),
		"raw_error_code", result.RawErrorCode,
		"raw_message", result.RawMessage)

	if _, recordErr := h.providerTxs.RecordSyncResponse(c.Request.Context(), result.RequestID, disbursement.SyncResult{
		Accepted:     accepted,
		InvoiceNo:    result.ProviderRef,
		RawErrorCode: result.RawErrorCode,
		RawMessage:   result.RawMessage,
	}); recordErr != nil {
		h.logger.Warn("manual disbursement: record sync failed",
			"request_id", result.RequestID, "error", recordErr)
	}

	updatedRow, _ := h.providerTxs.GetByTxnID(c.Request.Context(), row.TxnID.String())
	if updatedRow != nil {
		row = updatedRow
	}

	h.logger.Info("manual disbursement: initiated",
		"txn_id", row.TxnID.String(), "request_id", requestID,
		"amount", body.Amount, "status", row.Status)
	response.Success(c, toManualResponse(row), "transfer initiated")
}

// Status handles GET /admin/manual-disbursement/:txn_id.
func (h *ManualDisbursementHandler) Status(c *gin.Context) {
	txnID := c.Param("txn_id")
	if txnID == "" {
		response.BadRequest(c, "txn_id is required")
		return
	}
	row, err := h.providerTxs.GetByTxnID(c.Request.Context(), txnID)
	if err != nil {
		if errors.Is(err, domaintx.ErrNotFound) {
			response.NotFound(c, "transaction not found")
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, toManualResponse(row), "status")
}

// List handles GET /admin/manual-disbursement.
func (h *ManualDisbursementHandler) List(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	rows, err := h.providerTxs.ListRecent(c.Request.Context(), limit)
	if err != nil {
		response.InternalServerError(c, "failed to list transactions")
		return
	}
	out := make([]manualDisbursementResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toManualResponse(row))
	}
	response.Success(c, out, "list")
}

// checkAccountRequest is the payload for POST /admin/manual-disbursement/check-account.
type checkAccountRequest struct {
	BankCode    string `json:"bank_code"    binding:"required"`
	AccountNo   string `json:"account_no"   binding:"required"`
	AccountType string `json:"account_type"`
	AccountName string `json:"account_name"`
}

// CheckAccount handles POST /admin/manual-disbursement/check-account.
func (h *ManualDisbursementHandler) CheckAccount(c *gin.Context) {
	var body checkAccountRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	provider, err := h.resolveProvider(c)
	if err != nil {
		return
	}

	verifier, ok := provider.(infrastructure.AccountVerifier)
	if !ok {
		response.BadRequest(c, "provider does not support account verification")
		return
	}

	accountType := body.AccountType
	if accountType == "" {
		accountType = "0"
	}

	swiftCode := h.resolveSwiftCode(c.Request.Context(), body.BankCode)

	result, err := verifier.CheckAccount(c.Request.Context(), infrastructure.AccountCheckRequest{
		RequestID:   "mdcheck" + strings.ReplaceAll(uuid.New().String()[:8], "-", ""),
		BankCode:    body.BankCode,
		SwiftCode:   swiftCode,
		AccountNo:   body.AccountNo,
		AccountType: accountType,
		AccountName: body.AccountName,
	})
	if err != nil {
		h.logger.Warn("manual disbursement: check-account failed",
			"bank_code", body.BankCode, "error", err)
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, result, "account check completed")
}

// Banks handles GET /admin/manual-disbursement/banks.
func (h *ManualDisbursementHandler) Banks(c *gin.Context) {
	if h.bankRepo == nil {
		response.Success(c, []bankInfoResponse{}, "banks")
		return
	}
	banks, err := h.bankRepo.List(c.Request.Context(), domain.BankFilters{})
	if err != nil {
		h.logger.Warn("manual disbursement: bank list failed", "error", err)
		response.Success(c, []bankInfoResponse{}, "banks")
		return
	}
	out := make([]bankInfoResponse, 0, len(banks))
	for _, b := range banks {
		if b.BankCode == "" && b.SwiftCode == "" {
			continue
		}
		out = append(out, bankInfoResponse{
			BankCode:  b.BankCode,
			SwiftCode: b.SwiftCode,
			BankName:  b.BranchName,
		})
	}
	response.Success(c, out, "banks")
}

// resolveProvider resolves the active disbursement provider from the registry.
func (h *ManualDisbursementHandler) resolveProvider(c *gin.Context) (infrastructure.DisbursementProvider, error) {
	if h.registry == nil {
		response.BadRequest(c, "no disbursement provider is currently active")
		return nil, disbursement.ErrNoActiveProvider
	}
	p, err := h.registry.Active(c.Request.Context())
	if err != nil {
		if errors.Is(err, disbursement.ErrNoActiveProvider) {
			response.BadRequest(c, "no disbursement provider is currently active")
		} else {
			response.InternalServerError(c, err.Error())
		}
		return nil, err
	}
	return p, nil
}

func (h *ManualDisbursementHandler) resolveSwiftCode(ctx context.Context, input string) string {
	if h.bankRepo == nil {
		return fallbackSwiftCode(input)
	}
	if bank, err := h.bankRepo.FindByBankCode(ctx, input); err == nil && bank != nil && bank.SwiftCode != "" {
		return bank.SwiftCode
	}
	if bank, err := h.bankRepo.FindBySwiftCode(ctx, input); err == nil && bank != nil && bank.SwiftCode != "" {
		return bank.SwiftCode
	}
	return fallbackSwiftCode(input)
}

func fallbackSwiftCode(bankCode string) string {
	if isSwiftCode(bankCode) {
		return bankCode
	}
	return ""
}

func isSwiftCode(s string) bool {
	if len(s) < 8 || len(s) > 11 {
		return false
	}
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
