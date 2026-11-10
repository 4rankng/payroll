package services

import (
	"testing"
	"time"

	excelparser "api-server/internal/app/services/excel"

	"golang.org/x/text/unicode/norm"
)

func TestValidateSheetPositions_Normalization(t *testing.T) {
	resolver := mustTestResolver(t, time.Local,
		testPayrate(1, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.Local), `{
			"Chia chọn.ngày thường.ca ngày": 37500,
			"Chia chọn.ngày thường.tăng ca": 56250,
			"Xuất hàng.ngày thường.ca ngày": 48750,
			"Xuất hàng.ngày thường.tăng ca": 56250,
			"Lái xe nâng.ngày thường.ca ngày": 42307,
			"Lái xe nâng.ngày thường.tăng ca": 63460
		}`))
	monthStart := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, -1)

	// Test matching when sheet name has different casing, spacing, and NFD diacritics
	parsed := &excelparser.MultiPositionImportData{
		Sheets: []excelparser.PositionSheetData{
			{Position: "Chia chọn"},
			{Position: "  Lái xe nâng  "},
			{Position: norm.NFD.String("Xuất Hàng")}, // NFD macOS decomposed
		},
	}

	reason := validateSheetPositions(parsed, resolver, monthStart, monthEnd)
	if reason != "" {
		t.Fatalf("expected valid positions, got error: %s", reason)
	}

	// Test missing position
	parsedMissing := &excelparser.MultiPositionImportData{
		Sheets: []excelparser.PositionSheetData{
			{Position: "Bốc xếp"},
		},
	}
	reasonMissing := validateSheetPositions(parsedMissing, resolver, monthStart, monthEnd)
	if reasonMissing == "" {
		t.Fatal("expected failure for unconfigured position 'Bốc xếp', got empty reason")
	}
}

func TestBuildRateToTarget_OneVNDTolerance(t *testing.T) {
	flatRates := map[string]int{
		"Lái xe nâng.ngày thường.ca ngày": 42307,
		"Lái xe nâng.ngày thường.tăng ca": 63460, // 42307 * 1.5 = 63460.5 -> rounded to 63461 in Excel
	}

	rateToTarget := buildRateToTarget(flatRates, bccNormName("Lái xe nâng")+".")
	if len(rateToTarget) != 2 {
		t.Fatalf("expected 2 targets, got %d: %v", len(rateToTarget), rateToTarget)
	}

	// Exact match through the same lookup the import uses
	if target, ok := lookupRateTarget(rateToTarget, 42307); !ok || target.hourType != "ca ngày" {
		t.Fatalf("expected 42307 -> ca ngày, got %v, ok=%v", target, ok)
	}

	// Entry rate 63461 (Excel formatted integer) vs config 63460
	target, ok := lookupRateTarget(rateToTarget, 63461)
	if !ok || target.hourType != "tăng ca" {
		t.Fatalf("expected entryRate 63461 to match 63460 via 1-VND tolerance -> tăng ca, got %v, ok=%v", target, ok)
	}

	// An exact config rate still wins over its ±1 neighbours
	if target, ok := lookupRateTarget(rateToTarget, 63460); !ok || target.hourType != "tăng ca" {
		t.Fatalf("expected exact 63460 -> tăng ca, got %v, ok=%v", target, ok)
	}

	// Unrelated rates match nothing
	if _, ok := lookupRateTarget(rateToTarget, 50000); ok {
		t.Fatal("expected 50000 to match no target")
	}
}
