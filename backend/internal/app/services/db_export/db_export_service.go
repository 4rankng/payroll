package db_export

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	infraServices "api-server/internal/app/services/infrastructure"
	"api-server/internal/infra/persistence"

	"github.com/xuri/excelize/v2"
)

// ExportJobStatus represents the status of a DB export job
type ExportJobStatus struct {
	JobID       string     `json:"job_id"`
	Status      string     `json:"status"` // pending, processing, completed, failed
	Progress    int        `json:"progress"`
	TotalTables int        `json:"total_tables"`
	DoneCount   int        `json:"done_count"`
	DownloadURL string     `json:"download_url,omitempty"`
	Filename    string     `json:"filename,omitempty"`
	Error       string     `json:"error,omitempty"`
	CreatedBy   uint       `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// DBExportService handles full-database Excel export jobs.
// Jobs are tracked in Redis with a 48-hour TTL.
// Exported files are saved under <storagePath>/db_exports/.
type DBExportService struct {
	db          *persistence.Database
	cache       *infraServices.CacheService
	storagePath string
	logger      *slog.Logger
}

// NewDBExportService creates a new DBExportService
func NewDBExportService(
	db *persistence.Database,
	cache *infraServices.CacheService,
	storagePath string,
) *DBExportService {
	return &DBExportService{
		db:          db,
		cache:       cache,
		storagePath: storagePath,
		logger:      slog.Default().With("component", "DBExportService"),
	}
}

func (s *DBExportService) jobKey(jobID string) string {
	return fmt.Sprintf("db_export:job:%s", jobID)
}

func (s *DBExportService) allJobsKey() string {
	return "db_export:jobs"
}

// CreateJob creates a new export job and returns its ID
func (s *DBExportService) CreateJob(ctx context.Context, createdBy uint) (string, error) {
	jobID := fmt.Sprintf("dbexport-%d", clock.Now().UnixNano())
	job := &ExportJobStatus{
		JobID:     jobID,
		Status:    "pending",
		CreatedBy: createdBy,
		CreatedAt: clock.Now(),
		UpdatedAt: clock.Now(),
	}
	if err := s.cache.Set(ctx, s.jobKey(jobID), job, 48*time.Hour); err != nil {
		return "", fmt.Errorf("failed to create export job: %w", err)
	}
	// Track job ID in a list (stored as a JSON array in Redis)
	s.addJobToList(ctx, jobID)
	return jobID, nil
}

// GetJob retrieves the status of an export job
func (s *DBExportService) GetJob(ctx context.Context, jobID string) (*ExportJobStatus, error) {
	var job ExportJobStatus
	if err := s.cache.Get(ctx, s.jobKey(jobID), &job); err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	return &job, nil
}

// ListPendingJobs returns all known export jobs (pending + processing + recent completed)
func (s *DBExportService) ListPendingJobs(ctx context.Context) ([]*ExportJobStatus, error) {
	jobIDs, err := s.getJobList(ctx)
	if err != nil {
		return []*ExportJobStatus{}, nil
	}

	var jobs []*ExportJobStatus
	for _, id := range jobIDs {
		job, err := s.GetJob(ctx, id)
		if err != nil {
			continue // job may have expired
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// excludedTables lists tables that are too large or noisy to include in a full DB export.
var excludedTables = map[string]bool{
	"audit_logs":  true,
	"api_metrics": true,
}

// passwordColumns lists column names whose values must be redacted in exports.
var passwordColumns = map[string]bool{
	"password":           true,
	"password_hash":      true,
	"hashed_password":    true,
	"encrypted_password": true,
	"password_digest":    true,
}

func isPasswordColumn(col string) bool {
	return passwordColumns[col]
}

// StartExport starts the export job asynchronously
func (s *DBExportService) StartExport(ctx context.Context, jobID string) {
	go s.runExport(jobID)
}

func (s *DBExportService) runExport(jobID string) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("panic in db export", "job_id", jobID, "panic", r)
			s.updateJobFailed(ctx, jobID, fmt.Sprintf("internal error: %v", r))
		}
	}()

	// Mark as processing
	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		s.logger.Error("failed to get job", "job_id", jobID, "error", err)
		return
	}
	job.Status = "processing"
	job.UpdatedAt = clock.Now()
	_ = s.cache.Set(ctx, s.jobKey(jobID), job, 48*time.Hour)

	// Get all table names from the database
	tables, err := s.getTableNames(ctx)
	if err != nil {
		s.updateJobFailed(ctx, jobID, fmt.Sprintf("failed to list tables: %v", err))
		return
	}

	job.TotalTables = len(tables)
	job.UpdatedAt = clock.Now()
	_ = s.cache.Set(ctx, s.jobKey(jobID), job, 48*time.Hour)

	s.logger.Info("starting db export", "job_id", jobID, "tables", len(tables))

	// Create Excel workbook.
	// excelize always starts with "Sheet1". We rename it to the first table,
	// then create the remaining tables as new sheets. This avoids the
	// "cannot delete the only sheet" restriction.
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	for i, table := range tables {
		var sheetErr error
		if i == 0 {
			// Reuse the default Sheet1 by renaming it instead of deleting it
			sheetErr = f.SetSheetName("Sheet1", sanitizeSheetName(table))
			if sheetErr == nil {
				sheetErr = s.writeTableToSheet(ctx, f, sanitizeSheetName(table), table)
			}
		} else {
			sheetErr = s.exportTableToSheet(ctx, f, table)
		}
		if sheetErr != nil {
			s.logger.Warn("failed to export table", "table", table, "error", sheetErr)
		}

		// Update progress
		job.DoneCount = i + 1
		job.Progress = int(float64(i+1) / float64(len(tables)) * 100)
		job.UpdatedAt = clock.Now()
		_ = s.cache.Set(ctx, s.jobKey(jobID), job, 48*time.Hour)
	}

	// Save file
	filename := fmt.Sprintf("db_export_%s.xlsx", clock.Now().Format("20060102_150405"))
	exportDir := filepath.Join(s.storagePath, "db_exports")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		s.updateJobFailed(ctx, jobID, fmt.Sprintf("failed to create export directory: %v", err))
		return
	}

	filePath := filepath.Join(exportDir, filename)
	if err := f.SaveAs(filePath); err != nil {
		s.updateJobFailed(ctx, jobID, fmt.Sprintf("failed to save export file: %v", err))
		return
	}

	// Build download URL — informational only; the handler serves the file directly
	downloadURL := fmt.Sprintf("/api/v1/db-export/%s/download", jobID)

	relPath := filepath.Join("db_exports", filename)
	now := clock.Now()
	job.Status = "completed"
	job.Progress = 100
	job.Filename = filename
	job.DownloadURL = downloadURL
	job.CompletedAt = &now
	job.UpdatedAt = now
	// Store relative file path in DownloadURL temporarily; handler will use filename
	_ = s.cache.Set(ctx, s.jobKey(jobID)+":file", relPath, 48*time.Hour)
	_ = s.cache.Set(ctx, s.jobKey(jobID), job, 48*time.Hour)

	s.logger.Info("db export completed", "job_id", jobID, "file", filePath, "tables", len(tables))
}

func (s *DBExportService) getTableNames(ctx context.Context) ([]string, error) {
	sqlDB, err := s.db.DB.DB()
	if err != nil {
		return nil, err
	}

	// Try MySQL first, fall back to PostgreSQL
	rows, err := sqlDB.QueryContext(ctx, "SHOW TABLES")
	if err != nil {
		// Try PostgreSQL
		rows, err = sqlDB.QueryContext(ctx,
			"SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename")
		if err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		if !excludedTables[name] {
			tables = append(tables, name)
		}
	}
	return tables, rows.Err()
}

func sanitizeSheetName(table string) string {
	if len(table) > 31 {
		return table[:31]
	}
	return table
}

func (s *DBExportService) exportTableToSheet(ctx context.Context, f *excelize.File, table string) error {
	sheetName := sanitizeSheetName(table)
	_, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
	}
	return s.writeTableToSheet(ctx, f, sheetName, table)
}

func (s *DBExportService) writeTableToSheet(ctx context.Context, f *excelize.File, sheetName, table string) error {
	sqlDB, err := s.db.DB.DB()
	if err != nil {
		return err
	}

	rows, err := sqlDB.QueryContext(ctx, fmt.Sprintf("SELECT * FROM `%s` LIMIT 100000", table))
	if err != nil {
		// Try without backticks (PostgreSQL)
		rows, err = sqlDB.QueryContext(ctx, fmt.Sprintf(`SELECT * FROM "%s" LIMIT 100000`, table))
		if err != nil {
			return fmt.Errorf("failed to query table %s: %w", table, err)
		}
	}
	defer func() { _ = rows.Close() }()

	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns for %s: %w", table, err)
	}

	// Write header row
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
	})
	for ci, col := range cols {
		cell, _ := excelize.CoordinatesToCellName(ci+1, 1)
		_ = f.SetCellValue(sheetName, cell, col)
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Write data rows
	rowIdx := 2
	vals := make([]interface{}, len(cols))
	valPtrs := make([]interface{}, len(cols))
	for i := range vals {
		valPtrs[i] = &vals[i]
	}

	for rows.Next() {
		if err := rows.Scan(valPtrs...); err != nil {
			continue
		}
		for ci, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(ci+1, rowIdx)
			if isPasswordColumn(cols[ci]) {
				_ = f.SetCellValue(sheetName, cell, "")
				continue
			}
			switch v := val.(type) {
			case []byte:
				_ = f.SetCellValue(sheetName, cell, string(v))
			case time.Time:
				_ = f.SetCellValue(sheetName, cell, v.Format("2006-01-02 15:04:05"))
			default:
				_ = f.SetCellValue(sheetName, cell, v)
			}
		}
		rowIdx++
	}

	// Set reasonable column widths
	for ci, col := range cols {
		colName, _ := excelize.ColumnNumberToName(ci + 1)
		width := float64(len(col) + 4)
		if width < 10 {
			width = 10
		}
		if width > 40 {
			width = 40
		}
		_ = f.SetColWidth(sheetName, colName, colName, width)
	}

	return rows.Err()
}

func (s *DBExportService) updateJobFailed(ctx context.Context, jobID, errMsg string) {
	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		s.logger.Error("failed to get job for failure update", "job_id", jobID)
		return
	}
	job.Status = "failed"
	job.Error = errMsg
	job.UpdatedAt = clock.Now()
	_ = s.cache.Set(ctx, s.jobKey(jobID), job, 48*time.Hour)
}

// GetFilePath returns the local file path for a completed export job
func (s *DBExportService) GetFilePath(ctx context.Context, jobID string) (string, error) {
	var relPath string
	if err := s.cache.Get(ctx, s.jobKey(jobID)+":file", &relPath); err != nil {
		return "", fmt.Errorf("file path not found for job %s", jobID)
	}
	return filepath.Join(s.storagePath, relPath), nil
}

// addJobToList adds a job ID to the tracked list
func (s *DBExportService) addJobToList(ctx context.Context, jobID string) {
	var ids []string
	_ = s.cache.Get(ctx, s.allJobsKey(), &ids)
	ids = append(ids, jobID)
	// Keep only last 50 jobs
	if len(ids) > 50 {
		ids = ids[len(ids)-50:]
	}
	_ = s.cache.Set(ctx, s.allJobsKey(), ids, 48*time.Hour)
}

// getJobList returns all tracked job IDs
func (s *DBExportService) getJobList(ctx context.Context) ([]string, error) {
	var ids []string
	if err := s.cache.Get(ctx, s.allJobsKey(), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}
