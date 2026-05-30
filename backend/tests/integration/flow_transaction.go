package main

import (
	"fmt"
)

const flowTransaction = "Transaction"

func runTransactionTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Transactions")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()

	var testTxnID uint

	reporter.RunTest(flowTransaction, "Get transaction metadata", func() error {
		var resp TransactionMetadataResponse
		if _, err := admin.GetInto("/api/v1/transactions/metadata", &resp); err != nil {
			return fmt.Errorf("metadata: %w", err)
		}
		fmt.Printf("    Transaction types: %d, Statuses: %d\n", len(resp.TransactionTypes), len(resp.Statuses))
		return AssertSliceMinLen("transaction_types", len(resp.TransactionTypes), 1)
	})

	reporter.RunTest(flowTransaction, "Create pending transaction", func() error {
		body := CreateTransactionRequest{
			Description:     prefix + " test transaction",
			TransactionType: "expense",
			Amount:          2000000,
			Party:           prefix + "_vendor",
			Status:          "pending",
		}
		var wrapper TransactionWithLedgerResponse
		if _, err := admin.PostInto("/api/v1/transactions", body, &wrapper); err != nil {
			return fmt.Errorf("create transaction: %w", err)
		}
		resp := wrapper.Transaction
		testTxnID = resp.ID
		fmt.Printf("    Created transaction ID %d, code: %s, status: %s\n", resp.ID, resp.TransactionCode, resp.Status)
		if err := AssertGreaterThan("id", uint(0), resp.ID); err != nil {
			return err
		}
		return AssertEqual("status", "pending", resp.Status)
	})

	defer func() {
		if testTxnID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/transactions/%d", testTxnID))
		}
	}()

	reporter.RunTest(flowTransaction, "List transactions", func() error {
		var list interface{}
		if _, err := admin.GetInto("/api/v1/transactions?pageSize=10", &list); err != nil {
			return fmt.Errorf("list transactions: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowTransaction, "Get transaction by ID", func() error {
		if testTxnID == 0 {
			return fmt.Errorf("no test transaction ID")
		}
		var resp TransactionResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/transactions/%d", testTxnID), &resp); err != nil {
			return fmt.Errorf("get transaction: %w", err)
		}
		return AssertEqual("id", testTxnID, resp.ID)
	})

	reporter.RunTest(flowTransaction, "Settle transaction partially", func() error {
		if testTxnID == 0 {
			return fmt.Errorf("no test transaction ID")
		}
		body := SettleTransactionRequest{
			Amount:         1000000,
			SettlementDate: today(),
		}
		if _, err := admin.PostInto(fmt.Sprintf("/api/v1/transactions/%d/settle", testTxnID), body, &map[string]interface{}{}); err != nil {
			return fmt.Errorf("partial settle: %w", err)
		}
		var resp TransactionResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/transactions/%d", testTxnID), &resp); err != nil {
			return fmt.Errorf("get after settle: %w", err)
		}
		fmt.Printf("    Status after partial settle: %s\n", resp.Status)
		return nil
	})

	reporter.RunTest(flowTransaction, "Settle transaction fully", func() error {
		if testTxnID == 0 {
			return fmt.Errorf("no test transaction ID")
		}
		body := SettleTransactionRequest{
			Amount:         1000000,
			SettlementDate: today(),
		}
		if _, err := admin.PostInto(fmt.Sprintf("/api/v1/transactions/%d/settle", testTxnID), body, &map[string]interface{}{}); err != nil {
			return fmt.Errorf("full settle: %w", err)
		}
		var resp TransactionResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/transactions/%d", testTxnID), &resp); err != nil {
			return fmt.Errorf("get after full settle: %w", err)
		}
		fmt.Printf("    Status after full settle: %s\n", resp.Status)
		return AssertEqual("status", "settled", resp.Status)
	})

	// Create another transaction for reversal
	var reversalTxnID uint

	reporter.RunTest(flowTransaction, "Create and reverse transaction", func() error {
		body := CreateTransactionRequest{
			Description:     prefix + " reversal test",
			TransactionType: "revenue",
			Amount:          500000,
			Party:           prefix + "_client",
			Status:          "settled",
		}
		var revWrapper TransactionWithLedgerResponse
		if _, err := admin.PostInto("/api/v1/transactions", body, &revWrapper); err != nil {
			return fmt.Errorf("create for reversal: %w", err)
		}
		resp := revWrapper.Transaction
		reversalTxnID = resp.ID

		revBody := ReverseTransactionRequest{Reason: "itest reversal"}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/transactions/%d/reverse", reversalTxnID), revBody); err != nil {
			return fmt.Errorf("reverse: %w", err)
		}
		fmt.Printf("    Transaction %d reversed\n", reversalTxnID)
		return nil
	})

	reporter.RunTest(flowTransaction, "Export transactions (binary)", func() error {
		data, _, statusCode, err := admin.DownloadGet("/api/v1/transactions/export")
		if err != nil {
			return fmt.Errorf("export: %w", err)
		}
		if err := AssertGreaterOrEqual("status", 200, statusCode); err != nil {
			return err
		}
		return AssertGreaterThan("file_size", 0, len(data))
	})

	reporter.RunTest(flowTransaction, "Edge: create transaction with zero amount", func() error {
		body := CreateTransactionRequest{
			Description:     "zero amount",
			TransactionType: "expense",
			Amount:          0,
			Party:           "nobody",
			Status:          "pending",
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/transactions", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})
}
