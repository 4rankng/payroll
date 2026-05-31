package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	excelparser "api-server/internal/app/services/excel"
	timesheetSvc "api-server/internal/app/services/timesheet"
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"
	"api-server/internal/infra/storage"
	"api-server/internal/pkg/clock"

	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// BCCImportStats holds common import statistics shared between service result,
// stored metadata, and handler DTOs.
type BCCImportStats struct {
	ProjectID    uint       `json:"project_id"`
	OriginalName string     `json:"original_name"`
	ForMonth     string     `json:"for_month"`
	Status       string     `json:"status"`
	TotalRows    int        `json:"total_rows"`
	CreatedCount int        `json:"created_count"`
	SkippedCount int        `json:"skipped_count"`
	ErrorCount   int        `json:"error_count"`
	ErrorDetail  *string    `json:"error_detail,omitempty"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
}

// BCCImportResult is returned by ProcessUpload and serialized to the API response.
type BCCImportResult struct {
	BCCImportStats
	ID         uint      `json:"id"`
	UploadedBy uint      `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// bccImportMetadata is stored in the Asset.Metadata JSON column.
type bccImportMetadata = BCCImportStats

// buildResult creates a BCCImportResult from stats and asset fields.
func buildResult(stats BCCImportStats, id, uploaderID uint, createdAt time.Time) *BCCImportResult {
	return &BCCImportResult{
		BCCImportStats: stats,
		ID:             id,
		UploadedBy:     uploaderID,
		CreatedAt:      createdAt,
	}
}

type BCCImportService struct {
	projectEmpRepo   domain.ProjectEmployeeRepository
	payrateRepo      domain.PayrateRepository
	timesheetRepo    domain.TimesheetRepository
	timesheetService *timesheetSvc.TimesheetService
	assetRepo        domain.AssetRepository
	fileStorage      storage.FileStorage
	db               *gorm.DB
	redis            *redis.Client
}

func NewBCCImportService(
	projectEmpRepo domain.ProjectEmployeeRepository,
	payrateRepo domain.PayrateRepository,
	timesheetRepo domain.TimesheetRepository,
	timesheetService *timesheetSvc.TimesheetService,
	assetRepo domain.AssetRepository,
	fileStorage storage.FileStorage,
	db *gorm.DB,
	redisClient *redis.Client,
) *BCCImportService {
	return &BCCImportService{
		projectEmpRepo:   projectEmpRepo,
		payrateRepo:      payrateRepo,
		timesheetRepo:    timesheetRepo,
		timesheetService: timesheetService,
		assetRepo:        assetRepo,
		fileStorage:      fileStorage,
		db:               db,
		redis:            redisClient,
	}
}

// ProcessUpload saves the BCC file, parses it, and creates timesheets.
// forMonth is "YYYY-MM" from the frontend; day numbers in the Excel belong to this month.
// Conflict policy: latest upload wins — unapproved entries are overwritten, approved/paid are protected.
func (s *BCCImportService) ProcessUpload(
	ctx context.Context,
	fileData io.Reader,
	filename string,
	projectID uint,
	uploaderID uint,
	uploaderRole string,
	forMonth string,
) (*BCCImportResult, error) {
	// 1. Read file bytes (needed for both storage and parsing).
	const maxUploadSize = 10 << 20
	data, err := io.ReadAll(io.LimitReader(fileData, maxUploadSize+1))
	if err != nil {
		return nil, fmt.Errorf("BCCImportService.ProcessUpload: read file: %w", err)
	}
	if len(data) > maxUploadSize {
		return nil, fmt.Errorf("BCCImportService.ProcessUpload: file quá lớn (tối đa 10MB)")
	}

	// 2. Save raw file to storage.
	stored, err := s.fileStorage.StoreBytes(data, filename, domain.UploadTypePartnerBCCImport)
	if err != nil {
		return nil, fmt.Errorf("BCCImportService.ProcessUpload: store file: %w", err)
	}

	// 3. Create asset record. On DB failure, clean up the orphaned file.
	var createdAsset *domain.Asset
	asset := &domain.Asset{
		Filename:   stored.OriginalFilename,
		FilePath:   stored.FilePath,
		UploadType: domain.UploadTypePartnerBCCImport,
		UploadedBy: uploaderID,
	}
	createdAsset, err = s.assetRepo.Create(ctx, asset)
	if err != nil {
		_ = s.fileStorage.Delete(stored.FilePath)
		return nil, fmt.Errorf("BCCImportService.ProcessUpload: create asset: %w", err)
	}

	// effectiveMonth is captured by the fail() closure; set before first use of fail()
	// to ensure early failures still include the month when available.
	var effectiveMonth string

	// Helper to update asset metadata and return result.
	fail := func(status, reason string) (*BCCImportResult, error) {
		errs := []domain.ImportError{{Reason: reason}}
		detail := marshalErrors(errs)
		now := clock.Now()
		stats := BCCImportStats{
			ProjectID:    projectID,
			OriginalName: filename,
			ForMonth:     effectiveMonth,
			Status:       status,
			ErrorCount:   1,
			ErrorDetail:  detail,
			ProcessedAt:  &now,
		}
		if metaErr := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); metaErr != nil {
			slog.Error("BCCImport: metadata update failed in fail path", "asset_id", createdAsset.ID, "error", metaErr)
		}
		return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt),
			fmt.Errorf("import failed: %s", FirstErrorReason(detail))
	}

	// 4. Parse the Excel file.
	xf, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return fail("failed", fmt.Sprintf("không thể mở file Excel: %v", err))
	}
	defer func() { _ = xf.Close() }()

	parsed, err := excelparser.ParseBCCFile(xf)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi phân tích file BCC: %v", err))
	}

	// 5. Use the frontend-supplied month.
	effectiveMonth = forMonth
	loc := clock.Now().Location()
	year, month, err := parseForMonth(effectiveMonth)
	if err != nil {
		return fail("failed", fmt.Sprintf("tháng không hợp lệ: %v", err))
	}

	// Acquire distributed lock to prevent concurrent imports for same project+month.
	lockValue, lockErr := s.acquireImportLock(ctx, projectID, effectiveMonth)
	if lockErr != nil {
		return fail("failed", lockErr.Error())
	}
	defer s.releaseImportLock(ctx, projectID, effectiveMonth, lockValue)

	// 6. Get active payrate for the project.
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, projectID, monthStart)
	if err != nil {
		return fail("failed", fmt.Sprintf("không tìm thấy bảng lương cho dự án: %v", err))
	}
	flatRates, err := payrate.Payrate.Flatten()
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi phân tích cấu hình lương: %v", err))
	}

	// Build rate -> (dayType, hourType) lookup from the project payrate.
	// Rate is king: as long as the rate matches, the entry is valid.
	// On collision (same rate, multiple paths), prefer "ngay thuong".
	dayTypePriority := map[string]int{"ngày thường": 0, "ngày nghỉ": 1, "ngày lễ": 2}
	type rateTarget struct{ dayType, hourType string }
	rateToTarget := make(map[int]rateTarget)
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
			continue // skip unrecognized day types
		}
		existing, exists := rateToTarget[rate]
		if !exists || candPri < dayTypePriority[existing.dayType] {
			rateToTarget[rate] = candidate
		}
	}

	// 7. Load all active project employees, build lookup maps.
	assignments, err := s.projectEmpRepo.GetActiveAssignments(ctx, projectID)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi tải danh sách nhân viên: %v", err))
	}
	byCCCD := make(map[string]*domain.ProjectEmployee, len(assignments))
	byCode := make(map[string]*domain.ProjectEmployee, len(assignments))
	empNames := make(map[uint]string, len(assignments))
	for _, a := range assignments {
		byCCCD[a.EmployeeCCCD] = a
		if a.EmployeeCode != "" {
			byCode[a.EmployeeCode] = a
		}
		empNames[a.EmployeeID] = a.EmployeeName
	}

	// 8. Build timesheet entries, collecting employee errors.
	var (
		importErrors []domain.ImportError
		entries      []domainservices.BulkCreateTimesheetEntry
		rowNum       = 12
	)

	for _, emp := range parsed.Employees {
		assignment := byCCCD[emp.CCCD]
		if assignment == nil && emp.EmployeeCode != "" {
			assignment = byCode[emp.EmployeeCode]
		}
		if assignment == nil {
			importErrors = append(importErrors, domain.ImportError{
				Row:      rowNum,
				Employee: emp.FullName,
				Reason:   "nhân viên không tìm thấy trong hệ thống",
			})
			rowNum++
			continue
		}

		if assignment.PaymentSchedule == string(domain.PaymentScheduleFlexible) {
			importErrors = append(importErrors, domain.ImportError{
				Row:      rowNum,
				Employee: emp.FullName,
				Reason:   "nhân viên lương linh hoạt không áp dụng BCC import",
			})
			rowNum++
			continue
		}

		for _, entry := range emp.Entries {
			date := time.Date(year, month, entry.DayNum, 0, 0, 0, 0, loc)
			if date.Month() != month {
				continue
			}

			rate := int(parsed.ShiftRates[entry.ShiftLabel])
			target, ok := rateToTarget[rate]
			if !ok {
				importErrors = append(importErrors, domain.ImportError{
					Row:      rowNum,
					Employee: emp.FullName,
					Reason:   fmt.Sprintf("không tìm thấy mức lương cho ca %s (%d VND)", entry.ShiftLabel, parsed.ShiftRates[entry.ShiftLabel]),
				})
				continue
			}

			entries = append(entries, domainservices.BulkCreateTimesheetEntry{
				ProjectID:   projectID,
				EmployeeID:  assignment.EmployeeID,
				Date:        date.Format("2006-01-02"),
				HoursWorked: entry.Hours,
				HourType:    target.hourType,
				DayType:     &target.dayType,
			})
		}
		rowNum++
	}

	totalRows := len(parsed.Employees)

	// 9. "Latest wins" overwrite: for any (employee, date) in the new import,
	// delete previously-imported unapproved entries so the latest upload
	// fully replaces them. Approved or paid entries are protected.
	if len(entries) > 0 {
		monthEnd := time.Date(year, month+1, 0, 23, 59, 59, 0, loc)
		existingTS, terr := s.timesheetRepo.GetByProject(ctx, projectID, monthStart, monthEnd)
		if terr != nil {
			return fail("failed", fmt.Sprintf("lỗi tải bảng chấm công hiện có: %v", terr))
		}

		type dk struct {
			empID uint
			date  string
		}

		importDates := make(map[dk]bool, len(entries))
		for _, e := range entries {
			importDates[dk{e.EmployeeID, e.Date}] = true
		}

		blocked := make(map[dk]string)
		var staleIDs []uint
		for _, ts := range existingTS {
			k := dk{ts.EmployeeID, ts.Date.Format("2006-01-02")}
			if !importDates[k] {
				continue
			}
			switch {
			case ts.Status == domain.TimesheetStatusApproved:
				blocked[k] = "đã được phê duyệt"
			case ts.PaymentStatus == domain.PaymentStatusPaid,
				ts.PaymentStatus == domain.PaymentStatusFailed,
				ts.PaymentStatus == domain.PaymentStatusCancelled:
				blocked[k] = "đã thanh toán"
			default:
				staleIDs = append(staleIDs, ts.ID)
			}
		}

		if len(blocked) > 0 {
			warned := make(map[dk]bool, len(blocked))
			filtered := entries[:0]
			for _, e := range entries {
				k := dk{e.EmployeeID, e.Date}
				if reason, isBlocked := blocked[k]; isBlocked {
					if !warned[k] {
						warned[k] = true
						name := empNames[e.EmployeeID]
						if name == "" {
							name = fmt.Sprintf("ID %d", e.EmployeeID)
						}
						importErrors = append(importErrors, domain.ImportError{
							Employee: name,
							Reason:   fmt.Sprintf("ngày %s: %s, không ghi đè", e.Date, reason),
						})
					}
					continue
				}
				filtered = append(filtered, e)
			}
			entries = filtered
		}

		for _, id := range staleIDs {
			if delErr := s.timesheetRepo.HardDelete(ctx, id); delErr != nil {
				slog.Warn("BCCImport: failed to hard-delete stale timesheet for overwrite",
					"timesheet_id", id, "error", delErr)
			}
		}
		if len(staleIDs) > 0 {
			slog.Warn("BCCImport: hard-deleted stale unapproved timesheets",
				"deleted_count", len(staleIDs), "project_id", projectID)
		}
	}

	// 10. Call BulkCreateTimesheets.
	if len(entries) == 0 {
		reason := "không có dữ liệu hợp lệ để tạo bảng chấm công"
		if len(importErrors) > 0 {
			detail := marshalErrors(importErrors)
			now := clock.Now()
			stats := BCCImportStats{
				ProjectID:    projectID,
				OriginalName: filename,
				ForMonth:     effectiveMonth,
				Status:       "failed",
				TotalRows:    totalRows,
				ErrorCount:   len(importErrors),
				ErrorDetail:  detail,
				ProcessedAt:  &now,
			}
			if metaErr := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); metaErr != nil {
				slog.Error("BCCImport: metadata update failed for empty-entries path", "asset_id", createdAsset.ID, "error", metaErr)
			}
			return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt),
				fmt.Errorf("import failed: %s", FirstErrorReason(detail))
		}
		return fail("failed", reason)
	}

	result, err := s.timesheetService.BulkCreateTimesheets(ctx, entries, uploaderID, uploaderRole)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi tạo bảng chấm công: %v", err))
	}

	// 11. Count results.
	createdCount := len(result.CreatedTimesheets)
	skippedCount := len(result.DeletedTimesheets)
	for _, f := range result.FailedEntries {
		importErrors = append(importErrors, domain.ImportError{
			Row:      0,
			Employee: fmt.Sprintf("employee_id=%d date=%s", f.Request.EmployeeID, f.Request.Date),
			Reason:   f.Error,
		})
	}
	errorCount := len(importErrors)

	now := clock.Now()
	finalStatus := "completed"
	if createdCount == 0 && errorCount > 0 {
		finalStatus = "failed"
	}

	// 12. Update asset metadata with final results.
	errDetail := marshalErrors(importErrors)
	stats := BCCImportStats{
		ProjectID:    projectID,
		OriginalName: filename,
		ForMonth:     effectiveMonth,
		Status:       finalStatus,
		TotalRows:    totalRows,
		CreatedCount: createdCount,
		SkippedCount: skippedCount,
		ErrorCount:   errorCount,
		ErrorDetail:  errDetail,
		ProcessedAt:  &now,
	}
	if metaErr := s.updateAssetMetadata(ctx, createdAsset.ID, &stats); metaErr != nil {
		slog.Error("BCCImport: metadata update failed for success path", "asset_id", createdAsset.ID, "error", metaErr)
	}

	return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt), nil
}

