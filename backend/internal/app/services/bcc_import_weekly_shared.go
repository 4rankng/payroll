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
