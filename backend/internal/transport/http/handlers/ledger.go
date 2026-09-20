package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/excel"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/timeutil"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"
	"api-server/internal/transport/http/uploadguard"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"api-server/internal/app/services/settlement"
	"api-server/internal/pkg/clock"
)

type LedgerHandler struct {
	ledgerService          *settlement.LedgerService
	onePayFeeImportService *settlement.OnePayFeeImportService
	clock                  clock.Clock
}

// getValidationErrorMessage converts validation errors to user-friendly Vietnamese messages
func getValidationErrorMessage(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			field := fieldError.Field()
			tag := fieldError.Tag()

			switch field {
			case "Account":
				if tag == "required" {
					return constants.MsgLedgerAccountRequiredVN
				}
			case "Party":
				if tag == "required" {
					return constants.MsgLedgerPartyRequiredVN
				}
			case "Date":
				if tag == "required" {
					return constants.MsgLedgerDateRequiredVN
				}
			case "Debit":
				if tag == "min" {
					return constants.MsgLedgerDebitCannotBeNegativeVN
				}
			case "Credit":
				if tag == "min" {
					return constants.MsgLedgerCreditCannotBeNegativeVN
				}
			}
		}
	}

	// Default message for unknown validation errors
	return fmt.Sprintf("%s: %v", constants.MsgInvalidRequestBodyVN, err)
}

func NewLedgerHandler(ledgerService *settlement.LedgerService, onePayFeeImportService *settlement.OnePayFeeImportService, clk clock.Clock) *LedgerHandler {
	if clk == nil {
		clk = clock.New()
	}
	return &LedgerHandler{
		ledgerService:          ledgerService,
		onePayFeeImportService: onePayFeeImportService,
		clock:                  clk,
	}
}

func (h *LedgerHandler) UploadOnePayFeeReport(c *gin.Context) {
	if h.onePayFeeImportService == nil {
		response.InternalServerError(c, "Chức năng nhập phí OnePay chưa được cấu hình")
		return
	}

	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}

	fileHeader, ok := uploadguard.Validate(c, false)
	if !ok {
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.BadRequest(c, "Không thể mở file Excel phí OnePay")
		return
	}
	defer func() { _ = file.Close() }()

	result, err := h.onePayFeeImportService.Import(c.Request.Context(), file, fileHeader.Filename, userID)
	if err != nil {
		var validationErr *settlement.OnePayFeeImportValidationError
		if errors.As(err, &validationErr) {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":      "error",
				"message":     validationErr.Error(),
				"http_status": http.StatusBadRequest,
				"details": gin.H{
					"issues": validationErr.Issues,
				},
			})
			return
		}
		response.HandleDomainError(c, err)
		return
	}

	response.SuccessCreated(c, result, "Đã tạo chi phí OnePay trong sổ cái")
}

// parseLedgerFilters extracts common ledger filter parameters from query string.
func parseLedgerFilters(c *gin.Context) domain.LedgerFilters {
	filters := domain.LedgerFilters{
		SortBy:    "date",
		SortOrder: "desc",
	}

	if account := c.Query("account"); account != "" {
		for _, acc := range strings.Split(account, ",") {
			filters.Account = append(filters.Account, domain.LedgerAccount(strings.TrimSpace(acc)))
		}
	}
	if party := c.Query("party"); party != "" {
		filters.Party = &party
	}
	if fromDate := c.Query("fromDate"); fromDate != "" {
		if date, err := time.Parse(timeutil.DateFormat, fromDate); err == nil {
			filters.FromDate = &date
		}
	}
	if toDate := c.Query("toDate"); toDate != "" {
		if date, err := time.Parse(timeutil.DateFormat, toDate); err == nil {
			filters.ToDate = &date
		}
	}
	if createdBy := c.Query("created_by"); createdBy != "" {
		if id, err := strconv.ParseUint(createdBy, 10, 32); err == nil {
			uid := uint(id)
			filters.CreatedBy = &uid
		}
	}
	if sortBy := c.Query("sortBy"); sortBy != "" {
		filters.SortBy = sortBy
	}
	if sortOrder := c.Query("sortOrder"); sortOrder != "" {
		filters.SortOrder = sortOrder
	}
	return filters
}

