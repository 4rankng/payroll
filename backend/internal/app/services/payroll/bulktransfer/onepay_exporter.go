package bulktransfer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"github.com/xuri/excelize/v2"
)

// OnePay sheet/column layout — matches what the wallet_bulk.YeuCauChuyenTienParser
// expects. Header row 2, data row 3+. 7 columns.
const (
	onePaySheetName     = "eMB_BulkPayment"
	onePayHeaderRow     = 2
	onePayFirstDataRow  = 3
	onePayColWidthSwift = 16
)

// onePayHeaderVN is the bilingual header the exporter writes — every alias
// in wallet_bulk.headerAliases must resolve to one of these cells.
var onePayHeaders = []string{
	"STT\n(Ord. No.)",
	"Số tài khoản\n(Account No.)",
	"Tên người thụ hưởng\n(Beneficiary)",
	"Ngân hàng thụ hưởng/Chi nhánh\n(Beneficiary Bank)",
	"Mã SWIFT\n(SWIFT Code)",
	"Số tiền\n(Amount)",
	"Nội dung chuyển khoản\n(Payment Detail)",
}

// SkippedEmployeeReason constants surfaced to the frontend.
const (
	skipReasonMissingBankInfo = "missing_bank_info"
	skipReasonUnresolvedSwift = "unresolved_swift"
)

// OnePayExporter generates a "Yêu cầu chuyển tiền" .xlsx in OnePay-API-
// compatible eMB_BulkPayment format, enriched with a SWIFT code column
// resolved from each employee's bank record. Each row gets a fresh VFIC
// transaction code (planned by ExportPlanner, persisted here via
// transactionCodeRepo.CreateBatch).
//
// The output file is the canonical input for the wallet-page upload pipeline
// (Phase 3+). The wallet-page parser reads SWIFT directly from column G,
// avoiding any bank-name → SWIFT resolution at upload time.
type OnePayExporter struct {
	planner             *ExportPlanner
	bankRepo            domain.BankRepository
	transactionCodeRepo domain.TransactionCodeRepository
	clock               func() time.Time
	logger              *slog.Logger
}

// OnePayExportRow is one output row of the .xlsx (7 columns).
type OnePayExportRow struct {
	OrderNo       int
	AccountNo     string
	AccountName   string
	Bank          string // Vietnamese display name for KQ column D
	SwiftCode     string // resolved from employee.bank.bank_code → Bank.SwiftCode
	Amount        int64  // VND
	PaymentDetail string // contains the VFIC code
	VFICCode      string
}

