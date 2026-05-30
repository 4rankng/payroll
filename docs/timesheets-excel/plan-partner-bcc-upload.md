# Plan: Partner BCC Excel Upload → Auto-Create Timesheets + Upload History

## Overview

Partners upload a monthly attendance Excel file (BCC format, e.g. "BCC LƯƠNG DỰ ÁN EVA THÁNG 05.2026.xlsx"). The system parses it, auto-creates timesheet entries, stores every uploaded file, and lets anyone with access review/download the upload history.

---

## 1. Understanding the BCC File Format

From analysing the uploaded file the BCC sheet has this structure:

| Rows | Content |
|------|---------|
| 1–3 | Company header (name, tax code, address) |
| 5–6 | Title "BẢNG CHẤM CÔNG THÁNG" + month (e.g. "Tháng 05/2026") |
| 8 | Column headers: STT, Mã nhân viên, CCCD, Họ và tên, Ngày vào, Ngày nghỉ, Bộ phận, then Day 1…Day 31 (each spanning 4 sub-columns), then summary columns |
| 9 | Day-of-week labels (T2…CN) |
| 10 | Per-shift hourly rates (e.g. 36000 for CB N, 54000 for OT N…) |
| 11 | Sub-column type labels per day: **CB N** (regular day shift), **OT N** (OT day shift), **CB Đ** (regular night shift), **OT Đ** (OT night shift) |
| 12+ | One row per employee: employee code, CCCD, name, start date, end date, department, then hours for each day×shift cell |

Key mappings to the existing domain:
- `CB N` → `HourType = "Ca ngày"`, `DayType = "Ngày thường"`
- `OT N` → `HourType = "Ca tăng ca"`, `DayType = "Ngày thường"`  
- `CB Đ` → `HourType = "Ca đêm"`, `DayType = "Ngày thường"`
- `OT Đ` → `HourType = "Ca tăng ca đêm"`, `DayType = "Ngày thường"`
- Sunday column (CN) → `DayType = "Ngày nghỉ"`
- Saturday column (T7) → `DayType = "Ngày nghỉ"`  
- Regular weekdays (T2-T6) → `DayType = "Ngày thường"`

Employee matching strategy: match by **CCCD** (column C, "Citizen ID") against `users.cccd`. Fall back to employee code (`Mã nhân viên`, column B) if CCCD is blank. Employees not found in the system → logged as warnings, not hard failures.

Month/year is extracted from row 6: "Tháng MM/YYYY". Day numbers in row 8 are Excel date serials (day-of-month offset from 1900-01-01); treat them as 1-based day numbers for the parsed month.

---

## 2. New Domain Entity: `PartnerImportFile`

A new table stores every BCC file a partner uploads, linking it to the stored asset and recording the import result.

```go
// domain/partner_import_file.go
type PartnerImportFile struct {
    ID              uint       `gorm:"primarykey;type:bigint unsigned"`
    AssetID         uint       `gorm:"not null;type:bigint unsigned;index"` // FK → assets
    ProjectID       uint       `gorm:"not null;type:bigint unsigned;index"` // which project
    UploadedBy      uint       `gorm:"not null;type:bigint unsigned"`        // partner user ID
    OriginalName    string     `gorm:"type:varchar(255);not null"`          // original filename
    ForMonth        string     `gorm:"type:char(7);not null"`               // "2026-05"
    Status          string     `gorm:"type:enum('pending','processing','completed','failed');not null;default:'pending'"`
    TotalRows       int        `gorm:"not null;default:0"`
    CreatedCount    int        `gorm:"not null;default:0"`
    SkippedCount    int        `gorm:"not null;default:0"`
    ErrorCount      int        `gorm:"not null;default:0"`
    ErrorDetail     *string    `gorm:"type:json"`                           // JSON array of {row, employee, error}
    ProcessedAt     *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time

    // Relationships
    Asset    Asset `gorm:"foreignKey:AssetID"`
    Uploader User  `gorm:"foreignKey:UploadedBy"`
}

func (PartnerImportFile) TableName() string { return "partner_import_files" }
```

Repository interface mirrors the pattern of `BulkTransferFileRepository`:

```go
type PartnerImportFileRepository interface {
    Create(ctx, *PartnerImportFile) error
    GetByID(ctx, id uint) (*PartnerImportFile, error)
    UpdateStatus(ctx, id uint, status string, result *ImportResult) error
    List(ctx, filters PartnerImportFileFilters) ([]*PartnerImportFile, int64, error)
}
```

---

## 3. New Asset Upload Type

Add a constant in `domain/asset.go`:

```go
UploadTypePartnerBCCImport = "partner_bcc_import"
```

---

## 4. BCC Excel Parser (`app/services/excel/bcc_parser.go`)

New file, separate from the existing `timesheet_import_service.go` (which handles the internal template format). Key functions:

