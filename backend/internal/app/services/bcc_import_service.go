package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
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
)

type BCCImportService struct {
	payrateRepo         domain.PayrateRepository
	timesheetReader     domain.TimesheetReader
	timesheetWriter     domain.TimesheetWriter
	timesheetService    *timesheetSvc.TimesheetService
	assetRepo           domain.AssetRepository
	fileStorage         storage.FileStorage
	transactionManager  domain.TransactionManager
	redis               *redis.Client
	employeeService     *employee.EmployeeService
	employeeUserService *employee.EmployeeUserService
	projectEmployeeSvc  *project.ProjectEmployeeService
	importJobRepo       domain.TimesheetImportJobRepository
	importEnqueuer      domain.TimesheetImportEnqueuer
}

var (
	ErrBCCIdempotencyConflict = errors.New("idempotency key đã được dùng cho một tệp khác")
	ErrBCCImportScopeBusy     = errors.New("dự án và tháng này đang có một tệp BCC được xử lý")
)

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
	importJobRepo domain.TimesheetImportJobRepository,
	importEnqueuer domain.TimesheetImportEnqueuer,
) *BCCImportService {
	return &BCCImportService{
		payrateRepo:         payrateRepo,
		timesheetReader:     timesheetRepo,
		timesheetWriter:     timesheetRepo,
		timesheetService:    timesheetService,
		assetRepo:           assetRepo,
		fileStorage:         fileStorage,
		transactionManager:  transactionManager,
		redis:               redisClient,
		employeeService:     employeeService,
		employeeUserService: employeeUserService,
		projectEmployeeSvc:  projectEmployeeSvc,
		importJobRepo:       importJobRepo,
		importEnqueuer:      importEnqueuer,
	}
}

const maxBCCUploadSize = 10 << 20

type deferBCCTerminalMetadataKey struct{}

// AcceptUpload durably stores a BCC upload and returns before spreadsheet
// processing starts. The idempotency key belongs to one uploader and one exact
// project/month/file fingerprint.
func (s *BCCImportService) AcceptUpload(
	ctx context.Context,
	fileData io.Reader,
	filename string,
	projectID uint,
	uploaderID uint,
	uploaderRole string,
	forMonth string,
	includeFlexibleEmployees bool,
	idempotencyKey string,
) (*BCCImportResult, error) {
	data, err := io.ReadAll(io.LimitReader(fileData, maxBCCUploadSize+1))
	if err != nil {
		return nil, fmt.Errorf("BCCImportService.AcceptUpload: read file: %w", err)
	}
	if len(data) > maxBCCUploadSize {
		return nil, fmt.Errorf("file quá lớn (tối đa 10MB)")
	}
	if _, _, err := parseForMonth(forMonth); err != nil {
		return nil, fmt.Errorf("tháng không hợp lệ: %w", err)
	}

	fingerprint := bccRequestFingerprint(projectID, forMonth, includeFlexibleEmployees, data)
	if existing, lookupErr := s.importJobRepo.GetByIdempotencyKey(ctx, uploaderID, idempotencyKey); lookupErr == nil {
		if existing.RequestFingerprint != fingerprint {
			return nil, ErrBCCIdempotencyConflict
		}
		return s.resultForAsset(ctx, existing.AssetID)
	} else if !domain.IsNotFoundError(lookupErr) {
		return nil, lookupErr
	}

	stored, err := s.fileStorage.StoreBytes(data, filename, domain.UploadTypePartnerBCCImport)
	if err != nil {
		return nil, fmt.Errorf("BCCImportService.AcceptUpload: store file: %w", err)
	}

	now := clock.Now()
	stats := BCCImportStats{
		ProjectID:    projectID,
		OriginalName: filename,
		ForMonth:     forMonth,
		Status:       domain.TimesheetImportStatusPending,
	}
	metadata, err := json.Marshal(stats)
	if err != nil {
		_ = s.fileStorage.Delete(stored.FilePath)
		return nil, fmt.Errorf("BCCImportService.AcceptUpload: metadata: %w", err)
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(data))
	activeScope := fmt.Sprintf("%d:%s", projectID, forMonth)
	asset := &domain.Asset{
		Filename:   stored.OriginalFilename,
		FilePath:   stored.FilePath,
		UploadType: domain.UploadTypePartnerBCCImport,
		Checksum:   &checksum,
		UploadedBy: uploaderID,
		Metadata:   stringPointer(string(metadata)),
	}

	err = s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		created, createErr := s.assetRepo.Create(txCtx, asset)
		if createErr != nil {
			return createErr
		}
		asset = created
		return s.importJobRepo.Create(txCtx, &domain.TimesheetImportJob{
			AssetID:                  asset.ID,
			ProjectID:                projectID,
			ForMonth:                 forMonth,
			UploadedBy:               uploaderID,
			UploaderRole:             uploaderRole,
			IncludeFlexibleEmployees: includeFlexibleEmployees,
			Status:                   domain.TimesheetImportStatusPending,
			IdempotencyKey:           idempotencyKey,
			RequestFingerprint:       fingerprint,
			ActiveScopeKey:           &activeScope,
			CreatedAt:                now,
			UpdatedAt:                now,
		})
	})
	if err != nil {
		_ = s.fileStorage.Delete(stored.FilePath)
		if existing, lookupErr := s.importJobRepo.GetByIdempotencyKey(ctx, uploaderID, idempotencyKey); lookupErr == nil {
			if existing.RequestFingerprint != fingerprint {
				return nil, ErrBCCIdempotencyConflict
			}
			return s.resultForAsset(ctx, existing.AssetID)
		}
		if strings.Contains(strings.ToLower(err.Error()), "active_scope") ||
			strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, ErrBCCImportScopeBusy
		}
		return nil, err
	}

	if enqueueErr := s.importEnqueuer.EnqueueBCCImport(asset.ID); enqueueErr != nil {
		// The durable pending row is an outbox. The periodic recovery task will
		// enqueue it after a transient Redis/queue outage.
		slog.Error("BCCImport: initial enqueue failed; recovery will retry",
			"asset_id", asset.ID, "error", enqueueErr)
	}
	return buildResult(stats, asset.ID, uploaderID, asset.CreatedAt), nil
}

