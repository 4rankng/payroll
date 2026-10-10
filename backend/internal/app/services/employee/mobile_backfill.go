package employee

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/excelkit"
)

// MobileBackfillRow is one actionable data row from the "thiếu số điện
// thoại" workbook: match the employee by CCCD, fill in the mobile number.
type MobileBackfillRow struct {
	RowNumber int
	CCCD      string
	Mobile    string
}

// cleanMobile trims whitespace and strips the separators humans type into
// phone columns (spaces, dashes, dots, parentheses) so "0987 654 321" and
// "0987-654-321" both land as "0987654321". Anything else (leading "+",
// letters) survives and is rejected by validation below — the employee path
// deliberately stores plain digit strings without +84 normalization.
func cleanMobile(s string) string {
	r := strings.NewReplacer(" ", "", "-", "", ".", "", "(", "", ")", "")
	return strings.TrimSpace(r.Replace(s))
}

// ParseMobileBackfillExcel parses the workbook exported by
// /employees/export-paid-without-mobile after an admin filled the Mobile
// column in. Columns are located by normalized header ("cccd" / "mobile",
// "sđt" accepted as an alias), so extra or reordered columns are fine. Rows
// whose Mobile cell is blank are dropped silently — there is nothing to
// fill. Rows missing a CCCD come back as row errors instead.
func ParseMobileBackfillExcel(body []byte) ([]MobileBackfillRow, []dto.RowError, error) {
	f, err := excelkit.OpenReader(bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, fmt.Errorf("workbook has no sheets")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, nil, err
	}
	if len(rows) < 2 {
		return nil, nil, nil
	}

	cccdIdx, mobileIdx := -1, -1
	for i, cell := range rows[0] {
		switch excelkit.NormalizeHeader(cell) {
		case "cccd":
			if cccdIdx == -1 {
				cccdIdx = i
			}
		case "mobile", "sđt", "sdt":
			if mobileIdx == -1 {
				mobileIdx = i
			}
		}
	}
	if cccdIdx == -1 || mobileIdx == -1 {
		return nil, nil, fmt.Errorf("missing required columns: need 'CCCD' and 'Mobile'")
	}

	var parsed []MobileBackfillRow
	var rowErrs []dto.RowError
	for i, row := range rows[1:] {
		rowNumber := i + 2
		cell := func(idx int) string {
			if idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			return ""
		}
		cccd := cell(cccdIdx)
		mobile := cleanMobile(cell(mobileIdx))
		if cccd == "" && mobile == "" {
			continue // fully blank row
		}
		if cccd == "" {
			rowErrs = append(rowErrs, dto.RowError{RowNumber: rowNumber, Message: "Thiếu CCCD"})
			continue
		}
		if mobile == "" {
			continue // nothing to fill for this employee yet
		}
		parsed = append(parsed, MobileBackfillRow{RowNumber: rowNumber, CCCD: cccd, Mobile: mobile})
	}
	return parsed, rowErrs, nil
}

// mobileBackfillUpdate is one resolved write: set employees.mobile (currently
// empty) on this employee.
type mobileBackfillUpdate struct {
	EmployeeID uint
	Mobile     string
}

