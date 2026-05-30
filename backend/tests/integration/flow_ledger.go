package main

import (
	"fmt"
)

const flowLedger = "Ledger"

func runLedgerTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Ledger")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()

	var testEntryID uint
	var bulkEntryIDs []uint

	defer func() {
		for _, id := range bulkEntryIDs {
			body := map[string]any{"reason": "itest cleanup"}
			_, _, _ = admin.Post(fmt.Sprintf("/api/v1/ledger/entries/%d/reverse", id), body)
		}
	}()

	reporter.RunTest(flowLedger, "Get accounts metadata", func() error {
		var resp any
		if _, err := admin.GetInto("/api/v1/ledger/accounts/metadata", &resp); err != nil {
			return fmt.Errorf("accounts metadata: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowLedger, "Create single ledger entry", func() error {
		body := []CreateLedgerEntryRequest{
			{Account: "cash", Party: prefix + "_party", Debit: 1000000, Credit: 0, Date: today()},
		}
		var resp BulkLedgerEntriesResponse
		if _, err := admin.PostInto("/api/v1/ledger/entries", body, &resp); err != nil {
			return fmt.Errorf("create entry: %w", err)
		}
		if len(resp.Entries) == 0 {
			return fmt.Errorf("no entries returned")
		}
		testEntryID = resp.Entries[0].ID
		fmt.Printf("    Created ledger entry ID %d\n", testEntryID)
		return AssertGreaterThan("id", uint(0), testEntryID)
	})

	reporter.RunTest(flowLedger, "Get entry by ID", func() error {
		if testEntryID == 0 {
			return fmt.Errorf("no test entry ID")
		}
		var resp LedgerEntryResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/ledger/entries/%d", testEntryID), &resp); err != nil {
			return fmt.Errorf("get entry: %w", err)
		}
		return AssertEqual("id", testEntryID, resp.ID)
	})

	reporter.RunTest(flowLedger, "Create bulk entries", func() error {
		body := []CreateLedgerEntryRequest{
			{Account: "cash", Party: prefix + "_bulk_1", Debit: 500000, Credit: 0, Date: today()},
			{Account: "payable", Party: prefix + "_bulk_2", Debit: 0, Credit: 500000, Date: today()},
		}
		var resp BulkLedgerEntriesResponse
		if _, err := admin.PostInto("/api/v1/ledger/entries", body, &resp); err != nil {
			return fmt.Errorf("bulk create: %w", err)
		}
		for _, e := range resp.Entries {
			bulkEntryIDs = append(bulkEntryIDs, e.ID)
		}
		fmt.Printf("    Bulk created %d entries\n", resp.TotalEntries)
		return AssertEqual("total_entries", 2, resp.TotalEntries)
	})

	reporter.RunTest(flowLedger, "List entries with filters", func() error {
		var list any
		if _, err := admin.GetInto("/api/v1/ledger/entries?pageSize=10", &list); err != nil {
			return fmt.Errorf("list entries: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowLedger, "Get cash flow summary", func() error {
		var resp CashFlowSummaryResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/ledger/cash-flow?fromDate=%s&toDate=%s", weekAgo(), today()), &resp); err != nil {
			return fmt.Errorf("cash flow: %w", err)
		}
		fmt.Printf("    Net cash flow: %d\n", resp.NetCashFlow)
		return nil
	})

	reporter.RunTest(flowLedger, "Get ledger summary", func() error {
		var resp LedgerSummaryResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/ledger/summary?from=%s&to=%s", weekAgo(), today()), &resp); err != nil {
			return fmt.Errorf("ledger summary: %w", err)
		}
		fmt.Printf("    Net cashflow: %d\n", resp.Totals.NetCashflow)
		return nil
	})

	reporter.RunTest(flowLedger, "Reverse entry", func() error {
		if testEntryID == 0 {
			return fmt.Errorf("no test entry ID")
		}
		body := map[string]any{"reason": "itest reversal"}
		if _, _, err := admin.Post(fmt.Sprintf("/api/v1/ledger/entries/%d/reverse", testEntryID), body); err != nil {
			return fmt.Errorf("reverse entry: %w", err)
		}
		fmt.Printf("    Entry %d reversed\n", testEntryID)
		return nil
	})

	reporter.RunTest(flowLedger, "Export entries (binary)", func() error {
		data, _, statusCode, err := admin.DownloadGet(fmt.Sprintf("/api/v1/ledger/export?from=%s&to=%s", weekAgo(), today()))
		if err != nil {
			return fmt.Errorf("export: %w", err)
		}
		if err := AssertGreaterOrEqual("status", 200, statusCode); err != nil {
			return err
		}
		return AssertGreaterThan("file_size", 0, len(data))
	})

	reporter.RunTest(flowLedger, "Edge: create entry with zero debit and credit", func() error {
		body := []CreateLedgerEntryRequest{
			{Account: "cash", Party: "zero_test", Debit: 0, Credit: 0, Date: today()},
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/ledger/entries", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})
}
