package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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

	reporter.RunTest(flowSettings, "Configure bulk transfer workbook limit", func() error {
		const settingKey = "bulk_transfer_workbook_limit_vnd"

		var current SettingResponse
		if _, err := admin.GetInto("/api/v1/settings/key/"+settingKey, &current); err != nil {
			return fmt.Errorf("get bulk transfer workbook limit: %w", err)
		}
		if current.Value == nil {
			return fmt.Errorf("bulk transfer workbook limit must have a value")
		}

		originalValue := *current.Value
		defer func() {
			restore := originalValue
			_, _, _ = admin.Put(
				fmt.Sprintf("/api/v1/settings/%d", current.ID),
				UpdateSettingRequest{Value: &restore},
			)
		}()

		for _, invalidValue := range []string{"1", "+400000000", "0400000000"} {
			value := invalidValue
			_, invalidStatus, _ := admin.Put(
				fmt.Sprintf("/api/v1/settings/%d", current.ID),
				UpdateSettingRequest{Value: &value},
			)
			if invalidStatus < 400 {
				return fmt.Errorf("expected non-canonical workbook limit %q to be rejected", invalidValue)
			}
		}

		updatedValue := "399000000"
		_, status, err := admin.Put(
			fmt.Sprintf("/api/v1/settings/%d", current.ID),
			UpdateSettingRequest{Value: &updatedValue},
		)
		if err != nil {
			return fmt.Errorf("update bulk transfer workbook limit: %w", err)
		}
		if status >= 400 {
			return fmt.Errorf("update bulk transfer workbook limit returned status %d", status)
		}

		var updated SettingResponse
		if _, err := admin.GetInto("/api/v1/settings/key/"+settingKey, &updated); err != nil {
			return fmt.Errorf("get bulk transfer workbook limit after update: %w", err)
		}
		if updated.Value == nil {
			return fmt.Errorf("updated bulk transfer workbook limit must have a value")
		}
		return AssertEqual("bulk transfer workbook limit", updatedValue, *updated.Value)
	})

	reporter.RunTest(flowSettings, "Configure self check-in advance wait", func() error {
		const settingKey = "self_check_in_advance_hold_hours"

		var current SettingResponse
		if _, err := admin.GetInto("/api/v1/settings/key/"+settingKey, &current); err != nil {
			return fmt.Errorf("get self check-in advance wait: %w", err)
		}
		if current.Value == nil {
			return fmt.Errorf("self check-in advance wait must have a value")
		}

		originalValue := *current.Value
		defer func() {
			restore := originalValue
			_, _, _ = admin.Put(
				fmt.Sprintf("/api/v1/settings/%d", current.ID),
				UpdateSettingRequest{Value: &restore},
			)
		}()

		for _, invalidValue := range []string{"-1", "24.5", "024", "721"} {
			value := invalidValue
			_, invalidStatus, _ := admin.Put(
				fmt.Sprintf("/api/v1/settings/%d", current.ID),
				UpdateSettingRequest{Value: &value},
			)
			if invalidStatus < 400 {
				return fmt.Errorf("expected invalid self check-in advance wait %q to be rejected", invalidValue)
			}
		}

		updatedValue := "6"
		_, status, err := admin.Put(
			fmt.Sprintf("/api/v1/settings/%d", current.ID),
			UpdateSettingRequest{Value: &updatedValue},
		)
		if err != nil {
			return fmt.Errorf("update self check-in advance wait: %w", err)
		}
		if status >= 400 {
			return fmt.Errorf("update self check-in advance wait returned status %d", status)
		}

		var updated SettingResponse
		if _, err := admin.GetInto("/api/v1/settings/key/"+settingKey, &updated); err != nil {
			return fmt.Errorf("get self check-in advance wait after update: %w", err)
		}
		if updated.Value == nil {
			return fmt.Errorf("updated self check-in advance wait must have a value")
		}
		return AssertEqual("self check-in advance wait", updatedValue, *updated.Value)
	})

	reporter.RunTest(flowSettings, "Configure beneficiary bank accounts (weekly and FlexPay)", func() error {
		// These rows are optional: an environment that never saved the sao kê
		// bank panel has no row for them, so each key is created on demand and
		// every touched row is restored (or removed) when the test ends.
		var restores []func()
		defer func() {
			for i := len(restores) - 1; i >= 0; i-- {
				restores[i]()
			}
		}()

		// rowIDFor returns the row id for key, creating it (and registering its
		// removal) when the key was never seeded.
		rowIDFor := func(key string) (uint, error) {
			resp, status, err := admin.Get("/api/v1/settings/key/" + key)
			if err != nil {
				return 0, fmt.Errorf("get %s: %w", key, err)
			}
			switch status {
			case http.StatusOK:
				var existing SettingResponse
				if err := json.Unmarshal(resp.Data, &existing); err != nil {
					return 0, fmt.Errorf("decode %s: %w", key, err)
				}
				original := ""
				if existing.Value != nil {
					original = *existing.Value
				}
				restores = append(restores, func() {
					restore := original
					_, _, _ = admin.Put(fmt.Sprintf("/api/v1/settings/%d", existing.ID), UpdateSettingRequest{Value: &restore})
				})
				return existing.ID, nil
			case http.StatusNotFound:
				seed := "true"
				var created SettingResponse
				if _, err := admin.PostInto("/api/v1/settings", CreateSettingRequest{Key: key, Value: &seed, ValueType: "string"}, &created); err != nil {
					return 0, fmt.Errorf("create %s: %w", key, err)
				}
				restores = append(restores, func() {
					_, _, _ = admin.Delete(fmt.Sprintf("/api/v1/settings/%d", created.ID))
				})
				return created.ID, nil
			default:
				return 0, fmt.Errorf("get %s returned HTTP %d: %s", key, status, resp.Message)
			}
		}

		// The two beneficiary accounts are configured independently: the weekly
		// payroll statement and the FlexPay (OnePay) reconciliation statement
		// each print their own holder, account number, bank and visibility.
		values := map[string]string{
			"transfer_bank_account_holder":         "CONG TY ITEST LUONG TUAN",
			"transfer_bank_account_number":         "111222333",
			"transfer_bank_name":                   "Ngân hàng Tuần ITest",
			"transfer_bank_visible":                "true",
			"flexpay_transfer_bank_account_holder": "CONG TY ITEST FLEXPAY",
			"flexpay_transfer_bank_account_number": "444555666",
			"flexpay_transfer_bank_name":           "Ngân hàng FlexPay ITest",
			"flexpay_transfer_bank_visible":        "false",
		}

		for key, value := range values {
			id, err := rowIDFor(key)
			if err != nil {
				return err
			}
			if _, status, err := admin.Put(fmt.Sprintf("/api/v1/settings/%d", id), UpdateSettingRequest{Value: &value}); err != nil || status >= 400 {
				return fmt.Errorf("update %s: HTTP %d: %v", key, status, err)
			}
		}

		for key, want := range values {
			var row SettingResponse
			if _, err := admin.GetInto("/api/v1/settings/key/"+key, &row); err != nil {
				return fmt.Errorf("read back %s: %w", key, err)
			}
			if row.Value == nil {
				return fmt.Errorf("%s has no value after save", key)
			}
			if err := AssertEqual(key, want, *row.Value); err != nil {
				return err
			}
		}

		fmt.Printf("    Weekly and FlexPay beneficiary accounts persist independently\n")
		return nil
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
		_, statusCode, _ := admin.Put(fmt.Sprintf("/api/v1/settings/%d", nonexistentID), body)
		if statusCode < 400 {
			return fmt.Errorf("expected error updating non-existent setting")
		}
		return nil
	})
}
