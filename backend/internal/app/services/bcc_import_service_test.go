package services

import (
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"
	"api-server/internal/pkg/clock"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
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

func TestSafeWeeklyPaymentParseError(t *testing.T) {
	if got := safeWeeklyPaymentParseError(errors.New("cột F: ngày 32 nằm ngoài kỳ nhập 08/2026")); got != "Ngày trong tệp nằm ngoài tháng đã chọn. Vui lòng kiểm tra tệp." {
		t.Errorf("out-of-month date = %q", got)
	}
	if got := safeWeeklyPaymentParseError(errors.New("driver: malformed cell")); got != "Không thể đọc cấu trúc tệp chấm công. Vui lòng dùng đúng mẫu tệp." {
		t.Errorf("unexpected parser failure = %q", got)
	}
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
	base := bccRequestFingerprint(10, "2026-07", false, []byte("same workbook"))
	if base != bccRequestFingerprint(10, "2026-07", false, []byte("same workbook")) {
		t.Fatal("same request must have a stable fingerprint")
	}
	if base == bccRequestFingerprint(11, "2026-07", false, []byte("same workbook")) {
		t.Fatal("project must be part of the fingerprint")
	}
	if base == bccRequestFingerprint(10, "2026-08", false, []byte("same workbook")) {
		t.Fatal("month must be part of the fingerprint")
	}
	if base == bccRequestFingerprint(10, "2026-07", false, []byte("different workbook")) {
		t.Fatal("file content must be part of the fingerprint")
	}
	if base == bccRequestFingerprint(10, "2026-07", true, []byte("same workbook")) {
		t.Fatal("flexible-employee import mode must be part of the fingerprint")
	}
}

func TestRequireBCCImportApproval(t *testing.T) {
	entries := []domainservices.BulkCreateTimesheetEntry{
		{ProjectID: 1, EmployeeID: 10},
		{ProjectID: 1, EmployeeID: 11},
	}

	requireBCCImportApproval(entries)

	for _, entry := range entries {
		if !entry.RequireApproval {
			t.Fatalf("BCC entry for employee %d must require approval", entry.EmployeeID)
		}
	}
}

func TestAdminBCCImportEntryRemainsPendingApproval(t *testing.T) {
	entries := []domainservices.BulkCreateTimesheetEntry{{ProjectID: 1, EmployeeID: 10}}
	requireBCCImportApproval(entries)

	timesheet := &domain.Timesheet{}
	service := &domainservices.TimesheetDomainService{}
	if err := service.SetInitialTimesheetStatusForBulk(context.Background(), timesheet, 42, "admin", entries[0].RequireApproval); err != nil {
		t.Fatalf("SetInitialTimesheetStatusForBulk() error = %v", err)
	}
	if timesheet.Status != domain.TimesheetStatusPendingApproval {
		t.Fatalf("status = %q, want %q", timesheet.Status, domain.TimesheetStatusPendingApproval)
	}
	if timesheet.ApprovedBy != nil || timesheet.ApprovedAt != nil {
		t.Fatal("admin BCC import must not have approval metadata")
	}
}

func TestPlanBCCReplacementSkipsReviewedAndUpsertsPendingOrMissing(t *testing.T) {
	date := time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC)
	entries := []domainservices.BulkCreateTimesheetEntry{
		{EmployeeID: 1, Date: "2026-08-08", HourType: "HC"},
		{EmployeeID: 2, Date: "2026-08-08", HourType: "TCN"},
		{EmployeeID: 3, Date: "2026-08-08", HourType: "HC"},
		{EmployeeID: 4, Date: "2026-08-08", HourType: "HC"},
		{EmployeeID: 5, Date: "2026-08-08", HourType: "HC"},
	}
	existing := []*domain.Timesheet{
		{ID: 11, EmployeeID: 1, Date: date, PayType: "worker.ngày thường.HC", Status: domain.TimesheetStatusApproved, PaymentStatus: domain.PaymentStatusPending},
		// A pending row sharing an approved key must remain untouched because the
		// incoming key is protected as a unit.
		{ID: 12, EmployeeID: 1, Date: date, PayType: "worker.ngày thường.HC", Status: domain.TimesheetStatusPendingApproval, PaymentStatus: domain.PaymentStatusPending},
		{ID: 21, EmployeeID: 2, Date: date, PayType: "worker.ngày thường.TCN", Status: domain.TimesheetStatusPendingApproval, PaymentStatus: domain.PaymentStatusPending},
		{ID: 41, EmployeeID: 4, Date: date, PayType: "worker.ngày thường.HC", Status: domain.TimesheetStatusRejected, PaymentStatus: domain.PaymentStatusPending},
		{ID: 51, EmployeeID: 5, Date: date, PayType: "worker.ngày thường.HC", Status: domain.TimesheetStatusPendingApproval, PaymentStatus: domain.PaymentStatusPending},
	}

	filtered, staleIDs, protectedSkipped, flexibleSkipped := planBCCReplacement(
		entries,
		existing,
		map[uint]struct{}{5: {}},
		true,
	)

	if len(filtered) != 2 || filtered[0].EmployeeID != 2 || filtered[1].EmployeeID != 3 {
		t.Fatalf("filtered entries = %#v, want pending employee 2 and missing employee 3", filtered)
	}
	if len(staleIDs) != 1 || staleIDs[0] != 21 {
		t.Fatalf("stale IDs = %v, want only pending timesheet 21", staleIDs)
	}
	if protectedSkipped != 2 {
		t.Fatalf("protected skipped = %d, want approved and rejected entries", protectedSkipped)
	}
	if flexibleSkipped != 1 {
		t.Fatalf("flexible skipped = %d, want 1", flexibleSkipped)
	}
}

