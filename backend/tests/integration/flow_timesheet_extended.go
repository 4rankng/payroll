package main

import (
	"fmt"
	"time"
)

const flowTimesheetExt = "TimesheetExtended"

func runTimesheetExtendedTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Timesheet Extended")

	admin := client.WithToken(data.AdminToken)

	// Use an existing project and employee for timesheet operations
	if data.WeeklyProject == nil || data.WeeklyEmployee == nil {
		reporter.Skip(flowTimesheetExt, "All tests", "no weekly project/employee available")
		return
	}

	// Cleanup only after discovery has established both prerequisites.
	cleanupEmployeeTimesheets(admin, data.WeeklyProject.ID, data.WeeklyEmployee.ID)

	projectID := data.WeeklyProject.ID
	employeeID := data.WeeklyEmployee.ID
	// Find an available date without existing timesheets, respecting assignment start date
	var minDate *time.Time
	if data.WeeklyAssignment != nil && data.WeeklyAssignment.StartDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", data.WeeklyAssignment.StartDate, time.Local); err == nil {
			minDate = &t
		}
	}
	timesheetDate := findAvailableDateWithMin(admin, projectID, employeeID, time.Now().AddDate(0, 0, -1), minDate)
	if timesheetDate == "" {
		reporter.Skip(flowTimesheetExt, "All tests", "no available date found for timesheet operations")
		return
	}
	fmt.Printf("    Using timesheet date: %s\n", timesheetDate)

	hourType := data.WeeklyHourType
	if hourType == "" {
		hourType = "Ca ngày"
	}
	reporter.RunTest(flowTimesheetExt, "Get timesheet summary", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/timesheets/summary", &resp); err != nil {
			return fmt.Errorf("timesheet summary: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowTimesheetExt, "Preview timesheet creation", func() error {
		body := []BulkCreateTimesheetEntry{
			{
				ProjectID:   projectID,
				EmployeeID:  employeeID,
				Date:        timesheetDate,
				HoursWorked: 8,
				HourType:    hourType,
			},
		}
		var resp interface{}
		if _, err := admin.PostInto("/api/v1/timesheets/preview", body, &resp); err != nil {
			return fmt.Errorf("preview: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowTimesheetExt, "Export entries template (binary)", func() error {
		data, _, statusCode, err := admin.DownloadPost("/api/v1/timesheets/export", map[string]any{
			"project_id": projectID,
		})
		if err != nil {
			return fmt.Errorf("export template: %w", err)
		}
		if err := AssertGreaterOrEqual("status", 200, statusCode); err != nil {
			return err
		}
		return AssertGreaterThan("file_size", 0, len(data))
	})

	reporter.RunTest(flowTimesheetExt, "Get grouped timesheets", func() error {
		var resp interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/timesheets/grouped?project_id=%d", projectID), &resp); err != nil {
			return fmt.Errorf("grouped timesheets: %w", err)
		}
		return nil
	})

	// Edge cases

	reporter.RunTest(flowTimesheetExt, "Edge: create timesheet with invalid project", func() error {
		body := []BulkCreateTimesheetEntry{
			{
				ProjectID:   nonexistentID,
				EmployeeID:  employeeID,
				Date:        today(),
				HoursWorked: 8,
				HourType:    hourType,
			},
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/timesheets", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowTimesheetExt, "Edge: delete non-existent timesheet", func() error {
		_, statusCode, _ := admin.Delete(fmt.Sprintf("/api/v1/timesheets/%d", nonexistentID))
		if statusCode < 400 {
			return fmt.Errorf("expected error deleting non-existent timesheet, got HTTP %d", statusCode)
		}
		return nil
	})
}
