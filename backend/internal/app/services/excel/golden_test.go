package excel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Golden-file harness for Phase 0 characterization (excel-parsing-refactor).
// Goldens live in testdata/goldens/*.golden.json and lock CURRENT parse
// behavior. Regenerate only after an intentional behavior change:
//
//	GOLDEN_UPDATE=1 go test ./internal/app/services/excel/ -run TestCharacterization
func assertGoldenJSON(t *testing.T, name string, got any) {
	t.Helper()
	dir := filepath.Join("testdata", "goldens")
	path := filepath.Join(dir, name+".golden.json")

	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}
	data = append(data, '\n')

	if os.Getenv("GOLDEN_UPDATE") == "1" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir goldens: %v", err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		t.Logf("golden updated: %s", path)
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden missing (regenerate with GOLDEN_UPDATE=1): %s", path)
		}
		t.Fatalf("read golden %s: %v", path, err)
	}
	if strings.TrimSpace(string(want)) != strings.TrimSpace(string(data)) {
		t.Errorf("golden mismatch: %s\ndiff: run with GOLDEN_UPDATE=1 and inspect the change", name)
	}
}

// bccFixturePath resolves a path under backend/tests/fixtures/bcc/.
func bccFixturePath(t *testing.T, rel ...string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(append([]string{cwd, "..", "..", "..", "..", "tests", "fixtures", "bcc"}, rel...)...)
}

// openBCCFixture opens a committed real-partner fixture workbook; skips the
// test when the file is absent (e.g. fresh clone before fixture commit).
func openBCCFixture(t *testing.T, rel ...string) *excelize.File {
	t.Helper()
	f, err := excelize.OpenFile(bccFixturePath(t, rel...))
	if err != nil {
		t.Skipf("fixture unavailable: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}
