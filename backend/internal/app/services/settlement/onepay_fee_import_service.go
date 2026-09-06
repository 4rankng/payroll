package settlement

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	infraServices "api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/infra/disbursement/onepay"
	"api-server/internal/infra/observability"
	bankmapping "api-server/internal/pkg/bank"
	"api-server/internal/pkg/timeutil"

	"api-server/internal/pkg/excelkit"
	"github.com/gosimple/unidecode"

	"github.com/xuri/excelize/v2"
)

const onePayFeeParty = "OnePay"

// Statement banners (company header, title, signature blocks) sit above the
// data, so sheet-content scans only need a bounded row prefix.
const detailHeaderScanRows = 30

// Header spellings shared by every observed OnePay statement layout. The
// diacritic-free spellings catch files stored in a different unicode form.
var commonDetailHeaderAliases = map[string]string{
	"merchant id":               "merchant id",
	"merchant name":             "merchant name",
	"merchant fund transfer id": "merchant fund transfer id",
	"op transaction id":         "op transaction id",
	"currency":                  "currency",
	"beneficiary account":       "beneficiary account",
	"số tài khoản":              "beneficiary account",
	"so tai khoan":              "beneficiary account",
	"beneficiary account name":  "beneficiary account name",
	"tên chủ tài khoản":         "beneficiary account name",
	"ten chu tai khoan":         "beneficiary account name",
	"beneficiary bank":          "beneficiary bank",
	"ngân hàng":                 "beneficiary bank",
	"ngan hang":                 "beneficiary bank",
	"state":                     "state",
	"trạng thái":                "state",
	"trang thai":                "state",
}

// onePayDetailTemplate is the parsing strategy for one known OnePay statement
// layout: which sheet carries the transactions, how its headers are spelled,
// and which columns must be present. A sheet matches a template only when some
// header row resolves every required column through the template's aliases,
// so templates never collide and a renamed sheet still parses.
type onePayDetailTemplate struct {
	name            string
	sheetNames      []string
	headerAliases   map[string]string
	requiredColumns []string
}

// Known OnePay statement layouts, tried in order. To support a new export
// format, describe it here — the parsing loop stays untouched.
var onePayDetailTemplates = []*onePayDetailTemplate{
	{
		// Legacy transaction export: "GD" sheet, flat English headers on the
		// first row, State column mandatory.
		name:       "GD",
		sheetNames: []string{"GD"},
		headerAliases: mergeDetailHeaderAliases(map[string]string{
			"create date": "create date",
			"amount":      "amount",
		}),
		requiredColumns: []string{
			"merchant id", "merchant fund transfer id", "op transaction id", "create date",
			"beneficiary account", "beneficiary account name", "beneficiary bank", "amount", "state",
		},
	},
	{
		// Monthly statement: "CHI TIET THANG" sheet behind banner rows,
		// bilingual VN/EN header rows, no State column. Also accepts the
		// legacy English spellings so future layout merges keep parsing.
		name:       "CHI TIET THANG",
		sheetNames: []string{"CHI TIET THANG"},
		headerAliases: mergeDetailHeaderAliases(map[string]string{
			"create date":         "create date",
			"transaction date":    "create date",
			"thời gian giao dịch": "create date",
			"thoi gian giao dich": "create date",
			"amount":              "amount",
			"transaction amount":  "amount",
			"giá trị gd":          "amount",
			"gia tri gd":          "amount",
		}),
		requiredColumns: []string{
			"merchant id", "merchant fund transfer id", "op transaction id", "create date",
			"beneficiary account", "beneficiary account name", "beneficiary bank", "amount",
		},
	},
}

func mergeDetailHeaderAliases(templateSpecific map[string]string) map[string]string {
	merged := make(map[string]string, len(commonDetailHeaderAliases)+len(templateSpecific))
	maps.Copy(merged, commonDetailHeaderAliases)
	maps.Copy(merged, templateSpecific)
	return merged
}

