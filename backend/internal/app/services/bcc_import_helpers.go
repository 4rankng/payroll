package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

func marshalErrors(errs []domain.ImportError) *string {
	if len(errs) == 0 {
		return nil
	}
	b, err := json.Marshal(errs)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

// FirstErrorReason extracts the first error reason from a JSON error detail string.
func FirstErrorReason(detail *string) string {
	if detail == nil {
		return "unknown"
	}
	var errs []domain.ImportError
	if err := json.Unmarshal([]byte(*detail), &errs); err != nil || len(errs) == 0 {
		return *detail
	}
	return errs[0].Reason
}

func parseForMonth(forMonth string) (int, time.Month, error) {
	t, err := clock.ParseMonth(forMonth)
	if err != nil {
		return 0, 0, fmt.Errorf("định dạng tháng không hợp lệ: %q: %w", forMonth, err)
	}
	if t.Year() < 2000 || t.Year() > 2100 {
		return 0, 0, fmt.Errorf("năm không hợp lệ: %d", t.Year())
	}
	return t.Year(), t.Month(), nil
}

// deducePosition determines the best position for auto-created employees by matching
// BCC shift rates against the project's payrate configuration.
// Payrate paths are "position.dayType.hourType" → rate. We find which position
// has the most matching rates with the BCC shift rates.
func deducePosition(flatRates map[string]int, shiftRates map[string]int64) string {
	if len(flatRates) == 0 || len(shiftRates) == 0 {
		return "phổ thông"
	}

	// Collect unique BCC rate values (the VND amounts from row 10 of BCC sheet).
	bccRates := make(map[int]bool, len(shiftRates))
	for _, r := range shiftRates {
		if r > 0 {
			bccRates[int(r)] = true
		}
	}
	if len(bccRates) == 0 {
		return "phổ thông"
	}

	// For each position, count how many of its payrate values match BCC rates.
	positionHits := make(map[string]int)
	for path, rate := range flatRates {
		if rate == 0 {
			continue
		}
		if !bccRates[rate] {
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) < 1 || parts[0] == "" {
			continue
		}
		positionHits[parts[0]]++
	}

	if len(positionHits) == 0 {
		return "phổ thông"
	}

	// Pick the position with the most matching rates.
	best := "phổ thông"
	bestCount := 0
	for pos, count := range positionHits {
		if count > bestCount {
			bestCount = count
			best = pos
		}
	}
	return best
}

// shiftLabelHourType maps a rateless BCC shift label to a payrate hour type.
// Rateless templates (e.g. Samsung SDS) carry no VND rate row; their columns
// only distinguish regular hours (CB, CN, HC) from overtime (OT, OT CN).
// Regular labels are matched as whole tokens — substring matching would
// mis-bucket compact codes from other templates (e.g. "TCN" = tăng ca đêm)
// as regular hours and silently underpay them. Returns false for labels
// outside the vocabulary so unknown columns still surface the missing-rate
// error instead of being silently mis-mapped.
func shiftLabelHourType(label string) (string, bool) {
	v := strings.ToUpper(strings.TrimSpace(label))
	if v == "" {
		return "", false
	}
	if strings.Contains(v, "OT") {
		return "tăng ca", true
	}
	for _, tok := range strings.Fields(v) {
		switch tok {
		case "CB", "CN", "HC":
			return "ca ngày", true
		}
	}
	return "", false
}

// flatRatesHaveBucket reports whether any flattened payrate path carries the
// given (dayType, hourType) combination, regardless of position. The rateless
// label fallback uses it so unknown buckets surface the standard missing-rate
// row error instead of a generic bulk-create failure downstream.
func flatRatesHaveBucket(flatRates map[string]int, dayType, hourType string) bool {
	wantDay := canonicalBCCRateKeySegment(dayType)
	wantHour := canonicalBCCRateKeySegment(hourType)
	for path, rate := range flatRates {
		if rate == 0 {
			// A bucket explicitly configured at 0 VND is equivalent to absent
			// (the legacy rate path skips zero rates) — surface the standard
			// missing-rate error instead of passing the import gate.
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) != 3 {
			continue
		}
		if canonicalBCCRateKeySegment(parts[1]) == wantDay &&
			canonicalBCCRateKeySegment(parts[2]) == wantHour {
			return true
		}
	}
	return false
}

// applySTKBankFields populates the bank-related fields on an Employee being
// created from an STK row. It is intentionally tolerant of incomplete STK
// data: bank info is only attached when we can build a *complete* banking
// record (bank id + account number + account name). Otherwise all bank fields
// are left empty and the employee is created without banking info — it can be
// filled in later via the admin UI or a subsequent BCC upload.
//
// This avoids the all-or-nothing rejection that previously surfaced as
// "bank selection is required when providing banking information" when the
// STK sheet had an account number but no bank name (and therefore no
// resolvable BankID). Creating the profile and the timesheet is more
// valuable than enforcing partial bank-info integrity at import time.
//
// fullName is used to synthesize the BankAccountName when banking info is
// complete (matches the historical convention of upper-casing the full name).
// Note: EmployeeService.CreateEmployee re-normalizes BankAccountName to
// Vietnamese title case before persisting, so the uppercased value set here
// is not what reaches the database — it is kept only to preserve the prior
// inline behavior at this layer.
func applySTKBankFields(emp *domain.Employee, row excelparser.STKRow, bankID *uint, fullName string) {
	emp.BankID = nil
	emp.BankAccountNumber = ""
	emp.BankAccountName = ""

	// Need at least an account number AND a resolvable bank to proceed.
	// Without a bank id, the row's account number is untrustworthy on its own
	// (we cannot tell which institution it belongs to), so we skip bank fields
	// entirely rather than persisting a half-populated record.
	if bankID == nil || row.BankAccount == "" {
		return
	}

	emp.BankID = bankID
	emp.BankAccountNumber = row.BankAccount
	emp.BankAccountName = strings.ToUpper(strings.TrimSpace(fullName))
}

// buildSTKBankUpdates returns a complete targeted update when an STK row
// changes an existing employee's bank account, bank, or account-holder name.
// Empty account numbers remain non-authoritative. A supplied but unresolved
// bank clears the stale bank relation so the account cannot be checked or paid
// against the wrong institution.
func buildSTKBankUpdates(
	emp *domain.Employee,
	row excelparser.STKRow,
	bankID *uint,
	fullName string,
) map[string]any {
	if emp == nil {
		return nil
	}

	accountNumber := strings.TrimSpace(row.BankAccount)
	if accountNumber == "" {
		return nil
	}
	accountName := strings.ToUpper(strings.TrimSpace(fullName))
	bankNameProvided := strings.TrimSpace(row.BankName) != ""
	bankChanged := bankNameProvided &&
		((bankID == nil && emp.BankID != nil) ||
			(bankID != nil && (emp.BankID == nil || *emp.BankID != *bankID)))
	accountChanged := strings.TrimSpace(emp.BankAccountNumber) != accountNumber
	nameChanged := !strings.EqualFold(strings.TrimSpace(emp.BankAccountName), accountName)
	if !bankChanged && !accountChanged && !nameChanged {
		return nil
	}

	updates := map[string]any{
		"bank_account_number": accountNumber,
		"bank_account_name":   accountName,
	}
	if bankNameProvided && bankID == nil {
		// A supplied but unknown bank is authoritative evidence that the old
		// bank must not be reused for the new account. Clear the relation so
		// the employee appears in the unified warning list for correction.
		updates["bank_id"] = nil
	} else if bankID != nil {
		updates["bank_id"] = *bankID
	}
	return updates
}

// rateTarget is the (dayType, hourType) payrate bucket a BCC entry resolves to.
type rateTarget struct{ dayType, hourType string }

// dayTypePriority orders payrate day types when several buckets could serve
// the same rate or label: normal days first, then rest days, then holidays.
var dayTypePriority = map[string]int{
	"ngày thường": 0, "thường": 0,
	"ngày nghỉ": 1, "nghỉ": 1,
	"ngày lễ": 2, "lễ": 2,
}

// labelRateTarget resolves a shift label directly against payrate hour-type
// leaves (label-keyed model): the file's column codes must exist in the
// config, and the rate is determined purely by the label. Position is
// intentionally ignored — single-position projects do not key rates by
// assignment position. When several day types carry the same label,
// dayTypePriority picks the winner (ngày thường first); equal-priority ties
// (synonym day types, multi-position configs) resolve to the smallest day
// type string so the result never depends on map iteration order.
func labelRateTarget(flatRates map[string]int, label string) (rateTarget, bool) {
	want := canonicalBCCRateKeySegment(label)
	if want == "" {
		return rateTarget{}, false
	}
	found := false
	best := rateTarget{}
	bestPri := 0
	for path, rate := range flatRates {
		if rate == 0 {
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) != 3 {
			continue
		}
		if canonicalBCCRateKeySegment(parts[2]) != want {
			continue
		}
		pri, known := dayTypePriority[parts[1]]
		if !known {
			continue
		}
		if !found || pri < bestPri || (pri == bestPri && parts[1] < best.dayType) {
			best = rateTarget{parts[1], parts[2]}
			bestPri = pri
			found = true
		}
	}
	return best, found
}

// dateRowEmployeeToSTKRows converts date-row template employees into STK rows
// so the existing employee upsert block (create-if-missing + bank refresh +
// project assignment) runs unchanged for the new template.
func dateRowEmployeeToSTKRows(employees []excelparser.BCCEmployeeData) []excelparser.STKRow {
	rows := make([]excelparser.STKRow, 0, len(employees))
	for _, emp := range employees {
		rows = append(rows, excelparser.STKRow{
			CCCD:        emp.CCCD,
			FullName:    emp.FullName,
			BankAccount: emp.BankAccount,
			BankName:    emp.BankName,
			Mobile:      emp.Mobile,
		})
	}
	return rows
}

// mergeSTKRows folds rows parsed from a real STK sheet into the BCC-derived
// rows. STK is the bank-dedicated sheet, so for the same CCCD its account,
// bank, and mobile win over the BCC sheet's columns; STK-only rows (hires
// absent from the BCC sheet) are appended so they are created and assigned.
func mergeSTKRows(base, stk []excelparser.STKRow) []excelparser.STKRow {
	idx := make(map[string]int, len(base))
	for i, r := range base {
		if r.CCCD != "" {
			idx[r.CCCD] = i
		}
	}
	for _, r := range stk {
		if r.CCCD == "" {
			continue
		}
		if i, ok := idx[r.CCCD]; ok {
			if r.BankAccount != "" {
				base[i].BankAccount = r.BankAccount
			}
			if r.BankName != "" {
				base[i].BankName = r.BankName
			}
			if r.Mobile != "" {
				base[i].Mobile = r.Mobile
			}
			continue
		}
		idx[r.CCCD] = len(base)
		base = append(base, r)
	}
	return base
}

// earliestInMonthDay returns the smallest day-of-month carrying hours inside
// the import month, or 0 when the file has no in-month entries. Date-row
// files straddle months (e.g. 21/08–24/09): out-of-month days must not feed
// the payrate probe, which targets a day of the import month.
func earliestInMonthDay(employees []excelparser.BCCEmployeeData, year int, month time.Month) int {
	earliest := 0
	for _, emp := range employees {
		for _, e := range emp.Entries {
			if e.DayNum <= 0 {
				continue
			}
			if e.FullDate != nil && (e.FullDate.Year() != year || e.FullDate.Month() != month) {
				continue
			}
			if earliest == 0 || e.DayNum < earliest {
				earliest = e.DayNum
			}
		}
	}
	return earliest
}

// resolveBCCEntryTarget maps one imported entry to its payrate bucket. Order:
// the file's own rate row first (rate is king); then label-keyed resolution —
// only for date-row templates, whose column codes name payrate shift leaves;
// then the rateless calendar fallback (label regular/overtime + calendar day
// type). labelKeyed must be false for legacy files: their CN means "ca ngày"
// while a date-row config's CN leaf means Chủ Nhật, so label-first would
// silently re-price old templates. The calendar fallback applies to rateless
// templates only — rate-carrying files must fail loudly on an unknown rate.
func resolveBCCEntryTarget(
	shiftRates map[string]int64,
	rateToTarget map[int]rateTarget,
	flatRates map[string]int,
	label string,
	date time.Time,
	rateless, labelKeyed bool,
) (rateTarget, bool) {
	rate := int(shiftRates[label])
	if target, ok := rateToTarget[rate]; ok {
		return target, true
	}
	if labelKeyed {
		if cand, found := labelRateTarget(flatRates, label); found {
			return cand, true
		}
	}
	if rateless {
		if hourType, labelOK := shiftLabelHourType(label); labelOK {
			cand := rateTarget{dayType: determineDayType(date), hourType: hourType}
			if flatRatesHaveBucket(flatRates, cand.dayType, cand.hourType) {
				return cand, true
			}
		}
	}
	return rateTarget{}, false
}
