package main

import (
	"fmt"
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
		if _, err := admin.PostInto(fmt.Sprintf("/api/v1/projects/%d/payrate", projectID), body, &resp); err != nil {
			return fmt.Errorf("create payrate: %w", err)
		}
		testPayrateID = resp.ID
		fmt.Printf("    Created payrate ID %d for project %d\n", resp.ID, projectID)
		return AssertGreaterThan("id", uint(0), resp.ID)
	})

	defer func() {
		if testPayrateID != 0 {
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
		_, statusCode, _ := admin.Put("/api/v1/payrates/999999", body)
		if statusCode < 400 {
			return fmt.Errorf("expected error updating non-existent payrate")
		}
		return nil
	})
}