// resolveHeader maps a raw header cell to this template's canonical field.
// Delegates to the shared excelkit AliasTable, keeping one resolution
// algorithm across importers. The kit's order (whole cell → collapsed →
// diacritic-free collapse → newline segments) preserves this template's
// collapsed-first outcomes: segments run last and cannot preempt a
// whole-cell match.
func (tpl *onePayDetailTemplate) resolveHeader(s string) string {
	key := normalizeHeader(s)
	if key == "" {
		return ""
	}
	if canonical, ok := excelkit.AliasTable(tpl.headerAliases).Resolve(s); ok {
		return canonical
	}
	return ""
}

// locateDetailSheet returns the template and sheet matching the workbook.
// Preferred sheet names are tried first; otherwise every non-summary sheet is
// scanned for a satisfying header row, so renamed sheets still parse.
func locateDetailSheet(f *excelize.File, summarySheet string) (*onePayDetailTemplate, string) {
	for _, tpl := range onePayDetailTemplates {
		for _, want := range tpl.sheetNames {
			if name := findSheet(f, want); name != "" && name != summarySheet && tpl.hasHeaderRow(f, name) {
				return tpl, name
			}
		}
	}
	for _, tpl := range onePayDetailTemplates {
		for _, name := range f.GetSheetList() {
			if name == summarySheet {
				continue
			}
			if tpl.hasHeaderRow(f, name) {
				return tpl, name
			}
		}
	}
	return nil, ""
}

// hasHeaderRow reports whether one of the sheet's first rows resolves every
// required column of the template.
func (tpl *onePayDetailTemplate) hasHeaderRow(f *excelize.File, sheet string) bool {
	rows, err := f.Rows(sheet)
	if err != nil {
		return false
	}
	defer func() { _ = rows.Close() }()
	for i := 0; i < detailHeaderScanRows && rows.Next(); i++ {
		cols, err := rows.Columns()
		if err != nil {
			return false
		}
		if tpl.headerRowMatches(cols) {
			return true
		}
	}
	return false
}

func (tpl *onePayDetailTemplate) headerRowMatches(cols []string) bool {
	resolved := make(map[string]bool, len(cols))
	for _, c := range cols {
		if canonical := tpl.resolveHeader(c); canonical != "" {
			resolved[canonical] = true
		}
	}
	for _, required := range tpl.requiredColumns {
		if !resolved[required] {
			return false
		}
	}
	return true
}

// detailHeaderIndex finds the header row within the full row set. Row indices
// here can differ from the streaming scan when a sheet omits leading empty
// rows, so detection always re-runs on the same slice the parser walks.
func (tpl *onePayDetailTemplate) detailHeaderIndex(rows [][]string) int {
	limit := min(len(rows), detailHeaderScanRows)
	for i := range limit {
		if tpl.headerRowMatches(rows[i]) {
			return i
		}
	}
	return -1
}

type OnePayFeeImportValidationError struct {
	Message string
	Issues  []dto.OnePayFeeReportIssue
}

func (e *OnePayFeeImportValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "báo cáo phí OnePay không hợp lệ"
}

type OnePayFeeImportService struct {
	walletPayments     domaintx.WalletPaymentRepository
	bankRepo           domain.BankRepository
	transactionRepo    domain.TransactionRepository
	transactionService *TransactionService
	auditService       *infraServices.AuditService
	logger             *slog.Logger
}

type onePayFeeReport struct {
	summary dto.OnePayFeeReportSummary
	details []onePayFeeReportDetail
}

type onePayFeeReportDetail struct {
	row                    int
	merchantID             string
	merchantName           string
	fundTransferID         string
	opTransactionID        string
	createDate             time.Time
	currency               string
	beneficiaryAccount     string
	beneficiaryAccountName string
	beneficiaryBank        string
	amount                 int64
	state                  string
}

