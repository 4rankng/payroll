package persistence

import (
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/timeutil"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BulkTransferFileRepository implements domain.BulkTransferFileRepository
type BulkTransferFileRepository struct {
	DB                  *gorm.DB
	transactionCodeRepo domain.TransactionCodeRepository
}

// NewBulkTransferFileRepository creates a new BulkTransferFileRepository
func NewBulkTransferFileRepository(db *gorm.DB, tcRepo domain.TransactionCodeRepository) *BulkTransferFileRepository {
	return &BulkTransferFileRepository{
		DB:                  db,
		transactionCodeRepo: tcRepo,
	}
}

// Create creates a new bulk transfer file record
func (r *BulkTransferFileRepository) Create(ctx context.Context, file *domain.BulkTransferFile) error {
	return r.DB.WithContext(ctx).Create(file).Error
}

// GetByID retrieves a bulk transfer file by ID with related entities
func (r *BulkTransferFileRepository) GetByID(ctx context.Context, id uint) (*domain.BulkTransferFile, error) {
	var file domain.BulkTransferFile
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		First(&file, id).Error

	if err != nil {
		return nil, err
	}

	return &file, nil
}

// GetByIDWithAsset retrieves a bulk transfer file by ID with Asset preload
func (r *BulkTransferFileRepository) GetByIDWithAsset(ctx context.Context, id uint) (*domain.BulkTransferFile, error) {
	var file domain.BulkTransferFile
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Preload("Asset").
		First(&file, id).Error

	if err != nil {
		return nil, err
	}

	return &file, nil
}

// GetByFilename retrieves a bulk transfer file by filename
func (r *BulkTransferFileRepository) GetByFilename(ctx context.Context, filename string) (*domain.BulkTransferFile, error) {
	var file domain.BulkTransferFile
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Where("filename = ?", filename).
		First(&file).Error

	if err != nil {
		return nil, err
	}

	return &file, nil
}

// GetByAssetID retrieves a bulk transfer file by asset_id
func (r *BulkTransferFileRepository) GetByAssetID(ctx context.Context, assetID uint) (*domain.BulkTransferFile, error) {
	var file domain.BulkTransferFile
	err := r.DB.WithContext(ctx).
		Preload("Creator").
		Preload("Asset").
		Where("asset_id = ?", assetID).
		First(&file).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("bulk transfer file not found")
		}
		return nil, err
	}

	return &file, nil
}

// ListWithFilters retrieves bulk transfer files with pagination and filtering
func (r *BulkTransferFileRepository) ListWithFilters(ctx context.Context, cycle string, fromDate, toDate *time.Time, limit, offset int) ([]*domain.BulkTransferFile, int64, error) {
	var files []*domain.BulkTransferFile
	var total int64

	query := r.DB.WithContext(ctx).Model(&domain.BulkTransferFile{})

	// Apply cycle filter
	if cycle != "" {
		query = query.Where("cycle = ?", cycle)
	}

	// Apply date filters on created_at
	if fromDate != nil {
		query = query.Where("created_at >= ?", *fromDate)
	}
	if toDate != nil {
		endOfDay := timeutil.EndOfDay(*toDate)
		query = query.Where("created_at <= ?", endOfDay)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get records with preloading
	err := query.
		Preload("Creator").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&files).Error

	return files, total, err
}

