package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain/wallet"
)

type walletPaymentRepository struct {
	db *sql.DB
}

func NewWalletPaymentRepository(db *sql.DB) wallet.WalletPaymentRepository {
	return &walletPaymentRepository{db: db}
}

const paymentColumns = `
	id, txn_id, request_id, invoice_no, provider,
	requested_amount, fee,
	recipient_name, recipient_account_no, recipient_bank,
	description, status, error_code, error_message,
	entity_id, created_by, version, created_at, updated_at, settled_at`

func (r *walletPaymentRepository) Create(ctx context.Context, payment *wallet.WalletPayment) error {
	query := `
		INSERT INTO wallet_payments (
			txn_id, request_id, invoice_no, provider,
			requested_amount, fee,
			recipient_name, recipient_account_no, recipient_bank,
			description, status, error_code, error_message,
			entity_id, created_by, version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
	`

	result, err := r.db.ExecContext(ctx, query,
		payment.TxnID,
		payment.RequestID,
		payment.InvoiceNo,
		payment.Provider,
		payment.RequestedAmount,
		payment.Fee,
		payment.RecipientName,
		payment.RecipientAccountNo,
		payment.RecipientBank,
		payment.Description,
		payment.Status,
		payment.ErrorCode,
		payment.ErrorMessage,
		payment.EntityID,
		payment.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create wallet payment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	payment.ID = uint64(id)
	return nil
}

func (r *walletPaymentRepository) GetByID(ctx context.Context, id uint64) (*wallet.WalletPayment, error) {
	query := fmt.Sprintf("SELECT %s FROM wallet_payments WHERE id = ?", paymentColumns)
	return r.scanOne(r.db.QueryRowContext(ctx, query, id))
}

func (r *walletPaymentRepository) GetByTxnID(ctx context.Context, txnID string) (*wallet.WalletPayment, error) {
	query := fmt.Sprintf("SELECT %s FROM wallet_payments WHERE txn_id = ?", paymentColumns)
	return r.scanOne(r.db.QueryRowContext(ctx, query, txnID))
}

func (r *walletPaymentRepository) GetByRequestID(ctx context.Context, requestID string) (*wallet.WalletPayment, error) {
	query := fmt.Sprintf("SELECT %s FROM wallet_payments WHERE request_id = ?", paymentColumns)
	return r.scanOne(r.db.QueryRowContext(ctx, query, requestID))
}

func (r *walletPaymentRepository) GetByProviderInvoiceNo(ctx context.Context, provider, invoiceNo string) (*wallet.WalletPayment, error) {
	query := fmt.Sprintf("SELECT %s FROM wallet_payments WHERE provider = ? AND invoice_no = ?", paymentColumns)
	return r.scanOne(r.db.QueryRowContext(ctx, query, provider, invoiceNo))
}

func (r *walletPaymentRepository) Update(ctx context.Context, payment *wallet.WalletPayment) error {
	query := `
		UPDATE wallet_payments SET
			invoice_no = ?,
			fee = ?,
			description = ?,
			status = ?,
			error_code = ?,
			error_message = ?,
			settled_at = ?,
			version = version + 1,
			updated_at = NOW(3)
		WHERE id = ? AND version = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		payment.InvoiceNo,
		payment.Fee,
		payment.Description,
		payment.Status,
		payment.ErrorCode,
		payment.ErrorMessage,
		payment.SettledAt,
		payment.ID,
		payment.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to update wallet payment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("wallet payment not found or version mismatch")
	}

	payment.Version++
	return nil
}

func (r *walletPaymentRepository) List(ctx context.Context, filter wallet.WalletPaymentFilter) ([]*wallet.WalletPayment, int64, error) {
	whereClause, args := r.buildWhereClause(filter)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM wallet_payments %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count wallet payments: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT %s FROM wallet_payments %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, paymentColumns, whereClause)

	offset := (filter.Page - 1) * filter.PageSize
	args = append(args, filter.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list wallet payments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var payments []*wallet.WalletPayment
	for rows.Next() {
		payment, err := r.scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		payments = append(payments, payment)
	}

	return payments, total, nil
}

func (r *walletPaymentRepository) SumByStatuses(ctx context.Context, statuses []string) (int64, error) {
	placeholders := make([]string, len(statuses))
	args := make([]any, len(statuses))
	for i, s := range statuses {
		placeholders[i] = "?"
		args[i] = s
	}

	query := fmt.Sprintf(
		"SELECT COALESCE(SUM(requested_amount + fee), 0) FROM wallet_payments WHERE status IN (%s)",
		strings.Join(placeholders, ","),
	)

	var total int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to sum wallet payments by statuses: %w", err)
	}
	return total, nil
}

func (r *walletPaymentRepository) SumUnreconciledByStatuses(ctx context.Context, statuses []string) (int64, error) {
	placeholders := make([]string, len(statuses))
	args := make([]any, len(statuses))
	for i, s := range statuses {
		placeholders[i] = "?"
		args[i] = s
	}

	query := fmt.Sprintf(
		"SELECT COALESCE(SUM(requested_amount + fee), 0) FROM wallet_payments WHERE status IN (%s) AND reconciled_at IS NULL",
		strings.Join(placeholders, ","),
	)

	var total int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to sum unreconciled wallet payments by statuses: %w", err)
	}
	return total, nil
}

func (r *walletPaymentRepository) HasPendingForRecipient(ctx context.Context, accountNo, bank, provider string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM wallet_payments
		 WHERE recipient_account_no = ? AND recipient_bank = ? AND provider = ? AND status IN ('pending', 'authorised'))`,
		accountNo, bank, provider,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check pending payments for recipient: %w", err)
	}
	return exists, nil
}

