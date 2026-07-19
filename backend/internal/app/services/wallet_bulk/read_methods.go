package wallet_bulk

import (
	"context"
	"errors"
	"fmt"

	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"
)

// BatchDetail is the full read-side projection of a batch + its rows.
// Returned by GetBatch for the frontend detail view.
type BatchDetail struct {
	*domain.BulkTransferBatch
	Rows []*domaintx.WalletPayment `json:"rows"`
}

// BatchListEntry is the list projection (no rows — fetched on demand).
type BatchListEntry struct {
	*domain.BulkTransferBatch
}

// BatchListResponse carries paginated batches + totals.
type BatchListResponse struct {
	Batches []*domain.BulkTransferBatch `json:"batches"`
	Total   int64                       `json:"total"`
	Page    int                         `json:"page"`
	PageSize int                       `json:"page_size"`
}

// GetBatch returns the batch + its wallet_payments rows ordered by
// bulk_transfer_order ASC (matches the original input order).
func (s *WalletBulkTransferService) GetBatch(ctx context.Context, id uint64) (*BatchDetail, error) {
	batch, err := s.batchRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get batch: %w", err)
	}
	rows, err := s.paymentRepo.ListByBatchIDOrdered(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list rows: %w", err)
	}
	return &BatchDetail{BulkTransferBatch: batch, Rows: rows}, nil
}

// ListBatches returns paginated batches newest-first.
func (s *WalletBulkTransferService) ListBatches(ctx context.Context, page, pageSize int) (*BatchListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	filter := domain.BulkTransferBatchFilter{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
	batches, total, err := s.batchRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list batches: %w", err)
	}
	return &BatchListResponse{
		Batches:  batches,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// DownloadKQ generates the KQ Excel for a batch and returns (bytes, filename).
// Works at any batch status — the KQ reflects the current state of each row.
func (s *WalletBulkTransferService) DownloadKQ(ctx context.Context, id uint64) ([]byte, string, error) {
	batch, err := s.batchRepo.GetByID(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("get batch: %w", err)
	}
	rows, err := s.paymentRepo.ListByBatchIDOrdered(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("list rows: %w", err)
	}
	gen := NewKQExcelGenerator(s.clock)
	bytes, err := gen.Generate(ctx, batch, rows)
	if err != nil {
		return nil, "", fmt.Errorf("generate kq: %w", err)
	}
	filename := FilenameForKQ(id, s.clock())
	return bytes, filename, nil
}

// FindBatchByContentHash exposes the dup-check lookup for tests + admin
// "is this file already uploaded?" queries.
func (s *WalletBulkTransferService) FindBatchByContentHash(ctx context.Context, hash string) (*domain.BulkTransferBatch, error) {
	b, err := s.batchRepo.GetByContentHash(ctx, hash)
	if err != nil && !errors.Is(err, domain.ErrBulkTransferBatchNotFound) {
		return nil, err
	}
	return b, nil
}
