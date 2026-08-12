package attendance

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// checkInShiftWindow is the half-width of the check-in window around the
// configured shift start T: check-in is allowed in (T - checkInShiftWindow,
// T + checkInShiftWindow).
const checkInShiftWindow = 1 * time.Hour

const (
	// checkOutLowerGrace is how long before the configured shift end K a checkout
	// is still allowed.
	checkOutLowerGrace = 1 * time.Hour
	// checkOutUpperGrace is how long after the configured shift end K a checkout is
	// still allowed: checkout is valid in [K - checkOutLowerGrace, K + checkOutUpperGrace].
	checkOutUpperGrace = 4 * time.Hour
)

const (
	attendanceCheckInWindowCode  = "ATTENDANCE_CHECK_IN_WINDOW"
	attendanceCheckOutWindowCode = "ATTENDANCE_CHECK_OUT_WINDOW"
)

func newTimingValidationError(code, action, message string, earliest, latest time.Time) *domain.DomainError {
	return domain.NewValidationErrorWithCode(code, message).
		WithContext("guidance_type", "timing").
		WithContext("action", action).
		WithContext("window_start", earliest.Format("15:04")).
		WithContext("window_end", latest.Format("15:04"))
}

// checkInFitsShift reports whether checkInTime lies within the allowed check-in
// window (shift.start - checkInShiftWindow, shift.start + checkInShiftWindow).
// Shared by the check-in gate (validateCheckInWindow) and the earning match so
// the two cannot diverge.
func checkInFitsShift(shift *parsedShift, checkInTime time.Time) bool {
	return checkInTime.After(shift.start.Add(-checkInShiftWindow)) && checkInTime.Before(shift.start.Add(checkInShiftWindow))
}

// checkOutFitsShift reports whether checkOutTime lies within the allowed
// checkout window [shift.end - checkOutLowerGrace, shift.end + checkOutUpperGrace].
// Shared by the checkout gate (validateCheckOutWindow) and the earning match.
func checkOutFitsShift(shift *parsedShift, checkOutTime time.Time) bool {
	earliest := shift.end.Add(-checkOutLowerGrace)
	latest := shift.end.Add(checkOutUpperGrace)
	return (checkOutTime.After(earliest) || checkOutTime.Equal(earliest)) && !checkOutTime.After(latest)
}

// validateCheckInWindow rejects a check-in that falls outside the allowed
// (shift.start - checkInShiftWindow, shift.start + checkInShiftWindow) window
// around the configured shift start T.
func validateCheckInWindow(shift *parsedShift, checkInTime time.Time) error {
	if checkInFitsShift(shift, checkInTime) {
		return nil
	}
	earliest := shift.start.Add(-checkInShiftWindow)
	latest := shift.start.Add(checkInShiftWindow)
	return newTimingValidationError(attendanceCheckInWindowCode, "check_in", fmt.Sprintf(
		"Giờ vào làm không hợp lệ. Bạn chỉ được vào làm từ %s đến %s.",
		earliest.Format("15:04"),
		latest.Format("15:04"),
	), earliest, latest)
}

// validateCheckOutWindow rejects a checkout that falls outside the allowed
// [shift.end - checkOutLowerGrace, shift.end + checkOutUpperGrace] window, where
// shift.end is the configured shift end K. The message points the employee at
// the allowed checkout period.
func validateCheckOutWindow(shift *parsedShift, checkInTime, checkOutTime time.Time) error {
	earliest := shift.end.Add(-checkOutLowerGrace)
	latest := shift.end.Add(checkOutUpperGrace)
	if checkOutTime.Before(earliest) {
		return newTimingValidationError(attendanceCheckOutWindowCode, "check_out", fmt.Sprintf(
			"Bạn mới vào làm lúc %s. Chỉ có thể tan ca từ %s đến %s.",
			checkInTime.Format("15:04"),
			earliest.Format("15:04"),
			latest.Format("15:04"),
		), earliest, latest)
	}
	if checkOutTime.After(latest) {
		return newTimingValidationError(attendanceCheckOutWindowCode, "check_out", fmt.Sprintf(
			"Đã quá giờ tan ca. Bạn chỉ được tan ca từ %s đến %s.",
			earliest.Format("15:04"),
			latest.Format("15:04"),
		), earliest, latest)
	}
	return nil
}

// ShiftWindow is an advisory representation of a configured shift and the
// attendance windows derived from it. It is intentionally separate from the
// domain Attendance model: these times are informational and the validation
// methods remain authoritative.
type ShiftWindow struct {
	ShiftStart          time.Time
	ShiftEnd            time.Time
	CheckInWindowStart  time.Time
	CheckInWindowEnd    time.Time
	CheckOutWindowStart time.Time
	CheckOutWindowEnd   time.Time
	// Name is the admin-chosen display name for this shift's time-range (e.g.
	// "Ca làm" for "09:00-18:00"), looked up from the project's ShiftNames.
	// Empty when no name is configured — callers fall back to default labels.
	Name string
}

