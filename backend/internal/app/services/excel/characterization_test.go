package excel

import "testing"

// Phase 0 characterization tests (excel-parsing-refactor): real partner files
// from tests/fixtures/bcc/<format>/, parse output locked as goldens. These
// guard the routing/toolkit refactor — a golden diff means behavior changed.

func TestCharacterization_Legacy_EVA_T08(t *testing.T) {
	f := openBCCFixture(t, "legacy", "eva_t08.xlsx")

	det, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if det.Format != FormatLegacy {
		t.Fatalf("DetectFormat = %v, want FormatLegacy", det.Format)
	}

	data, err := ParseBCCFile(f)
	if err != nil {
		t.Fatalf("ParseBCCFile: %v", err)
	}
	assertGoldenJSON(t, "legacy_eva_t08", data)
}

// BUMHAN T08 ships its date-row layout under a "BCC"-named sheet (T09
// family). DetectFormat sees the sheet name (FormatLegacy); the strategy
// router must resolve the real layout — the winner-format contract.
func TestCharacterization_DateRow_Bumhan_T08(t *testing.T) {
	f := openBCCFixture(t, "date_row", "bumhan_t08.xlsx")

	det, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if det.Format != FormatLegacy {
		t.Fatalf("DetectFormat = %v, want FormatLegacy (sheet-name detection)", det.Format)
	}

	data, winner, err := ParseBCCData(f)
	if err != nil {
		t.Fatalf("ParseBCCData: %v", err)
	}
	if winner != FormatDateRow {
		t.Fatalf("winner format = %v, want FormatDateRow (strategy fallback contract)", winner)
	}
	assertGoldenJSON(t, "date_row_bumhan_t08", data)
}

func TestCharacterization_WeeklyBCC_LGD(t *testing.T) {
	f := openBCCFixture(t, "weekly_bcc", "lgd_weekly.xlsx")

	det, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if det.Format != FormatWeeklyBCC {
		t.Fatalf("DetectFormat = %v, want FormatWeeklyBCC", det.Format)
	}
	if len(det.WeeklyBCCSheets) != 7 {
		t.Fatalf("WeeklyBCCSheets = %v, want 7 (BCC-HC…OT390)", det.WeeklyBCCSheets)
	}

	data, err := ParseWeeklyBCCFile(f, det.WeeklyBCCSheets)
	if err != nil {
		t.Fatalf("ParseWeeklyBCCFile: %v", err)
	}
	assertGoldenJSON(t, "weekly_bcc_lgd", data)

	stk, err := ParseSTKSheet(f)
	if err != nil {
		t.Fatalf("ParseSTKSheet: %v", err)
	}
	assertGoldenJSON(t, "weekly_bcc_lgd_stk", stk)
}

func TestCharacterization_WeeklyPayment_TBD(t *testing.T) {
	f := openBCCFixture(t, "weekly_payment", "tbd_weekly_payment.xlsx")

	det, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if det.Format != FormatWeeklyPayment {
		t.Fatalf("DetectFormat = %v, want FormatWeeklyPayment", det.Format)
	}

	// forMonth comes from the upload request; the file covers kỳ 4 tháng 08/2026.
	data, err := ParseWeeklyPaymentFile(f, det.WeeklyPaymentSheets, "2026-08")
	if err != nil {
		t.Fatalf("ParseWeeklyPaymentFile: %v", err)
	}
	assertGoldenJSON(t, "weekly_payment_tbd", data)

	stk, err := ParseSTKSheet(f)
	if err != nil {
		t.Fatalf("ParseSTKSheet: %v", err)
	}
	assertGoldenJSON(t, "weekly_payment_tbd_stk", stk)
}

func TestCharacterization_MultiPosition_PQC(t *testing.T) {
	f := openBCCFixture(t, "multi_position", "pqc_haiphong.xlsx")

	det, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if det.Format != FormatMultiPosition {
		t.Fatalf("DetectFormat = %v, want FormatMultiPosition", det.Format)
	}

	data, err := ParseMultiPositionFile(f, det.PositionSheets)
	if err != nil {
		t.Fatalf("ParseMultiPositionFile: %v", err)
	}
	assertGoldenJSON(t, "multi_position_pqc", data)
}