// mapLedgerEntryToResponse converts a domain LedgerEntry to its response DTO.
func mapLedgerEntryToResponse(entry *domain.LedgerEntry) dto.LedgerEntryResponse {
	var assetResponse *dto.AssetResponse
	if entry.Asset != nil {
		assetResponse = &dto.AssetResponse{
			ID:         entry.Asset.ID,
			Filename:   entry.Asset.Filename,
			UploadType: entry.Asset.UploadType,
			UploadedBy: entry.Asset.UploadedBy,
			CreatedAt:  entry.Asset.CreatedAt,
		}
	}
	return dto.LedgerEntryResponse{
		ID:                entry.ID,
		Account:           string(entry.Account),
		Party:             entry.Party,
		Debit:             entry.Debit,
		Credit:            entry.Credit,
		Balance:           entry.Balance,
		NetAmount:         domain.GetUserBalanceFromEntry(entry),
		Date:              entry.Date.Format(timeutil.DateFormat),
		AssetID:           entry.AssetID,
		Asset:             assetResponse,
		CreatedBy:         entry.CreatedBy,
		ReversalOfEntryID: entry.ReversalOfEntryID,
		ReversalReason:    entry.ReversalReason,
		CreatedAt:         entry.CreatedAt,
		UpdatedAt:         entry.UpdatedAt,
	}
}

func (h *LedgerHandler) GetEntry(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID bản ghi phải là số hợp lệ")
	if !ok {
		return
	}

	entry, err := h.ledgerService.GetEntry(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	resp := mapLedgerEntryToResponse(entry)
	// Tell the client whether this entry already has a mirror: the reversal
	// endpoint refuses a second reversal, so offering the action would only
	// produce an error the operator cannot act on.
	if entry.ReversalOfEntryID == nil {
		if reversed, hasErr := h.ledgerService.HasReversal(c.Request.Context(), entry.ID); hasErr == nil {
			resp.IsReversed = reversed
		}
	}
	response.Success(c, resp, "Lấy bản ghi sổ cái thành công")
}

func (h *LedgerHandler) ListEntries(c *gin.Context) {
	pg := helpers.ParsePagination(c, 100)
	filters := parseLedgerFilters(c)
	filters.Limit = pg.Limit
	filters.Offset = pg.Offset

	// Legacy offset support
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filters.Offset = o
		}
	}

	entries, err := h.ledgerService.ListEntries(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể lấy danh sách bản ghi sổ cái")
		return
	}

	total, err := h.ledgerService.CountEntries(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể đếm số lượng bản ghi sổ cái")
		return
	}

	entryResponses := make([]dto.LedgerEntryResponse, 0, len(entries))
	for _, entry := range entries {
		entryResponses = append(entryResponses, mapLedgerEntryToResponse(entry))
	}

	pagination := helpers.CalculatePagination(pg.Page, pg.PageSize, total)

	if len(entryResponses) == 0 {
		response.SuccessEmptyWithPagination(c, "Không tìm thấy bản ghi sổ cái nào", pagination)
		return
	}

	message := "Lấy danh sách bản ghi sổ cái thành công"
	if pg.Page > 1 {
		message = "Trang " + strconv.Itoa(pg.Page) + " của danh sách bản ghi sổ cái được lấy thành công"
	}
	response.SuccessWithPagination(c, entryResponses, message, pagination)
}

