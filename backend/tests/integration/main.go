package main

import (
	"api-server/internal/pkg/clock"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func main() {
	cfg := LoadTestConfig()
	reporter := NewReporter()

	// Set up dual output: stdout + timestamped log file
	resultsDir := filepath.Join("tests", "integration", "results")
	_ = os.MkdirAll(resultsDir, 0o755)
	logFileName := fmt.Sprintf("test-result-%s.log", time.Now().Format("20060102-150405"))
	logFile, err := os.Create(filepath.Join(resultsDir, logFileName))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create log file: %v\n", err)
	} else {
		defer func() { _ = logFile.Close() }()
		reporter.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}

	reporter.PrintHeader(cfg.BaseURL)

	client := NewAPIClient(cfg.BaseURL)

	// Reset server clock at start to ensure no state leakage from prior runs.
	// This is a no-op if the clock endpoint is unavailable (e.g., production).
	_ = ResetServerTime(client)

	// Phase 1: Discovery
	reporter.PrintSection("DISCOVERY: Test Data")
	testData, err := discoverTestData(client, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  FATAL: discovery failed: %v\n\n", err)
		os.Exit(2)
	}
	reporter.PrintDiscovery(testData)

	// Phase 1.5: Clock manipulation (admin-only, tests time control endpoints)
	runClockManipulationTests(client, reporter)

	// Phase 2: Run Flow 1 - Bulk Transfer (9Pay auto)
	runBulkTransferTests(client, testData, reporter)

	// Phase 2.5: Wallet Bulk Transfer Pipeline (OnePay) — Stage 2 upload/process.
	runWalletBulkTransferTests(client, testData, reporter)

	// Phase 3: Run Flow 4 - Partner creates timesheets → Admin approves
	runPartnerTimesheetTests(client, testData, reporter)

	// Phase 3.5: Partner BCC Excel import
	runBCCImportTests(client, testData, reporter)

	// Phase 3.6: Weekly BCC (BCC-<shiftType> sheet format)
	runWeeklyBCCImportTests(client, testData, reporter)

	// Phase 3.7: Weekly Payment (Thai Binh Duong format - numeric sheets)
	runWeeklyPaymentImportTests(client, testData, reporter)

	// Phase 4: Run Flow 5 - Manual bulk transfer export & import
	runManualBulkTransferTests(client, testData, reporter)

	// Settlement simulation — read-only, safe to run anytime after bulk transfer.
	runSettlementSimulationTests(client, testData, reporter, cfg)

	// Phase 5: Run Flow 3 - FlexPay Import (must run before advance payment)
	runFlexPayImportTests(client, testData, reporter, cfg)

	// Phase 6: Run Flow 2 - Advance Payment (needs FlexPay data)
	runAdvancePaymentTests(client, testData, reporter)

	// Phase 6.5: Advance Payment - Removal Visibility
	runAdvanceRemovalVisibilityTests(client, testData, reporter)

	// Phase 6.7: Advance request kill switch (tạm ngừng ứng lương)
	runAdvanceRequestKillSwitchTests(client, testData, reporter)

	// Phase 7: Auth & User Management
	runAuthUserTests(client, testData, reporter, cfg)

	// Phase 7.5: Self-service password reset (no-token contracts; see flow file)
	runPasswordResetTests(client, testData, reporter, cfg)

	// Phase 7.6: Integration API (chatbot API-key channel)
	runIntegrationResetTests(client, testData, reporter, cfg)

	// Phase 8: CRUD flows
	runBankCRUDTests(client, testData, reporter, cfg)
	runEmployeeCRUDTests(client, testData, reporter, cfg)
	runProjectCRUDTests(client, testData, reporter, cfg)

	// Phase 9: Domain flows
	runPayrateCRUDTests(client, testData, reporter, cfg)
	runPayratePartnerVisibilityTests(client, testData, reporter, cfg)
	runSettingsTests(client, testData, reporter, cfg)
	runAssetTests(client, testData, reporter, cfg)
	runLoanTests(client, testData, reporter, cfg)
	runLedgerTests(client, testData, reporter, cfg)
	runTransactionTests(client, testData, reporter, cfg)

	// Phase 10: Read-only flows
	runNotificationTests(client, testData, reporter, cfg)
	runDashboardTests(client, testData, reporter, cfg)
	runAuditTests(client, testData, reporter, cfg)
	runCronTests(client, testData, reporter, cfg)
	runMetricsTests(client, testData, reporter, cfg)
	runWalletTests(client, testData, reporter, cfg)
	runDisbursementTests(client, testData, reporter, cfg)

	// Phase 11: Infrastructure invariants (schema, asynq migration guard)
	runInfraTests(reporter)

	// Phase 12: Sao kê (statement email & settlement)
	runSaoKeTests(client, testData, reporter, cfg)

	// Phase 11: Extended flows
	runEmployeeSelfServiceTests(client, testData, reporter, cfg)
	runTimesheetExtendedTests(client, testData, reporter, cfg)
	runAdBannerTests(client, testData, reporter, cfg)

	// Summary
	reporter.PrintSummary()
	os.Exit(reporter.ExitCode())
}

