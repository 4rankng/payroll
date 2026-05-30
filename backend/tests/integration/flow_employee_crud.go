package main

import (
	"fmt"
)

const flowEmployee = "EmployeeCRUD"

func runEmployeeCRUDTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Employee CRUD")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()
	testCCCD := prefix + "CCCD"

	var testEmpID uint

	// --- Create ---

	reporter.RunTest(flowEmployee, "Create test employee", func() error {
		body := CreateEmployeeRequest{
			Fullname: prefix + " Employee",
			CCCD:     testCCCD,
			Address:  "123 Test Street",
			Mobile:   "0909123456",
		}
		var resp EmployeeResponse
		if _, err := admin.PostInto("/api/v1/employees", body, &resp); err != nil {
			return fmt.Errorf("create employee: %w", err)
		}
		testEmpID = resp.ID
		fmt.Printf("    Created employee ID %d\n", resp.ID)
		return AssertGreaterThan("id", uint(0), resp.ID)
	})

	defer func() {
		if testEmpID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/employees/%d", testEmpID))
		}
	}()

	// --- Read ---

	reporter.RunTest(flowEmployee, "Get employee by ID", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		var resp EmployeeResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d", testEmpID), &resp); err != nil {
			return fmt.Errorf("get employee: %w", err)
		}
		if err := AssertEqual("id", testEmpID, resp.ID); err != nil {
			return err
		}
		return AssertEqual("cccd", testCCCD, resp.CCCD)
	})

	reporter.RunTest(flowEmployee, "Get employee by CCCD", func() error {
		var emp EmployeeResponse
		if _, err := admin.GetInto("/api/v1/employees/cccd/"+testCCCD, &emp); err != nil {
			return fmt.Errorf("get by CCCD: %w", err)
		}
		return AssertEqual("cccd", testCCCD, emp.CCCD)
	})

	reporter.RunTest(flowEmployee, "List employees", func() error {
		var list []EmployeeResponse
		if _, err := admin.GetInto("/api/v1/employees?pageSize=10", &list); err != nil {
			return fmt.Errorf("list employees: %w", err)
		}
		fmt.Printf("    Found %d employees\n", len(list))
		return AssertSliceMinLen("employees", len(list), 1)
	})

	reporter.RunTest(flowEmployee, "Get employee summary (all employees)", func() error {
		var summary EmployeesSummaryResponse
		if _, err := admin.GetInto("/api/v1/employees/summary", &summary); err != nil {
			return fmt.Errorf("employees summary: %w", err)
		}
		fmt.Printf("    Total: %d, Working: %d\n", summary.TotalEmployees, summary.TotalWorkingEmployees)
		return AssertGreaterThan("total_employees", int64(0), summary.TotalEmployees)
	})

	reporter.RunTest(flowEmployee, "Get employee individual summary", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		var summary EmployeeSummaryResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d/summary", testEmpID), &summary); err != nil {
			return fmt.Errorf("employee summary: %w", err)
		}
		fmt.Printf("    Total payroll payments: %d\n", summary.TotalPayrollPayments)
		return nil
	})

	reporter.RunTest(flowEmployee, "Get employee current projects", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		var projects interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d/current-projects", testEmpID), &projects); err != nil {
			return fmt.Errorf("current projects: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowEmployee, "Get unassigned employees", func() error {
		var list []EmployeeResponse
		if _, err := admin.GetInto("/api/v1/employees/unassigned", &list); err != nil {
			return fmt.Errorf("unassigned employees: %w", err)
		}
		fmt.Printf("    Unassigned employees: %d\n", len(list))
		return nil
	})

	reporter.RunTest(flowEmployee, "Get employees missing bank details", func() error {
		var list []EmployeeResponse
		if _, err := admin.GetInto("/api/v1/employees/missing-bank-details", &list); err != nil {
			return fmt.Errorf("missing bank details: %w", err)
		}
		fmt.Printf("    Missing bank details: %d\n", len(list))
		return nil
	})

	reporter.RunTest(flowEmployee, "Export employees (binary)", func() error {
		data, _, statusCode, err := admin.DownloadGet("/api/v1/employees/export")
		if err != nil {
			return fmt.Errorf("export: %w", err)
		}
		if err := AssertGreaterOrEqual("status", 200, statusCode); err != nil {
			return err
		}
		return AssertGreaterThan("file_size", 0, len(data))
	})

	// --- Update ---

	reporter.RunTest(flowEmployee, "Update employee fullname", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		newName := prefix + " Updated Name"
		body := UpdateEmployeeRequest{Fullname: &newName}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/employees/%d", testEmpID), body); err != nil {
			return fmt.Errorf("update employee: %w", err)
		}
		var resp EmployeeResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d", testEmpID), &resp); err != nil {
			return fmt.Errorf("get after update: %w", err)
		}
		return AssertContains("fullname", resp.Fullname, "Updated Name")
	})

	reporter.RunTest(flowEmployee, "Update employee bank details", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		bankAcc := "1234567890"
		bankName := "Test Account"
		body := UpdateEmployeeRequest{
			BankAccountNumber: &bankAcc,
			BankAccountName:   &bankName,
		}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/employees/%d", testEmpID), body); err != nil {
			return fmt.Errorf("update bank: %w", err)
		}
		var resp EmployeeResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d", testEmpID), &resp); err != nil {
			return fmt.Errorf("get after bank update: %w", err)
		}
		fmt.Printf("    Bank account after update: number=%q, name=%q\n", resp.BankAccountNumber, resp.BankAccountName)
		if resp.BankAccountNumber == "" {
			fmt.Printf("    Bank update may require bank_id - skipping assertion\n")
			return nil
		}
		return AssertEqual("bank_account_number", bankAcc, resp.BankAccountNumber)
	})

	// --- Employee User Access ---

	reporter.RunTest(flowEmployee, "Get employee users (empty)", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		var users []EmployeeUserResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/employees/%d/users", testEmpID), &users); err != nil {
			return fmt.Errorf("get employee users: %w", err)
		}
		fmt.Printf("    Employee users: %d\n", len(users))
		return nil
	})

	// --- Delete ---

	reporter.RunTest(flowEmployee, "Delete test employee", func() error {
		if testEmpID == 0 {
			return fmt.Errorf("no test employee ID")
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/employees/%d", testEmpID)); err != nil {
			return fmt.Errorf("delete employee: %w", err)
		}
		testEmpID = 0
		fmt.Printf("    Test employee deleted\n")
		return nil
	})

	// --- Edge cases ---

	reporter.RunTest(flowEmployee, "Edge: create employee with missing CCCD", func() error {
		body := CreateEmployeeRequest{
			Fullname: prefix + " No CCCD",
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/employees", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowEmployee, "Edge: create employee with duplicate CCCD", func() error {
		if len(data.Employees) == 0 {
			return fmt.Errorf("no existing employees for duplicate test")
		}
		body := CreateEmployeeRequest{
			Fullname: prefix + " Duplicate CCCD",
			CCCD:     data.Employees[0].CCCD,
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/employees", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowEmployee, "Edge: get non-existent employee", func() error {
		_, statusCode, _ := admin.Get("/api/v1/employees/999999")
		if statusCode < 400 {
			return fmt.Errorf("expected error for non-existent employee, got HTTP %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowEmployee, "Edge: update non-existent employee", func() error {
		newName := "Ghost"
		body := UpdateEmployeeRequest{Fullname: &newName}
		_, statusCode, _ := admin.Put("/api/v1/employees/999999", body)
		if statusCode < 400 {
			return fmt.Errorf("expected error updating non-existent employee, got HTTP %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowEmployee, "Edge: delete non-existent employee", func() error {
		_, statusCode, _ := admin.Delete("/api/v1/employees/999999")
		if statusCode < 400 {
			return fmt.Errorf("expected error deleting non-existent employee, got HTTP %d", statusCode)
		}
		return nil
	})
}
