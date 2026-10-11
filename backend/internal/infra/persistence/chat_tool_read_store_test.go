package persistence

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestChatPeriodMonthsNormalisation(t *testing.T) {
	now := time.Date(2026, 10, 15, 0, 0, 0, 0, time.Local)

	cases := []struct {
		name     string
		from, to string
		wantOK   bool
		wantFrom string
		wantTo   string
	}{
		{name: "both empty means current month", wantOK: true, wantFrom: "2026-10", wantTo: "2026-10"},
		{name: "full year range", from: "2026-01", to: "2026-12", wantOK: true, wantFrom: "2026-01", wantTo: "2026-12"},
		{name: "inverted bounds are swapped", from: "2026-12", to: "2026-01", wantOK: true, wantFrom: "2026-01", wantTo: "2026-12"},
		{name: "open start bound", to: "2026-05", wantOK: true, wantFrom: "2026-05", wantTo: "2026-05"},
		{name: "open end bound", from: "2026-05", wantOK: true, wantFrom: "2026-05", wantTo: "2026-05"},
		{name: "malformed year-only rejected", from: "2026", to: "2026-12"},
		{name: "month out of range rejected", from: "2026-13"},
		{name: "garbage rejected", from: "abc", to: "xyz"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			label, lo, hi, ok := chatPeriodMonths(tc.from, tc.to, now)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				if label != "" || lo != "" || hi != "" {
					t.Fatalf("rejected period must return empty bounds, got %q/%q/%q", label, lo, hi)
				}
				return
			}
			if lo != tc.wantFrom || hi != tc.wantTo {
				t.Fatalf("bounds = [%s,%s], want [%s,%s]", lo, hi, tc.wantFrom, tc.wantTo)
			}
			// The label must name the window so the tool copy can restate it.
			if !strings.Contains(label, tc.wantFrom) {
				t.Fatalf("label %q does not name the scanned window %s", label, tc.wantFrom)
			}
		})
	}
}

// The curated report catalog is the ONLY SQL the chatbot can reach. Assert the
// allow-list shape and, most importantly, that no statement exposes a secrets
// or identity column — settings.value holds the MiMo API key, employees hold
// CCCD and bank numbers.
func TestCuratedReportsAreSafe(t *testing.T) {
	specs := CuratedReportSpecs()
	if len(specs) == 0 {
		t.Fatal("report catalog is empty — run_readonly_report would refuse everything")
	}

	banned := []string{
		"password", "cccd", "bank", "settings", "secret", "token",
		"api_key", "apikey", "hash", "national_id", "phone", "email",
	}

	for key, entry := range curatedChatReports {
		sql := strings.ToLower(entry.SQL)
		for _, b := range banned {
			if strings.Contains(sql, b) {
				t.Errorf("report %q references banned identifier %q", key, b)
			}
		}
		// Every statement must carry the scope placeholder so the caller's
		// scope predicate is substituted at runtime — never a fixed 1=1.
		if !strings.Contains(entry.SQL, chatScopeCondPlaceholder) {
			t.Errorf("report %q has no scope placeholder — it would run unscoped", key)
		}
		// Guard against someone "fixing" a report by concatenating arguments.
		if strings.Contains(entry.SQL, "%s") {
			t.Errorf("report %q uses printf-style interpolation", key)
		}
	}
}

func TestCuratedReportSpecsAreDeterministicallyOrdered(t *testing.T) {
	a := CuratedReportSpecs()
	b := CuratedReportSpecs()
	if len(a) != len(b) {
		t.Fatal("spec count differs between calls")
	}
	for i := range a {
		if a[i].Key != b[i].Key {
			t.Fatalf("spec order not stable at %d: %s vs %s", i, a[i].Key, b[i].Key)
		}
		if i > 0 && a[i-1].Key >= a[i].Key {
			t.Fatalf("specs not sorted: %s before %s", a[i-1].Key, a[i].Key)
		}
	}
}

// The operational list statements are just as reachable by the model as the
// curated reports, so they carry the same safety bar: a scope placeholder
// (never a hardcoded 1=1) and no projected secret/PII column.
func TestOperationalListSQLIsScopedAndSafe(t *testing.T) {
	statements := map[string]string{
		"missing_timesheets":     chatSQLMissingTimesheets,
		"advance_limit_breaches": chatSQLAdvanceLimitBreaches,
		"missing_bank_details":   chatSQLMissingBankDetails,
	}

	for name, sql := range statements {
		if !strings.Contains(sql, chatScopeCondPlaceholder) {
			t.Errorf("%s has no scope placeholder — it would run unscoped", name)
		}
		if strings.Contains(sql, "%s") {
			t.Errorf("%s uses printf-style interpolation", name)
		}
	}

	// MissingBankDetails may READ bank columns inside CASE expressions to count
	// what is blank — that only ever yields 1/0. What it must never do is
	// PROJECT the account number itself, which would let the chatbot echo a
	// live bank credential back to the user.
	bank := strings.ToLower(chatSQLMissingBankDetails)
	selectPart := bank[strings.Index(bank, "select"):strings.Index(bank, "from employees")]
	for _, leak := range []string{
		"bank_account_number,",
		"bank_account_number as",
		"as bank_account_number",
		"bank_account_name,",
		"bank_account_name as",
	} {
		if strings.Contains(selectPart, leak) {
			t.Errorf("missing_bank_details projects a bank credential (%q)", leak)
		}
	}
	if !strings.Contains(bank, "e.bank_account_number") {
		t.Error("missing_bank_details should still detect a blank account number")
	}
	if !strings.Contains(selectPart, "case when e.bank_account_number = ''") {
		t.Error("bank columns must only be read inside CASE expressions")
	}
}

func TestChatMonthStartFallsBackToCurrentMonth(t *testing.T) {
	now := time.Date(2026, 10, 15, 0, 0, 0, 0, time.Local)

	if got := chatMonthStart("2026-03", now); got.Format("2006-01-02") != "2026-03-01" {
		t.Errorf("valid month: got %s", got.Format("2006-01-02"))
	}
	for _, bad := range []string{"", "2026", "abc", "2026-13"} {
		if got := chatMonthStart(bad, now); got.Format("2006-01-02") != "2026-10-01" {
			t.Errorf("month %q should fall back to current month, got %s", bad, got.Format("2006-01-02"))
		}
	}
}

func TestScanChatListRejectsEmptyScope(t *testing.T) {
	// Defence in depth: an empty predicate must abort rather than run unscoped.
	s := &ChatToolReadStore{}
	if _, _, err := s.scanChatList(context.Background(), chatSQLMissingTimesheets, "", nil); err == nil {
		t.Fatal("scanChatList must refuse an empty scope predicate")
	}
}