func (r *BulkTransferFileRepository) FindByTimesheetIDs(ctx context.Context, timesheetIDs []uint) (*domain.BulkTransferFile, error) {
	if len(timesheetIDs) == 0 {
		return nil, domain.NewValidationError("không có timesheet để đối soát")
	}

	var files []domain.BulkTransferFile
	if err := r.DB.WithContext(ctx).
		Preload("Creator").
		Preload("Asset").
		Where("asset_id IS NOT NULL").
		Order("created_at DESC").
		Find(&files).Error; err != nil {
		return nil, err
	}

	target := make(map[uint]struct{}, len(timesheetIDs))
	for _, id := range timesheetIDs {
		target[id] = struct{}{}
	}

	type record struct {
		TimesheetIDs []uint `json:"timesheet_ids"`
	}

	for _, file := range files {
		data := strings.TrimSpace(file.Data)
		if data == "" || data == "{}" {
			continue
		}

		var entries []record
		if err := json.Unmarshal([]byte(file.Data), &entries); err != nil {
			continue
		}

		fileSet := make(map[uint]struct{})
		for _, entry := range entries {
			for _, tsID := range entry.TimesheetIDs {
				fileSet[tsID] = struct{}{}
			}
		}

		found := true
		for tsID := range target {
			if _, ok := fileSet[tsID]; !ok {
				found = false
				break
			}
		}

		if found {
			return &file, nil
		}
	}

	return nil, domain.NewNotFoundError("bulk transfer file not found for provided timesheets")
}

func (r *BulkTransferFileRepository) FindByTimesheetIDGroups(ctx context.Context, timesheetIDs []uint) ([]*domain.BulkTransferFile, error) {
	if len(timesheetIDs) == 0 {
		return nil, domain.NewValidationError("không có timesheet để đối soát")
	}

	var files []domain.BulkTransferFile
	if err := r.DB.WithContext(ctx).
		Preload("Creator").
		Preload("Asset").
		Where("asset_id IS NOT NULL").
		Order("created_at DESC").
		Find(&files).Error; err != nil {
		return nil, err
	}

	target := make(map[uint]struct{}, len(timesheetIDs))
	for _, id := range timesheetIDs {
		target[id] = struct{}{}
	}

	type record struct {
		TimesheetIDs []uint `json:"timesheet_ids"`
	}

	var matchingFiles []*domain.BulkTransferFile
	for i := range files {
		data := strings.TrimSpace(files[i].Data)
		if data == "" || data == "{}" {
			continue
		}

		var entries []record
		if err := json.Unmarshal([]byte(data), &entries); err != nil {
			continue
		}

		fileSet := make(map[uint]struct{})
		for _, entry := range entries {
			for _, tsID := range entry.TimesheetIDs {
				fileSet[tsID] = struct{}{}
			}
		}

		// Check if this file contains any of our target timesheet IDs
		hasMatch := false
		for tsID := range target {
			if _, ok := fileSet[tsID]; ok {
				hasMatch = true
				break
			}
		}

		if hasMatch {
			matchingFiles = append(matchingFiles, &files[i])
		}
	}

	if len(matchingFiles) == 0 {
		return nil, domain.NewNotFoundError("no bulk transfer files found for provided timesheet IDs")
	}

	return matchingFiles, nil
}

// UpdateTransactionData updates data and asset_id atomically
// Must be called within a transaction
func (r *BulkTransferFileRepository) UpdateTransactionData(ctx context.Context, tx interface{}, id uint, updatedData string, assetID uint) error {
	db, ok := tx.(*gorm.DB)
	if !ok {
		return fmt.Errorf("invalid transaction type: expected *gorm.DB, got %T", tx)
	}
	return db.WithContext(ctx).
		Model(&domain.BulkTransferFile{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"data":       updatedData,
			"asset_id":   assetID,
			"updated_at": clock.Now(),
		}).Error
}

// UpdateWithLock updates a bulk transfer file with row-level locking
func (r *BulkTransferFileRepository) UpdateWithLock(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var file domain.BulkTransferFile
		if err := tx.Clauses(
			clause.Locking{
				Strength: "UPDATE",
			},
		).First(&file, id).Error; err != nil {
			return err
		}

		if err := tx.Model(&file).Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})
}

