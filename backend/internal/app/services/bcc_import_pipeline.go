package services

import (
	"context"
	"strings"
	"time"

	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"
	"api-server/internal/pkg/clock"
)

// Shared pipeline pieces for the per-format BCC processors (legacy/date-row,
// weekly BCC, weekly payment, multi-position), extracted from their
// copy-pasted fail/empty/finalize tails and rate-map builders. Semantics are
// frozen: every helper reproduces the processors' original behavior exactly,
// and format-specific logic stays in the per-format files.

// bccFailStatsBase identifies the import in failure stats.
type bccFailStatsBase struct {
	projectID    uint
	originalName string
	forMonth     string
}

// failBCCImport marks the import failed with a single reason — the shared
// shape of every processor's former fail() closure (all call sites passed
// status "failed" and no row count).
func (s *BCCImportService) failBCCImport(
	ctx context.Context,
	asset *domain.Asset,
	uploaderID uint,
	base bccFailStatsBase,
	reason string,
) (*BCCImportResult, error) {
	return s.failWithImportErrors(ctx, asset, uploaderID, BCCImportStats{
		ProjectID:    base.projectID,
		OriginalName: base.originalName,
		ForMonth:     base.forMonth,
	}, []domain.ImportError{{Reason: reason}})
}

// bccFailer returns a processor's fail closure bound to its identifying base.
func (s *BCCImportService) bccFailer(
	ctx context.Context,
	asset *domain.Asset,
	uploaderID uint,
	projectID uint,
	filename, forMonth string,
) func(reason string) (*BCCImportResult, error) {
	base := bccFailStatsBase{projectID: projectID, originalName: filename, forMonth: forMonth}
	return func(reason string) (*BCCImportResult, error) {
		return s.failBCCImport(ctx, asset, uploaderID, base, reason)
	}
}

// buildRateToTarget maps VND rate → (dayType, hourType) from flattened payrate
// paths. Rate is king: as long as the rate matches, the entry is valid. On
// collision (same rate, multiple paths), the lowest day-type priority wins
// (prefer "ngày thường"). posPrefix scopes the map to one position
// ("position.") for position-keyed formats; empty builds from every path
// (legacy behavior).
func buildRateToTarget(flatRates map[string]int, posPrefix string) map[int]rateTarget {
	rateToTarget := make(map[int]rateTarget)
	for path, rate := range flatRates {
		if rate == 0 {
			continue
		}
		if posPrefix != "" && !strings.HasPrefix(strings.ToLower(path), posPrefix) {
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) != 3 {
			continue
		}
		candidate := rateTarget{parts[1], parts[2]}
		candPri, candKnown := dayTypePriority[candidate.dayType]
		if !candKnown {
			continue // skip unrecognized day types
		}
		existing, exists := rateToTarget[rate]
		if !exists || candPri < dayTypePriority[existing.dayType] {
			rateToTarget[rate] = candidate
		}
	}
	return rateToTarget
}

// dedupBCCEntries merges entries that resolved to the same payrate bucket for
// the same employee on the same date, summing hours.
//
// Templates carry one cell per (day, shift column), but several columns can
// legitimately price to a single payrate leaf. The EPE weekly template is the
// worked example: its Sunday pair NN (ngày nghỉ) and TCNN (tăng ca ngày nghỉ)
// both declare 66,000 VND, and the project payrate has exactly one leaf at
// that rate (ngày nghỉ.ca ngày — ngày nghỉ.tăng ca is 0). Rate-keyed
// resolution therefore sends both cells to the same bucket, producing two
// entries with an identical (project_id, employee_id, date, hour_type,
// day_type). BulkCreateTimesheets rejects the second as an in-batch duplicate,
// which surfaced to partners as the misleading "Dữ liệu đã tồn tại".
//
// Summing is the correct merge, not first-wins: the columns are separate
// tallies of hours paid at one rate, so the bucket's true total is their sum.
// Two invariants fall out of it for free:
//  1. Net hours are preserved (NN=8 + TCNN=0 → 8; NN=8 + TCNN=2 → 10).
//  2. Deletion intent survives (NN=0 + TCNN=0 → 0, still an explicit
//     zero that deletes the matching chờ duyệt row, not "no entry").
//
// The key here is deliberately the same tuple BulkCreateTimesheets uses for
// its in-batch check, so colliding pairs collapse before they can reach it.
func dedupBCCEntries(entries []domainservices.BulkCreateTimesheetEntry) []domainservices.BulkCreateTimesheetEntry {
	if len(entries) <= 1 {
		return entries
	}
	type key struct {
		project  uint
		employee uint
		date     string
		hourType string
		dayType  string
	}
	indexByKey := make(map[key]int, len(entries))
	out := make([]domainservices.BulkCreateTimesheetEntry, 0, len(entries))
	for _, e := range entries {
		dt := ""
		if e.DayType != nil {
			dt = *e.DayType
		}
		k := key{e.ProjectID, e.EmployeeID, e.Date, e.HourType, dt}
		if idx, ok := indexByKey[k]; ok {
			out[idx].HoursWorked += e.HoursWorked
			continue
		}
		indexByKey[k] = len(out)
		out = append(out, e)
	}
	return out
}