func (h *LedgerHandler) CreateEntries(c *gin.Context) {
	var req dto.CreateBulkLedgerEntriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, getValidationErrorMessage(err))
		return
	}

	if len(req) == 0 {
		response.BadRequest(c, "Cần ít nhất một bản ghi")
		return
	}

	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}

	var entries []*domain.LedgerEntry
	var totalDebits, totalCredits int64

	for i, entryReq := range req {
		// Validate date format
		date, err := time.Parse(timeutil.DateFormat, entryReq.Date)
		if err != nil {
			response.BadRequest(c, fmt.Sprintf("Định dạng ngày không hợp lệ ở bản ghi %d. Sử dụng YYYY-MM-DD", i+1))
			return
		}

		// Validate that both debit and credit are not non-zero (only one should have value)
		if entryReq.Debit > 0 && entryReq.Credit > 0 {
			response.BadRequest(c, fmt.Sprintf("Bản ghi %d không thể có cả giá trị ghi nợ và ghi có", i+1))
			return
		}

		// Validate that at least one of debit or credit has a value
		if entryReq.Debit == 0 && entryReq.Credit == 0 {
			response.BadRequest(c, fmt.Sprintf("Bản ghi %d phải có ít nhất một giá trị ghi nợ hoặc ghi có", i+1))
			return
		}

		// Validate required fields
		if strings.TrimSpace(entryReq.Account) == "" {
			response.BadRequest(c, fmt.Sprintf("Tài khoản là bắt buộc cho bản ghi %d", i+1))
			return
		}

		if strings.TrimSpace(entryReq.Party) == "" {
			response.BadRequest(c, fmt.Sprintf("Quản lý là bắt buộc cho bản ghi %d", i+1))
			return
		}

		entry := &domain.LedgerEntry{
			Account: domain.LedgerAccount(entryReq.Account),
			Party:   entryReq.Party,
			Debit:   entryReq.Debit,
			Credit:  entryReq.Credit,
			Date:    date,
			AssetID: entryReq.AssetID,
		}

		totalDebits += entry.Debit
		totalCredits += entry.Credit
		entries = append(entries, entry)
	}

	createdEntries, err := h.ledgerService.CreateEntries(c.Request.Context(), entries, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	entryResponses := make([]dto.LedgerEntryResponse, 0, len(createdEntries))
	for _, entry := range createdEntries {
		entryResponses = append(entryResponses, mapLedgerEntryToResponse(entry))
	}

	bulkResponse := dto.BulkLedgerEntriesResponse{
		Entries:      entryResponses,
		TotalEntries: len(entryResponses),
		TotalDebits:  totalDebits,
		TotalCredits: totalCredits,
	}

	response.SuccessCreated(c, bulkResponse, "Tạo các bản ghi sổ cái thành công")
}

func (h *LedgerHandler) GetBalance(c *gin.Context) {
	// Check if account query parameter is provided
	account := c.Query("account")

	if account != "" {
		// Get balance for specific account
		validAccounts := []string{"cash", "receivable", "payable", "expense", "revenue", "equity"}
		isValid := false
		for _, validAccount := range validAccounts {
			if account == validAccount {
				isValid = true
				break
			}
		}

		if !isValid {
			response.BadRequest(c, fmt.Sprintf("Loại tài khoản không hợp lệ. Phải là một trong: %s", strings.Join(validAccounts, ", ")))
			return
		}

		balance, err := h.ledgerService.GetAccountBalance(c.Request.Context(), domain.LedgerAccount(account))
		if err != nil {
			response.HandleDomainError(c, err)
			return
		}

		balanceResponse := map[string]any{
			"balance": balance,
			"account": account,
			"as_of":   h.clock.Now().Format(time.RFC3339),
		}

		response.Success(c, balanceResponse, fmt.Sprintf("Lấy số dư tài khoản %s thành công", account))
		return
	}

	// Original behavior - return overall balance
	balance, err := h.ledgerService.GetBalance(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Không thể lấy số dư")
		return
	}

	balanceResponse := map[string]any{
		"balance": balance,
	}

	response.Success(c, balanceResponse, "Lấy số dư thành công")
}