// ProcessPendingJob claims and processes one durable import. Duplicate Asynq
// deliveries are harmless because only one pending/expired job can be claimed.
func (s *BCCImportService) ProcessPendingJob(ctx context.Context, assetID uint) error {
	job, err := s.importJobRepo.GetByAssetID(ctx, assetID)
	if err != nil {
		return err
	}
	if job.IsTerminal() {
		return nil
	}
	attempt, claimed, err := s.importJobRepo.Claim(ctx, assetID, clock.Now().Add(15*time.Minute))
	if err != nil || !claimed {
		return err
	}

	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return s.importJobRepo.Fail(ctx, assetID, attempt, err.Error(), clock.Now())
		}
		_ = s.importJobRepo.ReleaseForRetry(ctx, assetID, attempt, err.Error())
		return err
	}
	if asset.Metadata != nil {
		var persisted BCCImportStats
		if json.Unmarshal([]byte(*asset.Metadata), &persisted) == nil {
			switch persisted.Status {
			case domain.TimesheetImportStatusCompleted:
				return s.importJobRepo.Complete(ctx, assetID, attempt, clock.Now())
			case domain.TimesheetImportStatusFailed:
				return s.importJobRepo.Fail(ctx, assetID, attempt, FirstErrorReason(persisted.ErrorDetail), clock.Now())
			}
		}
	}
	data, err := os.ReadFile(s.fileStorage.GetFilePath(asset.FilePath))
	if err != nil {
		result := failedBCCImportResult(
			asset,
			job,
			fmt.Sprintf("không thể đọc tệp BCC đã lưu: %v", err),
		)
		return s.finalizeImportJob(ctx, assetID, attempt, result)
	}
	processingStats := BCCImportStats{
		ProjectID:    job.ProjectID,
		OriginalName: asset.Filename,
		ForMonth:     job.ForMonth,
		Status:       domain.TimesheetImportStatusProcessing,
	}
	if err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.importJobRepo.LockProcessingAttempt(txCtx, assetID, attempt); err != nil {
			return err
		}
		return s.updateAssetMetadata(txCtx, assetID, &processingStats)
	}); err != nil {
		_ = s.importJobRepo.ReleaseForRetry(ctx, assetID, attempt, err.Error())
		return err
	}

	var result *BCCImportResult
	processErr := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.importJobRepo.LockProcessingAttempt(txCtx, assetID, attempt); err != nil {
			return err
		}
		processingCtx := context.WithValue(txCtx, deferBCCTerminalMetadataKey{}, true)
		var importErr error
		result, importErr = s.processAssetData(
			processingCtx,
			data,
			asset.Filename,
			job.ProjectID,
			job.UploadedBy,
			job.UploaderRole,
			asset,
			job.ForMonth,
			job.IncludeFlexibleEmployees,
		)
		if importErr != nil && result == nil {
			return importErr
		}
		return s.finalizeImportJob(txCtx, assetID, attempt, result)
	})
	if processErr != nil {
		_ = s.importJobRepo.ReleaseForRetry(ctx, assetID, attempt, processErr.Error())
		return processErr
	}
	return nil
}

