package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"api-server/internal/pkg/clock"
)

const flowPayrate = "PayrateCRUD"

func runPayrateCRUDTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Payrate CRUD")

	admin := client.WithToken(data.AdminToken)

	if data.WeeklyProject == nil {
		reporter.Skip(flowPayrate, "All tests", "no weekly project available")
		return
	}

	projectID := data.WeeklyProject.ID
	var testPayrateID uint
	adopted := false

	reporter.RunTest(flowPayrate, "List payrates for project", func() error {
		var resp interface{}
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/payrates?project_id=%d", projectID), &resp); err != nil {
			return fmt.Errorf("list payrates: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowPayrate, "Create payrate", func() error {
		body := CreatePayrateRequest{
			Rates:         []byte(`{"Nhân viên":{"Ngày thường":{"Ca ngày":250000}}}`),
			EffectiveFrom: today(),
		}
		var resp PayrateResponse
		if _, err := admin.PostInto(fmt.Sprintf("/api/v1/projects/%d/payrate", projectID), body, &resp); err == nil {
			testPayrateID = resp.ID
			fmt.Printf("    Created payrate ID %d for project %d\n", resp.ID, projectID)
			return AssertGreaterThan("id", uint(0), resp.ID)
		} else if existing, found := findTodayPayrate(admin, projectID); found {
			// A same-day payrate already exists. If it carries this test's exact
			// rates it is residue from an interrupted earlier run (or a concurrent
			// suite instance) — remove it and retry. Otherwise it is real business
			// data: adopt it for the update steps without claiming ownership.
			if isTestPayrateRates(existing.Rates) {
				if _, _, delErr := admin.Delete(fmt.Sprintf("/api/v1/payrates/%d", existing.ID)); delErr != nil {
					return fmt.Errorf("remove leftover test payrate %d: %v (create: %w)", existing.ID, delErr, err)
				}
				var retry PayrateResponse
				if _, retryErr := admin.PostInto(fmt.Sprintf("/api/v1/projects/%d/payrate", projectID), body, &retry); retryErr != nil {
					return fmt.Errorf("create payrate after cleanup: %w", retryErr)
				}
				testPayrateID = retry.ID
				fmt.Printf("    Created payrate ID %d for project %d (after residue cleanup)\n", retry.ID, projectID)
				return AssertGreaterThan("id", uint(0), retry.ID)
			}
			testPayrateID = existing.ID
			adopted = true
			fmt.Printf("    Adopted pre-existing payrate ID %d for project %d\n", existing.ID, projectID)
			return nil
		} else {
			return fmt.Errorf("create payrate: %w", err)
		}
	})

	defer func() {
		if testPayrateID != 0 && !adopted {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/payrates/%d", testPayrateID))
		}
	}()

	reporter.RunTest(flowPayrate, "Get payrate by ID", func() error {
		if testPayrateID == 0 {
			return fmt.Errorf("no test payrate ID")
		}
		var resp PayrateResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/payrates/%d", testPayrateID), &resp); err != nil {
			return fmt.Errorf("get payrate: %w", err)
		}
		return AssertEqual("id", testPayrateID, resp.ID)
	})

	reporter.RunTest(flowPayrate, "Update payrate", func() error {
		if testPayrateID == 0 {
			return fmt.Errorf("no test payrate ID")
		}
		body := UpdatePayrateRequest{
			Rates:         []byte(`{"Nhân viên":{"Ngày thường":{"Ca ngày":300000}}}`),
			EffectiveFrom: today(),
		}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/payrates/%d", testPayrateID), body); err != nil {
			return fmt.Errorf("update payrate: %w", err)
		}
		var resp PayrateResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/payrates/%d", testPayrateID), &resp); err != nil {
			return fmt.Errorf("get after update: %w", err)
		}
		return AssertEqual("id", testPayrateID, resp.ID)
	})

	reporter.RunTest(flowPayrate, "Delete test payrate", func() error {
		if testPayrateID == 0 {
			return fmt.Errorf("no test payrate ID")
		}
		if adopted {
			fmt.Printf("    Skipping delete — payrate %d pre-existed this run\n", testPayrateID)
			return nil
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/payrates/%d", testPayrateID)); err != nil {
			return fmt.Errorf("delete payrate: %w", err)
		}
		testPayrateID = 0
		fmt.Printf("    Test payrate deleted\n")
		return nil
	})

	reporter.RunTest(flowPayrate, "Edge: create payrate with missing effective_from", func() error {
		body := CreatePayrateRequest{
			Rates: []byte(`{"Nhân viên":{"Ngày thường":{"Ca ngày":250000}}}`),
		}
		_, statusCode, err := admin.PostExpectError(fmt.Sprintf("/api/v1/projects/%d/payrate", projectID), body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowPayrate, "Edge: update non-existent payrate", func() error {
		body := UpdatePayrateRequest{
			Rates:         []byte(`{}`),
			EffectiveFrom: today(),
		}
		_, statusCode, _ := admin.Put(fmt.Sprintf("/api/v1/payrates/%d", nonexistentID), body)
		if statusCode < 400 {
			return fmt.Errorf("expected error updating non-existent payrate")
		}
		return nil
	})

	// ── From-date-only model ─────────────────────────────────────────────
	// Rate changes append a new config strictly after the latest; superseded
	// configs are view-only; deletes heal the timeline by date resolution.
	// These scenarios own their v1/v2 pair — the base flow above deletes its
	// payrate before this point.
	tomorrow := clockHelperTomorrow()
	var v1ID, v2ID uint

	reporter.RunTest(flowPayrate, "New model: create current version succeeds", func() error {
		body := CreatePayrateRequest{
			Rates:         []byte(`{"Nhân viên":{"Ngày thường":{"Ca ngày":270000}}}`),
			EffectiveFrom: today(),
		}
		var resp PayrateResponse
		if _, err := admin.PostInto(fmt.Sprintf("/api/v1/projects/%d/payrate", projectID), body, &resp); err == nil {
			v1ID = resp.ID
		} else if existing, found := findPayrateFromDate(admin, projectID, today()); found && isTestPayrateRates(existing.Rates) {
			v1ID = existing.ID
			fmt.Printf("    Adopted existing v1 payrate ID %d\n", v1ID)
		} else {
			return fmt.Errorf("create v1: %w", err)
		}
		return AssertGreaterThan("v1 id", uint(0), v1ID)
	})

	defer func() {
		if v1ID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/payrates/%d", v1ID))
		}
	}()

	reporter.RunTest(flowPayrate, "New model: create on/before latest rejected", func() error {
		body := CreatePayrateRequest{
			Rates:         []byte(`{"Nhân viên":{"Ngày thường":{"Ca ngày":250000}}}`),
			EffectiveFrom: today(),
		}
		apiErr, statusCode, err := admin.PostExpectError(fmt.Sprintf("/api/v1/projects/%d/payrate", projectID), body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		if statusCode != 400 {
			return fmt.Errorf("expected 400, got %d", statusCode)
		}
		return AssertEqual("code", "PAYRATE_NOT_LATEST", apiErr.Code)
	})

	reporter.RunTest(flowPayrate, "New model: create version after latest succeeds", func() error {
		body := CreatePayrateRequest{
			Rates:         []byte(`{"Nhân viên":{"Ngày thường":{"Ca ngày":260000}}}`),
			EffectiveFrom: tomorrow,
		}
		var resp PayrateResponse
		if _, err := admin.PostInto(fmt.Sprintf("/api/v1/projects/%d/payrate", projectID), body, &resp); err == nil {
			v2ID = resp.ID
		} else if existing, found := findPayrateFromDate(admin, projectID, tomorrow); found && isTestPayrateRates(existing.Rates) {
			v2ID = existing.ID
			fmt.Printf("    Adopted existing v2 payrate ID %d\n", v2ID)
		} else {
			return fmt.Errorf("create v2: %w", err)
		}
		return AssertGreaterThan("v2 id", uint(0), v2ID)
	})

	defer func() {
		if v2ID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/payrates/%d", v2ID))
		}
	}()

	reporter.RunTest(flowPayrate, "New model: superseded config is view-only", func() error {
		body := UpdatePayrateRequest{
			Rates:         []byte(`{"Nhân viên":{"Ngày thường":{"Ca ngày":999000}}}`),
			EffectiveFrom: today(),
		}
		apiErr, statusCode, err := admin.PutExpectError(fmt.Sprintf("/api/v1/payrates/%d", v1ID), body)
		if err != nil {
			return fmt.Errorf("put: %w", err)
		}
		if statusCode != 400 {
			return fmt.Errorf("expected 400, got %d", statusCode)
		}
		return AssertEqual("code", "PAYRATE_NOT_LATEST", apiErr.Code)
	})

	reporter.RunTest(flowPayrate, "New model: no end date on the wire (from-date-only)", func() error {
		// Migration 138 dropped payrates.to_date; the API must not carry any
		// toDate key — neither derived nor stale.
		rows, _, err := vendorRawRows(admin, fmt.Sprintf("/api/v1/payrates?project_id=%d&pageSize=100", projectID))
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}
		if len(rows) == 0 {
			return fmt.Errorf("expected at least one payrate row")
		}
		for i, row := range rows {
			if _, present := row["toDate"]; present {
				return fmt.Errorf("row %d carries a toDate key — end dates must not be on the wire", i)
			}
		}
		return nil
	})

	reporter.RunTest(flowPayrate, "New model: supplied effective_to is rejected (400)", func() error {
		toDate := clockHelperDayAfter(clockHelperTomorrow())
		body := CreatePayrateRequest{
			Rates:         []byte(`{"Nhân viên":{"Ngày thường":{"Ca ngày":260000}}}`),
			EffectiveFrom: clockHelperTomorrow(),
			EffectiveTo:   &toDate,
		}
		_, statusCode, err := admin.PostExpectError(fmt.Sprintf("/api/v1/projects/%d/payrate", projectID), body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		if statusCode != 400 {
			return fmt.Errorf("expected 400, got %d", statusCode)
		}
		return nil
	})

	reporter.RunTest(flowPayrate, "New model: dry-run valid for a later date", func() error {
		body := map[string]any{
			"rates":          json.RawMessage(`{"Nhân viên":{"Ngày thường":{"Ca ngày":260000}}}`),
			"effective_from": clockHelperDayAfter(tomorrow),
		}
		// The validate endpoint responds with a bare body (no status/data
		// envelope), so read raw bytes instead of PostInto.
		raw, _, statusCode, err := admin.DownloadPost("/api/v1/payrates/validate?project_id="+fmt.Sprint(projectID), body)
		if err != nil {
			return fmt.Errorf("validate: %w", err)
		}
		if statusCode != 200 {
			return fmt.Errorf("expected 200, got %d: %s", statusCode, string(raw))
		}
		var resp struct {
			Valid                     bool   `json:"valid"`
			MutableTimesheetsAffected *int64 `json:"mutable_timesheets_affected"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		if !resp.Valid {
			return fmt.Errorf("expected valid dry-run for a date after the latest config")
		}
		if resp.MutableTimesheetsAffected == nil {
			return fmt.Errorf("mutable_timesheets_affected missing from response")
		}
		return nil
	})

	reporter.RunTest(flowPayrate, "New model: dry-run rejects fold-colliding keys", func() error {
		body := map[string]any{
			"rates":          json.RawMessage(`{"Nhân viên":{"Ngày thường":{"Ca ngày":10000}},"nhân viên":{"Ngày thường":{"Ca ngày":20000}}}`),
			"effective_from": clockHelperDayAfter(tomorrow),
		}
		raw, _, statusCode, err := admin.DownloadPost("/api/v1/payrates/validate?project_id="+fmt.Sprint(projectID), body)
		if err != nil {
			return fmt.Errorf("validate: %w", err)
		}
		if statusCode != 200 {
			return fmt.Errorf("expected 200, got %d: %s", statusCode, string(raw))
		}
		var resp struct {
			Valid  bool `json:"valid"`
			Fields struct {
				Rates struct {
					Status  string `json:"status"`
					Message string `json:"message"`
				} `json:"rates"`
			} `json:"fields"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}
		if resp.Valid || resp.Fields.Rates.Status != "error" {
			return fmt.Errorf("expected fold-collision rejection, got valid=%v status=%q", resp.Valid, resp.Fields.Rates.Status)
		}
		return AssertContains("message", resp.Fields.Rates.Message, "Trùng tên")
	})

	reporter.RunTest(flowPayrate, "New model: delete heals timeline", func() error {
		if v2ID == 0 {
			return fmt.Errorf("no v2 payrate")
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/payrates/%d", v2ID)); err != nil {
			return fmt.Errorf("delete v2: %w", err)
		}
		v2ID = 0 // deferred cleanup must skip
		var payrates []PayrateResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/payrates?project_id=%d", projectID), &payrates); err != nil {
			return fmt.Errorf("list: %w", err)
		}
		for i := range payrates {
			if payrates[i].ID == v1ID {
				return nil
			}
		}
		return fmt.Errorf("v1 %d not found in list", v1ID)
	})
}

