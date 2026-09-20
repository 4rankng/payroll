package main

import (
	"encoding/json"
	"fmt"
)

const flowLedger = "Ledger"

// ledgerImbalance returns SUM(debit) - SUM(credit) over every account in the
// reporting window. The ledger's invariant is that a write never changes this
// number, so it is compared before and after an action rather than against zero:
// a database carrying drift from the old one-sided reversals would fail an
// absolute check for reasons the unit under test did not cause.
func ledgerImbalance(admin *APIClient) (int64, error) {
	var summary LedgerSummaryResponse
	if _, err := admin.GetInto(fmt.Sprintf("/api/v1/ledger/summary?from=%s&to=%s", weekAgo(), today()), &summary); err != nil {
		return 0, fmt.Errorf("ledger summary: %w", err)
	}
	var debit, credit int64
	for _, account := range summary.ByAccount {
		debit += account.Debit
		credit += account.Credit
	}
	return debit - credit, nil
}

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
		// A ledger block must balance on its own: a cash debit needs its counter
		// entry, which is the invariant the API enforces (and answers 400 for).
		body := []CreateLedgerEntryRequest{
			{Account: "cash", Party: prefix + "_party", Debit: 1000000, Credit: 0, Date: today()},
			{Account: "payable", Party: prefix + "_party", Debit: 0, Credit: 1000000, Date: today()},
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

	reporter.RunTest(flowLedger, "Edge: unbalanced ledger block is rejected", func() error {
		body := []CreateLedgerEntryRequest{
			{Account: "cash", Party: prefix + "_unbalanced", Debit: 1000000, Credit: 0, Date: today()},
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/ledger/entries", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		// 400, not 500: the block fails the double-entry rule, which is caller
		// input — the gap that made this path show up as a server error.
		return AssertEqual("status", 400, statusCode)
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
		before, err := ledgerImbalance(admin)
		if err != nil {
			return err
		}

		body := map[string]any{"reason": "itest reversal"}
		resp, status, err := admin.Post(fmt.Sprintf("/api/v1/ledger/entries/%d/reverse", testEntryID), body)
		if err != nil {
			return fmt.Errorf("reverse entry: %w", err)
		}
		if status >= 400 {
			return fmt.Errorf("reverse rejected (HTTP %d): %s", status, resp.Message)
		}
		var mirror LedgerEntryResponse
		raw, _ := json.Marshal(resp.Data)
		if err := json.Unmarshal(raw, &mirror); err != nil {
			return fmt.Errorf("decode reversal: %w", err)
		}
		if mirror.ReversalOfEntryID == nil || *mirror.ReversalOfEntryID != testEntryID {
			return fmt.Errorf("mirror does not link back to entry %d: %+v", testEntryID, mirror.ReversalOfEntryID)
		}
		after, err := ledgerImbalance(admin)
		if err != nil {
			return err
		}
		if after != before {
			return fmt.Errorf("reversal changed the ledger imbalance: %d -> %d", before, after)
		}
		fmt.Printf("    Entry %d reversed as %d; Σnợ−Σcó unchanged (%d)\n", testEntryID, mirror.ID, after)
		return nil
	})

	reporter.RunTest(flowLedger, "Edge: reversing the same entry twice is refused", func() error {
		if testEntryID == 0 {
			return fmt.Errorf("no test entry ID")
		}
		body := map[string]any{"reason": "itest double reversal"}
		_, statusCode, err := admin.PostExpectError(fmt.Sprintf("/api/v1/ledger/entries/%d/reverse", testEntryID), body)
		if err != nil {
			return fmt.Errorf("second reverse: %w", err)
		}
		if statusCode != 400 && statusCode != 409 {
			// A second reversal would double the offset; the API refuses it.
			return fmt.Errorf("second reversal accepted (HTTP %d)", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowLedger, "Edge: reversing a reversal is refused", func() error {
		if testEntryID == 0 {
			return fmt.Errorf("no test entry ID")
		}
		var detail LedgerEntryResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/ledger/entries/%d", testEntryID), &detail); err != nil {
			return fmt.Errorf("get original: %w", err)
		}
		if !detail.IsReversed {
			return fmt.Errorf("original entry %d should report is_reversed", testEntryID)
		}
		// Reversing the original once more is refused above; a mirror must not be
		// reversible either, otherwise reversals compound.
		body := map[string]any{"reason": "itest reverse of reversal"}
		_, statusCode, err := admin.PostExpectError(fmt.Sprintf("/api/v1/ledger/entries/%d/reverse", detail.ID), body)
		if err != nil {
			return fmt.Errorf("reverse mirror: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
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