func NewOnePayFeeImportService(
	walletPayments domaintx.WalletPaymentRepository,
	bankRepo domain.BankRepository,
	transactionRepo domain.TransactionRepository,
	transactionService *TransactionService,
	auditService *infraServices.AuditService,
	logger *slog.Logger,
) *OnePayFeeImportService {
	if logger == nil {
		logger = observability.GetLogger()
	}
	return &OnePayFeeImportService{
		walletPayments:     walletPayments,
		bankRepo:           bankRepo,
		transactionRepo:    transactionRepo,
		transactionService: transactionService,
		auditService:       auditService,
		logger:             logger,
	}
}

func (s *OnePayFeeImportService) Import(ctx context.Context, r io.Reader, filename string, userID uint) (*dto.OnePayFeeImportResponse, error) {
	if s.walletPayments == nil || s.transactionRepo == nil || s.transactionService == nil {
		return nil, errors.New("onepay fee import service is not configured")
	}

	report, issues, err := parseOnePayFeeReport(r)
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		issues = append(issues, s.validateDuplicate(ctx, report.summary.ImportReference)...)
	}
	if len(issues) == 0 {
		issues = append(issues, s.validateWalletPayments(ctx, report)...)
	}
	if len(issues) > 0 {
		return nil, &OnePayFeeImportValidationError{
			Message: "Báo cáo phí OnePay có lỗi đối soát",
			Issues:  issues,
		}
	}

	txn := &domain.Transaction{
		Description:     fmt.Sprintf("Phí OnePay %s - %s", report.summary.PeriodLabel, report.summary.ImportReference),
		TransactionType: domain.TransactionTypeExpense,
		Amount:          report.summary.TotalFee,
		Party:           onePayFeeParty,
		Status:          domain.TransactionStatusSettled,
		CreatedBy:       userID,
	}

	createdTxn, ledgerEntries, err := s.transactionService.CreateTransaction(ctx, txn)
	if err != nil {
		return nil, err
	}

	if s.auditService != nil {
		recordCount := report.summary.TransactionCount
		go func() {
			if err := s.auditService.LogFileImport(ctx, "onepay_fee_report", recordCount, filename); err != nil {
				s.logger.Warn("failed to log onepay fee report import", "error", err)
			}
		}()
	}

	ledgerEntryIDs := make([]uint, 0, len(ledgerEntries))
	for _, entry := range ledgerEntries {
		ledgerEntryIDs = append(ledgerEntryIDs, entry.ID)
	}

	transactionCode := ""
	if createdTxn.TransactionCode != nil {
		transactionCode = *createdTxn.TransactionCode
	}

	return &dto.OnePayFeeImportResponse{
		Summary:         report.summary,
		TransactionID:   createdTxn.ID,
		TransactionCode: transactionCode,
		LedgerEntryIDs:  ledgerEntryIDs,
		CreatedAt:       createdTxn.CreatedAt,
	}, nil
}

func (s *OnePayFeeImportService) validateDuplicate(ctx context.Context, reference string) []dto.OnePayFeeReportIssue {
	txType := string(domain.TransactionTypeExpense)
	party := onePayFeeParty
	existing, err := s.transactionRepo.List(ctx, domain.TransactionFilters{
		TransactionType: &txType,
		Party:           &party,
		Search:          reference,
		Limit:           1,
	})
	if err != nil {
		return []dto.OnePayFeeReportIssue{{
			Code:    "transaction_lookup_failed",
			Message: "Không thể kiểm tra giao dịch phí OnePay đã tồn tại",
		}}
	}
	if len(existing) > 0 {
		return []dto.OnePayFeeReportIssue{{
			Code:      "duplicate_import",
			Reference: reference,
			Message:   fmt.Sprintf("Đã tồn tại chi phí OnePay cho kỳ này (giao dịch #%d)", existing[0].ID),
		}}
	}
	return nil
}

