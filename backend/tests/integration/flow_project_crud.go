package main

import (
	"fmt"
)

const flowProject = "ProjectCRUD"

func runProjectCRUDTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Project CRUD")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()

	var testProjectID uint

	// --- Create ---

	reporter.RunTest(flowProject, "Create test project", func() error {
		body := CreateProjectRequest{
			ClientName: prefix + " Client",
			Name:       prefix + " Project",
			Code:       prefix,
		}
		var resp ProjectResponse
		if _, err := admin.PostInto("/api/v1/projects", body, &resp); err != nil {
			return fmt.Errorf("create project: %w", err)
		}
		testProjectID = resp.ID
		fmt.Printf("    Created project ID %d\n", resp.ID)
		if err := AssertGreaterThan("id", uint(0), resp.ID); err != nil {
			return err
		}
		return AssertEqual("code", prefix, resp.Code)
	})

	defer func() {
		if testProjectID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/projects/%d", testProjectID))
		}
	}()

	// --- Read ---

	reporter.RunTest(flowProject, "Get project by ID", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		var resp ProjectResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d", testProjectID), &resp); err != nil {
			return fmt.Errorf("get project: %w", err)
		}
		if err := AssertEqual("id", testProjectID, resp.ID); err != nil {
			return err
		}
		return AssertHasPrefix("code", resp.Code, prefix)
	})

	reporter.RunTest(flowProject, "List projects", func() error {
		var list []ProjectResponse
		if _, err := admin.GetInto("/api/v1/projects", &list); err != nil {
			return fmt.Errorf("list projects: %w", err)
		}
		fmt.Printf("    Found %d projects\n", len(list))
		return AssertSliceMinLen("projects", len(list), 1)
	})

	reporter.RunTest(flowProject, "Get project summary", func() error {
		var summary ProjectSummaryResponse
		if _, err := admin.GetInto("/api/v1/projects/summary", &summary); err != nil {
			return fmt.Errorf("project summary: %w", err)
		}
		fmt.Printf("    Active Projects: %d\n", summary.TotalActiveProjects)
		return AssertGreaterThan("total_projects", int64(0), summary.TotalActiveProjects)
	})

	reporter.RunTest(flowProject, "Get project partner summary", func() error {
		var summary interface{}
		if _, err := admin.GetInto("/api/v1/projects/partner-summary", &summary); err != nil {
			return fmt.Errorf("partner summary: %w", err)
		}
		return nil
	})

	// --- Update ---

	reporter.RunTest(flowProject, "Update project name", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		newName := prefix + " Updated Project"
		body := UpdateProjectRequest{Name: &newName}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/projects/%d", testProjectID), body); err != nil {
			return fmt.Errorf("update project: %w", err)
		}
		var resp ProjectResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d", testProjectID), &resp); err != nil {
			return fmt.Errorf("get after update: %w", err)
		}
		return AssertEqual("name", newName, resp.Name)
	})

	// --- Employee Assignment ---

	var assignmentID uint

	reporter.RunTest(flowProject, "Assign employee to project", func() error {
		if testProjectID == 0 || len(data.Employees) == 0 {
			return fmt.Errorf("missing test project or employees")
		}
		emp := data.Employees[0]
		req := []map[string]any{
			{
				"employee_id":      emp.ID,
				"position":         "Nhân viên test",
				"payment_schedule": "weekly",
			},
		}
		var resp []ProjectEmployeeResponse
		if _, err := admin.PostInto(fmt.Sprintf("/api/v1/projects/%d/employees", testProjectID), req, &resp); err != nil {
			return fmt.Errorf("assign employee: %w", err)
		}
		if len(resp) == 0 {
			return fmt.Errorf("no assignment returned")
		}
		assignmentID = resp[0].ID
		fmt.Printf("    Assigned employee %d to project %d (assignment %d)\n", emp.ID, testProjectID, assignmentID)
		return AssertGreaterThan("assignment_id", uint(0), assignmentID)
	})

	reporter.RunTest(flowProject, "List project employees", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		var list interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d/employees", testProjectID), &list); err != nil {
			return fmt.Errorf("list project employees: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowProject, "Remove employee from project", func() error {
		if testProjectID == 0 || len(data.Employees) == 0 {
			return fmt.Errorf("missing test project or employees")
		}
		emp := data.Employees[0]
		body := map[string]any{
			"employee_ids": []uint{emp.ID},
		}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/projects/%d/employees/remove", testProjectID), body); err != nil {
			return fmt.Errorf("remove employee: %w", err)
		}
		fmt.Printf("    Removed employee %d from project %d\n", emp.ID, testProjectID)
		return nil
	})

	// --- Assignment Update Tests (via PUT /employees/:id/projects) ---

	var updateAssignmentID uint
	var updateEmployee EmployeeResponse

	reporter.RunTest(flowProject, "Setup: assign employee for update tests", func() error {
		if testProjectID == 0 || len(data.Employees) < 2 {
			return fmt.Errorf("need test project and at least 2 employees")
		}
		updateEmployee = data.Employees[1]
		req := []map[string]any{
			{
				"employee_id":      updateEmployee.ID,
				"position":         "Ph\u1ed5 th\u00f4ng",
				"payment_schedule": "weekly",
				"start_date":       "2026-05-01",
			},
		}
		var resp []ProjectEmployeeResponse
		if _, err := admin.PostInto(fmt.Sprintf("/api/v1/projects/%d/employees", testProjectID), req, &resp); err != nil {
			return fmt.Errorf("assign employee for update test: %w", err)
		}
		if len(resp) == 0 {
			return fmt.Errorf("no assignment returned")
		}
		updateAssignmentID = resp[0].ID
		fmt.Printf("    Assigned employee %d for update tests (assignment %d)\n", updateEmployee.ID, updateAssignmentID)
		return nil
	})

	reporter.RunTest(flowProject, "Update assignment position", func() error {
		if testProjectID == 0 || updateEmployee.ID == 0 {
			return fmt.Errorf("missing test data")
		}
		newPos := "Co tay nghe"
		body := map[string]any{
			"project_id":  testProjectID,
			"employee_id": updateEmployee.ID,
			"position":    newPos,
		}
		var resp AssignmentResponse
		if _, err := admin.PutInto(fmt.Sprintf("/api/v1/employees/%d/projects", updateEmployee.ID), body, &resp); err != nil {
			return fmt.Errorf("update position: %w", err)
		}
		if err := AssertEqual("position", newPos, resp.Position); err != nil {
			return err
		}
		if err := AssertEqual("payment_schedule", "weekly", resp.PaymentSchedule); err != nil {
			return err
		}
		fmt.Printf("    Updated position, payment_schedule correctly returned as %s\n", resp.PaymentSchedule)
		return nil
	})

	reporter.RunTest(flowProject, "Update assignment payment_schedule", func() error {
		if testProjectID == 0 || updateEmployee.ID == 0 {
			return fmt.Errorf("missing test data")
		}
		body := map[string]any{
			"project_id":       testProjectID,
			"employee_id":      updateEmployee.ID,
			"payment_schedule": "monthly",
		}
		var resp AssignmentResponse
		if _, err := admin.PutInto(fmt.Sprintf("/api/v1/employees/%d/projects", updateEmployee.ID), body, &resp); err != nil {
			return fmt.Errorf("update payment_schedule: %w", err)
		}
		if err := AssertEqual("payment_schedule", "monthly", resp.PaymentSchedule); err != nil {
			return err
		}
		fmt.Printf("    Updated payment_schedule to monthly\n")
		return nil
	})

	reporter.RunTest(flowProject, "Update assignment start_date", func() error {
		if testProjectID == 0 || updateEmployee.ID == 0 {
			return fmt.Errorf("missing test data")
		}
		body := map[string]any{
			"project_id":  testProjectID,
			"employee_id": updateEmployee.ID,
			"start_date":  "2026-05-05",
		}
		var resp AssignmentResponse
		if _, err := admin.PutInto(fmt.Sprintf("/api/v1/employees/%d/projects", updateEmployee.ID), body, &resp); err != nil {
			return fmt.Errorf("update start_date: %w", err)
		}
		if err := AssertContains("start_date", resp.StartDate, "2026-05-05"); err != nil {
			return err
		}
		fmt.Printf("    Updated start_date to 2026-05-05\n")
		return nil
	})

	reporter.RunTest(flowProject, "Reject last_date before start_date", func() error {
		if testProjectID == 0 || updateEmployee.ID == 0 {
			return fmt.Errorf("missing test data")
		}
		body := map[string]any{
			"project_id":  testProjectID,
			"employee_id": updateEmployee.ID,
			"start_date":  "2026-05-10",
			"end_date":    "2026-05-01",
		}
		_, statusCode, _ := admin.Put(fmt.Sprintf("/api/v1/employees/%d/projects", updateEmployee.ID), body)
		if err := AssertGreaterOrEqual("status", 400, statusCode); err != nil {
			return err
		}
		fmt.Printf("    Correctly rejected end_date before start_date (HTTP %d)\n", statusCode)
		return nil
	})

	reporter.RunTest(flowProject, "Reject invalid payment_schedule", func() error {
		if testProjectID == 0 || updateEmployee.ID == 0 {
			return fmt.Errorf("missing test data")
		}
		body := map[string]any{
			"project_id":       testProjectID,
			"employee_id":      updateEmployee.ID,
			"payment_schedule": "bi-weekly",
		}
		_, statusCode, _ := admin.Put(fmt.Sprintf("/api/v1/employees/%d/projects", updateEmployee.ID), body)
		if err := AssertGreaterOrEqual("status", 400, statusCode); err != nil {
			return err
		}
		fmt.Printf("    Correctly rejected invalid payment_schedule (HTTP %d)\n", statusCode)
		return nil
	})

	reporter.RunTest(flowProject, "Clear last_date by setting end_date to empty", func() error {
		if testProjectID == 0 || updateEmployee.ID == 0 {
			return fmt.Errorf("missing test data")
		}
		setBody := map[string]any{
			"project_id":  testProjectID,
			"employee_id": updateEmployee.ID,
			"end_date":    "2026-06-30",
		}
		var setResp AssignmentResponse
		if _, err := admin.PutInto(fmt.Sprintf("/api/v1/employees/%d/projects", updateEmployee.ID), setBody, &setResp); err != nil {
			return fmt.Errorf("set last_date: %w", err)
		}
		if setResp.LastDate == nil {
			return fmt.Errorf("expected last_date to be set, got nil")
		}
		clearBody := map[string]any{
			"project_id":  testProjectID,
			"employee_id": updateEmployee.ID,
			"end_date":    "",
		}
		var clearResp AssignmentResponse
		if _, err := admin.PutInto(fmt.Sprintf("/api/v1/employees/%d/projects", updateEmployee.ID), clearBody, &clearResp); err != nil {
			return fmt.Errorf("clear last_date: %w", err)
		}
		if err := AssertEqual("last_date", (*string)(nil), clearResp.LastDate); err != nil {
			return err
		}
		fmt.Printf("    Correctly cleared last_date to null\n")
		return nil
	})

	reporter.RunTest(flowProject, "Cleanup: remove update test employee", func() error {
		if testProjectID == 0 || updateEmployee.ID == 0 {
			return nil
		}
		body := map[string]any{
			"employee_ids": []uint{updateEmployee.ID},
		}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/projects/%d/employees/remove", testProjectID), body); err != nil {
			return fmt.Errorf("cleanup remove employee: %w", err)
		}
		fmt.Printf("    Cleaned up employee %d from project %d\n", updateEmployee.ID, testProjectID)
		return nil
	})

	// --- Project Users ---

	reporter.RunTest(flowProject, "Get project users", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		var users interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d/users", testProjectID), &users); err != nil {
			return fmt.Errorf("get project users: %w", err)
		}
		return nil
	})

	// --- Project Timesheets ---

	reporter.RunTest(flowProject, "Get project timesheets", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		var resp interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d/timesheets?pageSize=5", testProjectID), &resp); err != nil {
			return fmt.Errorf("get project timesheets: %w", err)
		}
		return nil
	})

	// --- Project Payrate ---

	reporter.RunTest(flowProject, "Get project payrates", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		var resp interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d/payrate", testProjectID), &resp); err != nil {
			// May return 404 if no payrate exists - that's ok
			fmt.Printf("    No payrate for test project (expected): %v\n", err)
			return nil
		}
		return nil
	})

	// --- Delete ---

	reporter.RunTest(flowProject, "Delete test project", func() error {
		if testProjectID == 0 {
			return fmt.Errorf("no test project ID")
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/projects/%d", testProjectID)); err != nil {
			return fmt.Errorf("delete project: %w", err)
		}
		testProjectID = 0
		fmt.Printf("    Test project deleted\n")
		return nil
	})

	// --- Edge cases ---

	reporter.RunTest(flowProject, "Edge: create project with missing required fields", func() error {
		body := map[string]any{}
		_, statusCode, err := admin.PostExpectError("/api/v1/projects", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowProject, "Edge: get non-existent project", func() error {
		_, statusCode, _ := admin.Get("/api/v1/projects/999999")
		if statusCode < 400 {
			return fmt.Errorf("expected error for non-existent project, got HTTP %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowProject, "Edge: update non-existent project", func() error {
		newName := "Ghost"
		body := UpdateProjectRequest{Name: &newName}
		_, statusCode, _ := admin.Put("/api/v1/projects/999999", body)
		if statusCode < 400 {
			return fmt.Errorf("expected error updating non-existent project, got HTTP %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowProject, "Edge: delete non-existent project", func() error {
		_, statusCode, _ := admin.Delete("/api/v1/projects/999999")
		if statusCode < 400 {
			return fmt.Errorf("expected error deleting non-existent project, got HTTP %d", statusCode)
		}
		return nil
	})

	// --- Salary Period Change Guard ---

	// Create a dedicated project for salary period change tests
	var salaryTestProjectID uint

	reporter.RunTest(flowProject, "Setup: create project for salary period test", func() error {
		body := CreateProjectRequest{
			ClientName: prefix + " Salary Guard Client",
			Name:       prefix + " Salary Guard Project",
			Code:       prefix + "SG",
		}
		var resp ProjectResponse
		if _, err := admin.PostInto("/api/v1/projects", body, &resp); err != nil {
			return fmt.Errorf("create salary test project: %w", err)
		}
		salaryTestProjectID = resp.ID
		fmt.Printf("    Created salary test project ID %d\n", resp.ID)
		return AssertGreaterThan("id", uint(0), resp.ID)
	})

	defer func() {
		if salaryTestProjectID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/projects/%d", salaryTestProjectID))
		}
	}()

	reporter.RunTest(flowProject, "Salary period change allowed when no unsettled timesheets", func() error {
		if salaryTestProjectID == 0 {
			return fmt.Errorf("no salary test project")
		}
		spFrom := 26
		spTo := 25
		body := UpdateProjectRequest{SalaryPeriodFrom: &spFrom, SalaryPeriodTo: &spTo}
		var resp ProjectResponse
		if _, err := admin.PutInto(fmt.Sprintf("/api/v1/projects/%d", salaryTestProjectID), body, &resp); err != nil {
			return fmt.Errorf("update salary period (no unsettled): %w", err)
		}
		if err := AssertEqual("salary_period_from", 26, resp.SalaryPeriodFrom); err != nil {
			return err
		}
		return AssertEqual("salary_period_to", 25, resp.SalaryPeriodTo)
	})

	reporter.RunTest(flowProject, "Admin can change salary period", func() error {
		projects := data.Projects
		if len(projects) == 0 {
			fmt.Printf("    No projects available — skipping\n")
			return nil
		}

		p := projects[0]
		var current ProjectResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/projects/%d", p.ID), &current); err != nil {
			return err
		}
		origFrom := current.SalaryPeriodFrom
		origTo := current.SalaryPeriodTo

		newFrom := origFrom + 1
		if newFrom > 28 {
			newFrom = 0
		}

		var updated ProjectResponse
		if _, err := admin.PutInto(fmt.Sprintf("/api/v1/projects/%d", p.ID), UpdateProjectRequest{
			SalaryPeriodFrom: &newFrom,
		}, &updated); err != nil {
			return fmt.Errorf("admin should be able to change salary period: %w", err)
		}

		// Restore original value
		_, _, _ = admin.Put(fmt.Sprintf("/api/v1/projects/%d", p.ID), UpdateProjectRequest{
			SalaryPeriodFrom: &origFrom,
			SalaryPeriodTo:   &origTo,
		})

		fmt.Printf("    Admin correctly changed salary period for project %d\n", p.ID)
		return nil
	})
}