func failedBCCImportResult(
	asset *domain.Asset,
	job *domain.TimesheetImportJob,
	reason string,
) *BCCImportResult {
	now := clock.Now()
	detail := marshalErrors([]domain.ImportError{{Reason: reason}})
	stats := BCCImportStats{
		ProjectID:    job.ProjectID,
		OriginalName: asset.Filename,
		ForMonth:     job.ForMonth,
		Status:       domain.TimesheetImportStatusFailed,
		ErrorCount:   1,
		ErrorDetail:  detail,
		ProcessedAt:  &now,
	}
	return buildResult(stats, asset.ID, job.UploadedBy, asset.CreatedAt)
}

func (s *BCCImportService) finalizeImportJob(
	ctx context.Context,
	assetID uint,
	attempt uint,
	result *BCCImportResult,
) error {
	if result == nil {
		return fmt.Errorf("BCCImportService.finalizeImportJob: missing result")
	}
	processedAt := clock.Now()
	return s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.importJobRepo.LockProcessingAttempt(txCtx, assetID, attempt); err != nil {
			return err
		}
		stats := BCCImportStats{
			ProjectID:    result.ProjectID,
			OriginalName: result.OriginalName,
			ForMonth:     result.ForMonth,
			Status:       result.Status,
			TotalRows:    result.TotalRows,
			CreatedCount: result.CreatedCount,
			SkippedCount: result.SkippedCount,
			ErrorCount:   result.ErrorCount,
			ErrorDetail:  result.ErrorDetail,
			ProcessedAt:  result.ProcessedAt,
		}
		if err := s.updateAssetMetadata(txCtx, assetID, &stats); err != nil {
			return err
		}
		if result.Status == domain.TimesheetImportStatusFailed {
			return s.importJobRepo.Fail(txCtx, assetID, attempt, FirstErrorReason(result.ErrorDetail), processedAt)
		}
		return s.importJobRepo.Complete(txCtx, assetID, attempt, processedAt)
	})
}

func (s *BCCImportService) RecoverPendingJobs(ctx context.Context) error {
	jobs, err := s.importJobRepo.ListRecoverable(ctx, 100, clock.Now())
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err := s.importEnqueuer.EnqueueBCCImport(job.AssetID); err != nil {
			slog.Warn("BCCImport: recovery enqueue failed", "asset_id", job.AssetID, "error", err)
		}
	}
	return nil
}

func (s *BCCImportService) resultForAsset(ctx context.Context, assetID uint) (*BCCImportResult, error) {
	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if asset.Metadata == nil {
		return nil, fmt.Errorf("BCC import %d has no metadata", assetID)
	}
	var stats BCCImportStats
	if err := json.Unmarshal([]byte(*asset.Metadata), &stats); err != nil {
		return nil, err
	}
	return buildResult(stats, asset.ID, asset.UploadedBy, asset.CreatedAt), nil
}

func (s *BCCImportService) PendingTerminalAudit(ctx context.Context, assetID uint) (*BCCImportResult, bool, error) {
	job, err := s.importJobRepo.GetByAssetID(ctx, assetID)
	if err != nil {
		return nil, false, err
	}
	if !job.IsTerminal() || job.AuditLoggedAt != nil {
		return nil, false, nil
	}
	result, err := s.resultForAsset(ctx, assetID)
	return result, true, err
}

func (s *BCCImportService) MarkTerminalAuditLogged(ctx context.Context, assetID uint) error {
	_, err := s.importJobRepo.MarkTerminalAuditLogged(ctx, assetID, clock.Now())
	return err
}

func stringPointer(value string) *string { return &value }

