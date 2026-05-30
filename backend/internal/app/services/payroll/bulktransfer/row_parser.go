package bulktransfer

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"api-server/internal/constants"
)

// RowParser handles parsing of bulk transfer result row data
type RowParser struct{}

// NewRowParser creates a new row parser instance
func NewRowParser() *RowParser {
	return &RowParser{}
}

// ParseTrackingData parses the tracking data format to extract project ID and timesheet IDs
// Expected format: "[project_id][timesheet_id1 timesheet_id2 ...]"
// Returns: projectID, timesheetIDs, error
func (rp *RowParser) ParseTrackingData(trackingData string) (uint, []uint, error) {
	if trackingData == "" {
		return 0, nil, fmt.Errorf("tracking data is empty")
	}

	// Regular expression to match format: [projectID][id1 id2 id3 ...]
	re := regexp.MustCompile(`^\[(\d+)\]\[([0-9\s]+)\]$`)
	matches := re.FindStringSubmatch(trackingData)

	if len(matches) != 3 {
		return 0, nil, fmt.Errorf("tracking data does not match expected format '[projectID][id1 id2 ...]'")
	}

	// Parse project ID
	var projectID uint
	if _, err := fmt.Sscanf(matches[1], "%d", &projectID); err != nil {
		return 0, nil, fmt.Errorf("invalid project ID in tracking data: %v", err)
	}

	// Parse timesheet IDs
	idsStr := strings.TrimSpace(matches[2])
	if idsStr == "" {
		return 0, nil, fmt.Errorf("no timesheet IDs found in tracking data")
	}

	idStrings := strings.Fields(idsStr)
	timesheetIDs := make([]uint, 0, len(idStrings))
	for _, idStr := range idStrings {
		var id uint
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			return 0, nil, fmt.Errorf("invalid timesheet ID '%s' in tracking data: %v", idStr, err)
		}
		timesheetIDs = append(timesheetIDs, id)
	}

	return projectID, timesheetIDs, nil
}

// ParsePaymentPeriodWithProject parses the payment description format with project information
// Expected format: "{project name} [{project_id}][YYYY-MM-DD den YYYY-MM-DD]"
// Returns: projectName, projectID, fromDate, toDate, error
func (rp *RowParser) ParsePaymentPeriodWithProject(description string) (string, uint, time.Time, time.Time, error) {
	// Regular expression to match the format: project name, project ID, dates
	re := regexp.MustCompile(`^(.+?)\s*\[(\d+)\]\s*\[(\d{4}-\d{2}-\d{2})\s+den\s+(\d{4}-\d{2}-\d{2})\]$`)
	matches := re.FindStringSubmatch(description)

	if len(matches) != 5 {
		return "", 0, time.Time{}, time.Time{}, fmt.Errorf("description does not match expected format '{project name} [{project_id}][YYYY-MM-DD den YYYY-MM-DD]'")
	}

	projectName := strings.TrimSpace(matches[1])
	projectIDStr := matches[2]
	fromDateStr := matches[3]
	toDateStr := matches[4]

	// Parse project ID
	var projectID uint
	if _, err := fmt.Sscanf(projectIDStr, "%d", &projectID); err != nil {
		return "", 0, time.Time{}, time.Time{}, fmt.Errorf("invalid project ID format: %v", err)
	}

	// Parse dates
	loc, _ := time.LoadLocation("Local")
	fromDate, err := time.ParseInLocation("2006-01-02", fromDateStr, loc)
	if err != nil {
		return "", 0, time.Time{}, time.Time{}, fmt.Errorf("invalid from date format: %v", err)
	}

	toDate, err := time.ParseInLocation("2006-01-02", toDateStr, loc)
	if err != nil {
		return "", 0, time.Time{}, time.Time{}, fmt.Errorf("invalid to date format: %v", err)
	}

	if fromDate.After(toDate) {
		return "", 0, time.Time{}, time.Time{}, fmt.Errorf(constants.MsgFromDateAfterToDateVN)
	}

	return projectName, projectID, fromDate, toDate, nil
}

// ParseAmountFromDetail parses amount string from detail, handling various formats
func (rp *RowParser) ParseAmountFromDetail(amountStr string) (int64, error) {
	// Remove common formatting characters
	cleanAmount := strings.ReplaceAll(amountStr, ",", "")
	cleanAmount = strings.ReplaceAll(cleanAmount, ".", "")
	cleanAmount = strings.ReplaceAll(cleanAmount, " ", "")
	cleanAmount = strings.TrimSpace(cleanAmount)

	// Handle empty or invalid amounts
	if cleanAmount == "" || cleanAmount == "0" {
		return 0, nil
	}

	var amount int64
	if _, err := fmt.Sscanf(cleanAmount, "%d", &amount); err != nil {
		return 0, fmt.Errorf("invalid amount format: %s", amountStr)
	}

	return amount, nil
}

// IsEmptyRow checks if a row is empty by verifying all cells are blank
func (rp *RowParser) IsEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
