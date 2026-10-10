package services

import (
	"context"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"
)

// Pieces shared by the two weekly BCC processors (weekly BCC-* sheets and
// weekly payment tier sheets), extracted verbatim from their formerly
// copy-pasted bodies.

// weeklyRateResolver resolves the payrate config active on a given date,
// cached per day. A payrate can change mid-month; a weekly file's visible
// days (e.g. June 8-14) may fall under a different payrate than the 1st of
// the month, so rates are looked up per entry date, never at monthStart.
type weeklyRateResolver struct {
	s         *BCCImportService
	ctx       context.Context
	projectID uint
	cache     map[string]map[string]int
}

func newWeeklyRateResolver(s *BCCImportService, ctx context.Context, projectID uint) *weeklyRateResolver {
	return &weeklyRateResolver{s: s, ctx: ctx, projectID: projectID, cache: make(map[string]map[string]int)}
}

// forDate returns the flattened payrate active on d (nil when no payrate
// covers the date — cached as such).
func (r *weeklyRateResolver) forDate(d time.Time) map[string]int {
	key := d.Format("2006-01-02")
	if fr, ok := r.cache[key]; ok {
		return fr
	}
	var fr map[string]int
	if pr, err := r.s.payrateRepo.GetActiveByProjectAndDate(r.ctx, r.projectID, d); err == nil {
		if f, ferr := pr.Payrate.Flatten(); ferr == nil {
			fr = f
		}
	}
	r.cache[key] = fr // cache (nil if no payrate covers this date)
	return fr
}

// weeklySTKNameLookup builds the STK CCCD→Name map for cross-validation.
func weeklySTKNameLookup(stkRows []excelparser.STKRow) map[string]string {
	stkNameByCCCD := make(map[string]string, len(stkRows))
	for _, row := range stkRows {
		if row.CCCD != "" && row.FullName != "" {
			stkNameByCCCD[row.CCCD] = row.FullName
		}
	}
	return stkNameByCCCD
}

// weeklySTKCCCDLookup inverts the STK sheet into a normalized name→CCCD map, so
// a weekly row whose "Mã nhân viên" cell is blank can still be matched by the
// identifier the workbook itself carries. A name listed twice in STK maps to
// nothing: with duplicates there is no way to tell the two people apart, and
// guessing is exactly the hazard the CCCD column exists to prevent.
func weeklySTKCCCDLookup(stkRows []excelparser.STKRow) map[string]string {
	counts := make(map[string]int, len(stkRows))
	cccdByCCCDName := make(map[string]string, len(stkRows))
	for _, row := range stkRows {
		name := bccNormName(row.FullName)
		if name == "" || row.CCCD == "" {
			continue
		}
		counts[name]++
		cccdByCCCDName[name] = row.CCCD
	}
	lookup := make(map[string]string, len(cccdByCCCDName))
	for name, cccd := range cccdByCCCDName {
		if counts[name] == 1 {
			lookup[name] = cccd
		}
	}
	return lookup
}

// weeklyRowResolver maps a weekly sheet row to the project assignment it
// belongs to. Identifier precedence is deliberate:
//
//  1. the row's own "Mã nhân viên" (CCCD) cell,
//  2. the CCCD the workbook's STK sheet carries for that exact name
//     (weeklySTKCCCDLookup — the sheet's own identifier, so a stale or
//     differently-spelled assignment snapshot name still resolves),
//  3. the employee name, matched against the project's active assignments and
//     accepted only when exactly one assignment carries it (bccNameIndex).
//
// Step 3 is the last resort: a name is not an identifier, so a duplicate name
// (in the project or in STK) must fail loudly instead of posting hours against
// whichever employee happened to be found first.
type weeklyRowResolver struct {
	byCCCD        map[string]*domain.ProjectEmployee
	cccdBySTKName map[string]string
	byName        bccNameIndex
}

// newWeeklyRowResolver builds the resolver over the project's assignments as
// loaded/created for this import (byCCCD is the caller's map, mutated in place
// when the import auto-creates hires) plus the workbook's STK rows.
func newWeeklyRowResolver(
	byCCCD map[string]*domain.ProjectEmployee,
	assignments []*domain.ProjectEmployee,
	stkRows []excelparser.STKRow,
) weeklyRowResolver {
	return weeklyRowResolver{
		byCCCD:        byCCCD,
		cccdBySTKName: weeklySTKCCCDLookup(stkRows),
		byName:        newBCCNameIndex(assignments),
	}
}

// resolve returns the assignment for one row. ambiguous reports that the row
// carried no identifier and several active assignments share its name.
func (r weeklyRowResolver) resolve(employeeCode, fullName string) (assignment *domain.ProjectEmployee, ambiguous bool) {
	if employeeCode != "" {
		return r.byCCCD[employeeCode], false
	}
	if cccd := r.cccdBySTKName[bccNormName(fullName)]; cccd != "" {
		if assignment := r.byCCCD[cccd]; assignment != nil {
			return assignment, false
		}
	}
	return r.byName.lookup(fullName)
}

// weeklyCrossCheckSTKName rejects a BCC row whose CCCD maps to a different
// person in STK. Names are compared NFC-normalized, then diacritic-stripped
// (a single accent typo is data-entry noise, not a CCCD mismatch). suffix
// tailors the message per format (weekly BCC appends "— có thể sai CCCD").
// Returns nil when the row may proceed.
func weeklyCrossCheckSTKName(cccd, fullName string, stkNameByCCCD map[string]string, suffix string) *domain.ImportError {
	stkName, ok := stkNameByCCCD[cccd]
	if !ok || stkName == "" {
		return nil
	}
	bccNorm := bccNormName(fullName)
	stkNorm := bccNormName(stkName)
	if bccNorm != stkNorm && bccNormNameLoose(fullName) != bccNormNameLoose(stkName) {
		return &domain.ImportError{
			Employee: fullName,
			Reason:   "tên BCC (" + fullName + ") và tên STK (" + stkName + ") khác nhau cho cùng CCCD " + cccd + suffix,
		}
	}
	return nil
}
