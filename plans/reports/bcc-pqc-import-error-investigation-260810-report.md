# BCC PQC Weekly Import Error Investigation Report

**Date:** 2026-08-10
**Project:** PQC Hải Phòng / PQC Nam Định
**File:** BCC LƯƠNG TUẦN KỲ 1 PQC HẢI PHÒNG THÁNG 08.2026.xlsx
**Month:** 2026-08

---

## Executive Summary

Investigated 1 failed row during BCC weekly import with error "Không thể xử lý dòng dữ liệu này". Root cause identified as **rate value mismatch between Excel file and payrate configuration** for the "Không có tay nghề" position.

---

## File Format Analysis

### Format Detection Result

**Format:** `FormatMultiPosition`

**Sheet Structure:**
- **Sheet Names:** "Có tay nghề", "Không có tay nghề", "Stk", "Hỗ trợ khác"
- These are position names (skill levels), NOT "BCC" format
- Correctly detected as multi-position format via `isPositionSheet()` check

### Excel File Layout

| Row | Content |
|-----|---------|
| 4   | Headers: STT, Mã nhân viên, Họ và tên, [Position], Ngày nghỉ, 26, 27, 28... |
| 5   | Rate values (43750, 60000, 87500 for skilled; 37500, 50000, 75000 for unskilled) |
| 6   | Shift codes: CB (regular), OT (overtime), CN (Sunday) |
| 7+  | Employee data rows |

### Rate Values by Position

**Có tay nghề (Skilled):**
| Day Type | Shift | Rate |
|----------|-------|------|
| Weekday  | CB    | 43,750 |
| Weekday  | OT    | 60,000 |
| Sunday   | CN    | 87,500 |

**Không có tay nghề (Unskilled):**
| Day Type | Shift | Rate |
|----------|-------|------|
| Weekday  | CB    | 37,500 |
| Weekday  | OT    | 50,000 |
| Sunday   | CN    | 75,000 |

---

## Root Cause Analysis

### Parser Behavior

The multi-position parser (`multi_position_parser.go`):
1. Reads row 4 for headers and day numbers ✓
2. Reads row 5 for rate values ✓
3. Reads row 7+ for employee data ✓
4. Creates entries with `RateVND` field from row 5 ✓

### Import Validation

The import service (`bcc_import_multi_position.go` line 378-384):
```go
target, ok := rateToTarget[entry.RateVND]
if !ok || entry.RateVND == 0 {
    importErrors = append(importErrors, domain.ImportError{
        Employee: emp.FullName,
        Reason:   fmt.Sprintf("không tìm thấy mức lương cho ngày %d (%d VND) ở vị trí %s", 
                             entry.DayNum, entry.RateVND, sheet.Position),
    })
    continue
}
```

### The Issue

**Rate Mismatch:** The Excel file contains rates that **do NOT exist** in the PQC project's payrate configuration:

| Excel Rate | Expected in Config | Status |
|------------|-------------------|--------|
| 37,500 VND | "không có tay nghề.ngày thường.giờ thường" | ❌ Missing or different value |
| 50,000 VND | "không có tay nghề.ngày thường.giờ tăng ca" | ❌ Missing or different value |
| 75,000 VND | "không có tay nghề.ngày nghỉ.giờ thường" | ❌ Missing or different value |
| 43,750 VND | "có tay nghề.ngày thường.giờ thường" | ❌ Missing or different value |
| 60,000 VND | "có tay nghề.ngày thường.giờ tăng ca" | ❌ Missing or different value |
| 87,500 VND | "có tay nghề.ngày nghỉ.giờ thường" | ❌ Missing or different value |

**Position Name Normalization:**
The system normalizes position names to lowercase for matching. The Excel sheet names need to match exactly (case-insensitive) with the payrate configuration position names.

---

## Failed Employee Details

**Employee:** Lưu Tuyết Nhung (CCCD: 031198007385)
**Sheet:** "Không có tay nghề"
**Row:** 8
**Entries:** 
- Day 3: 7.5 hours @ 37,500 VND
- Day 5: 8 hours @ 37,500 VND  
- Day 6: 8 hours @ 37,500 VND

**Issue:** Rate 37,500 VND not found in payrate configuration for "không có tay nghề" position.

---

## Solution Options

### Option 1: Update Payrate Configuration (Recommended) ✅

Add the missing rates to the PQC project's payrate configuration:

**For "Có tay nghề" position:**
- ngày thường.giờ thường: 43,750 VND
- ngày thường.giờ tăng ca: 60,000 VND
- ngày nghỉ.giờ thường: 87,500 VND

**For "Không có tay nghề" position:**
- ngày thường.giờ thường: 37,500 VND
- ngày thường.giờ tăng ca: 50,000 VND
- ngày nghỉ.giờ thường: 75,000 VND