// ListForPayrollReport retrieves processed bulk transfer files for payroll report
func (r *BulkTransferFileRepository) ListForPayrollReport(ctx context.Context, fromDate, toDate time.Time) ([]*domain.BulkTransferFile, error) {
	var files []*domain.BulkTransferFile

	err := r.DB.WithContext(ctx).
		Where("asset_id IS NOT NULL").
		Where(r.DB.Where("cycle = ? AND from_date IS NOT NULL AND to_date IS NOT NULL AND from_date >= ? AND to_date <= ?", "weekly", fromDate, toDate).
			Or("cycle = ? AND for_month IS NOT NULL AND STR_TO_DATE(CONCAT(for_month, '-01'), '%Y-%m-%d') BETWEEN ? AND ?", "monthly", fromDate, toDate).
			Or("cycle = ? AND for_month IS NOT NULL AND LAST_DAY(STR_TO_DATE(CONCAT(for_month, '-01'), '%Y-%m-%d')) BETWEEN ? AND ?", "monthly", fromDate, toDate)).
		Order("created_at ASC").
		Find(&files).Error

	return files, err
}

// DeleteOrphanFiles permanently deletes bulk transfer files where asset_id is null and created_at is older than 7 days
func (r *BulkTransferFileRepository) DeleteOrphanFiles(ctx context.Context) (int64, error) {
	sevenDaysAgo := clock.Now().Add(-7 * 24 * time.Hour)

	result := r.DB.WithContext(ctx).
		Where("asset_id IS NULL AND created_at < ?", sevenDaysAgo).
		Delete(&domain.BulkTransferFile{})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// GetRecentHighValueTransfers retrieves transfer amounts from processed bulk transfer files
// within the specified date range, filtering for amounts > 1,000,000 VND
func (r *BulkTransferFileRepository) GetRecentHighValueTransfers(ctx context.Context, fromDate, toDate time.Time) (map[uint][]int64, error) {
	var files []*domain.BulkTransferFile

	err := r.DB.WithContext(ctx).
		Where("asset_id IS NOT NULL").
		Where("created_at >= ?", fromDate).
		Where("created_at <= ?", toDate).
		Find(&files).Error

	if err != nil {
		return nil, err
	}

	employeeTransfers := make(map[uint][]int64)

	for _, file := range files {
		data := strings.TrimSpace(file.Data)
		if data == "" || data == "{}" {
			continue
		}

		var entries []struct {
			STT             int     `json:"stt"`
			EmployeeID      uint    `json:"employee_id"`
			ProjectID       uint    `json:"project_id"`
			TimesheetIDs    []uint  `json:"timesheet_ids"`
			AccountNumber   string  `json:"account_number"`
			AccountName     string  `json:"account_name"`
			BankName        string  `json:"bank_name"`
			Amount          int64   `json:"amount"`
			TransactionCode string  `json:"transaction_code"`
			TransferStatus  string  `json:"transfer_status,omitempty"`
			BankTxnRef      string  `json:"bank_txn_ref,omitempty"`
			ErrorMessage    string  `json:"error_message,omitempty"`
			UploadedAt      *string `json:"uploaded_at,omitempty"`
		}

		if err := json.Unmarshal([]byte(file.Data), &entries); err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.Amount > 1000000 && entry.TransferStatus == "completed" {
				employeeTransfers[entry.EmployeeID] = append(employeeTransfers[entry.EmployeeID], entry.Amount)
			}
		}
	}

	return employeeTransfers, nil
}

// CountEmployeesPaidInPeriod counts distinct employees who received completed payments in the given period
func (r *BulkTransferFileRepository) CountEmployeesPaidInPeriod(ctx context.Context, fromDate, toDate time.Time) (int, error) {
	var result struct {
		TotalEmployees int `gorm:"column:total_employees"`
	}

	err := r.DB.WithContext(ctx).
		Model(&domain.BulkTransferFile{}).
		Select("COALESCE(SUM(completed_count), 0) as total_employees").
		Where("asset_id IS NOT NULL").
		Where("created_at >= ?", fromDate).
		Where("created_at <= ?", toDate).
		Scan(&result).Error

	if err != nil {
		return 0, err
	}

	return result.TotalEmployees, nil
}

// FindAllInDateRange finds all bulk transfer files within the given date range
func (r *BulkTransferFileRepository) FindAllInDateRange(ctx context.Context, fromDate, toDate time.Time, files *[]*domain.BulkTransferFile) error {
	err := r.DB.WithContext(ctx).
		Where("asset_id IS NOT NULL").
		Where("created_at >= ?", fromDate).
		Where("created_at <= ?", toDate).
		Order("created_at ASC").
		Find(files).Error

	return err
}

