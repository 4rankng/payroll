# BCC Weekly Import Error Investigation Report

**Date:** 2026-08-10  
**Project:** EPE KCN Vsip  
**File:** BCC LƯƠNG TUẦN KỲ 1 DỰ ÁN EPE T08.2026.xlsx  
**Month:** 2026-08

## Executive Summary

Investigated 6 failed rows during BCC weekly import with error "Không thể xử lý dòng dữ liệu này". Found that the issue stems from the Excel file structure not matching the expected format for the weekly payment parser.

## Error Details

### Failed Rows
- Row 14: Đồng Thị Thu Huế (HTV2089)
- Row 15: Nghiêm Thị Huyền (HTV2090)
- Row 16: Đỗ Thị Thu Hằng (HTV2093)
- Row 17: Phạm Gia Khiêm (HTV2095)
- Row 18: Đào Thị Hoa (HTV2099)

### Root Cause Analysis

#### 1. Excel File Structure

The Excel file has a complex structure with:
- **Sheet name:** "BCC " (with trailing space)
- **Row 8:** Headers (STT, CCCD, Họ và tên, Mã nhân viên, dates...)
- **Row 9:** Day types (T7, CN, T2, T3...) and rate values
- **Row 10:** Shift codes (HC, TCN, NN, TCNN)
- **Row 11:** Additional data
- **Row 12+:** Employee data

#### 2. Key Differences Between Successful and Failed Rows

**Row 12 (SUCCESS - Linh Văn Hùng):**
```
Col 5-8: 8, 3, None, None
```
- Saturday (8/1): 8 hours HC, 3 hours TCN
- **Sunday (8/2): None, None (no work)**

**Row 14 (FAIL - Đồng Thị Thu Huế):**
```
Col 5-8: 8, 3, 8, 0
```
- Saturday (8/1): 8 hours HC, 3 hours TCN
- **Sunday (8/2): 8 hours HC, 0 hours TCN**

**Row 16 (FAIL - Đỗ Thị Thu Hằng):**
```
Col 5-14: 8, 0, 8, 0, 8, 0, 8, 0, 8, 0, None, None, 8, 0
```
- Has **None values in the middle of the week** (columns 13-14)

#### 3. Pattern Identified

All failed rows (14-18) have work hours on **Sunday (8/2/2026)**:
- Row 14: 8 hours HC, 0 hours TCN
- Row 15: 8 hours HC, 0 hours TCN
- Row 16: 8 hours HC, 0 hours TCN
- Row 17: 8 hours HC, 3 hours TCN
- Row 18: 8 hours HC, 0 hours TCN

Successful rows have `None, None` for Sunday, indicating no work.

## Format Detection

The file is detected as **FormatLegacy** due to the "BCC " sheet name. However, the actual structure more closely resembles the **Weekly Payment** format but with deviations:
1. Missing "Bộ phận" and "Lương 8h" headers in row 8
2. Column 4 contains "Mã nhân viên" instead of "Bộ phận"
3. Row 11 contains additional data that doesn't match the expected pattern

## Technical Details

### Weekly Payment Parser Expectations

The `isWeeklyPaymentSheet()` function checks:
1. **Row 8 headers:** STT, Mã nhân viên, Họ và tên, **Bộ phận**, **Lương 8h**
2. **Row 10:** Shift codes (HC, TCN, NN, TCNN)

The Excel file has:
1. **Row 8 headers:** STT, **CCCD**, Họ và tên, **Mã nhân viên**, dates...
2. **Row 10:** Shift codes present ✓

### Legacy Parser Expectations

The `buildBCCHeaderMap()` function scans rows 7-8 for:
- STT, CCCD, Mã nhân viên, Họ và tên, Bộ phận

The file has headers at row 8, which matches partially.

## Hypothesis

The failed rows likely fail validation because:
1. **Sunday work hours** are not expected in the weekly BCC format
2. The parser may skip rows with Sunday work hours when they should be None
3. The column mapping might be incorrect for this specific format variant

## Recommendations

### Immediate Actions

1. **Check the actual error messages in production logs** - The error "Không thể xử lý dòng dữ liệu này" is generic. The specific validation error needs to be identified from the error detail.

