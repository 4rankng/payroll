package main

import (
	"fmt"
)

const flowLoan = "Loan"

func runLoanTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Loan & Lender Management")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()

	var testLenderID uint
	var testLoanID uint

	// --- Lender CRUD ---

	reporter.RunTest(flowLoan, "Create test lender", func() error {
		body := CreateLenderRequest{
			Name:  prefix + " Lender",
			Email: strPtr(prefix + "@itest.local"),
		}
		var resp LenderResponse
		if _, err := admin.PostInto("/api/v1/lenders", body, &resp); err != nil {
			return fmt.Errorf("create lender: %w", err)
		}
		testLenderID = resp.ID
		fmt.Printf("    Created lender ID %d\n", resp.ID)
		return AssertGreaterThan("id", uint(0), resp.ID)
	})

	defer func() {
		if testLoanID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/loans/%d", testLoanID))
		}
		if testLenderID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/lenders/%d", testLenderID))
		}
	}()

	reporter.RunTest(flowLoan, "List lenders", func() error {
		var list interface{}
		if _, err := admin.GetInto("/api/v1/lenders", &list); err != nil {
			return fmt.Errorf("list lenders: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowLoan, "Get lender by ID", func() error {
		if testLenderID == 0 {
			return fmt.Errorf("no test lender ID")
		}
		var resp LenderResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/lenders/%d", testLenderID), &resp); err != nil {
			return fmt.Errorf("get lender: %w", err)
		}
		return AssertEqual("id", testLenderID, resp.ID)
	})

	reporter.RunTest(flowLoan, "Update lender name", func() error {
		if testLenderID == 0 {
			return fmt.Errorf("no test lender ID")
		}
		newName := prefix + " Updated Lender"
		body := UpdateLenderRequest{Name: &newName}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/lenders/%d", testLenderID), body); err != nil {
			return fmt.Errorf("update lender: %w", err)
		}
		var resp LenderResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/lenders/%d", testLenderID), &resp); err != nil {
			return fmt.Errorf("get after update: %w", err)
		}
		return AssertEqual("name", newName, resp.Name)
	})

	// --- Loan CRUD ---

	reporter.RunTest(flowLoan, "Create loan with lender", func() error {
		if testLenderID == 0 {
			return fmt.Errorf("no test lender ID")
		}
		body := CreateLoanRequest{
			LenderID:        testLenderID,
			PrincipalAmount: 5000000,
			StartDate:       today(),
			Schedules: []RepaymentScheduleRequest{
				{DueDate: "2026-06-10", Amount: 1000000},
				{DueDate: "2026-07-10", Amount: 1000000},
				{DueDate: "2026-08-10", Amount: 5000000},
			},
		}
		var resp LoanResponse
		if _, err := admin.PostInto("/api/v1/loans", body, &resp); err != nil {
			return fmt.Errorf("create loan: %w", err)
		}
		testLoanID = resp.ID
		fmt.Printf("    Created loan ID %d, status: %s\n", resp.ID, resp.Status)
		if err := AssertGreaterThan("id", uint(0), resp.ID); err != nil {
			return err
		}
		return AssertGreaterThan("id", uint(0), resp.ID)
	})

	reporter.RunTest(flowLoan, "List loans", func() error {
		var list interface{}
		if _, err := admin.GetInto("/api/v1/loans", &list); err != nil {
			return fmt.Errorf("list loans: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowLoan, "Get loan by ID", func() error {
		if testLoanID == 0 {
			return fmt.Errorf("no test loan ID")
		}
		var resp LoanResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/loans/%d", testLoanID), &resp); err != nil {
			return fmt.Errorf("get loan: %w", err)
		}
		return AssertEqual("id", testLoanID, resp.ID)
	})

	reporter.RunTest(flowLoan, "Get loan schedule", func() error {
		if testLoanID == 0 {
			return fmt.Errorf("no test loan ID")
		}
		var schedule interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/loans/%d/schedule", testLoanID), &schedule); err != nil {
			return fmt.Errorf("get schedule: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowLoan, "Disburse loan", func() error {
		if testLoanID == 0 {
			return fmt.Errorf("no test loan ID")
		}
		body := DisburseLoanRequest{
			DisbursementDate: today(),
		}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/loans/%d/disburse", testLoanID), body); err != nil {
			return fmt.Errorf("disburse loan: %w", err)
		}
		var resp LoanResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/loans/%d", testLoanID), &resp); err != nil {
			return fmt.Errorf("get after disburse: %w", err)
		}
		fmt.Printf("    Loan status after disburse: %s\n", resp.Status)
		return AssertNotEqual("status", "pending", resp.Status)
	})

	reporter.RunTest(flowLoan, "Pay interest-only schedule without reducing principal", func() error {
		if testLoanID == 0 {
			return fmt.Errorf("no test loan ID")
		}
		var detail LoanDetailResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/loans/%d", testLoanID), &detail); err != nil {
			return fmt.Errorf("get loan detail: %w", err)
		}
		if len(detail.Schedules) == 0 {
			return fmt.Errorf("loan has no repayment schedules")
		}

		body := ProcessScheduledPaymentRequest{
			ScheduleID:  detail.Schedules[0].ID,
			PaymentDate: today(),
		}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/loans/%d/repay-schedule", testLoanID), body); err != nil {
			return fmt.Errorf("pay interest-only schedule: %w", err)
		}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/loans/%d", testLoanID), &detail); err != nil {
			return fmt.Errorf("get loan after interest payment: %w", err)
		}
		if err := AssertEqual("outstanding principal", int64(5000000), detail.OutstandingPrincipal); err != nil {
			return err
		}
		return AssertEqual("total interest paid", int64(1000000), detail.TotalInterestPaid)
	})

	reporter.RunTest(flowLoan, "Repay loan partial", func() error {
		if testLoanID == 0 {
			return fmt.Errorf("no test loan ID")
		}
		body := RepayLoanRequest{
			Amount:      1000000,
			PaymentDate: today(),
		}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/loans/%d/repay", testLoanID), body); err != nil {
			return fmt.Errorf("repay loan: %w", err)
		}
		fmt.Printf("    Partial repayment of 1000000 made\n")
		return nil
	})

	reporter.RunTest(flowLoan, "Update loan description", func() error {
		if testLoanID == 0 {
			return fmt.Errorf("no test loan ID")
		}
		desc := "ITest loan description"
		body := map[string]any{"description": desc}
		if _, _, err := admin.Patch(fmt.Sprintf("/api/v1/loans/%d", testLoanID), body); err != nil {
			return fmt.Errorf("update loan: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowLoan, "Delete test loan", func() error {
		if testLoanID == 0 {
			return fmt.Errorf("no test loan ID")
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/loans/%d", testLoanID)); err != nil {
			return fmt.Errorf("delete loan: %w", err)
		}
		testLoanID = 0
		fmt.Printf("    Test loan deleted\n")
		return nil
	})

	reporter.RunTest(flowLoan, "Delete test lender", func() error {
		if testLenderID == 0 {
			return fmt.Errorf("no test lender ID")
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/lenders/%d", testLenderID)); err != nil {
			return fmt.Errorf("delete lender: %w", err)
		}
		testLenderID = 0
		fmt.Printf("    Test lender deleted\n")
		return nil
	})

	// --- Edge cases ---

	reporter.RunTest(flowLoan, "Edge: create loan with zero principal", func() error {
		if testLenderID != 0 {
			// Lender already deleted, skip
			return nil
		}
		body := CreateLoanRequest{
			LenderID:        999999,
			PrincipalAmount: 0,
			StartDate:       today(),
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/loans", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})
}

func strPtr(s string) *string { return &s }