func (s *OnePayFeeImportService) validateWalletPayments(ctx context.Context, report *onePayFeeReport) []dto.OnePayFeeReportIssue {
	var issues []dto.OnePayFeeReportIssue
	var appFeeTotal int64

	for _, detail := range report.details {
		row, err := s.walletPayments.GetByRequestID(ctx, detail.fundTransferID)
		if err != nil {
			if errors.Is(err, domaintx.ErrNotFound) {
				issues = append(issues, issue("missing_app_payment", detail.row, detail.fundTransferID, "Không tìm thấy giao dịch chi hộ trong hệ thống"))
				continue
			}
			issues = append(issues, issue("payment_lookup_failed", detail.row, detail.fundTransferID, "Không thể kiểm tra giao dịch chi hộ trong hệ thống"))
			continue
		}

		appFeeTotal += row.Fee
		if row.Provider != onepay.ProviderName {
			issues = append(issues, issue("provider_mismatch", detail.row, detail.fundTransferID, fmt.Sprintf("Nhà cung cấp trong hệ thống là %s, không phải OnePay", row.Provider)))
		}
		if row.Status != domaintx.StateCompleted {
			issues = append(issues, issue("status_mismatch", detail.row, detail.fundTransferID, fmt.Sprintf("Trạng thái trong hệ thống là %s, chưa phải completed", row.Status)))
		}
		if row.RequestedAmount != detail.amount {
			issues = append(issues, issue("amount_mismatch", detail.row, detail.fundTransferID, fmt.Sprintf("Số tiền hệ thống %d khác báo cáo %d", row.RequestedAmount, detail.amount)))
		}
		if normalizeText(row.RecipientAccountNo) != normalizeText(detail.beneficiaryAccount) {
			issues = append(issues, issue("account_mismatch", detail.row, detail.fundTransferID, "Số tài khoản thụ hưởng không khớp"))
		}
		if !s.bankMatches(ctx, row.RecipientBank, detail.beneficiaryBank) {
			issues = append(issues, issue("bank_mismatch", detail.row, detail.fundTransferID, "Ngân hàng thụ hưởng không khớp"))
		}
		if normalizeName(row.RecipientName) != normalizeName(detail.beneficiaryAccountName) {
			issues = append(issues, issue("recipient_name_mismatch", detail.row, detail.fundTransferID, "Tên người thụ hưởng không khớp"))
		}
		if row.GetInvoiceNo() != "" && detail.opTransactionID != "" && normalizeText(row.GetInvoiceNo()) != normalizeText(detail.opTransactionID) {
			issues = append(issues, issue("provider_reference_mismatch", detail.row, detail.fundTransferID, "Mã giao dịch OnePay không khớp"))
		}
	}

	report.summary.AppRecordedFeeTotal = appFeeTotal
	return issues
}

