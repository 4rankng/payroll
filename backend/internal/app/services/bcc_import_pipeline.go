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

// planMonthReplacement loads the project's existing month timesheets and
// plans the replacement: reviewed rows are preserved, pending rows become
// stale (to delete), and protected/flexible rows are counted as skipped.
// hourTypeKeyed makes HourType part of the replacement key so an OT import
// cannot replace HC (weekly formats); legacy/multi formats key on
// employee+date only. No-op when there are no entries to plan against.
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
func (s *BCCImportService) finalizeCreatedBCCImport(
	ctx context.Context,
	asset *domain.Asset,
	uploaderID uint,
	projectID uint,
	filename, forMonth string,
	totalRows, createdCount, protectedSkipped, flexibleSkipped, deletedCount, zeroHourCount int,
	importErrors []domain.ImportError,
) (*BCCImportResult, error) {
	skippedCount := deletedCount + protectedSkipped + flexibleSkipped + zeroHourCount
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
