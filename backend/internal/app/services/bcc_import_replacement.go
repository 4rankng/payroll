package services

import (
	"context"
	"fmt"
	"strings"

	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"
)

type bccReplacementKey struct {
	employeeID uint
	date       string
	hourType   string
}

func bccEntryReplacementKey(entry domainservices.BulkCreateTimesheetEntry, includeHourType bool) bccReplacementKey {
	key := bccReplacementKey{employeeID: entry.EmployeeID, date: entry.Date}
	if includeHourType {
		key.hourType = strings.ToLower(entry.HourType)
	}
	return key
}

func bccTimesheetReplacementKey(timesheet *domain.Timesheet, includeHourType bool) bccReplacementKey {
	key := bccReplacementKey{
		employeeID: timesheet.EmployeeID,
		date:       timesheet.Date.Format("2006-01-02"),
	}
	if includeHourType {
		parts := strings.Split(timesheet.PayType, ".")
		key.hourType = strings.ToLower(parts[len(parts)-1])
	}
	return key
}

// planBCCReplacement preserves reviewed payroll data, replaces only pending
// rows, and leaves missing rows for the bulk-create path. Paid and other
// non-pending rows are also protected because an import must not reopen them.
// A row linked to a revenue transaction is protected too: the transfer result
// may still be finishing its payment-status update asynchronously, so replacing
// that row would orphan the provider result from the newly-created timesheet.
func planBCCReplacement(
	entries []domainservices.BulkCreateTimesheetEntry,
	existingTimesheets []*domain.Timesheet,
	flexibleEmployeeIDs map[uint]struct{},
	includeHourType bool,
) (
	filteredEntries []domainservices.BulkCreateTimesheetEntry,
	staleIDs []uint,
	protectedSkippedCount int,
	flexibleSkippedCount int,
) {
	requestedKeys := make(map[bccReplacementKey]struct{}, len(entries))
	for _, entry := range entries {
		requestedKeys[bccEntryReplacementKey(entry, includeHourType)] = struct{}{}
	}

	protectedKeys := make(map[bccReplacementKey]struct{})
	flexibleKeys := make(map[bccReplacementKey]struct{})
	for _, timesheet := range existingTimesheets {
		key := bccTimesheetReplacementKey(timesheet, includeHourType)
		if _, requested := requestedKeys[key]; !requested {
			continue
		}
		if _, flexible := flexibleEmployeeIDs[timesheet.EmployeeID]; flexible {
			flexibleKeys[key] = struct{}{}
			continue
		}
		if timesheet.Status != domain.TimesheetStatusPendingApproval ||
			timesheet.TransactionID != nil ||
			timesheet.PaymentStatus == domain.PaymentStatusPaid ||
			timesheet.PaymentStatus == domain.PaymentStatusFailed ||
			timesheet.PaymentStatus == domain.PaymentStatusCancelled {
			protectedKeys[key] = struct{}{}
		}
	}

	for _, timesheet := range existingTimesheets {
		key := bccTimesheetReplacementKey(timesheet, includeHourType)
		if _, requested := requestedKeys[key]; !requested {
			continue
		}
		if _, flexible := flexibleKeys[key]; flexible {
			continue
		}
		if _, protected := protectedKeys[key]; protected {
			continue
		}
		if timesheet.Status == domain.TimesheetStatusPendingApproval {
			staleIDs = append(staleIDs, timesheet.ID)
		}
	}

	filteredEntries = make([]domainservices.BulkCreateTimesheetEntry, 0, len(entries))
	for _, entry := range entries {
		key := bccEntryReplacementKey(entry, includeHourType)
		if _, flexible := flexibleKeys[key]; flexible {
			flexibleSkippedCount++
			continue
		}
		if _, protected := protectedKeys[key]; protected {
			protectedSkippedCount++
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}

	return filteredEntries, staleIDs, protectedSkippedCount, flexibleSkippedCount
}

// countZeroHourEntries counts entries whose cell explicitly held 0 — deletion
// requests. They surface as skipped rows: nothing is created for them, the
// matching pending row (when it existed) was hard-deleted as stale. Negative
// hours are invalid data on the failure path, not deletion requests.
func countZeroHourEntries(entries []domainservices.BulkCreateTimesheetEntry) int {
	n := 0
	for _, e := range entries {
		if e.HoursWorked == 0 {
			n++
		}
	}
	return n
}

func (s *BCCImportService) applyTimesheetReplacement(
	ctx context.Context,
	staleIDs []uint,
	entries []domainservices.BulkCreateTimesheetEntry,
	uploaderID uint,
	uploaderRole string,
) (*domainservices.BulkCreateTimesheetResult, error) {
	return s.applyTimesheetReplacementPrepared(ctx, staleIDs, entries, uploaderID, uploaderRole, nil)
}

func (s *BCCImportService) applyTimesheetReplacementPrepared(
	ctx context.Context,
	staleIDs []uint,
	entries []domainservices.BulkCreateTimesheetEntry,
	uploaderID uint,
	uploaderRole string,
	prepare func(context.Context) error,
) (*domainservices.BulkCreateTimesheetResult, error) {
	requireBCCImportApproval(entries)

	var result *domainservices.BulkCreateTimesheetResult
	err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if prepare != nil {
			if err := prepare(txCtx); err != nil {
				return err
			}
		}
		for _, id := range staleIDs {
			if err := s.timesheetWriter.HardDelete(txCtx, id); err != nil {
				return fmt.Errorf("xóa bảng chấm công cũ %d: %w", id, err)
			}
		}

		// Carry deleted IDs into validation so a replacement never treats its
		// own stale rows as duplicates while this transaction is pending.
		replacementCtx := domainservices.WithTimesheetReplacementDeletes(txCtx, staleIDs)

		var err error
		result, err = s.timesheetService.BulkCreateTimesheetsInTransaction(
			replacementCtx,
			entries,
			uploaderID,
			uploaderRole,
		)
		if err != nil {
			return err
		}
		if len(result.FailedEntries) > 0 {
			return fmt.Errorf("không thể thay thế an toàn: %d dòng không hợp lệ", len(result.FailedEntries))
		}
		return nil
	})
	if err != nil {
		// The transaction correctly rolls back every replacement when one or more
		// rows are invalid, but the caller still needs the collected row failures
		// to tell the uploader what to correct.
		return result, err
	}
	return result, nil
}

func requireBCCImportApproval(entries []domainservices.BulkCreateTimesheetEntry) {
	for i := range entries {
		// BCC rows are imported records and always require an explicit review,
		// independent of whether an admin or partner uploaded the workbook.
		entries[i].RequireApproval = true
	}
}
