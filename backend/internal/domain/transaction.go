package domain

import (
	"context"
	"time"

	"api-server/internal/constants"
	"gorm.io/gorm"
)

// TransactionType represents the type of transaction from user perspective
type TransactionType string

const (
	TransactionTypeExpense  TransactionType = "expense"
	TransactionTypeRevenue  TransactionType = "revenue"
	TransactionTypeCapital  TransactionType = "capital"
	TransactionTypeWriteOff TransactionType = "write_off"
	// Loan-specific transaction types (user-facing records for financing flows)
	TransactionTypeLoanDisbursement TransactionType = "loan_disbursement"
	TransactionTypeLoanRepayment    TransactionType = "loan_repayment"
)

// Label returns the Vietnamese label for the transaction type
func (t TransactionType) Label() string {
	switch t {
	case TransactionTypeRevenue:
		return "Doanh thu"
	case TransactionTypeCapital:
		return "Vốn"
	case TransactionTypeExpense:
		return "Chi phí"
	case TransactionTypeWriteOff:
		return "Xóa nợ"
	case TransactionTypeLoanDisbursement:
		return "Giải ngân khoản vay"
	case TransactionTypeLoanRepayment:
		return "Thanh toán khoản vay"
	default:
		return "Chi phí"
	}
}

// TransactionStatus represents the settlement status of a transaction
type TransactionStatus string

const (
	TransactionStatusSettled          TransactionStatus = "settled"
	TransactionStatusPending          TransactionStatus = "pending"
	TransactionStatusPartiallySettled TransactionStatus = "partially_settled"
)

// Label returns the Vietnamese label for the transaction status
func (s TransactionStatus) Label() string {
	switch s {
	case TransactionStatusSettled:
		return "Đã thanh toán"
	case TransactionStatusPending:
		return "Chờ thanh toán"
	case TransactionStatusPartiallySettled:
		return "Thanh toán thiếu"
	default:
		return "Chờ thanh toán"
	}
}

// Transaction represents a user-facing transaction that abstracts double-entry complexity
type Transaction struct {
	ID                    uint              `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Description           string            `json:"description" gorm:"type:text;not null"`
	TransactionCode       *string           `json:"transaction_code" gorm:"type:varchar(36)"`
	TransactionType       TransactionType   `json:"transaction_type" gorm:"type:varchar(50);not null"`
	Amount                int64             `json:"amount" gorm:"type:bigint;not null"`
	LoanPrincipalAmount   int64             `json:"loan_principal_amount" gorm:"type:bigint;not null;default:0"`
	LoanInterestAmount    int64             `json:"loan_interest_amount" gorm:"type:bigint;not null;default:0"`
	Party                 string            `json:"party" gorm:"type:varchar(255);not null"`
	Status                TransactionStatus `json:"status" gorm:"type:varchar(50);not null;default:'pending'"`
	SettledAmount         int64             `json:"settled_amount" gorm:"type:bigint;not null;default:0"`
	URL                   string            `json:"url" gorm:"type:varchar(500)"`
	AssetID               *uint             `json:"asset_id" gorm:"type:bigint unsigned"`
	UserID                *uint             `json:"user_id" gorm:"type:bigint unsigned"`
	LoanID                *uint             `json:"loan_id" gorm:"type:bigint unsigned"`
	ReversedTransactionID *uint             `json:"reversed_transaction_id" gorm:"type:bigint unsigned"`
	CreatedBy             uint              `json:"created_by" gorm:"not null"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
	DeletedAt             gorm.DeletedAt    `json:"-" gorm:"index"`

	// Relationships
	Asset                 *Asset        `json:"asset,omitempty" gorm:"foreignKey:AssetID;references:ID"`
	Creator               User          `json:"creator" gorm:"foreignKey:CreatedBy;references:ID"`
	LedgerEntries         []LedgerEntry `json:"ledger_entries,omitempty" gorm:"foreignKey:TransactionID;references:ID"`
	Settlements           []Settlement  `json:"settlements,omitempty" gorm:"foreignKey:TransactionID;references:ID"`
	ReversedByTransaction *Transaction  `json:"reversed_by_transaction,omitempty" gorm:"foreignKey:ReversedTransactionID;references:ID"`
}

