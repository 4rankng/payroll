package settlement

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
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

	"github.com/gosimple/unidecode"
	"github.com/xuri/excelize/v2"
)

const onePayFeeParty = "OnePay"

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
	f, err := excelize.OpenReader(r)
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

	summarySheet := findSheet(f, "PHI THANG")
	detailSheet := findSheet(f, "GD")
	var issues []dto.OnePayFeeReportIssue
	if summarySheet == "" {
		issues = append(issues, issue("missing_sheet", 0, "PHI THANG", "Không tìm thấy sheet tổng hợp phí PHI THANG"))
	}
	if detailSheet == "" {
		issues = append(issues, issue("missing_sheet", 0, "GD", "Không tìm thấy sheet chi tiết giao dịch GD"))
	}
	if len(issues) > 0 {
		return nil, issues, nil
	}

	summary, summaryIssues := parseSummarySheet(f, summarySheet)
	issues = append(issues, summaryIssues...)
	details, detailTotalAmount, detailIssues := parseDetailSheet(f, detailSheet, summary.PeriodFrom, summary.PeriodTo)
	summary.DetailTotalAmount = detailTotalAmount
	issues = append(issues, detailIssues...)

	if len(details) != summary.TransactionCount {
		issues = append(issues, issue("transaction_count_mismatch", 0, "", fmt.Sprintf("Sheet GD có %d giao dịch, sheet PHI THANG ghi %d giao dịch", len(details), summary.TransactionCount)))
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

func parseDetailSheet(f *excelize.File, sheet string, periodFrom, periodTo string) ([]onePayFeeReportDetail, int64, []dto.OnePayFeeReportIssue) {
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return nil, 0, []dto.OnePayFeeReportIssue{issue("detail_read_failed", 0, "", "Không thể đọc sheet GD")}
	}

	headers := make(map[string]int)
	for i, h := range rows[0] {
		headers[normalizeHeader(h)] = i
	}
	// OnePay's monthly export has two valid layouts. Older exports include the
	// merchant-name and currency metadata; current VND-only exports omit both.
	// Keep every field used to reconcile an application payment mandatory.
	required := []string{"merchant id", "merchant fund transfer id", "op transaction id", "create date", "beneficiary account", "beneficiary account name", "beneficiary bank", "amount", "state"}
	var issues []dto.OnePayFeeReportIssue
	for _, h := range required {
		if _, ok := headers[h]; !ok {
			issues = append(issues, issue("missing_column", 1, h, fmt.Sprintf("Thiếu cột %s trong sheet GD", h)))
		}
	}
	if len(issues) > 0 {
		return nil, 0, issues
	}

	var from, to time.Time
	if periodFrom != "" && periodTo != "" {
		from, _ = time.Parse(timeutil.DateFormat, periodFrom)
		to, _ = time.Parse(timeutil.DateFormat, periodTo)
		to = to.Add(24*time.Hour - time.Nanosecond)
	}

	seen := map[string]int{}
	details := make([]onePayFeeReportDetail, 0, len(rows)-1)
	var totalAmount int64
	for idx := 1; idx < len(rows); idx++ {
		row := rows[idx]
		if strings.TrimSpace(cell(row, headers["merchant fund transfer id"])) == "" {
			continue
		}

		merchantName := ""
		if merchantNameColumn, ok := headers["merchant name"]; ok {
			merchantName = strings.TrimSpace(cell(row, merchantNameColumn))
		}
		currency := "VND"
		if currencyColumn, ok := headers["currency"]; ok {
			currency = strings.TrimSpace(cell(row, currencyColumn))
		}

		detail := onePayFeeReportDetail{
			row:                    idx + 1,
			merchantID:             strings.TrimSpace(cell(row, headers["merchant id"])),
			merchantName:           merchantName,
			fundTransferID:         strings.TrimSpace(cell(row, headers["merchant fund transfer id"])),
			opTransactionID:        strings.TrimSpace(cell(row, headers["op transaction id"])),
			currency:               currency,
			beneficiaryAccount:     strings.TrimSpace(cell(row, headers["beneficiary account"])),
			beneficiaryAccountName: strings.TrimSpace(cell(row, headers["beneficiary account name"])),
			beneficiaryBank:        strings.TrimSpace(cell(row, headers["beneficiary bank"])),
			amount:                 parseMoney(cell(row, headers["amount"])),
			state:                  strings.TrimSpace(cell(row, headers["state"])),
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
		if !strings.EqualFold(detail.state, "Approved") {
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
		issues = append(issues, issue("no_detail_rows", 0, "", "Sheet GD không có giao dịch hợp lệ"))
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