// clockHelperTomorrow returns tomorrow's date string on the business clock
// (Asia/Ho_Chi_Minh), same source as today(). Host-local time.Now() desyncs
// from the business clock between 23:00 ICT and midnight whenever the host
// runs a different timezone (e.g. +08), which flipped the day and made the
// derived-end-date assertion fail near midnight ICT.
func clockHelperTomorrow() string {
	return clock.Now().AddDate(0, 0, 1).Format("2006-01-02")
}

// clockHelperDayAfter returns the day after the given date string.
func clockHelperDayAfter(dateStr string) string {
	t, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		return dateStr
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

// findPayrateFromDate returns the project's payrate effective on the given
// date, if one exists.
func findPayrateFromDate(admin *APIClient, projectID uint, date string) (*PayrateResponse, bool) {
	var payrates []PayrateResponse
	if _, err := admin.GetInto(fmt.Sprintf("/api/v1/payrates?project_id=%d", projectID), &payrates); err != nil {
		return nil, false
	}
	for i := range payrates {
		if payrates[i].FromDate == date {
			return &payrates[i], true
		}
	}
	return nil, false
}

// findTodayPayrate returns the project's active payrate whose effective date is
// today, if one exists.
func findTodayPayrate(admin *APIClient, projectID uint) (*PayrateResponse, bool) {
	var payrates []PayrateResponse
	if _, err := admin.GetInto(fmt.Sprintf("/api/v1/payrates?project_id=%d", projectID), &payrates); err != nil {
		return nil, false
	}
	for i := range payrates {
		if payrates[i].FromDate == today() {
			return &payrates[i], true
		}
	}
	return nil, false
}

// isTestPayrateRates reports whether raw matches the exact rates document this
// flow writes (create signature 250000 or post-update signature 300000), i.e.
// a leftover from an earlier run of this test rather than business data.
func isTestPayrateRates(raw json.RawMessage) bool {
	var got interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		return false
	}
	for _, sig := range []string{
		`{"Nhân viên":{"Ngày thường":{"Ca ngày":250000}}}`,
		`{"Nhân viên":{"Ngày thường":{"Ca ngày":300000}}}`,
	} {
		var want interface{}
		if err := json.Unmarshal([]byte(sig), &want); err == nil && reflect.DeepEqual(got, want) {
			return true
		}
	}
	return false
}
