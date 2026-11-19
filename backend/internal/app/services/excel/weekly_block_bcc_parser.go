package excel

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Weekly block BCC template family: a calendar-period sheet organized in
// week blocks (CN→T7) in which one header row carries the identity headers
// AND one merged date cell per day; below it a weekday row and a category
// label row; data rows follow. Each day spans a 2-4 sub-column block whose
// label-row cells carry the project's payrate labels ("HC 100%", "OT 150%",
// "Đêm 130%", "Đêm 200%", "CN 200%", "CN 270%", "Lễ 300%", "Lễ 390%"); a
// summary block headed "Tổng hợp" starts right of the day region and must
// never be parsed as days. The sheet name is arbitrary (partners ship it as
// "BCC " with a trailing space), so the content fingerprint below — not the
// sheet name — claims it for FormatWeeklyBCC.
//
// The file is rateless — its labels name payrate leaves priced by the weekly
// BCC pipeline (buildWeeklyBCCEntries: HourType = label, rate key
// (position, ngày thường)). Parsing emits one WeeklyBCCSheetData group per
// distinct label with ≥ 1 entry; ShiftType is the label verbatim and entries
// carry the block anchor date.

// weeklyBlockHeaderScanMaxCol caps the identity-header search width: STT /
// Mã nhân viên / Họ và tên always sit in the leftmost columns of the sheet.
const weeklyBlockHeaderScanMaxCol = 10

// weeklyBlockRosterMaxRows bounds the employee walk below the label row
// (multi-position convention: 500 rows of headroom).
const weeklyBlockRosterMaxRows = 500

// weeklyBlockRosterMaxBlankRows tolerates blank separator rows inside the
// roster before the scan stops (multi-position convention).
const weeklyBlockRosterMaxBlankRows = 4

// weeklyBlockCanonicalLabels is the canonical row-of-labels set
// (normHeader-folded) — the labels partner payrate configs key leaves by.
// The fingerprint requires ≥ 2 of them on one row; parsing itself passes any
// non-empty label through (rate resolution errors downstream if the config
// lacks it).
var weeklyBlockCanonicalLabels = map[string]bool{
	"hc 100%":  true,
	"ot 150%":  true,
	"đêm 130%": true,
	"đêm 200%": true,
	"cn 200%":  true,
	"cn 270%":  true,
	"lễ 300%":  true,
	"lễ 390%":  true,
}

// weeklyBlockLayout is the detected geometry of one weekly block sheet,
// shared by the fingerprint (isWeeklyBlockBCCSheet) and the parser so a
// sheet accepted here always parses — the isDateRowBCCSheet →
// findDateRowAndCols pattern.
type weeklyBlockLayout struct {
	sttCol         int
	empCodeCol     int // "Mã nhân viên" — carries the citizen ID (CCCD) in this family
	nameCol        int // "Họ và tên"
	labelRow       int
	firstAnchorCol int
	stopCol        int // first "tổng…" header cell right of the day region ("Tổng hợp")
	anchors        map[int]time.Time
}

// findWeeklyBlockLayout scans rows 1..12 for the header row: one row
// carrying STT + Mã nhân viên + Họ và tên (normHeader contains-match) AND
// ≥ 3 full-date cells (raw serials, 2020–2040 guard) right of the name
// column. Legacy day-number serials (1–31) fail the date guard, so a legacy
// day-number grid can never match. The stop column is the first "tổng…"
// cell right of the last date anchor; the label row is the first row below
// the header with ≥ 2 canonical category labels inside the day region.
//
// Scans run to wbccMaxColScan (200), not dateRowBCCScanMax (96): real files
// run the day region out to col EA (131) with the summary block at EB (132).
func findWeeklyBlockLayout(f *excelize.File, sheet string) (weeklyBlockLayout, error) {
	for headerRow := 1; headerRow <= 12; headerRow++ {
		var sttCol, empCodeCol, nameCol int
		for col := 1; col <= weeklyBlockHeaderScanMaxCol; col++ {
			val := normHeader(bccCell(f, sheet, col, headerRow))
			if val == "" {
				continue
			}
			if sttCol == 0 && strings.Contains(val, "stt") {
				sttCol = col
			}
			if empCodeCol == 0 && strings.Contains(val, "mã nhân viên") {
				empCodeCol = col
			}
			if nameCol == 0 && strings.Contains(val, "họ và tên") {
				nameCol = col
			}
		}
		if sttCol == 0 || empCodeCol == 0 || nameCol == 0 {
			continue
		}

		anchors := make(map[int]time.Time)
		for col := nameCol + 1; col <= wbccMaxColScan; col++ {
			cn, err := excelize.CoordinatesToCellName(col, headerRow)
			if err != nil {
				break
			}
			val, err := f.GetCellValue(sheet, cn, excelize.Options{RawCellValue: true})
			if err != nil || val == "" {
				continue
			}
			if t, ok := parseExcelDate(strings.TrimSpace(val)); ok {
				anchors[col] = t
			}
		}
		if len(anchors) < 3 {
			continue
		}
		firstAnchor, lastAnchor := wbccMaxColScan, 0
		for col := range anchors {
			if col < firstAnchor {
				firstAnchor = col
			}
			if col > lastAnchor {
				lastAnchor = col
			}
		}

		stopCol := 0
		for col := lastAnchor + 1; col <= wbccMaxColScan; col++ {
			if strings.Contains(normHeader(bccCell(f, sheet, col, headerRow)), "tổng") {
				stopCol = col
				break
			}
		}
		if stopCol == 0 {
			continue
		}

		labelRow := 0
		for row := headerRow + 1; row <= headerRow+3; row++ {
			canonical := 0
			for col := firstAnchor; col < stopCol; col++ {
				if weeklyBlockCanonicalLabels[normHeader(bccCell(f, sheet, col, row))] {
					canonical++
					if canonical >= 2 {
						break
					}
				}
			}
			if canonical >= 2 {
				labelRow = row
				break
			}
		}
		if labelRow == 0 {
			continue
		}

		return weeklyBlockLayout{
			sttCol:         sttCol,
			empCodeCol:     empCodeCol,
			nameCol:        nameCol,
			labelRow:       labelRow,
			firstAnchorCol: firstAnchor,
			stopCol:        stopCol,
			anchors:        anchors,
		}, nil
	}
	return weeklyBlockLayout{}, fmt.Errorf("không tìm thấy dòng tiêu đề khối tuần trong sheet")
}