func discoverTestData(client *APIClient, cfg *TestConfig) (*TestData, error) {
	data := &TestData{}

	// Login as admin
	loginResp, err := client.Login(cfg.AdminUsername, cfg.AdminPassword)
	if err != nil {
		return nil, fmt.Errorf("admin login: %w", err)
	}
	data.AdminToken = loginResp.AccessToken
	fmt.Printf("  Logged in as admin: %s (ID %d)\n", loginResp.User.Username, loginResp.User.ID)

	// Fetch projects
	if data.Projects, err = loadAllPages[ProjectResponse](client, "/api/v1/projects"); err != nil {
		return nil, fmt.Errorf("fetch projects: %w", err)
	}
	fmt.Printf("  Found %d projects\n", len(data.Projects))

	// Fetch employees
	if data.Employees, err = loadAllPages[EmployeeResponse](client, "/api/v1/employees"); err != nil {
		return nil, fmt.Errorf("fetch employees: %w", err)
	}
	fmt.Printf("  Found %d employees\n", len(data.Employees))

	// Fetch banks
	if data.Banks, err = loadAllPages[BankResponse](client, "/api/v1/banks"); err != nil {
		return nil, fmt.Errorf("fetch banks: %w", err)
	}
	fmt.Printf("  Found %d banks\n", len(data.Banks))

	// Identify weekly/monthly projects and employees
	for i := range data.Projects {
		p := &data.Projects[i]
		if p.WeeklySalaryEmployeeCount > 0 && data.WeeklyProject == nil {
			data.WeeklyProject = p
		}
		if p.MonthlySalaryEmployeeCount > 0 && data.MonthlyProject == nil {
			data.MonthlyProject = p
		}
	}

	// Discover payrate hour/day types for the weekly project
	if data.WeeklyProject != nil {
		discoverPayrateTypes(client, data)
	}

	// If no monthly project exists, assign a weekly employee to a project with monthly schedule
	if data.MonthlyProject == nil && data.WeeklyProject != nil {
		if err := setupMonthlyData(client, data); err != nil {
			fmt.Printf("  Warning: could not set up monthly test data: %v\n", err)
		}
	}

	// Find employees for weekly/monthly projects
	// Match employee to the specific project they need to be assigned to
	for i := range data.Employees {
		emp := &data.Employees[i]
		if emp.BankAccountNumber == "" {
			continue
		}
		for _, proj := range emp.CurrentProjects {
			if data.WeeklyProject != nil && proj.ProjectID == data.WeeklyProject.ID &&
				proj.PaymentSchedule == "weekly" && data.WeeklyEmployee == nil {
				data.WeeklyEmployee = emp
				projCopy := proj
				data.WeeklyAssignment = &projCopy
			}
			if data.MonthlyProject != nil && proj.ProjectID == data.MonthlyProject.ID &&
				proj.PaymentSchedule == "monthly" && data.MonthlyEmployee == nil {
				data.MonthlyEmployee = emp
			}
		}
	}

	// Fallback: if no employee matched the project specifically, pick any with the right schedule
	if data.WeeklyEmployee == nil {
		for i := range data.Employees {
			emp := &data.Employees[i]
			if emp.BankAccountNumber == "" {
				continue
			}
			for _, proj := range emp.CurrentProjects {
				if proj.PaymentSchedule == "weekly" {
					data.WeeklyEmployee = emp
					projCopy := proj
					data.WeeklyAssignment = &projCopy
					break
				}
			}
			if data.WeeklyEmployee != nil {
				break
			}
		}
	}
	if data.MonthlyEmployee == nil {
		for i := range data.Employees {
			emp := &data.Employees[i]
			if emp.BankAccountNumber == "" {
				continue
			}
			for _, proj := range emp.CurrentProjects {
				if proj.PaymentSchedule == "monthly" {
					data.MonthlyEmployee = emp
					break
				}
			}
			if data.MonthlyEmployee != nil {
				break
			}
		}
	}

	// Find employee user for advance payment testing
	if err := findEmployeeForAdvance(client, data, cfg); err != nil {
		fmt.Printf("  Warning: could not find employee for advance payment: %v\n", err)
	}

	// Find partner users
	findPartners(client, data, cfg)

	return data, nil
}