```go
// BCCImportData is the result of parsing one BCC file
type BCCImportData struct {
    ForMonth  string               // "2026-05"
    Employees []BCCEmployeeData
}

type BCCEmployeeData struct {
    EmployeeCode string
    CCCD         string
    FullName     string
    Department   string
    Entries      []BCCEntryData
}

type BCCEntryData struct {
    Date    time.Time         // full date in the month
    Shifts  map[string]float64 // shift label → hours worked
    DayType string             // "Ngày thường" | "Ngày nghỉ"
}

func ParseBCCFile(f *excelize.File) (*BCCImportData, error)
```

**Parser logic (step by step):**

1. Open sheet `BCC` (sheet index 0, or first sheet).
2. Find row 6 → extract month string → parse to `time.Month` and year.
3. Scan row 8 for day columns: each non-null cell is a day number (1–31). Record the column index and build a `date = time.Date(year, month, day, 0,0,0,0, loc)`.
4. Scan row 9 for the day-of-week label at each day's starting column → determine `DayType` (T7/CN → "Ngày nghỉ", else "Ngày thường").
5. Scan row 11 for the 4 sub-column labels per day (CB N / OT N / CB Đ / OT Đ) → build a map `colIndex → shiftLabel`.
6. For each data row (row ≥ 12) where col A is a number (STT):
   - Read CCCD (col C), employee code (col B), name (col D), department (col G).
   - For each day, read the 4 shift columns → accumulate non-zero hours as `BCCEntryData`.
7. Return `BCCImportData`.

---

## 5. Import Service (`app/services/bcc_import_service.go`)

```go
type BCCImportService struct {
    employeeRepo      domain.EmployeeRepository
    timesheetService  domain.TimesheetService
    importFileRepo    domain.PartnerImportFileRepository
    assetRepo         domain.AssetRepository
    assetStorage      AssetStorageService
}

func (s *BCCImportService) ProcessUpload(
    ctx context.Context,
    file multipart.File,
    header *multipart.FileHeader,
    projectID uint,
    uploaderID uint,
    uploaderRole string,
) (*domain.PartnerImportFile, error)
```

**Steps inside `ProcessUpload`:**

1. Save the raw file to storage (same `AssetStorageService` used for other assets) → get `asset.ID`.
2. Create a `PartnerImportFile` record with `status = "processing"` → persist to DB → get `importFile.ID`.
3. Call `ParseBCCFile(excelFile)` → on parse error → update record to `status = "failed"`, return error.
4. For each `BCCEmployeeData`:
   a. Look up employee by CCCD (then by code fallback) → if not found, add to error list, continue.
   b. Validate partner has access to that employee (reuse existing `employeePermissionService`).
   c. Build `[]BulkCreateTimesheetEntry` from the entries.
5. Call `timesheetService.BulkCreateTimesheets(ctx, entries, uploaderID, uploaderRole)` for the whole batch.
6. Update `PartnerImportFile` with final counts and `status = "completed"` (or `"failed"` if zero rows created).
7. Return the `PartnerImportFile`.

The service runs **synchronously** (same request). Because BCC files are monthly and typically 20–100 rows, response time is acceptable (<5 s). If future files grow larger, the service can be extracted to a background worker without changing the API contract.

---

## 6. HTTP Handler (`transport/http/handlers/timesheet/bcc_import_handler.go`)

### `POST /api/v1/timesheets/partner-import`

Accepts: `multipart/form-data` with fields:
- `file` – the BCC `.xlsx` file
- `project_id` – which project these timesheets belong to

Auth: requires `partner` or `adv_partner` role (or admin). Casbin rule: `partner` can POST to this endpoint.

```go
func (h *Handler) UploadPartnerBCC(c *gin.Context) {
    userID, userRole, ok := h.getUserContext(c)
    // parse project_id from form
    // parse file from form
    // call bccImportService.ProcessUpload(...)
    // return PartnerImportFile summary + any error rows
}
```

Response body:

```json
{
  "data": {
    "import_id": 42,
    "for_month": "2026-05",
    "status": "completed",
    "total_rows": 35,
    "created_count": 280,
    "skipped_count": 12,
    "error_count": 3,
    "errors": [
      { "row": 14, "employee": "Nguyễn Văn X", "reason": "Employee not found in system" }
    ]
  },
  "message": "Import thành công: 280 timesheet tạo mới, 12 bỏ qua, 3 lỗi"
}
```

### `GET /api/v1/timesheets/partner-import`

List upload history. Partners see only their own uploads; admins see all.

Query params: `project_id`, `from_date`, `to_date`, `page`, `page_size`.

Response: paginated list of `PartnerImportFile` summaries (id, original_name, for_month, status, counts, uploaded_by, created_at) + a pre-signed download URL for each asset.

### `GET /api/v1/timesheets/partner-import/:id`

Get single import detail including the full `error_detail` JSON array.

### `GET /api/v1/timesheets/partner-import/:id/download`

Redirects (302) or returns a pre-signed URL to download the original Excel file from asset storage. Uses the same asset-download mechanism as existing bulk transfer files.