func TestPlanBCCReplacementKeepsDifferentHourTypesIndependent(t *testing.T) {
	date := time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC)
	entries := []domainservices.BulkCreateTimesheetEntry{
		{EmployeeID: 1, Date: "2026-08-08", HourType: "HC"},
		{EmployeeID: 1, Date: "2026-08-08", HourType: "TCN"},
	}
	existing := []*domain.Timesheet{{
		ID: 11, EmployeeID: 1, Date: date, PayType: "worker.ngày thường.HC",
		Status: domain.TimesheetStatusApproved, PaymentStatus: domain.PaymentStatusPending,
	}}

	filtered, staleIDs, protectedSkipped, _ := planBCCReplacement(entries, existing, nil, true)

	if len(filtered) != 1 || filtered[0].HourType != "TCN" {
		t.Fatalf("filtered entries = %#v, want only TCN", filtered)
	}
	if len(staleIDs) != 0 || protectedSkipped != 1 {
		t.Fatalf("stale IDs = %v, protected skipped = %d", staleIDs, protectedSkipped)
	}
}

func TestPlanBCCReplacementProtectsEveryNonEditablePaymentState(t *testing.T) {
	date := time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC)
	for _, paymentStatus := range []domain.PaymentStatus{
		domain.PaymentStatusPaid,
		domain.PaymentStatusFailed,
		domain.PaymentStatusCancelled,
	} {
		t.Run(string(paymentStatus), func(t *testing.T) {
			entries := []domainservices.BulkCreateTimesheetEntry{{
				EmployeeID: 1, Date: "2026-08-08", HourType: "HC",
			}}
			existing := []*domain.Timesheet{{
				ID: 11, EmployeeID: 1, Date: date, PayType: "worker.ngày thường.HC",
				Status: domain.TimesheetStatusPendingApproval, PaymentStatus: paymentStatus,
			}}

			filtered, staleIDs, protectedSkipped, _ := planBCCReplacement(entries, existing, nil, true)
			if len(filtered) != 0 || len(staleIDs) != 0 || protectedSkipped != 1 {
				t.Fatalf("filtered=%v stale=%v protected=%d", filtered, staleIDs, protectedSkipped)
			}
		})
	}
}

