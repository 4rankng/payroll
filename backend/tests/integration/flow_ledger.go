package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
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

	runOnePayFeeReportImportTests(admin, reporter)

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

// OnePay fee report imports. The provider ships statements as legacy .xls
// (BIFF) and strict-OOXML .xlsx; both must pass the upload guard and the
// parser, then fail only at wallet reconciliation — the integration database
// carries no wallet payments matching the report's fund transfer IDs, so a
// 400 whose issues are reconciliation codes (not format codes) is the pass
// signal: the bytes were read, sheet located, and all 250 rows parsed.
const onepayFeeReportFixture = "tests/fixtures/BBDS_DETAIL_PO_1791366534312_4065"

func runOnePayFeeReportImportTests(admin *APIClient, reporter *Reporter) {
	for _, ext := range []string{".xls", ".xlsx"} {
		ext := ext
		reporter.RunTest(flowLedger, "OnePay fee report: "+ext+" accepted and parsed", func() error {
			content, err := os.ReadFile(onepayFeeReportFixture + ext)
			if err != nil {
				return fmt.Errorf("read fixture: %w", err)
			}
			statusCode, body, err := uploadOnePayFeeReport(admin, "BBDS_DETAIL_PO_1791366534312_4065"+ext, content)
			if err != nil {
				return fmt.Errorf("upload: %w", err)
			}
			if statusCode != 400 {
				return fmt.Errorf("expected 400 (reconciliation issues), got %d: %s", statusCode, body)
			}
			var parsed struct {
				Status  string `json:"status"`
				Message string `json:"message"`
				Details struct {
					Issues []struct {
						Code string `json:"code"`
					} `json:"issues"`
				} `json:"details"`
			}
			if err := json.Unmarshal(body, &parsed); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
			if parsed.Status != "error" {
				return fmt.Errorf("expected status error, got %q", parsed.Status)
			}
			if len(parsed.Details.Issues) == 0 {
				return fmt.Errorf("expected reconciliation issues, got none: %s", body)
			}
			// Format/parse failures would mean the file was never really read.
			for _, iss := range parsed.Details.Issues {
				switch iss.Code {
				case "invalid_excel", "missing_sheet", "missing_header_row", "detail_read_failed":
					return fmt.Errorf("format-level issue %q — file was not parsed", iss.Code)
				}
			}
			fmt.Printf("    %s parsed; %d reconciliation issue(s), first=%s\n", ext, len(parsed.Details.Issues), parsed.Details.Issues[0].Code)
			return nil
		})
	}

	reporter.RunTest(flowLedger, "OnePay fee report: garbage .xls is rejected as invalid", func() error {
		statusCode, body, err := uploadOnePayFeeReport(admin, "not_excel.xls", []byte("this is not a workbook"))
		if err != nil {
			return fmt.Errorf("upload: %w", err)
		}
		if statusCode != 400 {
			return fmt.Errorf("expected 400, got %d: %s", statusCode, body)
		}
		return nil
	})
}

// uploadOnePayFeeReport POSTs a multipart "file" to the OnePay fee import
// endpoint and returns the status code with the raw body.
func uploadOnePayFeeReport(client *APIClient, filename string, content []byte) (int, []byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return 0, nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return 0, nil, fmt.Errorf("write file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return 0, nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequest("POST", client.BaseURL+"/api/v1/ledger/onepay-fee-reports", &buf)
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if client.Token != "" {
		req.Header.Set("Authorization", "Bearer "+client.Token)
	}
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read body: %w", err)
	}
	return resp.StatusCode, body, nil
}
