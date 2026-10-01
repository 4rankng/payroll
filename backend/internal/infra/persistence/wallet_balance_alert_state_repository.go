package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"api-server/internal/domain/wallet"
	"api-server/internal/pkg/clock"
)

type walletBalanceAlertStateRepository struct {
	db *sql.DB
}

// NewWalletBalanceAlertStateRepository returns the raw-SQL repository for the
// single-row wallet low-balance alert state.
func NewWalletBalanceAlertStateRepository(db *sql.DB) wallet.BalanceAlertStateRepository {
	return &walletBalanceAlertStateRepository{db: db}
}

// walletBalanceAlertStateID is the fixed single-row id.
const walletBalanceAlertStateID = 1

func (r *walletBalanceAlertStateRepository) Get(ctx context.Context) (*wallet.BalanceAlertState, error) {
	query := `
		SELECT id, is_below, last_notified_at, created_at, updated_at
		FROM wallet_balance_alert_states
		WHERE id = ?
	`
	var state wallet.BalanceAlertState
	err := r.db.QueryRowContext(ctx, query, walletBalanceAlertStateID).
		Scan(&state.ID, &state.IsBelow, &state.LastNotifiedAt, &state.CreatedAt, &state.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Never evaluated: counts as "above" so the first downward
			// crossing after deploy still notifies exactly once.
			return &wallet.BalanceAlertState{ID: walletBalanceAlertStateID}, nil
		}
		return nil, fmt.Errorf("failed to get wallet balance alert state: %w", err)
	}
	return &state, nil
}

func (r *walletBalanceAlertStateRepository) Save(ctx context.Context, state *wallet.BalanceAlertState) error {
	query := `
		INSERT INTO wallet_balance_alert_states (id, is_below, last_notified_at)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			is_below = VALUES(is_below),
			last_notified_at = VALUES(last_notified_at)
	`
	now := clock.Now()
	if state.ID == 0 {
		state.ID = walletBalanceAlertStateID
	}

	lastNotified := state.LastNotifiedAt
	if _, err := r.db.ExecContext(ctx, query, state.ID, state.IsBelow, lastNotified); err != nil {
		return fmt.Errorf("failed to save wallet balance alert state: %w", err)
	}
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	state.UpdatedAt = now
	return nil
}