func (s *BCCImportService) updateAssetMetadata(ctx context.Context, assetID uint, meta *bccImportMetadata) error {
	b, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("BCCImport: failed to marshal metadata: %w", err)
	}
	if err := s.assetRepo.UpdateMetadata(ctx, assetID, string(b)); err != nil {
		return fmt.Errorf("BCCImport: failed to update asset metadata (id=%d): %w", assetID, err)
	}
	return nil
}

const bccImportLockTTL = 30 * time.Second

func (s *BCCImportService) acquireImportLock(ctx context.Context, projectID uint, forMonth string) (string, error) {
	lockKey := fmt.Sprintf("bcc_import:lock:%d:%s", projectID, forMonth)
	lockValue := fmt.Sprintf("%d-%d", projectID, time.Now().UnixNano())
	acquired, err := s.redis.SetNX(ctx, lockKey, lockValue, bccImportLockTTL).Result()
	if err != nil {
		return "", fmt.Errorf("failed to acquire import lock: %w", err)
	}
	if !acquired {
		return "", fmt.Errorf("một thao tác import khác đang chạy cho dự án %d tháng %s, vui lòng thử lại sau", projectID, forMonth)
	}
	return lockValue, nil
}

func (s *BCCImportService) releaseImportLock(ctx context.Context, projectID uint, forMonth, lockValue string) {
	lockKey := fmt.Sprintf("bcc_import:lock:%d:%s", projectID, forMonth)
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := s.redis.Eval(ctx, script, []string{lockKey}, lockValue).Result()
	if err != nil {
		slog.Warn("BCCImport: failed to release import lock", "key", lockKey, "error", err)
	}
}

func marshalErrors(errs []domain.ImportError) *string {
	if len(errs) == 0 {
		return nil
	}
	b, err := json.Marshal(errs)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

// FirstErrorReason extracts the first error reason from a JSON error detail string.
func FirstErrorReason(detail *string) string {
	if detail == nil {
		return "unknown"
	}
	var errs []domain.ImportError
	if err := json.Unmarshal([]byte(*detail), &errs); err != nil || len(errs) == 0 {
		return *detail
	}
	return errs[0].Reason
}

func parseForMonth(forMonth string) (int, time.Month, error) {
	t, err := clock.ParseMonth(forMonth)
	if err != nil {
		return 0, 0, fmt.Errorf("định dạng tháng không hợp lệ: %q: %w", forMonth, err)
	}
	if t.Year() < 2000 || t.Year() > 2100 {
		return 0, 0, fmt.Errorf("năm không hợp lệ: %d", t.Year())
	}
	return t.Year(), t.Month(), nil
}