func (h *LedgerHandler) GetCashFlowSummary(c *gin.Context) {
	// Parse date parameters
	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")

	if fromDateStr == "" || toDateStr == "" {
		response.BadRequest(c, "Cả hai tham số fromDate và toDate đều là bắt buộc")
		return
	}

	fromDate, err := time.Parse(timeutil.DateFormat, fromDateStr)
	if err != nil {
		response.BadRequest(c, "Định dạng fromDate không hợp lệ. Sử dụng YYYY-MM-DD")
		return
	}

	toDate, err := time.Parse(timeutil.DateFormat, toDateStr)
	if err != nil {
		response.BadRequest(c, "Định dạng toDate không hợp lệ. Sử dụng YYYY-MM-DD")
		return
	}

	if fromDate.After(toDate) {
		response.BadRequest(c, "fromDate không thể sau toDate")
		return
	}

	summary, err := h.ledgerService.GetCashFlowSummary(c.Request.Context(), fromDate, toDate)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	summaryResponse := dto.CashFlowSummaryResponse{
		StartDate:      summary.StartDate.Format(timeutil.DateFormat),
		EndDate:        summary.EndDate.Format(timeutil.DateFormat),
		TotalInflow:    summary.TotalInflow,
		TotalOutflow:   summary.TotalOutflow,
		NetCashFlow:    summary.NetCashFlow,
		OpeningBalance: summary.OpeningBalance,
		ClosingBalance: summary.ClosingBalance,
		ByAccount:      make(map[string]dto.AccountSummaryResponse),
	}

	// Convert account summaries
	for account, accountSummary := range summary.ByAccount {
		summaryResponse.ByAccount[string(account)] = dto.AccountSummaryResponse{
			TotalDebits:  accountSummary.TotalDebit,
			TotalCredits: accountSummary.TotalCredit,
			NetAmount:    accountSummary.NetAmount,
		}
	}

	response.Success(c, summaryResponse, "Lấy báo cáo dòng tiền thành công")
}

func (h *LedgerHandler) RecalculateBalances(c *gin.Context) {
	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}

	if err := h.ledgerService.RecalculateAllBalances(c.Request.Context(), userID); err != nil {
		response.InternalServerError(c, "Không thể tính lại số dư")
		return
	}

	response.Success(c, nil, "Tính lại số dư sổ cái thành công")
}

func (h *LedgerHandler) ReverseEntry(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", "ID bản ghi phải là số hợp lệ")
	if !ok {
		return
	}

	var req dto.ReverseLedgerEntryRequest
	if !helpers.BindJSONWithError(c, &req, "Nội dung yêu cầu không hợp lệ") {
		return
	}

	userID, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return
	}

	reason := strings.TrimSpace(req.Reason)
	result, err := h.ledgerService.ReverseEntry(c.Request.Context(), id, reason, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// The response carries the mirror of the requested entry and says how many
	// rows the reversal covered: a block is reversed as a whole, so the operator
	// has to know the action touched more than the row they opened.
	response.SuccessCreated(c, mapLedgerEntryToResponse(result.Reversed),
		fmt.Sprintf(constants.MsgLedgerReversedBlockVN, result.Entries))
}

func (h *LedgerHandler) GetLedgerSummary(c *gin.Context) {
	// Parse and validate query parameters
	fromDateStr := c.Query("from")
	toDateStr := c.Query("to")

	var fromDate, toDate time.Time
	var err error

	// If dates are not provided, use full date range (from earliest to current date)
	if fromDateStr == "" || toDateStr == "" {
		// Use a very early date as the starting point (e.g., year 2000)
		fromDate = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		toDate = h.clock.NowUTC()
	} else {
		fromDate, err = time.Parse(timeutil.DateFormat, fromDateStr)
		if err != nil {
			response.BadRequest(c, "Định dạng ngày 'from' không hợp lệ. Sử dụng YYYY-MM-DD")
			return
		}

		toDate, err = time.Parse(timeutil.DateFormat, toDateStr)
		if err != nil {
			response.BadRequest(c, "Định dạng ngày 'to' không hợp lệ. Sử dụng YYYY-MM-DD")
			return
		}
	}

	// Call service
	summary, err := h.ledgerService.GetLedgerSummary(c.Request.Context(), fromDate, toDate)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, summary, "Lấy tổng quan sổ cái thành công")
}

