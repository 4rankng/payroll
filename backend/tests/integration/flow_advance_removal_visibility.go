package main

import (
	"fmt"
	"time"
)

const flowAdvRemoval = "AdvanceRemovalVisibility"

func runAdvanceRemovalVisibilityTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW: Advance Payment - Employee Visibility After Removal")

	if data.EmployeeForAdvance == nil || data.EmployeeTokenForAdv == "" {
		reporter.Skip(flowAdvRemoval, "All tests", "no employee user account found")
		return
	}

	adminClient := client.WithToken(data.AdminToken)
	emp := data.EmployeeForAdvance

	var flexProject *EmployeeProjectInfo
	for i := range emp.CurrentProjects {
		p := &emp.CurrentProjects[i]
		if p.PaymentSchedule == "flexible" && p.LastDate == nil {
			flexProject = p
			break
		}
	}
	if flexProject == nil {
		reporter.Skip(flowAdvRemoval, "All tests", "employee has no active flexible project assignment")
		return
	}

	projectID := flexProject.ProjectID
	employeeID := emp.ID

	var originalForMonth string
	var found bool

	reporter.RunTest(flowAdvRemoval, "Record employee's forMonth from advance-payments/employees", func() error {
		var items []EmployeeAdvanceItem
		if _, err := adminClient.GetInto("/api/v1/advance-payments/employees?pageSize=100", &items); err != nil {
			return fmt.Errorf("list advance employees: %w", err)
		}
		for _, item := range items {
			if item.EmployeeID == employeeID && item.Project != nil && item.Project.ID == projectID {
				originalForMonth = item.ForMonth
				found = true
				fmt.Printf("    Employee %s (ID %d) found with forMonth=%s in project %d\n",
					emp.Fullname, employeeID, originalForMonth, projectID)
				break
			}
		}
		if !found {
			return fmt.Errorf("employee %d not found in advance-payments/employees for project %d", employeeID, projectID)
		}
		return nil
	})

	if !found {
		reporter.Skip(flowAdvRemoval, "All remaining tests", "employee not found in advance-payments/employees list")
		return
	}

	reporter.RunTest(flowAdvRemoval, "Baseline: employee visible for original forMonth before removal", func() error {
		var items []EmployeeAdvanceItem
		path := fmt.Sprintf("/api/v1/advance-payments/employees?forMonth=%s&pageSize=100", originalForMonth)
		if _, err := adminClient.GetInto(path, &items); err != nil {
			return fmt.Errorf("list for original month: %w", err)
		}
		for _, item := range items {
			if item.EmployeeID == employeeID && item.Project != nil && item.Project.ID == projectID {
				fmt.Printf("    Confirmed: employee visible for %s\n", originalForMonth)
				return nil
			}
		}
		return fmt.Errorf("employee %d not visible for original forMonth %s", employeeID, originalForMonth)
	})

	reporter.RunTest(flowAdvRemoval, "Remove employee from project", func() error {
		today := time.Now().Format("2006-01-02")
		removeBody := []map[string]any{
			{
				"employee_id": employeeID,
				"last_date":   today,
			},
		}
		if _, _, err := adminClient.Post(fmt.Sprintf("/api/v1/projects/%d/employees/remove", projectID), removeBody); err != nil {
			return fmt.Errorf("remove employee: %w", err)
		}
		fmt.Printf("    Removed employee %d from project %d with last_date=%s\n", employeeID, projectID, today)
		return nil
	})

	reporter.RunTest(flowAdvRemoval, "After removal: employee STILL visible for original forMonth", func() error {
		var items []EmployeeAdvanceItem
		path := fmt.Sprintf("/api/v1/advance-payments/employees?forMonth=%s&pageSize=100", originalForMonth)
		if _, err := adminClient.GetInto(path, &items); err != nil {
			return fmt.Errorf("list for original month after removal: %w", err)
		}
		for _, item := range items {
			if item.EmployeeID == employeeID && item.Project != nil && item.Project.ID == projectID {
				fmt.Printf("    PASS: employee still visible for %s after removal\n", originalForMonth)
				return nil
			}
		}
		return fmt.Errorf("BUG: employee %d disappeared from forMonth %s after removal (should still be visible)", employeeID, originalForMonth)
	})

	curMonth := currentMonth()
	if curMonth != originalForMonth {
		reporter.RunTest(flowAdvRemoval, "After removal: employee NOT visible for current month", func() error {
			var items []EmployeeAdvanceItem
			path := fmt.Sprintf("/api/v1/advance-payments/employees?forMonth=%s&pageSize=100", curMonth)
			if _, err := adminClient.GetInto(path, &items); err != nil {
				return fmt.Errorf("list for current month after removal: %w", err)
			}
			for _, item := range items {
				if item.EmployeeID == employeeID && item.Project != nil && item.Project.ID == projectID {
					return fmt.Errorf("BUG: employee %d still visible for current month %s after removal", employeeID, curMonth)
				}
			}
			fmt.Printf("    PASS: employee not visible for current month %s\n", curMonth)
			return nil
		})

		reporter.RunTest(flowAdvRemoval, "After removal: employee not in unfiltered list for current month", func() error {
			var items []EmployeeAdvanceItem
			if _, err := adminClient.GetInto("/api/v1/advance-payments/employees?pageSize=100", &items); err != nil {
				return fmt.Errorf("list all advance employees: %w", err)
			}
			for _, item := range items {
				if item.EmployeeID == employeeID && item.Project != nil && item.Project.ID == projectID && item.ForMonth == curMonth {
					return fmt.Errorf("BUG: employee found in unfiltered list with forMonth=%s", curMonth)
				}
			}
			fmt.Printf("    PASS: employee not in unfiltered list with current month forMonth\n")
			return nil
		})
	} else {
		reporter.Skip(flowAdvRemoval, "After removal: employee NOT visible for current month",
			fmt.Sprintf("current month (%s) == original forMonth, cannot verify non-visibility", curMonth))
		reporter.Skip(flowAdvRemoval, "After removal: employee not in unfiltered list for current month", "same as above")
	}

	reporter.RunTest(flowAdvRemoval, "Cleanup: re-add employee to project", func() error {
		addBody := []map[string]any{
			{
				"employee_id":      employeeID,
				"position":         "phổ thông",
				"payment_schedule": "flexible",
			},
		}
		if _, _, err := adminClient.Post(fmt.Sprintf("/api/v1/projects/%d/employees", projectID), addBody); err != nil {
			return fmt.Errorf("re-add employee: %w", err)
		}
		fmt.Printf("    Re-added employee %d to project %d\n", employeeID, projectID)
		return nil
	})

	reporter.RunTest(flowAdvRemoval, "Verify employee restored after re-add", func() error {
		var items []EmployeeAdvanceItem
		path := fmt.Sprintf("/api/v1/advance-payments/employees?forMonth=%s&pageSize=100", originalForMonth)
		if _, err := adminClient.GetInto(path, &items); err != nil {
			return fmt.Errorf("list for original month after restore: %w", err)
		}
		for _, item := range items {
			if item.EmployeeID == employeeID && item.Project != nil && item.Project.ID == projectID {
				fmt.Printf("    PASS: employee restored and visible for %s\n", originalForMonth)
				return nil
			}
		}
		return fmt.Errorf("employee %d not found after re-add for forMonth %s", employeeID, originalForMonth)
	})
}
