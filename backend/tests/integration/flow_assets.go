package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const flowAssets = "Assets"

func runAssetTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Assets")

	admin := client.WithToken(data.AdminToken)
	prefix := cfg.UniquePrefix()

	var testAssetID uint

	// Create a temp file for upload
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, prefix+"_upload.txt")
	if err := os.WriteFile(tmpFile, []byte("itest upload content"), 0644); err != nil {
		return
	}
	defer func() { _ = os.Remove(tmpFile) }()

	reporter.RunTest(flowAssets, "Upload test file", func() error {
		resp, status, err := admin.UploadFile("/api/v1/assets/upload", "file", tmpFile, map[string]string{
			"upload_type": "document",
		})
		if err != nil {
			return fmt.Errorf("upload file: %w", err)
		}
		if status >= 400 || resp.Status == "error" {
			return fmt.Errorf("upload returned HTTP %d: %s", status, resp.Message)
		}
		var asset AssetResponse
		if err := json.Unmarshal(resp.Data, &asset); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
		testAssetID = asset.ID
		fmt.Printf("    Uploaded asset ID %d\n", asset.ID)
		return AssertGreaterThan("id", uint(0), asset.ID)
	})

	reporter.RunTest(flowAssets, "List assets", func() error {
		var list interface{}
		if _, err := admin.GetInto("/api/v1/assets?pageSize=10", &list); err != nil {
			return fmt.Errorf("list assets: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowAssets, "Get asset metadata by ID", func() error {
		if testAssetID == 0 {
			return fmt.Errorf("no test asset ID")
		}
		var resp AssetResponse
		if _, err := admin.GetInto(fmt.Sprintf("/api/v1/assets/%d", testAssetID), &resp); err != nil {
			return fmt.Errorf("get asset: %w", err)
		}
		return AssertEqual("id", testAssetID, resp.ID)
	})

	reporter.RunTest(flowAssets, "Download asset binary", func() error {
		if testAssetID == 0 {
			return fmt.Errorf("no test asset ID")
		}
		assetData, _, statusCode, err := admin.DownloadGet(fmt.Sprintf("/api/v1/assets/%d/download", testAssetID))
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}
		if err := AssertGreaterOrEqual("status", 200, statusCode); err != nil {
			return err
		}
		return AssertGreaterThan("file_size", 0, len(assetData))
	})

	reporter.RunTest(flowAssets, "Edge: get non-existent asset", func() error {
		_, statusCode, _ := admin.Get("/api/v1/assets/999999")
		if statusCode < 400 {
			return fmt.Errorf("expected error for non-existent asset, got HTTP %d", statusCode)
		}
		return nil
	})
}
