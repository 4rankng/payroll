package main

import (
	"fmt"
)

const flowSettings = "Settings"

func runSettingsTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Settings CRUD")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()

	var testSettingID uint

	reporter.RunTest(flowSettings, "List current settings", func() error {
		var list interface{}
		if _, err := admin.GetInto("/api/v1/settings", &list); err != nil {
			return fmt.Errorf("list settings: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowSettings, "Create test setting", func() error {
		val := prefix + "_value"
		body := CreateSettingRequest{
			Key:       prefix + "_test_key",
			Value:     &val,
			ValueType: "string",
		}
		var resp SettingResponse
		if _, err := admin.PostInto("/api/v1/settings", body, &resp); err != nil {
			return fmt.Errorf("create setting: %w", err)
		}
		testSettingID = resp.ID
		fmt.Printf("    Created setting ID %d, key: %s\n", resp.ID, resp.Key)
		if err := AssertGreaterThan("id", uint(0), resp.ID); err != nil {
			return err
		}
		return AssertEqual("value_type", "string", resp.ValueType)
	})

	defer func() {
		if testSettingID != 0 {
			_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/settings/%d", testSettingID))
		}
	}()

	reporter.RunTest(flowSettings, "Get setting by ID", func() error {
		if testSettingID == 0 {
			return fmt.Errorf("no test setting ID")
		}
		var resp SettingResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/settings/%d", testSettingID), &resp); err != nil {
			return fmt.Errorf("get setting: %w", err)
		}
		return AssertEqual("id", testSettingID, resp.ID)
	})

	reporter.RunTest(flowSettings, "Update setting value", func() error {
		if testSettingID == 0 {
			return fmt.Errorf("no test setting ID")
		}
		newVal := prefix + "_updated"
		body := UpdateSettingRequest{Value: &newVal}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/settings/%d", testSettingID), body); err != nil {
			return fmt.Errorf("update setting: %w", err)
		}
		var resp SettingResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/settings/%d", testSettingID), &resp); err != nil {
			return fmt.Errorf("get after update: %w", err)
		}
		if resp.Value == nil {
			return fmt.Errorf("expected non-nil value after update")
		}
		return AssertEqual("value", newVal, *resp.Value)
	})

	reporter.RunTest(flowSettings, "Delete test setting", func() error {
		if testSettingID == 0 {
			return fmt.Errorf("no test setting ID")
		}
		if _, _, err := admin.Delete(fmt.Sprintf("/api/v1/settings/%d", testSettingID)); err != nil {
			return fmt.Errorf("delete setting: %w", err)
		}
		testSettingID = 0
		fmt.Printf("    Test setting deleted\n")
		return nil
	})

	reporter.RunTest(flowSettings, "Edge: create setting with invalid value_type", func() error {
		body := CreateSettingRequest{
			Key:       prefix + "_bad_type",
			ValueType: "invalid_type",
		}
		_, statusCode, err := admin.PostExpectError("/api/v1/settings", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})

	reporter.RunTest(flowSettings, "Edge: update non-existent setting", func() error {
		newVal := "ghost"
		body := UpdateSettingRequest{Value: &newVal}
		_, statusCode, _ := admin.Put("/api/v1/settings/999999", body)
		if statusCode < 400 {
			return fmt.Errorf("expected error updating non-existent setting")
		}
		return nil
	})
}
