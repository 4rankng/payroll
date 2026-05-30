package services

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

// BulkZeroRateEntry is the minimal info needed for bulk zero-rate validation.
type BulkZeroRateEntry struct {
	ProjectID  uint
	EmployeeID uint
	Date       time.Time
	PayType    string
}

// ValidateZeroRate checks whether the payrate bucket for the given payType is 0.
// A zero rate means the project does not pay for that hour/day-type combination,
// so timesheet creation should be rejected.
// Returns nil if the rate is valid (> 0), or a PreviewError if it is zero or missing.
func (s *TimesheetValidationService) ValidateZeroRate(ctx context.Context, projectID uint, date time.Time, payType string, employeeID uint) *PreviewError {
	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, projectID, date)
	if err != nil || payrate == nil {
		// No active payrate — already caught by ValidatePayrate; skip here to avoid duplicate errors.
		return nil
	}

	rate, err := payrate.GetRateValue(payType)
	if err != nil {
		// Path not found in config — treat as zero-rate.
		_, hourType := s.parsePaytype(payType)
		return &PreviewError{
			EmployeeID: employeeID,
			Message:    fmt.Sprintf(constants.MsgCannotCreateTimesheetZeroRateVN, hourType),
		}
	}

	if rate == 0 {
		_, hourType := s.parsePaytype(payType)
		return &PreviewError{
			EmployeeID: employeeID,
			Message:    fmt.Sprintf(constants.MsgCannotCreateTimesheetZeroRateVN, hourType),
		}
	}

	return nil
}

// ValidateBulkZeroRates checks zero-rate buckets for a batch of entries in bulk.
// It batches payrate lookups by (projectID, date) to minimise DB round-trips.
// Errors for the same employee are combined into a single message listing all missing hour types.
func (s *TimesheetValidationService) ValidateBulkZeroRates(ctx context.Context, entries []BulkZeroRateEntry) []PreviewError {
	// Cache payrates by "projectID-date" to avoid repeated DB calls.
	payrateCache := make(map[string]*domain.Payrate)

	// Collect missing hour types per employee, preserving insertion order.
	type employeeKey struct {
		EmployeeID uint
	}
	missingHourTypes := make(map[employeeKey][]string)
	seen := make(map[string]bool) // deduplicate (employeeID, hourType) pairs

	for _, entry := range entries {
		cacheKey := fmt.Sprintf("%d-%s", entry.ProjectID, entry.Date.Format("2006-01-02"))

		payrate, cached := payrateCache[cacheKey]
		if !cached {
			pr, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, entry.ProjectID, entry.Date)
			if err != nil || pr == nil {
				payrateCache[cacheKey] = nil
				continue
			}
			payrateCache[cacheKey] = pr
			payrate = pr
		}

		if payrate == nil {
			continue
		}

		rate, err := payrate.GetRateValue(entry.PayType)
		if err != nil || rate == 0 {
			_, hourType := s.parsePaytype(entry.PayType)
			key := employeeKey{EmployeeID: entry.EmployeeID}
			dedupeKey := fmt.Sprintf("%d-%s", entry.EmployeeID, hourType)
			if !seen[dedupeKey] {
				seen[dedupeKey] = true
				missingHourTypes[key] = append(missingHourTypes[key], fmt.Sprintf("'%s'", hourType))
			}
		}
	}

	var errors []PreviewError
	for key, hourTypes := range missingHourTypes {
		combined := hourTypes[0]
		for i := 1; i < len(hourTypes); i++ {
			combined += ", " + hourTypes[i]
		}
		errors = append(errors, PreviewError{
			EmployeeID: key.EmployeeID,
			Message:    fmt.Sprintf("Dự án không có cấu hình cho loại giờ %s", combined),
		})
	}

	return errors
}