// resolveShifts parses the flattened payrate for the given position and returns:
//   - effectivePosition: the configured position matching `position` (case- and
//     diacritic-insensitive), falling back to the only configured position when the
//     requested one is absent.
//   - positionFound: whether the effective position exists in the configuration.
//   - shifts: every parseable shift for the effective position, generated for the
//     check-in day and its ±1 neighbors so night shifts and early arrivals anchor
//     to the correct calendar day.
//
// The caller derives shiftFound as len(shifts) > 0.
func resolveShifts(flattened map[string]int, position string, ci time.Time) (effectivePosition string, positionFound bool, shifts []parsedShift) {
	configuredPositions := make(map[string]string)
	for key := range flattened {
		parts := strings.Split(key, ".")
		if len(parts) < 3 {
			continue
		}
		pos := strings.Join(parts[:len(parts)-2], ".")
		configuredPositions[strings.ToLower(pos)] = pos
	}

	effectivePosition = position
	for _, configuredPosition := range configuredPositions {
		if strings.EqualFold(configuredPosition, position) {
			effectivePosition = configuredPosition
			break
		}
	}
	if !hasConfiguredPosition(configuredPositions, effectivePosition) && len(configuredPositions) == 1 {
		for _, onlyPosition := range configuredPositions {
			effectivePosition = onlyPosition
		}
	}
	positionFound = hasConfiguredPosition(configuredPositions, effectivePosition)

	for key, amount := range flattened {
		// key is position.dayType.HH:MM-HH:MM
		parts := strings.Split(key, ".")
		if len(parts) < 3 {
			continue
		}

		pos := strings.Join(parts[:len(parts)-2], ".")
		if !strings.EqualFold(pos, effectivePosition) {
			continue
		}

		timeRange := parts[len(parts)-1]
		timeParts := strings.Split(timeRange, "-")
		if len(timeParts) != 2 {
			continue
		}

		start, startErr := time.Parse("15:04", timeParts[0])
		end, endErr := time.Parse("15:04", timeParts[1])
		if startErr != nil || endErr != nil {
			continue
		}

		// Generate candidate shifts starting on the day before, the day of, and
		// the day after check-in. Each candidate keeps absolute [start, end] with
		// cross-midnight handled by adding 24h to end. closestShift then picks the
		// candidate whose start is nearest the check-in, anchoring night shifts and
		// early arrivals (e.g. 19:50 for 20:00-04:00) to the correct calendar day —
		// replacing the old rollback hack that miscomputed K for early arrivals.
		for dayOffset := -1; dayOffset <= 1; dayOffset++ {
			day := ci.AddDate(0, 0, dayOffset)
			shiftStart := time.Date(day.Year(), day.Month(), day.Day(), start.Hour(), start.Minute(), 0, 0, ci.Location())
			shiftEnd := time.Date(day.Year(), day.Month(), day.Day(), end.Hour(), end.Minute(), 0, 0, ci.Location())
			if shiftEnd.Before(shiftStart) {
				shiftEnd = shiftEnd.Add(24 * time.Hour) // Night shift crosses midnight
			}
			shifts = append(shifts, parsedShift{start: shiftStart, end: shiftEnd, amount: amount})
		}
	}

	return effectivePosition, positionFound, shifts
}

// closestShift returns the shift the check-in belongs to, or nil when there are
// no shifts. It prefers a shift whose [start, end] actually contains the
// check-in (the worker is mid-shift) and only falls back to nearest start when
// none contains ci. The "contains" preference stops a short neighboring shift
// from stealing the anchor near a long shift's end (e.g. a 04:00-05:00 shift
// winning over a 20:00-04:00 night shift for a 03:55 check-in) and avoids
// resolving off-hours check-ins to a future shift. Used to anchor the
// check-in/checkout/earning windows and to report expected shift boundaries.
func closestShift(shifts []parsedShift, ci time.Time) *parsedShift {
	contains := func(sh *parsedShift) bool {
		return (ci.After(sh.start) || ci.Equal(sh.start)) && (ci.Before(sh.end) || ci.Equal(sh.end))
	}
	var closest *parsedShift
	var closestDistance time.Duration
	for i := range shifts {
		sh := &shifts[i]
		distance := ci.Sub(sh.start).Abs()
		switch {
		case closest == nil:
			closest, closestDistance = sh, distance
		case contains(sh) && !contains(closest):
			closest, closestDistance = sh, distance
		case contains(sh) == contains(closest) && distance < closestDistance:
			closest, closestDistance = sh, distance
		}
	}
	return closest
}

// resolveShift returns the configured shift whose start is closest to checkInTime
// (resolved across ±1 day so night shifts and early arrivals anchor to the correct
// calendar day), or nil when no payrate/position/shift can be resolved. Callers
// derive the check-in/checkout windows from the returned shift:
//   - check-in valid in (shift.start - checkInShiftWindow, shift.start + checkInShiftWindow)
//   - checkout valid in [shift.end - checkOutLowerGrace, shift.end + checkOutUpperGrace]
//
// A nil result means the project has no valid shift configuration for the position,
// so the check-in/checkout is rejected rather than falling back to a fixed duration.
func (s *AttendanceService) resolveShift(payrate *domain.Payrate, position string, checkInTime time.Time) *parsedShift {
	if payrate == nil {
		return nil
	}
	flattened, err := payrate.Payrate.Flatten()
	if err != nil {
		// A malformed payrate is operationally distinct from "no shift configured";
		// log it so ops can tell a broken config from a missing one (the user-facing
		// message is the same generic "not configured" either way).
		observability.GetLogger().Warn("failed to flatten payrate; cannot resolve shift", "error", err)
		return nil
	}
	_, _, shifts := resolveShifts(flattened, position, checkInTime)
	return closestShift(shifts, checkInTime)
}