// TransactionRepository defines the interface for transaction persistence operations
type TransactionRepository interface {
	Create(ctx context.Context, txn *Transaction) error
	GetByID(ctx context.Context, id uint) (*Transaction, error)
	GetByIDForUpdate(ctx context.Context, id uint) (*Transaction, error)
	GetByIDs(ctx context.Context, ids []uint) ([]*Transaction, error)
	GetByAssetID(ctx context.Context, assetID uint) (*Transaction, error)
	List(ctx context.Context, filters TransactionFilters) ([]*Transaction, error)
	Count(ctx context.Context, filters TransactionFilters) (int64, error)
	Update(ctx context.Context, txn *Transaction) error
	GetByParty(ctx context.Context, party string) ([]*Transaction, error)
	GetPendingRevenueByParty(ctx context.Context, party string) ([]*Transaction, error)
	GetSettledTransactionsByLoan(ctx context.Context, loanID uint) ([]*Transaction, error)
	GetWithRoundingImbalance(ctx context.Context, tolerance int64, since time.Time) ([]*Transaction, error)
	Delete(ctx context.Context, id uint) error
	GetPendingByDescription(ctx context.Context, description string) (*Transaction, error)
	// IncrementAmount atomically adds delta to the transaction's amount
	// (UPDATE transactions SET amount = amount + delta WHERE id = ?). Used by the
	// wallet-settlement recovery path to keep Amount in sync when late-arriving
	// payments are appended to an existing day's transaction. Honors tx context.
	IncrementAmount(ctx context.Context, id uint, delta int64) error
}

// TransactionFilters represents filtering options for transaction queries
type TransactionFilters struct {
	// Query parameters
	Page            int
	PageSize        int
	SortBy          string
	SortOrder       string
	TransactionType *string // Filter by transaction_type (revenue/expense)
	Status          *string // Filter by status (pending/settled)
	Party           *string
	Search          string // Search in description and party
	FromDate        *time.Time
	ToDate          *time.Time
	CreatedBy       *uint

	// Internal fields for repository (computed from Page/PageSize)
	Limit  int
	Offset int
}

// Validate validates the transaction
func (t *Transaction) Validate() error {
	if t.Description == "" {
		return NewValidationError("mô tả là bắt buộc")
	}

	if t.Amount <= 0 {
		return NewValidationError("số tiền phải lớn hơn 0")
	}

	if t.Party == "" {
		return NewValidationError("quản lý là bắt buộc")
	}

	if t.TransactionType != TransactionTypeExpense &&
		t.TransactionType != TransactionTypeRevenue &&
		t.TransactionType != TransactionTypeCapital &&
		t.TransactionType != TransactionTypeWriteOff &&
		t.TransactionType != TransactionTypeLoanDisbursement &&
		t.TransactionType != TransactionTypeLoanRepayment {
		return NewValidationError("loại giao dịch không hợp lệ")
	}

	if t.Status != TransactionStatusSettled &&
		t.Status != TransactionStatusPending &&
		t.Status != TransactionStatusPartiallySettled {
		return NewValidationError("trạng thái giao dịch không hợp lệ")
	}

	if t.TransactionType == TransactionTypeLoanRepayment {
		if t.LoanPrincipalAmount < 0 || t.LoanInterestAmount < 0 {
			return NewValidationError("Phân bổ gốc và lãi không được âm")
		}
		if (t.LoanPrincipalAmount != 0 || t.LoanInterestAmount != 0) && t.LoanPrincipalAmount+t.LoanInterestAmount != t.Amount {
			return NewValidationError("Phân bổ gốc và lãi không khớp số tiền giao dịch")
		}
	}

	return nil
}

// IsPending returns true if transaction is pending settlement
func (t *Transaction) IsPending() bool {
	return t.Status == TransactionStatusPending
}

// IsSettled returns true if transaction is settled
func (t *Transaction) IsSettled() bool {
	return t.Status == TransactionStatusSettled
}

// IsExpense returns true if transaction is an expense
func (t *Transaction) IsExpense() bool {
	return t.TransactionType == TransactionTypeExpense
}

// IsRevenue returns true if transaction is revenue
func (t *Transaction) IsRevenue() bool {
	return t.TransactionType == TransactionTypeRevenue
}

// IsCapital returns true if transaction is capital
func (t *Transaction) IsCapital() bool {
	return t.TransactionType == TransactionTypeCapital
}

func (t *Transaction) IsWriteOff() bool {
	return t.TransactionType == TransactionTypeWriteOff
}