func parseOnePayFeeReport(r io.Reader) (*onePayFeeReport, []dto.OnePayFeeReportIssue, error) {
	f, err := excelkit.OpenReader(r)
	if err != nil {
		return nil, nil, &OnePayFeeImportValidationError{
			Message: "Không thể đọc file Excel OnePay",
			Issues: []dto.OnePayFeeReportIssue{{
				Code:    "invalid_excel",
				Message: "File không đúng định dạng Excel hoặc đã bị lỗi",
			}},
		}
	}
	defer func() { _ = f.Close() }()

	summarySheet := findSummarySheet(f)
	detailTemplate, detailSheet := locateDetailSheet(f, summarySheet)
	var issues []dto.OnePayFeeReportIssue
	if summarySheet == "" {
		issues = append(issues, issue("missing_sheet", 0, "", "Không tìm thấy sheet tổng hợp phí PHI THANG"))
	}
	if detailSheet == "" {
		issues = append(issues, issue("missing_sheet", 0, "", "Không tìm thấy sheet chi tiết giao dịch (GD hoặc CHI TIET THANG)"))
	}
	if len(issues) > 0 {
		return nil, issues, nil
	}

	summary, summaryIssues := parseSummarySheet(f, summarySheet)
	issues = append(issues, summaryIssues...)
	details, detailTotalAmount, detailIssues := parseDetailSheet(f, detailSheet, detailTemplate, summary.PeriodFrom, summary.PeriodTo)
	summary.DetailTotalAmount = detailTotalAmount
	issues = append(issues, detailIssues...)

	if len(details) != summary.TransactionCount {
		issues = append(issues, issue("transaction_count_mismatch", 0, "", fmt.Sprintf("Sheet chi tiết có %d giao dịch, sheet PHI THANG ghi %d giao dịch", len(details), summary.TransactionCount)))
	}
	expectedTotalFee := summary.FeePerTransaction * int64(summary.TransactionCount)
	if expectedTotalFee != summary.TotalFee {
		issues = append(issues, issue("fee_total_mismatch", 0, "", fmt.Sprintf("Tổng phí %d không bằng SLGD %d x phí XLGD %d", summary.TotalFee, summary.TransactionCount, summary.FeePerTransaction)))
	}

	merchantID, merchantName := inferMerchant(details)
	if summary.MerchantID == "" {
		summary.MerchantID = merchantID
	}
	if summary.MerchantName == "" {
		summary.MerchantName = merchantName
	}
	if summary.ImportReference == "" {
		periodKey := summary.PeriodFrom
		if len(periodKey) >= 7 {
			periodKey = periodKey[:7]
		}
		summary.ImportReference = fmt.Sprintf("ONEPAY-FEE:%s:%s", summary.MerchantID, periodKey)
	}

	return &onePayFeeReport{summary: summary, details: details}, issues, nil
}

func parseSummarySheet(f *excelize.File, sheet string) (dto.OnePayFeeReportSummary, []dto.OnePayFeeReportIssue) {
	var summary dto.OnePayFeeReportSummary
	var issues []dto.OnePayFeeReportIssue
	rows, err := f.GetRows(sheet)
	if err != nil {
		return summary, []dto.OnePayFeeReportIssue{issue("summary_read_failed", 0, "", "Không thể đọc sheet PHI THANG")}
	}

	for i, row := range rows {
		line := strings.Join(row, " ")
		if summary.PeriodFrom == "" && strings.Contains(line, "Từ ngày") {
			from, to, ok := parseVietnameseDateRange(line)
			if ok {
				summary.PeriodFrom = from.Format(timeutil.DateFormat)
				summary.PeriodTo = to.Format(timeutil.DateFormat)
			}
		}
		if summary.MerchantID == "" && strings.Contains(line, " - PO - ") {
			parts := strings.Split(line, " - PO - ")
			summary.MerchantID = strings.TrimSpace(parts[len(parts)-1])
		}
		if len(row) >= 5 && strings.TrimSpace(cell(row, 0)) == "1" {
			summary.PeriodLabel = strings.TrimSpace(cell(row, 1))
			summary.TransactionCount = int(mustCellInt(f, sheet, i+1, 3))
			summary.FeePerTransaction = mustCellInt(f, sheet, i+1, 4)
			summary.TotalFee = mustCellInt(f, sheet, i+1, 5)
		}
	}

	if summary.PeriodFrom == "" || summary.PeriodTo == "" {
		issues = append(issues, issue("period_missing", 0, "", "Không đọc được kỳ đối soát trong sheet PHI THANG"))
	}
	if summary.PeriodLabel == "" {
		issues = append(issues, issue("period_label_missing", 0, "", "Không đọc được dòng phí tháng trong sheet PHI THANG"))
	}
	if summary.TransactionCount <= 0 {
		issues = append(issues, issue("transaction_count_missing", 0, "", "Số lượng giao dịch phải lớn hơn 0"))
	}
	if summary.FeePerTransaction <= 0 {
		issues = append(issues, issue("fee_per_transaction_missing", 0, "", "Phí xử lý giao dịch phải lớn hơn 0"))
	}
	if summary.TotalFee <= 0 {
		issues = append(issues, issue("total_fee_missing", 0, "", "Tổng phí phải lớn hơn 0"))
	}
	return summary, issues
}

