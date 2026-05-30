package dto

import (
	"encoding/json"
	"time"
)

// TimesheetEntryTableResponse represents the response for timesheet entry table endpoint
type TimesheetEntryTableResponse struct {
	Data []EmployeeEntriesData `json:"data"`
}

// EmployeeEntriesData represents timesheet entries for a single employee
type EmployeeEntriesData struct {
	EmployeeID uint            `json:"employee_id"`
	Entries    []EntryResponse `json:"entries"`
}

// EntryResponse represents a single day's timesheet entry with day types and hour types
type EntryResponse struct {
	Date     string                        `json:"date"`
	DayTypes map[string]map[string]float64 `json:"-"` // Will be flattened in JSON marshaling
}

// MarshalJSON customizes the JSON output to flatten day types
func (e EntryResponse) MarshalJSON() ([]byte, error) {
	// Create a map that includes date and flattened day types
	result := map[string]interface{}{
		"date": e.Date,
	}

	// Add day types directly to the result (flattened)
	for dayType, hourTypes := range e.DayTypes {
		result[dayType] = hourTypes
	}

	return json.Marshal(result)
}

// TimesheetEntryTableQueryParams represents the query parameters for the entry table endpoint
type TimesheetEntryTableQueryParams struct {
	FromDate    string `form:"fromDate" binding:"required"`
	ToDate      string `form:"toDate" binding:"required"`
	EmployeeIDs string `form:"employeeIDs"` // Will be parsed as comma-separated values
	ProjectID   uint   // Not bound from query, set from path parameter
}

// EntryTableData represents the raw data structure used for processing
type EntryTableData struct {
	EmployeeID uint
	Date       time.Time
	DayType    string
	HourType   string
	Hours      float64
}
