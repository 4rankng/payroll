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