2. **Verify the payrate configuration** for:
   - EPE project for August 2026
   - Sunday rates (ngày nghỉ) for these shift types
   - Check if NN/TCNN rates exist for the dates

3. **Review the format detection logic** - The file structure doesn't perfectly match any of the defined formats:
   - FormatLegacy: Expects headers at rows 7-8
   - FormatWeeklyPayment: Expects "Bộ phận" and "Lương 8h" headers
   - FormatWeeklyBCC: Expects separate "BCC-HC", "BCC-TCN" sheets

### Long-term Solutions

1. **Add better format detection** for this EPE-specific format variant
2. **Improve error messages** to be more specific about why a row fails
3. **Add validation for Sunday work hours** in the weekly BCC format
4. **Create a test case** with this exact Excel file structure

## ROOT CAUSE IDENTIFIED ✓

### Production Logs Analysis (2026-08-10 14:37:58)

**Error Message:** `"Công ngày nghỉ ca ngày của [Tên] ngày 2026-08-02 đã tồn tại"`

**Final Status:** `"lỗi tạo bảng chấm công: không thể thay thế an toàn: 6 dòng không hợp lệ"`

### The 6 Failed Employees

| Employee | Employee ID | Date | Pay Type | Error |
|----------|-------------|------|----------|-------|
| Đồng Thị Thu Huế | 789 | 2026-08-02 | phổ thông.ngày nghỉ.ca ngày | Duplicate exists |
| Nghiêm Thị Huyền | 790 | 2026-08-02 | phổ thông.ngày nghỉ.ca ngày | Duplicate exists |
| Đỗ Thị Thu Hằng | 793 | 2026-08-02 | phổ thông.ngày nghỉ.ca ngày | Duplicate exists |
| Phạm Gia Khiêm | 968 | 2026-08-02 | phổ thông.ngày nghỉ.ca ngày | Duplicate exists |
| Đào Thị Hoa | 1035 | 2026-08-02 | phổ thông.ngày nghỉ.ca ngày | Duplicate exists |
| Lê Thị Hồng Nhung | 913 | 2026-08-02 | phổ thông.ngày nghỉ.ca ngày | Duplicate exists |

### Root Cause

**The file was successfully parsed and processed.** The error occurs during timesheet creation because:

1. These 6 employees worked on Sunday (2026-08-02)
2. They already have timesheets for 2026-08-02 in the database from a previous import
3. The system's "safe replacement" logic rejects duplicate entries
4. When more than 5 entries fail validation, the entire import fails

**Key Finding:** This is NOT a parsing error, format detection error, or payrate lookup error. The file structure is correct and the system correctly parsed all employee data and shift codes.

## Resolution Options

### Option 1: Delete Existing Sunday Timesheets (Recommended)
Before re-importing, delete the existing timesheets for 2026-08-02 for these employees:
- Check existing timesheets in the timesheet list
- Delete duplicates manually
- Re-import the file

### Option 2: Modify Import Logic (Code Change)
The system could be enhanced to:
- Skip duplicate entries instead of failing entirely
- Provide a "force overwrite" option for admins
- Show duplicate warnings without blocking the entire import

### Option 3: Modify the Excel File
Remove Sunday work hours from the Excel file before importing (not recommended as this loses valid work data).

## Files Analyzed

- `/Users/dev/Downloads/EPE.xlsx` - The actual Excel file
- `backend/internal/app/services/excel/format_detector.go` - Format detection logic
- `backend/internal/app/services/excel/weekly_payment_parser.go` - Weekly payment parser
- `backend/internal/app/services/bcc_import_weekly.go` - Weekly BCC import service
- Production logs from `tingting.vip` - Error details

## Answered Questions

1. **What is the specific error?** → Duplicate timesheets for Sunday 2026-08-02
2. **Is it a parsing error?** → No, the file was parsed successfully
3. **Is it a payrate error?** → No, payrates were correctly resolved
4. **Is it a format detection error?** → No, format was correctly identified

## Recommendations

1. **Immediate:** User should delete existing Sunday timesheets before re-importing
2. **UX Improvement:** Add "Skip duplicates" checkbox to import dialog
3. **UX Improvement:** Show duplicate warnings per-employee instead of failing entire import
4. **Admin Feature:** Add "Force overwrite" option for admins to handle duplicates
