package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"database/sql"
	"fmt"
	"time"

	"api-server/internal/domain/wallet"
)

type walletTopupRepository struct {
	db *sql.DB
}

func NewWalletTopupRepository(db *sql.DB) wallet.WalletTopupRepository {
	return &walletTopupRepository{db: db}
}

func (r *walletTopupRepository) Create(ctx context.Context, topup *wallet.WalletTopup) error {
	query := `
		INSERT INTO wallet_topups (amount, bank_ref, occurred_at, note, created_by, version)
		VALUES (?, ?, ?, ?, ?, 0)
	`

	now := clock.Now()
	if topup.CreatedAt.IsZero() {
		topup.CreatedAt = now
	}
	topup.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, query,
		topup.Amount,
		topup.BankRef,
		topup.OccurredAt,
		topup.Note,
		topup.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to create wallet topup: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	topup.ID = uint64(id)
	return nil
}

func (r *walletTopupRepository) GetByID(ctx context.Context, id uint64) (*wallet.WalletTopup, error) {
	query := `
		SELECT id, amount, bank_ref, occurred_at, note, created_by, created_at, updated_at, version
		FROM wallet_topups WHERE id = ?
	`
	return r.scanOne(r.db.QueryRowContext(ctx, query, id))
}

func (r *walletTopupRepository) GetByBankRef(ctx context.Context, bankRef string) (*wallet.WalletTopup, error) {
	query := `
		SELECT id, amount, bank_ref, occurred_at, note, created_by, created_at, updated_at, version
		FROM wallet_topups WHERE bank_ref = ?
	`
	return r.scanOne(r.db.QueryRowContext(ctx, query, bankRef))
}

func (r *walletTopupRepository) Sum(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(amount), 0) FROM wallet_topups").Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to sum wallet topups: %w", err)
	}
	return total, nil
}

func (r *walletTopupRepository) SumByDateRange(ctx context.Context, start, end time.Time) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(amount), 0) FROM wallet_topups WHERE occurred_at BETWEEN ? AND ?",
		start, end,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to sum wallet topups by date range: %w", err)
	}
	return total, nil
}

func (r *walletTopupRepository) List(ctx context.Context, filter wallet.WalletTopupFilter) ([]*wallet.WalletTopup, int64, error) {
	whereClause, args := r.buildWhereClause(filter)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM wallet_topups %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count wallet topups: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, amount, bank_ref, occurred_at, note, created_by, created_at, updated_at, version
		FROM wallet_topups %s
		ORDER BY occurred_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	offset := (filter.Page - 1) * filter.PageSize
	args = append(args, filter.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list wallet topups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var topups []*wallet.WalletTopup
	for rows.Next() {
		topup, err := r.scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		topups = append(topups, topup)
	}

	return topups, total, nil
}

func (r *walletTopupRepository) buildWhereClause(filter wallet.WalletTopupFilter) (string, []any) {
	var conditions []string
	var args []any

	if filter.BankRef != "" {
		conditions = append(conditions, "bank_ref = ?")
		args = append(args, filter.BankRef)
	}
	if filter.StartDate != nil {
		conditions = append(conditions, "occurred_at >= ?")
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		conditions = append(conditions, "occurred_at <= ?")
		args = append(args, *filter.EndDate)
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

func (r *walletTopupRepository) scanOne(row *sql.Row) (*wallet.WalletTopup, error) {
	var t wallet.WalletTopup
	err := row.Scan(&t.ID, &t.Amount, &t.BankRef, &t.OccurredAt, &t.Note, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.Version)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet topup not found")
		}
		return nil, fmt.Errorf("failed to scan wallet topup: %w", err)
	}
	return &t, nil
}

func (r *walletTopupRepository) scanRow(rows *sql.Rows) (*wallet.WalletTopup, error) {
	var t wallet.WalletTopup
	err := rows.Scan(&t.ID, &t.Amount, &t.BankRef, &t.OccurredAt, &t.Note, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to scan wallet topup row: %w", err)
	}
	return &t, nil
}
