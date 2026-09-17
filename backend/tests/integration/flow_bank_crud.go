package main

import (
	"fmt"
)

const flowBank = "BankCRUD"

func runBankCRUDTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Bank CRUD")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()

	var testBankID uint

	reporter.RunTest(flowBank, "Create test bank", func() error {
		body := CreateBankRequest{
			BranchName: prefix + " Bank Branch",
		}
		var resp BankResponse
		if _, err := admin.PostInto("/api/v1/banks", body, &resp); err != nil {
			return fmt.Errorf("create bank: %w", err)
		}
		testBankID = resp.ID
		fmt.Printf("    Created bank ID %d\n", resp.ID)
		if err := AssertGreaterThan("id", uint(0), resp.ID); err != nil {
			return err
		}
		return AssertEqual("branch_name", prefix+" Bank Branch", resp.BranchName)
	})

	defer func() {
		if testBankID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/banks/%d", testBankID))
		}
	}()

	reporter.RunTest(flowBank, "List banks", func() error {
		var list []BankResponse
		if _, err := admin.GetInto("/api/v1/banks?pageSize=10", &list); err != nil {
			return fmt.Errorf("list banks: %w", err)
		}
		fmt.Printf("    Found %d banks\n", len(list))
		return AssertSliceMinLen("banks", len(list), 1)
	})

	reporter.RunTest(flowBank, "Get bank by ID", func() error {
		if testBankID == 0 {
			return fmt.Errorf("no test bank ID")
		}
		var resp BankResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/banks/%d", testBankID), &resp); err != nil {
			return fmt.Errorf("get bank: %w", err)
		}
		if err := AssertEqual("id", testBankID, resp.ID); err != nil {
			return err
		}
		return AssertHasPrefix("branch_name", resp.BranchName, prefix)
	})

	reporter.RunTest(flowBank, "Update bank branch name", func() error {
		if testBankID == 0 {
			return fmt.Errorf("no test bank ID")
		}
		newName := prefix + " Updated Branch"
		body := UpdateBankRequest{BranchName: &newName}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/banks/%d", testBankID), body); err != nil {
			return fmt.Errorf("update bank: %w", err)
		}
		var resp BankResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/banks/%d", testBankID), &resp); err != nil {
			return fmt.Errorf("get after update: %w", err)
		}
		return AssertEqual("branch_name", newName, resp.BranchName)
	})

	reporter.RunTest(flowBank, "Delete test bank", func() error {
		if testBankID == 0 {
			return fmt.Errorf("no test bank ID")
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/banks/%d", testBankID)); err != nil {
			return fmt.Errorf("delete bank: %w", err)
		}
		testBankID = 0
		fmt.Printf("    Test bank deleted\n")
		return nil
	})

	// --- Edge cases ---

	reporter.RunTest(flowBank, "Edge: create bank with missing branch_name", func() error {
		body := map[string]any{}
		_, statusCode, err := admin.PostExpectError("/api/v1/banks", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowBank, "Edge: update non-existent bank", func() error {
		newName := "Ghost"
		body := UpdateBankRequest{BranchName: &newName}
		_, statusCode, _ := admin.Put(fmt.Sprintf("/api/v1/banks/%d", nonexistentID), body)
		if statusCode < 400 {
			return fmt.Errorf("expected error updating non-existent bank, got HTTP %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowBank, "Edge: delete non-existent bank", func() error {
		_, statusCode, _ := admin.Delete(fmt.Sprintf("/api/v1/banks/%d", nonexistentID))
		if statusCode < 400 {
			return fmt.Errorf("expected error deleting non-existent bank, got HTTP %d", statusCode)
		}
		return nil
	})
}