func findEmployeeForAdvance(client *APIClient, data *TestData, cfg *TestConfig) error {
	// First, check admin endpoint for employees with available advance amounts
	var advanceEmployees []EmployeeAdvanceItem
	if _, err := client.GetInto("/api/v1/advance-payments/employees", &advanceEmployees); err == nil {
		for _, ae := range advanceEmployees {
			if ae.AvailableAmount == 0 || ae.MaxAdvanceAmount == 0 {
				continue
			}
			// The cutoff scenarios exercise imported monthly quotas. Self check-in
			// assignments deliberately use a different eligibility policy.
			if ae.Project != nil && ae.Project.CheckInEnabled {
				continue
			}
			// Try to login as this employee
			empClient := NewAPIClient(client.BaseURL)
			loginResp, err := empClient.Login(ae.Username, cfg.CommonPassword)
			if err != nil {
				continue
			}

			// Get employee detail
			var detail EmployeeDetailedResponse
			if _, err := client.GetInto(fmt.Sprintf("/api/v1/employees/%d", ae.EmployeeID), &detail); err != nil {
				continue
			}
			if !hasImportedAdvanceAssignment(detail.CurrentProjects) {
				continue
			}

			data.EmployeeForAdvance = &detail
			data.EmployeeTokenForAdv = loginResp.AccessToken
			fmt.Printf("  Found advance payment employee: %s (username: %s, available: %d)\n",
				ae.Fullname, ae.Username, ae.AvailableAmount)
			return nil
		}
	}

	// Fallback: look for employees with flexible payment schedule who have user accounts
	for _, emp := range data.Employees {
		if !hasImportedAdvanceAssignment(emp.CurrentProjects) {
			continue
		}

		var detail EmployeeDetailedResponse
		if _, err := client.GetInto(fmt.Sprintf("/api/v1/employees/%d", emp.ID), &detail); err != nil {
			continue
		}
		if detail.Username == nil || *detail.Username == "" {
			continue
		}

		empClient := NewAPIClient(client.BaseURL)
		loginResp, err := empClient.Login(*detail.Username, cfg.CommonPassword)
		if err != nil {
			continue
		}

		data.EmployeeForAdvance = &detail
		data.EmployeeTokenForAdv = loginResp.AccessToken
		fmt.Printf("  Found advance payment employee: %s (username: %s) - note: may not have available amount\n",
			emp.Fullname, *detail.Username)
		return nil
	}

	return fmt.Errorf("no employee user accounts found")
}

func findPartners(client *APIClient, data *TestData, cfg *TestConfig) {
	for _, username := range cfg.PartnerUsernames {
		partnerClient := NewAPIClient(client.BaseURL)
		loginResp, err := partnerClient.Login(username, cfg.CommonPassword)
		if err != nil {
			fmt.Printf("  Could not login as partner %s: %v\n", username, err)
			continue
		}
		data.Partners = append(data.Partners, PartnerUser{
			ID:       loginResp.User.ID,
			Username: username,
			Fullname: loginResp.User.Fullname,
			Token:    loginResp.AccessToken,
		})
		fmt.Printf("  Found partner: %s (%s)\n", loginResp.User.Fullname, username)
	}
}

func setupMonthlyData(client *APIClient, data *TestData) error {
	// Find a weekly employee with bank details not already in the weekly project
	var candidate *EmployeeResponse
	for i := range data.Employees {
		emp := &data.Employees[i]
		if emp.BankAccountNumber == "" {
			continue
		}
		hasWeekly := false
		inProject := false
		for _, proj := range emp.CurrentProjects {
			if proj.PaymentSchedule == "weekly" {
				hasWeekly = true
			}
			if data.WeeklyProject != nil && proj.ProjectID == data.WeeklyProject.ID {
				inProject = true
			}
		}
		if hasWeekly && !inProject {
			candidate = emp
			break
		}
	}

	if candidate == nil {
		return fmt.Errorf("no suitable employee found for monthly assignment")
	}

	projectID := data.WeeklyProject.ID
	fmt.Printf("  Setting up monthly data: assigning %s (ID %d) to project %d as monthly\n",
		candidate.Fullname, candidate.ID, projectID)

	req := []map[string]any{
		{
			"employee_id":      candidate.ID,
			"position":         "phổ thông",
			"payment_schedule": "monthly",
		},
	}
	if _, _, err := client.Post(fmt.Sprintf("/api/v1/projects/%d/employees", projectID), req); err != nil {
		return fmt.Errorf("assign monthly employee: %w", err)
	}

	// Re-fetch projects and employees to pick up the change
	var err error
	if data.Projects, err = loadAllPages[ProjectResponse](client, "/api/v1/projects"); err != nil {
		return fmt.Errorf("re-fetch projects: %w", err)
	}
	if data.Employees, err = loadAllPages[EmployeeResponse](client, "/api/v1/employees"); err != nil {
		return fmt.Errorf("re-fetch employees: %w", err)
	}

	// Re-identify monthly project
	for i := range data.Projects {
		p := &data.Projects[i]
		if p.MonthlySalaryEmployeeCount > 0 && data.MonthlyProject == nil {
			data.MonthlyProject = p
		}
	}

	return nil
}