### Option 2: Update Excel File

Change the rate values in the Excel file (row 5) to match existing payrate configuration values. This requires knowing the currently configured rates.

### Option 3: Verify Position Names

Ensure the payrate configuration uses the exact same position names as the Excel sheet names (case-insensitive):
- Excel: "Có tay nghề", "Không có tay nghề"
- Config should use: "có tay nghề", "không có tay nghề" (lowercase for matching)

---

## Investigation Steps Taken

1. ✅ Analyzed Excel file structure (rows 1-15)
2. ✅ Identified format detection (FormatMultiPosition)
3. ✅ Extracted rate values from row 5 for both positions
4. ✅ Traced parser logic (multi_position_parser.go)
5. ✅ Traced validation logic (bcc_import_multi_position.go)
6. ✅ Identified rate lookup validation failure point
7. ❌ Could not verify production payrate config (SSH tunnel not available)

---

## Next Steps

1. **Verify PQC payrate configuration** in production database:
   ```sql
   SELECT position, day_type, hour_type, rate 
   FROM payrates 
   WHERE project_id = (SELECT id FROM projects WHERE name LIKE '%PQC%')
   ORDER BY position, day_type, hour_type;
   ```

2. **Compare** the configured rates with Excel file rates

3. **Update** either:
   - The payrate configuration (if Excel rates are correct), OR
   - The Excel file (if configuration is correct)

4. **Re-import** the corrected file

---

## Files Analyzed

- `/Users/dev/Downloads/BCC+LƯƠNG+TUẦN+KỲ+1+PQC+HẢI+PHÒNG+THÁNG+08.2026.xlsx` - The Excel file
- `backend/internal/app/services/excel/format_detector.go` - Format detection
- `backend/internal/app/services/excel/multi_position_parser.go` - Multi-position parser
- `backend/internal/app/services/bcc_import_multi_position.go` - Import service

---

## Technical Notes

### Multi-Position Format Detection

The file is correctly detected as `FormatMultiPosition` because:
1. No "BCC" sheet found
2. No "BCC-" prefix sheets found
3. `isPositionSheet()` succeeds: Row 4 has STT + Mã nhân viên + Họ và tên
4. Sheet names are used as position values

### Rate Matching Logic

The `rateToTarget` map is built from flattened payrate paths:
- Format: `{position}.{day_type}.{hour_type}` → rate value
- Example: `"có tay nghề.ngày thường.giờ thường"` → 43750

The parser looks up each entry's `RateVND` value in this map. If not found → validation error.

### Position Name Normalization

Position names are normalized to lowercase for matching:
- Excel: "Có tay nghề" → normalized: "có tay nghề"
- Config: Must use "có tay nghề" (or any case that lowercases to this)

---

## Root Cause (Updated)

**Day Type Name Mismatch:**

The payrate configuration uses short day type names:
- `Thường` (Regular)
- `Nghỉ` (Rest)
- `Lễ` (Holiday)

But the code expected full names:
- `ngày thường`
- `ngày nghỉ`
- `ngày lễ`

This caused ALL rates to be skipped when building the `rateToTarget` map because the code only recognized the full names.

---

## Solution Implemented

**Code changes made to accept both formats:**

### 1. Updated `bcc_import_multi_position.go`

**Line 290-296:** Extended `dayTypePriority` map to include both formats:
```go
dayTypePriority := map[string]int{
    "ngày thường": 0, "thường": 0,
    "ngày nghỉ": 1, "nghỉ": 1,
    "ngày lễ": 2, "lễ": 2,
}
```

**Added `normalizeDayType()` function** (line ~580): Converts short names to full names for consistent storage.

**Updated timesheet entry creation** (line ~398): Uses normalized day type when storing to database.

### 2. Updated `bcc_import_service.go` (Legacy Import)

**Line 549-556:** Extended `dayTypePriority` map with same fix.

**Added `normalizeDayType()` function** (line ~1092): Same normalization logic.

**Updated timesheet entry creation** (line ~867): Uses normalized day type.

---

## Testing Required

1. Re-import the PQC file with the updated code
2. Verify day types are stored as full names ("ngày thường", "ngày nghỉ", "ngày lễ")
3. Test with existing payrate configs that use both formats

---

## Files Modified

1. `backend/internal/app/services/bcc_import_multi_position.go`
   - Updated dayTypePriority map (line 290-296)
   - Added normalizeDayType function (~line 580)
   - Updated timesheet entry creation (line ~398)

2. `backend/internal/app/services/bcc_import_service.go`
   - Updated dayTypePriority map (line 549-556)
   - Added normalizeDayType function (~line 1092)
   - Updated timesheet entry creation (line ~867)

---

**Status:** ✅ Fix implemented, ready for testing

**Note:** The fix maintains backward compatibility - configs using full names ("ngày thường") continue to work as before.