func parseDetailSheet(f *excelize.File, sheet string, tpl *onePayDetailTemplate, periodFrom, periodTo string) ([]onePayFeeReportDetail, int64, []dto.OnePayFeeReportIssue) {
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return nil, 0, []dto.OnePayFeeReportIssue{issue("detail_read_failed", 0, "", "Không thể đọc sheet chi tiết giao dịch")}
	}

	headerIdx := tpl.detailHeaderIndex(rows)
	if headerIdx < 0 {
		return nil, 0, []dto.OnePayFeeReportIssue{issue("missing_header_row", 0, "", "Không tìm thấy dòng tiêu đề trong sheet chi tiết giao dịch")}
	}
	headers := make(map[string]int)
	for i, h := range rows[headerIdx] {
		if canonical := tpl.resolveHeader(h); canonical != "" {
			if _, exists := headers[canonical]; !exists {
				headers[canonical] = i
			}
		}
	}
	// Every field used to reconcile an application payment stays mandatory.
	// State is validated whenever the column exists, but only the legacy GD
	// template demands it — monthly statements omit it entirely.
	var issues []dto.OnePayFeeReportIssue
	for _, required := range tpl.requiredColumns {
		if _, ok := headers[required]; !ok {
			issues = append(issues, issue("missing_column", headerIdx+1, required, fmt.Sprintf("Thiếu cột %s trong sheet chi tiết giao dịch", required)))
		}
	}
	if len(issues) > 0 {
		return nil, 0, issues
	}
	stateColumn, hasState := headers["state"]

	var from, to time.Time
	if periodFrom != "" && periodTo != "" {
		from, _ = time.Parse(timeutil.DateFormat, periodFrom)
		to, _ = time.Parse(timeutil.DateFormat, periodTo)
		to = to.Add(24*time.Hour - time.Nanosecond)
	}

	seen := map[string]int{}
	details := make([]onePayFeeReportDetail, 0, len(rows)-headerIdx-1)
	var totalAmount int64
	for idx := headerIdx + 1; idx < len(rows); idx++ {
		row := rows[idx]
		fundTransferID := strings.TrimSpace(cell(row, headers["merchant fund transfer id"]))
		if fundTransferID == "" || normalizeHeader(fundTransferID) == "merchant fund transfer id" {
			continue // blank/banner row, trailing total, or repeated bilingual header
		}

		merchantName := ""
		if merchantNameColumn, ok := headers["merchant name"]; ok {
			merchantName = strings.TrimSpace(cell(row, merchantNameColumn))
		}
		currency := "VND"
		if currencyColumn, ok := headers["currency"]; ok {
			currency = strings.TrimSpace(cell(row, currencyColumn))
		}
		state := ""
		if hasState {
			state = strings.TrimSpace(cell(row, stateColumn))
		}

		detail := onePayFeeReportDetail{
			row:                    idx + 1,
			merchantID:             strings.TrimSpace(cell(row, headers["merchant id"])),
			merchantName:           merchantName,
			fundTransferID:         fundTransferID,
			opTransactionID:        strings.TrimSpace(cell(row, headers["op transaction id"])),
			currency:               currency,
			beneficiaryAccount:     strings.TrimSpace(cell(row, headers["beneficiary account"])),
			beneficiaryAccountName: strings.TrimSpace(cell(row, headers["beneficiary account name"])),
			beneficiaryBank:        strings.TrimSpace(cell(row, headers["beneficiary bank"])),
			amount:                 parseMoney(cell(row, headers["amount"])),
			state:                  state,
		}
		detail.createDate, _ = parseOnePayDate(cell(row, headers["create date"]))

		if previousRow, ok := seen[detail.fundTransferID]; ok {
			issues = append(issues, issue("duplicate_fund_transfer_id", detail.row, detail.fundTransferID, fmt.Sprintf("Trùng Merchant Fund Transfer ID với dòng %d", previousRow)))
		}
		seen[detail.fundTransferID] = detail.row
		if detail.opTransactionID == "" {
			issues = append(issues, issue("op_transaction_id_missing", detail.row, detail.fundTransferID, "Thiếu mã giao dịch OnePay"))
		}
		if !strings.EqualFold(detail.currency, "VND") {
			issues = append(issues, issue("currency_mismatch", detail.row, detail.fundTransferID, "Đơn vị tiền tệ không phải VND"))
		}
		if hasState && !strings.EqualFold(detail.state, "Approved") {
			issues = append(issues, issue("state_mismatch", detail.row, detail.fundTransferID, fmt.Sprintf("Trạng thái OnePay là %s, không phải Approved", detail.state)))
		}
		if detail.amount <= 0 {
			issues = append(issues, issue("amount_missing", detail.row, detail.fundTransferID, "Số tiền giao dịch phải lớn hơn 0"))
		}
		if !from.IsZero() && (detail.createDate.Before(from) || detail.createDate.After(to)) {
			issues = append(issues, issue("date_out_of_period", detail.row, detail.fundTransferID, "Ngày giao dịch nằm ngoài kỳ đối soát"))
		}
		totalAmount += detail.amount
		details = append(details, detail)
	}

	if len(details) == 0 {
		issues = append(issues, issue("no_detail_rows", 0, "", "Sheet chi tiết giao dịch không có giao dịch hợp lệ"))
	}
	return details, totalAmount, issues
}

