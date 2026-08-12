package services

import (
	"fmt"
	"sort"
	"strings"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"
	"api-server/internal/pkg/utils"

	"golang.org/x/text/unicode/norm"
)

// wbccRateKey groups rate lookups by (position, dayType) for weekly BCC imports.
type wbccRateKey struct {
	position string
	dayType  string
}

func weeklyBCCMissingAssignmentError(
	cccd string,
	fullName string,
	blockedCCCDs map[string]struct{},
	reportedCCCDs map[string]struct{},
) *domain.ImportError {
	if isWeeklyBCCEmployeeBlocked(blockedCCCDs, cccd) {
		return nil
	}
	if _, reported := reportedCCCDs[cccd]; reported {
		return nil
	}
	reportedCCCDs[cccd] = struct{}{}
	return &domain.ImportError{
		Employee: fullName,
		Reason:   fmt.Sprintf("không tìm thấy nhân viên với CCCD \"%s\" trong dự án", cccd),
	}
}

func isWeeklyBCCEmployeeBlocked(blockedCCCDs map[string]struct{}, cccd string) bool {
	_, blocked := blockedCCCDs[cccd]
	return blocked
}

// shiftInConfig reports whether shiftType appears as a leaf key in the flattened
// payrate (case-insensitive), regardless of its rate value. Used to validate that
// a BCC sheet's shift exists in the payrate active for a given date.
func shiftInConfig(flatRates map[string]int, shiftType string) bool {
	if len(flatRates) == 0 {
		return false
	}
	target := strings.ToLower(shiftType)
	for path := range flatRates {
		parts := strings.Split(path, ".")
		if len(parts) >= 3 && strings.ToLower(parts[len(parts)-1]) == target {
			return true
		}
	}
	return false
}

// buildShiftRatesForShift collects every non-zero rate path whose leaf matches
// shiftType (case-insensitive), grouped by (position, dayType) for quick lookup.
func buildShiftRatesForShift(flatRates map[string]int, shiftType string) map[wbccRateKey]int {
	shiftRates := make(map[wbccRateKey]int)
	target := strings.ToLower(shiftType)
	for path, rate := range flatRates {
		if rate == 0 {
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) < 3 {
			continue
		}
		if strings.ToLower(parts[len(parts)-1]) != target {
			continue
		}
		position := parts[0]
		dayType := strings.Join(parts[1:len(parts)-1], ".")
		key := wbccRateKey{position: canonicalBCCRateKeySegment(position), dayType: canonicalBCCRateKeySegment(dayType)}
		shiftRates[key] = rate
	}
	return shiftRates
}

// canonicalBCCRateKeySegment makes human-facing Vietnamese labels comparable
// with flattened payrate path segments. Shift keys are matched separately.
func canonicalBCCRateKeySegment(value string) string {
	return strings.TrimSpace(utils.NormalizeVietnamese(norm.NFC.String(value)))
}

func weeklyBCCWeekendFallbackKey(position string) wbccRateKey {
	return wbccRateKey{
		position: canonicalBCCRateKeySegment(position),
		dayType:  canonicalBCCRateKeySegment("ngày nghỉ"),
	}
}

func weeklyPaymentRateKey(sheetPosition string) wbccRateKey {
	return wbccRateKey{
		position: canonicalBCCRateKeySegment(sheetPosition),
		dayType:  canonicalBCCRateKeySegment("ngày thường"),
	}
}