func today() string {
	return clock.Now().Format("2006-01-02")
}

// discoverPayrateTypes fetches the current payrate config for the weekly project
// and extracts the first available hour type and day type with a non-zero rate.
func discoverPayrateTypes(client *APIClient, data *TestData) {
	var payrates []PayrateResponse
	if _, err := client.GetInto(fmt.Sprintf("/api/v1/payrates?project_id=%d", data.WeeklyProject.ID), &payrates); err != nil {
		fmt.Printf("  Warning: could not fetch payrate for project %d: %v\n", data.WeeklyProject.ID, err)
		return
	}
	if len(payrates) == 0 {
		return
	}

	var rates map[string]map[string]map[string]float64
	if err := json.Unmarshal(payrates[0].Rates, &rates); err != nil {
		return
	}
	for _, dayTypes := range rates {
		for dayType, hourTypes := range dayTypes {
			for hourType, rate := range hourTypes {
				if rate > 0 {
					data.WeeklyDayType = dayType
					data.WeeklyHourType = hourType
					fmt.Printf("  Weekly project payrate: dayType=%q hourType=%q rate=%.0f\n", dayType, hourType, rate)
					return
				}
			}
		}
	}
}

func weekAgo() string {
	return clock.Now().AddDate(0, 0, -7).Format("2006-01-02")
}

func currentMonth() string {
	return time.Now().Format("2006-01")
}

// cleanupEmployeeTimesheets deletes non-paid timesheets for a given employee in a project.
// This ensures test idempotency by removing leftover approved-but-unpaid timesheets from prior runs.
func cleanupEmployeeTimesheets(client *APIClient, projectID, employeeID uint) {
	path := fmt.Sprintf("/api/v1/timesheets?project_ids=%d&employee_id=%d&pageSize=200",
		projectID, employeeID)
	var timesheets []TimesheetWithDetailsResponse
	if _, err := client.GetInto(path, &timesheets); err != nil {
		return
	}
	for _, ts := range timesheets {
		if ts.PaymentStatus == "paid" {
			continue
		}
		deletePath := fmt.Sprintf("/api/v1/timesheets/%d", ts.ID)
		_, _, _ = client.Delete(deletePath)
	}
}

func findAvailableDateWithMin(client *APIClient, projectID, employeeID uint, start time.Time, minDate *time.Time) string {
	// Get ALL timesheets for this employee in the project (no date filter due to timezone issues)
	path := fmt.Sprintf("/api/v1/timesheets?project_ids=%d&employee_id=%d&pageSize=200",
		projectID, employeeID)
	var timesheets []TimesheetWithDetailsResponse
	if _, err := client.GetInto(path, &timesheets); err != nil {
		return ""
	}

	// Build set of occupied dates (only non-deleted, since deleted timesheets free up the date)
	occupied := make(map[string]bool)
	for _, ts := range timesheets {
		occupied[ts.Date] = true
	}

	// Search backward from start, respecting minDate
	for i := 0; i < 365; i++ {
		d := start.AddDate(0, 0, -i)
		if d.Weekday() == time.Sunday {
			continue
		}
		if minDate != nil && d.Before(*minDate) {
			break
		}
		dateStr := d.Format("2006-01-02")
		if !occupied[dateStr] {
			return dateStr
		}
	}

	// Search forward (but not into the future)
	today := time.Now()
	for i := 1; i < 365; i++ {
		d := start.AddDate(0, 0, i)
		if d.After(today) {
			break
		}
		if d.Weekday() == time.Sunday {
			continue
		}
		dateStr := d.Format("2006-01-02")
		if !occupied[dateStr] {
			return dateStr
		}
	}
	return ""
}