func inferMerchant(details []onePayFeeReportDetail) (string, string) {
	if len(details) == 0 {
		return "", ""
	}
	return details[0].merchantID, details[0].merchantName
}

func findSheet(f *excelize.File, want string) string {
	for _, name := range f.GetSheetList() {
		if normalizeHeader(name) == normalizeHeader(want) {
			return name
		}
	}
	return ""
}

// findSummarySheet prefers the canonical summary sheet name and falls back to
// any sheet carrying the SLGD summary header, so renamed copies still parse.
func findSummarySheet(f *excelize.File) string {
	if name := findSheet(f, "PHI THANG"); name != "" {
		return name
	}
	for _, name := range f.GetSheetList() {
		if sheetPrefixHasCell(f, name, "SLGD") {
			return name
		}
	}
	return ""
}

func sheetPrefixHasCell(f *excelize.File, sheet, want string) bool {
	key := normalizeHeader(want)
	if key == "" {
		return false
	}
	rows, err := f.Rows(sheet)
	if err != nil {
		return false
	}
	defer func() { _ = rows.Close() }()
	for i := 0; i < detailHeaderScanRows && rows.Next(); i++ {
		cols, err := rows.Columns()
		if err != nil {
			return false
		}
		for _, c := range cols {
			if normalizeHeader(c) == key {
				return true
			}
		}
	}
	return false
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func normalizeHeader(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

func mustCellInt(f *excelize.File, sheet string, row, col int) int64 {
	cellName, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return 0
	}
	value, err := f.CalcCellValue(sheet, cellName)
	if err != nil || strings.TrimSpace(value) == "" {
		value, _ = f.GetCellValue(sheet, cellName)
	}
	return parseMoney(value)
}

func parseMoney(value string) int64 {
	cleaned := strings.NewReplacer(" ", "", "₫", "", "VND", "", "vnd", "").Replace(strings.TrimSpace(value))
	if cleaned == "" {
		return 0
	}
	if strings.Contains(cleaned, ".") && !strings.Contains(cleaned, ",") {
		parts := strings.Split(cleaned, ".")
		if len(parts[len(parts)-1]) == 3 {
			cleaned = strings.ReplaceAll(cleaned, ".", "")
		}
	}
	if strings.Contains(cleaned, ",") && !strings.Contains(cleaned, ".") {
		parts := strings.Split(cleaned, ",")
		if len(parts[len(parts)-1]) == 3 {
			cleaned = strings.ReplaceAll(cleaned, ",", "")
		} else {
			cleaned = strings.ReplaceAll(cleaned, ",", ".")
		}
	}
	if strings.Contains(cleaned, ",") && strings.Contains(cleaned, ".") {
		cleaned = strings.ReplaceAll(cleaned, ",", "")
	}
	n, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return int64(math.Round(n))
}

func parseVietnameseDateRange(s string) (time.Time, time.Time, bool) {
	re := regexp.MustCompile(`Từ ngày\s+(\d{2}/\d{2}/\d{4})\s+Đến ngày\s+(\d{2}/\d{2}/\d{4})`)
	match := re.FindStringSubmatch(s)
	if len(match) != 3 {
		return time.Time{}, time.Time{}, false
	}
	from, errFrom := time.Parse("02/01/2006", match[1])
	to, errTo := time.Parse("02/01/2006", match[2])
	return from, to, errFrom == nil && errTo == nil
}

func parseOnePayDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		"02-01-2006 03:04 PM",
		"02/01/2006 03:04 PM",
		"02/01/2006 15:04:05",
		"02-01-2006 15:04:05",
		"2006-01-02 15:04:05",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func normalizeText(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(strings.TrimSpace(s)), ""))
}

