package transactions

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// WalletPayment is one row in the wallet_payments table.
//
// It tracks every money-sending interaction with a disbursement provider
// (currently 9pay). Status is managed by the FSM in state_machine.go.
type WalletPayment struct {
	ID        uint64    `gorm:"column:id;primaryKey;type:bigint unsigned;autoIncrement"`
	TxnID     uuid.UUID `gorm:"column:txn_id;type:char(36);uniqueIndex;not null"`
	RequestID string    `gorm:"column:request_id;type:varchar(64);uniqueIndex;not null"`
	InvoiceNo *string   `gorm:"column:invoice_no;type:varchar(128)"`
	Provider  string    `gorm:"column:provider;type:varchar(16);not null;default:'9pay'"`

	RequestedAmount int64 `gorm:"column:requested_amount;not null"`
	Fee             int64 `gorm:"column:fee;not null;default:0"`

	RecipientName      string `gorm:"column:recipient_name;type:varchar(255);not null"`
	RecipientAccountNo string `gorm:"column:recipient_account_no;type:varchar(64);not null"`
	RecipientBank      string `gorm:"column:recipient_bank;type:varchar(64);not null"`

	Description *string `gorm:"column:description;type:text"`

	Status       State   `gorm:"column:status;type:varchar(32);not null;default:'pending';index:idx_wp_status_created,priority:1"`
	ErrorCode    *string `gorm:"column:error_code;type:varchar(32)"`
	ErrorMessage *string `gorm:"column:error_message;type:text"`

	EntityID  *uint64 `gorm:"column:entity_id;type:bigint unsigned;index"`
	CreatedBy *uint64 `gorm:"column:created_by;type:bigint unsigned;index:idx_wp_created_by"`
	BatchID   *string `gorm:"column:batch_id;type:varchar(64);index:idx_wallet_payments_batch_id"`

	// Bulk-transfer linkage (migration 093). Populated by the bulk-transfer
	// row worker via WalletPaymentRepository.UpdateBulkBatchLink so we can
	// SUM(fee) GROUP BY bulk_transfer_batch_id and render KQ rows in input
	// order via bulk_transfer_order. vfic_code mirrors request_id for the
	// admin search index (request_id already has idx_wp_request_id UNIQUE).
	BulkTransferBatchID *uint64 `gorm:"column:bulk_transfer_batch_id;type:bigint unsigned;index:idx_wp_bulk_batch"`
	BulkTransferOrder   *uint   `gorm:"column:bulk_transfer_order;type:int unsigned"`
	VFICCode            *string `gorm:"column:vfic_code;type:varchar(32);index:idx_wp_vfic_code"`
	EnqueueState        *string `gorm:"column:enqueue_state;type:varchar(16);default:'enqueued'"`
	SweeperRetryCount   uint    `gorm:"column:sweeper_retry_count;type:int unsigned;not null;default:0"`

	Version int64 `gorm:"column:version;not null;default:0"`

	CreatedAt        time.Time  `gorm:"column:created_at;type:datetime(3);not null;autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;type:datetime(3);not null;autoUpdateTime"`
	SettledAt        *time.Time `gorm:"column:settled_at;type:datetime(3)"`
	ReconciledAt     *time.Time `gorm:"column:reconciled_at;type:datetime(3)"`
	ResolutionSource *string    `gorm:"column:resolution_source;type:varchar(32)"`
}

// TableName binds the entity to its physical table.
func (WalletPayment) TableName() string { return "wallet_payments" }

// GetInvoiceNo returns the invoice_no as a string, or "" when nil.
func (t *WalletPayment) GetInvoiceNo() string {
	if t.InvoiceNo == nil {
		return ""
	}
	return *t.InvoiceNo
}

// IsTerminal reports whether this row's status is a final, immutable state.
func (t *WalletPayment) IsTerminal() bool { return IsTerminal(t.Status) }

// IsReconciled reports whether this row has been confirmed by reconciliation.
func (t *WalletPayment) IsReconciled() bool { return t.ReconciledAt != nil }

var ErrConcurrentModification = errors.New("wallet_payments: concurrent modification (version mismatch)")

var ErrNotFound = errors.New("wallet_payments: not found")

// ResolutionSource constants identify what triggered a terminal state transition.
const (
	ResolutionSourceIPN            = "ipn"
	ResolutionSourceStatusInquiry  = "status_inquiry"
	ResolutionSourceManual         = "manual"
	ResolutionSourceReconciliation = "reconciliation"
)

// validResolutionSources is the allow-list for resolution_source values.
var validResolutionSources = map[string]bool{
	ResolutionSourceIPN:            true,
	ResolutionSourceStatusInquiry:  true,
	ResolutionSourceManual:         true,
	ResolutionSourceReconciliation: true,
}

// ValidResolutionSource reports whether s is a recognised resolution source.
func ValidResolutionSource(s string) bool { return validResolutionSources[s] }
