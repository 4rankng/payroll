package bulktransfer

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

// FileHistoryService handles bulk transfer file operations for upload histories
type FileHistoryService struct {
	fileRepo BulkTransferFileRepository
}

// NewFileHistoryService creates a new file history service instance
func NewFileHistoryService(fileRepo BulkTransferFileRepository) *FileHistoryService {
	return &FileHistoryService{
		fileRepo: fileRepo,
	}
}

// GetBulkTransferUploadHistories retrieves bulk transfer result upload histories from bulk_transfer_files
func (fhs *FileHistoryService) GetBulkTransferUploadHistories(
	ctx context.Context,
	req *dto.ListBulkTransferHistoriesRequest,
	periodCalculator *PeriodCalculator,
) (*dto.ListBulkTransferHistoriesResponse, error) {
	// Parse and validate date parameters
	fromDate, toDate, err := periodCalculator.ParseDateRange(req.FromDate, req.ToDate)
	if err != nil {
		return nil, err
	}

	// Calculate offset
	offset := (req.Page - 1) * req.PageSize

	// Retrieve result upload records (asset_id IS NOT NULL) with pagination
	files, total, err := fhs.fileRepo.ListResultUploads(ctx, fromDate, toDate, req.PageSize, offset, req.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve bulk transfer result files: %w", err)
	}

	// Build summaries from bulk transfer files
	summaries := make([]dto.BulkTransferHistorySummary, 0, len(files))
	for _, file := range files {
		summary := fhs.buildSummaryFromFile(file)
		summaries = append(summaries, summary)
	}

	// Calculate total pages
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize != 0 {
		totalPages++
	}

	return &dto.ListBulkTransferHistoriesResponse{
		Data: summaries,
		Pagination: dto.PaginationResponse{
			Page:         req.Page,
			PageSize:     req.PageSize,
			TotalPages:   totalPages,
			TotalRecords: total,
		},
	}, nil
}

// buildSummaryFromFile creates a BulkTransferHistorySummary from a BulkTransferFile record
func (fhs *FileHistoryService) buildSummaryFromFile(file *domain.BulkTransferFile) dto.BulkTransferHistorySummary {
	uploaderName := ""
	if file.Creator.ID != 0 {
		uploaderName = file.Creator.Fullname
	}

	return dto.BulkTransferHistorySummary{
		ID:           file.ID,
		Filename:     file.Filename,
		TotalTxn:     file.TransactionsCount,
		CompletedTxn: file.CompletedCount,
		FailedTxn:    file.FailedCount,
		UploadedBy:   uploaderName,
		UploadedAt:   file.CreatedAt.Format(time.RFC3339),
	}
}
