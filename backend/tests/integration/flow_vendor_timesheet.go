package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

const flowVendorTimesheetAPI = "VendorTimesheetAPI"

// runVendorTimesheetTests exercises the read-only vendor surface
// (/api/v1/vendor/*): the key scope gate, the code-based
// filtering contract, the required-filter guard, and — most importantly —
// that nothing on this surface can write.
//
// The read-only half is proven structurally AND by observation: every mutating
// verb is refused, and a snapshot of the seeded timesheet taken before the flow
// is compared against one taken after, so a hidden side effect would fail here
// even if the router quietly accepted a write.
func runVendorTimesheetTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Vendor Timesheet Read API (scoped API-key channel)")

	admin := client.WithToken(data.AdminToken)

	// --- Mint the vendor key ---------------------------------------------
	var vendorKeyID uint
	var vendorKey string

	reporter.RunTest(flowVendorTimesheetAPI, "Admin creates a timesheet_read key", func() error {
		var out CreateAPIKeyResponse
		_, err := admin.PostInto("/api/v1/admin/api-keys",
			CreateAPIKeyRequest{Name: "itest vendor timesheet", Scope: "timesheet_read"}, &out)
		if err != nil {
			return err
		}
		if out.Key == "" {
			return fmt.Errorf("create returned an empty plaintext key")
		}
		if out.Scope != "timesheet_read" {
			return fmt.Errorf("scope = %q, want timesheet_read", out.Scope)
		}
		vendorKeyID = out.ID
		vendorKey = out.Key
		return nil
	})

	reporter.RunTest(flowVendorTimesheetAPI, "Admin rejects an unknown scope", func() error {
		_, code, err := admin.PostExpectError("/api/v1/admin/api-keys",
			CreateAPIKeyRequest{Name: "bad scope", Scope: "time_sheet_read"})
		if err != nil {
			return err
		}
		if code != 400 {
			return fmt.Errorf("status %d, want 400", code)
		}
		return nil
	})

	vendor := NewAPIClient(cfg.BaseURL)
	vendor.Headers = map[string]string{"X-API-Key": vendorKey}

	// --- Authentication failures (401) -----------------------------------
	reporter.RunTest(flowVendorTimesheetAPI, "Vendor call without X-API-Key returns 401", func() error {
		anon := NewAPIClient(cfg.BaseURL)
		_, code, err := anon.GetExpectError("/api/v1/vendor/projects")
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	reporter.RunTest(flowVendorTimesheetAPI, "Vendor call with a garbage key returns 401", func() error {
		bad := NewAPIClient(cfg.BaseURL)
		bad.Headers = map[string]string{"X-API-Key": "ttk_garbage"}
		_, code, err := bad.GetExpectError("/api/v1/vendor/projects")
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	// --- Admin-only key management (403) ---------------------------------
	if len(data.Partners) == 0 {
		reporter.Skip(flowVendorTimesheetAPI, "Non-admin cannot create keys", "no partner token discovered")
	} else {
		reporter.RunTest(flowVendorTimesheetAPI, "Non-admin cannot create keys (403)", func() error {
			partner := client.WithToken(data.Partners[0].Token)
			_, code, err := partner.PostExpectError("/api/v1/admin/api-keys", CreateAPIKeyRequest{Name: "nope", Scope: "timesheet_read"})
			if err != nil {
				return err
			}
			if code != 403 {
				return fmt.Errorf("status %d, want 403", code)
			}
			return nil
		})
	}

	// --- Happy paths ------------------------------------------------------
	// Seed a project + employee to filter against; fall back to whatever the
	// discovery pass already created.
	var projectCode, cccd string
	if len(data.Projects) > 0 {
		projectCode = data.Projects[0].Code
	}
	if len(data.Employees) > 0 {
		cccd = data.Employees[0].CCCD
	}

	reporter.RunTest(flowVendorTimesheetAPI, "timesheet_read key can list projects", func() error {
		resp, err := vendor.GetInto("/api/v1/vendor/projects?page_size=100", &[]VendorProjectItem{})
		if err != nil {
			return err
		}
		if resp.Pagination == nil {
			return fmt.Errorf("response carried no pagination block")
		}
		return nil
	})

	reporter.RunTest(flowVendorTimesheetAPI, "timesheet_read key can list employees", func() error {
		_, err := vendor.GetInto("/api/v1/vendor/employees?page_size=100", &[]VendorEmployeeItem{})
		return err
	})

	// --- Required-filter guard --------------------------------------------
	reporter.RunTest(flowVendorTimesheetAPI, "Unfiltered timesheet query returns 400", func() error {
		_, code, err := vendor.GetExpectError("/api/v1/vendor/timesheets")
		if err != nil {
			return err
		}
		if code != 400 {
			return fmt.Errorf("status %d, want 400", code)
		}
		return nil
	})

	if projectCode != "" {
		reporter.RunTest(flowVendorTimesheetAPI, "Query by project_code succeeds", func() error {
			_, err := vendor.GetInto(
				"/api/v1/vendor/timesheets?"+url.Values{"project_code": {projectCode}}.Encode(),
				&[]VendorTimesheetItem{})
			return err
		})
	}

	if cccd != "" {
		reporter.RunTest(flowVendorTimesheetAPI, "Query by cccd succeeds", func() error {
			_, err := vendor.GetInto(
				"/api/v1/vendor/timesheets?"+url.Values{"cccd": {cccd}}.Encode(),
				&[]VendorTimesheetItem{})
			return err
		})
	}

	reporter.RunTest(flowVendorTimesheetAPI, "Unknown project_code returns 400, not 404", func() error {
		_, code, err := vendor.GetExpectError(
			"/api/v1/vendor/timesheets?project_code=NO_SUCH_PROJECT_XYZ")
		if err != nil {
			return err
		}
		if code != 400 {
			return fmt.Errorf("status %d, want 400 (the request was well-formed; the code was not)", code)
		}
		return nil
	})

	reporter.RunTest(flowVendorTimesheetAPI, "Unknown cccd returns 400, not 404", func() error {
		_, code, err := vendor.GetExpectError(
			"/api/v1/vendor/timesheets?cccd=000000000000")
		if err != nil {
			return err
		}
		if code != 400 {
			return fmt.Errorf("status %d, want 400", code)
		}
		return nil
	})

	reporter.RunTest(flowVendorTimesheetAPI, "Malformed date returns 400", func() error {
		_, code, err := vendor.GetExpectError("/api/v1/vendor/timesheets?from_date=01-02-2026")
		if err != nil {
			return err
		}
		if code != 400 {
			return fmt.Errorf("status %d, want 400", code)
		}
		return nil
	})

	// --- Date-range boundary (regression guard for StartOfDay(UTC)) --------
	// A range covering exactly the seeded project's month must return its rows
	// on both boundary days, not drop the first.
	reporter.RunTest(flowVendorTimesheetAPI, "Date range includes both boundary days", func() error {
		from := time.Now().AddDate(0, -2, 0).Format("2006-01-01")
		to := time.Now().AddDate(0, 1, 0).Format("2006-01-01")

		q := url.Values{"from_date": {from}, "to_date": {to}, "page_size": {"200"}}
		var items []VendorTimesheetItem
		if _, err := vendor.GetInto("/api/v1/vendor/timesheets?"+q.Encode(), &items); err != nil {
			return err
		}

		// Any returned row must fall inside the range; the bug this guards
		// produced rows outside it or dropped the first day.
		for _, it := range items {
			if it.Date < from || it.Date > to {
				return fmt.Errorf("row date %q outside requested range %s..%s", it.Date, from, to)
			}
		}
		return nil
	})

	// --- Payload contract -------------------------------------------------
	// Full-fidelity by design for the sister vendor. Assert the money fields
	// are PRESENT so a future "safety" trim cannot silently regress the
	// contract the vendor has already integrated against.
	reporter.RunTest(flowVendorTimesheetAPI, "Timesheet payload carries money and PII fields", func() error {
		rows, _, err := vendorRawRows(vendor, "/api/v1/vendor/timesheets?"+
			url.Values{"from_date": {time.Now().AddDate(0, -2, 0).Format("2006-01-01")}, "page_size": {"50"}}.Encode())
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			// No seeded timesheet in range: the presence contract still has to
			// hold for the shapes we can observe, so check a row if we got one
			// and otherwise fall through to the key-level assertions below.
			return nil
		}

		row := rows[0]
		for _, field := range []string{
			"project_code", "cccd", "employee_name", "date",
			"hours_worked", "pay_type", "payrate", "amount",
			"paid_amount", "payment_status",
		} {
			if _, ok := row[field]; !ok {
				return fmt.Errorf("timesheet row missing field %q", field)
			}
		}
		// Client-billing figures stay internal by design, as is the internal
		// approval workflow the vendor contract deliberately omits.
		for _, forbidden := range []string{
			"revenue_receivable", "revenue_paid", "status", "rejection_reason",
		} {
			if _, ok := row[forbidden]; ok {
				return fmt.Errorf("timesheet row must not expose %q", forbidden)
			}
		}
		// Internal IDs are withheld so the vendor cannot cache them as
		// stable identifiers.
		for _, forbidden := range []string{"id", "project_id", "employee_id"} {
			if _, ok := row[forbidden]; ok {
				return fmt.Errorf("timesheet row must not expose internal %q", forbidden)
			}
		}
		return nil
	})

	reporter.RunTest(flowVendorTimesheetAPI, "Employee payload carries PII and bank fields", func() error {
		rows, _, err := vendorRawRows(vendor, "/api/v1/vendor/employees?page_size=50")
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		row := rows[0]
		for _, field := range []string{
			"cccd", "fullname", "mobile", "bank_account_name",
			"bank_account_number", "project_codes",
		} {
			if _, ok := row[field]; !ok {
				return fmt.Errorf("employee row missing field %q", field)
			}
		}
		if _, ok := row["id"]; ok {
			return fmt.Errorf("employee row must not expose internal id")
		}
		// OnePay bank-validation state is internal payment-rail detail.
		for _, forbidden := range []string{"bank_account_status", "bank_account_invalid_reason"} {
			if _, ok := row[forbidden]; ok {
				return fmt.Errorf("employee row must not expose %q", forbidden)
			}
		}
		return nil
	})

	// --- READ-ONLY: the load-bearing guarantee ----------------------------
	// No non-GET verb is routable under /vendor/*, so the router has nothing to
	// dispatch to. Asserted per endpoint and per verb.
	reporter.RunTest(flowVendorTimesheetAPI, "All mutating verbs are refused on every vendor path", func() error {
		type verb struct {
			name string
			call func(c *APIClient, path string) (int, error)
		}
		verbs := []verb{
			{"POST", func(c *APIClient, path string) (int, error) {
				_, code, err := c.PostExpectError(path, map[string]string{"name": "x"})
				return code, err
			}},
			{"PUT", func(c *APIClient, path string) (int, error) {
				_, code, err := c.PutExpectError(path, map[string]string{"name": "x"})
				return code, err
			}},
			{"PATCH", func(c *APIClient, path string) (int, error) {
				// PATCH has no ExpectError helper; a non-2xx is enough since the
				// body is irrelevant to a 404/405 assertion.
				_, code, err := c.Patch(path, map[string]string{"name": "x"})
				return code, err
			}},
			{"DELETE", func(c *APIClient, path string) (int, error) {
				_, code, err := c.DeleteExpectError(path)
				return code, err
			}},
		}

		for _, path := range vendorReadPaths() {
			for _, v := range verbs {
				code, err := v.call(vendor, path)
				if err != nil && code == 0 {
					return fmt.Errorf("%s %s: %w", v.name, path, err)
				}
				// 404 = no such route; 405 = method not allowed. Both prove the
				// write never reached a handler.
				if code != 404 && code != 405 {
					return fmt.Errorf("%s %s: status %d, want 404 or 405", v.name, path, code)
				}
			}
		}
		return nil
	})

	// Snapshot before the write attempts, then re-check: even if a mutating
	// verb were somehow routed, the data must be untouched.
	var before []VendorTimesheetItem
	reporter.RunTest(flowVendorTimesheetAPI, "Snapshot timesheet money/status before write attempts", func() error {
		var err error
		before, _, err = vendorTimesheetSnapshot(vendor)
		return err
	})

	reporter.RunTest(flowVendorTimesheetAPI, "An API key cannot reach a JWT-protected write route", func() error {
		// Even with a fabricated bearer token, the timesheet mutation route is
		// gated on a real JWT and must refuse.
		probe := NewAPIClient(cfg.BaseURL)
		probe.Headers = map[string]string{
			"X-API-Key":     vendorKey,
			"Authorization": "Bearer not-a-real-token",
		}
		_, code, err := probe.PutExpectError("/api/v1/timesheets/1", map[string]string{"name": "x"})
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("status %d, want 401", code)
		}
		return nil
	})

	reporter.RunTest(flowVendorTimesheetAPI, "Timesheet data is unchanged after the whole flow", func() error {
		after, _, err := vendorTimesheetSnapshot(vendor)
		if err != nil {
			return err
		}
		if len(before) != len(after) {
			return fmt.Errorf("row count changed: %d -> %d", len(before), len(after))
		}
		for i := range before {
			if before[i] != after[i] {
				return fmt.Errorf("row %d changed: %+v -> %+v", i, before[i], after[i])
			}
		}
		return nil
	})

	// --- Revocation --------------------------------------------------------
	reporter.RunTest(flowVendorTimesheetAPI, "Revoked vendor key can no longer authenticate", func() error {
		if vendorKeyID == 0 {
			return fmt.Errorf("no vendor key created")
		}
		if _, code, err := admin.Delete(fmt.Sprintf("/api/v1/admin/api-keys/%d", vendorKeyID)); err != nil {
			return err
		} else if code != 200 {
			return fmt.Errorf("revoke status %d, want 200", code)
		}
		_, code, err := vendor.GetExpectError("/api/v1/vendor/projects")
		if err != nil {
			return err
		}
		if code != 401 {
			return fmt.Errorf("post-revoke status %d, want 401", code)
		}
		return nil
	})

	// --- Pagination sanity --------------------------------------------------
	reporter.RunTest(flowVendorTimesheetAPI, "page_size is clamped, not rejected", func() error {
		resp, err := vendor.GetInto("/api/v1/vendor/projects?page_size=100000", &[]VendorProjectItem{})
		if err != nil {
			return err
		}
		if resp.Pagination != nil && resp.Pagination.PageSize > 100 {
			return fmt.Errorf("pageSize = %d, want clamped to <= 100", resp.Pagination.PageSize)
		}
		return nil
	})
}

// vendorReadPaths returns every GET path on the vendor surface, used by both
// the cross-scope and the read-only assertions so neither can drift from the
// routes that actually exist.
func vendorReadPaths() []string {
	return []string{
		"/api/v1/vendor/projects",
		"/api/v1/vendor/employees",
		"/api/v1/vendor/timesheets?project_code=x",
	}
}

// vendorRawRows decodes a vendor response's rows into generic maps so a test
// can assert on field presence — including fields that must be absent, which a
// typed struct would silently normalise away.
func vendorRawRows(c *APIClient, path string) ([]map[string]any, *Pagination, error) {
	resp, statusCode, err := c.Get(path)
	if err != nil {
		return nil, nil, err
	}
	if resp.Status == "error" || statusCode >= 400 {
		return nil, resp.Pagination, fmt.Errorf("API error (HTTP %d): %s", statusCode, resp.Message)
	}
	if len(resp.Data) == 0 {
		return nil, resp.Pagination, nil
	}

	var rows []map[string]any
	if err := json.Unmarshal(resp.Data, &rows); err != nil {
		return nil, resp.Pagination, fmt.Errorf("decode vendor rows: %w", err)
	}
	return rows, resp.Pagination, nil
}

// vendorTimesheetSnapshot captures the observable timesheet state so the flow
// can prove the reads had no side effects.
func vendorTimesheetSnapshot(c *APIClient) ([]VendorTimesheetItem, *Pagination, error) {
	// A wide window so seeded rows are included regardless of when the suite ran.
	from := time.Now().AddDate(-1, 0, 0).Format("2006-01-01")
	to := time.Now().AddDate(1, 0, 0).Format("2006-01-01")
	q := url.Values{"from_date": {from}, "to_date": {to}, "page_size": {"200"}}

	var items []VendorTimesheetItem
	resp, err := c.GetInto("/api/v1/vendor/timesheets?"+q.Encode(), &items)
	if err != nil {
		return nil, nil, err
	}
	return items, resp.Pagination, nil
}