// isWeeklyBlockBCCSheet reports whether a sheet matches the weekly block
// fingerprint. It delegates to findWeeklyBlockLayout so detection and
// parsing share one definition of the layout — a sheet accepted here always
// parses.
func isWeeklyBlockBCCSheet(f *excelize.File, sheet string) bool {
	_, err := findWeeklyBlockLayout(f, sheet)
	return err == nil
}

// parseWeeklyBlockBCCSheet parses one weekly block sheet into
// WeeklyBCCSheetData groups — one per distinct label row cell that carries
// ≥ 1 entry, in first-appearance order. The "Mã nhân viên" column holds the
// citizen ID (WeeklyBCCEmployeeData.EmployeeCode). Entries under unlabeled
// spare sub-columns (enhanced-Saturday blocks leave 1st/3rd cells blank in
// the template) are skipped, with one warning per column holding hours > 0
// so the partner can be told those hours are not imported.
func parseWeeklyBlockBCCSheet(f *excelize.File, sheet string) ([]WeeklyBCCSheetData, error) {
	lay, err := findWeeklyBlockLayout(f, sheet)
	if err != nil {
		return nil, err
	}

	warnedSpareCols := make(map[int]bool)

	// Groups in first-appearance order. Labels are few (the canonical eight
	// plus strays), so a plain slice with a linear lookup beats a map plus a
	// separate ordering slice.
	var groups []WeeklyBCCSheetData

	blankStreak := 0
	for row := lay.labelRow + 1; row <= lay.labelRow+weeklyBlockRosterMaxRows; row++ {
		stt := bccCell(f, sheet, lay.sttCol, row)
		cccd := bccCell(f, sheet, lay.empCodeCol, row)
		name := bccCell(f, sheet, lay.nameCol, row)

		if stt == "" && cccd == "" && name == "" {
			blankStreak++
			if blankStreak > weeklyBlockRosterMaxBlankRows {
				break
			}
			continue
		}
		blankStreak = 0
		if isSummaryRow(name, stt) {
			break
		}
		if cccd == "" && name == "" {
			// A bare STT number with no identity is roster noise.
			continue
		}

		// cur maps label → index of this roster row's employee inside that
		// group, so a row contributing to several labels appends one
		// employee per group. Index-based: safe across slice reallocation.
		cur := make(map[string]int)

		// Walk the day region left to right. A sub-column's date is its
		// block anchor — the nearest merged date cell at or to its left —
		// so a 2-4 sub-column day block inherits its anchor the way the
		// date-row family inherits col-1. Unlabeled columns are skipped
		// before the hour read; explicit 0-hour cells are kept (deletion
		// requests), blank cells skipped.
		currentDate := time.Time{}
		for col := lay.firstAnchorCol; col < lay.stopCol; col++ {
			if d, ok := lay.anchors[col]; ok {
				currentDate = d
			}
			if currentDate.IsZero() {
				continue
			}
			label := bccCell(f, sheet, col, lay.labelRow)
			if label == "" {
				if hours, present := dateRowNumericCell(f, sheet, col, row); present && hours > 0 && !warnedSpareCols[col] {
					warnedSpareCols[col] = true
					slog.Warn("weekly_block_bcc: hours under an unlabeled spare sub-column are not imported",
						"sheet", sheet, "col", col, "row", row, "hours", hours)
				}
				continue
			}
			hours, present := dateRowNumericCell(f, sheet, col, row)
			if !present {
				continue
			}
			gi := slices.IndexFunc(groups, func(g WeeklyBCCSheetData) bool { return g.ShiftType == label })
			if gi < 0 {
				groups = append(groups, WeeklyBCCSheetData{ShiftType: label})
				gi = len(groups) - 1
			}
			empIdx, ok := cur[label]
			if !ok {
				groups[gi].Employees = append(groups[gi].Employees, WeeklyBCCEmployeeData{
					EmployeeCode: cccd,
					FullName:     name,
				})
				empIdx = len(groups[gi].Employees) - 1
				cur[label] = empIdx
			}
			groups[gi].Employees[empIdx].Entries = append(groups[gi].Employees[empIdx].Entries, WeeklyBCCEntryData{
				Date:  currentDate,
				Hours: hours,
			})
		}
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("không tìm thấy dữ liệu nhân viên trong sheet")
	}
	return groups, nil
}