func normalizeName(s string) string {
	return normalizeText(unidecode.Unidecode(s))
}

func (s *OnePayFeeImportService) bankMatches(ctx context.Context, storedBank, reportBank string) bool {
	storedKey := s.canonicalBankKey(ctx, storedBank)
	reportKey := s.canonicalBankKey(ctx, reportBank)
	return storedKey != "" && reportKey != "" && storedKey == reportKey
}

func (s *OnePayFeeImportService) canonicalBankKey(ctx context.Context, value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	if s.bankRepo != nil {
		for _, candidate := range bankLookupCandidates(raw) {
			if bank := s.findBank(ctx, candidate); bank != nil {
				return bankKey(bank)
			}
		}
	}
	return normalizeText(raw)
}

func bankLookupCandidates(value string) []string {
	candidates := []string{value}
	ascii := unidecode.Unidecode(value)
	if normalizeText(ascii) != normalizeText(value) {
		candidates = append(candidates, ascii)
	}
	for _, candidate := range append([]string{}, candidates...) {
		mapped := bankmapping.MapName(candidate)
		if normalizeText(mapped) != normalizeText(candidate) {
			candidates = append(candidates, mapped)
		}
	}
	return candidates
}

func (s *OnePayFeeImportService) findBank(ctx context.Context, value string) *domain.Bank {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if bank, err := s.bankRepo.FindBySwiftCode(ctx, value); err == nil && bank != nil {
		return bank
	}
	if bank, err := s.bankRepo.FindByBankCode(ctx, value); err == nil && bank != nil {
		return bank
	}
	banks, err := s.bankRepo.SearchByBranchName(ctx, value, 5)
	if err != nil {
		return nil
	}
	for _, bank := range banks {
		if normalizeName(bank.BranchName) == normalizeName(value) {
			return bank
		}
	}
	return nil
}

func bankKey(bank *domain.Bank) string {
	if bank == nil {
		return ""
	}
	if strings.TrimSpace(bank.SwiftCode) != "" {
		return normalizeText(bank.SwiftCode)
	}
	if strings.TrimSpace(bank.BankCode) != "" {
		return normalizeText(bank.BankCode)
	}
	return normalizeName(bank.BranchName)
}

func issue(code string, row int, reference string, message string) dto.OnePayFeeReportIssue {
	return dto.OnePayFeeReportIssue{
		Code:      code,
		Row:       row,
		Reference: reference,
		Message:   message,
	}
}