---

## 7. Database Migration (`migrations/065_partner_import_files.up.sql`)

```sql
CREATE TABLE partner_import_files (
    id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    asset_id         BIGINT UNSIGNED NOT NULL,
    project_id       BIGINT UNSIGNED NOT NULL,
    uploaded_by      BIGINT UNSIGNED NOT NULL,
    original_name    VARCHAR(255) NOT NULL,
    for_month        CHAR(7) NOT NULL,
    status           ENUM('pending','processing','completed','failed') NOT NULL DEFAULT 'pending',
    total_rows       INT NOT NULL DEFAULT 0,
    created_count    INT NOT NULL DEFAULT 0,
    skipped_count    INT NOT NULL DEFAULT 0,
    error_count      INT NOT NULL DEFAULT 0,
    error_detail     JSON,
    processed_at     TIMESTAMP NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_pif_project (project_id),
    INDEX idx_pif_uploaded_by (uploaded_by),
    INDEX idx_pif_asset (asset_id),
    CONSTRAINT fk_pif_asset    FOREIGN KEY (asset_id)    REFERENCES assets(id),
    CONSTRAINT fk_pif_uploader FOREIGN KEY (uploaded_by) REFERENCES users(id),
    CONSTRAINT fk_pif_project  FOREIGN KEY (project_id)  REFERENCES projects(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 8. Frontend

### 8a. Upload Modal (existing Timesheets page or new Partner Imports page)

A new button **"Tải lên BCC"** (visible only to partner/admin roles). Opens a modal:

```
┌─────────────────────────────────────────────────────┐
│  Tải lên Bảng Chấm Công (BCC)                       │
│                                                     │
│  Dự án:  [ Dropdown chọn dự án ]                   │
│                                                     │
│  File:   [ Drag & drop / Browse .xlsx ]            │
│                                                     │
│  [ Hủy ]                    [ Tải lên & Xử lý ]   │
└─────────────────────────────────────────────────────┘
```

On success, show a summary card:
- ✅ Created: 280 timesheets
- ⏭ Skipped: 12 (duplicates)
- ❌ Errors: 3 rows (expandable list)

### 8b. Upload History Page/Tab (`/partner-imports` or tab inside Timesheets)

A table with columns:

| # | File name | Project | Month | Uploaded by | Status | Created | Skipped | Errors | Actions |
|---|-----------|---------|-------|-------------|--------|---------|---------|--------|---------|
| 1 | BCC LƯƠNG DỰ ÁN EVA… | EVA | 05/2026 | Nguyễn A | ✅ completed | 280 | 12 | 3 | 👁 Detail · ⬇ Download |

Clicking **Detail** opens a side-panel with the full error list. Clicking **Download** fetches `/partner-import/:id/download` and triggers browser download of the original file.

Filter bar: date range picker + project selector.

---

## 9. Casbin / Authorization Rules

Add to the Casbin policy (or the equivalent DB seed):

```
p, partner,   /api/v1/timesheets/partner-import,     POST
p, partner,   /api/v1/timesheets/partner-import,     GET
p, partner,   /api/v1/timesheets/partner-import/*,   GET
p, adv_partner, /api/v1/timesheets/partner-import,   POST
p, adv_partner, /api/v1/timesheets/partner-import,   GET
p, adv_partner, /api/v1/timesheets/partner-import/*, GET
```

Partners must also already be assigned to the project (`project_users` table) for the import to be accepted — validated inside the service using the existing `projectPermissionService`.

---

## 10. Integration Tests (`tests/integration/partner_bcc_import_test.go`)

Scenarios to cover:

1. **Happy path** – upload a valid BCC file, all employees exist, all timesheets created.
2. **Partial match** – some CCCDs not in system → those rows error, others succeed.
3. **Duplicate prevention** – re-upload same month → existing timesheets skipped (not duplicated).
4. **Wrong file format** – upload a non-BCC xlsx (e.g., internal template) → parser returns error, `status = failed`.
5. **No project_id / invalid project_id** → 400.
6. **Partner without project access** → 403.
7. **Download endpoint** – verify original file bytes match the uploaded file.
8. **History list** – verify pagination, filtering by month, and that partners only see their own uploads.

---

## 11. Implementation Order

1. **Migration** – add `partner_import_files` table.
2. **Domain** – `PartnerImportFile` struct + `PartnerImportFileRepository` interface.
3. **Infra** – GORM repository implementation in `infra/`.
4. **Parser** – `app/services/excel/bcc_parser.go` + unit tests.
5. **Import service** – `app/services/bcc_import_service.go`.
6. **Wire up DI** – register new repo + service in `app/bootstrap/`.
7. **HTTP handlers** – 4 endpoints above + router registration.
8. **Casbin rules** – add policy entries.
9. **Integration tests** – `tests/integration/partner_bcc_import_test.go`.
10. **Frontend** – Upload modal + History table.
11. **Run `make api-test`** to guard against regression.