// OnePaySkippedEmployee records an employee the export could not include
// (missing bank_id, empty bank account, or unresolved SWIFT). Surfaced to
// the admin via the X-Skipped-Count header and a detail dialog.
type OnePaySkippedEmployee struct {
	EmployeeID   uint   `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	ProjectID    uint   `json:"project_id,omitempty"`
	ProjectName  string `json:"project_name,omitempty"`
	Reason       string `json:"reason"`
}

// OnePayExportResult is the return shape of Export. ExcelBytes is the .xlsx
// blob (NOT serialized to JSON when sent over the wire — the handler writes
// it as a binary body with metadata in X- headers).
type OnePayExportResult struct {
	ExcelBytes       []byte
	Filename         string
	Cycle            string
	TotalCount       int
	TransferAmount   int64
	SkippedEmployees []OnePaySkippedEmployee
	// TransactionCodes persists the VFIC → (employee, project, timesheets)
	// mapping for later audit resolution (links bulk rows back to source).
	TransactionCodes []*domain.TransactionCode
}

// NewOnePayExporter wires the exporter. planner MUST already be constructed
// with its read-side dependencies (timesheet/employee/project repos). The
// clock defaults to clock.Now if nil.
func NewOnePayExporter(
	planner *ExportPlanner,
	bankRepo domain.BankRepository,
	transactionCodeRepo domain.TransactionCodeRepository,
	logger *slog.Logger,
) *OnePayExporter {
	if logger == nil {
		logger = slog.Default()
	}
	return &OnePayExporter{
		planner:             planner,
		bankRepo:            bankRepo,
		transactionCodeRepo: transactionCodeRepo,
		clock:               clock.Now,
		logger:              logger,
	}
}

// Export runs the full Stage-1 pipeline: plan → resolve SWIFT for each row →
// persist VFIC codes → generate the .xlsx. Employees missing bank info are
// skipped (not failed) and surfaced in SkippedEmployees.
//
// Returns the bytes, filename, and bookkeeping metadata. The handler streams
// bytes as a blob + metadata in custom X- headers.
func (e *OnePayExporter) Export(ctx context.Context, req *dto.ExportBulkTransferRequest) (*OnePayExportResult, error) {
	plan, err := e.planner.Plan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("onepay export: plan: %w", err)
	}
	if plan.ValidatedData == nil || plan.ValidatedData.ValidData == nil {
		// No timesheet matched the cohort (approved + pending-payment) for the
		// requested cycle/range. This is a user-correctable condition, not a
		// server fault — surface as EMPTY_RESULT (HTTP 422) so the admin gets
		// actionable guidance instead of a generic 500.
		cycle := ""
		if plan != nil {
			cycle = plan.Cycle
		}
		return nil, domain.NewEmptyResultError(
			fmt.Sprintf("Không có timesheet nào hợp lệ để xuất cho chu kỳ %s. Vui lòng kiểm tra trạng thái duyệt/thanh toán và khoảng ngày rồi thử lại.", cycleLabelVN(cycle)),
		)
	}
	data := plan.ValidatedData.ValidData

	rows, skipped, txnCodes, err := e.buildRowsWithSwift(ctx, data, plan.Cycle, plan.FromDate, plan.ToDate, plan.PaymentPercentage)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		// Every candidate was filtered out. Surface the three skip buckets so
		// the admin can tell what to fix: (a) pre-validation missing account
		// number/name, (b) missing bank linkage, (c) unresolved SWIFT.
		return nil, domain.NewEmptyResultError(buildEmptySwiftMessage(plan, skipped))
	}

	// Persist VFIC codes BEFORE generating the workbook so we don't surface a
	// download link whose transaction codes weren't recorded. Mirrors the
	// existing 9Pay export pattern (export_service.go:205).
	if len(txnCodes) > 0 {
		if err := e.transactionCodeRepo.CreateBatch(ctx, txnCodes); err != nil {
			return nil, fmt.Errorf("onepay export: persist transaction codes: %w", err)
		}
	}

	excelBytes, err := GenerateOnePayExcel(rows)
	if err != nil {
		return nil, fmt.Errorf("onepay export: generate excel: %w", err)
	}

	var totalAmount int64
	for _, r := range rows {
		totalAmount += r.Amount
	}

	return &OnePayExportResult{
		ExcelBytes:       excelBytes,
		Filename:         buildOnePayFilename(plan.Cycle, e.clock()),
		Cycle:            plan.Cycle,
		TotalCount:       len(rows),
		TransferAmount:   totalAmount,
		SkippedEmployees: skipped,
		TransactionCodes: txnCodes,
	}, nil
}

// buildRowsWithSwift iterates the planned EmployeeProjectAmounts in deterministic
// order, looks up each employee's bank → SWIFT, and produces:
//   - []OnePayExportRow for the workbook (skipping employees missing bank info)
//   - []OnePaySkippedEmployee for the warning list
//   - []*domain.TransactionCode for transactionCodeRepo.CreateBatch
//
// SWIFT resolution: employee.bank.bank_code → BankRepository.FindByBankCode → .SwiftCode.
// Two skip cases: (a) no bank_id / empty bank_account → missing_bank_info;
// (b) bank exists but SwiftCode is empty → unresolved_swift.
func (e *OnePayExporter) buildRowsWithSwift(
	ctx context.Context,
	data *excel.BulkTransferData,
	cycle string,
	fromDate, toDate time.Time,
	paymentPercentage float64,
) ([]OnePayExportRow, []OnePaySkippedEmployee, []*domain.TransactionCode, error) {
	if math.IsNaN(paymentPercentage) || math.IsInf(paymentPercentage, 0) || paymentPercentage <= 0 || paymentPercentage > 1 {
		return nil, nil, nil, fmt.Errorf("OnePay payment percentage must be finite and between 0 and 1")
	}
	// Deterministic iteration order: by (employee_id, project_id) ascending.
	keys := make([]excel.EmployeeProjectKey, 0, len(data.EmployeeProjectAmounts))
	for k := range data.EmployeeProjectAmounts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].EmployeeID != keys[j].EmployeeID {
			return keys[i].EmployeeID < keys[j].EmployeeID
		}
		return keys[i].ProjectID < keys[j].ProjectID
	})

	// Cache bank lookups — the same employee's bank_id appears once per row.
	bankCache := make(map[uint]*domain.Bank)

	rows := make([]OnePayExportRow, 0, len(keys))
	skipped := make([]OnePaySkippedEmployee, 0)
	txnCodes := make([]*domain.TransactionCode, 0, len(keys))

	for i, key := range keys {
		grossAmount := data.EmployeeProjectAmounts[key]
		if grossAmount <= 0 {
			continue
		}
		// Match the bank-file exporter: use the percentage captured by the
		// plan and round down to whole VND exactly once per employee/project.
		calculated := float64(grossAmount) * paymentPercentage
		if calculated < 1 || calculated >= float64(math.MaxInt64) {
			return nil, nil, nil, fmt.Errorf("OnePay transfer amount for employee %d is outside the supported VND range", key.EmployeeID)
		}
		amount := int64(calculated)
		emp, ok := data.EmployeeData[key.EmployeeID]
		if !ok {
			e.logger.Warn("onepay export: employee data missing", "employee_id", key.EmployeeID)
			continue
		}
		project := data.ProjectData[key.ProjectID]
		timesheetIDs := data.EmployeeProjectTimesheets[key]
		vfic := data.TransactionCodes[key]

		// Skip case (a): missing bank linkage or empty account.
		if emp.Bank == nil || emp.Bank.BankCode == "" || emp.BankAccountNumber == "" {
			skipped = append(skipped, OnePaySkippedEmployee{
				EmployeeID:   emp.ID,
				EmployeeName: emp.Fullname,
				ProjectID:    project.ID,
				ProjectName:  project.Name,
				Reason:       skipReasonMissingBankInfo,
			})
			continue
		}

		// Resolve the canonical bank (data.EmployeeData[id].Bank is the slim
		// projection; the banks table has SwiftCode).
		bank, ok := bankCache[emp.Bank.ID]
		if !ok {
			resolved, err := e.bankRepo.FindByBankCode(ctx, emp.Bank.BankCode)
			if err != nil {
				e.logger.Warn("onepay export: bank lookup failed",
					"employee_id", emp.ID, "bank_code", emp.Bank.BankCode, "error", err)
				skipped = append(skipped, OnePaySkippedEmployee{
					EmployeeID:   emp.ID,
					EmployeeName: emp.Fullname,
					ProjectID:    project.ID,
					ProjectName:  project.Name,
					Reason:       skipReasonUnresolvedSwift,
				})
				continue
			}
			bank = resolved
			bankCache[emp.Bank.ID] = resolved
		}

		// Skip case (b): bank exists but no SWIFT.
		if bank == nil || bank.SwiftCode == "" {
			skipped = append(skipped, OnePaySkippedEmployee{
				EmployeeID:   emp.ID,
				EmployeeName: emp.Fullname,
				ProjectID:    project.ID,
				ProjectName:  project.Name,
				Reason:       skipReasonUnresolvedSwift,
			})
			continue
		}

		row := OnePayExportRow{
			OrderNo:     i + 1, // 1-indexed after skips (KQ display order)
			AccountNo:   emp.BankAccountNumber,
			AccountName: emp.BankAccountName,
			Bank:        bank.BranchName,
			SwiftCode:   bank.SwiftCode,
			Amount:      amount,
			// PaymentDetail MUST be the bare VFIC code only.
			//
			// This string flows: xlsx column G → wallet_bulk parser →
			// WalletPayment.Description → OnePay FundsTransferRequest.Remark.
			//
			// OnePay rejects remarks containing hyphens (mirrored by our sandbox
			// mock at sandbox/onepay/handler.go:hasHyphen). A verbose format
			// like "CT VFICxxx Lâm Văn Bách Hilex - KCN nomura" picks up the
			// project name's hyphen and OnePay returns INVALID_PARAMETERS.
			//
			// Reconciliation already keys off the VFIC code (regex
			// `VFIC[0-9a-f]+` in wallet_bulk/parser.go + bulk_transfer_batches
			// lookup) — including the name/project adds no matching value and
			// actively breaks the transfer. Employee + project context is
			// preserved via the TransactionCode row (buildTransactionCode).
			PaymentDetail: vfic,
			VFICCode:      vfic,
		}
		rows = append(rows, row)

		txnCodes = append(txnCodes, buildTransactionCode(vfic, emp, project, timesheetIDs, amount, cycle, fromDate, toDate))
	}

	return rows, skipped, txnCodes, nil
}

// buildTransactionCode constructs one TransactionCode row linking the VFIC
// back to its source timesheets so the worker (Phase 3) can resolve the
// employee + project for notification purposes. Mirrors export_service.go:172.
// fromDate/toDate populate CyclePayData so the history view resolves the cycle
// without a timesheet fetch (Phase A, red-team M1).
func buildTransactionCode(vfic string, emp excel.Employee, project excel.Project, timesheetIDs []uint, amount int64, cycle string, fromDate, toDate time.Time) *domain.TransactionCode {
	cycleNum := 1
	if cycle != string(domain.PaymentScheduleMonthly) {
		cycleNum = clock.KyFromWorkDay(fromDate.Day())
	}
	fd, td := fromDate, toDate
	var tcData domain.TransactionCodeData
	if cycle == string(domain.PaymentScheduleMonthly) {
		tcData = domain.TransactionCodeData{
			MonthlyPay: &domain.CyclePayData{
				TimesheetIDs:           timesheetIDs,
				EmployeeID:             emp.ID,
				ProjectID:              project.ID,
				Amount:                 amount,
				TransferAmountSnapshot: &amount,
				FromDate:               &fd,
				ToDate:                 &td,
				CycleNum:               cycleNum,
			},
		}
	} else {
		tcData = domain.TransactionCodeData{
			WeeklyPay: &domain.CyclePayData{
				TimesheetIDs:           timesheetIDs,
				EmployeeID:             emp.ID,
				ProjectID:              project.ID,
				Amount:                 amount,
				TransferAmountSnapshot: &amount,
				FromDate:               &fd,
				ToDate:                 &td,
				CycleNum:               cycleNum,
			},
		}
	}
	tcBytes, _ := json.Marshal(tcData)
	return &domain.TransactionCode{
		Code: vfic,
		Data: tcBytes,
	}
}

// buildOnePayFilename matches the plan's spec: Yeu_cau_chuyen_tien_<cycle>_<YYYYMMDD_HHMMSS>.xlsx
func buildOnePayFilename(cycle string, now time.Time) string {
	cycle = strings.TrimSpace(strings.ToLower(cycle))
	if cycle == "" {
		cycle = "chuyen"
	}
	stamp := now.Format("20060102_150405")
	return fmt.Sprintf("Yeu_cau_chuyen_tien_%s_%s.xlsx", cycle, stamp)
}

// cycleLabelVN returns a Vietnamese label for a cycle string used in
// user-facing messages. Falls back to a neutral label when the cycle
// is empty or unrecognized.
func cycleLabelVN(cycle string) string {
	switch strings.TrimSpace(strings.ToLower(cycle)) {
	case string(domain.PaymentScheduleWeekly):
		return "tuần"
	case string(domain.PaymentScheduleMonthly):
		return "tháng"
	default:
		return "đã chọn"
	}
}

// buildEmptySwiftMessage composes the EMPTY_RESULT message for the
// post-SWIFT-resolution zero-rows case. It distinguishes three skip buckets
// so the admin can tell what to fix:
//   - preValidationSkipped: planner's ValidateAndFilterBulkTransferData dropped
//     these (missing account number or name) — surfaced via plan.ValidatedData.SkippedCount
//   - missingBankInfo:      no bank_id / empty account — surfaced via the
//     buildRowsWithSwift skipped slice (skipReasonMissingBankInfo)
//   - unresolvedSwift:      bank exists but SwiftCode empty / lookup failed —
//     surfaced via the buildRowsWithSwift skipped slice (skipReasonUnresolvedSwift)
func buildEmptySwiftMessage(plan *ExportPlan, skipped []OnePaySkippedEmployee) string {
	var missingBank, unresolvedSwift int
	for _, s := range skipped {
		switch s.Reason {
		case skipReasonMissingBankInfo:
			missingBank++
		case skipReasonUnresolvedSwift:
			unresolvedSwift++
		}
	}
	preValidation := 0
	if plan != nil && plan.ValidatedData != nil {
		preValidation = plan.ValidatedData.SkippedCount
	}
	return fmt.Sprintf(
		"Không có dòng nào xuất được sau khi đối chiếu SWIFT. "+
			"%d nhân viên thiếu thông tin ngân hàng (chưa liên kết ngân hàng/số tài khoản), "+
			"%d nhân viên chưa có mã SWIFT trên hệ thống, "+
			"%d nhân viên bị bỏ qua trước đó do thiếu số tài khoản hoặc tên thụ hưởng.",
		missingBank, unresolvedSwift, preValidation,
	)
}

// GenerateOnePayExcel builds the .xlsx workbook from scratch (no template
// dependency — header-name mapping in the parser handles layout variations).
// Sheet `eMB_BulkPayment`, header row 2, data row 3+, 7 columns.
func GenerateOnePayExcel(rows []OnePayExportRow) ([]byte, error) {
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	// Rename default Sheet1 → eMB_BulkPayment (avoids leaving an empty Sheet1).
	if idx, _ := f.GetSheetIndex("Sheet1"); idx == 0 {
		if err := f.SetSheetName("Sheet1", onePaySheetName); err != nil {
			return nil, fmt.Errorf("set sheet name: %w", err)
		}
	} else if _, err := f.NewSheet(onePaySheetName); err != nil {
		return nil, fmt.Errorf("new sheet: %w", err)
	}

	// Header row 2.
	for colIdx, h := range onePayHeaders {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, onePayHeaderRow)
		if err := f.SetCellStyle(onePaySheetName, cell, cell, onePayHeaderStyle(f)); err != nil {
			return nil, fmt.Errorf("set header style %s: %w", cell, err)
		}
		if err := f.SetCellValue(onePaySheetName, cell, h); err != nil {
			return nil, fmt.Errorf("set header %s: %w", cell, err)
		}
	}

	// Data rows starting at row 3.
	for offset, row := range rows {
		excelRow := onePayFirstDataRow + offset
		cells := []struct {
			col string
			val any
		}{
			{"A", row.OrderNo},
			{"B", row.AccountNo},
			{"C", row.AccountName},
			{"D", row.Bank},
			{"E", row.SwiftCode},
			{"F", row.Amount},
			{"G", row.PaymentDetail},
		}
		for _, c := range cells {
			cell := fmt.Sprintf("%s%d", c.col, excelRow)
			if err := f.SetCellValue(onePaySheetName, cell, c.val); err != nil {
				return nil, fmt.Errorf("set cell %s: %w", cell, err)
			}
		}
	}

	// Column widths — SWIFT codes are 8-11 chars, amounts can be large.
	widths := map[string]float64{
		"A": 6, "B": 22, "C": 28, "D": 36,
		"E": float64(onePayColWidthSwift), "F": 18, "G": 40,
	}
	for col, w := range widths {
		if err := f.SetColWidth(onePaySheetName, col, col, w); err != nil {
			return nil, fmt.Errorf("set col width %s: %w", col, err)
		}
	}

	// Keep worksheet dimension aligned with data (Excel/other readers depend on it).
	lastCol := "G"
	lastRow := onePayFirstDataRow + len(rows) - 1
	if lastRow < onePayHeaderRow {
		lastRow = onePayHeaderRow
	}
	if err := f.SetSheetDimension(onePaySheetName, fmt.Sprintf("A1:%s%d", lastCol, lastRow)); err != nil {
		return nil, fmt.Errorf("set sheet dimension: %w", err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write xlsx: %w", err)
	}
	return buf.Bytes(), nil
}

// onePayHeaderStyle returns a bold-style handle for the header row. Errors
// here are non-fatal — the cell still renders; we just lose bolding.
func onePayHeaderStyle(f *excelize.File) int {
	style, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#E2E8F0"},
		},
	})
	if err != nil {
		return 0
	}
	return style
}

// SkippedEmployeesAsJSON marshals the slice into a compact JSON string for
// persistence into bulk_transfer_files.data when the caller wants an audit
// trail of who was skipped and why.
func SkippedEmployeesAsJSON(s []OnePaySkippedEmployee) string {
	if len(s) == 0 {
		return "[]"
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// SkippedCount helper for handlers that only need the count.
func SkippedCount(s []OnePaySkippedEmployee) int { return len(s) }

// TotalCount helper (string form for X-Total-Count header).
func TotalCount(n int) string { return strconv.Itoa(n) }