// GetSalaryDistribution retrieves individual salary values from processed bulk transfer files,
// grouped by cycle type (weekly, monthly, flexible).
// If fromDate/toDate are provided, only returns data within that date range.
func (r *BulkTransferFileRepository) GetSalaryDistribution(ctx context.Context, fromDate, toDate *time.Time) (map[string][]int64, error) {
	var files []*domain.BulkTransferFile

	query := r.DB.WithContext(ctx).
		Where("asset_id IS NOT NULL").
		Where("data IS NOT NULL AND data != ''")

	if fromDate != nil && toDate != nil {
		// Extend toDate to end of day so records created throughout the last day are included
		toDateEndOfDay := time.Date(toDate.Year(), toDate.Month(), toDate.Day(), 23, 59, 59, 0, toDate.Location())
		query = query.Where(
			// Weekly: include if the pay period overlaps with the requested range
			// (from_date <= toDate AND to_date >= fromDate), falling back to from_date only when to_date is missing
			r.DB.Where("cycle = ? AND from_date IS NOT NULL AND to_date IS NOT NULL AND from_date <= ? AND to_date >= ?", "weekly", toDateEndOfDay, *fromDate).
				Or("cycle = ? AND from_date IS NOT NULL AND to_date IS NULL AND from_date >= ? AND from_date <= ?", "weekly", *fromDate, toDateEndOfDay).
				Or("cycle = ? AND for_month IS NOT NULL AND STR_TO_DATE(CONCAT(for_month, '-01'), '%Y-%m-%d') BETWEEN ? AND ?", "monthly", *fromDate, toDateEndOfDay).
				Or("cycle = ? AND for_month IS NOT NULL AND LAST_DAY(STR_TO_DATE(CONCAT(for_month, '-01'), '%Y-%m-%d')) BETWEEN ? AND ?", "monthly", *fromDate, toDateEndOfDay).
				Or("(cycle NOT IN (?, ?) OR cycle IS NULL) AND created_at >= ? AND created_at <= ?", "weekly", "monthly", *fromDate, toDateEndOfDay),
		)
	}

	err := query.Order("created_at ASC").Find(&files).Error
	if err != nil {
		return nil, err
	}

	cycleSalaries := make(map[string][]int64)

	for _, file := range files {
		data := strings.TrimSpace(file.Data)
		if data == "" || data == "{}" {
			continue
		}

		var entries []struct {
			STK             int     `json:"stt"`
			EmployeeID      uint    `json:"employee_id"`
			ProjectID       uint    `json:"project_id"`
			TimesheetIDs    []uint  `json:"timesheet_ids"`
			AccountNumber   string  `json:"account_number"`
			AccountName     string  `json:"account_name"`
			BankName        string  `json:"bank_name"`
			Amount          int64   `json:"amount"`
			TransactionCode string  `json:"transaction_code"`
			TransferStatus  string  `json:"transfer_status,omitempty"`
			BankTxnRef      string  `json:"bank_txn_ref,omitempty"`
			ErrorMessage    string  `json:"error_message,omitempty"`
			UploadedAt      *string `json:"uploaded_at,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &entries); err != nil {
			continue
		}

		cycle := "flexible"
		if file.Cycle != nil {
			cycle = *file.Cycle
		}

		for _, entry := range entries {
			if entry.Amount > 0 {
				cycleSalaries[cycle] = append(cycleSalaries[cycle], entry.Amount)
			}
		}
	}

	// Sort each cycle's values
	for cycle := range cycleSalaries {
		sort.Slice(cycleSalaries[cycle], func(i, j int) bool {
			return cycleSalaries[cycle][i] < cycleSalaries[cycle][j]
		})
	}

	return cycleSalaries, nil
}

// GetMostRecentWithoutAsset retrieves the most recent bulk transfer file that doesn't have an asset linked
func (r *BulkTransferFileRepository) GetMostRecentWithoutAsset(ctx context.Context) (*domain.BulkTransferFile, error) {
	var file domain.BulkTransferFile
	err := r.DB.WithContext(ctx).
		Where("asset_id IS NULL").
		Order("created_at DESC").
		First(&file).Error

	if err != nil {
		return nil, err
	}

	return &file, nil
}

// ListResultUploads retrieves bulk transfer result upload records (asset_id IS NOT NULL)
// with pagination and optional date filtering
func (r *BulkTransferFileRepository) ListResultUploads(ctx context.Context, fromDate, toDate *time.Time, limit, offset int, sortOrder string) ([]*domain.BulkTransferFile, int64, error) {
	var files []*domain.BulkTransferFile
	var total int64

	query := r.DB.WithContext(ctx).Model(&domain.BulkTransferFile{}).
		Where("asset_id IS NOT NULL OR source = ?", "ninepay")

	if fromDate != nil {
		query = query.Where("created_at >= ?", *fromDate)
	}
	if toDate != nil {
		endOfDay := timeutil.EndOfDay(*toDate)
		query = query.Where("created_at <= ?", endOfDay)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "created_at DESC"
	if sortOrder == "asc" {
		order = "created_at ASC"
	}

	err := query.
		Preload("Creator").
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&files).Error

	return files, total, err
}

// FindDataByTransactionCodes searches transaction_codes table for entries matching the given transaction codes
// Also returns the detected payment cycle from the first matched entry
func (r *BulkTransferFileRepository) FindDataByTransactionCodes(ctx context.Context, transactionCodes []string) ([]domain.BulkTransferFileDataEntry, string, error) {
	if len(transactionCodes) == 0 {
		return nil, "", domain.NewValidationError("không có mã giao dịch để đối soát")
	}

	tcs, err := r.transactionCodeRepo.FindByCodes(ctx, transactionCodes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to find transaction codes: %w", err)
	}

	foundCodes := make(map[string]*domain.TransactionCode, len(tcs))
	for _, tc := range tcs {
		foundCodes[tc.Code] = tc
	}

	var results []domain.BulkTransferFileDataEntry
	var detectedCycle string

	for _, code := range transactionCodes {
		tc, found := foundCodes[code]
		if !found {
			return nil, "", domain.NewValidationError(
				fmt.Sprintf("không tìm thấy mã giao dịch trong hệ thống: %s", code),
			)
		}

		var tcData domain.TransactionCodeData
		if err := json.Unmarshal(tc.Data, &tcData); err != nil {
			return nil, "", fmt.Errorf("failed to parse transaction code data for %s: %w", code, err)
		}

		if detectedCycle == "" {
			detectedCycle = tcData.GetCycle()
		} else if cycle := tcData.GetCycle(); cycle != "" && cycle != detectedCycle {
			return nil, "", fmt.Errorf("mixed cycles detected in result file: %s and %s", detectedCycle, cycle)
		}

		entry := domain.BulkTransferFileDataEntry{
			FileID:          tcData.GetFileID(),
			TransactionID:   0,
			TimesheetIDs:    tcData.GetTimesheetIDs(),
			EmployeeID:      tcData.GetEmployeeID(),
			Amount:          tcData.GetAmount(),
			TransactionCode: code,
		}
		results = append(results, entry)
	}

	return results, detectedCycle, nil
}

// UpdateCounts updates the completed_count and failed_count for a bulk transfer file.
func (r *BulkTransferFileRepository) UpdateCounts(ctx context.Context, id uint, completedCount, failedCount int) error {
	return r.DB.WithContext(ctx).
		Model(&domain.BulkTransferFile{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"completed_count": completedCount,
			"failed_count":    failedCount,
		}).Error
}

func (r *BulkTransferFileRepository) GetPendingUploadsByUser(ctx context.Context, userID uint) ([]*domain.BulkTransferFile, error) {
	var files []*domain.BulkTransferFile
	err := r.DB.WithContext(ctx).
		Where("created_by = ? AND status = ?", userID, "exported").
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}
