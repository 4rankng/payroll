package main

import (
	"encoding/json"
	"fmt"
	"reflect"
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
}

// findTodayPayrate returns the project's active payrate whose effective date is
// today, if one exists.
func findTodayPayrate(admin *APIClient, projectID uint) (*PayrateResponse, bool) {
	var payrates []PayrateResponse
	if _, err := admin.GetInto(fmt.Sprintf("/api/v1/payrates?project_id=%d", projectID), &payrates); err != nil {
		return nil, false
	}
	for i := range payrates {
		if payrates[i].FromDate == today() && payrates[i].ToDate == nil {
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