// GetRemainingAmount returns the remaining amount to be settled
func (t *Transaction) GetRemainingAmount() int64 {
	remaining := t.Amount - t.SettledAmount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsFullySettled returns true if transaction is fully settled
func (t *Transaction) IsFullySettled() bool {
	remaining := t.GetRemainingAmount()
	return t.SettledAmount >= t.Amount || remaining <= constants.RoundingTolerance
}

// IsPartiallySettled returns true if transaction has partial settlements
func (t *Transaction) IsPartiallySettled() bool {
	remaining := t.GetRemainingAmount()
	return t.SettledAmount > 0 && remaining > constants.RoundingTolerance
}

// CanSettle validates if a settlement amount can be applied
func (t *Transaction) CanSettle(amount int64) error {
	if amount <= 0 {
		return NewValidationError("số tiền thanh toán phải lớn hơn 0")
	}

	remaining := t.GetRemainingAmount()
	if amount > remaining {
		return NewValidationError("số tiền thanh toán vượt quá số còn lại")
	}

	return nil
}

// NormalizeSettledAmount ensures the settled_amount reflects the transaction status.
// When there are no settlement records yet, fully settled transactions should report
// settled_amount == amount while pending transactions should report zero.
func (t *Transaction) NormalizeSettledAmount(hasSettlements bool) {
	if t.SettledAmount < 0 {
		t.SettledAmount = 0
	}

	if t.SettledAmount > t.Amount {
		t.SettledAmount = t.Amount
	}

	if hasSettlements {
		return
	}

	if t.Status == TransactionStatusSettled {
		t.SettledAmount = t.Amount
		return
	}

	t.SettledAmount = 0
}

// PrepareForCreate applies status-based defaults before persisting a new transaction.
func (t *Transaction) PrepareForCreate() {
	t.NormalizeSettledAmount(false)
}

// UpdateStatus updates transaction status based on settled amount
func (t *Transaction) UpdateStatus() {
	if t.IsFullySettled() {
		t.Status = TransactionStatusSettled
	} else if t.IsPartiallySettled() {
		t.Status = TransactionStatusPartiallySettled
	} else {
		t.Status = TransactionStatusPending
	}
}

// IsReversed returns true if transaction has been reversed
func (t *Transaction) IsReversed() bool {
	return t.ReversedTransactionID != nil
}

// CanReverse validates if a transaction can be reversed
func (t *Transaction) CanReverse() error {
	if t.IsReversed() {
		return NewValidationError("giao dịch đã được đảo ngược")
	}

	if t.IsPending() {
		return NewValidationError("chỉ có thể đảo ngược giao dịch đã thanh toán (giao dịch chờ nên được xóa)")
	}

	return nil
}

// GetLedgerAccounts returns the accounts to be used for ledger entries
// based on transaction type and settlement status
func (t *Transaction) GetLedgerAccounts() (debitAccount, creditAccount LedgerAccount) {
	if t.IsExpense() {
		// Expense always debits expense account
		debitAccount = AccountExpense

		if t.Status == TransactionStatusSettled {
			// Settled expense: Credit cash
			creditAccount = AccountCash
		} else {
			// Pending expense: Credit payable
			creditAccount = AccountPayable
		}
	} else if t.IsRevenue() {
		// Revenue always credits revenue account
		creditAccount = AccountRevenue

		if t.Status == TransactionStatusSettled {
			// Settled revenue: Debit cash
			debitAccount = AccountCash
		} else {
			// Pending revenue: Debit receivable
			debitAccount = AccountReceivable
		}
	} else if t.IsCapital() {
		// Capital contribution: Owner invests money into the business
		// Always credits equity account (increases owner's equity)
		creditAccount = AccountEquity

		if t.Status == TransactionStatusSettled {
			// Settled capital: Debit cash (money received)
			debitAccount = AccountCash
		} else {
			// Pending capital: Debit receivable (owner promised to invest)
			debitAccount = AccountReceivable
		}
	} else if t.TransactionType == TransactionTypeLoanDisbursement {
		// Loan disbursement: receive cash, increase loan liability
		debitAccount = AccountCash
		creditAccount = AccountLoan
	} else if t.TransactionType == TransactionTypeLoanRepayment {
		// Loan repayment: when pending, move liability to payable; when settled, reduce loan and cash
		debitAccount = AccountLoan
		if t.Status == TransactionStatusPending {
			creditAccount = AccountPayable
		} else {
			creditAccount = AccountCash
		}
	} else if t.IsWriteOff() {
		// Write-off transaction: reduce receivable, record loss as expense
		// Always credits receivable (reduce what clients owe)
		creditAccount = AccountReceivable

		// Write-offs should always expense the loss
		debitAccount = AccountExpense
	}

	return debitAccount, creditAccount
}