// buildWeeklyPaymentPositions makes the salary sheet the authoritative source
// for each employee's project position. An employee cannot safely belong to two
// different salary-tier sheets because assignment position is later used to
// construct the persisted paytype and resolve the final rate.
func buildWeeklyPaymentPositions(
	sheets []excelparser.WeeklyPaymentSheetData,
) (map[string]string, map[string]struct{}, []domain.ImportError) {
	positions := make(map[string]string)
	blocked := make(map[string]struct{})
	var importErrors []domain.ImportError

	for _, sheet := range sheets {
		for _, employee := range sheet.Employees {
			cccd := strings.TrimSpace(employee.EmployeeCode)
			if cccd == "" {
				continue
			}
			if _, alreadyBlocked := blocked[cccd]; alreadyBlocked {
				continue
			}

			existingPosition, exists := positions[cccd]
			if !exists {
				positions[cccd] = sheet.Position
				continue
			}
			if canonicalBCCRateKeySegment(existingPosition) == canonicalBCCRateKeySegment(sheet.Position) {
				continue
			}

			delete(positions, cccd)
			blocked[cccd] = struct{}{}
			importErrors = append(importErrors, domain.ImportError{
				Employee: employee.FullName,
				Reason: fmt.Sprintf("nhân viên %q xuất hiện ở nhiều sheet vị trí (%s, %s)",
					employee.FullName, existingPosition, sheet.Position),
			})
		}
	}

	return positions, blocked, importErrors
}

func planWeeklyPaymentPositionCorrections(
	assignments []*domain.ProjectEmployee,
	positionByCCCD map[string]string,
	includeFlexibleEmployees bool,
) map[uint]posCorrection {
	corrections := make(map[uint]posCorrection)
	for _, assignment := range assignments {
		expectedPosition := positionByCCCD[assignment.EmployeeCCCD]
		if expectedPosition == "" || canonicalBCCRateKeySegment(assignment.Position) == canonicalBCCRateKeySegment(expectedPosition) {
			continue
		}
		if assignment.PaymentSchedule == string(domain.PaymentScheduleFlexible) && !includeFlexibleEmployees {
			continue
		}
		corrections[assignment.EmployeeID] = posCorrection{
			assignmentID: assignment.ID,
			employeeID:   assignment.EmployeeID,
			oldPosition:  assignment.Position,
			newPosition:  expectedPosition,
			employeeName: assignment.EmployeeName,
		}
	}
	return corrections
}

func selectWeeklyPaymentPositionCorrections(
	entries []domainservices.BulkCreateTimesheetEntry,
	planned map[uint]posCorrection,
) []posCorrection {
	selectedByAssignment := make(map[uint]posCorrection)
	for _, entry := range entries {
		if correction, ok := planned[entry.EmployeeID]; ok {
			selectedByAssignment[correction.assignmentID] = correction
		}
	}
	assignmentIDs := make([]uint, 0, len(selectedByAssignment))
	for assignmentID := range selectedByAssignment {
		assignmentIDs = append(assignmentIDs, assignmentID)
	}
	sort.Slice(assignmentIDs, func(i, j int) bool { return assignmentIDs[i] < assignmentIDs[j] })

	selected := make([]posCorrection, 0, len(assignmentIDs))
	for _, assignmentID := range assignmentIDs {
		selected = append(selected, selectedByAssignment[assignmentID])
	}
	return selected
}

// getShiftTypes extracts unique leaf-level shift type names from flattened payrate paths.
// For paths like "pho thong.ngay thuong.HC" → returns ["HC"].
func getShiftTypes(flatRates map[string]int) []string {
	seen := make(map[string]bool)
	var shiftTypes []string
	for path := range flatRates {
		parts := strings.Split(path, ".")
		if len(parts) >= 3 {
			leaf := parts[len(parts)-1]
			if !seen[leaf] {
				seen[leaf] = true
				shiftTypes = append(shiftTypes, leaf)
			}
		}
	}
	return shiftTypes
}

// determineDayType returns the Vietnamese day type string based on the calendar date.
// NOTE: Only distinguishes weekday vs weekend; public holidays are NOT detected.
func determineDayType(date time.Time) string {
	if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		return "ngày nghỉ"
	}
	return "ngày thường"
}

// findSTKRow finds an STK row by CCCD for bank info lookup.
func findSTKRow(stkRows []excelparser.STKRow, cccd string) *excelparser.STKRow {
	for i := range stkRows {
		if stkRows[i].CCCD == cccd {
			return &stkRows[i]
		}
	}
	return nil
}