func bccRequestFingerprint(projectID uint, forMonth string, includeFlexibleEmployees bool, data []byte) string {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%d:%s:%t:", projectID, forMonth, includeFlexibleEmployees)
	_, _ = hash.Write(data)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (s *BCCImportService) applyTimesheetReplacement(
	ctx context.Context,
	staleIDs []uint,
	entries []domainservices.BulkCreateTimesheetEntry,
	uploaderID uint,
	uploaderRole string,
) (*domainservices.BulkCreateTimesheetResult, error) {
	var result *domainservices.BulkCreateTimesheetResult
	err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, id := range staleIDs {
			if err := s.timesheetWriter.HardDelete(txCtx, id); err != nil {
				return fmt.Errorf("xóa bảng chấm công cũ %d: %w", id, err)
			}
		}

		var err error
		result, err = s.timesheetService.BulkCreateTimesheetsInTransaction(
			txCtx,
			entries,
			uploaderID,
			uploaderRole,
		)
		if err != nil {
			return err
		}
		if len(result.FailedEntries) > 0 {
			return fmt.Errorf("không thể thay thế an toàn: %d dòng không hợp lệ", len(result.FailedEntries))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
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
	data, err := io.ReadAll(io.LimitReader(fileData, maxBCCUploadSize+1))
	if err != nil {
		return nil, fmt.Errorf("BCCImportService.ProcessUpload: read file: %w", err)
	}
	if len(data) > maxBCCUploadSize {
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

	return s.processAssetData(ctx, data, filename, projectID, uploaderID, uploaderRole, createdAsset, forMonth, false)
}

func (s *BCCImportService) processAssetData(
	ctx context.Context,
	data []byte,
	filename string,
	projectID uint,
	uploaderID uint,
	uploaderRole string,
	createdAsset *domain.Asset,
	forMonth string,
	includeFlexibleEmployees bool,
) (*BCCImportResult, error) {

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
		return s.processMultiPositionUpload(ctx, xf, formatResult, filename, projectID, uploaderID, uploaderRole, createdAsset, effectiveMonth, includeFlexibleEmployees)
	case excelparser.FormatWeeklyBCC:
		return s.processWeeklyBCCUpload(ctx, xf, formatResult, filename, projectID, uploaderID, uploaderRole, createdAsset, effectiveMonth, includeFlexibleEmployees)
	case excelparser.FormatWeeklyPayment:
		return s.processWeeklyPaymentUpload(ctx, xf, formatResult, filename, projectID, uploaderID, uploaderRole, createdAsset, effectiveMonth, includeFlexibleEmployees)
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
	// Normalize day type: accept both short names (Thường, Nghỉ, Lễ) and full names (ngày thường, ngày nghỉ, ngày lễ)
	dayTypePriority := map[string]int{
		"ngày thường": 0, "thường": 0,
		"ngày nghỉ": 1, "nghỉ": 1,
		"ngày lễ": 2, "lễ": 2,
	}
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
						Fullname:  fullName,
						CCCD:      cccd,
						Mobile:    row.Mobile,
						CreatedBy: uploaderID,
					}
					// Attach bank fields only when STK provides enough info to
					// build a complete record; otherwise the profile is created
					// without banking info and can be filled in later.
					applySTKBankFields(emp, row, bankID, fullName)
					createdEmp, createErr := s.employeeService.CreateEmployeeFromImport(txCtx, emp, uploaderID)
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

					// Apply changed bank info from STK using a targeted,
					// atomically validated update.
					if bankUpdates := buildSTKBankUpdates(emp, row, bankID, fullName); bankUpdates != nil {
						if updateErr := s.employeeService.UpdateBankInfo(txCtx, emp.ID, bankUpdates); updateErr != nil {
							slog.Error("BCCImport: failed to update bank info for employee",
								"employee_id", emp.ID, "error", updateErr)
							importErrors = append(importErrors, domain.ImportError{
								Employee: fullName,
								Reason:   "Không thể cập nhật thông tin ngân hàng",
							})
						} else {
							slog.Info("BCCImport: updated bank info for existing employee",
								"employee_id", emp.ID, "cccd", cccd)
						}
					}

					// Fill missing mobile from STK when employee has none.
					if row.Mobile != "" && emp.Mobile == "" {
						if err := s.employeeService.UpdateMobile(txCtx, emp.ID, row.Mobile); err != nil {
							slog.Error("BCCImport: failed to fill mobile for existing employee",
								"employee_id", emp.ID, "error", err)
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
		entries             []domainservices.BulkCreateTimesheetEntry
		flexibleEmployeeIDs = make(map[uint]struct{})
		rowNum              = 12
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
			if !includeFlexibleEmployees {
				importErrors = append(importErrors, domain.ImportError{
					Row:      rowNum,
					Employee: emp.FullName,
					Reason:   "nhân viên lương linh hoạt không áp dụng BCC import",
				})
				rowNum++
				continue
			}
			flexibleEmployeeIDs[assignment.EmployeeID] = struct{}{}
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

			// Normalize day type for consistent storage (Thường → ngày thường, etc.)
			normalizedDayType := normalizeDayType(target.dayType)
			entries = append(entries, domainservices.BulkCreateTimesheetEntry{
				ProjectID:   projectID,
				EmployeeID:  assignment.EmployeeID,
				Date:        date.Format("2006-01-02"),
				HoursWorked: entry.Hours,
				HourType:    target.hourType,
				DayType:     &normalizedDayType,
			})
		}
		rowNum++
	}

	totalRows := len(parsed.Employees)

	// 9. "Latest wins" overwrite: for any (employee, date) in the new import,
	// delete previously-imported unapproved entries so the latest upload
	// fully replaces them. Approved or paid entries are protected.
	var staleIDs []uint
	flexibleSkippedCount := 0
	if len(entries) > 0 {
		monthEnd := time.Date(year, month+1, 0, 23, 59, 59, 0, loc)
		existingTS, terr := s.timesheetReader.GetByProject(ctx, projectID, monthStart, monthEnd)
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
		existingFlexible := make(map[dk]bool)
		for _, ts := range existingTS {
			k := dk{ts.EmployeeID, ts.Date.Format("2006-01-02")}
			if !importDates[k] {
				continue
			}
			if _, isFlexible := flexibleEmployeeIDs[ts.EmployeeID]; isFlexible {
				existingFlexible[k] = true
				continue
			}

			isPaid := ts.PaymentStatus == domain.PaymentStatusPaid ||
				ts.PaymentStatus == domain.PaymentStatusFailed ||
				ts.PaymentStatus == domain.PaymentStatusCancelled

			switch {
			case isPaid:
				blocked[k] = "đã thanh toán"
			case ts.Status == domain.TimesheetStatusApproved:
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
		if len(existingFlexible) > 0 {
			filtered := entries[:0]
			for _, entry := range entries {
				if existingFlexible[dk{entry.EmployeeID, entry.Date}] {
					flexibleSkippedCount++
					continue
				}
				filtered = append(filtered, entry)
			}
			entries = filtered
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

	result, err := s.applyTimesheetReplacement(ctx, staleIDs, entries, uploaderID, uploaderRole)
	if err != nil {
		return fail("failed", fmt.Sprintf("lỗi tạo bảng chấm công: %v", err))
	}

	// 11. Count results.
	createdCount := len(result.CreatedTimesheets)
	skippedCount := len(result.DeletedTimesheets) + flexibleSkippedCount
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
	if deferTerminal, _ := ctx.Value(deferBCCTerminalMetadataKey{}).(bool); deferTerminal &&
		(meta.Status == domain.TimesheetImportStatusCompleted ||
			meta.Status == domain.TimesheetImportStatusFailed) {
		return nil
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("BCCImport: failed to marshal metadata: %w", err)
	}
	if err := s.assetRepo.UpdateMetadata(ctx, assetID, string(b)); err != nil {
		return fmt.Errorf("BCCImport: failed to update asset metadata (id=%d): %w", assetID, err)
	}
	return nil
}

// prepareImportContext performs the shared setup: month parsing, distributed lock,
// and payrate lookup. Returns the context and a cleanup function that must be
// deferred by the caller to release the lock.
func (s *BCCImportService) prepareImportContext(ctx context.Context, projectID uint, effectiveMonth string) (*bccImportContext, func(), error) {
	// Use time.Local so the lookup date matches the MySQL driver's location: the DSN uses
	// loc=Local, so cfg.Loc == time.Local. clock.Now() is pinned to Asia/Ho_Chi_Minh, which
	// diverges from time.Local in UTC containers (e.g. the scratch prod image) and shifts the
	// day boundary by 7h — breaking `from_date <= monthStart` when the import month starts on
	// the payrate's effective date. Same loc=Local precedent as timesheet_entry_table.go.
	loc := time.Local
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

// normalizeDayType converts short day type names to full names for consistent storage.
// Maps: Thường → ngày thường, Nghỉ → ngày nghỉ, Lễ → ngày lễ
// Full names are returned as-is (case-insensitive).
func normalizeDayType(dayType string) string {
	normalized := strings.ToLower(strings.TrimSpace(dayType))
	switch normalized {
	case "thường":
		return "ngày thường"
	case "nghỉ":
		return "ngày nghỉ"
	case "lễ":
		return "ngày lễ"
	default:
		// Return original if it's already a full name or unknown
		if strings.HasPrefix(normalized, "ngày ") {
			return dayType // Return as-is to preserve case
		}
		return dayType // Return unknown values as-is
	}
}
