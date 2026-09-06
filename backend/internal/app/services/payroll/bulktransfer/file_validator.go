package bulktransfer

import (
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/excelkit"
)

// ValidatedFile represents a validated and parsed file
type ValidatedFile struct {
	Asset      *domain.Asset
	ExcelFile  *ExcelFile
	Rows       [][]string
	Parser     BankResultParser
	ExportDate *time.Time
}

// FileValidator handles file validation, duplicate checking, and parsing
type FileValidator struct {
	assetService   AssetService
	ledgerService  LedgerService
	excelConverter ExcelConverter
	parsers        []BankResultParser
}

// NewFileValidator creates a new FileValidator instance
func NewFileValidator(
	assetService AssetService,
	ledgerService LedgerService,
	excelConverter ExcelConverter,
	parsers []BankResultParser,
) *FileValidator {
	return &FileValidator{
		assetService:   assetService,
		ledgerService:  ledgerService,
		excelConverter: excelConverter,
		parsers:        parsers,
	}
}

// ValidateAndParse validates the uploaded file and parses its content
func (v *FileValidator) ValidateAndParse(
	ctx context.Context,
	fileHeader *multipart.FileHeader,
	uploadedBy uint,
) (*ValidatedFile, error) {
	logger := observability.GetLogger()

	// Upload file as asset
	asset, err := v.uploadAsset(ctx, fileHeader, uploadedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to upload asset: %w", err)
	}

	// Check for duplicates
	if err := v.checkDuplicate(ctx, asset.ID); err != nil {
		return nil, err
	}

	// Read export date
	exportDate := v.readExportDate(ctx, asset.ID, logger)

	// Process Excel file
	excelFile, rows, err := v.processExcelFile(fileHeader)
	if err != nil {
		return nil, err
	}

	// Detect parser
	parser, err := v.detectParser(fileHeader.Filename, rows)
	if err != nil {
		return nil, err
	}

	return &ValidatedFile{
		Asset:      asset,
		ExcelFile:  excelFile,
		Rows:       rows,
		Parser:     parser,
		ExportDate: exportDate,
	}, nil
}

// uploadAsset uploads the file as an asset
func (v *FileValidator) uploadAsset(
	ctx context.Context,
	fileHeader *multipart.FileHeader,
	uploadedBy uint,
) (*domain.Asset, error) {
	logger := observability.GetLogger()

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("Failed to close uploaded file", "error", err)
		}
	}()

	assetRequest := domain.AssetUploadRequest{
		UploadType: domain.UploadTypeBulkTransferResult,
	}

	asset, err := v.assetService.UploadAsset(ctx, file, fileHeader, assetRequest, uploadedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to save bulk transfer result file as asset: %w", err)
	}

	return asset, nil
}

// checkDuplicate checks if the file has already been processed
func (v *FileValidator) checkDuplicate(ctx context.Context, assetID uint) error {
	existingEntries, err := v.ledgerService.GetEntriesByAssetID(ctx, assetID)
	if err != nil {
		return fmt.Errorf("failed to check for duplicate processing: %w", err)
	}

	if len(existingEntries) > 0 {
		return &ErrFileAlreadyProcessed{EntryCount: len(existingEntries)}
	}

	return nil
}

// readExportDate reads the export date from cell H2
func (v *FileValidator) readExportDate(
	ctx context.Context,
	assetID uint,
	logger *slog.Logger,
) *time.Time {
	assetInfo, err := v.assetService.GetAsset(ctx, assetID)
	if err != nil {
		logger.Warn("Failed to retrieve asset information", "error", err)
		return nil
	}

	exists, assetFilePath, err := v.assetService.CheckFileExists(assetInfo.FilePath)
	if err != nil || !exists {
		return nil
	}

	f, err := excelkit.OpenFile(assetFilePath)
	if err != nil {
		logger.Warn("Failed to open asset file to read H2", "error", err)
		return nil
	}
	defer func() {
		if err := f.Close(); err != nil {
			logger.Error("Failed to close Excel file", "error", err)
		}
	}()

	sheetNames := f.GetSheetList()
	if len(sheetNames) == 0 {
		return nil
	}

	exportDateStr, err := f.GetCellValue(sheetNames[0], "H2")
	if err != nil || exportDateStr == "" {
		return nil
	}

	parsedDate, err := time.Parse("2006-01-02", exportDateStr)
	if err != nil {
		logger.Warn("Failed to parse date from cell H2", "date", exportDateStr, "error", err)
		return nil
	}

	return &parsedDate
}

// processExcelFile processes the uploaded Excel file
func (v *FileValidator) processExcelFile(
	fileHeader *multipart.FileHeader,
) (*ExcelFile, [][]string, error) {
	excelFile, err := v.excelConverter.ProcessExcelFile(fileHeader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to process Excel file: %w", err)
	}

	sheetNames := v.excelConverter.GetSheetNames(excelFile)
	if len(sheetNames) == 0 {
		return nil, nil, &ErrInvalidFileFormat{Reason: "no sheets found"}
	}

	rows, err := v.excelConverter.GetRowsFromSheet(excelFile, sheetNames[0])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read Excel sheet: %w", err)
	}

	if len(rows) < 2 {
		return nil, nil, &ErrInvalidFileFormat{Reason: "file must have at least 2 rows"}
	}

	return excelFile, rows, nil
}

// detectParser detects the appropriate bank parser for the file
func (v *FileValidator) detectParser(
	filename string,
	rows [][]string,
) (BankResultParser, error) {
	for _, parser := range v.parsers {
		if parser.Detect(filename, rows) {
			return parser, nil
		}
	}

	return nil, &ErrInvalidFileFormat{Reason: "expected MBank format"}
}