func hasConfiguredPosition(configuredPositions map[string]string, position string) bool {
	for _, configuredPosition := range configuredPositions {
		if strings.EqualFold(configuredPosition, position) {
			return true
		}
	}
	return false
}

// ResolveShiftWindow is retained for callers that only need the check-in
// advisory window. New callers that also need checkout guidance should use
// ResolveShiftWindows.
func ResolveShiftWindow(flattened map[string]int, position string, now time.Time) (shiftStart, shiftEnd, windowStart, windowEnd time.Time, ok bool) {
	shiftStart, shiftEnd, windowStart, windowEnd, _, _, ok = ResolveShiftWindows(flattened, position, now)
	return shiftStart, shiftEnd, windowStart, windowEnd, ok
}

// ResolveShiftWindows derives both advisory attendance windows for display on
// the employee portal. It reuses the same resolveShifts → closestShift logic as
// CheckIn and CheckOut, including cross-midnight shift resolution. The returned
// checkout bounds are [shift.end - 1h, shift.end + 4h], matching
// validateCheckOutWindow exactly.
func ResolveShiftWindows(flattened map[string]int, position string, now time.Time) (shiftStart, shiftEnd, checkInWindowStart, checkInWindowEnd, checkOutWindowStart, checkOutWindowEnd time.Time, ok bool) {
	_, _, shifts := resolveShifts(flattened, position, now)
	if len(shifts) == 0 {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, false
	}
	shift := closestShift(shifts, now)
	if shift == nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, time.Time{}, false
	}
	return shift.start,
		shift.end,
		shift.start.Add(-checkInShiftWindow),
		shift.start.Add(checkInShiftWindow),
		shift.end.Add(-checkOutLowerGrace),
		shift.end.Add(checkOutUpperGrace),
		true
}

// ResolveAllShiftWindows returns one advisory window for every configured shift
// for the resolved position. The absolute date is anchored to now, but callers
// should display the time-of-day values; duplicate ±1-day resolver candidates
// are removed. Cross-midnight end and checkout times retain their correct
// following-day instants.
//
// names optionally maps a "HH:MM-HH:MM" time-range to an admin-chosen display
// name; when present, the matching ShiftWindow.Name is populated. Pass nil when
// the caller has no name config (windows come back with empty Name).
func ResolveAllShiftWindows(flattened map[string]int, position string, now time.Time, names map[string]string) []ShiftWindow {
	_, _, shifts := resolveShifts(flattened, position, now)
	windowsByTimeRange := make(map[string]ShiftWindow)
	for _, shift := range shifts {
		if shift.start.Year() != now.Year() || shift.start.YearDay() != now.YearDay() {
			continue
		}
		key := shift.start.Format("15:04") + "-" + shift.end.Format("15:04")
		window := ShiftWindow{
			ShiftStart:          shift.start,
			ShiftEnd:            shift.end,
			CheckInWindowStart:  shift.start.Add(-checkInShiftWindow),
			CheckInWindowEnd:    shift.start.Add(checkInShiftWindow),
			CheckOutWindowStart: shift.end.Add(-checkOutLowerGrace),
			CheckOutWindowEnd:   shift.end.Add(checkOutUpperGrace),
		}
		if names != nil {
			window.Name = names[key]
		}
		windowsByTimeRange[key] = window
	}

	windows := make([]ShiftWindow, 0, len(windowsByTimeRange))
	for _, window := range windowsByTimeRange {
		windows = append(windows, window)
	}
	sort.Slice(windows, func(i, j int) bool {
		return windows[i].ShiftStart.Before(windows[j].ShiftStart)
	})
	return windows
}

// ExtractShiftRanges returns the set of distinct "HH:MM-HH:MM" shift time-ranges
// configured in the flattened payrate, across every position. Used by project
// create/update handlers to validate that admin-named shifts (Project.ShiftNames)
// correspond to shifts the payrate actually defines. The returned slice is
// deduplicated but unordered.
func ExtractShiftRanges(flattened map[string]int) []string {
	seen := make(map[string]struct{})
	ranges := make([]string, 0)
	for key := range flattened {
		parts := strings.Split(key, ".")
		if len(parts) < 3 {
			continue
		}
		timeRange := parts[len(parts)-1]
		if _, ok := seen[timeRange]; ok {
			continue
		}
		if domain.IsValidShiftRange(timeRange) {
			seen[timeRange] = struct{}{}
			ranges = append(ranges, timeRange)
		}
	}
	return ranges
}
