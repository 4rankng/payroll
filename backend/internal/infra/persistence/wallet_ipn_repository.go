package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"encoding/json"
	"fmt"

	"api-server/internal/domain/wallet"

	"gorm.io/gorm"
)

type walletIPNRepository struct {
	db *gorm.DB
}

func NewWalletIPNRepository(db *Database) wallet.WalletIPNRepository {
	return &walletIPNRepository{db: db.DB}
}

func (r *walletIPNRepository) Create(ctx context.Context, ipn *wallet.WalletIPN) (uint64, error) {
	if err := r.db.WithContext(ctx).Table("wallet_ipn").Create(ipn).Error; err != nil {
		return 0, fmt.Errorf("wallet_ipn: create: %w", err)
	}
	return ipn.ID, nil
}

func (r *walletIPNRepository) UpdateProcessingResult(ctx context.Context, id uint64, status string, walletPaymentID *uint64, processingError string) error {
	updates := map[string]any{
		"processing_status": status,
		"processed_at":      clock.Now(),
	}
	if walletPaymentID != nil {
		updates["wallet_payment_id"] = *walletPaymentID
	}
	if processingError != "" {
		updates["processing_error"] = processingError
	}
	res := r.db.WithContext(ctx).
		Table("wallet_ipn").
		Where("id = ?", id).
		Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("wallet_ipn: update processing result: %w", res.Error)
	}
	return nil
}

func (r *walletIPNRepository) ListByInvoiceNo(ctx context.Context, invoiceNo string) ([]*wallet.WalletIPN, error) {
	var rows []*wallet.WalletIPN
	if err := r.db.WithContext(ctx).Table("wallet_ipn").
		Where("invoice_no = ?", invoiceNo).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("wallet_ipn: list by invoice_no: %w", err)
	}
	return rows, nil
}

// MarshalPayload converts a map to JSON bytes for RawPayload.
func MarshalPayload(m map[string]any) []byte {
	if m == nil {
		return nil
	}
	b, _ := json.Marshal(m)
	return b
}
