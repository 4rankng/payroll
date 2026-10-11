package llmchat

import (
	"regexp"
	"strings"
)

// Go-side port of frontend/src/utils/import-errors.ts — the semantic source of
// truth for import error explanations. The chat error-diagnosis tool serves
// this catalog so chat answers and the upload UI always agree. Matching order
// and wording mirror the TS file; keep them in lockstep when either changes.

var (
	technicalValuePattern = regexp.MustCompile(`(?i)^(?:id\s+\d+)\b|\b(?:onepay|9pay|provider|error|failed|internal|timeout|employee_id)\b|[_{}[\]=]`)
	isoDatePattern        = regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
	namedShiftPattern     = regexp.MustCompile(`(ca làm|vị trí)\s+"([^"]{1,40})"\s+không có trong cấu hình lương`)
)

// CatalogEntry is one error category as exposed by the lookup_import_error
// tool: stable code, the shared Vietnamese explanation, and a fix hint (the
// Go-side extension over the FE catalog).
type CatalogEntry struct {
	Code        string   `json:"code"`
	Explanation string   `json:"explanation"`
	FixHint     string   `json:"fix_hint"`
	Aliases     []string `json:"aliases"`
}

// ImportCatalog is the full error catalog. Order mirrors the match order in
// getSafeImportErrorReason.
var ImportCatalog = []CatalogEntry{
	{
		Code:        "approved",
		Explanation: "Bảng chấm công đã được phê duyệt",
		FixHint:     "Dòng đã duyệt không sửa được qua import — nếu cần chỉnh, gửi yêu cầu chỉnh sửa bảng công.",
		Aliases:     []string{"phê duyệt", "đã duyệt", "approved"},
	},
	{
		Code:        "duplicate",
		Explanation: "Dữ liệu đã tồn tại",
		FixHint:     "Bỏ qua dòng trùng, hoặc kiểm tra file đã import trước đó trước khi import lại.",
		Aliases:     []string{"trùng", "đã có", "tồn tại", "duplicate"},
	},
	{
		Code:        "employee_not_found",
		Explanation: "Không tìm thấy nhân viên",
		FixHint:     "Đối chiếu họ tên và số điện thoại trong file với danh sách nhân viên trước khi import lại.",
		Aliases:     []string{"không tìm thấy nhân viên"},
	},
	{
		Code:        "employee_create_failed",
		Explanation: "Không thể tạo hồ sơ nhân viên",
		FixHint:     "Kiểm tra dữ liệu bắt buộc của nhân viên (họ tên, số điện thoại) hoặc tạo nhân viên trước khi import.",
		Aliases:     []string{"không thể tạo nhân viên"},
	},
	{
		Code:        "payrate_missing_shift",
		Explanation: "Chưa cấu hình mức lương phù hợp cho ca làm việc",
		FixHint:     "Thêm mức lương cho ca làm việc trong bảng lương của dự án, hoặc sửa tên ca trong file theo bảng lương.",
		Aliases:     []string{"không tìm thấy mức lương"},
	},
	{
		Code:        "payrate_missing_named",
		Explanation: "Ca làm hoặc vị trí chưa có trong bảng lương của dự án",
		FixHint:     "Thêm ca làm/vị trí vào bảng lương của dự án, hoặc sửa tên ca/vị trí trong file cho khớp cấu hình.",
		Aliases:     []string{"cấu hình lương"},
	},
	{
		Code:        "payrate_missing",
		Explanation: "Chưa cấu hình bảng lương cho dự án",
		FixHint:     "Tạo bảng lương (mức lương theo ca/vị trí) cho dự án trước khi import.",
		Aliases:     []string{"bảng lương", "payrate"},
	},
	{
		Code:        "future_date",
		Explanation: "Ngày chấm công chưa đến",
		FixHint:     "Chỉ import các ngày đã diễn ra; kiểm tra lại cột ngày trong file.",
		Aliases:     []string{"ngày trong tương lai", "ngày chấm công chưa đến"},
	},
	{
		Code:        "paid",
		Explanation: "Bảng chấm công đã thanh toán",
		FixHint:     "Dòng đã thanh toán không thể sửa qua import — liên hệ quản trị nếu cần điều chỉnh.",
		Aliases:     []string{"thanh toán", "paid"},
	},
	{
		Code:        "timesheet_create_failed",
		Explanation: "Không thể tạo bảng chấm công",
		FixHint:     "Kiểm tra dữ liệu dòng (ngày, ca, số giờ) rồi import lại.",
		Aliases:     []string{"không thể tạo bảng chấm công"},
	},
	{
		Code:        "missing_required",
		Explanation: "Thiếu dữ liệu bắt buộc",
		FixHint:     "Bổ sung các cột bắt buộc còn trống trong file.",
		Aliases:     []string{"thiếu"},
	},
	{
		Code:        "invalid",
		Explanation: "Dữ liệu không hợp lệ",
		FixHint:     "Kiểm tra định dạng ngày, số giờ và số tiền trong file.",
		Aliases:     []string{"không hợp lệ"},
	},
	{
		Code:        "out_of_month",
		Explanation: "Dữ liệu nằm ngoài tháng đang thao tác",
		FixHint:     "Giữ nguyên thông báo từ hệ thống — file chứa ngày ngoài tháng được chọn.",
		Aliases:     []string{"ngoài tháng"},
	},
	{
		Code:        "unknown",
		Explanation: "Không thể xử lý dòng dữ liệu này",
		FixHint:     "Kiểm tra kỹ dòng được nêu, hoặc gửi file cho quản trị kiểm tra.",
		Aliases:     nil,
	},
}