// classifyMobileBackfill resolves parsed rows against current data and
// returns the writes to perform plus the user-facing summary. Pure on
// purpose: the fill-only, duplicate-rejection and in-file-conflict rules are
// the same ones the admin applied by hand in the October 2026 pass, and they
// deserve direct unit tests without I/O.
//
//   - fill-only: an employee that already has a mobile is counted as
//     skipped_existing and never overwritten
//   - a mobile already carried by a DIFFERENT employee is a row error — this
//     import must not manufacture new duplicate numbers
//   - the same CCCD twice with different mobiles is a row error; with the
//     same mobile it is applied once
//   - the same employee reached through several rows is written once
func classifyMobileBackfill(
	rows []MobileBackfillRow,
	byCCCD map[string][]*domain.Employee,
	ownersByMobile map[string][]*domain.Employee,
) ([]mobileBackfillUpdate, *dto.MobileBackfillResult) {
	result := &dto.MobileBackfillResult{TotalRows: len(rows), Errors: []dto.RowError{}}
	filled := map[string]string{} // cccd -> mobile already applied by this file
	assigned := map[string]string{} // mobile -> cccd this file already gave it to
	seen := map[uint]bool{}
	var updates []mobileBackfillUpdate

	for _, row := range rows {
		if prev, ok := filled[row.CCCD]; ok && prev != row.Mobile {
			result.ErrorCount++
			result.Errors = append(result.Errors, dto.RowError{
				RowNumber: row.RowNumber,
				CCCD:      row.CCCD,
				Message:   "CCCD xuất hiện nhiều lần trong file với số điện thoại khác nhau",
			})
			continue
		}

		matches := byCCCD[row.CCCD]
		if len(matches) == 0 {
			result.NotFound++
			continue
		}

		if owners := ownersByMobile[row.Mobile]; len(owners) > 0 {
			ownedBySameCCCD := false
			for _, o := range owners {
				if o.CCCD == row.CCCD {
					ownedBySameCCCD = true
					break
				}
			}
			if !ownedBySameCCCD {
				result.ErrorCount++
				result.Errors = append(result.Errors, dto.RowError{
					RowNumber: row.RowNumber,
					CCCD:      row.CCCD,
					Message: fmt.Sprintf("SĐT %s đã được sử dụng bởi nhân viên CCCD %s — bỏ qua để tránh trùng số",
						row.Mobile, owners[0].CCCD),
				})
				continue
			}
		}

		// The DB-owner snapshot above cannot see numbers this same file
		// assigned moments ago — track those separately so one typo'd number
		// cannot be handed to two different people in a single upload.
		if prevCCCD, ok := assigned[row.Mobile]; ok && prevCCCD != row.CCCD {
			result.ErrorCount++
			result.Errors = append(result.Errors, dto.RowError{
				RowNumber: row.RowNumber,
				CCCD:      row.CCCD,
				Message: fmt.Sprintf("SĐT %s được điền cho nhiều nhân viên khác nhau trong file (CCCD %s và %s) — bỏ qua để tránh trùng số",
					row.Mobile, prevCCCD, row.CCCD),
			})
			continue
		}

		if err := (&domain.Employee{Mobile: row.Mobile}).ValidateMobile(); err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, dto.RowError{
				RowNumber: row.RowNumber,
				CCCD:      row.CCCD,
				Message:   "Số điện thoại không hợp lệ: chỉ chứa chữ số, tối đa 15 số, không phải CCCD",
			})
			continue
		}

		for _, e := range matches {
			if e.Mobile != "" {
				result.SkippedExisting++
				continue
			}
			if seen[e.ID] {
				continue
			}
			seen[e.ID] = true
			updates = append(updates, mobileBackfillUpdate{EmployeeID: e.ID, Mobile: row.Mobile})
			result.Updated++
		}
		filled[row.CCCD] = row.Mobile
		assigned[row.Mobile] = row.CCCD
	}
	return updates, result
}

// BackfillMobiles applies a parsed mobile-backfill workbook: fill-only
// writes of employees.mobile for rows matched by CCCD, all inside one
// transaction, returning the classification summary. Row errors never abort
// the batch — same partial-success semantics as the employee import.
func (s *EmployeeService) BackfillMobiles(ctx context.Context, rows []MobileBackfillRow, parseErrors []dto.RowError) (*dto.MobileBackfillResult, error) {
	cccdSet := make(map[string]bool, len(rows))
	mobileSet := make(map[string]bool, len(rows))
	cccds := make([]string, 0, len(rows))
	mobiles := make([]string, 0, len(rows))
	for _, r := range rows {
		if !cccdSet[r.CCCD] {
			cccdSet[r.CCCD] = true
			cccds = append(cccds, r.CCCD)
		}
		if !mobileSet[r.Mobile] {
			mobileSet[r.Mobile] = true
			mobiles = append(mobiles, r.Mobile)
		}
	}

	employees, err := s.EmployeeRepo.ListByCCCDs(ctx, cccds)
	if err != nil {
		return nil, fmt.Errorf("list employees by cccd: %w", err)
	}
	owners, err := s.EmployeeRepo.ListByMobiles(ctx, mobiles)
	if err != nil {
		return nil, fmt.Errorf("list employees by mobile: %w", err)
	}

	byCCCD := make(map[string][]*domain.Employee, len(employees))
	for _, e := range employees {
		byCCCD[e.CCCD] = append(byCCCD[e.CCCD], e)
	}
	ownersByMobile := make(map[string][]*domain.Employee, len(owners))
	for _, e := range owners {
		ownersByMobile[e.Mobile] = append(ownersByMobile[e.Mobile], e)
	}

	updates, result := classifyMobileBackfill(rows, byCCCD, ownersByMobile)

	// Fold parser-level errors (missing CCCD) into the same summary.
	result.TotalRows += len(parseErrors)
	for _, pe := range parseErrors {
		result.ErrorCount++
		result.Errors = append(result.Errors, pe)
	}

	if len(updates) > 0 {
		if err := s.TransactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
			for _, u := range updates {
				if err := s.EmployeeRepo.UpdateColumns(txCtx, u.EmployeeID, map[string]any{"mobile": u.Mobile}); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("%s: %w", constants.MsgFailedToBackfillMobilesVN, err)
		}
	}

	// Employee list/count caches are TTL-based on this service's write paths
	// (UpdateEmployee does not invalidate either); the list refreshes within
	// the same TTL after a backfill.
	return result, nil
}