// planMonthReplacement loads the project's existing month timesheets and
// plans the replacement: reviewed rows are preserved, pending rows become
// stale (to delete), and protected/flexible rows are counted as skipped.
// hourTypeKeyed makes HourType part of the replacement key so an OT import
// cannot replace HC (weekly formats); legacy/multi formats key on
// employee+date only. No-op when there are no entries to plan against.
//
// Entries are deduplicated first. Every template family funnels through here
// on its way to BulkCreateTimesheets, so this is the one place that can
// guarantee no format ships a same-bucket collision downstream — see
// dedupBCCEntries. Both rate-keyed paths (legacy, multi-position) and both
// label-keyed ones (weekly BCC, weekly payment) can produce them.
func (s *BCCImportService) planMonthReplacement(
	ctx context.Context,
	projectID uint,
	year int,
	month time.Month,
	monthStart time.Time,
	loc *time.Location,
	entries []domainservices.BulkCreateTimesheetEntry,
	flexibleEmployeeIDs map[uint]struct{},
	hourTypeKeyed bool,
) (plannedEntries []domainservices.BulkCreateTimesheetEntry, staleIDs []uint, protectedSkipped, flexibleSkipped int, err error) {
	if len(entries) == 0 {
		return entries, nil, 0, 0, nil
	}
	entries = dedupBCCEntries(entries)
	monthEnd := time.Date(year, month+1, 0, 23, 59, 59, 0, loc)
	existingTS, terr := s.timesheetReader.GetByProject(ctx, projectID, monthStart, monthEnd)
	if terr != nil {
		return nil, nil, 0, 0, terr
	}
	entries, staleIDs, protected, flexible := planBCCReplacement(entries, existingTS, flexibleEmployeeIDs, hourTypeKeyed)
	return entries, staleIDs, protected, flexible, nil
}

// finishEmptyBCCImport resolves the no-entries tail shared by every
// processor: an all-skipped import completes as skipped; row errors persist
// as failed; otherwise the supplied reason fails the import.
func (s *BCCImportService) finishEmptyBCCImport(
	ctx context.Context,
	asset *domain.Asset,
	uploaderID uint,
	projectID uint,
	filename, forMonth string,
	totalRows, protectedSkipped, flexibleSkipped int,
	importErrors []domain.ImportError,
	reason string,
) (*BCCImportResult, error) {
	if len(importErrors) == 0 && protectedSkipped+flexibleSkipped > 0 {
		return s.completeSkippedBCCImport(ctx, asset, uploaderID, BCCImportStats{
			ProjectID:    projectID,
			OriginalName: filename,
			ForMonth:     forMonth,
			TotalRows:    totalRows,
			SkippedCount: protectedSkipped + flexibleSkipped,
		})
	}
	if len(importErrors) > 0 {
		return s.failWithImportErrors(ctx, asset, uploaderID, BCCImportStats{
			ProjectID:    projectID,
			OriginalName: filename,
			ForMonth:     forMonth,
			TotalRows:    totalRows,
		}, importErrors)
	}
	return s.failBCCImport(ctx, asset, uploaderID,
		bccFailStatsBase{projectID: projectID, originalName: filename, forMonth: forMonth}, reason)
}

// finalizeCreatedBCCImport computes the final counts and persists the
// completed import's stats — the shared success tail after
// applyTimesheetReplacement.
//
// "Skipped" covers work we did that did not produce a new timesheet:
// a pending row we replaced (deletedCount), an approved/paid row we left
// untouched (protectedSkipped), or a flexible-employee row we ignored
// (flexibleSkipped). It does NOT count zero-cell no-ops: when a zero entry
// finds no pending row to delete, nothing visible happens, and inflating
// the count turns a successful import into a partner-visible "almost
// empty" result (the bug behind the EVA T09.2026 complaint).
// bccImportSkippedCount returns the count of import rows that did not produce
// a new timesheet: a pending row we replaced (deletedCount), an approved or
// paid row we left untouched (protectedSkipped), or a flexible-employee row
// we ignored (flexibleSkipped).
//
// Zero-cell no-ops are deliberately NOT counted here. When a zero entry finds
// no pending row to delete, nothing visible happens — the partner sees no
// change. Counting these inflated the "Skipped" figure into "almost empty"
// territory on the EVA T09.2026 import (217 created vs. 692 skipped, the
// partner read it as a near-total failure).
func bccImportSkippedCount(deletedCount, protectedSkipped, flexibleSkipped int) int {
	return deletedCount + protectedSkipped + flexibleSkipped
}

func (s *BCCImportService) finalizeCreatedBCCImport(
	ctx context.Context,
	asset *domain.Asset,
	uploaderID uint,
	projectID uint,
	filename, forMonth string,
	totalRows, createdCount, protectedSkipped, flexibleSkipped, deletedCount int,
	importErrors []domain.ImportError,
) (*BCCImportResult, error) {
	skippedCount := bccImportSkippedCount(deletedCount, protectedSkipped, flexibleSkipped)
	errorCount := len(importErrors)
	now := clock.Now()
	finalStatus := "completed"
	if createdCount == 0 && errorCount > 0 {
		finalStatus = "failed"
	}
	stats := BCCImportStats{
		ProjectID:    projectID,
		OriginalName: filename,
		ForMonth:     forMonth,
		Status:       finalStatus,
		TotalRows:    totalRows,
		CreatedCount: createdCount,
		SkippedCount: skippedCount,
		ErrorCount:   errorCount,
		ErrorDetail:  marshalErrors(importErrors),
		ProcessedAt:  &now,
	}
	return s.finalizeBCCImport(ctx, asset, uploaderID, stats)
}
