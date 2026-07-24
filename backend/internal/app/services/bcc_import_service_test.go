package services

import (
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseForMonth(t *testing.T) {
	tests := []struct {
		input   string
		wantY   int
		wantM   int
		wantErr bool
	}{
		{"2026-05", 2026, 5, false},
		{"2026-01", 2026, 1, false},
		{"2026-12", 2026, 12, false},
		{"2026-13", 0, 0, true},
		{"2026-00", 0, 0, true},
		{"0000-05", 0, 0, true},
		{"1999-05", 0, 0, true},
		{"2101-05", 0, 0, true},
		{"invalid", 0, 0, true},
		{"2026/05", 0, 0, true},
		{"", 0, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			y, m, err := parseForMonth(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parseForMonth(%q) = (%d, %s, nil), want error", tc.input, y, m)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseForMonth(%q) unexpected error: %v", tc.input, err)
			}
			if y != tc.wantY || int(m) != tc.wantM {
				t.Errorf("parseForMonth(%q) = (%d, %d, nil), want (%d, %d)", tc.input, y, m, tc.wantY, tc.wantM)
			}
		})
	}
}

func TestFirstErrorReason(t *testing.T) {
	t.Run("nil detail", func(t *testing.T) {
		if got := FirstErrorReason(nil); got != "unknown" {
			t.Errorf("FirstErrorReason(nil) = %q, want %q", got, "unknown")
		}
	})
	t.Run("plain string", func(t *testing.T) {
		s := "some plain error"
		if got := FirstErrorReason(&s); got != "some plain error" {
			t.Errorf("got %q, want %q", got, "some plain error")
		}
	})
	t.Run("JSON array", func(t *testing.T) {
		s := `[{"row":1,"employee":"test","reason":"lỗi A"}]`
		if got := FirstErrorReason(&s); got != "lỗi A" {
			t.Errorf("got %q, want %q", got, "lỗi A")
		}
	})
	t.Run("empty JSON array", func(t *testing.T) {
		s := `[]`
		if got := FirstErrorReason(&s); got != "[]" {
			t.Errorf("got %q, want original string", got)
		}
	})
}

func TestMarshalErrors(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		if got := marshalErrors(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("empty slice", func(t *testing.T) {
		if got := marshalErrors([]domain.ImportError{}); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("non-empty", func(t *testing.T) {
		errs := []domain.ImportError{{Row: 1, Employee: "Test", Reason: "reason"}}
		got := marshalErrors(errs)
		if got == nil {
			t.Fatal("expected non-nil")
		}
		// Verify it round-trips through JSON.
		var parsed []domain.ImportError
		if err := json.Unmarshal([]byte(*got), &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if len(parsed) != 1 || parsed[0].Reason != "reason" {
			t.Errorf("unexpected JSON: %s", *got)
		}
	})
}

func TestRateMatchingCollisionPriority(t *testing.T) {
	dayTypePriority := map[string]int{"ngày thường": 0, "ngày nghỉ": 1, "ngày lễ": 2}
	type rateTarget struct{ dayType, hourType string }

	rateToTarget := make(map[int]rateTarget)
	flatRates := map[string]int64{
		"project.ngày thường.regular": 36000,
		"project.ngày nghỉ.regular":   36000, // same rate, lower priority
	}
	for path, rate := range flatRates {
		if rate == 0 {
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) != 3 {
			continue
		}
		candidate := rateTarget{parts[1], parts[2]}
		candPri, candKnown := dayTypePriority[candidate.dayType]
		if !candKnown {
			continue
		}
		existing, exists := rateToTarget[int(rate)]
		if !exists || candPri < dayTypePriority[existing.dayType] {
			rateToTarget[int(rate)] = candidate
		}
	}

	got := rateToTarget[36000]
	if got.dayType != "ngày thường" {
		t.Errorf("expected 'ngày thường' to win on collision, got %q", got.dayType)
	}
}

func TestBuildResult(t *testing.T) {
	detail := "some error"
	now := clock.Now()
	stats := BCCImportStats{
		ProjectID:    1,
		OriginalName: "test.xlsx",
		ForMonth:     "2026-05",
		Status:       "completed",
		TotalRows:    10,
		CreatedCount: 8,
		SkippedCount: 1,
		ErrorCount:   1,
		ErrorDetail:  &detail,
		ProcessedAt:  &now,
	}
	r := buildResult(stats, 42, 7, now)
	if r.ID != 42 {
		t.Errorf("ID = %d, want 42", r.ID)
	}
	if r.UploadedBy != 7 {
		t.Errorf("UploadedBy = %d, want 7", r.UploadedBy)
	}
	if r.ProjectID != 1 {
		t.Errorf("ProjectID = %d, want 1", r.ProjectID)
	}
	if r.CreatedCount != 8 {
		t.Errorf("CreatedCount = %d, want 8", r.CreatedCount)
	}
	if r.Status != "completed" {
		t.Errorf("Status = %q, want completed", r.Status)
	}
}

func TestBCCRequestFingerprintBindsScopeAndContent(t *testing.T) {
	base := bccRequestFingerprint(10, "2026-07", []byte("same workbook"))
	if base != bccRequestFingerprint(10, "2026-07", []byte("same workbook")) {
		t.Fatal("same request must have a stable fingerprint")
	}
	if base == bccRequestFingerprint(11, "2026-07", []byte("same workbook")) {
		t.Fatal("project must be part of the fingerprint")
	}
	if base == bccRequestFingerprint(10, "2026-08", []byte("same workbook")) {
		t.Fatal("month must be part of the fingerprint")
	}
	if base == bccRequestFingerprint(10, "2026-07", []byte("different workbook")) {
		t.Fatal("file content must be part of the fingerprint")
	}
}

func TestSTKCrossCheckLooseMatch(t *testing.T) {
	// Simulate the STK cross-check logic from ProcessUpload
	bccName := "Lò Thị Dương"
	stkName := "Lò Thì Dương"

	bccNorm := bccNormName(bccName)
	stkNorm := bccNormName(stkName)

	if bccNorm == stkNorm {
		t.Fatal("Expected strict normalization to fail")
	}

	if bccNormNameLoose(bccName) != bccNormNameLoose(stkName) {
		t.Errorf("Expected loose normalization to match for %q and %q", bccName, stkName)
	}
}