func TestPlanBCCReplacementLegacyKeyProtectsWholeEmployeeDay(t *testing.T) {
	date := time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC)
	entries := []domainservices.BulkCreateTimesheetEntry{
		{EmployeeID: 1, Date: "2026-08-08", HourType: "HC"},
		{EmployeeID: 1, Date: "2026-08-08", HourType: "TCN"},
	}
	existing := []*domain.Timesheet{{
		ID: 11, EmployeeID: 1, Date: date, PayType: "worker.ngày thường.HC",
		Status: domain.TimesheetStatusApproved, PaymentStatus: domain.PaymentStatusPending,
	}}

	filtered, staleIDs, protectedSkipped, _ := planBCCReplacement(entries, existing, nil, false)
	if len(filtered) != 0 || len(staleIDs) != 0 || protectedSkipped != 2 {
		t.Fatalf("filtered=%v stale=%v protected=%d, want whole day protected", filtered, staleIDs, protectedSkipped)
	}
}

func TestCompleteSkippedBCCImportReportsSuccessfulNoOp(t *testing.T) {
	createdAt := clock.Now()
	asset := &domain.Asset{ID: 42, CreatedAt: createdAt}
	ctx := context.WithValue(context.Background(), deferBCCTerminalMetadataKey{}, true)

	result, err := (&BCCImportService{}).completeSkippedBCCImport(ctx, asset, 7, BCCImportStats{
		ProjectID:    74,
		OriginalName: "thai-binh-duong.xlsx",
		ForMonth:     "2026-08",
		TotalRows:    3,
		SkippedCount: 3,
	})
	if err != nil {
		t.Fatalf("completeSkippedBCCImport() error = %v", err)
	}
	if result.Status != domain.TimesheetImportStatusCompleted || result.CreatedCount != 0 ||
		result.ErrorCount != 0 || result.SkippedCount != 3 {
		t.Fatalf("result = %#v, want completed no-op with 3 skipped rows", result)
	}
}

func TestImportErrorsFromBulkFailuresKeepsEmployeeAndDate(t *testing.T) {
	errors := importErrorsFromBulkFailures([]domainservices.BulkCreateFailure{
		{
			Request: domainservices.BulkCreateTimesheetEntry{
				EmployeeID: 10,
				Date:       "2026-08-16",
			},
			Error: "Ngày không thể là ngày trong tương lai",
		},
	}, map[uint]string{10: "Nguyễn Văn An"})

	if len(errors) != 1 {
		t.Fatalf("error count = %d, want 1", len(errors))
	}
	if errors[0].Employee != "Nguyễn Văn An" {
		t.Errorf("employee = %q, want employee name", errors[0].Employee)
	}
	if errors[0].Reason != "ngày 2026-08-16: Ngày chấm công chưa đến" {
		t.Errorf("reason = %q", errors[0].Reason)
	}
}

func TestImportErrorsFromBulkFailuresUsesSafeFallback(t *testing.T) {
	errors := importErrorsFromBulkFailures([]domainservices.BulkCreateFailure{{
		Request: domainservices.BulkCreateTimesheetEntry{EmployeeID: 10, Date: "2026-08-16"},
		Error:   "Ngày không thể là ngày trong tương lai",
	}}, nil)

	if len(errors) != 1 || errors[0].Employee != "Nhân viên chưa xác định" {
		t.Fatalf("errors = %#v, want safe employee fallback", errors)
	}
}

func TestImportErrorsFromBulkFailuresDoesNotExposeInfrastructureError(t *testing.T) {
	errors := importErrorsFromBulkFailures([]domainservices.BulkCreateFailure{{
		Request: domainservices.BulkCreateTimesheetEntry{EmployeeID: 10, Date: "2026-08-16"},
		Error:   "Error 1062: duplicate key timesheets_unique",
	}}, map[uint]string{10: "Nguyễn Văn An"})

	if len(errors) != 1 || errors[0].Reason != "ngày 2026-08-16: Dữ liệu đã tồn tại" {
		t.Fatalf("errors = %#v, want user-safe duplicate error", errors)
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
