package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"api-server/internal/app/services/employee"
	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/app/services/project"
	timesheetSvc "api-server/internal/app/services/timesheet"
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"
	"api-server/internal/infra/storage"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/utils"

	"github.com/redis/go-redis/v9"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"unicode"
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
	payrateRepo         domain.PayrateRepository
	timesheetRepo       domain.TimesheetRepository
	timesheetService    *timesheetSvc.TimesheetService
	assetRepo           domain.AssetRepository
	fileStorage         storage.FileStorage
	transactionManager  domain.TransactionManager
	redis               *redis.Client
	employeeService     *employee.EmployeeService
	employeeUserService *employee.EmployeeUserService
	projectEmployeeSvc  *project.ProjectEmployeeService
}

func NewBCCImportService(
	payrateRepo domain.PayrateRepository,
	timesheetRepo domain.TimesheetRepository,
	timesheetService *timesheetSvc.TimesheetService,
	assetRepo domain.AssetRepository,
	fileStorage storage.FileStorage,
	transactionManager domain.TransactionManager,
	redisClient *redis.Client,
	employeeService *employee.EmployeeService,
	employeeUserService *employee.EmployeeUserService,
	projectEmployeeSvc *project.ProjectEmployeeService,
) *BCCImportService {
	return &BCCImportService{
		payrateRepo:         payrateRepo,
		timesheetRepo:       timesheetRepo,
		timesheetService:    timesheetService,
		assetRepo:           assetRepo,
		fileStorage:         fileStorage,
		transactionManager:  transactionManager,
		redis:               redisClient,
		employeeService:     employeeService,
		employeeUserService: employeeUserService,
		projectEmployeeSvc:  projectEmployeeSvc,
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

	// effectiveMonth is captured by the fail() closure.
	effectiveMonth := forMonth

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

	// 4a. Detect format: legacy BCC vs multi-position
	formatResult, detectErr := excelparser.DetectFormat(xf)
	if detectErr != nil {
		return fail("failed", fmt.Sprintf("không nhận diện được định dạng file: %v", detectErr))
	}

	switch formatResult.Format {
	case excelparser.FormatMultiPosition:
		return s.processMultiPositionUpload(ctx, xf, formatResult, filename, projectID, uploaderID, uploaderRole, createdAsset, effectiveMonth)
	default:
		// FormatLegacy — continue with existing BCC parsing below
	}

	parsed, err := excelparser.ParseBCCFile(xf)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi phân tích file BCC: %v", err))
	}

	// 5-6. Shared setup: month parsing, lock, payrate lookup.
	ictx, releaseLock, err := s.prepareImportContext(ctx, projectID, effectiveMonth)
	if err != nil {
		return fail("failed", err.Error())
	}
	defer releaseLock()
	year, month, monthStart, flatRates := ictx.year, ictx.month, ictx.monthStart, ictx.flatRates
	loc := monthStart.Location()

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

	// 6.5 Auto-create and assign employees from STK sheet if it exists.
	// Deduce the best position from payrate rates matching BCC shift rates.
	position := deducePosition(flatRates, parsed.ShiftRates)
	monthStartDate := time.Date(year, month, 1, 0, 0, 0, 0, loc)

	// Pre-declare importErrors so STK auto-creation failures are surfaced to the user.
	var importErrors []domain.ImportError

	stkRows, stkErr := excelparser.ParseSTKSheet(xf)
	if stkErr != nil {
		slog.Warn("BCCImport: failed to parse STK sheet", "error", stkErr)
	}

	// Build STK CCCD→Name lookup for cross-validation against BCC employee names.
	stkNameByCCCD := make(map[string]string, len(stkRows))
	for _, row := range stkRows {
		if row.CCCD != "" && row.FullName != "" {
			stkNameByCCCD[row.CCCD] = row.FullName
		}
	}

	if len(stkRows) > 0 {
		slog.Info("BCCImport: found STK sheet, processing employee auto-creation",
			"count", len(stkRows), "position", position)

		// Cache bank name → ID to avoid redundant DB queries per row.
		// Wrap STK auto-creation in a transaction to prevent orphaned records.
		stkErr := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
			bankCache := make(map[string]*uint)

			for _, row := range stkRows {
				cccd := row.CCCD
				fullName := row.FullName
				if cccd == "" || fullName == "" {
					continue
				}

				// 1. Resolve Bank ID (cached)
				var bankID *uint
				if row.BankName != "" {
					if cached, ok := bankCache[row.BankName]; ok {
						bankID = cached
					} else {
						bankID = s.employeeService.ResolveBankID(txCtx, row.BankName)
						bankCache[row.BankName] = bankID
					}
				}

				// 2. Check if employee already exists by CCCD
				existingEmp, empErr := s.employeeService.GetEmployeeByCCCD(txCtx, cccd)
				if empErr != nil && !domain.IsNotFoundError(empErr) {
					// Real DB error — don't conflate with "not found".
					slog.Error("BCCImport: DB error looking up employee by CCCD",
						"cccd", cccd, "error", empErr)
					importErrors = append(importErrors, domain.ImportError{
						Employee: fullName,
						Reason:   fmt.Sprintf("lỗi tra cứu nhân viên CCCD %s: %v", cccd, empErr),
					})
					continue
				}

				var emp *domain.Employee
				if existingEmp == nil {
					// Create new employee
					emp = &domain.Employee{
						Fullname:          fullName,
						CCCD:              cccd,
						BankAccountNumber: row.BankAccount,
						BankAccountName:   strings.ToUpper(fullName),
						BankID:            bankID,
						CreatedBy:         uploaderID,
					}
					createdEmp, createErr := s.employeeService.CreateEmployee(txCtx, emp, uploaderID)
					if createErr != nil {
						slog.Error("BCCImport: failed to auto-create employee",
							"cccd", cccd, "name", fullName, "error", createErr)
						importErrors = append(importErrors, domain.ImportError{
							Employee: fullName,
							Reason:   fmt.Sprintf("không thể tạo nhân viên CCCD %s: %v", cccd, createErr),
						})
						continue
					}
					emp = createdEmp
					slog.Info("BCCImport: auto-created employee profile and user account",
						"cccd", cccd, "employee_id", emp.ID)
				} else {
					emp = existingEmp

					// Fill missing bank info from STK (targeted update to avoid full-record Save)
					if row.BankAccount != "" && emp.BankAccountNumber == "" {
						bankUpdates := map[string]any{
							"bank_account_number": row.BankAccount,
							"bank_account_name":   strings.ToUpper(fullName),
						}
						if bankID != nil {
							bankUpdates["bank_id"] = *bankID
						}
						if updateErr := s.employeeService.UpdateBankInfo(txCtx, emp.ID, bankUpdates); updateErr != nil {
							slog.Error("BCCImport: failed to fill bank info for employee",
								"employee_id", emp.ID, "error", updateErr)
						} else {
							slog.Info("BCCImport: filled bank info for existing employee",
								"employee_id", emp.ID, "cccd", cccd)
						}
					}

					// If employee exists but has no user account, create one
					if emp.UserID == nil && s.employeeUserService != nil {
						baseUsername := utils.GenerateUsername(emp.Fullname)
						if baseUsername != "" {
							username := s.employeeUserService.EnsureUniqueUsername(txCtx, baseUsername)
							userID, userErr := s.employeeUserService.CreateUserForEmployee(txCtx, emp, username)
							if userErr != nil {
								slog.Error("BCCImport: failed to create user for existing employee",
									"employee_id", emp.ID, "cccd", cccd, "error", userErr)
							} else {
								// Targeted update: only set user_id to avoid full-record Save overwriting concurrent changes.
								if updateErr := s.employeeService.UpdateUserLink(txCtx, emp.ID, userID); updateErr != nil {
									slog.Error("BCCImport: failed to update employee with user_id",
										"employee_id", emp.ID, "user_id", userID, "error", updateErr)
								} else {
									slog.Info("BCCImport: created user account for existing employee",
										"employee_id", emp.ID, "cccd", cccd, "username", username)
								}
							}
						}
					}
				}

				// 3. Ensure employee is assigned to the project
				existingAssignment, assignErr := s.employeeService.GetActiveAssignment(txCtx, projectID, emp.ID)
				if assignErr != nil || existingAssignment == nil {
					// Deduce position specifically for this employee based on their timesheet entries
					empPosition := position
					var matchedParsedEmp *excelparser.BCCEmployeeData
					for idx := range parsed.Employees {
						if parsed.Employees[idx].CCCD == cccd {
							matchedParsedEmp = &parsed.Employees[idx]
							break
						}
					}
					if matchedParsedEmp != nil && len(matchedParsedEmp.Entries) > 0 {
						empRates := make(map[string]int64)
						for _, entry := range matchedParsedEmp.Entries {
							if rate, ok := parsed.ShiftRates[entry.ShiftLabel]; ok {
								empRates[entry.ShiftLabel] = rate
							}
						}
						empPosition = deducePosition(flatRates, empRates)
					}

					assignment := &domain.ProjectEmployee{
						ProjectID:       projectID,
						EmployeeID:      emp.ID,
						EmployeeName:    emp.Fullname,
						EmployeeCCCD:    emp.CCCD,
						Position:        empPosition,
						StartDate:       monthStartDate,
						PaymentSchedule: string(domain.PaymentScheduleWeekly),
						CreatedBy:       uploaderID,
					}

					if createErr := s.employeeService.CreateAssignment(txCtx, assignment); createErr != nil {
						slog.Error("BCCImport: failed to auto-assign employee to project",
							"employee_id", emp.ID, "project_id", projectID, "error", createErr)
						importErrors = append(importErrors, domain.ImportError{
							Employee: fullName,
							Reason:   fmt.Sprintf("không thể phân công nhân viên %s vào dự án: %v", fullName, createErr),
						})
					} else {
						slog.Info("BCCImport: auto-assigned employee to project",
							"employee_id", emp.ID, "project_id", projectID,
							"position", empPosition, "start_date", monthStartDate.Format("2006-01-02"))
					}
				}
			}
			return nil
		})
		if stkErr != nil {
			slog.Error("BCCImport: STK auto-creation transaction failed", "error", stkErr)
		}
	}

	// 7. Load all active project employees, build lookup maps.
	assignments, err := s.employeeService.GetActiveAssignments(ctx, projectID)
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
		entries []domainservices.BulkCreateTimesheetEntry
		rowNum  = 12
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

		// STK cross-check: if the same CCCD appears in STK with a different name,
		// the BCC sheet likely has a CCCD typo (one CCCD assigned to two different people).
		if stkName, ok := stkNameByCCCD[emp.CCCD]; ok && stkName != "" {
			// Normalize both names for comparison: trim, NFC, lower-case. NFC is
			// important because Excel files from macOS can ship Vietnamese text
			// in NFD form, which would otherwise produce a spurious mismatch.
			bccNorm := bccNormName(emp.FullName)
			stkNorm := bccNormName(stkName)
			if bccNorm != stkNorm {
				// Names differ after NFC normalization. Try a diacritic-stripped
				// comparison before rejecting: a single accent typo (e.g. "Thì" vs
				// "Thị") is data-entry noise, not a real CCCD mismatch. Allow the
				// import and log a warning so the partner can correct the typo.
				if bccNormNameLoose(emp.FullName) == bccNormNameLoose(stkName) {
					slog.Warn("BCCImport: STK/BCC name differs only by diacritic, allowing import",
						"cccd", emp.CCCD, "bcc_name", emp.FullName, "stk_name", stkName)
				} else {
					cleanBCCName := strings.TrimSpace(emp.FullName)
					cleanSTKName := strings.TrimSpace(stkName)
					importErrors = append(importErrors, domain.ImportError{
						Row:      rowNum,
						Employee: cleanBCCName,
						Reason:   fmt.Sprintf("CCCD %s thuộc về %s (theo STK), không phải %s — có thể sai CCCD", emp.CCCD, cleanSTKName, cleanBCCName),
					})
					rowNum++
					continue
				}
			}
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
		isAdmin := uploaderRole == string(domain.RoleAdmin)
		for _, ts := range existingTS {
			k := dk{ts.EmployeeID, ts.Date.Format("2006-01-02")}
			if !importDates[k] {
				continue
			}

			isPaid := ts.PaymentStatus == domain.PaymentStatusPaid ||
				ts.PaymentStatus == domain.PaymentStatusFailed ||
				ts.PaymentStatus == domain.PaymentStatusCancelled

			switch {
			case isPaid:
				blocked[k] = "đã thanh toán"
			case ts.Status == domain.TimesheetStatusApproved && !isAdmin:
				blocked[k] = "đã được phê duyệt"
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

// bccImportContext holds the shared setup results for BCC import flows.
// Both single-position and multi-position flows perform the same month parsing,
// lock acquisition, and payrate lookup — this struct consolidates that setup.
type bccImportContext struct {
	year       int
	month      time.Month
	monthStart time.Time
	flatRates  map[string]int
}

// prepareImportContext performs the shared setup: month parsing, distributed lock,
// and payrate lookup. Returns the context and a cleanup function that must be
// deferred by the caller to release the lock.
func (s *BCCImportService) prepareImportContext(ctx context.Context, projectID uint, effectiveMonth string) (*bccImportContext, func(), error) {
	loc := clock.Now().Location()
	year, month, err := parseForMonth(effectiveMonth)
	if err != nil {
		return nil, nil, fmt.Errorf("tháng không hợp lệ: %v", err)
	}
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, loc)

	lockValue, lockErr := s.acquireImportLock(ctx, projectID, effectiveMonth)
	if lockErr != nil {
		return nil, nil, lockErr
	}
	release := func() { s.releaseImportLock(ctx, projectID, effectiveMonth, lockValue) }

	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, projectID, monthStart)
	if err != nil {
		release()
		return nil, nil, fmt.Errorf("không tìm thấy bảng lương cho dự án: %v", err)
	}
	flatRates, err := payrate.Payrate.Flatten()
	if err != nil {
		release()
		return nil, nil, fmt.Errorf("lỗi phân tích cấu hình lương: %v", err)
	}

	return &bccImportContext{
		year:       year,
		month:      month,
		monthStart: monthStart,
		flatRates:  flatRates,
	}, release, nil
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

// deducePosition determines the best position for auto-created employees by matching
// BCC shift rates against the project's payrate configuration.
// Payrate paths are "position.dayType.hourType" → rate. We find which position
// has the most matching rates with the BCC shift rates.
func deducePosition(flatRates map[string]int, shiftRates map[string]int64) string {
	if len(flatRates) == 0 || len(shiftRates) == 0 {
		return "phổ thông"
	}

	// Collect unique BCC rate values (the VND amounts from row 10 of BCC sheet).
	bccRates := make(map[int]bool, len(shiftRates))
	for _, r := range shiftRates {
		if r > 0 {
			bccRates[int(r)] = true
		}
	}
	if len(bccRates) == 0 {
		return "phổ thông"
	}

	// For each position, count how many of its payrate values match BCC rates.
	positionHits := make(map[string]int)
	for path, rate := range flatRates {
		if rate == 0 {
			continue
		}
		if !bccRates[rate] {
			continue
		}
		parts := strings.Split(path, ".")
		if len(parts) < 1 || parts[0] == "" {
			continue
		}
		positionHits[parts[0]]++
	}

	if len(positionHits) == 0 {
		return "phổ thông"
	}

	// Pick the position with the most matching rates.
	best := "phổ thông"
	bestCount := 0
	for pos, count := range positionHits {
		if count > bestCount {
			bestCount = count
			best = pos
		}
	}
	return best
}

// posCorrection records a pending position update to apply after the STK transaction.
// Applying outside the transaction via ProjectEmployeeService.UpdateAssignmentPosition
// ensures cache invalidation, event publishing, and timesheet recalculation while
// using a targeted column update to avoid full-row Save() overwrites.
type posCorrection struct {
	assignmentID uint
	employeeID   uint
	oldPosition  string
	newPosition  string
	employeeName string
}

// processMultiPositionUpload handles the new multi-position BCC format where each
// sheet (beyond STK) represents a position. It mirrors ProcessUpload's orchestration
// but uses the multi-position parser and position-from-sheet-name logic.
func (s *BCCImportService) processMultiPositionUpload(
	ctx context.Context,
	xf *excelize.File,
	formatResult *excelparser.FormatDetectionResult,
	filename string,
	projectID uint,
	uploaderID uint,
	uploaderRole string,
	createdAsset *domain.Asset,
	effectiveMonth string,
) (*BCCImportResult, error) {
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
			slog.Error("BCCImport(MP): metadata update failed in fail path", "asset_id", createdAsset.ID, "error", metaErr)
		}
		return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt),
			fmt.Errorf("import failed: %s", FirstErrorReason(detail))
	}

	// 1. Parse the multi-position sheets.
	parsed, err := excelparser.ParseMultiPositionFile(xf, formatResult.PositionSheets)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi phân tích file BCC đa vị trí: %v", err))
	}

	// 2-4. Shared setup: month parsing, lock, payrate lookup.
	ictx, releaseLock, err := s.prepareImportContext(ctx, projectID, effectiveMonth)
	if err != nil {
		return fail("failed", err.Error())
	}
	defer releaseLock()
	year, month, monthStart, flatRates := ictx.year, ictx.month, ictx.monthStart, ictx.flatRates
	loc := monthStart.Location()

	// Validate all sheet positions exist in payrate config.
	availablePositions := getPositions(flatRates)
	availableSet := make(map[string]bool, len(availablePositions))
	for _, p := range availablePositions {
		availableSet[strings.ToLower(p)] = true
	}
	for _, sheet := range parsed.Sheets {
		if !availableSet[strings.ToLower(sheet.Position)] {
			return fail("failed",
				fmt.Sprintf("vị trí \"%s\" không có trong cấu hình lương. Các vị trí khả dụng: %s",
					sheet.Position, strings.Join(availablePositions, ", ")))
		}
	}

	// 5. Build CCCD→position map from all position sheets (for STK auto-creation).
	// Note: PositionEmployeeData.EmployeeCode holds CCCD in the multi-position template.
	cccdToPosition := make(map[string]string)
	var importErrors []domain.ImportError
	for _, sheet := range parsed.Sheets {
		for _, emp := range sheet.Employees {
			if emp.EmployeeCode == "" { // EmployeeCode is CCCD in multi-position template
				continue
			}
			if existing, dup := cccdToPosition[emp.EmployeeCode]; dup && existing != sheet.Position {
				importErrors = append(importErrors, domain.ImportError{
					Employee: emp.FullName,
					Reason:   fmt.Sprintf("nhân viên \"%s\" xuất hiện ở nhiều sheet vị trí (%s, %s)", emp.FullName, existing, sheet.Position),
				})
				continue
			}
			cccdToPosition[emp.EmployeeCode] = sheet.Position
		}
	}

	// 6. STK auto-creation with CCCD→position lookup.
	var positionCorrections []posCorrection
	stkRows, stkErr := excelparser.ParseSTKSheet(xf)
	if stkErr != nil {
		slog.Warn("BCCImport(MP): failed to parse STK sheet", "error", stkErr)
	} else if len(stkRows) > 0 {
		slog.Info("BCCImport(MP): found STK sheet, processing employee auto-creation", "count", len(stkRows))

		sort.Strings(availablePositions)

		stkErr := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
			bankCache := make(map[string]*uint)

			for _, row := range stkRows {
				cccd := row.CCCD
				fullName := row.FullName
				if cccd == "" || fullName == "" {
					continue
				}

				// Resolve bank ID.
				var bankID *uint
				if row.BankName != "" {
					if cached, ok := bankCache[row.BankName]; ok {
						bankID = cached
					} else {
						bankID = s.employeeService.ResolveBankID(txCtx, row.BankName)
						bankCache[row.BankName] = bankID
					}
				}

				// Check if employee exists.
				existingEmp, empErr := s.employeeService.GetEmployeeByCCCD(txCtx, cccd)
				if empErr != nil && !domain.IsNotFoundError(empErr) {
					importErrors = append(importErrors, domain.ImportError{
						Employee: fullName,
						Reason:   fmt.Sprintf("lỗi tra cứu nhân viên CCCD %s: %v", cccd, empErr),
					})
					continue
				}

				var emp *domain.Employee
				if existingEmp == nil {
					emp = &domain.Employee{
						Fullname:          fullName,
						CCCD:              cccd,
						BankAccountNumber: row.BankAccount,
						BankAccountName:   strings.ToUpper(fullName),
						BankID:            bankID,
						CreatedBy:         uploaderID,
					}
					createdEmp, createErr := s.employeeService.CreateEmployee(txCtx, emp, uploaderID)
					if createErr != nil {
						importErrors = append(importErrors, domain.ImportError{
							Employee: fullName,
							Reason:   fmt.Sprintf("không thể tạo nhân viên CCCD %s: %v", cccd, createErr),
						})
						continue
					}
					emp = createdEmp
				} else {
					emp = existingEmp
					if row.BankAccount != "" && emp.BankAccountNumber == "" {
						bankUpdates := map[string]any{
							"bank_account_number": row.BankAccount,
							"bank_account_name":   strings.ToUpper(fullName),
						}
						if bankID != nil {
							bankUpdates["bank_id"] = *bankID
						}
						if updateErr := s.employeeService.UpdateBankInfo(txCtx, emp.ID, bankUpdates); updateErr != nil {
							slog.Error("BCCImport(MP): failed to fill bank info for employee",
								"employee_id", emp.ID, "error", updateErr)
						}
					}
					if emp.UserID == nil && s.employeeUserService != nil {
						baseUsername := utils.GenerateUsername(emp.Fullname)
						if baseUsername != "" {
							username := s.employeeUserService.EnsureUniqueUsername(txCtx, baseUsername)
							userID, userErr := s.employeeUserService.CreateUserForEmployee(txCtx, emp, username)
							if userErr != nil {
								slog.Error("BCCImport(MP): failed to create user for existing employee",
									"employee_id", emp.ID, "cccd", cccd, "error", userErr)
							} else {
								if linkErr := s.employeeService.UpdateUserLink(txCtx, emp.ID, userID); linkErr != nil {
									slog.Error("BCCImport(MP): failed to update employee with user_id",
										"employee_id", emp.ID, "user_id", userID, "error", linkErr)
								}
							}
						}
					}
				}

				// Ensure assignment with position from CCCD→position map.
				existingAssignment, assignErr := s.employeeService.GetActiveAssignment(txCtx, projectID, emp.ID)
				if assignErr != nil && !domain.IsNotFoundError(assignErr) {
					importErrors = append(importErrors, domain.ImportError{
						Employee: fullName,
						Reason:   fmt.Sprintf("lỗi kiểm tra phân công nhân viên %s: %v", fullName, assignErr),
					})
					continue
				}
				if existingAssignment == nil {
					empPosition := cccdToPosition[cccd]
					if empPosition == "" && len(availablePositions) > 0 {
						empPosition = availablePositions[0]
					}
					assignment := &domain.ProjectEmployee{
						ProjectID:       projectID,
						EmployeeID:      emp.ID,
						EmployeeName:    emp.Fullname,
						EmployeeCCCD:    emp.CCCD,
						Position:        empPosition,
						StartDate:       monthStart,
						PaymentSchedule: string(domain.PaymentScheduleWeekly),
						CreatedBy:       uploaderID,
					}
					if createErr := s.employeeService.CreateAssignment(txCtx, assignment); createErr != nil {
						importErrors = append(importErrors, domain.ImportError{
							Employee: fullName,
							Reason:   fmt.Sprintf("không thể phân công nhân viên %s: %v", fullName, createErr),
						})
					}
				} else if expectedPos := cccdToPosition[cccd]; expectedPos != "" && !strings.EqualFold(existingAssignment.Position, expectedPos) {
					// Collect position correction to apply after transaction via ProjectEmployeeService
					// (which handles cache invalidation, events, validation, and timesheet recalculation).
					positionCorrections = append(positionCorrections, posCorrection{
						assignmentID: existingAssignment.ID,
						employeeID:   emp.ID,
						oldPosition:  existingAssignment.Position,
						newPosition:  expectedPos,
						employeeName: fullName,
					})
				}
			}
			return nil
		})
		if stkErr != nil {
			slog.Error("BCCImport(MP): STK auto-creation transaction failed", "error", stkErr)
		}
	}

	// 6.5. Apply position corrections via ProjectEmployeeService (outside STK transaction)
	// to get cache invalidation, event publishing, and timesheet recalculation with
	// targeted column update (no full-row Save() overwrite risk).
	for _, corr := range positionCorrections {
		if updateErr := s.projectEmployeeSvc.UpdateAssignmentPosition(ctx, corr.assignmentID, corr.newPosition, uploaderID); updateErr != nil {
			slog.Error("BCCImport(MP): failed to update assignment position",
				"assignment_id", corr.assignmentID, "position", corr.newPosition, "error", updateErr)
			importErrors = append(importErrors, domain.ImportError{
				Employee: corr.employeeName,
				Reason:   fmt.Sprintf("lỗi cập nhật vị trí cho nhân viên %s: %v", corr.employeeName, updateErr),
			})
		} else {
			slog.Info("BCCImport(MP): updated assignment position",
				"employee_id", corr.employeeID, "old", corr.oldPosition, "new", corr.newPosition)
		}
	}

	// 7. Load assignments.
	assignments, err := s.employeeService.GetActiveAssignments(ctx, projectID)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi tải danh sách nhân viên: %v", err))
	}
	byCCCD := make(map[string]*domain.ProjectEmployee, len(assignments))
	byCode := make(map[string]*domain.ProjectEmployee, len(assignments))
	byCCCDAndPosition := make(map[string]*domain.ProjectEmployee, len(assignments))
	empNames := make(map[uint]string, len(assignments))
	for _, a := range assignments {
		byCCCD[a.EmployeeCCCD] = a
		byCCCDAndPosition[a.EmployeeCCCD+"|"+strings.ToLower(a.Position)] = a
		if a.EmployeeCode != "" {
			byCode[a.EmployeeCode] = a
		}
		empNames[a.EmployeeID] = a.EmployeeName
	}

	// 8. Build timesheet entries for each position sheet.
	dayTypePriority := map[string]int{"ngày thường": 0, "ngày nghỉ": 1, "ngày lễ": 2}
	type rateTarget struct{ dayType, hourType string }

	var entries []domainservices.BulkCreateTimesheetEntry
	totalRows := 0

	for _, sheet := range parsed.Sheets {
		// Build position-scoped rate-to-target map.
		posPrefix := strings.ToLower(sheet.Position) + "."
		rateToTarget := make(map[int]rateTarget)
		for path, rate := range flatRates {
			if rate == 0 || !strings.HasPrefix(strings.ToLower(path), posPrefix) {
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
			existing, exists := rateToTarget[rate]
			if !exists || candPri < dayTypePriority[existing.dayType] {
				rateToTarget[rate] = candidate
			}
		}

		for _, emp := range sheet.Employees {
			totalRows++

			// Match by CCCD+position first, then fall back.
			// Note: emp.EmployeeCode holds CCCD in the multi-position template, so it matches
			// the byCCCDAndPosition map keyed by assignment.EmployeeCCCD.
			empLookupKey := emp.EmployeeCode // CCCD in multi-position template
			assignment := byCCCDAndPosition[empLookupKey+"|"+strings.ToLower(sheet.Position)]
			if assignment == nil {
				assignment = byCCCD[empLookupKey]
			}
			if assignment == nil {
				assignment = byCode[empLookupKey]
			}
			if assignment == nil {
				importErrors = append(importErrors, domain.ImportError{
					Employee: emp.FullName,
					Reason:   fmt.Sprintf("không tìm thấy nhân viên với mã \"%s\" trong dự án", emp.EmployeeCode),
				})
				continue
			}

			// Correct position for non-STK employees whose assignment position doesn't match this sheet.
			if !strings.EqualFold(assignment.Position, sheet.Position) {
				oldPos := assignment.Position
				if updateErr := s.projectEmployeeSvc.UpdateAssignmentPosition(ctx, assignment.ID, sheet.Position, uploaderID); updateErr != nil {
					slog.Error("BCCImport(MP): failed to correct position in BCC sheet",
						"assignment_id", assignment.ID, "position", sheet.Position, "error", updateErr)
					importErrors = append(importErrors, domain.ImportError{
						Employee: emp.FullName,
						Reason:   fmt.Sprintf("lỗi cập nhật vị trí cho nhân viên %s: %v", emp.FullName, updateErr),
					})
				} else {
					slog.Info("BCCImport(MP): corrected assignment position from BCC sheet",
						"employee_id", assignment.EmployeeID, "old", oldPos, "new", sheet.Position)
					assignment.Position = sheet.Position
					// Update lookup maps so subsequent sheets find the corrected position.
					byCCCDAndPosition[assignment.EmployeeCCCD+"|"+strings.ToLower(sheet.Position)] = assignment
				}
			}

			if assignment.PaymentSchedule == string(domain.PaymentScheduleFlexible) {
				importErrors = append(importErrors, domain.ImportError{
					Employee: emp.FullName,
					Reason:   "nhân viên lương linh hoạt không áp dụng BCC import",
				})
				continue
			}

			for _, entry := range emp.Entries {
				date := time.Date(year, month, entry.DayNum, 0, 0, 0, 0, loc)
				if date.Month() != month {
					continue
				}

				target, ok := rateToTarget[entry.RateVND]
				if !ok || entry.RateVND == 0 {
					importErrors = append(importErrors, domain.ImportError{
						Employee: emp.FullName,
						Reason:   fmt.Sprintf("không tìm thấy mức lương cho ngày %d (%d VND) ở vị trí %s", entry.DayNum, entry.RateVND, sheet.Position),
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
		}
	}

	// 9. "Latest wins" overwrite (same logic as legacy path).
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
		isAdmin := uploaderRole == string(domain.RoleAdmin)
		for _, ts := range existingTS {
			k := dk{ts.EmployeeID, ts.Date.Format("2006-01-02")}
			if !importDates[k] {
				continue
			}

			isPaid := ts.PaymentStatus == domain.PaymentStatusPaid ||
				ts.PaymentStatus == domain.PaymentStatusFailed ||
				ts.PaymentStatus == domain.PaymentStatusCancelled

			switch {
			case isPaid:
				blocked[k] = "đã thanh toán"
			case ts.Status == domain.TimesheetStatusApproved && !isAdmin:
				blocked[k] = "đã được phê duyệt"
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
				slog.Warn("BCCImport(MP): failed to hard-delete stale timesheet", "timesheet_id", id, "error", delErr)
			}
		}
		if len(staleIDs) > 0 {
			slog.Warn("BCCImport(MP): hard-deleted stale unapproved timesheets",
				"deleted_count", len(staleIDs), "project_id", projectID)
		}
	}

	// 10. Bulk create or return failure.
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
				slog.Error("BCCImport(MP): metadata update failed in empty-entries path",
					"asset_id", createdAsset.ID, "error", metaErr)
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

	// 11. Finalize.
	createdCount := len(result.CreatedTimesheets)
	skippedCount := len(result.DeletedTimesheets)
	for _, f := range result.FailedEntries {
		importErrors = append(importErrors, domain.ImportError{
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
		slog.Error("BCCImport(MP): metadata update failed", "asset_id", createdAsset.ID, "error", metaErr)
	}

	return buildResult(stats, createdAsset.ID, uploaderID, createdAsset.CreatedAt), nil
}

// getPositions extracts unique position names from flattened payrate paths.
func getPositions(flatRates map[string]int) []string {
	seen := make(map[string]bool)
	var positions []string
	for path := range flatRates {
		parts := strings.Split(path, ".")
		if len(parts) >= 1 && parts[0] != "" && !seen[parts[0]] {
			seen[parts[0]] = true
			positions = append(positions, parts[0])
		}
	}
	return positions
}

// bccNormName normalizes a Vietnamese name for case/whitespace/Unicode-form
// insensitive comparison: trim, NFC, lower-case. Excel files from macOS can
// ship Vietnamese diacritics in NFD (decomposed) form, which would otherwise
// produce false mismatches against NFC text from other tools.
func bccNormName(s string) string {
	return strings.ToLower(strings.TrimSpace(norm.NFC.String(s)))
}

// bccNormNameLoose normalizes a Vietnamese name the same way as bccNormName,
// then strips diacritics (e.g. "Lò Thì Dương" → "lo thi duong"). Used to
// detect names that differ only by an accent mark (a common data-entry typo
// where the same person is recorded with a slightly different diacritic in
// different sheets — STK vs BCC, or BCC vs the employee profile). Two names
// whose loose-normalized forms are equal almost certainly refer to the
// same person, even when the strict-normalized forms differ.
func bccNormNameLoose(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	stripped, _, _ := transform.String(t, bccNormName(s))
	return stripped
}