func (r *walletPaymentRepository) buildWhereClause(filter wallet.WalletPaymentFilter) (string, []any) {
	var conditions []string
	var args []any

	if filter.TxnID != "" {
		conditions = append(conditions, "txn_id = ?")
		args = append(args, filter.TxnID)
	}
	if filter.RequestID != "" {
		conditions = append(conditions, "request_id = ?")
		args = append(args, filter.RequestID)
	}
	if filter.InvoiceNo != "" {
		conditions = append(conditions, "invoice_no = ?")
		args = append(args, filter.InvoiceNo)
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.ErrorCode != "" {
		conditions = append(conditions, "error_code = ?")
		args = append(args, filter.ErrorCode)
	}
	if filter.RecipientName != "" {
		conditions = append(conditions, "recipient_name LIKE ?")
		args = append(args, "%"+filter.RecipientName+"%")
	}
	if filter.RecipientAccount != "" {
		conditions = append(conditions, "recipient_account_no = ?")
		args = append(args, filter.RecipientAccount)
	}
	if filter.RecipientBank != "" {
		conditions = append(conditions, "recipient_bank = ?")
		args = append(args, filter.RecipientBank)
	}
	if filter.StartDate != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, *filter.EndDate)
	}
	if filter.EntityID != nil {
		conditions = append(conditions, "entity_id = ?")
		args = append(args, *filter.EntityID)
	}

	if len(conditions) == 0 {
		return "", args
	}

	clause := "WHERE " + conditions[0]
	for i := 1; i < len(conditions); i++ {
		clause += " AND " + conditions[i]
	}
	return clause, args
}

func (r *walletPaymentRepository) scanOne(row *sql.Row) (*wallet.WalletPayment, error) {
	var p wallet.WalletPayment
	err := row.Scan(
		&p.ID, &p.TxnID, &p.RequestID, &p.InvoiceNo, &p.Provider,
		&p.RequestedAmount, &p.Fee,
		&p.RecipientName, &p.RecipientAccountNo, &p.RecipientBank,
		&p.Description, &p.Status, &p.ErrorCode, &p.ErrorMessage,
		&p.EntityID, &p.CreatedBy, &p.Version, &p.CreatedAt, &p.UpdatedAt, &p.SettledAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet payment not found")
		}
		return nil, fmt.Errorf("failed to scan wallet payment: %w", err)
	}
	return &p, nil
}

func (r *walletPaymentRepository) scanRow(rows *sql.Rows) (*wallet.WalletPayment, error) {
	var p wallet.WalletPayment
	err := rows.Scan(
		&p.ID, &p.TxnID, &p.RequestID, &p.InvoiceNo, &p.Provider,
		&p.RequestedAmount, &p.Fee,
		&p.RecipientName, &p.RecipientAccountNo, &p.RecipientBank,
		&p.Description, &p.Status, &p.ErrorCode, &p.ErrorMessage,
		&p.EntityID, &p.CreatedBy, &p.Version, &p.CreatedAt, &p.UpdatedAt, &p.SettledAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan wallet payment row: %w", err)
	}
	return &p, nil
}

func (r *walletPaymentRepository) UpdateStatus(ctx context.Context, id uint64, status string, settledAt, reconciledAt *time.Time) error {
	query := `UPDATE wallet_payments SET status = ?, settled_at = ?, reconciled_at = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, settledAt, reconciledAt, id)
	if err != nil {
		return fmt.Errorf("failed to update wallet payment status: %w", err)
	}
	return nil
}

func (r *walletPaymentRepository) GetCompletedUnsettled(ctx context.Context, date time.Time) ([]*wallet.WalletPayment, error) {
	start := date.Truncate(24 * time.Hour)
	end := start.AddDate(0, 0, 1)

	query := `
		SELECT wp.id, wp.txn_id, wp.request_id, wp.invoice_no, wp.provider,
		       wp.requested_amount, wp.fee,
		       wp.recipient_name, wp.recipient_account_no, wp.recipient_bank,
		       wp.description, wp.status, wp.error_code, wp.error_message,
		       wp.entity_id, wp.created_by, wp.version, wp.created_at, wp.updated_at, wp.settled_at
		FROM wallet_payments wp
		INNER JOIN advance_payment_requests apr ON wp.entity_id = apr.id
		WHERE wp.status = ?
		  AND wp.updated_at >= ? AND wp.updated_at < ?
		  AND apr.settlement_transaction_id IS NULL`

	rows, err := r.db.QueryContext(ctx, query, "completed", start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query completed unsettled wallet payments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var payments []*wallet.WalletPayment
	for rows.Next() {
		p, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}