func (h *LedgerHandler) GetAccountsMetadata(c *gin.Context) {
	metadata, err := h.ledgerService.GetAccountsMetadata(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Không thể lấy thông tin tài khoản")
		return
	}

	// Convert to DTO format - return as direct array to match specification
	accounts := make([]dto.AccountMetadataResponse, 0, len(metadata))
	for _, meta := range metadata {
		accounts = append(accounts, dto.AccountMetadataResponse{
			Value:      string(meta.Value),
			Label:      meta.Label,
			Category:   string(meta.Category),
			NormalSide: meta.NormalSide,
		})
	}

	response.Success(c, accounts, "Lấy thông tin tài khoản thành công")
}

func (h *LedgerHandler) ExportEntries(c *gin.Context) {
	filters := parseLedgerFilters(c)
	filters.Limit = 10000 // Get all entries for export
	filters.Offset = 0

	// Get entries from service
	entries, err := h.ledgerService.ListEntries(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể lấy danh sách bản ghi sổ cái")
		return
	}

	// Create Excel file
	excelService := excel.NewExportService()
	f, err := excelService.CreateStyledWorkbook("Ledger Entries")
	if err != nil {
		response.InternalServerError(c, "Không thể tạo file Excel")
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
		"Loại tài khoản",
		"Quản lý",
		"Ghi nợ",
		"Ghi có",
		"Số dư",
		"Người tạo",
		"Ngày tạo",
	}

	// Write headers
	if err := excelService.WriteHeaders(f, "Ledger Entries", headers); err != nil {
		response.InternalServerError(c, "Không thể ghi tiêu đề Excel")
		return
	}

	// Prepare data rows
	var data [][]interface{}
	var totalDebit, totalCredit, totalBalance float64

	for i, entry := range entries {
		// Get Vietnamese account name
		accountDisplay := domain.GetAccountDisplayName(entry.Account)

		// Get creator name if available
		creatorName := ""
		if entry.Creator.Fullname != "" {
			creatorName = entry.Creator.Fullname
		} else {
			creatorName = fmt.Sprintf("User ID: %d", entry.CreatedBy)
		}

		// Create row data
		rowData := []interface{}{
			i + 1,                                    // STT
			excelService.FormatDateValue(entry.Date), // Ngày
			accountDisplay,                           // Loại tài khoản
			entry.Party,                              // Quản lý
			entry.Debit,                              // Ghi nợ
			entry.Credit,                             // Ghi có
			entry.Balance,                            // Số dư
			creatorName,                              // Người tạo
			excelService.FormatDateValue(entry.CreatedAt), // Ngày tạo
		}

		data = append(data, rowData)
		totalDebit += float64(entry.Debit)
		totalCredit += float64(entry.Credit)
		totalBalance = float64(entry.Balance) // Last balance is the current total balance
	}

	// Add summary row
	if len(entries) > 0 {
		summaryRow := []interface{}{
			"",           // STT
			"",           // Ngày
			"",           // Loại tài khoản
			"TỔNG CỘNG",  // Quản lý
			totalDebit,   // Ghi nợ
			totalCredit,  // Ghi có
			totalBalance, // Số dư
			"",           // Người tạo
			"",           // Ngày tạo
		}
		data = append(data, summaryRow)
	}

	// Currency columns (indices for Ghi nợ, Ghi có, Số dư)
	currencyColumns := []int{4, 5, 6}

	// Write data rows
	for i, rowData := range data {
		rowNum := i + 2 // Start from row 2 (row 1 is headers)
		if err := excelService.WriteDataRow(f, "Ledger Entries", rowNum, rowData, currencyColumns); err != nil {
			response.InternalServerError(c, "Không thể ghi dữ liệu Excel")
			return
		}
	}

	// Auto-size columns
	if err := excelService.AutoSizeColumns(f, "Ledger Entries", headers, data); err != nil {
		response.InternalServerError(c, "Không thể tự động điều chỉnh cột")
		return
	}

	// Save to buffer
	buffer, err := excelService.SaveToBuffer(f)
	if err != nil {
		response.InternalServerError(c, "Không thể tạo file Excel")
		return
	}

	// Generate filename with current date
	filename := fmt.Sprintf("ledger_export_%s.xlsx", h.clock.Now().Format("2006-01-02"))

	// Set headers for Excel file download
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(buffer)))

	// Send the Excel file
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}