// SafeImportErrorReason normalises a raw import error reason to its stable
// category message — the port of getSafeImportErrorReason. The "ngoài tháng"
// case passes the raw reason through (it already names the data range).
func SafeImportErrorReason(reason string) string {
	normalized := strings.ToLower(strings.TrimSpace(reason))

	switch {
	case strings.Contains(normalized, "phê duyệt"),
		strings.Contains(normalized, "đã duyệt"),
		strings.Contains(normalized, "approved"):
		return "Bảng chấm công đã được phê duyệt"
	case strings.Contains(normalized, "trùng"),
		strings.Contains(normalized, "đã có"),
		strings.Contains(normalized, "tồn tại"),
		strings.Contains(normalized, "duplicate"):
		return "Dữ liệu đã tồn tại"
	case strings.Contains(normalized, "không tìm thấy") && strings.Contains(normalized, "nhân viên"):
		return "Không tìm thấy nhân viên"
	case strings.Contains(normalized, "không thể tạo nhân viên"):
		return "Không thể tạo hồ sơ nhân viên"
	case strings.Contains(normalized, "không tìm thấy mức lương"):
		return "Chưa cấu hình mức lương phù hợp cho ca làm việc"
	case strings.Contains(normalized, "cấu hình lương"):
		if named := namedShiftPattern.FindStringSubmatch(normalized); named != nil {
			label := strings.TrimSpace(named[2])
			if !technicalValuePattern.MatchString(label) {
				noun := named[1]
				return strings.ToUpper(noun[:1]) + noun[1:] + " \"" + label + "\" chưa có trong bảng lương của dự án"
			}
		}
		return "Chưa cấu hình bảng lương cho dự án"
	case strings.Contains(normalized, "bảng lương"),
		strings.Contains(normalized, "payrate"):
		return "Chưa cấu hình bảng lương cho dự án"
	case strings.Contains(normalized, "ngày trong tương lai"),
		strings.Contains(normalized, "ngày chấm công chưa đến"):
		return "Ngày chấm công chưa đến"
	case strings.Contains(normalized, "thanh toán"),
		strings.Contains(normalized, "paid"):
		return "Bảng chấm công đã thanh toán"
	case strings.Contains(normalized, "không thể tạo bảng chấm công"):
		return "Không thể tạo bảng chấm công"
	case strings.Contains(normalized, "thiếu"):
		return "Thiếu dữ liệu bắt buộc"
	case strings.Contains(normalized, "không hợp lệ"):
		return "Dữ liệu không hợp lệ"
	case strings.Contains(normalized, "ngoài tháng"):
		return strings.TrimSpace(reason)
	}
	return "Không thể xử lý dòng dữ liệu này"
}

// SafeEmployeeLabel sanitises an employee name from error detail before it is
// echoed anywhere (prompt, chat answer): technical garbage is dropped so no
// raw internal identifier leaks to the user.
func SafeEmployeeLabel(employee string) string {
	normalized := strings.TrimSpace(employee)
	if normalized == "" || len(normalized) > 120 || technicalValuePattern.MatchString(normalized) {
		return ""
	}
	return normalized
}

// ExtractErrorDate returns the first ISO date in a reason formatted DD/MM/YYYY.
func ExtractErrorDate(reason string) string {
	m := isoDatePattern.FindStringSubmatch(reason)
	if m == nil {
		return ""
	}
	return m[3] + "/" + m[2] + "/" + m[1]
}

// ImportErrorItem is one normalised import error row.
type ImportErrorItem struct {
	Row      int    `json:"row"`
	Employee string `json:"employee"`
	Reason   string `json:"reason"`
	Date     string `json:"date,omitempty"`
}

// ParseImportErrors parses the JSON-encoded error_detail string stored on the
// import asset. Invalid or missing detail yields an empty slice — an import
// summary must never fail because the audit detail is malformed.
func ParseImportErrors(detail string) []ImportErrorItem {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return nil
	}
	// Lazy decode of [{row?, employee?, reason}] — mirror the TS flatMap with
	// its strict reason check.
	var raw []map[string]any
	if err := jsonUnmarshal(detail, &raw); err != nil {
		return nil
	}
	out := make([]ImportErrorItem, 0, len(raw))
	for _, value := range raw {
		reason, ok := value["reason"].(string)
		if !ok {
			continue
		}
		item := ImportErrorItem{Reason: SafeImportErrorReason(reason)}
		if row, ok := value["row"].(float64); ok {
			item.Row = int(row)
		}
		if employee, ok := value["employee"].(string); ok {
			item.Employee = SafeEmployeeLabel(employee)
		}
		item.Date = ExtractErrorDate(reason)
		out = append(out, item)
	}
	return out
}

// GroupedImportError collapses identical (employee, reason) errors into one
// summary line — the port of groupImportErrors.
type GroupedImportError struct {
	Employee string   `json:"employee"`
	Reason   string   `json:"reason"`
	Count    int      `json:"count"`
	Dates    []string `json:"dates"`
	Rows     []int    `json:"rows"`
}

const detailListCap = 5

// GroupImportErrors groups parsed errors per (employee, reason).
func GroupImportErrors(errors []ImportErrorItem) []GroupedImportError {
	type key struct{ employee, reason string }
	groups := map[key]*GroupedImportError{}
	order := []key{}
	for _, e := range errors {
		k := key{e.Employee, e.Reason}
		g, ok := groups[k]
		if !ok {
			g = &GroupedImportError{Employee: e.Employee, Reason: e.Reason}
			groups[k] = g
			order = append(order, k)
		}
		g.Count++
		if e.Date != "" && !containsString(g.Dates, e.Date) {
			g.Dates = append(g.Dates, e.Date)
		}
		if e.Row > 0 && !containsInt(g.Rows, e.Row) {
			g.Rows = append(g.Rows, e.Row)
		}
	}
	out := make([]GroupedImportError, 0, len(order))
	for _, k := range order {
		out = append(out, *groups[k])
	}
	return out
}

// DescribeGroupedError builds the context suffix for one group, e.g.
// " (các ngày 25/06/2026, 26/06/2026)". Lists are capped at 5.
func DescribeGroupedError(g GroupedImportError) string {
	parts := []string{}
	switch {
	case len(g.Dates) > 0:
		shown, truncated := joinCapped(g.Dates, detailListCap)
		label := "ngày"
		if len(g.Dates) > 1 {
			label = "các ngày"
		}
		parts = append(parts, label+" "+shown+truncatedSuffix(truncated))
	case len(g.Rows) > 0:
		shown, truncated := joinCapped(intSliceToStrings(g.Rows), detailListCap)
		label := "dòng"
		if len(g.Rows) > 1 {
			label = "các dòng"
		}
		parts = append(parts, label+" "+shown+truncatedSuffix(truncated))
	}
	if g.Count > 1 && (len(g.Dates) > detailListCap || (len(g.Dates) == 0 && len(g.Rows) == 0)) {
		parts = append(parts, itoa(g.Count)+" dòng")
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// LookupImportCatalog returns catalog entries matching a keyword (code,
// explanation, fix hint or alias). Empty keyword returns the whole catalog —
// bounded by construction (the catalog is small and fixed).
func LookupImportCatalog(keyword string) []CatalogEntry {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return ImportCatalog
	}
	out := []CatalogEntry{}
	for _, entry := range ImportCatalog {
		haystack := strings.ToLower(entry.Code + " " + entry.Explanation + " " + entry.FixHint + " " + strings.Join(entry.Aliases, " "))
		if strings.Contains(haystack, keyword) {
			out = append(out, entry)
		}
	}
	return out
}
